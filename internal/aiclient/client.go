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
			Timeout: 30 * time.Second,
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

func (c *Client) RecognizeIntent(req model.IntentRecognitionRequest) (*model.IntentRecognitionResponse, error) {
	bs, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", c.baseURL+"/intent/recognize", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ir model.IntentRecognitionResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return nil, err
	}
	return &ir, nil
}

func (c *Client) CreateTicket(req model.Ticket) (*model.Ticket, error) {
	bs, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", c.baseURL+"/ticket/create", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ticket model.Ticket
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (c *Client) CheckFlowInterrupt(req model.InterruptCheckRequest) (*model.InterruptCheckResponse, error) {
	bs, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", c.baseURL+"/flow/interrupt-check", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ir model.InterruptCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return nil, err
	}
	return &ir, nil
}
