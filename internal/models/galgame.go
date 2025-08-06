package models

import (
	"time"
)

type Galgame struct {
	ID              int64     `json:"id"`
	TitleJP         string    `json:"title_jp"`
	TitleCN         *string   `json:"title_cn,omitempty"`
	Brand           *string   `json:"brand,omitempty"`
	ReleaseDate     time.Time `json:"release_date"`
	Synopsis        *string   `json:"synopsis,omitempty"`
	CoverURL        *string   `json:"cover_url,omitempty"`
	PreviewURLs     *string   `json:"preview_urls,omitempty"`
	Tags            *string   `json:"tags,omitempty"`
	DownloadLink    *string   `json:"download_link,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	WordpressPostID *int64    `json:"wordpress_post_id,omitempty"`
}
