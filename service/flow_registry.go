package service

import (
	"ai-agent/model"
	"ai-agent/service/flows"
	"context"
	"log"
	"time"
)

// FlowStepHandler Flow步骤处理器类型
// 返回值说明：
//   - reply: 回复给用户的文本内容
//   - done: Flow流程是否已完成
//   - nextStep: 下一步骤的名称（如果done为false）
//   - err: 处理过程中的错误信息
type FlowStepHandler func(ctx context.Context, session *model.Session, userMessage string) (reply string, done bool, nextStep string, err error)

// FlowRegistry Flow注册表：FlowID -> 步骤名 -> 处理器
type FlowRegistry map[string]map[string]FlowStepHandler

// Flow处理器注册表，定义了所有可用的Flow流程
var Flows = FlowRegistry{
	// 退货流程
	"return_goods": {
		"start":        flows.HandleReturnGoodsStart,
		"ask_order_id": flows.HandleReturnGoodsAskOrderID,
		"ask_reason":   flows.HandleReturnGoodsAskReason,
		"confirm":      flows.HandleReturnGoodsConfirm,
		"processing":   flows.HandleReturnGoodsProcessing,
	},
	// 订单查询流程
	"order_query": {
		"start":      flows.HandleOrderQueryStart,
		"processing": flows.HandleOrderQueryProcessing,
	},
	// 客户服务流程
	"customer_service": {
		"start":           flows.HandleCSStart,
		"ask_category":    flows.HandleCSAskCategory,
		"ask_description": flows.HandleCSAskDescription,
		"ask_contact":     flows.HandleCSAskContact,
	},
	// 物流查询流程
	"logistics": {
		"start": flows.HandleLogisticsStart,
		"query": flows.HandleLogisticsQuery,
	},
}

// handleFlowStateMachine 状态机处理器
// 核心逻辑：从Session中获取当前步骤，调用对应的处理器，更新状态
func (s *ChatService) handleFlowStateMachine(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.ChatResponse, error) {
	// 获取当前Flow的处理器表
	flowSteps, exists := Flows[session.FlowID]
	if !exists {
		log.Printf("[Session %s] 未找到Flow: %s", session.ID, session.FlowID)

		// 重置会话状态
		session.State = model.SessionNew
		session.FlowID = ""
		session.CurrentStep = ""
		session.FlowState = nil

		if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
			log.Printf("[Session %s] 保存失败: %v", session.ID, err)
		}

		return &model.ChatResponse{
			Reply:     "抱歉，系统错误，请重新开始。",
			Type:      model.IntentUnknown,
			Session:   session.State,
			SessionID: session.ID,
		}, nil
	}

	// 获取当前步骤的处理器
	currentStep := session.CurrentStep
	if currentStep == "" {
		currentStep = "start"
	}

	handler, exists := flowSteps[currentStep]
	if !exists {
		log.Printf("[Session %s] 未找到步骤处理器: %s，尝试从start开始", session.ID, currentStep)

		// 如果找不到当前步骤，尝试从start重新开始
		if startHandler, exists := flowSteps["start"]; exists {
			reply, done, _, err := startHandler(ctx, session, req.Message)
			if err != nil {
				return nil, err
			}

			s.addMessage(session, model.RoleUser, req.Message)
			s.addMessage(session, model.RoleAssistant, reply)

			if done {
				session.State = model.SessionComplete
				session.CurrentStep = ""
			} else {
				session.CurrentStep = "start"
			}

			session.UpdatedAt = time.Now().Format(time.RFC3339Nano)

			if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
				log.Printf("[Session %s] 保存失败: %v", session.ID, err)
			}

			return &model.ChatResponse{
				Reply:     reply,
				Type:      model.IntentFlow,
				Session:   session.State,
				SessionID: session.ID,
				FlowStep:  session.CurrentStep,
			}, nil
		}

		// 确实找不到处理器，重置会话
		session.State = model.SessionNew
		session.FlowID = ""
		session.CurrentStep = ""
		session.FlowState = nil

		if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
			log.Printf("[Session %s] 保存失败: %v", session.ID, err)
		}

		return &model.ChatResponse{
			Reply:     "流程已结束，请重新开始。",
			Type:      model.IntentUnknown,
			Session:   session.State,
			SessionID: session.ID,
		}, nil
	}

	// 调用步骤处理器
	log.Printf("[Session %s] 执行步骤: %s", session.ID, currentStep)
	reply, done, nextStep, err := handler(ctx, session, req.Message)
	if err != nil {
		log.Printf("[Session %s] 步骤执行失败: %v", session.ID, err)
		return nil, err
	}

	// 记录消息
	s.addMessage(session, model.RoleUser, req.Message)
	s.addMessage(session, model.RoleAssistant, reply)

	// 更新会话状态
	if done {
		session.State = model.SessionComplete
		session.CurrentStep = ""
		session.FlowState = nil
		log.Printf("[Session %s] Flow完成: %s", session.ID, session.FlowID)
	} else {
		session.CurrentStep = nextStep
		log.Printf("[Session %s] 步骤完成，下一步: %s", session.ID, nextStep)
	}

	session.UpdatedAt = time.Now().Format(time.RFC3339Nano)

	// 使用乐观锁保存会话
	if err := s.store.SaveWithOptimisticLock(ctx, session, 3); err != nil {
		log.Printf("[Session %s] 保存失败: %v", session.ID, err)
		return nil, err
	}

	return &model.ChatResponse{
		Reply:     reply,
		Type:      model.IntentFlow,
		Session:   session.State,
		SessionID: session.ID,
		FlowStep:  session.CurrentStep,
	}, nil
}
