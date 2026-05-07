// server/internal/agent/executor_outline.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"story-maker/server/internal/model"
)

// OutlineTaskExecutor 大纲生成任务执行器（outline_generate）
// 从 History JSON 中读取 system prompt（Service 层模板渲染后传入），fallback 到默认值
type OutlineTaskExecutor struct{}

const outlineBatchSize = 10 // 每批生成的章节数

func (e *OutlineTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	// 从 History JSON 中读取 system prompt 和 chapter_num
	systemPrompt := "你是一位专业的小说策划师。根据用户提供的设定、人物和剧情思路，生成一个完整的小说大纲。\ntitle 只写纯标题（如\"暗夜追踪\"、\"命运的抉择\"），不要带\"第X章\"等章节序号前缀，系统会自动编号。\n你必须严格按照以下 JSON 格式输出，不要包含任何其他文字：\n[\n  {\"title\": \"暗夜追踪\", \"summary\": \"100-200字的章节概要...\"},\n  {\"title\": \"命运的抉择\", \"summary\": \"100-200字的章节概要...\"}\n]"
	chapterNum := 0

	if ec.Task.History != "" {
		var historyData map[string]interface{}
		if err := json.Unmarshal([]byte(ec.Task.History), &historyData); err == nil {
			if sp, ok := historyData["system_prompt"].(string); ok && sp != "" {
				systemPrompt = sp
			}
			if cn, ok := historyData["chapter_num"].(float64); ok {
				chapterNum = int(cn)
			}
		}
	}

	// 如果章节数 <= outlineBatchSize，单次生成
	if chapterNum <= outlineBatchSize {
		return e.generateSingle(ctx, ec, systemPrompt)
	}

	// 分批串行生成
	return e.generateBatched(ctx, ec, systemPrompt, chapterNum)
}

// generateSingle 单次生成（章节数较少时）
func (e *OutlineTaskExecutor) generateSingle(ctx context.Context, ec *ExecContext, systemPrompt string) (interface{}, error) {
	history := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	req := &TextRequest{
		Model:     ec.ModelVersion,
		Prompt:    ec.Task.Prompt,
		History:   history,
		MaxTokens: 8192,
	}

	resp, err := ec.Provider.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}

	chapters, parseErr := parseOutlineChapters(resp.Content)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse outline JSON: %w, raw: %s", parseErr, resp.Content)
	}

	result := map[string]interface{}{
		"chapters": chapters,
	}
	if resp.Usage != nil {
		result["usage"] = *resp.Usage
	}
	return result, nil
}

// generateBatched 分批串行生成（章节数较多时，每批 outlineBatchSize 章）
func (e *OutlineTaskExecutor) generateBatched(ctx context.Context, ec *ExecContext, systemPrompt string, totalChapters int) (interface{}, error) {
	var allChapters []outlineChapter
	var totalUsage TokenUsage

	generated := 0
	batchIdx := 0

	for generated < totalChapters {
		batchIdx++
		remaining := totalChapters - generated
		batchSize := outlineBatchSize
		if remaining < batchSize {
			batchSize = remaining
		}

		// 构建分批 prompt
		batchPrompt := e.buildBatchPrompt(ec.Task.Prompt, allChapters, generated+1, generated+batchSize, totalChapters)

		log.Printf("[outline-batch] batch %d: generating chapters %d-%d of %d", batchIdx, generated+1, generated+batchSize, totalChapters)

		history := []ChatMessage{
			{Role: "system", Content: systemPrompt},
		}

		req := &TextRequest{
			Model:     ec.ModelVersion,
			Prompt:    batchPrompt,
			History:   history,
			MaxTokens: 8192,
		}

		resp, err := ec.Provider.GenerateText(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("batch %d failed: %w", batchIdx, err)
		}

		chapters, parseErr := parseOutlineChapters(resp.Content)
		if parseErr != nil {
			return nil, fmt.Errorf("batch %d parse failed: %w, raw: %s", batchIdx, parseErr, resp.Content)
		}

		log.Printf("[outline-batch] batch %d: got %d chapters", batchIdx, len(chapters))

		allChapters = append(allChapters, chapters...)
		generated += len(chapters)

		// 累加 token 消耗
		if resp.Usage != nil {
			totalUsage.PromptTokens += resp.Usage.PromptTokens
			totalUsage.CompletionTokens += resp.Usage.CompletionTokens
			totalUsage.TotalTokens += resp.Usage.TotalTokens
		}
	}

	log.Printf("[outline-batch] completed: total %d chapters in %d batches", len(allChapters), batchIdx)

	result := map[string]interface{}{
		"chapters": allChapters,
		"usage":    totalUsage,
	}
	return result, nil
}

// buildBatchPrompt 构建分批 prompt，包含前面已生成章节的标题摘要
func (e *OutlineTaskExecutor) buildBatchPrompt(originalPrompt string, prevChapters []outlineChapter, startIdx, endIdx, totalChapters int) string {
	if len(prevChapters) == 0 {
		// 第一批：在原始 prompt 后追加批次指令
		return fmt.Sprintf("%s\n\n【本次生成范围】\n请先生成第 %d 到第 %d 章（共 %d 章中的前 %d 章）。严格按 JSON 数组格式输出，不要包含其他文字。",
			originalPrompt, startIdx, endIdx, totalChapters, endIdx-startIdx+1)
	}

	// 后续批次：近距离章节保留完整摘要，远距离章节只保留标题
	var prevSummary strings.Builder
	prevSummary.WriteString("【已生成的章节】\n")
	recentStart := len(prevChapters) - outlineBatchSize // 最近一批的起始位置
	if recentStart < 0 {
		recentStart = 0
	}
	for i, ch := range prevChapters {
		if i < recentStart {
			// 远距离章节：只保留标题
			prevSummary.WriteString(fmt.Sprintf("第%d章「%s」\n", i+1, ch.Title))
		} else {
			// 近距离章节：保留完整摘要
			prevSummary.WriteString(fmt.Sprintf("第%d章「%s」：%s\n", i+1, ch.Title, ch.Summary))
		}
	}

	return fmt.Sprintf("%s\n\n%s\n【本次生成范围】\n请继续生成第 %d 到第 %d 章（共 %d 章），与前面章节保持剧情连贯、节奏一致。严格按 JSON 数组格式输出，不要包含其他文字。",
		originalPrompt, prevSummary.String(), startIdx, endIdx, totalChapters)
}

// truncate 截断字符串
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}

// outlineChapter 大纲章节结构
type outlineChapter struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// parseOutlineChapters 解析 AI 返回的大纲章节 JSON，带多级容错
func parseOutlineChapters(raw string) ([]outlineChapter, error) {
	content := extractJSONFromMarkdown(raw)

	var chapters []outlineChapter

	// 尝试1：直接解析为数组
	if err := json.Unmarshal([]byte(content), &chapters); err == nil && len(chapters) > 0 {
		return chapters, nil
	}

	// 尝试2：包裹为数组（AI 可能返回 {}, {} 而非 [{}, {}]）
	wrapped := "[" + strings.TrimSpace(content) + "]"
	if err := json.Unmarshal([]byte(wrapped), &chapters); err == nil && len(chapters) > 0 {
		return chapters, nil
	}

	// 尝试3：JSON 被截断，修复不完整的 JSON
	repaired := repairTruncatedJSON(content)
	if repaired != content {
		if err := json.Unmarshal([]byte(repaired), &chapters); err == nil && len(chapters) > 0 {
			return chapters, nil
		}
	}

	// 尝试4：用正则逐个提取 {"title": "...", "summary": "..."} 对象
	chapters = extractChaptersByRegex(raw)
	if len(chapters) > 0 {
		return chapters, nil
	}

	return nil, fmt.Errorf("all parse attempts failed")
}

// repairTruncatedJSON 修复被截断的 JSON 数组
func repairTruncatedJSON(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "[") {
		content = "[" + content
	}

	// 找到最后一个完整的 } 位置
	lastBrace := strings.LastIndex(content, "}")
	if lastBrace < 0 {
		return content
	}

	// 截取到最后一个完整对象，补上 ]
	return content[:lastBrace+1] + "]"
}

// extractChaptersByRegex 用正则从文本中提取章节
func extractChaptersByRegex(raw string) []outlineChapter {
	// 匹配 "title": "..." 和 "summary": "..." 对
	re := regexp.MustCompile(`"title"\s*:\s*"((?:[^"\\]|\\.)*)"\s*,\s*"summary"\s*:\s*"((?:[^"\\]|\\.)*)"`)
	matches := re.FindAllStringSubmatch(raw, -1)

	var chapters []outlineChapter
	for _, m := range matches {
		if len(m) >= 3 {
			title := strings.ReplaceAll(m[1], `\"`, `"`)
			summary := strings.ReplaceAll(m[2], `\"`, `"`)
			chapters = append(chapters, outlineChapter{Title: title, Summary: summary})
		}
	}
	return chapters
}

// OutlineChapterExecutor 大纲页面章节级 AI 操作执行器
// 支持 outline_title_polish / outline_summary_polish / outline_summary_expand
// 通过 task.TaskType 区分 system prompt
type OutlineChapterExecutor struct{}

func (e *OutlineChapterExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	task := ec.Task

	// 根据任务类型选择默认 system prompt 和 maxTokens
	var defaultSystemPrompt string
	var maxTokens int
	switch task.TaskType {
	case model.TaskTypeOutlineTitlePolish:
		defaultSystemPrompt = "你是一位专业的小说策划师。对用户提供的章节标题进行润色，使其更加精炼、有吸引力，同时保持与章节内容的关联性。只输出润色后的标题文本，不要包含任何解释或额外内容。"
		maxTokens = 256
	case model.TaskTypeOutlineSummaryPolish:
		defaultSystemPrompt = "你是一位专业的小说策划师。对用户提供的章节概要进行润色，使其更加清晰、连贯、有吸引力，保持原有情节方向不变。只输出润色后的概要文本，不要包含标题或额外说明。"
		maxTokens = 4096
	case model.TaskTypeOutlineSummaryExpand:
		defaultSystemPrompt = "你是一位专业的小说策划师。对用户提供的章节概要进行扩写，丰富情节细节、人物动机和场景描写，使概要更加充实完整。只输出扩写后的概要文本，不要包含标题或额外说明。"
		maxTokens = 8192
	case model.TaskTypeOutlineGenerateCharacters:
		defaultSystemPrompt = "你是一位专业的小说策划师。根据用户提供的世界观/设定、背景信息和剧情思路，设计主要核心人物。为每个人物提供：姓名、身份/职业、性格特点、人物关系、在故事中的角色定位。人物之间要有合理的关系网络和冲突张力。只输出人物设定文本，不要包含额外说明。"
		maxTokens = 4096
	case model.TaskTypeButlerGenerateTopic:
		defaultSystemPrompt = "你是一位专业的小说策划师。根据用户提供的创作方向和偏好，生成一个完整的选题方案。方案需包含：小说标题、题材类型、核心卖点（3-5个）、目标读者画像。输出纯文本，不要使用 JSON 格式，不要包含额外说明。"
		maxTokens = 2048
	case model.TaskTypeButlerGenerateStoryline:
		defaultSystemPrompt = "你是一位专业的小说策划师。根据用户提供的选题信息和创作方向，生成一个完整的故事线方案。方案需包含：世界观设定、背景信息、剧情大纲（起承转合）。输出纯文本，不要使用 JSON 格式，不要包含额外说明。"
		maxTokens = 4096
	case model.TaskTypeButlerGenerateCharacters:
		defaultSystemPrompt = "你是一位专业的小说策划师。根据用户提供的故事线和创作方向，生成一套完整的人物群像设定。为每个人物提供：姓名、身份/职业、性格特点、人物关系、在故事中的角色定位、人物弧光。人物之间要有合理的关系网络和冲突张力。输出纯文本，不要使用 JSON 格式，不要包含额外说明。"
		maxTokens = 4096
	case model.TaskTypeButlerStorylineDraft:
		defaultSystemPrompt = "你是一位资深小说策划师，擅长构思引人入胜的故事线。根据用户提供的创作方向和选题结果，生成完整的故事线。输出纯文本，不要使用 JSON 格式。"
		maxTokens = 4096
	case model.TaskTypeButlerStorylineReview:
		defaultSystemPrompt = "你是一位严格的小说编辑，专门审查故事线质量。发现问题并直接修改，输出 JSON 格式的审查结果。"
		maxTokens = 8192
	case model.TaskTypeButlerCharactersDraft:
		defaultSystemPrompt = "你是一位资深小说策划师，擅长设计立体丰满的人物群像。根据用户提供的创作方向和故事线，生成完整的人物设计。输出纯文本。"
		maxTokens = 8192
	case model.TaskTypeButlerCharactersReview:
		defaultSystemPrompt = "你是一位严格的小说编辑，专门审查人物设计质量。发现问题并直接修改，输出 JSON 格式的审查结果。"
		maxTokens = 8192
	case model.TaskTypeButlerOpeningDraft:
		defaultSystemPrompt = "你是一位资深小说作家，擅长精细化章节概要。根据用户提供的章节大纲，对前5章概要进行精细化打磨，补充细节和情感节奏。输出纯文本。"
		maxTokens = 8192
	case model.TaskTypeButlerOpeningReview:
		defaultSystemPrompt = "你是一位严格的小说编辑，专门审查章节概要质量。发现问题并直接修改，输出 JSON 格式的审查结果。"
		maxTokens = 8192
	default:
		return nil, fmt.Errorf("unsupported outline chapter task type: %s", task.TaskType)
	}

	// 优先从 History JSON 中读取 system prompt（Service 层模板渲染后传入）
	systemPrompt := defaultSystemPrompt
	var contextInfo string
	var historyMessages []ChatMessage
	if task.History != "" {
		// 尝试1：解析为对象格式 {"system_prompt": "...", ...}（工作流节点传入）
		var historyData map[string]interface{}
		if err := json.Unmarshal([]byte(task.History), &historyData); err == nil {
			if sp, ok := historyData["system_prompt"].(string); ok && sp != "" {
				systemPrompt = sp
			}
			delete(historyData, "system_prompt")
			if len(historyData) > 0 {
				if ctxBytes, err := json.Marshal(historyData); err == nil {
					contextInfo = string(ctxBytes)
				}
			}
		} else {
			// 尝试2：解析为对话数组格式 [{role, content}, ...]（butler 迭代传入）
			var msgs []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(task.History), &msgs); err == nil {
				for _, m := range msgs {
					if m.Role == "system" && m.Content != "" {
						systemPrompt = m.Content
					} else if m.Role != "" {
						historyMessages = append(historyMessages, ChatMessage{Role: m.Role, Content: m.Content})
					}
				}
			}
		}
	}

	// 构建消息历史
	history := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	if contextInfo != "" {
		history = append(history, ChatMessage{
			Role: "assistant", Content: "好的，我已了解上下文信息：" + contextInfo,
		})
	}
	// 追加从数组格式解析出的非 system 消息
	history = append(history, historyMessages...)

	req := &TextRequest{
		Model:     ec.ModelVersion,
		Prompt:    task.Prompt,
		History:   history,
		MaxTokens: maxTokens,
	}

	resp, err := ec.Provider.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}

	// 清洗 AI 返回内容：review 类型返回 JSON 结构，不做清洗
	var cleaned string
	switch task.TaskType {
	case model.TaskTypeButlerStorylineReview, model.TaskTypeButlerCharactersReview:
		cleaned = resp.Content
	default:
		cleaned = CleanNovelContent(resp.Content)
	}
	result := map[string]interface{}{}

	// 附带 token 消耗统计
	if resp.Usage != nil {
		result["usage"] = *resp.Usage
	}

	if task.TaskType == model.TaskTypeOutlineTitlePolish {
		result["title"] = cleaned
	} else if task.TaskType == model.TaskTypeOutlineGenerateCharacters {
		result["characters"] = cleaned
	} else if task.TaskType == model.TaskTypeButlerGenerateTopic ||
		task.TaskType == model.TaskTypeButlerGenerateStoryline ||
		task.TaskType == model.TaskTypeButlerGenerateCharacters ||
		task.TaskType == model.TaskTypeButlerStorylineDraft ||
		task.TaskType == model.TaskTypeButlerStorylineReview ||
		task.TaskType == model.TaskTypeButlerCharactersDraft ||
		task.TaskType == model.TaskTypeButlerCharactersReview {
		result["content"] = cleaned
	} else {
		result["summary"] = cleaned
	}
	return result, nil
}
