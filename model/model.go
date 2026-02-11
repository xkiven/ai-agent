package model

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
	Type  string `json:"type"`
}
