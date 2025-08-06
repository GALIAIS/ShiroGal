package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"galgame-gui/internal/api"
	"galgame-gui/internal/database"
	"galgame-gui/internal/models"
	ggsync "galgame-gui/internal/sync"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	sourceRepo *api.SourceRepository

	userConfigPath string

	isSyncing bool
	syncMutex sync.Mutex

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

func decrypt(encryptedDataB64 string, secretKey string) (string, error) {
	key := []byte(secretKey)
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedDataB64)
	if err != nil {
		return "", fmt.Errorf("无法解码Base64数据: %w", err)
	}

	if len(encryptedData) < 12 {
		return "", fmt.Errorf("加密数据无效: 长度不足")
	}

	iv := encryptedData[:12]
	ciphertext := encryptedData[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建密码块失败: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

func fetchAndDecryptDSN(apiURL, encryptionKey string) (string, error) {
	resp, err := http.Get(apiURL + "/config")
	if err != nil {
		return "", fmt.Errorf("请求配置API失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("配置API返回错误状态: %s, 响应: %s", resp.Status, string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取API响应失败: %w", err)
	}

	var payload struct {
		EncryptedDSN string `json:"encrypted_dsn"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("解析API响应JSON失败: %w", err)
	}

	if payload.EncryptedDSN == "" {
		return "", fmt.Errorf("API响应中未找到加密的DSN")
	}

	return decrypt(payload.EncryptedDSN, encryptionKey)
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

	workerApiURL := ""
	encryptionKey := ""

	decryptedDSN, err := fetchAndDecryptDSN(workerApiURL, encryptionKey)
	if err != nil {
		log.Fatalf("应用初始化：无法获取或解密数据库配置: %v", err)
	}

	a.db, err = database.NewService("ShiroGal.db")
	if err != nil {
		log.Fatalf("应用初始化：无法初始化本地数据库: %v", err)
	}

	a.sourceRepo, err = api.NewSourceRepository(decryptedDSN)
	if err != nil {
		log.Fatalf("应用初始化：无法连接到主数据源: %v", err)
	}

	go a.runSync()

	a.readyMutex.Lock()
	a.isReady = true
	a.readyMutex.Unlock()

	runtime.EventsEmit(a.ctx, "backend-ready")
}

func (a *App) OnShutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
	if a.sourceRepo != nil {
		a.sourceRepo.Close()
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

	query := fmt.Sprintf(`
       SELECT id, title_jp, title_cn, brand, release_date, cover_url
       FROM games
       %s
       ORDER BY release_date DESC
       LIMIT ? OFFSET ?
    `, whereClause)

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
		var game models.Galgame
		if err := rows.Scan(&game.ID, &game.TitleJP, &game.TitleCN, &game.Brand, &game.ReleaseDate, &game.CoverURL); err != nil {
			continue
		}
		games = append(games, GameView{
			ID:          game.ID,
			TitleJP:     game.TitleJP,
			TitleCN:     stringFromPtr(game.TitleCN),
			Brand:       stringFromPtr(game.Brand),
			ReleaseDate: game.ReleaseDate.Format("2006-01-02"),
			CoverURL:    stringFromPtr(game.CoverURL),
		})
	}

	if len(games) > 0 {
		_, _ = json.Marshal(games[0])
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
		log.Println("数据同步：同步已在进行中。")
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
	err := ggsync.Run(a.db, a.sourceRepo)
	if err != nil {
		errorMsg := fmt.Sprintf("数据同步：同步失败: %v", err)
		log.Printf(errorMsg)
	} else {
		log.Println("数据同步：后台同步成功。")
	}
}

func (a *App) TriggerSync() {
	go a.runSync()
	go a.CheckBackendReady()
}

func (a *App) CheckBackendReady() bool {
	a.readyMutex.Lock()
	defer a.readyMutex.Unlock()
	return a.isReady
}
