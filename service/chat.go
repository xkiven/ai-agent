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

// ChatService 聊天服务结构体
type ChatService struct {
	ai    *aiclient.Client
	store *dao.RedisStore
}

// NewChatService 创建ChatService实例
func NewChatService(ai *aiclient.Client, store *dao.RedisStore) *ChatService {
	return &ChatService{
		ai:    ai,
		store: store,
	}
}

// HandleMessage 处理用户消息的主入口方法
// 这是整个聊天服务的核心入口点
func (s *ChatService) HandleMessage(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	// 如果前端没有提供SessionID，自动生成一个（支持无状态客户端）
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
		log.Printf("[Session %s] 自动生成SessionID", req.SessionID)
	}

	// 获取或创建会话
	session, err := s.getOrCreateSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		log.Printf("[Session %s] 获取会话失败: %v", req.SessionID, err)
		return nil, err
	}

	// 记录当前会话状态，便于调试
	log.Printf("[Session %s] 状态: %s, FlowID: %s, 步骤: %s, 消息数: %d, Version: %d",
		req.SessionID, session.State, session.FlowID, session.CurrentStep, len(session.Messages), session.Version)

	// 判断处理流程：Flow模式 or 正常模式
	if session.State == model.SessionOnFlow && session.FlowID != "" {
		return s.handleFlowStateMachine(ctx, req, session)
	}

	return s.handleNormalFlow(ctx, req, session)
}

// handleNormalFlow 正常（非Flow）流程处理
// 核心逻辑：识别用户意图，根据意图路由到不同处理器
func (s *ChatService) handleNormalFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.ChatResponse, error) {
	// 构造意图识别请求
	intentReq := model.IntentRecognitionRequest{
		Message:   req.Message,
		SessionID: req.SessionID,
		History:   s.getRecentHistory(session, 10),
	}

	log.Printf("[Session %s] 发送意图识别请求，消息: %s, 历史消息数: %d",
		req.SessionID, req.Message, len(intentReq.History))

	// 调用AI进行意图识别
	intentResp, err := s.ai.RecognizeIntent(intentReq)
	if err != nil {
		log.Printf("[Session %s] 意图识别失败: %v", req.SessionID, err)
		return nil, err
	}

	log.Printf("[Session %s] 识别到意图: %s (置信度: %.2f)", req.SessionID, intentResp.Intent, intentResp.Confidence)

	// 获取用于聊天对话的历史记录
	historyForChat := s.getRecentHistory(session, 10)

	// 根据意图路由到不同的处理器
	routeResp, err := s.RouteByIntent(ctx, intentResp.Intent, req, historyForChat, intentResp.FlowID)
	if err != nil {
		log.Printf("[Session %s] 路由失败: %v", req.SessionID, err)
		return nil, err
	}

	// 截取回复内容用于日志（避免日志过长）
	replyPreview := routeResp.Reply
	if len(replyPreview) > 100 {
		replyPreview = replyPreview[:100] + "..."
	}
	log.Printf("[Session %s] 路由响应: %s", req.SessionID, replyPreview)

	// 更新会话状态
	switch intentResp.Intent {
	case model.IntentFlow:
		session.State = model.SessionOnFlow
		session.FlowID = intentResp.FlowID
		session.CurrentStep = "start"
		if session.FlowState == nil {
			session.FlowState = make(map[string]interface{})
		}
		log.Printf("[Session %s] 进入Flow模式: %s", req.SessionID, session.FlowID)

	case model.IntentFAQ:
		session.State = model.SessionActive
		session.FlowID = ""
		session.CurrentStep = ""

	case model.IntentUnknown:
		session.State = model.SessionNew
		session.FlowID = ""
		session.CurrentStep = ""

	default:
		session.State = model.SessionActive
		session.FlowID = ""
		session.CurrentStep = ""
	}

	// 更新时间戳并添加消息
	session.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	s.addMessage(session, model.RoleUser, req.Message)
	s.addMessage(session, model.RoleAssistant, routeResp.Reply)

	// 使用乐观锁保存会话
	if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
		log.Printf("[Session %s] 保存失败: %v", session.ID, err)
		return nil, err
	}

	log.Printf("[Session %s] 处理完成，消息数: %d, 状态: %s", session.ID, len(session.Messages), session.State)

	return &model.ChatResponse{
		Reply:     routeResp.Reply,
		Type:      intentResp.Intent,
		Session:   session.State,
		SessionID: session.ID,
		FlowStep:  session.CurrentStep,
	}, nil
}

// getOrCreateSession 获取或创建会话
func (s *ChatService) getOrCreateSession(ctx context.Context, sessionID, userID string) (*model.Session, error) {
	// 尝试从Redis获取现有会话
	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		log.Printf("[Session %s] 获取会话失败: %v", sessionID, err)
		return nil, err
	}

	// 如果会话存在，直接返回
	if session != nil {
		return session, nil
	}

	// 创建新会话
	now := time.Now().Format(time.RFC3339Nano)
	newSession := &model.Session{
		ID:        sessionID,
		UserID:    userID,
		State:     model.SessionNew,
		Messages:  []model.Message{},
		FlowState: make(map[string]interface{}),
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	log.Printf("[Session %s] 创建新会话", sessionID)

	// 保存新会话
	if err := s.store.Save(ctx, newSession); err != nil {
		log.Printf("[Session %s] 保存新会话失败: %v", sessionID, err)
		return nil, err
	}

	return newSession, nil
}

// addMessage 添加消息到会话
func (s *ChatService) addMessage(session *model.Session, role model.MessageRole, content string) {
	session.Messages = append(session.Messages, model.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339Nano),
	})

	// 限制消息数量，最多保留最近100条
	if len(session.Messages) > 100 {
		session.Messages = session.Messages[len(session.Messages)-100:]
	}
}

// getRecentHistory 获取最近的历史消息
func (s *ChatService) getRecentHistory(session *model.Session, count int) []model.Message {
	if len(session.Messages) <= count {
		return session.Messages
	}
	return session.Messages[len(session.Messages)-count:]
}

// GetSessionHistory 获取会话历史
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

// ClearSession 清除会话
func (s *ChatService) ClearSession(ctx context.Context, sessionID string) error {
	return s.store.Delete(ctx, sessionID)
}

// RecognizeIntent 识别用户意图
func (s *ChatService) RecognizeIntent(ctx context.Context, req model.IntentRecognitionRequest) (*model.IntentRecognitionResponse, error) {
	return s.ai.RecognizeIntent(req)
}

// RouteByIntent 根据意图路由到不同的处理器
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

// handleFAQ 处理FAQ类型的问题
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

// handleFlow 处理Flow类型的请求
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

// handleUnknown 处理无法识别的意图
func (s *ChatService) handleUnknown(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	log.Printf("[handleUnknown] session=%s, message=%s", req.SessionID, req.Message)

	// 创建工单
	_, err := s.CreateTicket(ctx, req.UserID, req.SessionID, req.Message)
	if err != nil {
		log.Printf("[Session %s] 创建工单失败: %v", req.SessionID, err)
		return nil, err
	}

	return &model.ChatResponse{
		Reply: "您好，我无法准确理解您的问题。已为您创建工单，客服人员将尽快与您联系。",
		Type:  model.IntentUnknown,
	}, nil
}

// CreateTicket 创建工单
func (s *ChatService) CreateTicket(ctx context.Context, userID, sessionID, description string) (*model.Ticket, error) {
	now := time.Now().Format(time.RFC3339Nano)
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

	log.Printf("[Session %s] 创建工单: %s", sessionID, ticket.ID)

	return s.ai.CreateTicket(ticket)
}

// Ping 检查服务健康状态
func (s *ChatService) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}
