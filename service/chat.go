package service

import (
	"ai-agent/dao"
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

type ChatService struct {
	ai    *aiclient.Client
	store *dao.RedisStore
}

func NewChatService(ai *aiclient.Client, store *dao.RedisStore) *ChatService {
	return &ChatService{
		ai:    ai,
		store: store,
	}
}

func (s *ChatService) HandleMessage(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	session, err := s.getOrCreateSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

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
	routeResp, err := s.RouteByIntent(ctx, intentResp.Intent, req, historyForChat, intentResp.FlowID)
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

	if err := s.store.Save(ctx, session); err != nil {
		log.Printf("[Session %s] 保存失败: %v", req.SessionID, err)
	}

	resp := &model.ChatResponse{
		Reply:   routeResp.Reply,
		Type:    intentResp.Intent,
		Session: session.State,
	}

	return resp, nil
}

func (s *ChatService) getOrCreateSession(ctx context.Context, sessionID, userID string) (*model.Session, error) {
	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		return session, nil
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

	if err := s.store.Save(ctx, newSession); err != nil {
		return nil, err
	}

	return newSession, nil
}

func (s *ChatService) addMessage(session *model.Session, role model.MessageRole, content string) {
	session.Messages = append(session.Messages, model.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339),
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

func (s *ChatService) GetSessionHistory(ctx context.Context, sessionID string) (*model.SessionHistoryResponse, error) {
	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session == nil {
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

func (s *ChatService) ClearSession(ctx context.Context, sessionID string) error {
	return s.store.Delete(ctx, sessionID)
}

func (s *ChatService) RecognizeIntent(ctx context.Context, req model.IntentRecognitionRequest) (*model.IntentRecognitionResponse, error) {
	return s.ai.RecognizeIntent(req)
}

func (s *ChatService) RouteByIntent(ctx context.Context, intent model.IntentType, req model.ChatRequest, history []model.Message, flowID string) (*model.ChatResponse, error) {
	switch intent {
	case model.IntentFAQ:
		return s.handleFAQ(ctx, req, history)
	case model.IntentFlow:
		return s.handleFlow(ctx, req, history, flowID)
	case model.IntentUnknown:
		return s.handleUnknown(ctx, req)
	default:
		return s.handleUnknown(ctx, req)
	}
}

func (s *ChatService) handleFAQ(ctx context.Context, req model.ChatRequest, history []model.Message) (*model.ChatResponse, error) {
	log.Printf("[handleFAQ] session=%s, history_count=%d", req.SessionID, len(history))
	chatReq := model.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		UserID:    req.UserID,
		History:   history,
		Intent:    model.IntentFAQ,
		FlowID:    "faq_response",
	}
	return s.ai.Chat(chatReq)
}

func (s *ChatService) handleFlow(ctx context.Context, req model.ChatRequest, history []model.Message, flowID string) (*model.ChatResponse, error) {
	log.Printf("[handleFlow] session=%s, history_count=%d, flow_id=%s", req.SessionID, len(history), flowID)
	chatReq := model.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		UserID:    req.UserID,
		History:   history,
		Intent:    model.IntentFlow,
		FlowID:    flowID,
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

func (s *ChatService) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}
