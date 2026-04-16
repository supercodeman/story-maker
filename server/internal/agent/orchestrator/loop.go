// server/internal/agent/orchestrator/loop.go
package orchestrator

import "encoding/json"

// ExhaustedPolicy 定义循环耗尽时的处理策略
type ExhaustedPolicy int

const (
	// ExhaustedContinue 使用最后一轮结果继续，标记 warning
	ExhaustedContinue ExhaustedPolicy = iota
	// ExhaustedFail 整个工作流失败
	ExhaustedFail
)

// LoopConfig 循环节点的配置
type LoopConfig struct {
	MaxRounds      int                                        // 最大轮次上限
	ExitCondition  func(state *SharedState, round int) bool   // 退出条件：返回 true 表示可以退出循环
	SubGraphBuilder func(round int) *Graph                    // 每轮子 DAG 构建器（round 从 1 开始）
	OnExhausted    ExhaustedPolicy                            // 耗尽策略
}

// LoopNode 循环节点，在主 DAG 中作为可重复执行的子图容器
type LoopNode struct {
	ID        string     // 在主 DAG 中的唯一标识
	DependsOn []string   // 依赖的前置节点
	Config    LoopConfig // 循环配置
}

// LoopResult 循环执行结果元信息，写入 state
type LoopResult struct {
	TotalRounds int    `json:"total_rounds"`
	ExitReason  string `json:"exit_reason"` // "condition_met" | "exhausted"
	ExitRound   int    `json:"exit_round"`
}

// reviewResult 审核结果 JSON 解析，只解析需要的字段
type reviewResult struct {
	Passed       bool `json:"passed"`
	OverallScore int  `json:"overall_score"`
}

// parseReviewResult 从 state 中获取 reviewKey 对应的字符串并解析为 reviewResult
func parseReviewResult(state *SharedState, reviewKey string) (reviewResult, bool) {
	val, ok := state.Get(reviewKey)
	if !ok {
		return reviewResult{}, false
	}

	str, ok := val.(string)
	if !ok {
		return reviewResult{}, false
	}

	var result reviewResult
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return reviewResult{}, false
	}

	return result, true
}

// reviewNotPassed 修复版审核判断，替代 templates.go 中的字符串匹配版本。
// 解析失败视为不通过，触发修订。
func reviewNotPassed(state *SharedState, reviewKey string) bool {
	result, ok := parseReviewResult(state, reviewKey)
	if !ok {
		return true
	}
	return !result.Passed
}

// contentTooShort 检查 stateKey 对应的内容字数（rune 计数）是否低于 minChars。
// 内容不存在或非字符串视为不足。
func contentTooShort(state *SharedState, stateKey string, minChars int) bool {
	val, ok := state.Get(stateKey)
	if !ok {
		return true
	}
	// executor 返回 map[string]interface{}{"content": "..."} 或直接字符串
	var text string
	switch v := val.(type) {
	case string:
		text = v
	case map[string]interface{}:
		if c, ok := v["content"].(string); ok {
			text = c
		}
	}
	return len([]rune(text)) < minChars
}
