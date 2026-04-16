// server/internal/agent/executor_chapter.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-curton/server/internal/model"
)

// ChapterTaskExecutor 章节任务执行器（chapter_summary_polish / chapter_polish / chapter_expand / chapter_continue）
// 从 History JSON 读取 system_prompt + fallback 硬编码 + 概要润色特殊处理
type ChapterTaskExecutor struct{}

func (e *ChapterTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	task := ec.Task

	// 优先从 task.History JSON 中读取 service 层预渲染的 system_prompt
	var systemPrompt string
	if task.History != "" {
		var historyData map[string]interface{}
		if err := json.Unmarshal([]byte(task.History), &historyData); err == nil {
			if sp, ok := historyData["system_prompt"].(string); ok && sp != "" {
				systemPrompt = sp
			}
		}
	}

	// fallback：如果 History 中没有 system_prompt，使用硬编码逻辑（兼容旧任务）
	if systemPrompt == "" {
		switch task.TaskType {
		case model.TaskTypeChapterSummaryPolish:
			systemPrompt = "你是一位资深小说策划编辑。对章节概要进行深度润色和扩写，充分结合知识库中的角色档案、世界观设定、伏笔记录来充实概要内容，使其成为后续正文写作的高质量参考蓝图。直接输出润色后的概要文本，不要包含标题。"
		case model.TaskTypeChapterPolish:
			systemPrompt = "你是一位专业的文学编辑。基于章节概要对内容进行润色，提升文学性和可读性，保持原有情节不变。直接输出润色后的正文。"
		case model.TaskTypeChapterExpand:
			systemPrompt = "你是一位专业的小说作家。严格按照章节概要进行扩写，丰富细节描写和对话，不要偏离概要设定的情节方向。扩写后正文不少于3000字，这是硬性要求。直接输出扩写后的完整正文。"
		case model.TaskTypeChapterContinue:
			systemPrompt = "你是一位专业的小说作家。严格按照章节概要续写后续情节，不要偏离概要设定的情节方向。直接输出续写的内容。"
		}
	}

	// 构建对话历史：system prompt 作为第一条消息
	history := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// 如果 task.History 中有章节上下文，作为 user 消息提供背景
	// 注意：不能用 assistant 角色，通义 qwen 等 API 要求 system 后必须跟 user
	if task.History != "" {
		var historyData map[string]interface{}
		if err := json.Unmarshal([]byte(task.History), &historyData); err == nil {
			// 提取有意义的上下文字段，排除 system_prompt（已在上面使用）
			var contextParts []string
			for k, v := range historyData {
				if k == "system_prompt" {
					continue
				}
				if s, ok := v.(string); ok && s != "" {
					contextParts = append(contextParts, s)
				}
			}
			if len(contextParts) > 0 {
				history = append(history, ChatMessage{
					Role:    "user",
					Content: "以下是相关上下文信息：\n" + strings.Join(contextParts, "\n"),
				})
				history = append(history, ChatMessage{
					Role:    "assistant",
					Content: "好的，我已了解上下文信息，请提供需要处理的内容。",
				})
			}
		}
	}

	req := &TextRequest{
		Model:     ec.ModelVersion,
		Prompt:    task.Prompt,
		History:   history,
		MaxTokens: 16384,
	}

	resp, err := ec.Provider.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}

	// 清洗 AI 返回的正文内容，移除非正文信息
	cleaned := CleanNovelContent(resp.Content)

	var totalUsage TokenUsage
	if resp.Usage != nil {
		totalUsage = *resp.Usage
	}

	// 扩写硬校验：不足 3000 字自动追加续写，最多 2 轮
	if task.TaskType == model.TaskTypeChapterExpand {
		const minExpandWords = 3000
		const maxContinueRounds = 2
		for i := 0; i < maxContinueRounds && len([]rune(cleaned)) < minExpandWords; i++ {
			remaining := minExpandWords - len([]rune(cleaned))
			contReq := &TextRequest{
				Model: ec.ModelVersion,
				History: append(req.History,
					ChatMessage{Role: "user", Content: req.Prompt},
					ChatMessage{Role: "assistant", Content: cleaned},
				),
				Prompt:    fmt.Sprintf("当前正文仅%d字，不满足3000字的硬性要求，还差约%d字。请紧接上文继续扩写，不要重复已有内容，直接输出续写部分。", len([]rune(cleaned)), remaining),
				MaxTokens: 16384,
			}
			contResp, err := ec.Provider.GenerateText(ctx, contReq)
			if err != nil {
				break
			}
			cleaned += "\n\n" + CleanNovelContent(contResp.Content)
			if contResp.Usage != nil {
				totalUsage.PromptTokens += contResp.Usage.PromptTokens
				totalUsage.CompletionTokens += contResp.Usage.CompletionTokens
				totalUsage.TotalTokens += contResp.Usage.TotalTokens
			}
		}
	}

	// 返回结构化结果，包含 content 和 summary
	result := map[string]interface{}{
		"content": cleaned,
		"usage":   totalUsage,
	}

	// 概要润色任务，结果作为 summary 返回
	if task.TaskType == model.TaskTypeChapterSummaryPolish {
		// 全局审核 agent：用 deepseek 审核概要与设定一致性，不通过则修订 1 轮
		cleaned = e.reviewSummary(ctx, ec, cleaned, req)

		result["summary"] = cleaned
		result["content"] = ""
	}

	return result, nil
}

// reviewSummary 概要润色全局审核：deepseek 审核概要与故事线/伏笔/角色设定的一致性
// 不通过则带审核意见修订 1 轮，返回最终概要
func (e *ChapterTaskExecutor) reviewSummary(ctx context.Context, ec *ExecContext, summary string, origReq *TextRequest) string {
	// 需要 GetProvider 才能调用 deepseek
	if ec.GetProvider == nil {
		return summary
	}
	reviewProvider, err := ec.GetProvider("deepseek")
	if err != nil {
		return summary // deepseek 不可用，跳过审核
	}

	reviewPrompt := fmt.Sprintf(`审核以下润色后的章节概要，判断是否与已有设定一致。

【润色后概要】
%s

【原始上下文】
%s

请检查以下维度：
1. 角色行为是否符合人物档案中的性格设定
2. 故事线是否与前后章节概要连贯，有无逻辑断裂
3. 伏笔和剧情线索是否正确引用，有无矛盾
4. 世界观细节是否与设定一致

以JSON输出：{"passed":true/false,"issues":"问题描述（通过则为空）","revision_hint":"修订建议（通过则为空）"}
仅输出JSON。`, summary, ec.Task.Prompt)

	reviewResp, err := reviewProvider.GenerateText(ctx, &TextRequest{
		Prompt:    reviewPrompt,
		MaxTokens: 1024,
	})
	if err != nil {
		return summary
	}

	// 解析审核结果
	var review struct {
		Passed       bool   `json:"passed"`
		Issues       string `json:"issues"`
		RevisionHint string `json:"revision_hint"`
	}
	if err := json.Unmarshal([]byte(reviewResp.Content), &review); err != nil || review.Passed {
		return summary
	}

	// 不通过：带审核意见修订 1 轮
	revisionPrompt := fmt.Sprintf(`以下章节概要未通过设定一致性审核，请根据审核意见修订。

【审核问题】
%s

【修订建议】
%s

【当前概要】
%s

请修订概要，解决以上问题，保持概要在300-400字之间。直接输出修订后的概要文本。`, review.Issues, review.RevisionHint, summary)

	revReq := &TextRequest{
		Model:     ec.ModelVersion,
		History:   origReq.History,
		Prompt:    revisionPrompt,
		MaxTokens: 4096,
	}
	revResp, err := ec.Provider.GenerateText(ctx, revReq)
	if err != nil {
		return summary
	}
	return CleanNovelContent(revResp.Content)
}
