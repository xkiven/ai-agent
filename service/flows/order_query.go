package flows

import (
	"ai-agent/model"
	"context"
	"fmt"
	"log"
)

// ==================== 订单查询流程处理器 ====================

// 订单查询起始步骤
func HandleOrderQueryStart(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	return "请提供您要查询的订单号。", false, "processing", nil
}

// 查询订单状态
func HandleOrderQueryProcessing(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	orderID := userMessage

	// 调用 Python 的 Function Calling 工具查询订单信息
	if aiClient != nil {
		log.Printf("[Flow order_query] 调用工具 query_order, order_id=%s", orderID)
		result, err := aiClient.CallFlowTool("query_order", map[string]string{
			"order_id": orderID,
		})
		if err != nil {
			log.Printf("[Flow order_query] 调用工具失败: %v", err)
			return "查询失败，请稍后重试", true, "", nil
		}
		log.Printf("[Flow order_query] 工具返回结果: %s", result)
		return fmt.Sprintf("订单 %s 的状态：\n%s\n\n如需其他帮助，请继续提问。", orderID, result), true, "", nil
	}

	// 如果没有 aiClient，使用 Mock 数据
	var status string
	switch orderID {
	case "12345":
		status = "已发货，预计明天送达\n物流单号：SF1234567890"
	case "67890":
		status = "处理中，预计3个工作日内发货"
	case "11111":
		status = "已签收，签收时间：2024-01-15 14:30"
	default:
		status = "未找到该订单，请检查订单号是否正确。"
	}

	return fmt.Sprintf("订单 %s 的状态：%s\n\n如需其他帮助，请继续提问。", orderID, status), true, "", nil
}
