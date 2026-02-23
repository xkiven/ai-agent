package service

import (
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"log"
)

type DecisionLayer struct {
	aiClient     *aiclient.Client
	typeClassify *TypeClassify
}

// 创建决策层
func NewDecisionLayer(aiClient *aiclient.Client, intentDefs []model.IntentDefinition) *DecisionLayer {
	return &DecisionLayer{
		aiClient:     aiClient,
		typeClassify: NewTypeClassify(intentDefs),
	}
}

// Decide 核心决策方法
func (d *DecisionLayer) Decide(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.DecisionResult, error) {
	log.Printf("[DecisionLayer] session=%s, state=%s, current_step=%s",
		session.ID, session.State, session.CurrentStep)

	// 如果会话已完成，直接重置为新会话
	if session.State == model.SessionComplete {
		log.Printf("[DecisionLayer] 会话已完成，重置为新会话")
		session.State = model.SessionNew
		session.FlowID = ""
		session.CurrentStep = ""
		session.FlowState = nil
		return d.handleNotOnFlow(ctx, req, session)
	}

	// 场景1: 已在 Flow 中
	if session.State == model.SessionOnFlow {
		return d.handleOnFlow(ctx, req, session)
	}

	// 场景2: 不在 Flow 中
	return d.handleNotOnFlow(ctx, req, session)
}

// handleOnFlow 处理已在 Flow 中的情况
func (d *DecisionLayer) handleOnFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.DecisionResult, error) {
	log.Printf("[DecisionLayer] OnFlow, checking interrupt...")

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
		return &model.DecisionResult{
			Type:       model.DecisionContinueFlow,
			Confidence: 1.0,
		}, nil
	}

	if resp.ShouldInterrupt {
		log.Printf("[DecisionLayer] Flow被打断，重新决策 intent=%s", resp.NewIntent)
		return d.handleNotOnFlow(ctx, req, session)
	}

	log.Printf("[DecisionLayer] 继续当前 Flow")
	return &model.DecisionResult{
		Type:       model.DecisionContinueFlow,
		FlowID:     session.FlowID,
		Confidence: resp.Confidence,
	}, nil
}

// handleNotOnFlow 处理不在 Flow 中的情况
func (d *DecisionLayer) handleNotOnFlow(ctx context.Context, req model.ChatRequest, session *model.Session) (*model.DecisionResult, error) {
	log.Printf("[DecisionLayer] NotOnFlow, 调用 Python 做 Intent 识别")

	intentReq := model.IntentRecognitionRequest{
		SessionID: session.ID,
		Message:   req.Message,
		History:   session.Messages,
	}

	intentResp, err := d.aiClient.RecognizeIntent(intentReq)
	if err != nil {
		log.Printf("[DecisionLayer] RecognizeIntent error: %v", err)
		return nil, err
	}

	log.Printf("[DecisionLayer] Intent识别结果: intent=%s, confidence=%.2f, flow_id=%s",
		intentResp.Intent, intentResp.Confidence, intentResp.FlowID)

	// 使用TypeClassify进行类型路由
	// 当 intent 是 "faq" 类型时，直接走 RAG 流程
	if intentResp.Intent == "faq" {
		log.Printf("[DecisionLayer] 意图 %s -> RAG", intentResp.Intent)
		return &model.DecisionResult{
			Type:       model.DecisionRAG,
			Confidence: intentResp.Confidence,
			Reply:      intentResp.Reply,
		}, nil
	}

	// 当 intent 是 "flow" 类型时，使用 flow_id（优先用 Python 返回的，否则用当前会话的）
	if intentResp.Intent == "flow" {
		flowID := intentResp.FlowID
		if flowID == "" {
			flowID = session.FlowID
		}
		if flowID != "" {
			log.Printf("[DecisionLayer] 意图 %s -> Flow, flow_id=%s", intentResp.Intent, flowID)
			return &model.DecisionResult{
				Type:       model.DecisionNewIntent,
				FlowID:     flowID,
				Confidence: intentResp.Confidence,
				Reply:      intentResp.Reply,
			}, nil
		}
	}

	result := d.typeClassify.Classify(string(intentResp.Intent))
	result.Confidence = intentResp.Confidence
	result.Reply = intentResp.Reply
	if intentResp.FlowID != "" {
		result.FlowID = intentResp.FlowID
	}

	return result, nil
}
