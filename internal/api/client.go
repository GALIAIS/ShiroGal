package api

import (
	"encoding/json"
	"fmt"
	"galgame-gui/internal/models"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	BaseURL    string
	PublicKey  string
	PrivateKey string
	HTTPClient *http.Client
}

type TidbDataServiceResponse struct {
	Type string `json:"type"`
	Data struct {
		Rows   json.RawMessage `json:"rows"`
		Result struct {
			Code    json.Number `json:"code"`
			Message string      `json:"message"`
		} `json:"result"`
	} `json:"data"`
}

func NewClient(baseURL string, publicKey string, privateKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

func (c *Client) newAuthenticatedRequest(method, urlStr string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.PublicKey, c.PrivateKey)
	return req, nil
}

func (c *Client) GetUpdates(since time.Time) ([]models.Galgame, error) {
	fullURL, _ := url.Parse(c.BaseURL + "/games/updates")
	queryParams := fullURL.Query()
	queryParams.Set("since", since.Format(time.RFC3339))
	fullURL.RawQuery = queryParams.Encode()

	req, err := c.newAuthenticatedRequest("GET", fullURL.String())
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

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("读取API响应失败: %w", err)
	}

	var apiResponse TidbDataServiceResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("无法解析API响应外层JSON，原始响应: %s", string(body))
		return nil, fmt.Errorf("解析API响应外层JSON失败: %w", err)
	}

	resultCode, _ := strconv.Atoi(apiResponse.Data.Result.Code.String())
	if resultCode != 200 {
		log.Printf("API返回了非200的内部代码。完整响应: %s", string(body))
		return nil, fmt.Errorf("API返回错误代码 %d: %s", resultCode, apiResponse.Data.Result.Message)
	}

	var games []models.Galgame
	if err := json.Unmarshal(apiResponse.Data.Rows, &games); err != nil {
		log.Printf("解析games数组失败，原始rows数据: %s", string(apiResponse.Data.Rows))
		return nil, fmt.Errorf("解析游戏更新列表JSON失败: %w", err)
	}

	return games, nil
}

func (c *Client) GetAllActiveIDs() ([]int64, error) {
	fullURL := c.BaseURL + "/games/ids"

	req, err := c.newAuthenticatedRequest("GET", fullURL)
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

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("读取API响应失败: %w", err)
	}

	var apiResponse TidbDataServiceResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("无法解析API响应外层JSON，原始响应: %s", string(body))
		return nil, fmt.Errorf("解析API响应外层JSON失败: %w", err)
	}

	resultCode, _ := strconv.Atoi(apiResponse.Data.Result.Code.String())
	if resultCode != 200 {
		log.Printf("API返回了非200的内部代码。完整响应: %s", string(body))
		return nil, fmt.Errorf("API返回错误代码 %d: %s", resultCode, apiResponse.Data.Result.Message)
	}

	var idObjects []struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(apiResponse.Data.Rows, &idObjects); err != nil {
		log.Printf("解析id对象数组失败，原始rows数据: %s", string(apiResponse.Data.Rows))
		return nil, fmt.Errorf("解析ID列表JSON响应失败: %w", err)
	}

	ids := make([]int64, len(idObjects))
	for i, obj := range idObjects {
		id, err := obj.ID.Int64()
		if err != nil {
			log.Printf("无法将ID '%s' 转换为整数: %v", obj.ID.String(), err)
			continue
		}
		ids[i] = id
	}

	return ids, nil
}
