package service

import (
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ChatService struct {
	ai       *aiclient.Client
	sessions map[string]*model.Session
	mu       sync.RWMutex
}

func NewChatService(ai *aiclient.Client) *ChatService {
	return &ChatService{
		ai:       ai,
		sessions: make(map[string]*model.Session),
	}
}

func (s *ChatService) HandleMessage(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	session := s.getOrCreateSession(req.SessionID, req.UserID)

	log.Printf("[Session %s] 当前消息数: %d", req.SessionID, len(session.Messages))

	intentReq := model.IntentRecognitionRequest{
		Message:   req.Message,
		SessionID: req.SessionID,
		History:   s.getRecentHistory(session, 10),
	}

	log.Printf("[Session %s] 发送的历史记录数: %d", req.SessionID, len(intentReq.History))

	intentResp, err := s.ai.RecognizeIntent(intentReq)
	if err != nil {
		return nil, err
	}

	historyForChat := s.getRecentHistory(session, 10)
	routeResp, err := s.RouteByIntent(ctx, intentResp.Intent, req, historyForChat)
	if err != nil {
		return nil, err
	}

	log.Printf("[Session %s] 处理后消息数: %d", req.SessionID, len(session.Messages))

	switch intentResp.Intent {
	case model.IntentFlow:
		session.State = model.SessionOnFlow
		session.FlowID = intentResp.FlowID
	case model.IntentFAQ:
		session.State = model.SessionActive
	case model.IntentUnknown:
		session.State = model.SessionNew
	default:
		session.State = model.SessionActive
	}
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	s.addMessage(session, model.RoleUser, req.Message)
	s.addMessage(session, model.RoleAssistant, routeResp.Reply)

	resp := &model.ChatResponse{
		Reply:   routeResp.Reply,
		Type:    intentResp.Intent,
		Session: session.State,
	}

	return resp, nil
}

func (s *ChatService) getOrCreateSession(sessionID, userID string) *model.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		return session
	}

	now := time.Now().Format(time.RFC3339)
	newSession := &model.Session{
		ID:        sessionID,
		UserID:    userID,
		State:     model.SessionNew,
		Messages:  []model.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.sessions[sessionID] = newSession
	return newSession
}

func (s *ChatService) addMessage(session *model.Session, role model.MessageRole, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session.Messages = append(session.Messages, model.Message{
		Role:    role,
		Content: content,
	})

	if len(session.Messages) > 100 {
		session.Messages = session.Messages[len(session.Messages)-100:]
	}
}

func (s *ChatService) getRecentHistory(session *model.Session, count int) []model.Message {
	if len(session.Messages) <= count {
		return session.Messages
	}
	return session.Messages[len(session.Messages)-count:]
}

func (s *ChatService) GetSessionHistory(sessionID string) (*model.SessionHistoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return &model.SessionHistoryResponse{
			SessionID: sessionID,
			Messages:  []model.Message{},
			Count:     0,
		}, nil
	}

	return &model.SessionHistoryResponse{
		SessionID: sessionID,
		Messages:  session.Messages,
		Count:     len(session.Messages),
	}, nil
}

func (s *ChatService) ClearSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

func (s *ChatService) RecognizeIntent(ctx context.Context, req model.IntentRecognitionRequest) (*model.IntentRecognitionResponse, error) {
	return s.ai.RecognizeIntent(req)
}

func (s *ChatService) RouteByIntent(ctx context.Context, intent model.IntentType, req model.ChatRequest, history []model.Message) (*model.ChatResponse, error) {
	switch intent {
	case model.IntentFAQ:
		return s.handleFAQ(ctx, req, history)
	case model.IntentFlow:
		return s.handleFlow(ctx, req, history)
	case model.IntentUnknown:
		return s.handleUnknown(ctx, req)
	default:
		return s.handleUnknown(ctx, req)
	}
}

func (s *ChatService) handleFAQ(ctx context.Context, req model.ChatRequest, history []model.Message) (*model.ChatResponse, error) {
	chatReq := model.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		UserID:    req.UserID,
		History:   history,
	}
	return s.ai.Chat(chatReq)
}

func (s *ChatService) handleFlow(ctx context.Context, req model.ChatRequest, history []model.Message) (*model.ChatResponse, error) {
	chatReq := model.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		UserID:    req.UserID,
		History:   history,
	}
	return s.ai.Chat(chatReq)
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
