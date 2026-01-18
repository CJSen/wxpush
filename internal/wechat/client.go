package wechat

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) GetStableToken(appid, secret string) (string, error) {
	// 获取稳定 access_token，不记录敏感信息
	payload := map[string]any{
		"grant_type":    "client_credential",
		"appid":         appid,
		"secret":        secret,
		"force_refresh": false,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.weixin.qq.com/cgi-bin/stable_token", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("wechat stable_token status=%d duration=%s", resp.StatusCode, time.Since(start))
	var data struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.AccessToken, nil
}

type SendResponse struct {
	ErrMsg string `json:"errmsg"`
}

func (c *Client) SendMessage(accessToken, userid, templateID, targetURL, title, content string) (*SendResponse, error) {
	// 发送模板消息，避免记录 access_token 与正文
	sendURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s", url.QueryEscape(accessToken))
	beijingTime := time.Now().UTC().Add(8 * time.Hour)
	date := beijingTime.Format("2006-01-02 15:04:05")

	payload := map[string]any{
		"touser":      userid,
		"template_id": templateID,
		"url":         targetURL,
		"data": map[string]any{
			"title":   map[string]string{"value": title},
			"content": map[string]string{"value": content},
			"date":    map[string]string{"value": date},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, sendURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("wechat send status=%d duration=%s userid=%s template_id=%s", resp.StatusCode, time.Since(start), userid, templateID)
	var data SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
