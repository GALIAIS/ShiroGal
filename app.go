package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"galgame-gui/internal/api"
	"galgame-gui/internal/database"
	ggsync "galgame-gui/internal/sync"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	dataServiceURL string
	publicKey      string
	privateKey     string
)

type GameView struct {
	ID          int64  `json:"ID"`
	TitleJP     string `json:"TitleJP"`
	TitleCN     string `json:"TitleCN"`
	Brand       string `json:"Brand"`
	ReleaseDate string `json:"ReleaseDate"`
	CoverURL    string `json:"CoverURL"`
}

type GameDetailsView struct {
	ID           int64   `json:"id"`
	TitleJP      string  `json:"title_jp"`
	TitleCN      string  `json:"title_cn,omitempty"`
	Brand        string  `json:"brand"`
	ReleaseDate  string  `json:"release_date"`
	Synopsis     string  `json:"synopsis"`
	CoverURL     string  `json:"cover_url"`
	PreviewURLs  *string `json:"preview_urls,omitempty"`
	Tags         string  `json:"tags"`
	UpdatedAt    string  `json:"updated_at"`
	DownloadLink *string `json:"download_link,omitempty"`
}

type App struct {
	ctx        context.Context
	db         *database.Service
	apiClient  *api.Client
	isSyncing  bool
	syncMutex  sync.Mutex
	isReady    bool
	readyMutex sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func stringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (a *App) OnStartup(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("!!! OnStartup 发生严重错误 (panic): %v", r)
		}
	}()

	a.ctx = ctx
	log.SetPrefix("[ShiroGal] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if dataServiceURL == "" || publicKey == "" || privateKey == "" {
		log.Fatalf("应用初始化错误: 构建配置不完整, Data Service的URL或API密钥未设置。")
	}

	var err error
	a.db, err = database.NewService("ShiroGal.db")
	if err != nil {
		log.Fatalf("应用初始化：无法初始化本地数据库: %v", err)
	}

	a.apiClient = api.NewClient(dataServiceURL, publicKey, privateKey)

	go a.runSync()

	a.readyMutex.Lock()
	a.isReady = true
	a.readyMutex.Unlock()

	runtime.EventsEmit(a.ctx, "backend-ready")
}

func (a *App) OnShutdown() {
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) GetGames(keyword string, limit int, offset int) ([]GameView, error) {
	keyword = strings.TrimSpace(keyword)
	keywords := strings.Fields(keyword)
	var args []interface{}
	var whereClause string
	if len(keywords) > 0 {
		conditions := []string{}
		for _, kw := range keywords {
			likeKeyword := "%" + kw + "%"
			conditions = append(conditions, "(title_jp LIKE ? OR title_cn LIKE ? OR brand LIKE ?)")
			args = append(args, likeKeyword, likeKeyword, likeKeyword)
		}
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf(`SELECT id, title_jp, title_cn, brand, release_date, cover_url FROM games %s ORDER BY release_date DESC LIMIT ? OFFSET ?`, whereClause)
	args = append(args, limit, offset)
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("数据库查询失败: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)
	var games []GameView
	for rows.Next() {
		var id int64
		var titleJP, titleCN, brand, releaseDate, coverURL sql.NullString
		if err := rows.Scan(&id, &titleJP, &titleCN, &brand, &releaseDate, &coverURL); err != nil {
			log.Printf("扫描游戏列表行失败: %v", err)
			continue
		}

		formattedReleaseDate := ""
		if releaseDate.Valid && releaseDate.String != "" {
			t, err := time.Parse(time.RFC3339, releaseDate.String)
			if err == nil {
				formattedReleaseDate = t.Format("2006-01-02")
			} else {
				log.Printf("仍然无法解析 release_date '%s': %v", releaseDate.String, err)
			}
		}

		games = append(games, GameView{
			ID:          id,
			TitleJP:     titleJP.String,
			TitleCN:     titleCN.String,
			Brand:       brand.String,
			ReleaseDate: formattedReleaseDate,
			CoverURL:    coverURL.String,
		})
	}
	return games, nil
}

func (a *App) GetGameDetails(id int64) (GameDetailsView, error) {
	game, err := a.db.GetGameByID(id)
	if err != nil {
		return GameDetailsView{}, err
	}
	gameView := GameDetailsView{
		ID:           game.ID,
		TitleJP:      game.TitleJP,
		TitleCN:      stringFromPtr(game.TitleCN),
		Brand:        stringFromPtr(game.Brand),
		ReleaseDate:  game.ReleaseDate.Format("2006-01-02"),
		Synopsis:     stringFromPtr(game.Synopsis),
		CoverURL:     stringFromPtr(game.CoverURL),
		PreviewURLs:  game.PreviewURLs,
		Tags:         stringFromPtr(game.Tags),
		UpdatedAt:    game.UpdatedAt.Format(time.RFC3339),
		DownloadLink: game.DownloadLink,
	}
	return gameView, nil
}

func (a *App) runSync() {
	a.syncMutex.Lock()
	if a.isSyncing {
		a.syncMutex.Unlock()
		return
	}
	a.isSyncing = true
	a.syncMutex.Unlock()
	defer func() {
		a.syncMutex.Lock()
		a.isSyncing = false
		a.syncMutex.Unlock()
	}()
	log.Println("数据同步：后台数据同步开始...")
	err := ggsync.Run(a.db, a.apiClient)
	if err != nil {
		log.Printf("数据同步：同步失败: %v", err)
	} else {
		log.Println("数据同步：后台同步成功。")
	}
}

func (a *App) TriggerSync() {
	go a.runSync()
}

func (a *App) CheckBackendReady() bool {
	a.readyMutex.Lock()
	defer a.readyMutex.Unlock()
	return a.isReady
}
