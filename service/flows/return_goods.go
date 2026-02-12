package flows

import (
	"ai-agent/model"
	"ai-agent/utils"
	"context"
	"fmt"
)

// ==================== 退货流程处理器 ====================

// 退货流程起始步骤
func HandleReturnGoodsStart(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	return "欢迎使用退货服务！请提供您要退货的订单号。", false, "ask_order_id", nil
}

// 获取订单号
func HandleReturnGoodsAskOrderID(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	// 保存订单号到FlowState
	if session.FlowState == nil {
		session.FlowState = make(map[string]interface{})
	}
	session.FlowState["order_id"] = userMessage

	return fmt.Sprintf("订单号 %s 已记录。请问退货原因是什么？\n1. 商品质量问题\n2. 收到商品与描述不符\n3. 尺寸/颜色不合适\n4. 其他原因", userMessage), false, "ask_reason", nil
}

// 获取退货原因
func HandleReturnGoodsAskReason(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	// 保存退货原因
	if session.FlowState == nil {
		session.FlowState = make(map[string]interface{})
	}
	session.FlowState["reason"] = userMessage

	return fmt.Sprintf("退货原因: %s\n\n请确认以下信息是否正确？\n订单号: %s\n退货原因: %s\n\n回复【确认】提交退货申请，或回复【修改】重新填写。",
		userMessage, session.FlowState["order_id"], userMessage), false, "confirm", nil
}

// 确认退货信息
func HandleReturnGoodsConfirm(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	// 检查用户是否确认
	userMessage = utils.NormalizeConfirm(userMessage)

	if userMessage == "confirm" || userMessage == "yes" || userMessage == "y" {
		// 检查必要信息
		if session.FlowState == nil {
			return "抱歉，信息不完整，请重新开始退货流程。", true, "", nil
		}

		orderID, ok := session.FlowState["order_id"].(string)
		if !ok || orderID == "" {
			return "抱歉，订单号信息丢失，请重新开始。", true, "", nil
		}

		reason, _ := session.FlowState["reason"].(string)

		return fmt.Sprintf("退货申请已提交！\n\n订单号: %s\n退货原因: %s\n状态: 处理中\n\n我们的客服人员将在24小时内与您联系。",
			orderID, reason), true, "", nil

	} else if userMessage == "modify" || userMessage == "modify_order_id" {
		// 重新填写订单号
		session.FlowState["order_id"] = ""
		return "好的，请重新提供订单号。", false, "ask_order_id", nil

	} else {
		// 用户没有明确确认，引导重新确认
		return "请回复【确认】提交退货申请，或回复【修改】重新填写信息。", false, "confirm", nil
	}
}

// 退货处理中
func HandleReturnGoodsProcessing(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	return "您的退货申请正在处理中，请耐心等待客服人员联系您。", true, "", nil
}
