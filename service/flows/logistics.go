package flows

import (
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
)

var (
	aiClient *aiclient.Client
)

// SetAIClient 设置 AI 客户端（由 service 层调用）
func SetAIClient(client *aiclient.Client) {
	aiClient = client
}

// extractLogisticsOrderID 从用户消息中提取订单号（物流查询用）
func extractLogisticsOrderID(message string) string {
	// 先尝试直接匹配 5-20 位数字
	re := regexp.MustCompile(`[0-9]{5,20}`)
	match := re.FindString(message)
	if match != "" {
		return match
	}

	patterns := []string{
		`订单[号：:\s]*([0-9]{5,20})`,
		`(?:order[_\s]?id[：:\s=]*)([0-9]{5,20})`,
		`(?:单号|运单|快递)[：:\s]*([0-9]{5,20})`,
	}

	message = strings.TrimSpace(message)

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(message)
		if len(matches) > 1 && matches[1] != "" {
			return matches[1]
		}
	}

	return ""
}

// ==================== 物流查询流程处理器 ====================

// HandleLogisticsStart 物流查询起始步骤
func HandleLogisticsStart(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	// 尝试从用户消息中提取订单号
	orderID := extractLogisticsOrderID(userMessage)

	log.Printf("[Flow logistics] HandleLogisticsStart, userMessage=%s, extractedOrderID=%s", userMessage, orderID)

	if orderID != "" {
		// 用户已经提供了订单号，直接查询
		if aiClient != nil {
			log.Printf("[Flow logistics] 用户已提供订单号，直接查询, order_id=%s", orderID)
			result, err := aiClient.CallFlowTool("query_logistics", map[string]string{
				"order_id": orderID,
			})
			if err != nil {
				log.Printf("[Flow logistics] 调用工具失败: %v", err)
				return "查询失败，请稍后重试", true, "", nil
			}
			return fmt.Sprintf("订单 %s 的物流信息：\n%s\n\n如需其他帮助，请继续提问。", orderID, result), true, "", nil
		}

		// 如果没有 aiClient，使用 Mock 数据
		var logisticsInfo string
		switch orderID {
		case "12345":
			logisticsInfo = "快递公司：顺丰速运\n单号：SF1234567890\n当前状态：已到达【北京朝阳区】\n预计送达：今天下午"
		case "67890":
			logisticsInfo = "快递公司：中通快递\n单号：ZT9876543210\n当前状态：运输中【上海分拨中心】\n预计送达：明天"
		case "11111":
			logisticsInfo = "快递公司：圆通速递\n单号：YT5555666677\n当前状态：已签收\n签收人：本人"
		default:
			logisticsInfo = "未查询到该订单的物流信息，请检查订单号是否正确。"
		}
		return fmt.Sprintf("订单 %s 的物流信息：\n%s\n\n如需其他帮助，请继续提问。", orderID, logisticsInfo), true, "", nil
	}

	// 没有订单号，提示用户输入
	return "请提供您的订单号，我来帮您查询物流信息。", false, "query", nil
}

// HandleLogisticsQuery 查询物流信息
func HandleLogisticsQuery(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	orderID := userMessage

	log.Printf("[Flow logistics] 收到查询请求, order_id=%s", orderID)

	// 调用 Python 的 Function Calling 工具查询物流信息
	if aiClient != nil {
		log.Printf("[Flow logistics] 调用工具 query_logistics")
		result, err := aiClient.CallFlowTool("query_logistics", map[string]string{
			"order_id": orderID,
		})
		if err != nil {
			log.Printf("[Flow logistics] 调用工具失败: %v", err)
			return "查询失败，请稍后重试", true, "", nil
		}
		log.Printf("[Flow logistics] 工具返回结果: %s", result)
		return fmt.Sprintf("订单 %s 的物流信息：\n%s\n\n如需其他帮助，请继续提问。", orderID, result), true, "", nil
	}

	log.Printf("[Flow logistics] aiClient 为空，使用 Mock 数据")

	// 如果没有 aiClient，使用 Mock 数据
	var logisticsInfo string
	switch orderID {
	case "12345":
		logisticsInfo = "快递公司：顺丰速运\n单号：SF1234567890\n当前状态：已到达【北京朝阳区】\n预计送达：今天下午"
	case "67890":
		logisticsInfo = "快递公司：中通快递\n单号：ZT9876543210\n当前状态：运输中【上海分拨中心】\n预计送达：明天"
	case "11111":
		logisticsInfo = "快递公司：圆通速递\n单号：YT5555666677\n当前状态：已签收\n签收人：本人"
	default:
		logisticsInfo = "未查询到该订单的物流信息，请检查订单号是否正确。"
	}

	return fmt.Sprintf("订单 %s 的物流信息：\n%s\n\n如需其他帮助，请继续提问。", orderID, logisticsInfo), true, "", nil
}
