package service

import (
	"ai-agent/dao"
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"fmt"
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

// FlowStepResponse Flow步骤处理响应
type FlowStepResponse struct {
	Reply      string                 `json:"reply"`
	NextStep   string                 `json:"next_step"`
	FlowState  map[string]interface{} `json:"flow_state,omitempty"`
	IsComplete bool                   `json:"is_complete"`
}

func (s *ChatService) HandleMessage(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	// 如果前端没有提供SessionID，自动生成一个
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	session, err := s.getOrCreateSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	log.Printf("[Session %s] 当前消息数: %d, 状态: %s, FlowID: %s, 当前步骤: %s",
		req.SessionID, len(session.Messages), session.State, session.FlowID, session.CurrentStep)

	// 如果是Flow会话，先检查当前步骤
	if session.State == model.SessionOnFlow && session.FlowID != "" {
		return s.handleFlowStep(ctx, req, session)
	}

	// 正常意图识别流程
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

	// 更新会话状态
	switch intentResp.Intent {
	case model.IntentFlow:
		session.State = model.SessionOnFlow
		session.FlowID = intentResp.FlowID
		session.CurrentStep = "start" // 设置初始步骤
		if session.FlowState == nil {
			session.FlowState = make(map[string]interface{})
		}
	case model.IntentFAQ:
		session.State = model.SessionActive
		// 重置Flow状态
		session.FlowID = ""
		session.CurrentStep = ""
		session.FlowState = nil
	case model.IntentUnknown:
		session.State = model.SessionNew
	default:
		session.State = model.SessionActive
	}

	session.UpdatedAt = time.Now().Format(time.RFC3339)
	s.addMessage(session, model.RoleUser, req.Message)
	s.addMessage(session, model.RoleAssistant, routeResp.Reply)

	// 使用乐观锁保存session
	if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
		log.Printf("[Session %s] 保存失败: %v", req.SessionID, err)
		return nil, err
	}

	resp := &model.ChatResponse{
		Reply:     routeResp.Reply,
		Type:      intentResp.Intent,
		Session:   session.State,
		SessionID: session.ID,
		FlowStep:  session.CurrentStep,
	}

	log.Printf("[Session %s] 响应: 类型=%s, 状态=%s, 步骤=%s", req.SessionID, resp.Type, resp.Session, resp.FlowStep)

	return resp, nil
}

// handleFlowStep 处理Flow会话的步骤
func (s *ChatService) handleFlowStep(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.ChatResponse, error) {
	log.Printf("[Flow处理] Session=%s, 当前步骤=%s, FlowID=%s", req.SessionID, session.CurrentStep, session.FlowID)

	// 根据当前FlowID和步骤处理用户输入
	flowResp, err := s.processFlowStep(ctx, req, session)
	if err != nil {
		log.Printf("[Flow处理] Session=%s 处理失败: %v", req.SessionID, err)
		return nil, err
	}

	// 更新会话状态
	session.CurrentStep = flowResp.NextStep
	if flowResp.FlowState != nil {
		session.FlowState = flowResp.FlowState
	}

	// 检查Flow是否完成
	if flowResp.IsComplete {
		session.State = model.SessionComplete
		session.FlowID = ""
		session.CurrentStep = ""
		session.FlowState = nil
	}

	session.UpdatedAt = time.Now().Format(time.RFC3339)
	s.addMessage(session, model.RoleUser, req.Message)
	s.addMessage(session, model.RoleAssistant, flowResp.Reply)

	// 保存session
	if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
		log.Printf("[Session %s] 保存失败: %v", req.SessionID, err)
		return nil, err
	}

	resp := &model.ChatResponse{
		Reply:     flowResp.Reply,
		Type:      model.IntentFlow,
		Session:   session.State,
		SessionID: session.ID,
		FlowStep:  session.CurrentStep,
	}

	log.Printf("[Flow处理] Session=%s 完成: 下一步骤=%s, 完成=%v", req.SessionID, flowResp.NextStep, flowResp.IsComplete)

	return resp, nil
}

// processFlowStep 处理Flow步骤的具体逻辑
func (s *ChatService) processFlowStep(ctx context.Context, req model.ChatRequest, session *model.Session) (*FlowStepResponse, error) {
	// 根据FlowID路由到不同的流程处理器
	switch session.FlowID {
	case "order_query":
		return s.processOrderQueryFlow(ctx, req, session)
	case "customer_service":
		return s.processCustomerServiceFlow(ctx, req, session)
	case "product_info":
		return s.processProductInfoFlow(ctx, req, session)
	default:
		return s.processGenericFlow(ctx, req, session)
	}
}

// processOrderQueryFlow 处理订单查询流程
func (s *ChatService) processOrderQueryFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*FlowStepResponse, error) {
	switch session.CurrentStep {
	case "start":
		return &FlowStepResponse{
			Reply:      "请问您要查询哪个订单？请提供订单号。",
			NextStep:   "waiting_order_id",
			FlowState:  session.FlowState,
			IsComplete: false,
		}, nil

	case "waiting_order_id":
		// 处理订单ID
		if session.FlowState == nil {
			session.FlowState = make(map[string]interface{})
		}
		session.FlowState["order_id"] = req.Message

		return &FlowStepResponse{
			Reply:      "正在查询订单信息，请稍等...",
			NextStep:   "processing",
			FlowState:  session.FlowState,
			IsComplete: false,
		}, nil

	case "processing":
		// 模拟订单查询逻辑
		orderID, ok := session.FlowState["order_id"].(string)
		if !ok {
			orderID = "未知"
		}

		// 模拟不同的订单状态
		var status string
		switch orderID {
		case "12345":
			status = "已发货，预计明天送达"
		case "67890":
			status = "处理中，预计3个工作日内发货"
		default:
			status = "订单不存在或已取消"
		}

		return &FlowStepResponse{
			Reply:      fmt.Sprintf("订单 %s 的状态是：%s。还有什么可以帮您的吗？", orderID, status),
			NextStep:   "complete",
			FlowState:  session.FlowState,
			IsComplete: true,
		}, nil

	default:
		return &FlowStepResponse{
			Reply:      "流程步骤异常，已重置会话",
			NextStep:   "start",
			FlowState:  make(map[string]interface{}),
			IsComplete: false,
		}, nil
	}
}

// processCustomerServiceFlow 处理客户服务流程
func (s *ChatService) processCustomerServiceFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*FlowStepResponse, error) {
	switch session.CurrentStep {
	case "start":
		return &FlowStepResponse{
			Reply:      "请问您需要什么帮助？\n1. 产品问题\n2. 订单问题\n3. 退款问题\n4. 其他",
			NextStep:   "waiting_category",
			FlowState:  session.FlowState,
			IsComplete: false,
		}, nil

	case "waiting_category":
		if session.FlowState == nil {
			session.FlowState = make(map[string]interface{})
		}
		session.FlowState["category"] = req.Message

		return &FlowStepResponse{
			Reply:      "请详细描述您的问题，我会尽快为您处理。",
			NextStep:   "waiting_description",
			FlowState:  session.FlowState,
			IsComplete: false,
		}, nil

	case "waiting_description":
		session.FlowState["description"] = req.Message

		// 创建工单
		ticket, err := s.CreateTicket(ctx, session.UserID, session.ID, fmt.Sprintf("分类: %s, 问题: %s",
			session.FlowState["category"], session.FlowState["description"]))
		if err != nil {
			return &FlowStepResponse{
				Reply:      "创建工单失败，请稍后重试",
				NextStep:   "complete",
				FlowState:  session.FlowState,
				IsComplete: true,
			}, nil
		}

		return &FlowStepResponse{
			Reply:      fmt.Sprintf("已为您创建工单（ID: %s），客服人员将尽快与您联系。", ticket.ID),
			NextStep:   "complete",
			FlowState:  session.FlowState,
			IsComplete: true,
		}, nil

	default:
		return &FlowStepResponse{
			Reply:      "流程步骤异常，已重置会话",
			NextStep:   "start",
			FlowState:  make(map[string]interface{}),
			IsComplete: false,
		}, nil
	}
}

// processProductInfoFlow 处理产品信息查询流程
func (s *ChatService) processProductInfoFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*FlowStepResponse, error) {
	switch session.CurrentStep {
	case "start":
		return &FlowStepResponse{
			Reply:      "请问您想了解哪个产品？请提供产品名称或型号。",
			NextStep:   "waiting_product",
			FlowState:  session.FlowState,
			IsComplete: false,
		}, nil

	case "waiting_product":
		if session.FlowState == nil {
			session.FlowState = make(map[string]interface{})
		}
		session.FlowState["product"] = req.Message

		// 模拟产品信息查询
		productInfo := map[string]string{
			"iPhone 15":   "最新款iPhone，配备A17芯片，4800万像素摄像头",
			"MacBook Pro": "专业级笔记本电脑，M3芯片，Retina显示屏",
			"AirPods Pro": "主动降噪无线耳机，空间音频功能",
		}

		info, exists := productInfo[req.Message]
		if !exists {
			info = "抱歉，没有找到该产品的详细信息"
		}

		return &FlowStepResponse{
			Reply:      fmt.Sprintf("%s 的信息：%s。还需要了解其他产品吗？", req.Message, info),
			NextStep:   "complete",
			FlowState:  session.FlowState,
			IsComplete: true,
		}, nil

	default:
		return &FlowStepResponse{
			Reply:      "流程步骤异常，已重置会话",
			NextStep:   "start",
			FlowState:  make(map[string]interface{}),
			IsComplete: false,
		}, nil
	}
}

// processGenericFlow 处理通用流程
func (s *ChatService) processGenericFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*FlowStepResponse, error) {
	// 简单的问答式流程
	return &FlowStepResponse{
		Reply:      "这是一个通用流程处理，请提供更多信息。",
		NextStep:   "complete",
		FlowState:  session.FlowState,
		IsComplete: true,
	}, nil
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
		FlowState: make(map[string]interface{}),
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
