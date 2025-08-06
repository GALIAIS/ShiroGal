package api

import (
	"database/sql"
	"fmt"
	"galgame-gui/internal/models"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type SourceRepository struct {
	db *sql.DB
}

func NewSourceRepository(dsn string) (*SourceRepository, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_DSN 不能为空")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	return &SourceRepository{db: db}, nil
}

func (r *SourceRepository) Close() {
	if r.db != nil {
		err := r.db.Close()
		if err != nil {
			return
		}
	}
}

func (r *SourceRepository) GetAllGameIDs() ([]int64, error) {
	query := "SELECT id FROM games"
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("[ERROR] SourceRepo.GetAllGameIDs: 查询失败: %v", err)
		return nil, err
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
			log.Printf("[ERROR] SourceRepo.GetAllGameIDs: 扫描失败: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *SourceRepository) GetGamesSince(since time.Time) ([]models.Galgame, error) {
	query := `
        SELECT 
            id, title_jp, title_cn, brand, release_date, synopsis, 
            cover_url, preview_urls, tags, download_link, created_at,
            updated_at, wordpress_post_id
        FROM games 
        WHERE IFNULL(updated_at, '1970-01-01') > ? 
        ORDER BY updated_at ASC
    `
	rows, err := r.db.Query(query, since)
	if err != nil {
		log.Printf("[ERROR] SourceRepo.GetGamesSince: 查询失败: %v", err)
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var games []models.Galgame
	for rows.Next() {
		var g models.Galgame
		err := rows.Scan(
			&g.ID,
			&g.TitleJP,
			&g.TitleCN,
			&g.Brand,
			&g.ReleaseDate,
			&g.Synopsis,
			&g.CoverURL,
			&g.PreviewURLs,
			&g.Tags,
			&g.DownloadLink,
			&g.CreatedAt,
			&g.UpdatedAt,
			&g.WordpressPostID,
		)
		if err != nil {
			log.Printf("[ERROR] SourceRepo.GetGamesSince: 扫描数据失败: %v", err)
			return nil, err
		}
		games = append(games, g)
	}
	return games, nil
}
