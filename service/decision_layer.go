package service

import (
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"log"
)

type DecisionLayer struct {
	aiClient *aiclient.Client
}

// 创建决策层
func NewDecisionLayer(aiClient *aiclient.Client) *DecisionLayer {
	return &DecisionLayer{
		aiClient: aiClient,
	}
}

// Decide 核心决策方法
func (d *DecisionLayer) Decide(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.DecisionResult, error) {
	log.Printf("[DecisionLayer] session=%s, state=%s, current_step=%s",
		session.ID, session.State, session.CurrentStep)

	// 场景1: 已在 Flow 中
	if session.State == model.SessionOnFlow {
		return d.handleOnFlow(req, session)
	}

	// 场景2: 不在 Flow 中
	return d.handleNotOnFlow(req, session)
}

// handleOnFlow 处理已在 Flow 中的情况
func (d *DecisionLayer) handleOnFlow(req model.ChatRequest, session *model.Session) (*model.DecisionResult, error) {
	log.Printf("[DecisionLayer] OnFlow, checking interrupt...")

	// 调用 Python 判断是否打断 Flow
	checkReq := model.InterruptCheckRequest{
		SessionID:   session.ID,
		FlowID:      session.FlowID,
		CurrentStep: session.CurrentStep,
		UserMessage: req.Message,
		FlowState:   session.FlowState,
	}

	resp, err := d.aiClient.CheckFlowInterrupt(checkReq)
	if err != nil {
		log.Printf("[DecisionLayer] CheckFlowInterrupt error: %v, 使用本地处理", err)
		// 如果调用失败，默认继续 Flow
		return &model.DecisionResult{
			Type:       model.DecisionContinueFlow,
			Confidence: 1.0,
		}, nil
	}

	if resp.ShouldInterrupt {
		log.Printf("[DecisionLayer] Flow被打断，重新决策 intent=%s", resp.NewIntent)
		// 打断 Flow，重新走 Intent 决策
		return d.handleNotOnFlow(req, session)
	}

	log.Printf("[DecisionLayer] 继续当前 Flow")
	// 继续当前 Flow
	return &model.DecisionResult{
		Type:       model.DecisionContinueFlow,
		FlowID:     session.FlowID,
		Confidence: resp.Confidence,
	}, nil
}

// handleNotOnFlow 处理不在 Flow 中的情况
func (d *DecisionLayer) handleNotOnFlow(req model.ChatRequest, session *model.Session) (*model.DecisionResult, error) {
	log.Printf("[DecisionLayer] NotOnFlow, 调用 Python 做 Intent 识别")

	// 调用 Python Intent 识别
	intentReq := model.IntentRecognitionRequest{
		SessionID: session.ID,
		Message:   req.Message,
		History:   req.History,
	}

	intentResp, err := d.aiClient.RecognizeIntent(intentReq)
	if err != nil {
		log.Printf("[DecisionLayer] RecognizeIntent error: %v", err)
		return nil, err
	}

	log.Printf("[DecisionLayer] Intent识别结果: intent=%s, confidence=%.2f",
		intentResp.Intent, intentResp.Confidence)

	// 根据 Intent 类型决策
	switch intentResp.Intent {
	case model.IntentFlow:
		return &model.DecisionResult{
			Type:       model.DecisionNewIntent,
			FlowID:     intentResp.FlowID,
			Reply:      intentResp.Reply,
			Confidence: intentResp.Confidence,
		}, nil

	case model.IntentFAQ:
		return &model.DecisionResult{
			Type:       model.DecisionRAG,
			Reply:      intentResp.Reply,
			Confidence: intentResp.Confidence,
		}, nil

	case model.IntentUnknown:
		return &model.DecisionResult{
			Type:       model.DecisionTicket,
			Reply:      intentResp.Reply,
			Confidence: intentResp.Confidence,
		}, nil

	default:
		return &model.DecisionResult{
			Type:       model.DecisionNewIntent,
			FlowID:     string(intentResp.Intent),
			Reply:      intentResp.Reply,
			Confidence: intentResp.Confidence,
		}, nil
	}
}
