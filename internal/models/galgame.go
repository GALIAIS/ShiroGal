package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type Galgame struct {
	ID           int64
	TitleJP      string
	TitleCN      *string
	Brand        *string
	ReleaseDate  time.Time
	Synopsis     *string
	CoverURL     *string
	PreviewURLs  *string
	Tags         *string
	DownloadLink *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (g *Galgame) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID           json.Number `json:"id"`
		TitleJP      string      `json:"title_jp"`
		TitleCN      *string     `json:"title_cn"`
		Brand        *string     `json:"brand"`
		ReleaseDate  *string     `json:"release_date"`
		CreatedAt    *string     `json:"created_at"`
		UpdatedAt    *string     `json:"updated_at"`
		Synopsis     *string     `json:"synopsis"`
		CoverURL     *string     `json:"cover_url"`
		PreviewURLs  *string     `json:"preview_urls"`
		Tags         *string     `json:"tags"`
		DownloadLink *string     `json:"download_link"`
	}

	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	id, err := a.ID.Int64()
	if err != nil {
		return fmt.Errorf("could not parse ID %s as int64: %w", a.ID, err)
	}
	g.ID = id

	const layout = "2006-01-02 15:04:05" // 这是TiDB返回的主要日期格式

	if a.ReleaseDate != nil && *a.ReleaseDate != "" {
		releaseDate, err := time.Parse(layout, *a.ReleaseDate)
		if err == nil {
			g.ReleaseDate = releaseDate
		}
	}

	if a.CreatedAt != nil && *a.CreatedAt != "" {
		createdAt, err := time.Parse(layout, *a.CreatedAt)
		if err == nil {
			g.CreatedAt = createdAt
		}
	}

	if a.UpdatedAt != nil && *a.UpdatedAt != "" {
		updatedAt, err := time.Parse(layout, *a.UpdatedAt)
		if err == nil {
			g.UpdatedAt = updatedAt
		}
	}

	g.TitleJP = a.TitleJP
	g.TitleCN = a.TitleCN
	g.Brand = a.Brand
	g.Synopsis = a.Synopsis
	g.CoverURL = a.CoverURL
	g.PreviewURLs = a.PreviewURLs
	g.Tags = a.Tags
	g.DownloadLink = a.DownloadLink

	return nil
}
