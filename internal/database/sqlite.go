package database

import (
	"database/sql"
	"errors"
	"fmt"
	"galgame-gui/internal/models"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Service struct {
	db *sql.DB
}

func NewService(dbPath string) (*Service, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库: %w", err)
	}

	service := &Service{db: db}
	if err = service.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("无法创建表: %w", err)
	}

	return service, nil
}

func (s *Service) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

func (s *Service) createTables() error {
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS games (
        id INTEGER PRIMARY KEY,
        title_jp TEXT NOT NULL,
        title_cn TEXT,
        brand TEXT,
        release_date DATETIME,
        synopsis TEXT,
        cover_url TEXT,
        preview_urls TEXT,
        tags TEXT,
        download_link TEXT,
        wordpress_post_id INTEGER,
        created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
        updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
    );`

	createTriggerQuery := `
    CREATE TRIGGER IF NOT EXISTS update_games_updated_at
    AFTER UPDATE ON games
    FOR EACH ROW
    BEGIN
        UPDATE games SET updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now') WHERE id = OLD.id;
    END;`

	if _, err := s.db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("创建 games 表失败: %w", err)
	}
	if _, err := s.db.Exec(createTriggerQuery); err != nil {
		return fmt.Errorf("创建 updated_at 触发器失败: %w", err)
	}

	createIndexQuery := `
    CREATE INDEX IF NOT EXISTS idx_title_cn ON games(title_cn);
    CREATE INDEX IF NOT EXISTS idx_updated_at ON games(updated_at);`
	if _, err := s.db.Exec(createIndexQuery); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	return nil
}

func (s *Service) GetLatestTimestamp() (time.Time, error) {
	var maxTimeString sql.NullString
	query := `SELECT MAX(updated_at) FROM games;`

	err := s.db.QueryRow(query).Scan(&maxTimeString)
	if err != nil {
		return time.Time{}, err
	}

	if maxTimeString.Valid && maxTimeString.String != "" {
		const sqliteLayout = "2006-01-02 15:04:05"
		parsedTime, err := time.Parse(sqliteLayout, maxTimeString.String)
		if err != nil {
			return time.Time{}, fmt.Errorf("无法解析数据库中的时间字符串'%s': %w", maxTimeString.String, err)
		}
		return parsedTime, nil
	}

	return time.Time{}, nil
}

func (s *Service) GetAllGameIDs() ([]int64, error) {
	query := `SELECT id FROM games;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("获取所有游戏ID失败: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Printf("扫描游戏ID失败: %v", err)
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (s *Service) UpsertGames(games []models.Galgame) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			err := tx.Rollback()
			if err != nil {
				return
			}
			panic(p)
		} else if err != nil {
			err := tx.Rollback()
			if err != nil {
				return
			}
		} else {
			err = tx.Commit()
		}
	}()

	stmt, err := tx.Prepare(`
        INSERT INTO games (id, title_jp, title_cn, brand, release_date, synopsis, cover_url, preview_urls, tags, download_link, wordpress_post_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            title_jp=excluded.title_jp,
            title_cn=excluded.title_cn,
            brand=excluded.brand,
            release_date=excluded.release_date,
            synopsis=excluded.synopsis,
            cover_url=excluded.cover_url,
            preview_urls=excluded.preview_urls,
            tags=excluded.tags,
            download_link=excluded.download_link,
            wordpress_post_id=excluded.wordpress_post_id;
    `)
	if err != nil {
		return 0, err
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {

		}
	}(stmt)

	count := 0
	for _, game := range games {
		_, err := stmt.Exec(
			game.ID, game.TitleJP, game.TitleCN, game.Brand,
			game.ReleaseDate, game.Synopsis, game.CoverURL,
			game.PreviewURLs, game.Tags, game.DownloadLink, game.WordpressPostID,
		)
		if err != nil {
			log.Printf("插入/更新游戏ID %d 失败: %v", game.ID, err)
			continue
		}
		count++
	}

	return count, nil
}

func (s *Service) DeleteGames(ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(`DELETE FROM games WHERE id IN (%s);`, placeholders)

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量删除游戏失败: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取删除影响的行数失败: %w", err)
	}

	return int(rowsAffected), nil
}

func (s *Service) GetGameByID(id int64) (models.Galgame, error) {
	var game models.Galgame
	query := `SELECT 
                id, title_jp, title_cn, brand, release_date, 
                synopsis, cover_url, preview_urls, tags, download_link, 
                created_at, updated_at, wordpress_post_id
              FROM games WHERE id = ?;`

	err := s.db.QueryRow(query, id).Scan(
		&game.ID, &game.TitleJP, &game.TitleCN, &game.Brand, &game.ReleaseDate,
		&game.Synopsis, &game.CoverURL, &game.PreviewURLs, &game.Tags, &game.DownloadLink,
		&game.CreatedAt, &game.UpdatedAt, &game.WordpressPostID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Galgame{}, fmt.Errorf("未找到ID为 %d 的游戏", id)
		}
		return models.Galgame{}, fmt.Errorf("查询游戏详情失败: %w", err)
	}
	return game, nil
}

func (s *Service) UpdateDownloadLink(id int64, link string) error {
	query := `UPDATE games SET download_link = ? WHERE id = ?;`
	nullLink := sql.NullString{String: link, Valid: link != ""}

	res, err := s.db.Exec(query, nullLink, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("未找到ID为 %d 的游戏", id)
	}
	return nil
}

func (s *Service) Close() {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}
}
