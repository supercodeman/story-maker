// server/internal/agent/token.go
package agent

// TokenManager 上下文窗口管理器
type TokenManager struct {
	MaxContextTokens int // 最大上下文 token 数
	ReserveTokens    int // 为回复预留的 token 数
}

// NewTokenManager 创建 TokenManager
// maxContext: 模型最大上下文窗口（如 8192）
// reserveForReply: 为回复预留的 token 数（如 2048）
func NewTokenManager(maxContext, reserveForReply int) *TokenManager {
	return &TokenManager{
		MaxContextTokens: maxContext,
		ReserveTokens:    reserveForReply,
	}
}

// EstimateTokens 粗略估算文本 token 数
// 中文约 1 字 = 1.5 token，英文约 4 字符 = 1 token，取混合估算
func EstimateTokens(text string) int {
	runes := []rune(text)
	cjk := 0
	ascii := 0
	for _, r := range runes {
		if r > 0x2E80 {
			cjk++
		} else {
			ascii++
		}
	}
	// 中文字符 * 1.5 + 英文字符 / 4
	return int(float64(cjk)*1.5) + (ascii+3)/4
}

// TrimHistory 按 token 预算裁剪对话历史
// 保留 system prompt 的 token 开销，从最早的消息开始丢弃
// 返回裁剪后的历史和总 token 数
func (tm *TokenManager) TrimHistory(history []ChatMessage, systemTokens int) ([]ChatMessage, int) {
	budget := tm.MaxContextTokens - tm.ReserveTokens - systemTokens
	if budget <= 0 {
		return nil, 0
	}

	// 从后往前累加，保留尽可能多的最近消息
	totalTokens := 0
	startIdx := len(history)
	for i := len(history) - 1; i >= 0; i-- {
		msgTokens := EstimateTokens(history[i].Content)
		if totalTokens+msgTokens > budget {
			break
		}
		totalTokens += msgTokens
		startIdx = i
	}

	return history[startIdx:], totalTokens
}
