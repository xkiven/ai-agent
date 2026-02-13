package flows

import (
	"ai-agent/model"
	"context"
	"fmt"
	"github.com/google/uuid"
)

// ==================== 客户服务流程处理器 ====================

// 客户服务起始步骤
func HandleCSStart(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	return "请问您需要什么帮助？\n1. 产品问题\n2. 订单问题\n3. 退款问题\n4. 其他", false, "ask_category", nil
}

// 获取问题分类
func HandleCSAskCategory(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	if session.FlowState == nil {
		session.FlowState = make(map[string]interface{})
	}
	session.FlowState["category"] = userMessage

	return "请详细描述您的问题或需求。", false, "ask_description", nil
}

// 获取问题描述
func HandleCSAskDescription(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	if session.FlowState == nil {
		session.FlowState = make(map[string]interface{})
	}
	session.FlowState["description"] = userMessage

	return "请留下您的联系方式（电话或邮箱），以便我们及时回复您。", false, "ask_contact", nil
}

// 获取联系方式并创建工单
func HandleCSAskContact(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	if session.FlowState == nil {
		session.FlowState = make(map[string]interface{})
	}
	session.FlowState["contact"] = userMessage

	category, _ := session.FlowState["category"].(string)
	description, _ := session.FlowState["description"].(string)

	ticketID := uuid.New().String()

	return fmt.Sprintf("感谢您提供的信息！\n\n工单号: %s\n问题分类: %s\n问题描述: %s\n联系方式: %s\n\n我们的客服人员将尽快与您联系。",
		ticketID, category, description, userMessage), true, "", nil
}
