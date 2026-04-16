// server/internal/agent/token_budget.go
package agent

// TokenBudget 全局 Token 预算管理器
// 根据模型上下文窗口大小，按比例分配各模块的 token 预算
type TokenBudget struct {
	Total       int // 模型上下文窗口总量
	Reserve     int // 生成预留（30%）
	Knowledge   int // 知识图谱（25%）
	PrevContext int // 前文摘要（15%）
	Facts       int // 动态事实（12%）
	Memory      int // 写作记忆（10%）
	Style       int // 写作风格（8%）
}

// ModelContextWindows 模型上下文窗口映射
var ModelContextWindows = map[string]int{
	"deepseek": 65536,
	"qwen":     131072,
	"zhipu":    131072,
	"kimi":     131072,
}

// defaultContextWindow 未知模型的默认窗口大小
const defaultContextWindow = 65536

// NewTokenBudget 根据模型名创建 Token 预算
// 预算分配比例：Reserve 30%, Knowledge 25%, PrevContext 15%, Facts 12%, Memory 10%, Style 8%
func NewTokenBudget(modelName string) *TokenBudget {
	total, ok := ModelContextWindows[modelName]
	if !ok {
		total = defaultContextWindow
	}

	return &TokenBudget{
		Total:       total,
		Reserve:     total * 30 / 100,
		Knowledge:   total * 25 / 100,
		PrevContext: total * 15 / 100,
		Facts:       total * 12 / 100,
		Memory:      total * 10 / 100,
		Style:       total * 8 / 100,
	}
}

// tokensToChars 将 token 数转换为字符数（中文约 1 token ≈ 0.67 字符）
func tokensToChars(tokens int) int {
	return int(float64(tokens) * 0.67)
}

// CharsForKnowledge 知识图谱可用字符数
func (b *TokenBudget) CharsForKnowledge() int {
	return tokensToChars(b.Knowledge)
}

// CharsForPrevContext 前文摘要可用字符数
func (b *TokenBudget) CharsForPrevContext() int {
	return tokensToChars(b.PrevContext)
}

// CharsForFacts 动态事实可用字符数
func (b *TokenBudget) CharsForFacts() int {
	return tokensToChars(b.Facts)
}

// CharsForMemory 写作记忆可用字符数
func (b *TokenBudget) CharsForMemory() int {
	return tokensToChars(b.Memory)
}

// CharsForStyle 写作风格可用字符数
func (b *TokenBudget) CharsForStyle() int {
	return tokensToChars(b.Style)
}
