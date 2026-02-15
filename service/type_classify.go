package service

import (
	"ai-agent/model"
	"log"
)

type TypeClassify struct {
	intentDefs []model.IntentDefinition
}

func NewTypeClassify(defs []model.IntentDefinition) *TypeClassify {
	enabled := make([]model.IntentDefinition, 0)
	for _, d := range defs {
		if d.Enabled {
			enabled = append(enabled, d)
		}
	}
	return &TypeClassify{intentDefs: enabled}
}

func (r *TypeClassify) Classify(intentID string) *model.DecisionResult {
	intent := r.find(intentID)
	if intent == nil {
		log.Printf("[TypeClassify] 未找到意图定义: %s, 走工单", intentID)
		return &model.DecisionResult{
			Type:       model.DecisionTicket,
			Confidence: 0.5,
		}
	}

	switch intent.Type {
	case model.IntentFlow:
		log.Printf("[TypeClassify] 意图 %s -> Flow, flow_id=%s", intentID, intent.NextFlow)
		return &model.DecisionResult{
			Type:       model.DecisionNewIntent,
			FlowID:     intent.NextFlow,
			Confidence: 0.9,
		}

	case model.IntentFAQ:
		log.Printf("[TypeClassify] 意图 %s -> FAQ/RAG", intentID)
		return &model.DecisionResult{
			Type:       model.DecisionRAG,
			Confidence: 0.9,
		}

	default:
		log.Printf("[TypeClassify] 意图 %s -> Unknown, 走工单", intentID)
		return &model.DecisionResult{
			Type:       model.DecisionTicket,
			Confidence: 0.5,
		}
	}
}

func (r *TypeClassify) find(id string) *model.IntentDefinition {
	for i := range r.intentDefs {
		if r.intentDefs[i].ID == id {
			return &r.intentDefs[i]
		}
	}
	return nil
}

func (r *TypeClassify) GetIntentDef(intentID string) *model.IntentDefinition {
	return r.find(intentID)
}
