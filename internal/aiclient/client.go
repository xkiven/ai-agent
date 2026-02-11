package aiclient

import (
	"ai-agent/model"
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	httpCli *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpCli: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Chat(req model.ChatRequest) (*model.ChatResponse, error) {
	bs, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", c.baseURL+"/chat", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cr model.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, err
	}
	return &cr, nil
}
