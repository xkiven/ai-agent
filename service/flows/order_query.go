package flows

import (
	"ai-agent/model"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// ==================== 订单查询流程处理器 ====================

// extractOrderID 从用户消息中提取订单号
func extractOrderID(message string) string {
	log.Printf("[extractOrderID] input: %s", message)

	// 先尝试直接匹配 5-20 位数字
	re := regexp.MustCompile(`[0-9]{5,20}`)
	match := re.FindString(message)
	log.Printf("[extractOrderID] direct match: %s", match)
	if match != "" {
		return match
	}

	// 常见的订单号模式
	patterns := []string{
		`订单[号：:\s]*([0-9]{5,20})`,                // 订单/订单号 67890
		`(?:order[_\s]?id[：:\s=]*)([0-9]{5,20})`, // order_id=xxx
		`(?:单号|运单|快递)[：:\s]*([0-9]{5,20})`,       // 单号/运单/快递
	}

	message = strings.TrimSpace(message)

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(message)
		log.Printf("[extractOrderID] pattern: %s, matches: %v", pattern, matches)
		if len(matches) > 1 && matches[1] != "" {
			return matches[1]
		}
	}

	log.Printf("[extractOrderID] no match found")
	return ""
}

// 订单查询起始步骤
func HandleOrderQueryStart(ctx context.Context, session *model.Session, userMessage string) (string, bool, string, error) {
	// 尝试从用户消息中提取订单号
	orderID := extractOrderID(userMessage)

	log.Printf("[Flow order_query] HandleOrderQueryStart, userMessage=%s, extractedOrderID=%s", userMessage, orderID)

	if orderID != "" {
		// 用户已经提供了订单号，直接查询
		if aiClient != nil {
			log.Printf("[Flow order_query] 用户已提供订单号，直接查询, order_id=%s", orderID)
			result, err := aiClient.CallFlowTool("query_order", map[string]string{
				"order_id": orderID,
			})
			if err != nil {
				log.Printf("[Flow order_query] 调用工具失败: %v", err)
				return "查询失败，请稍后重试", true, "", nil
			}
			// 将 JSON 结果转换成自然语言
			formattedReply, err := aiClient.FormatToolResponse("query_order", result, userMessage)
			if err != nil {
				log.Printf("[Flow order_query] 格式化失败: %v", err)
				return fmt.Sprintf("订单 %s 的状态：\n%s\n\n如需其他帮助，请继续提问。", orderID, result), true, "", nil
			}
			return formattedReply, true, "", nil
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

	// 没有订单号，提示用户输入
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
		// 将 JSON 结果转换成自然语言
		formattedReply, err := aiClient.FormatToolResponse("query_order", result, userMessage)
		if err != nil {
			log.Printf("[Flow order_query] 格式化失败: %v", err)
			return fmt.Sprintf("订单 %s 的状态：\n%s\n\n如需其他帮助，请继续提问。", orderID, result), true, "", nil
		}
		return formattedReply, true, "", nil
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
