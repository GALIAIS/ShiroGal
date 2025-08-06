package api

import (
	"encoding/json"
	"fmt"
	"galgame-gui/internal/models"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

func (c *Client) GetUpdates(since time.Time) ([]models.Galgame, error) {
	url := fmt.Sprintf("%s/api/games/updates?since=%s", c.BaseURL, since.Format(time.RFC3339))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行API请求失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %s", res.Status)
	}

	var games []models.Galgame
	if err := json.NewDecoder(res.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return games, nil
}

func (c *Client) GetAllActiveIDs() ([]int64, error) {
	url := fmt.Sprintf("%s/api/games/ids", c.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建获取所有ID的请求失败: %w", err)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行API请求获取ID列表失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API获取ID列表返回错误状态: %s", res.Status)
	}

	var ids []int64
	if err := json.NewDecoder(res.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("解析ID列表JSON响应失败: %w", err)
	}

	return ids, nil
}
