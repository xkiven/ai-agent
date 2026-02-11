package service

import (
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"time"

	"github.com/google/uuid"
)

type ChatService struct {
	ai *aiclient.Client
}

func NewChatService(ai *aiclient.Client) *ChatService {
	return &ChatService{
		ai: ai,
	}
}

func (s *ChatService) HandleMessage(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	intentReq := model.IntentRecognitionRequest{
		Message:   req.Message,
		SessionID: req.SessionID,
	}

	intentResp, err := s.ai.RecognizeIntent(intentReq)
	if err != nil {
		return nil, err
	}

	routeResp, err := s.RouteByIntent(ctx, intentResp.Intent, req)
	if err != nil {
		return nil, err
	}

	var sessionState model.SessionState
	switch intentResp.Intent {
	case model.IntentFlow:
		sessionState = model.SessionOnFlow
	case model.IntentFAQ:
		sessionState = model.SessionActive
	default:
		sessionState = model.SessionNew
	}

	resp := &model.ChatResponse{
		Reply:   routeResp.Reply,
		Type:    intentResp.Intent,
		Session: sessionState,
	}

	return resp, nil
}

func (s *ChatService) RecognizeIntent(ctx context.Context, req model.IntentRecognitionRequest) (*model.IntentRecognitionResponse, error) {
	return s.ai.RecognizeIntent(req)
}

func (s *ChatService) RouteByIntent(ctx context.Context, intent model.IntentType, req model.ChatRequest) (*model.ChatResponse, error) {
	switch intent {
	case model.IntentFAQ:
		return s.handleFAQ(ctx, req)
	case model.IntentFlow:
		return s.handleFlow(ctx, req)
	case model.IntentUnknown:
		return s.handleUnknown(ctx, req)
	default:
		return s.handleUnknown(ctx, req)
	}
}

func (s *ChatService) handleFAQ(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	return s.ai.Chat(req)
}

func (s *ChatService) handleFlow(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	return s.ai.Chat(req)
}

func (s *ChatService) handleUnknown(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	_, err := s.CreateTicket(ctx, req.UserID, req.SessionID, req.Message)
	if err != nil {
		return nil, err
	}

	return &model.ChatResponse{
		Reply: "您好，我无法准确理解您的问题。已为您创建工单，客服人员将尽快与您联系。",
		Type:  model.IntentUnknown,
	}, nil
}

func (s *ChatService) CreateTicket(ctx context.Context, userID, sessionID, description string) (*model.Ticket, error) {
	now := time.Now().Format(time.RFC3339)
	ticket := model.Ticket{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		UserID:      userID,
		Intent:      model.IntentUnknown,
		Description: description,
		Status:      model.TicketOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.ai.CreateTicket(ticket)
}
