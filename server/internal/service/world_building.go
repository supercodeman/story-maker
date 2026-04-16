// server/internal/service/world_building.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// WorldBuildingService 世界构建服务，负责多阶段反思循环（生成→审查→优化）
type WorldBuildingService struct {
	dao          *dao.WorldBuildingDAO
	aiTaskDAO    *dao.AITaskDAO
	factDAO      *dao.NovelFactDAO
	knowledgeDAO *dao.KnowledgeDAO
	overviewDAO  *dao.OverviewDAO
	dispatcher   *agent.Dispatcher
	tplSvc       *PromptTemplateService
}

// NewWorldBuildingService 创建 WorldBuildingService 实例
func NewWorldBuildingService(aiTaskDAO *dao.AITaskDAO, dispatcher *agent.Dispatcher, tplSvc *PromptTemplateService) *WorldBuildingService {
	return &WorldBuildingService{
		dao:          dao.NewWorldBuildingDAO(),
		aiTaskDAO:    aiTaskDAO,
		factDAO:      dao.NewNovelFactDAO(),
		knowledgeDAO: dao.NewKnowledgeDAO(),
		overviewDAO:  dao.NewOverviewDAO(),
		dispatcher:   dispatcher,
		tplSvc:       tplSvc,
	}
}

// ========== 请求/响应定义 ==========

// WorldBuildingStartRequest 启动世界构建阶段请求
type WorldBuildingStartRequest struct {
	NovelID     uint                   `json:"novel_id" binding:"required"`
	PortfolioID uint                   `json:"portfolio_id" binding:"required"`
	Phase       string                 `json:"phase" binding:"required"`
	UserInput   string                 `json:"user_input"`
	Config      *model.ReflectionConfig `json:"config"`
}

// WorldBuildingStatusResponse 阶段状态响应
type WorldBuildingStatusResponse struct {
	Phase        string                `json:"phase"`
	Round        int                   `json:"round"`
	Status       string                `json:"status"` // pending / generating / reviewing / done
	Content      string                `json:"content"`
	ReviewResult *model.ReviewResult   `json:"review_result,omitempty"`
	Config       model.ReflectionConfig `json:"config"`
}

// PhaseResultResponse 阶段处理结果响应
type PhaseResultResponse struct {
	Done         bool                `json:"done"`
	Round        int                 `json:"round"`
	Content      string              `json:"content,omitempty"`
	Score        float64             `json:"score,omitempty"`
	ReviewResult *model.ReviewResult `json:"review_result,omitempty"`
}

// WorldBuildingSummary 世界构建概览
type WorldBuildingSummary struct {
	WorldSettings []model.NovelWorldSetting `json:"world_settings"`
	Foreshadows   []model.NovelForeshadow   `json:"foreshadows"`
	PlotOutlines  []model.NovelPlotOutline   `json:"plot_outlines"`
}

// 合法的反思阶段白名单
var validPhases = map[string]bool{
	model.ReflectionPhaseWorldview:  true,
	model.ReflectionPhaseCharacter:  true,
	model.ReflectionPhaseRelation:   true,
	model.ReflectionPhaseForeshadow: true,
	model.ReflectionPhasePlot:       true,
}

// ========== 核心方法 ==========

// StartPhase 启动某个阶段的反思循环（生成第一轮内容）
func (s *WorldBuildingService) StartPhase(ctx context.Context, userID uint, req *WorldBuildingStartRequest) (uint, error) {
	if !validPhases[req.Phase] {
		return 0, fmt.Errorf("invalid phase: %s", req.Phase)
	}

	cfg := s.resolveConfig(req.Config)
	modelName := cfg.ModelName
	if modelName == "" {
		modelName = "zhipu"
	}

	// 构建 Writer Prompt
	systemPrompt := s.buildWriterSystemPrompt(req.Phase)
	userPrompt := s.buildWriterUserPrompt(ctx, req.Phase, req.NovelID, req.UserInput, "")

	historyData := map[string]interface{}{"system_prompt": systemPrompt}
	historyJSON, _ := json.Marshal(historyData)

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: req.PortfolioID,
		NovelID:     req.NovelID,
		TaskType:    s.phaseToGenerateTaskType(req.Phase),
		ModelName:   modelName,
		Prompt:      userPrompt,
		History:     string(historyJSON),
		Status:      model.TaskStatusPending,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch generate task failed: %w", err)
	}
	return task.ID, nil
}

// ReviewPhaseResult 审查某轮生成结果，返回审查任务 ID
func (s *WorldBuildingService) ReviewPhaseResult(ctx context.Context, userID uint, novelID, portfolioID uint, phase string, generateTaskID uint) (uint, error) {
	if !validPhases[phase] {
		return 0, fmt.Errorf("invalid phase: %s", phase)
	}

	// 获取生成任务结果
	genTask, err := s.aiTaskDAO.GetTask(ctx, generateTaskID)
	if err != nil {
		return 0, fmt.Errorf("get generate task failed: %w", err)
	}
	if genTask.Status != model.TaskStatusCompleted {
		return 0, fmt.Errorf("generate task not completed, status: %s", genTask.Status)
	}

	// 构建 Editor Prompt
	systemPrompt := s.buildEditorSystemPrompt()
	userPrompt := s.buildEditorUserPrompt(phase, genTask.Result)

	historyData := map[string]interface{}{"system_prompt": systemPrompt}
	historyJSON, _ := json.Marshal(historyData)

	modelName := "zhipu"
	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		NovelID:     novelID,
		TaskType:    s.phaseToReviewTaskType(phase),
		ModelName:   modelName,
		Prompt:      userPrompt,
		History:     string(historyJSON),
		Status:      model.TaskStatusPending,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch review task failed: %w", err)
	}
	return task.ID, nil
}

// ProcessReviewResult 处理审查结果，决定是否继续优化
func (s *WorldBuildingService) ProcessReviewResult(ctx context.Context, userID uint, novelID uint, phase string, generateTaskID, reviewTaskID uint, cfg *model.ReflectionConfig) (*PhaseResultResponse, error) {
	if !validPhases[phase] {
		return nil, fmt.Errorf("invalid phase: %s", phase)
	}
	config := s.resolveConfig(cfg)

	// 获取审查任务结果
	reviewTask, err := s.aiTaskDAO.GetTask(ctx, reviewTaskID)
	if err != nil {
		return nil, fmt.Errorf("get review task failed: %w", err)
	}
	if reviewTask.Status != model.TaskStatusCompleted {
		return nil, fmt.Errorf("review task not completed, status: %s", reviewTask.Status)
	}

	// 获取生成任务结果（用于记录内容快照）
	genTask, err := s.aiTaskDAO.GetTask(ctx, generateTaskID)
	if err != nil {
		return nil, fmt.Errorf("get generate task failed: %w", err)
	}

	// 解析审查结果 JSON（容错：LLM 可能返回 markdown code block 包裹的 JSON）
	var reviewResult model.ReviewResult
	cleaned := extractJSONObject(reviewTask.Result)
	if err := json.Unmarshal([]byte(cleaned), &reviewResult); err != nil {
		log.Printf("[world-building] parse review result failed: %v, raw: %s", err, reviewTask.Result)
		// 降级处理：给默认低分，让流程继续（触发优化或达到最大轮次后结束）
		reviewResult = model.ReviewResult{
			TotalScore: 3.0,
			Summary:    "审查结果解析失败，使用默认低分",
			Suggestion: "请检查生成内容格式",
		}
	}

	// 查询当前轮次
	logs, _ := s.dao.ListReflectionLogs(novelID, phase)
	round := len(logs) + 1

	// 记录 ReflectionLog
	reviewJSON, _ := json.Marshal(reviewResult)
	reflectionLog := &model.ReflectionLog{
		NovelID:    novelID,
		UserID:     userID,
		Phase:      phase,
		Round:      round,
		Content:    genTask.Result,
		ReviewJSON: string(reviewJSON),
		TotalScore: reviewResult.TotalScore,
		TaskID:     generateTaskID,
		ReviewTask: reviewTaskID,
	}
	if err := s.dao.CreateReflectionLog(reflectionLog); err != nil {
		log.Printf("[world-building] create reflection log failed: %v", err)
	}

	// 判断是否达标或达到最大轮次
	if reviewResult.TotalScore >= config.Threshold || round >= config.MaxRounds {
		return &PhaseResultResponse{
			Done:    true,
			Round:   round,
			Content: genTask.Result,
			Score:   reviewResult.TotalScore,
		}, nil
	}

	return &PhaseResultResponse{
		Done:         false,
		Round:        round,
		ReviewResult: &reviewResult,
	}, nil
}

// OptimizePhase 基于审查意见优化，返回新的生成任务 ID
func (s *WorldBuildingService) OptimizePhase(ctx context.Context, userID uint, novelID, portfolioID uint, phase string, prevContent string, reviewResult *model.ReviewResult, cfg *model.ReflectionConfig) (uint, error) {
	if !validPhases[phase] {
		return 0, fmt.Errorf("invalid phase: %s", phase)
	}
	config := s.resolveConfig(cfg)
	modelName := config.ModelName
	if modelName == "" {
		modelName = "zhipu"
	}

	// 构建优化 Prompt
	systemPrompt := s.buildWriterSystemPrompt(phase)
	userPrompt := s.buildOptimizeUserPrompt(phase, prevContent, reviewResult)

	historyData := map[string]interface{}{"system_prompt": systemPrompt}
	historyJSON, _ := json.Marshal(historyData)

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		NovelID:     novelID,
		TaskType:    s.phaseToGenerateTaskType(phase),
		ModelName:   modelName,
		Prompt:      userPrompt,
		History:     string(historyJSON),
		Status:      model.TaskStatusPending,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch optimize task failed: %w", err)
	}
	return task.ID, nil
}

// AcceptPhaseResult 接受某阶段结果，写入数据库
func (s *WorldBuildingService) AcceptPhaseResult(ctx context.Context, userID, novelID uint, phase, content string, score float64) error {
	switch phase {
	case model.ReflectionPhaseWorldview:
		return s.acceptWorldview(userID, novelID, content, score)
	case model.ReflectionPhaseForeshadow:
		return s.acceptForeshadow(userID, novelID, content, score)
	case model.ReflectionPhasePlot:
		return s.acceptPlotOutline(userID, novelID, content, score)
	case model.ReflectionPhaseCharacter:
		return s.acceptCharacter(userID, novelID, content, score)
	case model.ReflectionPhaseRelation:
		return s.acceptRelation(userID, novelID, content, score)
	default:
		return fmt.Errorf("unsupported phase for accept: %s", phase)
	}
}

// GetPhaseStatus 获取某阶段的当前状态
func (s *WorldBuildingService) GetPhaseStatus(novelID uint, phase string) (*WorldBuildingStatusResponse, error) {
	if !validPhases[phase] {
		return nil, fmt.Errorf("invalid phase: %s", phase)
	}
	logs, err := s.dao.ListReflectionLogs(novelID, phase)
	if err != nil {
		return nil, fmt.Errorf("list reflection logs failed: %w", err)
	}

	resp := &WorldBuildingStatusResponse{
		Phase:  phase,
		Config: model.DefaultReflectionConfig(),
	}

	if len(logs) == 0 {
		resp.Status = "pending"
		return resp, nil
	}

	// 取最新一轮
	latest := logs[len(logs)-1]
	resp.Round = latest.Round
	resp.Content = latest.Content

	var reviewResult model.ReviewResult
	if latest.ReviewJSON != "" {
		if err := json.Unmarshal([]byte(latest.ReviewJSON), &reviewResult); err == nil {
			resp.ReviewResult = &reviewResult
		}
	}

	if latest.TotalScore >= resp.Config.Threshold || latest.Round >= resp.Config.MaxRounds {
		resp.Status = "done"
	} else {
		resp.Status = "reviewing"
	}
	return resp, nil
}

// GetWorldBuildingSummary 获取小说的完整世界构建概览
func (s *WorldBuildingService) GetWorldBuildingSummary(novelID uint) (*WorldBuildingSummary, error) {
	settings, err := s.dao.ListWorldSettingsByNovel(novelID)
	if err != nil {
		return nil, fmt.Errorf("list world settings failed: %w", err)
	}
	foreshadows, err := s.dao.ListForeshadowsByNovel(novelID)
	if err != nil {
		return nil, fmt.Errorf("list foreshadows failed: %w", err)
	}
	plots, err := s.dao.ListPlotOutlinesByNovel(novelID)
	if err != nil {
		return nil, fmt.Errorf("list plot outlines failed: %w", err)
	}
	return &WorldBuildingSummary{
		WorldSettings: settings,
		Foreshadows:   foreshadows,
		PlotOutlines:  plots,
	}, nil
}

// ========== 辅助方法（私有） ==========

// resolveConfig 合并用户配置与默认配置
func (s *WorldBuildingService) resolveConfig(cfg *model.ReflectionConfig) model.ReflectionConfig {
	defaults := model.DefaultReflectionConfig()
	if cfg == nil {
		return defaults
	}
	if cfg.MaxRounds > 0 {
		defaults.MaxRounds = cfg.MaxRounds
	}
	if cfg.Threshold > 0 {
		defaults.Threshold = cfg.Threshold
	}
	if cfg.ModelName != "" {
		defaults.ModelName = cfg.ModelName
	}
	defaults.AutoMode = cfg.AutoMode
	return defaults
}

// phaseToGenerateTaskType 阶段映射到生成任务类型
func (s *WorldBuildingService) phaseToGenerateTaskType(phase string) string {
	m := map[string]string{
		model.ReflectionPhaseWorldview:  model.TaskTypeWorldviewGenerate,
		model.ReflectionPhaseCharacter:  model.TaskTypeCharacterGenerate2,
		model.ReflectionPhaseRelation:   model.TaskTypeRelationGenerate,
		model.ReflectionPhaseForeshadow: model.TaskTypeForeshadowGenerate,
		model.ReflectionPhasePlot:       model.TaskTypePlotGenerate,
	}
	return m[phase]
}

// phaseToReviewTaskType 阶段映射到审查任务类型
func (s *WorldBuildingService) phaseToReviewTaskType(phase string) string {
	m := map[string]string{
		model.ReflectionPhaseWorldview:  model.TaskTypeWorldviewReview,
		model.ReflectionPhaseCharacter:  model.TaskTypeCharacterReview,
		model.ReflectionPhaseRelation:   model.TaskTypeRelationReview,
		model.ReflectionPhaseForeshadow: model.TaskTypeForeshadowReview,
		model.ReflectionPhasePlot:       model.TaskTypePlotReview,
	}
	return m[phase]
}
