// server/internal/agent/executor_text.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
)

// TextTaskExecutor 文本任务执行器（text_gen / text_polish / storyboard）
// 支持 History JSON 解析 + Token 裁剪 + Tool 循环（最多5轮）
type TextTaskExecutor struct{}

func (e *TextTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	// 解析历史对话，同时提取节点级配置
	var history []ChatMessage
	var systemPrompt string
	maxTokens := 16384 // 默认值，确保生成内容可达5000字以上
	if ec.Task.History != "" {
		// 先尝试解析为带配置的 JSON 对象（工作流节点传入）
		var historyObj map[string]interface{}
		if err := json.Unmarshal([]byte(ec.Task.History), &historyObj); err == nil {
			if mt, ok := historyObj["max_tokens"].(float64); ok && int(mt) > 0 {
				maxTokens = int(mt)
			}
			if sp, ok := historyObj["system_prompt"].(string); ok && sp != "" {
				systemPrompt = sp
			}
		}
		// 兼容原有的纯对话数组格式
		_ = json.Unmarshal([]byte(ec.Task.History), &history)
	}

	req := &TextRequest{
		Model:        ec.ModelVersion,
		Prompt:       ec.Task.Prompt,
		History:      history,
		MaxTokens:    maxTokens,
		CharacterCtx: systemPrompt,
	}

	// Token 窗口裁剪
	if ec.TokenMgr != nil && len(req.History) > 0 {
		systemTokens := EstimateTokens(req.Prompt)
		req.History, _ = ec.TokenMgr.TrimHistory(req.History, systemTokens)
	}

	// 注入工具定义
	if ec.ToolRegistry != nil && ec.ToolRegistry.HasTools() {
		req.Tools = ec.ToolRegistry.ToFunctionDefs()
	}

	// 累加多轮 tool call 的 token 消耗
	var totalUsage TokenUsage

	// Tool 执行循环（最多 5 轮，防止死循环）
	const maxToolRounds = 5
	for i := 0; i < maxToolRounds; i++ {
		resp, err := ec.Provider.GenerateText(ctx, req)
		if err != nil {
			return nil, err
		}

		// 累加本轮 token 消耗
		if resp.Usage != nil {
			totalUsage.PromptTokens += resp.Usage.PromptTokens
			totalUsage.CompletionTokens += resp.Usage.CompletionTokens
			totalUsage.TotalTokens += resp.Usage.TotalTokens
		}

		// 没有 tool_call，直接返回最终结果
		if len(resp.ToolCalls) == 0 {
			content := resp.Content
			// 仅对小说正文类任务做内容清洗，JSON 结构化输出（如 overview_extract / knowledge_extract）不清洗
			switch ec.Task.TaskType {
			case "overview_extract", "knowledge_extract":
				// 保持原始输出
			default:
				content = CleanNovelContent(content)
			}
			return map[string]interface{}{
				"content": content,
				"usage":   totalUsage,
			}, nil
		}

		// 有 tool_call：执行工具，将结果追加到 history 继续对话
		// 首轮时先将原始 user 消息加入 history，确保消息顺序正确：
		// user → assistant(tool_calls) → tool(result)
		if i == 0 && req.Prompt != "" {
			req.History = append(req.History, ChatMessage{
				Role:    "user",
				Content: req.Prompt,
			})
		}

		assistantMsg := ChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		req.History = append(req.History, assistantMsg)

		// 逐个执行工具，追加 tool 结果消息
		for _, call := range resp.ToolCalls {
			toolResult := executeToolCall(ctx, ec.ToolRegistry, call)
			req.History = append(req.History, ChatMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: call.ID,
			})
		}

		// tool 结果已注入 history，后续轮次不再提供 tools 定义，
		// 避免模型反复调用工具浪费 token
		req.Tools = nil
		// Prompt 已在 history 中，清空避免 provider 重复追加 user 消息
		req.Prompt = ""
	}

	return nil, errors.New("tool call loop exceeded max iterations")
}
