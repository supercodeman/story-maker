// server/internal/service/butler_iterative.go
// 管家多轮迭代 Service — 一次请求触发，后端全自动编排生成→审查→优化循环
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// 迭代配置
const (
	iterMaxRounds      = 5   // 最大 Review 轮次
	iterScoreThreshold = 8.0 // 达标分数
	iterPollInterval   = 2 * time.Second
)

// ConversationMessage 对话历史消息（用于步骤对话式调整）
type ConversationMessage struct {
	Role    string `json:"role"`    // "assistant" | "user"
	Content string `json:"content"` // 消息内容
}

// ButlerIterativeService 管家多轮迭代服务
type ButlerIterativeService struct {
	aiTaskDAO     *dao.AITaskDAO
	dispatcher    *agent.Dispatcher
	hub           agent.Notifier
	modelRegistry *ModelRegistryService

	mu         sync.RWMutex
	iterations map[string]*ButlerIterateStatus // iterationID -> status
}

// ButlerIterateRequest 启动多轮迭代请求
type ButlerIterateRequest struct {
	PortfolioID     uint   `json:"portfolio_id" binding:"required"`
	Action          string `json:"action" binding:"required"` // "storyline" | "characters"
	Setting         string `json:"setting"`                   // 创作方向
	PrevStepResult  string `json:"prev_step_result"`          // 上一步结果（选题/故事线）
	UserPrompt      string `json:"user_prompt"`
	ModelName       string `json:"model_name"`
	ButlerSessionID string `json:"butler_session_id"`
	EnableBeats     bool   `json:"enable_beats"`    // 启用段落细化（中长篇）
	EnableSubplots  bool   `json:"enable_subplots"` // 启用支线交织
	ChapterNum      int    `json:"chapter_num"`     // 预计章节数（用于判断是否中长篇）
	ConversationHistory []ConversationMessage `json:"conversation_history,omitempty"` // 对话历史（对话模式调整用）
}

// ButlerIterateStatus 迭代状态
type ButlerIterateStatus struct {
	Phase            string `json:"phase"`                     // "generating" | "reviewing" | "completed" | "failed"
	Round            int    `json:"round"`                     // 当前轮次（0=草稿生成中）
	MaxRounds        int    `json:"max_rounds"`                // 最大轮次
	FinalResult      string `json:"final_result"`              // 最终结果（completed 时有值）
	StructuredData   string `json:"structured_data,omitempty"` // 提取的结构化 JSON（如 STORY_STRUCTURE）
	TaskIDs          []uint `json:"task_ids"`                  // 所有关联 task ID
	Error            string `json:"error,omitempty"`
	PromptTokens     int    `json:"prompt_tokens"`             // 累计输入 token
	CompletionTokens int    `json:"completion_tokens"`         // 累计输出 token
	TotalTokens      int    `json:"total_tokens"`              // 累计总 token
}

// NewButlerIterativeService 创建实例
func NewButlerIterativeService(aiTaskDAO *dao.AITaskDAO, dispatcher *agent.Dispatcher, hub agent.Notifier) *ButlerIterativeService {
	return &ButlerIterativeService{
		aiTaskDAO:  aiTaskDAO,
		dispatcher: dispatcher,
		hub:        hub,
		iterations: make(map[string]*ButlerIterateStatus),
	}
}

// SetModelRegistry 注入模型注册服务
func (s *ButlerIterativeService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// StartIteration 启动多轮迭代（异步 goroutine）
func (s *ButlerIterativeService) StartIteration(ctx context.Context, userID uint, req *ButlerIterateRequest) (string, error) {
	// 校验 action
	if req.Action != "storyline" && req.Action != "characters" && req.Action != "opening_polish" {
		return "", fmt.Errorf("invalid action: %s, must be 'storyline', 'characters' or 'opening_polish'", req.Action)
	}

	iterationID := generateIterationID()

	// 初始化状态
	status := &ButlerIterateStatus{
		Phase:     "generating",
		Round:     0,
		MaxRounds: iterMaxRounds,
		TaskIDs:   []uint{},
	}
	s.mu.Lock()
	s.iterations[iterationID] = status
	s.mu.Unlock()

	// 异步执行
	go s.runIteration(context.Background(), userID, req, iterationID)

	return iterationID, nil
}

// GetIterationStatus 查询迭代状态
func (s *ButlerIterativeService) GetIterationStatus(iterationID string) (*ButlerIterateStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, ok := s.iterations[iterationID]
	if !ok {
		return nil, fmt.Errorf("iteration not found: %s", iterationID)
	}

	// 返回副本，避免并发读写
	cp := *status
	cp.TaskIDs = make([]uint, len(status.TaskIDs))
	copy(cp.TaskIDs, status.TaskIDs)
	return &cp, nil
}

// runIteration 异步执行多轮迭代
func (s *ButlerIterativeService) runIteration(ctx context.Context, userID uint, req *ButlerIterateRequest, iterationID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[butler-iter] panic in iteration %s: %v", iterationID, r)
			s.updatePhase(iterationID, "failed", fmt.Sprintf("internal error: %v", r))
		}
	}()

	modelName := req.ModelName
	if s.modelRegistry != nil {
		modelName = s.modelRegistry.ResolveModel(req.ModelName, model.CapTextGen)
	} else if modelName == "" {
		modelName = "deepseek"
	}

	// 第1步：生成草稿
	log.Printf("[butler-iter] %s: 开始生成草稿 action=%s", iterationID, req.Action)
	draftContent, draftTaskID, err := s.executeDraft(ctx, userID, req, modelName)
	if err != nil {
		log.Printf("[butler-iter] %s: 草稿生成失败: %v", iterationID, err)
		s.updatePhase(iterationID, "failed", err.Error())
		return
	}
	s.addTaskID(iterationID, draftTaskID)
	log.Printf("[butler-iter] %s: 草稿生成完成 task_id=%d len=%d", iterationID, draftTaskID, len(draftContent))

	// 第2步：Review 循环
	current := draftContent
	for round := 1; round <= iterMaxRounds; round++ {
		s.updateRound(iterationID, "reviewing", round)
		log.Printf("[butler-iter] %s: 第 %d 轮 Review", iterationID, round)

		score, revised, reviewTaskID, err := s.executeReview(ctx, userID, req, current, modelName)
		if err != nil {
			log.Printf("[butler-iter] %s: 第 %d 轮 Review 失败: %v，使用当前版本", iterationID, round, err)
			// Review 失败不中断，使用当前版本
			break
		}
		s.addTaskID(iterationID, reviewTaskID)
		log.Printf("[butler-iter] %s: 第 %d 轮 Review 完成 score=%.1f task_id=%d", iterationID, round, score, reviewTaskID)

		if revised != "" {
			current = revised
		}

		if score >= iterScoreThreshold {
			log.Printf("[butler-iter] %s: 达标 score=%.1f >= %.1f", iterationID, score, iterScoreThreshold)
			break
		}
	}

	// 完成：保存结果并通知
	s.mu.Lock()
	if st, ok := s.iterations[iterationID]; ok {
		st.Phase = "completed"
		st.FinalResult = current
		// 汇总所有关联任务的 token 消耗
		for _, tid := range st.TaskIDs {
			if task, err := s.aiTaskDAO.GetTask(ctx, tid); err == nil {
				st.PromptTokens += task.PromptTokens
				st.CompletionTokens += task.CompletionTokens
				st.TotalTokens += task.TotalTokens
			}
		}
		// 故事线完成时提取结构化 JSON，供后续步骤使用
		if req.Action == "storyline" {
			if structJSON := extractTagContent(current, "---STORY_STRUCTURE---", "---END_STRUCTURE---"); structJSON != "" {
				st.StructuredData = structJSON
			}
		}
		// 开篇打磨完成时提取精细化章节 JSON
		if req.Action == "opening_polish" {
			if openingJSON := extractTagContent(current, "---OPENING_CHAPTERS---", "---END_OPENING---"); openingJSON != "" {
				st.StructuredData = openingJSON
			}
		}
	}
	s.mu.Unlock()

	// WebSocket 推送完成通知
	if s.hub != nil {
		_ = s.hub.NotifyUserWithType(userID, "butler_iteration_done", map[string]interface{}{
			"iteration_id": iterationID,
			"action":       req.Action,
		})
	}

	log.Printf("[butler-iter] %s: 迭代完成", iterationID)
}

// executeDraft 执行草稿生成
func (s *ButlerIterativeService) executeDraft(ctx context.Context, userID uint, req *ButlerIterateRequest, modelName string) (string, uint, error) {
	var systemPrompt, userContent string
	var taskType string

	switch req.Action {
	case "storyline":
		systemPrompt, userContent = buildStorylineDraftPrompt(req.Setting, req.PrevStepResult, req.UserPrompt, req.EnableBeats, req.EnableSubplots, req.ChapterNum, req.ConversationHistory)
		taskType = model.TaskTypeButlerStorylineDraft
	case "characters":
		systemPrompt, userContent = buildCharactersDraftPrompt(req.Setting, req.PrevStepResult, req.UserPrompt, req.ConversationHistory)
		taskType = model.TaskTypeButlerCharactersDraft
	case "opening_polish":
		systemPrompt, userContent = buildOpeningSummaryPolishPrompt(req.PrevStepResult, req.Setting, req.UserPrompt)
		taskType = model.TaskTypeButlerOpeningDraft
	}

	return s.executeAITask(ctx, userID, req.PortfolioID, req.ButlerSessionID, taskType, modelName, systemPrompt, userContent)
}

// executeReview 执行 Review 审查
func (s *ButlerIterativeService) executeReview(ctx context.Context, userID uint, req *ButlerIterateRequest, content, modelName string) (float64, string, uint, error) {
	var systemPrompt, userContent string
	var taskType string

	switch req.Action {
	case "storyline":
		systemPrompt, userContent = buildStorylineReviewPrompt(content)
		taskType = model.TaskTypeButlerStorylineReview
	case "characters":
		systemPrompt, userContent = buildCharactersReviewPrompt(content)
		taskType = model.TaskTypeButlerCharactersReview
	case "opening_polish":
		systemPrompt, userContent = buildOpeningSummaryReviewPrompt(content)
		taskType = model.TaskTypeButlerOpeningReview
	}

	resultText, taskID, err := s.executeAITask(ctx, userID, req.PortfolioID, req.ButlerSessionID, taskType, modelName, systemPrompt, userContent)
	if err != nil {
		return 0, "", taskID, err
	}

	// 解析 Review JSON 结果
	score, revised, parseErr := parseReviewResult(resultText)
	if parseErr != nil {
		log.Printf("[butler-iter] Review 结果解析失败: %v, raw=%s", parseErr, truncateStr(resultText, 200))
		// 解析失败时给一个默认高分让循环结束，使用原始内容
		return iterScoreThreshold, content, taskID, nil
	}

	return score, revised, taskID, nil
}

// executeAITask 创建并同步执行 AI 任务
func (s *ButlerIterativeService) executeAITask(ctx context.Context, userID, portfolioID uint, butlerSessionID, taskType, modelName, systemPrompt, userContent string) (string, uint, error) {
	// 构建 History JSON（system + user 消息）
	history := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userContent},
	}
	historyJSON, _ := json.Marshal(history)

	task := &model.AITask{
		UserID:          userID,
		PortfolioID:     portfolioID,
		TaskType:        taskType,
		ModelName:       modelName,
		Prompt:          userContent,
		History:         string(historyJSON),
		Status:          model.TaskStatusPending,
		ButlerSessionID: butlerSessionID,
	}

	// 同步执行
	result, err := s.dispatcher.ExecuteSingle(ctx, task)
	if err != nil {
		return "", task.ID, fmt.Errorf("AI task failed: %w", err)
	}

	// 提取文本结果
	text := extractTextResult(result)
	return text, task.ID, nil
}

// extractTextResult 从 AI 返回结果中提取文本
func extractTextResult(result interface{}) string {
	if result == nil {
		return ""
	}

	switch v := result.(type) {
	case string:
		return v
	case map[string]interface{}:
		if content, ok := v["content"]; ok {
			return fmt.Sprintf("%v", content)
		}
		if text, ok := v["text"]; ok {
			return fmt.Sprintf("%v", text)
		}
		// 尝试整体 JSON
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// reviewResult Review 结果结构
type reviewResult struct {
	Score          float64 `json:"score"`
	RevisedContent string  `json:"revised_content"`
	Issues         []struct {
		Dimension  string `json:"dimension"`
		Problem    string `json:"problem"`
		Suggestion string `json:"suggestion"`
	} `json:"issues"`
}

// parseReviewResult 解析 Review JSON 结果
func parseReviewResult(text string) (float64, string, error) {
	// 尝试直接解析
	var result reviewResult
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result.Score, result.RevisedContent, nil
	}

	// 尝试提取 JSON 块（AI 可能包裹在 markdown 代码块中）
	cleaned := text
	if idx := strings.Index(cleaned, "{"); idx >= 0 {
		cleaned = cleaned[idx:]
	}
	if idx := strings.LastIndex(cleaned, "}"); idx >= 0 {
		cleaned = cleaned[:idx+1]
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// AI 常在 JSON 字符串值中输出真实换行符，标准 JSON 不允许
		// 修复：将字符串值内的真实换行替换为转义的 \n
		fixed := fixJSONNewlines(cleaned)
		if err2 := json.Unmarshal([]byte(fixed), &result); err2 != nil {
			return 0, "", fmt.Errorf("failed to parse review JSON: %w", err)
		}
	}

	return result.Score, result.RevisedContent, nil
}

// fixJSONNewlines 修复 JSON 字符串值中的未转义换行符
// 在双引号内的真实换行符替换为 \n
func fixJSONNewlines(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			buf.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' && inString {
			buf.WriteByte(c)
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			buf.WriteByte(c)
			continue
		}
		if inString && (c == '\n' || c == '\r') {
			if c == '\r' && i+1 < len(s) && s[i+1] == '\n' {
				i++ // skip \r in \r\n
			}
			buf.WriteString(`\n`)
			continue
		}
		buf.WriteByte(c)
	}
	return buf.String()
}

// updatePhase 更新迭代阶段
func (s *ButlerIterativeService) updatePhase(iterationID, phase, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.iterations[iterationID]; ok {
		st.Phase = phase
		if errMsg != "" {
			st.Error = errMsg
		}
	}
}

// updateRound 更新当前轮次
func (s *ButlerIterativeService) updateRound(iterationID, phase string, round int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.iterations[iterationID]; ok {
		st.Phase = phase
		st.Round = round
	}
}

// addTaskID 添加关联 task ID
func (s *ButlerIterativeService) addTaskID(iterationID string, taskID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.iterations[iterationID]; ok {
		st.TaskIDs = append(st.TaskIDs, taskID)
	}
}

// generateIterationID 生成唯一迭代 ID
func generateIterationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
