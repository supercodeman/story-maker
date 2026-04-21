// server/internal/service/novel.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"log"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// NovelService 小说工坊业务逻辑层
type NovelService struct {
	novelDAO           *dao.NovelDAO
	aiTaskDAO          *dao.AITaskDAO
	dispatcher         *agent.Dispatcher
	tplSvc             *PromptTemplateService
	knowledgeSvc       *KnowledgeService
	writingStyleSvc    *WritingStyleService
	plotStructureSvc   *PlotStructureService
	hitAnalysisSvc     *HitAnalysisService
	behaviorSvc        *UserBehaviorService
	intentSvc          *IntentService
	memorySvc          *MemoryService
	writerLevelSvc     *WriterLevelService
	factSvc            *NovelFactService
	chapterSummarySvc  *ChapterSummaryService
	modelRegistry      *ModelRegistryService
	chapterReviewDAO   *dao.ChapterReviewDAO
}

// NewNovelService 创建 NovelService 实例
func NewNovelService(aiTaskDAO *dao.AITaskDAO, dispatcher *agent.Dispatcher, tplSvc *PromptTemplateService, knowledgeSvc *KnowledgeService, writingStyleSvc *WritingStyleService, plotStructureSvc *PlotStructureService, hitAnalysisSvc *HitAnalysisService, behaviorSvc *UserBehaviorService, intentSvc *IntentService, memorySvc *MemoryService, factSvc *NovelFactService) *NovelService {
	return &NovelService{
		novelDAO:         dao.NewNovelDAO(),
		aiTaskDAO:        aiTaskDAO,
		dispatcher:       dispatcher,
		tplSvc:           tplSvc,
		knowledgeSvc:     knowledgeSvc,
		writingStyleSvc:  writingStyleSvc,
		plotStructureSvc: plotStructureSvc,
		hitAnalysisSvc:   hitAnalysisSvc,
		behaviorSvc:      behaviorSvc,
		intentSvc:        intentSvc,
		memorySvc:        memorySvc,
		writerLevelSvc:   NewWriterLevelService(),
		factSvc:          factSvc,
		chapterReviewDAO: dao.NewChapterReviewDAO(),
	}
}

// SetChapterSummarySvc 注入递归摘要树服务（避免循环依赖）
func (s *NovelService) SetChapterSummarySvc(svc *ChapterSummaryService) {
	s.chapterSummarySvc = svc
}

// SetModelRegistry 注入模型注册服务
func (s *NovelService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// ========== 请求参数定义 ==========

// CreateNovelRequest 创建小说请求
type CreateNovelRequest struct {
	PortfolioID uint   `json:"portfolio_id" binding:"required"`
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description"`
}

// UpdateNovelRequest 更新小说请求
type UpdateNovelRequest struct {
	Title       string `json:"title" binding:"omitempty,max=200"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"omitempty,oneof=draft writing completed"`
	TokenBudget *int   `json:"token_budget"` // nil 表示不更新，0 表示不限制
}

// CreateChapterRequest 创建章节请求
type CreateChapterRequest struct {
	Title   string `json:"title" binding:"required,max=200"`
	Summary string `json:"summary"` // 可选，扩写章节时带入概要
}

// UpdateChapterRequest 更新章节请求
type UpdateChapterRequest struct {
	Title   string `json:"title" binding:"omitempty,max=200"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

// ReorderChaptersRequest 章节排序请求
type ReorderChaptersRequest struct {
	ChapterIDs []uint `json:"chapter_ids" binding:"required"`
}

// ChapterAIActionRequest AI 操作请求
type ChapterAIActionRequest struct {
	Action        string `json:"action" binding:"required,oneof=summary_polish polish expand continue"`
	ModelName     string `json:"model_name"`
	Summary       string `json:"summary"`
	Content       string `json:"content"`
	SelectedText  string `json:"selected_text"`                                        // 用户选中的文本片段，为空表示全文操作
	ScenePresetID *uint  `json:"scene_preset_id"`                                      // 可选的场景预设 ID
	PolishMode    string `json:"polish_mode" binding:"omitempty,oneof=dialogue pacing sensory emotion trim"` // 润色方向预设
}

// OutlineChapterAIRequest 大纲页面章节级 AI 操作请求
type OutlineChapterAIRequest struct {
	PortfolioID     uint   `json:"portfolio_id" binding:"required"`
	Action          string `json:"action" binding:"required,oneof=title_polish summary_polish summary_expand generate_characters generate_topic generate_storyline generate_characters_ensemble"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	Context         *struct {
		Setting      string `json:"setting"`
		PrevChapters []struct {
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"prev_chapters"`
		NextChapters []struct {
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"next_chapters"`
	} `json:"context"`
	ModelName       string `json:"model_name"`
	UserPrompt      string `json:"user_prompt"`       // 用户自定义指令，追加到 system prompt 末尾
	ButlerSessionID string `json:"butler_session_id"` // 管家会话 ID
	ConversationHistory []ConversationMessage `json:"conversation_history,omitempty"` // 对话历史（对话模式调整用）
}

// AcceptAIResultRequest 采纳 AI 结果请求
type AcceptAIResultRequest struct {
	TaskID uint `json:"task_id" binding:"required"`
}

// ExpandChaptersRequest 扩写章节目录请求
type ExpandChaptersRequest struct {
	NovelID     uint   `json:"novel_id"`
	Mode        string `json:"mode" binding:"required,oneof=append insert"` // append=末尾追加, insert=中间插入
	InsertAfter int    `json:"insert_after"`                                // insert 模式：在第几章后面插入（sort_order）
	ChapterNum  int    `json:"chapter_num" binding:"required,min=1,max=20"`
	ModelName   string `json:"model_name"`
	UserPrompt  string `json:"user_prompt"` // 用户自定义指令
}

// GenerateOutlineRequest 大纲生成请求
type GenerateOutlineRequest struct {
	PortfolioID         uint   `json:"portfolio_id" binding:"required"`
	Setting             string `json:"setting" binding:"required"` // 世界观/设定
	Characters          string `json:"characters"`                 // 人物梗概
	Background          string `json:"background"`                 // 背景信息
	Plot                string `json:"plot" binding:"required"`    // 剧情思路
	ChapterNum          int    `json:"chapter_num"`                // 期望章节数，默认30
	ModelName           string `json:"model_name"`
	UserPrompt          string `json:"user_prompt"`           // 用户自定义指令
	StructureTemplateID uint   `json:"structure_template_id"` // 可选：剧情结构模板 ID
	HitAnalysisID       uint   `json:"hit_analysis_id"`       // 可选：爆款拆解报告 ID
	IterationTaskID     uint   `json:"iteration_task_id"`     // 可选：上一轮大纲的 task ID（多轮迭代）
	Feedback            string `json:"feedback"`               // 可选：用户对上一轮大纲的反馈
	ButlerSessionID     string `json:"butler_session_id"`     // 管家会话 ID
}

// AdoptOutlineRequest 采用大纲请求
type AdoptOutlineRequest struct {
	PortfolioID      uint   `json:"portfolio_id" binding:"required"`
	TaskID           uint   `json:"task_id" binding:"required"`
	Title            string `json:"title" binding:"required"`
	Description      string `json:"description"`
	Source           string `json:"source"`            // manual, butler, outline；为空默认 manual
	ButlerTopic      string `json:"butler_topic"`      // 管家选题结果
	ButlerStoryline  string `json:"butler_storyline"`  // 管家故事线结果
	ButlerCharacters string `json:"butler_characters"` // 管家人物设计结果
	ButlerSessionID  string `json:"butler_session_id"` // 管家会话 ID
	Chapters         []struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
	} `json:"chapters" binding:"required"`
}

// RevertVersionRequest 回退版本请求
type RevertVersionRequest struct {
	VersionID uint `json:"version_id" binding:"required"`
}

// ========== Novel CRUD ==========

// CreateNovel 创建小说
func (s *NovelService) CreateNovel(userID, portfolioID uint, req *CreateNovelRequest) (*model.Novel, error) {
	novel := &model.Novel{
		PortfolioID: portfolioID,
		Title:       req.Title,
		Description: req.Description,
		Status:      model.NovelStatusDraft,
	}
	if err := s.novelDAO.CreateNovel(novel); err != nil {
		return nil, err
	}
	return novel, nil
}

// GetNovel 获取小说详情
func (s *NovelService) GetNovel(novelID uint) (*model.Novel, error) {
	return s.novelDAO.GetNovel(novelID)
}

// ListNovels 获取作品集下的小说列表，可选按 source 过滤
func (s *NovelService) ListNovels(portfolioID uint, source string) ([]model.Novel, error) {
	return s.novelDAO.ListNovelsByPortfolio(portfolioID, source)
}

// UpdateNovel 更新小说
func (s *NovelService) UpdateNovel(novelID uint, req *UpdateNovelRequest) (*model.Novel, error) {
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		novel.Title = req.Title
	}
	novel.Description = req.Description
	if req.Status != "" && model.ValidNovelStatuses[req.Status] {
		novel.Status = req.Status
	}
	if req.TokenBudget != nil {
		novel.TokenBudget = *req.TokenBudget
	}

	if err := s.novelDAO.UpdateNovel(novel); err != nil {
		return nil, err
	}
	return novel, nil
}

// DeleteNovel 删除小说
func (s *NovelService) DeleteNovel(novelID uint) error {
	return s.novelDAO.DeleteNovel(novelID)
}

// TokenUsageResponse Token 使用情况响应
type TokenUsageResponse struct {
	Budget     int     `json:"budget"`
	Used       int     `json:"used"`
	Percentage float64 `json:"percentage"` // 0-100
}

// GetTokenUsage 获取小说的 token 使用情况
func (s *NovelService) GetTokenUsage(ctx context.Context, novelID uint) (*TokenUsageResponse, error) {
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return nil, err
	}

	// 实时查询 token 使用总量
	used, err := s.aiTaskDAO.SumTokensByNovel(ctx, novelID)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	if novel.TokenUsed != used {
		_ = s.novelDAO.UpdateTokenUsed(novelID, used)
	}

	var percentage float64
	if novel.TokenBudget > 0 {
		percentage = float64(used) / float64(novel.TokenBudget) * 100
		if percentage > 100 {
			percentage = 100
		}
	}

	return &TokenUsageResponse{
		Budget:     novel.TokenBudget,
		Used:       used,
		Percentage: percentage,
	}, nil
}

// UpdateTokenBudget 更新小说 token 预算
func (s *NovelService) UpdateTokenBudget(novelID uint, budget int) error {
	return s.novelDAO.UpdateTokenBudget(novelID, budget)
}

// RefreshTokenUsed 刷新小说已用 token 缓存，并通过 notifier 推送更新
func (s *NovelService) RefreshTokenUsed(ctx context.Context, novelID uint, notifier interface {
	NotifyUserWithType(userID uint, msgType string, message interface{}) error
}) {
	used, err := s.aiTaskDAO.SumTokensByNovel(ctx, novelID)
	if err != nil {
		return
	}
	_ = s.novelDAO.UpdateTokenUsed(novelID, used)

	// 获取小说信息用于推送
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return
	}

	var percentage float64
	if novel.TokenBudget > 0 {
		percentage = float64(used) / float64(novel.TokenBudget) * 100
		if percentage > 100 {
			percentage = 100
		}
	}

	// 通过 WebSocket 推送 token_update 消息
	// 注意：这里无法获取 userID，通过 novel.PortfolioID 间接推送
	// 由于 Novel 没有 UserID 字段，我们在 dispatcher 回调中传入 task.UserID
	if notifier != nil {
		_ = notifier.NotifyUserWithType(0, "token_update", map[string]interface{}{
			"novel_id":   novelID,
			"budget":     novel.TokenBudget,
			"used":       used,
			"percentage": percentage,
		})
	}
}

// RefreshTokenUsedForUser 刷新小说已用 token 缓存，指定用户推送
func (s *NovelService) RefreshTokenUsedForUser(ctx context.Context, novelID uint, userID uint, notifier interface {
	NotifyUserWithType(userID uint, msgType string, message interface{}) error
}) {
	used, err := s.aiTaskDAO.SumTokensByNovel(ctx, novelID)
	if err != nil {
		return
	}
	_ = s.novelDAO.UpdateTokenUsed(novelID, used)

	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return
	}

	var percentage float64
	if novel.TokenBudget > 0 {
		percentage = float64(used) / float64(novel.TokenBudget) * 100
		if percentage > 100 {
			percentage = 100
		}
	}

	if notifier != nil {
		_ = notifier.NotifyUserWithType(userID, "token_update", map[string]interface{}{
			"novel_id":   novelID,
			"budget":     novel.TokenBudget,
			"used":       used,
			"percentage": percentage,
		})
	}
}

// ========== Chapter CRUD ==========

// CreateChapter 创建章节（自动设置排序序号）
func (s *NovelService) CreateChapter(userID uint, novelID uint, req *CreateChapterRequest) (*model.Chapter, error) {
	// 获取当前最大排序序号
	maxOrder, err := s.novelDAO.GetMaxSortOrder(novelID)
	if err != nil {
		return nil, err
	}

	chapter := &model.Chapter{
		NovelID:        novelID,
		Title:          req.Title,
		Summary:        req.Summary,
		SortOrder:      maxOrder + 1,
		Status:         model.ChapterStatusDraft,
		CurrentVersion: 1,
	}
	if err := s.novelDAO.CreateChapter(chapter); err != nil {
		return nil, err
	}

	// 更新写手章节统计 + 检查成长解锁
	if userID > 0 && s.writerLevelSvc != nil {
		go func() {
			_ = s.writerLevelSvc.UpdateStats(userID, 0, 1)
			_, _ = s.writerLevelSvc.CheckAndUpgrade(userID)
		}()
	}

	// 更新小说章节数
	s.refreshNovelStats(novelID)

	return chapter, nil
}

// UpdateChapter 更新章节（内容变化时自动创建 manual 版本）
func (s *NovelService) UpdateChapter(userID, chapterID uint, req *UpdateChapterRequest) (*model.Chapter, error) {
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return nil, err
	}

	// 记录旧内容用于 diff 统计
	oldContent := chapter.Content
	oldWordCount := chapter.WordCount

	// 检测内容是否有变化
	contentChanged := (req.Content != "" && req.Content != chapter.Content) || req.Summary != chapter.Summary

	if req.Title != "" {
		chapter.Title = req.Title
	}
	if req.Summary != "" || req.Content != "" {
		// 只在有明确传值时覆盖，避免部分更新时误清空
		if req.Content != "" {
			chapter.Content = req.Content
			chapter.WordCount = utf8.RuneCountInString(req.Content)
		}
		chapter.Summary = req.Summary
	}

	// 内容变化时创建版本记录
	if contentChanged {
		chapter.CurrentVersion++
		version := &model.ChapterVersion{
			ChapterID: chapterID,
			Version:   chapter.CurrentVersion,
			Content:   chapter.Content,
			Summary:   chapter.Summary,
			Source:    model.VersionSourceManual,
			WordCount: chapter.WordCount,
		}
		if err := s.novelDAO.CreateVersion(version); err != nil {
			return nil, err
		}

		// 记录 chapter_save 行为事件（含 diff 统计）
		if s.behaviorSvc != nil && userID > 0 {
			diffStats := computeDiffStats(oldContent, req.Content)
			go s.behaviorSvc.RecordEvent(userID, chapter.NovelID, chapterID, model.BehaviorChapterSave, map[string]interface{}{
				"word_count":     chapter.WordCount,
				"old_word_count": oldWordCount,
				"diff_stats":     diffStats,
				"content":        req.Content, // 用于偏好提取的词汇分析
			})
		}
	}

	if err := s.novelDAO.UpdateChapter(chapter); err != nil {
		return nil, err
	}

	// 更新写手创作统计（字数增量）
	if contentChanged && userID > 0 && s.writerLevelSvc != nil {
		wordDelta := int64(chapter.WordCount - oldWordCount)
		if wordDelta > 0 {
			go func() {
				_ = s.writerLevelSvc.UpdateStats(userID, wordDelta, 0)
				// 检查是否满足成长解锁条件
				_, _ = s.writerLevelSvc.CheckAndUpgrade(userID)
			}()
		}
	}

	// 更新小说字数统计
	s.refreshNovelStats(chapter.NovelID)

	// 异步采集动态记忆事实
	if contentChanged && s.factSvc != nil {
		novel, novelErr := s.novelDAO.GetNovel(chapter.NovelID)
		if novelErr == nil {
			go s.factSvc.CollectFromChapter(context.Background(), novel, chapter, userID)
		}
	}

	// 异步触发递归摘要树增量更新
	if contentChanged && s.chapterSummarySvc != nil {
		go func() {
			if err := s.chapterSummarySvc.UpdateIncremental(context.Background(), chapter.NovelID, chapter.SortOrder); err != nil {
				log.Printf("[novel] 摘要树增量更新失败: %v", err)
			}
		}()
	}

	return chapter, nil
}

// DeleteChapter 删除章节
func (s *NovelService) DeleteChapter(chapterID uint) error {
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return err
	}

	if err := s.novelDAO.DeleteChapter(chapterID); err != nil {
		return err
	}

	s.refreshNovelStats(chapter.NovelID)
	return nil
}

// ListChapters 获取小说下的章节列表
func (s *NovelService) ListChapters(novelID uint) ([]model.Chapter, error) {
	return s.novelDAO.ListChaptersByNovel(novelID)
}

// ReorderChapters 重新排序章节
func (s *NovelService) ReorderChapters(novelID uint, req *ReorderChaptersRequest) error {
	return s.novelDAO.ReorderChapters(novelID, req.ChapterIDs)
}

// ========== AI 操作 ==========

// 章节 AI 操作的合法 action 白名单
var validChapterActions = map[string]string{
	"summary_polish": model.TaskTypeChapterSummaryPolish,
	"polish":         model.TaskTypeChapterPolish,
	"expand":         model.TaskTypeChapterExpand,
	"continue":       model.TaskTypeChapterContinue,
}

// 润色方向预设指令映射
var polishModeInstructions = map[string]string{
	"dialogue": "侧重对话润色：提升对话自然度，区分不同角色的语气和用词习惯，适当口语化，让对话更贴合人物性格。",
	"pacing":   "侧重节奏调整：优化长短句交替节奏，改善场景切换的过渡，让叙事张弛有度、松紧得当。",
	"sensory":  "侧重感官强化：丰富视觉、听觉、触觉、嗅觉等感官细节，增强画面感和沉浸感。",
	"emotion":  "侧重情感深化：加强心理描写和情感冲突的刻画，丰富内心独白，让情感表达更细腻有层次。",
	"trim":     "侧重精简去冗：删除冗余描写和重复表达，收紧语言，让文字更干练有力。",
}

// ChapterAIAction 对章节执行 AI 操作
func (s *NovelService) ChapterAIAction(ctx context.Context, userID, chapterID uint, req *ChapterAIActionRequest) (uint, error) {
	taskType, ok := validChapterActions[req.Action]
	if !ok {
		return 0, errors.New("invalid action")
	}

	// 获取当前章节
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return 0, fmt.Errorf("chapter not found: %w", err)
	}

	// 使用前端传来的最新编辑内容（用户可能还没保存）
	if req.Summary != "" {
		chapter.Summary = req.Summary
	}
	if req.Content != "" {
		chapter.Content = req.Content
	}

	// 获取所属小说
	novel, err := s.novelDAO.GetNovel(chapter.NovelID)
	if err != nil {
		return 0, fmt.Errorf("novel not found: %w", err)
	}

	// 获取前几章的概要作为上下文
	prevChapters, _ := s.novelDAO.GetPreviousChapters(chapter.NovelID, chapter.SortOrder, 5)

	// 构建模板渲染数据（按 action 差异化上下文）
	tplData := s.buildTemplateData(novel, chapter, prevChapters, req.SelectedText, req.Action)

	// summary_polish 额外注入后文章节概要，帮助 AI 理解前后文脉络
	if req.Action == "summary_polish" {
		nextChapters, _ := s.novelDAO.GetNextChapters(chapter.NovelID, chapter.SortOrder, 3)
		var nextParts []string
		for _, ch := range nextChapters {
			if ch.Summary != "" {
				nextParts = append(nextParts, fmt.Sprintf("【第%d章 %s】%s", ch.SortOrder, ch.Title, ch.Summary))
			}
		}
		if len(nextParts) > 0 {
			tplData.NextChapters = strings.Join(nextParts, "\n")
		}
	}

	// 注入润色方向指令
	if req.Action == "polish" && req.PolishMode != "" {
		tplData.PolishMode = req.PolishMode
		tplData.PolishModeInstruction = polishModeInstructions[req.PolishMode]
	}

	// 注入写作风格（含用户偏好）
	if s.writingStyleSvc != nil {
		// 优先使用请求中指定的场景预设，其次使用章节绑定的默认场景预设
		presetID := req.ScenePresetID
		if presetID == nil {
			presetID = chapter.ScenePresetID
		}
		tplData.WritingStyle = s.writingStyleSvc.FormatStyleWithUserPref(novel.ID, presetID, userID)
	}

	// 注入创作意图
	if s.intentSvc != nil && chapter.Content != "" {
		intent := s.intentSvc.InferIntent(chapter.Content, novel.ID)
		intentText := s.intentSvc.FormatIntentForPrompt(intent)
		if intentText != "" {
			tplData.WritingStyle += "\n" + intentText
		}
	}

	// 使用模板渲染 user prompt
	userPrompt, err := s.tplSvc.RenderPrompt(novel.ID, req.Action, "user", tplData)
	if err != nil {
		// fallback 到硬编码逻辑
		userPrompt = s.buildUserPrompt(req.Action, novel, chapter, prevChapters, req.SelectedText)
	}

	// 使用模板渲染 system prompt，写入 History JSON 传递给 dispatcher
	systemPrompt, sysErr := s.tplSvc.RenderPrompt(novel.ID, req.Action, "system", tplData)
	historyJSON := s.buildChapterContext(novel, chapter, prevChapters)
	if sysErr == nil && systemPrompt != "" {
		// 将 system_prompt 注入到 history JSON 中
		var historyData map[string]interface{}
		if err := json.Unmarshal([]byte(historyJSON), &historyData); err == nil {
			historyData["system_prompt"] = systemPrompt
			if data, err := json.Marshal(historyData); err == nil {
				historyJSON = string(data)
			}
		}
	}

	modelName := req.ModelName
	if s.modelRegistry != nil {
		modelName = s.modelRegistry.ResolveModel(req.ModelName, model.CapTextGen)
	} else if modelName == "" {
		modelName = "zhipu"
	}

	// 创建 AI 任务
	task := &model.AITask{
		UserID:      userID,
		PortfolioID: novel.PortfolioID,
		NovelID:     novel.ID,
		TaskType:    taskType,
		ModelName:   modelName,
		Prompt:      userPrompt,
		History:     historyJSON,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// AcceptAIResult 采纳 AI 生成结果
func (s *NovelService) AcceptAIResult(ctx context.Context, userID, chapterID, taskID uint) error {
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return err
	}

	// 从 AITaskDAO 获取任务结果
	task, err := s.aiTaskDAO.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}
	if task.Status != model.TaskStatusCompleted {
		return errors.New("task is not completed")
	}

	// 解析 AI 结果
	var result struct {
		Content string `json:"content"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(task.Result), &result); err != nil {
		return fmt.Errorf("failed to parse task result: %w", err)
	}

	// 确定版本来源
	sourceMap := map[string]string{
		model.TaskTypeChapterSummaryPolish: model.VersionSourceAIOutline,
		model.TaskTypeChapterPolish:        model.VersionSourceAIPolish,
		model.TaskTypeChapterExpand:        model.VersionSourceAIExpand,
		model.TaskTypeChapterContinue:      model.VersionSourceAIContinue,
	}
	source := sourceMap[task.TaskType]
	if source == "" {
		source = model.VersionSourceManual
	}

	// 创建新版本
	chapter.CurrentVersion++
	version := &model.ChapterVersion{
		ChapterID: chapterID,
		Version:   chapter.CurrentVersion,
		Content:   result.Content,
		Summary:   result.Summary,
		Source:    source,
		TaskID:    &taskID,
		WordCount: utf8.RuneCountInString(result.Content),
	}
	if err := s.novelDAO.CreateVersion(version); err != nil {
		return err
	}

	// 更新章节内容
	chapter.Content = result.Content
	if result.Summary != "" {
		chapter.Summary = result.Summary
	}
	chapter.WordCount = version.WordCount
	if err := s.novelDAO.UpdateChapter(chapter); err != nil {
		return err
	}

	// 记录 ai_accept 行为事件
	if s.behaviorSvc != nil {
		go s.behaviorSvc.RecordEvent(userID, chapter.NovelID, chapterID, model.BehaviorAIAccept, map[string]interface{}{
			"action":  task.TaskType,
			"model":   task.ModelName,
			"task_id": taskID,
		})
	}

	s.refreshNovelStats(chapter.NovelID)
	return nil
}

// ========== 版本管理 ==========

// ListVersions 获取章节的版本历史
func (s *NovelService) ListVersions(chapterID uint) ([]model.ChapterVersion, error) {
	return s.novelDAO.ListVersionsByChapter(chapterID)
}

// RevertToVersion 回退到指定版本
func (s *NovelService) RevertToVersion(chapterID, versionID uint) error {
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return err
	}

	version, err := s.novelDAO.GetVersion(versionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}
	if version.ChapterID != chapterID {
		return errors.New("version does not belong to this chapter")
	}

	// 创建新版本记录（回退也产生新版本）
	chapter.CurrentVersion++
	newVersion := &model.ChapterVersion{
		ChapterID: chapterID,
		Version:   chapter.CurrentVersion,
		Content:   version.Content,
		Summary:   version.Summary,
		Source:    model.VersionSourceManual,
		WordCount: version.WordCount,
	}
	if err := s.novelDAO.CreateVersion(newVersion); err != nil {
		return err
	}

	// 更新章节内容
	chapter.Content = version.Content
	chapter.Summary = version.Summary
	chapter.WordCount = version.WordCount
	if err := s.novelDAO.UpdateChapter(chapter); err != nil {
		return err
	}

	s.refreshNovelStats(chapter.NovelID)
	return nil
}

// ========== 内部辅助方法 ==========

// refreshNovelStats 刷新小说的章节数和字数统计
func (s *NovelService) refreshNovelStats(novelID uint) {
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return
	}

	count, _ := s.novelDAO.CountChaptersByNovel(novelID)
	wordCount, _ := s.novelDAO.SumWordCountByNovel(novelID)

	novel.ChapterCount = int(count)
	novel.WordCount = wordCount
	_ = s.novelDAO.UpdateNovel(novel)
}

// buildChapterContext 构建章节上下文 JSON（小说背景 + 前几章概要 + 前一章正文）
func (s *NovelService) buildChapterContext(novel *model.Novel, chapter *model.Chapter, prevChapters []model.Chapter) string {
	type chapterCtx struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
		Content string `json:"content,omitempty"`
	}
	ctx := struct {
		NovelTitle       string       `json:"novel_title"`
		NovelDescription string       `json:"novel_description"`
		ChapterTitle     string       `json:"chapter_title"`
		ChapterSummary   string       `json:"chapter_summary"`
		PrevChapters     []chapterCtx `json:"prev_chapters"`
	}{
		NovelTitle:       novel.Title,
		NovelDescription: novel.Description,
		ChapterTitle:     chapter.Title,
		ChapterSummary:   chapter.Summary,
	}
	for i, ch := range prevChapters {
		c := chapterCtx{
			Title:   ch.Title,
			Summary: ch.Summary,
		}
		// 最后一章（即紧邻的前一章）附带正文，保证情节衔接
		if i == len(prevChapters)-1 {
			c.Content = ch.Content
		}
		ctx.PrevChapters = append(ctx.PrevChapters, c)
	}
	data, _ := json.Marshal(ctx)
	return string(data)
}

// buildTemplateData 构建模板渲染所需的数据，按 action 差异化上下文以节省 Token
func (s *NovelService) buildTemplateData(novel *model.Novel, chapter *model.Chapter, prevChapters []model.Chapter, selectedText, action string) *model.PromptTemplateData {
	// 使用 Token Budget Manager 分配各模块预算
	budgetModel := "qwen"
	if s.modelRegistry != nil {
		budgetModel = s.modelRegistry.GetDefaultModel(model.CapTextGen)
	}
	budget := agent.NewTokenBudget(budgetModel) // buildTemplateData 用于单章 AI 任务

	// polish 不需要前文概要和前文正文；expand/continue 限制前 2 章、500 字
	var prevSummaries []string
	var prevContent string

	if action != "polish" {
		// 拼接前文概要（限制前 2 章）
		prevBudget := budget.CharsForPrevContext()
		prevCharsUsed := 0
		maxPrevChapters := 2
		count := 0
		for _, ch := range prevChapters {
			if count >= maxPrevChapters {
				break
			}
			if ch.Summary != "" {
				entry := fmt.Sprintf("【%s】%s", ch.Title, ch.Summary)
				entryChars := len([]rune(entry))
				if prevCharsUsed+entryChars > prevBudget {
					break
				}
				prevSummaries = append(prevSummaries, entry)
				prevCharsUsed += entryChars
				count++
			}
		}

		// 获取前一章正文（截取末尾 500 字）
		if len(prevChapters) > 0 {
			last := prevChapters[len(prevChapters)-1]
			runes := []rune(last.Content)
			const maxPrevContent = 500
			if len(runes) > maxPrevContent {
				prevContent = string(runes[len(runes)-maxPrevContent:])
			} else {
				prevContent = last.Content
			}
		}
	}

	wordCount := utf8.RuneCountInString(chapter.Content)
	targetWords := 3000
	if wordCount >= 3000 {
		targetWords = wordCount + 1000
	}

	// 注入知识库上下文（RAG Phase 1 — 按 Knowledge 预算截断）
	// polish 仅获取 characters；expand/continue 获取完整 knowledgeContext
	var knowledgeContext, characters, worldviewNotes string
	if s.knowledgeSvc != nil {
		knowledgeContext, characters, worldviewNotes = s.knowledgeSvc.BuildKnowledgeContext(novel.ID, budget.CharsForKnowledge())
		if action == "polish" {
			// polish 不需要 knowledgeContext 和 worldviewNotes，仅保留 characters
			knowledgeContext = ""
			worldviewNotes = ""
		}
	}

	// polish 跳过 RAG Phase 2/3，仅 expand/continue 注入
	if action != "polish" {
		// 注入记忆语义检索结果（RAG Phase 2 - Embedding — 按 Memory 预算截断）
		if s.memorySvc != nil {
			bindings, _ := s.memorySvc.ListBindings(novel.ID)
			if len(bindings) > 0 {
				queryText := chapter.Title + " " + chapter.Summary
				if queryText == " " {
					queryText = chapter.Content
					runes := []rune(queryText)
					if len(runes) > 500 {
						queryText = string(runes[:500])
					}
				}

				var ragChunks []string
				memoryCharsUsed := 0
				memoryBudget := budget.CharsForMemory()
				for _, b := range bindings {
					embModel := "qwen"
					if s.modelRegistry != nil {
						embModel = s.modelRegistry.GetDefaultModel(model.CapEmbedding)
					}
					chunks, err := s.memorySvc.GetRelevantChunks(
						context.Background(), b.MemoryID, queryText, embModel, 2,
					)
					if err != nil {
						log.Printf("[novel] embedding retrieval failed for memory %d: %v", b.MemoryID, err)
						continue
					}
					for _, chunk := range chunks {
						chunkChars := len([]rune(chunk))
						if memoryCharsUsed+chunkChars > memoryBudget {
							break
						}
						ragChunks = append(ragChunks, chunk)
						memoryCharsUsed += chunkChars
					}
				}

				if len(ragChunks) > 0 {
					ragContext := "【记忆语义检索】\n" + strings.Join(ragChunks, "\n---\n")
					knowledgeContext += "\n\n" + ragContext
				}
			}
		}

		// 注入动态记忆事实（RAG Phase 3 - Milvus 向量检索 — 按 Facts 预算截断）
		if s.factSvc != nil {
			dynamicContext := s.factSvc.Retrieve(context.Background(), novel.ID, chapter)
			if dynamicContext != "" {
				runes := []rune(dynamicContext)
				factsBudget := budget.CharsForFacts()
				if len(runes) > factsBudget {
					dynamicContext = string(runes[:factsBudget])
				}
				knowledgeContext += "\n\n" + dynamicContext
			}
		}
	}

	// 注入历史审核问题上下文（提取最近一次 review 中低分维度的 issues）
	var reviewContext string
	if s.chapterReviewDAO != nil && chapter.ID > 0 {
		reviewContext = s.buildReviewContext(chapter.ID)
	}

	return &model.PromptTemplateData{
		NovelTitle:       novel.Title,
		NovelDescription: novel.Description,
		ChapterTitle:     chapter.Title,
		ChapterSummary:   chapter.Summary,
		ChapterContent:   chapter.Content,
		PrevSummaries:    strings.Join(prevSummaries, "\n"),
		PrevContent:      prevContent,
		WordCount:        wordCount,
		TargetWords:      targetWords,
		SelectedText:     selectedText,
		KnowledgeContext: knowledgeContext,
		Characters:       characters,
		WorldviewNotes:   worldviewNotes,
		ReviewContext:    reviewContext,
	}
}

// buildReviewContext 从最近一次审核记录中提取低分维度的 issues，格式化为可读文本
// 仅提取 score < 75 的维度，限制总长度 1500 字符
func (s *NovelService) buildReviewContext(chapterID uint) string {
	review, err := s.chapterReviewDAO.GetLatestByChapter(chapterID)
	if err != nil || review == nil {
		return ""
	}

	type dimInfo struct {
		Name   string
		Score  int
		Issues string
	}
	dims := []dimInfo{
		{"情节连贯性", review.PlotScore, review.PlotIssues},
		{"叙事质量", review.NarrativeScore, review.NarrativeIssues},
		{"排版规范", review.FormattingScore, review.FormattingIssues},
		{"AI痕迹", review.AIArtifactsScore, review.AIArtifactsIssues},
	}

	var parts []string
	totalChars := 0
	const maxChars = 1500
	for _, d := range dims {
		if d.Score >= 75 || d.Score == 0 {
			continue
		}
		entry := fmt.Sprintf("- %s（%d分）：%s", d.Name, d.Score, d.Issues)
		entryChars := len([]rune(entry))
		if totalChars+entryChars > maxChars {
			break
		}
		parts = append(parts, entry)
		totalChars += entryChars
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// buildUserPrompt 根据 action 构建用户 prompt（硬编码 fallback）
// 所有操作都包含：小说背景 + 前文概要 + 前一章正文摘要，保证情节连续性
func (s *NovelService) buildUserPrompt(action string, novel *model.Novel, chapter *model.Chapter, prevChapters []model.Chapter, selectedText string) string {
	// 拼接前文概要
	var prevSummaries []string
	for _, ch := range prevChapters {
		if ch.Summary != "" {
			prevSummaries = append(prevSummaries, fmt.Sprintf("【%s】%s", ch.Title, ch.Summary))
		}
	}
	prevText := strings.Join(prevSummaries, "\n")

	// 获取前一章正文（截取末尾部分，避免 token 过长）
	var prevContent string
	if len(prevChapters) > 0 {
		last := prevChapters[len(prevChapters)-1]
		runes := []rune(last.Content)
		if len(runes) > 2000 {
			prevContent = string(runes[len(runes)-2000:])
		} else {
			prevContent = last.Content
		}
	}

	// 公共上下文头
	contextHeader := fmt.Sprintf("【小说背景】\n%s\n\n【前文各章概要】\n%s", novel.Description, prevText)
	if prevContent != "" {
		contextHeader += fmt.Sprintf("\n\n【前一章末尾内容】\n%s", prevContent)
	}

	// 注入知识库上下文（RAG Phase 1）
	if s.knowledgeSvc != nil {
		knowledgeCtx, characters, worldview := s.knowledgeSvc.BuildKnowledgeContext(novel.ID, 3000)
		if characters != "" {
			contextHeader += fmt.Sprintf("\n\n【人物档案】\n%s", characters)
		}
		if worldview != "" {
			contextHeader += fmt.Sprintf("\n\n【世界观设定】\n%s", worldview)
		}
		if knowledgeCtx != "" && characters == "" && worldview == "" {
			contextHeader += fmt.Sprintf("\n\n【知识库参考】\n%s", knowledgeCtx)
		}
	}

	switch action {
	case "summary_polish":
		return fmt.Sprintf("%s\n\n【当前章节】%s\n【当前章节概要】\n%s\n\n请对以上概要进行深度润色和扩写，要求：\n1. 结合角色档案，明确本章出场人物的行为动机和情感状态\n2. 与前文概要衔接，交代因果关系\n3. 融入知识库中与本章相关的伏笔和剧情线索\n4. 补充关键场景的环境要素和氛围基调\n5. 润色后概要应在300-400字之间，信息密度高，可直接指导正文写作\n\n直接输出润色后的概要文本。",
			contextHeader, chapter.Title, chapter.Summary)
	case "polish":
		if selectedText != "" {
			return fmt.Sprintf("%s\n\n【当前章节】%s\n【章节概要】%s\n【选中片段】\n%s\n\n请仅对以上选中片段进行润色，保持上下文语境一致。只输出替换后的片段文本，不要输出完整章节。",
				contextHeader, chapter.Title, chapter.Summary, selectedText)
		}
		return fmt.Sprintf("%s\n\n【当前章节】%s\n【章节概要】%s\n【原文】\n%s\n\n请基于章节概要和前文上下文对内容进行润色，提升文学性和可读性，保持情节连续性和人物一致性。",
			contextHeader, chapter.Title, chapter.Summary, chapter.Content)
	case "expand":
		wordCount := utf8.RuneCountInString(chapter.Content)
		targetWords := 3000
		if wordCount >= 3000 {
			targetWords = wordCount + 1000
		}
		if selectedText != "" {
			return fmt.Sprintf("%s\n\n【当前章节】%s\n【章节概要】%s\n【选中片段】\n%s\n\n请仅对以上选中片段进行扩写，丰富细节描写和对话，保持上下文语境一致。只输出替换后的片段文本，不要输出完整章节。",
				contextHeader, chapter.Title, chapter.Summary, selectedText)
		}
		return fmt.Sprintf("%s\n\n【当前章节】%s\n【章节概要】%s\n【当前内容（%d字）】\n%s\n\n请在保留原文所有情节和对话的基础上进行扩写，将内容从%d字扩充到%d字以上。扩写方法：\n1. 为每个场景补充感官细节（视觉、听觉、触觉）\n2. 扩展关键对话，增加角色的微表情、动作和心理活动\n3. 在场景转换处补充过渡段落\n4. 丰富环境描写和氛围渲染\n不要删减或改写原有内容，只做增量扩充。",
			contextHeader, chapter.Title, chapter.Summary, wordCount, chapter.Content, wordCount, targetWords)
	case "continue":
		return fmt.Sprintf("%s\n\n【当前章节】%s\n【章节概要】%s\n【当前章节已有内容】\n%s\n\n请严格按照章节概要续写约 500-1000 字，确保与前文情节自然衔接，不要偏离概要设定的情节方向。",
			contextHeader, chapter.Title, chapter.Summary, chapter.Content)
	default:
		return ""
	}
}

// ========== 大纲生成 ==========

// GenerateOutline 生成小说大纲：构建 prompt，创建 AITask，异步 Dispatch
func (s *NovelService) GenerateOutline(ctx context.Context, userID uint, req *GenerateOutlineRequest) (uint, error) {
	if req.ChapterNum <= 0 {
		req.ChapterNum = 30
	}

	modelName := req.ModelName
	if s.modelRegistry != nil {
		modelName = s.modelRegistry.ResolveModel(req.ModelName, model.CapTextGen)
	} else if modelName == "" {
		modelName = "zhipu"
	}

	// 增强大纲：注入剧情结构模板骨架
	var structureSkeleton string
	if req.StructureTemplateID > 0 && s.plotStructureSvc != nil {
		skeleton, err := s.plotStructureSvc.FormatStructureForPrompt(req.StructureTemplateID)
		if err == nil {
			structureSkeleton = skeleton
		}
	}

	// 增强大纲：注入爆款拆解参考
	var hitAnalysisRef string
	if req.HitAnalysisID > 0 && s.hitAnalysisSvc != nil {
		ref, err := s.hitAnalysisSvc.FormatReportForPrompt(ctx, req.HitAnalysisID, userID)
		if err == nil {
			hitAnalysisRef = ref
		}
	}

	// 增强大纲：多轮迭代（获取上一轮大纲结果）
	var prevOutline, userFeedback string
	if req.IterationTaskID > 0 && req.Feedback != "" {
		prevTask, err := s.aiTaskDAO.GetTask(ctx, req.IterationTaskID)
		if err == nil && prevTask.Result != "" {
			prevOutline = prevTask.Result
			userFeedback = req.Feedback
		}
	}

	// 构建模板渲染数据
	tplData := &model.PromptTemplateData{
		Setting:           req.Setting,
		Characters:        req.Characters,
		Background:        req.Background,
		Plot:              req.Plot,
		ChapterNum:        req.ChapterNum,
		UserInstruction:   req.UserPrompt,
		StructureSkeleton: structureSkeleton,
		HitAnalysisRef:    hitAnalysisRef,
		PrevOutline:       prevOutline,
		UserFeedback:      userFeedback,
	}

	// 渲染 system prompt（模板优先，fallback 到硬编码）
	systemPrompt, sysErr := s.tplSvc.RenderPrompt(0, "outline_generate", "system", tplData)
	if sysErr != nil {
		systemPrompt = "你是一位专业的小说策划师。根据用户提供的设定、人物和剧情思路，生成一个完整的小说大纲。\n每个章节的 title 必须是具体的、有吸引力的标题（如\"暗夜追踪\"、\"命运的抉择\"），绝对不能使用\"章节标题\"、\"章节题目\"等占位符。\n你必须严格按照以下 JSON 格式输出，不要包含任何其他文字：\n[\n  {\"title\": \"第一章 暗夜追踪\", \"summary\": \"100-200字的章节概要...\"},\n  {\"title\": \"第二章 命运的抉择\", \"summary\": \"100-200字的章节概要...\"}\n]"
	}

	// 注入剧情结构模板骨架到 system prompt
	if structureSkeleton != "" {
		systemPrompt += "\n\n" + structureSkeleton
	}

	// 注入爆款拆解参考到 system prompt
	if hitAnalysisRef != "" {
		systemPrompt += "\n\n" + hitAnalysisRef
	}

	// 注入多轮迭代上下文到 system prompt
	if prevOutline != "" && userFeedback != "" {
		systemPrompt += "\n\n【上一轮大纲】\n" + prevOutline
		systemPrompt += "\n\n【用户反馈】\n" + userFeedback
		systemPrompt += "\n\n请根据用户反馈对大纲进行优化调整，保留合理部分，修改不满意的部分。"
	}

	// 追加用户自定义指令
	if req.UserPrompt != "" {
		systemPrompt += "\n\n【用户额外指令】\n" + req.UserPrompt
	}

	// 渲染 user prompt（模板优先，fallback 到硬编码）
	userPrompt, userErr := s.tplSvc.RenderPrompt(0, "outline_generate", "user", tplData)
	if userErr != nil {
		userPrompt = s.buildOutlinePrompt(req)
	}

	// 将 system prompt 通过 History JSON 传给 executor
	historyData := map[string]interface{}{
		"system_prompt": systemPrompt,
		"chapter_num":   req.ChapterNum,
	}
	historyJSON, _ := json.Marshal(historyData)

	task := &model.AITask{
		UserID:          userID,
		PortfolioID:     req.PortfolioID,
		TaskType:        model.TaskTypeOutlineGenerate,
		ModelName:       modelName,
		Prompt:          userPrompt,
		History:         string(historyJSON),
		Status:          model.TaskStatusPending,
		ButlerSessionID: req.ButlerSessionID,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch outline task failed: %w", err)
	}

	return task.ID, nil
}

// AdoptOutline 采用大纲：事务内创建 Novel + 批量创建 Chapters
func (s *NovelService) AdoptOutline(ctx context.Context, userID uint, req *AdoptOutlineRequest) (*model.Novel, error) {
	if len(req.Chapters) == 0 {
		return nil, errors.New("chapters cannot be empty")
	}

	// 确定来源，默认 manual
	source := req.Source
	if source == "" {
		source = model.NovelSourceManual
	}

	// 创建小说
	novel := &model.Novel{
		PortfolioID:      req.PortfolioID,
		Title:            req.Title,
		Description:      req.Description,
		Status:           model.NovelStatusDraft,
		Source:           source,
		ButlerTopic:      req.ButlerTopic,
		ButlerStoryline:  req.ButlerStoryline,
		ButlerCharacters: req.ButlerCharacters,
	}
	if err := s.novelDAO.CreateNovel(novel); err != nil {
		return nil, fmt.Errorf("create novel failed: %w", err)
	}

	// 批量创建章节
	chapters := make([]model.Chapter, len(req.Chapters))
	for i, ch := range req.Chapters {
		chapters[i] = model.Chapter{
			NovelID:        novel.ID,
			Title:          ch.Title,
			Summary:        ch.Summary,
			SortOrder:      i + 1,
			Status:         model.ChapterStatusDraft,
			CurrentVersion: 1,
		}
	}
	if err := s.novelDAO.BatchCreateChapters(chapters); err != nil {
		return nil, fmt.Errorf("batch create chapters failed: %w", err)
	}

	// 刷新小说统计
	s.refreshNovelStats(novel.ID)

	// 按 butler_session_id 批量回填所有关联任务的 novel_id
	if req.ButlerSessionID != "" {
		if err := s.aiTaskDAO.UpdateNovelIDBySessionID(ctx, req.ButlerSessionID, novel.ID); err != nil {
			log.Printf("[AdoptOutline] 回填 butler_session_id=%s 的 novel_id 失败: %v", req.ButlerSessionID, err)
		}
	}

	return novel, nil
}

// 大纲页面章节级 AI 操作的合法 action 白名单
var validOutlineChapterActions = map[string]string{
	"title_polish":                model.TaskTypeOutlineTitlePolish,
	"summary_polish":              model.TaskTypeOutlineSummaryPolish,
	"summary_expand":              model.TaskTypeOutlineSummaryExpand,
	"generate_characters":         model.TaskTypeOutlineGenerateCharacters,
	"generate_topic":              model.TaskTypeButlerGenerateTopic,
	"generate_storyline":          model.TaskTypeButlerGenerateStoryline,
	"generate_characters_ensemble": model.TaskTypeButlerGenerateCharacters,
}

// OutlineChapterAIAction 大纲页面章节级 AI 操作（不依赖 chapter ID）
func (s *NovelService) OutlineChapterAIAction(ctx context.Context, userID uint, req *OutlineChapterAIRequest) (uint, error) {
	taskType, ok := validOutlineChapterActions[req.Action]
	if !ok {
		return 0, errors.New("invalid action")
	}

	// 映射 action 到模板 action（加 outline_ 前缀）
	tplAction := "outline_" + req.Action

	// 构建模板渲染数据
	tplData := &model.PromptTemplateData{
		ChapterTitle:    req.Title,
		ChapterSummary:  req.Summary,
		UserInstruction: req.UserPrompt,
	}
	if req.Context != nil {
		tplData.Setting = req.Context.Setting
		// 格式化前文章节
		var prevParts []string
		for _, ch := range req.Context.PrevChapters {
			prevParts = append(prevParts, fmt.Sprintf("- %s：%s", ch.Title, ch.Summary))
		}
		tplData.PrevChapters = strings.Join(prevParts, "\n")
		// 格式化后续章节
		var nextParts []string
		for _, ch := range req.Context.NextChapters {
			nextParts = append(nextParts, fmt.Sprintf("- %s：%s", ch.Title, ch.Summary))
		}
		tplData.NextChapters = strings.Join(nextParts, "\n")
	}

	// 渲染 system prompt（模板优先，fallback 到硬编码）
	systemPrompt, sysErr := s.tplSvc.RenderPrompt(0, tplAction, "system", tplData)
	if sysErr != nil {
		// fallback 到原始硬编码
		switch req.Action {
		case "title_polish":
			systemPrompt = "你是一位专业的小说策划师。对用户提供的章节标题进行润色，使其更加精炼、有吸引力，同时保持与章节内容的关联性。只输出润色后的标题文本，不要包含任何解释或额外内容。"
		case "summary_polish":
			systemPrompt = "你是一位专业的小说策划师。对用户提供的章节概要进行润色，使其更加清晰、连贯、有吸引力，保持原有情节方向不变。只输出润色后的概要文本，不要包含标题或额外说明。"
		case "summary_expand":
			systemPrompt = "你是一位专业的小说策划师。对用户提供的章节概要进行扩写，丰富情节细节、人物动机和场景描写，使概要更加充实完整。只输出扩写后的概要文本，不要包含标题或额外说明。"
		case "generate_characters":
			systemPrompt = "你是一位专业的小说策划师。根据用户提供的世界观/设定、背景信息和剧情思路，设计主要核心人物。为每个人物提供：姓名、身份/职业、性格特点、人物关系、在故事中的角色定位。人物之间要有合理的关系网络和冲突张力。只输出人物设定文本，不要包含额外说明。"
		case "generate_topic":
			systemPrompt = "你是一位专业的小说策划师。根据用户提供的创作方向和偏好，生成一个完整的选题方案。方案需包含：小说标题、题材类型、核心卖点（3-5个）、目标读者画像。输出纯文本，不要使用 JSON 格式，不要包含额外说明。"
		case "generate_storyline":
			systemPrompt = "你是一位专业的小说策划师。根据用户提供的选题信息和创作方向，生成一个完整的故事线方案。方案需包含：世界观设定、背景信息、剧情大纲（起承转合）。输出纯文本，不要使用 JSON 格式，不要包含额外说明。"
		case "generate_characters_ensemble":
			systemPrompt = "你是一位专业的小说策划师。根据用户提供的故事线和创作方向，生成一套完整的人物群像设定。为每个人物提供：姓名、身份/职业、性格特点、人物关系、在故事中的角色定位、人物弧光。人物之间要有合理的关系网络和冲突张力。输出纯文本，不要使用 JSON 格式，不要包含额外说明。"
		}
	}
	// 追加用户自定义指令
	if req.UserPrompt != "" {
		systemPrompt += "\n\n【用户额外指令】\n" + req.UserPrompt
	}

	// 渲染 user prompt（模板优先，fallback 到硬编码）
	userPrompt, userErr := s.tplSvc.RenderPrompt(0, tplAction, "user", tplData)
	if userErr != nil {
		userPrompt = s.buildOutlineChapterPrompt(req)
	}

	// 对话模式：注入对话历史到 userPrompt（仅 generate_topic 等管家步骤）
	if len(req.ConversationHistory) > 0 {
		if historyText := formatConversationHistory(req.ConversationHistory); historyText != "" {
			userPrompt += "\n\n" + historyText
		}
	}

	// 构建 History JSON（包含 system_prompt + 上下文）
	historyMap := map[string]interface{}{
		"system_prompt": systemPrompt,
	}
	if req.Context != nil {
		ctxData, err := json.Marshal(req.Context)
		if err == nil {
			var ctxMap map[string]interface{}
			if json.Unmarshal(ctxData, &ctxMap) == nil {
				for k, v := range ctxMap {
					historyMap[k] = v
				}
			}
		}
	}
	historyJSON, _ := json.Marshal(historyMap)

	modelName := req.ModelName
	if s.modelRegistry != nil {
		modelName = s.modelRegistry.ResolveModel(req.ModelName, model.CapTextGen)
	} else if modelName == "" {
		modelName = "zhipu"
	}

	task := &model.AITask{
		UserID:          userID,
		PortfolioID:     req.PortfolioID,
		TaskType:        taskType,
		ModelName:       modelName,
		Prompt:          userPrompt,
		History:         string(historyJSON),
		ButlerSessionID: req.ButlerSessionID,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// buildOutlineChapterPrompt 构建大纲章节级操作的 user prompt
func (s *NovelService) buildOutlineChapterPrompt(req *OutlineChapterAIRequest) string {
	var sb strings.Builder

	// 上下文信息：管家步骤使用"创作方向"标签，其他操作使用"故事设定"
	if req.Context != nil {
		if req.Context.Setting != "" {
			settingLabel := "故事设定"
			switch req.Action {
			case "generate_topic", "generate_storyline", "generate_characters_ensemble":
				settingLabel = "创作方向"
			}
			sb.WriteString(fmt.Sprintf("【%s】\n%s\n\n", settingLabel, req.Context.Setting))
		}
		if len(req.Context.PrevChapters) > 0 {
			sb.WriteString("【前文章节】\n")
			for _, ch := range req.Context.PrevChapters {
				sb.WriteString(fmt.Sprintf("- %s：%s\n", ch.Title, ch.Summary))
			}
			sb.WriteString("\n")
		}
		if len(req.Context.NextChapters) > 0 {
			sb.WriteString("【后续章节】\n")
			for _, ch := range req.Context.NextChapters {
				sb.WriteString(fmt.Sprintf("- %s：%s\n", ch.Title, ch.Summary))
			}
			sb.WriteString("\n")
		}
	}

	// 根据 action 构建具体指令
	switch req.Action {
	case "title_polish":
		sb.WriteString(fmt.Sprintf("【当前章节标题】\n%s\n\n", req.Title))
		if req.Summary != "" {
			sb.WriteString(fmt.Sprintf("【章节概要】\n%s\n\n", req.Summary))
		}
		sb.WriteString("请润色上述章节标题，使其更加精炼、有吸引力。")
	case "summary_polish":
		if req.Title != "" {
			sb.WriteString(fmt.Sprintf("【章节标题】\n%s\n\n", req.Title))
		}
		sb.WriteString(fmt.Sprintf("【当前章节概要】\n%s\n\n", req.Summary))
		sb.WriteString("请润色上述章节概要，使其更加清晰、连贯、有吸引力。")
	case "summary_expand":
		if req.Title != "" {
			sb.WriteString(fmt.Sprintf("【章节标题】\n%s\n\n", req.Title))
		}
		sb.WriteString(fmt.Sprintf("【当前章节概要】\n%s\n\n", req.Summary))
		sb.WriteString("请扩写上述章节概要，丰富情节细节、人物动机和场景描写。")
	case "generate_characters":
		// generate_characters 使用 context 中的 setting 作为世界观，Title 作为背景信息，Summary 作为剧情思路
		if req.Title != "" {
			sb.WriteString(fmt.Sprintf("【背景信息】\n%s\n\n", req.Title))
		}
		if req.Summary != "" {
			sb.WriteString(fmt.Sprintf("【剧情思路】\n%s\n\n", req.Summary))
		}
		sb.WriteString("请根据以上世界观/设定、背景信息和剧情思路，设计3-6个主要核心人物。每个人物包含：姓名、身份/职业、外在特征（标志性外貌或习惯动作）、性格特点（表层+深层）、语言特征（语气节奏、用词习惯、口头禅）、人物关系、角色定位。")
	case "generate_topic":
		// 使用 context.setting 作为用户给出的创作方向/偏好提示
		sb.WriteString("请根据以上创作方向和偏好，生成一个完整的选题方案，包含：小说标题、题材类型、核心卖点、目标读者画像。")
	case "generate_storyline":
		// 使用 context.setting + title（选题结果）作为输入
		if req.Title != "" {
			sb.WriteString(fmt.Sprintf("【选题方案】\n%s\n\n", req.Title))
		}
		sb.WriteString("请根据以上选题方案和创作方向，生成一个完整的故事线方案，包含：世界观设定、背景信息、剧情大纲。")
	case "generate_characters_ensemble":
		// 使用 context.setting + title（故事线结果）作为输入
		if req.Title != "" {
			sb.WriteString(fmt.Sprintf("【故事线方案】\n%s\n\n", req.Title))
		}
		sb.WriteString("请根据以上故事线和创作方向，生成一套完整的人物群像设定。每个人物包含：姓名、身份/职业、外在特征（标志性外貌或习惯动作）、性格特点（表层+深层）、语言特征（语气节奏、用词习惯、口头禅）、人物关系、角色定位、人物弧光。")
	}

	return sb.String()
}

// buildOutlinePrompt 构建大纲生成的用户 prompt
func (s *NovelService) buildOutlinePrompt(req *GenerateOutlineRequest) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【世界观/设定】\n%s\n\n", req.Setting))
	if req.Characters != "" {
		sb.WriteString(fmt.Sprintf("【主要人物】\n%s\n\n", req.Characters))
	}
	if req.Background != "" {
		sb.WriteString(fmt.Sprintf("【背景信息】\n%s\n\n", req.Background))
	}
	sb.WriteString(fmt.Sprintf("【剧情思路】\n%s\n\n", req.Plot))
	sb.WriteString(fmt.Sprintf("必须生成 %d 个章节的大纲（允许±3章浮动，严禁少于 %d 章），每个章节包含标题和 100-200 字的概要。", req.ChapterNum, int(float64(req.ChapterNum)*0.8)))
	return sb.String()
}

// ========== 扩写章节目录 ==========

// ExpandChapters 扩写章节目录：基于现有章节上下文生成新章节标题+概要
func (s *NovelService) ExpandChapters(ctx context.Context, userID uint, req *ExpandChaptersRequest) (uint, error) {
	// 获取小说信息
	novel, err := s.novelDAO.GetNovel(req.NovelID)
	if err != nil {
		return 0, fmt.Errorf("获取小说失败: %w", err)
	}

	// 获取所有章节（按 sort_order 排序）
	chapters, err := s.novelDAO.ListChaptersByNovel(req.NovelID)
	if err != nil {
		return 0, fmt.Errorf("获取章节列表失败: %w", err)
	}
	if len(chapters) == 0 {
		return 0, errors.New("小说暂无章节，请先创建章节")
	}

	modelName := req.ModelName
	if s.modelRegistry != nil {
		modelName = s.modelRegistry.ResolveModel(req.ModelName, model.CapTextGen)
	} else if modelName == "" {
		modelName = "zhipu"
	}

	// 构建 prompt
	systemPrompt, userPrompt := s.buildExpandPrompts(novel, chapters, req)

	// 将 system prompt 通过 History JSON 传给 executor（复用 OutlineTaskExecutor）
	historyData := map[string]interface{}{
		"system_prompt": systemPrompt,
	}
	historyJSON, _ := json.Marshal(historyData)

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: novel.PortfolioID,
		NovelID:     novel.ID,
		TaskType:    model.TaskTypeOutlineGenerate, // 复用大纲生成 executor
		ModelName:   modelName,
		Prompt:      userPrompt,
		History:     string(historyJSON),
		Status:      model.TaskStatusPending,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch expand chapters task failed: %w", err)
	}

	return task.ID, nil
}

// buildExpandPrompts 构建扩写章节的 system prompt 和 user prompt
func (s *NovelService) buildExpandPrompts(novel *model.Novel, chapters []model.Chapter, req *ExpandChaptersRequest) (string, string) {
	var sysSB, userSB strings.Builder

	if req.Mode == "append" {
		// 末尾追加模式
		sysSB.WriteString("你是一位专业的小说策划师。基于用户提供的现有章节大纲，续写新的章节。\n")
		sysSB.WriteString("新章节需要延续现有故事脉络，保持风格一致，情节自然衔接。\n")
		sysSB.WriteString("你必须严格按照以下 JSON 格式输出，不要包含任何其他文字：\n")
		sysSB.WriteString("[{\"title\": \"风起云涌\", \"summary\": \"100-200字的章节概要...\"}]")
		sysSB.WriteString("\n注意：title 只写纯标题，不要带\"第X章\"等章节序号前缀，系统会自动编号。")

		userSB.WriteString(fmt.Sprintf("【小说简介】\n%s\n\n", novel.Description))
		userSB.WriteString("【现有章节大纲】\n")
		for _, ch := range chapters {
			summary := ch.Summary
			if summary == "" {
				summary = "（暂无概要）"
			}
			userSB.WriteString(fmt.Sprintf("第%d章「%s」：%s\n", ch.SortOrder, ch.Title, summary))
		}
		userSB.WriteString(fmt.Sprintf("\n请在现有 %d 章之后，续写 %d 个新章节。每个章节包含标题和 100-200 字的概要。", len(chapters), req.ChapterNum))
	} else {
		// 中间插入模式
		sysSB.WriteString("你是一位专业的小说策划师。基于用户提供的前后章节上下文，在指定位置插入新的过渡章节。\n")
		sysSB.WriteString("新章节需要自然衔接前后情节，丰富故事细节。\n")
		sysSB.WriteString("你必须严格按照以下 JSON 格式输出，不要包含任何其他文字：\n")
		sysSB.WriteString("[{\"title\": \"暗流涌动\", \"summary\": \"100-200字的章节概要...\"}]")
		sysSB.WriteString("\n注意：title 只写纯标题，不要带\"第X章\"等章节序号前缀，系统会自动编号。")

		userSB.WriteString(fmt.Sprintf("【小说简介】\n%s\n\n", novel.Description))

		// 分割前后章节
		userSB.WriteString("【前面的章节】\n")
		for _, ch := range chapters {
			if ch.SortOrder <= req.InsertAfter {
				summary := ch.Summary
				if summary == "" {
					summary = "（暂无概要）"
				}
				userSB.WriteString(fmt.Sprintf("第%d章「%s」：%s\n", ch.SortOrder, ch.Title, summary))
			}
		}
		userSB.WriteString("\n【后面的章节】\n")
		for _, ch := range chapters {
			if ch.SortOrder > req.InsertAfter {
				summary := ch.Summary
				if summary == "" {
					summary = "（暂无概要）"
				}
				userSB.WriteString(fmt.Sprintf("第%d章「%s」：%s\n", ch.SortOrder, ch.Title, summary))
			}
		}
		userSB.WriteString(fmt.Sprintf("\n请在第%d章和第%d章之间，插入 %d 个过渡章节。每个章节包含标题和 100-200 字的概要。", req.InsertAfter, req.InsertAfter+1, req.ChapterNum))
	}

	// 追加用户自定义指令
	if req.UserPrompt != "" {
		userSB.WriteString(fmt.Sprintf("\n\n【用户补充指令】\n%s", req.UserPrompt))
	}

	// 注入写作风格到 system prompt
	if s.writingStyleSvc != nil {
		styleText := s.writingStyleSvc.FormatStyleForPrompt(req.NovelID, nil)
		if styleText != "" {
			sysSB.WriteString("\n\n【写作规范】\n")
			sysSB.WriteString(styleText)
		}
	}

	return sysSB.String(), userSB.String()
}

// RejectAIResult 拒绝 AI 生成结果（仅记录行为事件）
func (s *NovelService) RejectAIResult(ctx context.Context, userID, chapterID, taskID uint) error {
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return err
	}

	task, err := s.aiTaskDAO.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// 记录 ai_reject 行为事件
	if s.behaviorSvc != nil {
		go s.behaviorSvc.RecordEvent(userID, chapter.NovelID, chapterID, model.BehaviorAIReject, map[string]interface{}{
			"action":  task.TaskType,
			"model":   task.ModelName,
			"task_id": taskID,
		})
	}

	return nil
}

// computeDiffStats 计算简单的行级 diff 统计
func computeDiffStats(oldText, newText string) map[string]int {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	oldSet := make(map[string]int)
	for _, line := range oldLines {
		if strings.TrimSpace(line) != "" {
			oldSet[line]++
		}
	}

	newSet := make(map[string]int)
	for _, line := range newLines {
		if strings.TrimSpace(line) != "" {
			newSet[line]++
		}
	}

	added, removed := 0, 0
	for line, count := range newSet {
		if oldCount, ok := oldSet[line]; ok {
			if count > oldCount {
				added += count - oldCount
			}
		} else {
			added += count
		}
	}
	for line, count := range oldSet {
		if newCount, ok := newSet[line]; ok {
			if count > newCount {
				removed += count - newCount
			}
		} else {
			removed += count
		}
	}

	return map[string]int{
		"added":   added,
		"removed": removed,
		"total":   added + removed,
	}
}

// TriggerFactColdStart 触发全量冷启动事实采集
func (s *NovelService) TriggerFactColdStart(ctx context.Context, novelID, userID uint) error {
	return s.factSvc.FullColdStart(ctx, novelID, userID)
}

// RepairButlerNovelLinks 修复历史管家任务：为孤立的管家任务创建临时小说并关联
// 数据提取优先级：
//   1. outline_generate 的 prompt 字段（包含用户最终确认的选题+人物+故事线）
//   2. 步骤1-3 任务的 result.content（每步有4个并行方案，取最后一个作为兜底）
func (s *NovelService) RepairButlerNovelLinks(ctx context.Context, portfolioID uint) (int, error) {
	// ===== 第一步：清理旧修复数据 =====
	oldNovels, err := s.novelDAO.ListRepairedNovels(portfolioID)
	if err != nil {
		log.Printf("[RepairButlerNovelLinks] 查询旧修复小说失败: %v", err)
	} else if len(oldNovels) > 0 {
		// 收集旧小说 ID，重置关联任务的 novel_id
		oldIDs := make([]uint, len(oldNovels))
		for i, n := range oldNovels {
			oldIDs[i] = n.ID
		}
		if err := s.aiTaskDAO.ResetNovelIDByNovelIDs(ctx, oldIDs); err != nil {
			log.Printf("[RepairButlerNovelLinks] 重置旧任务 novel_id 失败: %v", err)
		}
		// 删除旧修复小说及其章节
		for _, n := range oldNovels {
			if err := s.novelDAO.DeleteNovelWithChapters(n.ID); err != nil {
				log.Printf("[RepairButlerNovelLinks] 删除旧修复小说 %d 失败: %v", n.ID, err)
			}
		}
		log.Printf("[RepairButlerNovelLinks] 已清理 %d 个旧修复小说", len(oldNovels))
	}

	// ===== 第二步：重新修复 =====
	butlerTaskTypes := []string{
		model.TaskTypeButlerGenerateTopic,
		model.TaskTypeButlerGenerateStoryline,
		model.TaskTypeButlerGenerateCharacters,
		model.TaskTypeOutlineGenerate,
		model.TaskTypeButlerStorylineDraft,
		model.TaskTypeButlerStorylineReview,
		model.TaskTypeButlerCharactersDraft,
		model.TaskTypeButlerCharactersReview,
	}

	// 查出所有孤立的管家任务
	tasks, err := s.aiTaskDAO.ListOrphanButlerTasks(ctx, portfolioID, butlerTaskTypes)
	if err != nil {
		return 0, fmt.Errorf("list orphan butler tasks failed: %w", err)
	}
	log.Printf("[RepairButlerNovelLinks] 查到 %d 个孤立管家任务", len(tasks))
	if len(tasks) == 0 {
		return 0, nil
	}

	// 按 10 分钟时间窗口聚类：相邻任务间隔 > 10 分钟则视为不同批次
	type taskGroup struct {
		tasks []*model.AITask
	}
	var groups []taskGroup
	var current taskGroup
	for _, task := range tasks {
		if len(current.tasks) > 0 {
			last := current.tasks[len(current.tasks)-1]
			if task.CreatedAt.Sub(last.CreatedAt) > 10*time.Minute {
				groups = append(groups, current)
				current = taskGroup{}
			}
		}
		current.tasks = append(current.tasks, task)
	}
	if len(current.tasks) > 0 {
		groups = append(groups, current)
	}

	log.Printf("[RepairButlerNovelLinks] 聚类为 %d 个任务组", len(groups))

	repaired := 0
	for gi, group := range groups {
		log.Printf("[RepairButlerNovelLinks] 处理第 %d 组，包含 %d 个任务", gi+1, len(group.tasks))
		// 检查同一时间窗口内是否已有关联真实小说的管家任务，有则跳过
		first := group.tasks[0]
		last := group.tasks[len(group.tasks)-1]
		windowStart := first.CreatedAt.Add(-1 * time.Minute)
		windowEnd := last.CreatedAt.Add(10 * time.Minute)
		hasLinked, _ := s.aiTaskDAO.HasLinkedButlerTasks(ctx, portfolioID, butlerTaskTypes, windowStart, windowEnd)
		if hasLinked {
			log.Printf("[RepairButlerNovelLinks] 跳过已关联小说的任务组 (时间: %v ~ %v)", first.CreatedAt, last.CreatedAt)
			continue
		}

		topicResult, storylineResult, charactersResult, outlineChapters := s.extractButlerGroupData(group.tasks)

		// 生成小说标题
		title := "管家创作（恢复）"
		if topicResult != "" {
			if t := extractTitle(topicResult); t != "" {
				title = t
			}
		}

		// 创建临时小说
		novel := &model.Novel{
			PortfolioID:      portfolioID,
			Title:            title,
			Description:      "由修复工具从历史管家任务中恢复",
			Status:           model.NovelStatusDraft,
			Source:           model.NovelSourceButler,
			ButlerTopic:      topicResult,
			ButlerStoryline:  storylineResult,
			ButlerCharacters: charactersResult,
		}
		if err := s.novelDAO.CreateNovel(novel); err != nil {
			log.Printf("[RepairButlerNovelLinks] 创建临时小说失败: %v", err)
			continue
		}

		// 如果有大纲章节，批量创建
		if len(outlineChapters) > 0 {
			chapters := make([]model.Chapter, len(outlineChapters))
			for i, ch := range outlineChapters {
				chapters[i] = model.Chapter{
					NovelID:        novel.ID,
					Title:          ch.Title,
					Summary:        ch.Summary,
					SortOrder:      i + 1,
					Status:         model.ChapterStatusDraft,
					CurrentVersion: 1,
				}
			}
			if err := s.novelDAO.BatchCreateChapters(chapters); err != nil {
				log.Printf("[RepairButlerNovelLinks] 批量创建章节失败: novel_id=%d err=%v", novel.ID, err)
			}
			s.refreshNovelStats(novel.ID)
		}

		// 回填所有任务的 novel_id
		taskIDs := make([]uint, len(group.tasks))
		for i, t := range group.tasks {
			taskIDs[i] = t.ID
		}
		if err := s.aiTaskDAO.BatchUpdateNovelID(ctx, taskIDs, novel.ID); err != nil {
			log.Printf("[RepairButlerNovelLinks] 批量回填 novel_id 失败: novel_id=%d err=%v", novel.ID, err)
		}

		repaired += len(group.tasks)
	}

	return repaired, nil
}

// extractButlerGroupData 从一组管家任务中提取选题、故事线、人物、大纲章节
// 优先从 outline_generate 的 prompt 中解析（包含用户最终确认的版本），
// 兜底从步骤1-3 的 result.content 中提取
func (s *NovelService) extractButlerGroupData(tasks []*model.AITask) (topic, storyline, characters string, chapters []struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}) {
	log.Printf("[extractButlerGroupData] 开始处理 %d 个任务", len(tasks))
	for _, task := range tasks {
		log.Printf("[extractButlerGroupData] 任务 id=%d type=%s novel_id=%d prompt_len=%d result_len=%d history_len=%d",
			task.ID, task.TaskType, task.NovelID, len(task.Prompt), len(task.Result), len(task.History))
	}

	// 第一优先级：从 outline_generate 的 prompt 中解析
	for _, task := range tasks {
		if task.TaskType != model.TaskTypeOutlineGenerate {
			continue
		}

		prompt := task.Prompt
		if setting := extractSection(prompt, "【世界观/设定】"); setting != "" {
			topic = setting
			log.Printf("[extractButlerGroupData] 从 outline_generate prompt 提取 topic, len=%d", len(topic))
		}
		if chars := extractSection(prompt, "【主要人物】"); chars != "" {
			characters = chars
			log.Printf("[extractButlerGroupData] 从 outline_generate prompt 提取 characters, len=%d", len(characters))
		}
		if plot := extractSection(prompt, "【剧情思路】"); plot != "" {
			storyline = plot
			log.Printf("[extractButlerGroupData] 从 outline_generate prompt 提取 storyline, len=%d", len(storyline))
		}

		// 从 history 的 setting 字段提取原始创作方向（作为 topic 的补充）
		if topic == "" && task.History != "" {
			var historyMap map[string]interface{}
			if json.Unmarshal([]byte(task.History), &historyMap) == nil {
				if setting, ok := historyMap["setting"].(string); ok && setting != "" {
					topic = setting
					log.Printf("[extractButlerGroupData] 从 outline_generate history.setting 提取 topic, len=%d", len(topic))
				}
			}
		}

		// 从 result 中提取章节
		if task.Result != "" {
			var resultMap map[string]interface{}
			if json.Unmarshal([]byte(task.Result), &resultMap) == nil {
				if chaptersRaw, ok := resultMap["chapters"]; ok {
					if chapBytes, err := json.Marshal(chaptersRaw); err == nil {
						json.Unmarshal(chapBytes, &chapters)
						log.Printf("[extractButlerGroupData] 从 outline_generate result 提取 chapters, count=%d", len(chapters))
					}
				}
			}
		}

		break
	}

	// 第二优先级：从步骤1-3 的 result.content 兜底
	for _, task := range tasks {
		if task.Result == "" {
			continue
		}
		var content, chars string
		var resultMap map[string]interface{}
		if json.Unmarshal([]byte(task.Result), &resultMap) == nil {
			content, _ = resultMap["content"].(string)
			chars, _ = resultMap["characters"].(string)
		} else {
			// result 不是 JSON，直接用原文
			content = task.Result
		}

		switch task.TaskType {
		case model.TaskTypeButlerGenerateTopic:
			if topic == "" && content != "" {
				topic = content
				log.Printf("[extractButlerGroupData] 从 butler_generate_topic result.content 提取 topic, len=%d", len(topic))
			}
		case model.TaskTypeButlerGenerateStoryline, model.TaskTypeButlerStorylineDraft, model.TaskTypeButlerStorylineReview:
			if storyline == "" && content != "" {
				storyline = content
				log.Printf("[extractButlerGroupData] 从 %s result.content 提取 storyline, len=%d", task.TaskType, len(storyline))
			}
		case model.TaskTypeButlerGenerateCharacters, model.TaskTypeButlerCharactersDraft, model.TaskTypeButlerCharactersReview:
			if characters == "" {
				if chars != "" {
					characters = chars
				} else if content != "" {
					characters = content
				}
				if characters != "" {
					log.Printf("[extractButlerGroupData] 从 %s result 提取 characters, len=%d", task.TaskType, len(characters))
				}
			}
		}
	}

	// 第三优先级：从步骤1-3 的 history.setting 中提取（管家步骤的 context.setting 包含创作方向）
	if topic == "" {
		for _, task := range tasks {
			if task.History == "" {
				continue
			}
			var historyMap map[string]interface{}
			if json.Unmarshal([]byte(task.History), &historyMap) == nil {
				if setting, ok := historyMap["setting"].(string); ok && setting != "" {
					topic = setting
					log.Printf("[extractButlerGroupData] 从 %s history.setting 提取 topic, len=%d", task.TaskType, len(topic))
					break
				}
			}
		}
	}

	log.Printf("[extractButlerGroupData] 最终结果: topic_len=%d storyline_len=%d characters_len=%d chapters_count=%d",
		len(topic), len(storyline), len(characters), len(chapters))

	return
}

// extractTitle 从选题文本中提取标题
func extractTitle(topicResult string) string {
	// 匹配 "标题：xxx" 或 "书名：xxx" 或 "小说标题：xxx" 格式
	for _, line := range strings.Split(topicResult, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"小说标题：", "小说标题:", "标题：", "标题:", "书名：", "书名:"} {
			if strings.HasPrefix(line, prefix) {
				t := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				t = strings.Trim(t, "《》「」")
				if t != "" && len([]rune(t)) <= 30 {
					return t
				}
			}
		}
	}
	return ""
}

// extractSection 从 prompt 文本中提取指定标记之间的内容
// 例如 extractSection(text, "【世界观/设定】") 提取该标记到下一个 【xxx】 之间的文本
func extractSection(text, marker string) string {
	idx := strings.Index(text, marker)
	if idx < 0 {
		return ""
	}
	content := text[idx+len(marker):]

	// 查找下一个 【xxx】 标记作为结束位置
	nextMarkers := []string{"【世界观/设定】", "【主要人物】", "【背景信息】", "【剧情思路】", "【用户额外指令】"}
	endIdx := len(content)
	for _, m := range nextMarkers {
		if m == marker {
			continue
		}
		if i := strings.Index(content, m); i >= 0 && i < endIdx {
			endIdx = i
		}
	}

	// 也检查 "请生成" 开头的指令行作为结束标记
	if i := strings.Index(content, "\n请生成"); i >= 0 && i < endIdx {
		endIdx = i
	}

	return strings.TrimSpace(content[:endIdx])
}