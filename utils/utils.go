package utils

// 规范化用户确认输入
func NormalizeConfirm(input string) string {
	input = NormalizeString(input)
	if input == "确认" || input == "确认提交" || input == "确认退货" {
		return "confirm"
	}
	if input == "修改" || input == "重新填写" {
		return "modify"
	}
	return input
}

// 规范化字符串
func NormalizeString(s string) string {
	result := ""
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			result += string(r + 32)
		case r == '　' || r == ' ':
			result += ""
		default:
			result += string(r)
		}
	}
	return result
}

// 返回两个整数中较小的一个
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
