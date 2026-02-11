package service

import (
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
)

type ChatService struct {
	ai *aiclient.Client
}

func NewChatService(ai *aiclient.Client) *ChatService {
	return &ChatService{
		ai: ai,
	}
}

// HandleMessage：先做最小闭环，后面你再加 session / RAG / 路由
func (s *ChatService) HandleMessage(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	// 直接转发给 Python AI
	resp, err := s.ai.Chat(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
