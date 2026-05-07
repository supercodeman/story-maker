// server/internal/service/novel_overview.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// OverviewService 总览业务逻辑层
type OverviewService struct {
	overviewDAO  *dao.OverviewDAO
	knowledgeDAO *dao.KnowledgeDAO
	novelDAO     *dao.NovelDAO
	aiTaskDAO    *dao.AITaskDAO
	dispatcher   *agent.Dispatcher
	workflowSvc  *WorkflowService
}

// NewOverviewService 创建 OverviewService 实例
func NewOverviewService(
	aiTaskDAO *dao.AITaskDAO,
	dispatcher *agent.Dispatcher,
	workflowSvc *WorkflowService,
) *OverviewService {
	return &OverviewService{
		overviewDAO:  dao.NewOverviewDAO(),
		knowledgeDAO: dao.NewKnowledgeDAO(),
		novelDAO:     dao.NewNovelDAO(),
		aiTaskDAO:    aiTaskDAO,
		dispatcher:   dispatcher,
		workflowSvc:  workflowSvc,
	}
}

// ========== 请求/响应定义 ==========

// OverviewResponse 总览聚合响应
type OverviewResponse struct {
	Plotlines   []model.NovelKnowledge         `json:"plotlines"`
	Characters  []model.NovelKnowledge         `json:"characters"`
	Foreshadows []model.NovelKnowledge         `json:"foreshadows"`
	Relations   []model.NovelCharacterRelation  `json:"relations"`
	Chapters    []ChapterBrief                  `json:"chapters"`
}

// ChapterBrief 章节摘要（不含正文，节省传输）
type ChapterBrief struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	SortOrder int    `json:"sort_order"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
}

// CreateRelationRequest 创建人物关系请求
type CreateRelationRequest struct {
	FromKnowledgeID uint   `json:"from_knowledge_id" binding:"required"`
	ToKnowledgeID   uint   `json:"to_knowledge_id" binding:"required"`
	RelationType    string `json:"relation_type" binding:"required"`
	Label           string `json:"label"`
	ChapterRef      string `json:"chapter_ref"`
}

// UpdateRelationRequest 更新人物关系请求（指针类型支持清空为空字符串）
type UpdateRelationRequest struct {
	RelationType *string `json:"relation_type"`
	Label        *string `json:"label"`
	ChapterRef   *string `json:"chapter_ref"`
}

// ExtractOverviewRequest AI 提取总览请求
type ExtractOverviewRequest struct {
	ModelName string `json:"model_name"`
}

// OverviewChange 前端变更条目
type OverviewChange struct {
	Type       string      `json:"type"`        // plotline/character/foreshadow/relation
	Action     string      `json:"action"`      // create/update/delete
	ID         uint        `json:"id"`          // 被修改的条目 ID（update/delete 时）
	Data       interface{} `json:"data"`        // 变更后的数据
	OldData    interface{} `json:"old_data"`    // 变更前的数据（用于 Diff）
}

// SubmitRevisionRequest 提交变更请求
type SubmitRevisionRequest struct {
	ModelName string           `json:"model_name"`
	Changes   []OverviewChange `json:"changes" binding:"required"`
}

// ExecuteRevisionRequest 确认执行变更请求
type ExecuteRevisionRequest struct {
	ModelName    string `json:"model_name"`
	WorkflowID   uint   `json:"workflow_id" binding:"required"`
	RevisionPlan string `json:"revision_plan" binding:"required"`
}

// ========== 总览查询 ==========

// GetOverview 聚合返回所有元数据
func (s *OverviewService) GetOverview(novelID uint) (*OverviewResponse, error) {
	// 获取元数据
	data, err := s.overviewDAO.GetOverviewData(novelID)
	if err != nil {
		return nil, fmt.Errorf("get overview data failed: %w", err)
	}

	// 获取章节摘要列表
	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return nil, fmt.Errorf("list chapters failed: %w", err)
	}

	briefs := make([]ChapterBrief, 0, len(chapters))
	for _, ch := range chapters {
		briefs = append(briefs, ChapterBrief{
			ID:        ch.ID,
			Title:     ch.Title,
			SortOrder: ch.SortOrder,
			Summary:   ch.Summary,
			Status:    ch.Status,
		})
	}

	return &OverviewResponse{
		Plotlines:   data.Plotlines,
		Characters:  data.Characters,
		Foreshadows: data.Foreshadows,
		Relations:   data.Relations,
		Chapters:    briefs,
	}, nil
}

// ========== 人物关系 CRUD ==========

// CreateRelation 创建人物关系
func (s *OverviewService) CreateRelation(novelID uint, req *CreateRelationRequest) (*model.NovelCharacterRelation, error) {
	if !model.ValidRelationTypes[req.RelationType] {
		return nil, fmt.Errorf("invalid relation type: %s", req.RelationType)
	}

	// 校验 FromKnowledgeID 归属当前小说且为 character 类别
	fromK, err := s.knowledgeDAO.Get(req.FromKnowledgeID)
	if err != nil {
		return nil, fmt.Errorf("from_knowledge_id not found: %w", err)
	}
	if fromK.NovelID != novelID || fromK.Category != model.KnowledgeCategoryCharacter {
		return nil, fmt.Errorf("from_knowledge_id does not belong to this novel or is not a character")
	}

	// 校验 ToKnowledgeID 归属当前小说且为 character 类别
	toK, err := s.knowledgeDAO.Get(req.ToKnowledgeID)
	if err != nil {
		return nil, fmt.Errorf("to_knowledge_id not found: %w", err)
	}
	if toK.NovelID != novelID || toK.Category != model.KnowledgeCategoryCharacter {
		return nil, fmt.Errorf("to_knowledge_id does not belong to this novel or is not a character")
	}

	r := &model.NovelCharacterRelation{
		NovelID:         novelID,
		FromKnowledgeID: req.FromKnowledgeID,
		ToKnowledgeID:   req.ToKnowledgeID,
		RelationType:    req.RelationType,
		Label:           req.Label,
		ChapterRef:      req.ChapterRef,
	}

	if err := s.overviewDAO.CreateRelation(r); err != nil {
		return nil, fmt.Errorf("create relation failed: %w", err)
	}
	return r, nil
}

// UpdateRelation 更新人物关系
func (s *OverviewService) UpdateRelation(novelID uint, relationID uint, req *UpdateRelationRequest) (*model.NovelCharacterRelation, error) {
	r, err := s.overviewDAO.GetRelation(relationID)
	if err != nil {
		return nil, fmt.Errorf("relation not found: %w", err)
	}
	if r.NovelID != novelID {
		return nil, fmt.Errorf("relation does not belong to this novel")
	}

	if req.RelationType != nil {
		if *req.RelationType != "" && !model.ValidRelationTypes[*req.RelationType] {
			return nil, fmt.Errorf("invalid relation type: %s", *req.RelationType)
		}
		if *req.RelationType != "" {
			r.RelationType = *req.RelationType
		}
	}
	if req.Label != nil {
		r.Label = *req.Label
	}
	if req.ChapterRef != nil {
		r.ChapterRef = *req.ChapterRef
	}

	if err := s.overviewDAO.UpdateRelation(r); err != nil {
		return nil, fmt.Errorf("update relation failed: %w", err)
	}
	return r, nil
}

// DeleteRelation 删除人物关系
func (s *OverviewService) DeleteRelation(novelID uint, relationID uint) error {
	r, err := s.overviewDAO.GetRelation(relationID)
	if err != nil {
		return fmt.Errorf("relation not found: %w", err)
	}
	if r.NovelID != novelID {
		return fmt.Errorf("relation does not belong to this novel")
	}
	return s.overviewDAO.DeleteRelation(relationID)
}

// ========== AI 提取总览 ==========

// ExtractOverview 从全书章节概要中 AI 提取总览元数据
func (s *OverviewService) ExtractOverview(ctx context.Context, userID uint, novelID uint, req *ExtractOverviewRequest) (uint, error) {
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return 0, fmt.Errorf("novel not found: %w", err)
	}

	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return 0, fmt.Errorf("list chapters failed: %w", err)
	}

	if len(chapters) == 0 {
		return 0, fmt.Errorf("novel has no chapters")
	}

	// 构建章节索引（只含标题+概要，节省 token）
	chapterIndex := buildChapterIndex(chapters)

	prompt := buildExtractOverviewPrompt(novel, chapterIndex)

	modelName := req.ModelName
	if modelName == "" {
		modelName = "zhipu"
	}

	// 创建 AI 任务并异步分发（Dispatch 内部会创建 task 记录）
	task := &model.AITask{
		UserID:    userID,
		NovelID:   novelID,
		TaskType:  model.TaskTypeOverviewExtract,
		ModelName: modelName,
		Prompt:    prompt,
		Status:    model.TaskStatusPending,
	}
	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch extract task failed: %w", err)
	}

	return task.ID, nil
}

// ========== 变更工作流 ==========

// SubmitRevision 提交变更，触发分析工作流
func (s *OverviewService) SubmitRevision(ctx context.Context, userID uint, novelID uint, portfolioID uint, req *SubmitRevisionRequest) (uint, error) {
	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return 0, fmt.Errorf("list chapters failed: %w", err)
	}

	chapterIndex := buildChapterIndex(chapters)
	diffText := buildRevisionDiff(req.Changes)

	modelName := req.ModelName
	if modelName == "" {
		modelName = "zhipu"
	}

	// 通过 WorkflowService 提交分析工作流
	wfReq := &SubmitWorkflowRequest{
		PortfolioID:  portfolioID,
		WorkflowType: model.WorkflowTypeNovelRevision,
		ModelName:    modelName,
		Params: map[string]interface{}{
			"novel_id":      novelID,
			"diff_text":     diffText,
			"chapter_index": chapterIndex,
		},
	}

	workflowID, err := s.workflowSvc.SubmitWorkflow(ctx, userID, wfReq)
	if err != nil {
		return 0, fmt.Errorf("submit revision workflow failed: %w", err)
	}

	return workflowID, nil
}

// ExecuteRevision 确认执行变更（触发执行工作流）
func (s *OverviewService) ExecuteRevision(ctx context.Context, userID uint, novelID uint, portfolioID uint, req *ExecuteRevisionRequest) (uint, error) {
	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return 0, fmt.Errorf("list chapters failed: %w", err)
	}

	if len(chapters) == 0 {
		return 0, fmt.Errorf("novel has no chapters")
	}

	chapterIndex := buildChapterIndex(chapters)

	modelName := req.ModelName
	if modelName == "" {
		modelName = "zhipu"
	}

	wfReq := &SubmitWorkflowRequest{
		PortfolioID:  portfolioID,
		WorkflowType: model.WorkflowTypeNovelRevisionExec,
		ModelName:    modelName,
		Params: map[string]interface{}{
			"novel_id":      novelID,
			"revision_plan": req.RevisionPlan,
			"chapter_index": chapterIndex,
			"chapter_count": float64(len(chapters)),
		},
	}

	// 预填充每个章节的内容到 Params（会被注入 SharedState）
	for i, ch := range chapters {
		wfReq.Params[fmt.Sprintf("chapter_content_%d", i)] = ch.Content
	}

	workflowID, err := s.workflowSvc.SubmitWorkflow(ctx, userID, wfReq)
	if err != nil {
		return 0, fmt.Errorf("submit revision execute workflow failed: %w", err)
	}

	return workflowID, nil
}

// ========== AI 提取结果解析 ==========

// extractedOverview AI 提取总览的 JSON 结构
type extractedOverview struct {
	Plotlines []struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		ChapterRef string `json:"chapter_ref"`
		SortOrder  int    `json:"sort_order"`
	} `json:"plotlines"`
	Characters []struct {
		Title      string          `json:"title"`
		Content    string          `json:"content"`
		ChapterRef string          `json:"chapter_ref"`
		Tags       json.RawMessage `json:"tags"`
	} `json:"characters"`
	Relations []struct {
		From         string `json:"from"`
		To           string `json:"to"`
		RelationType string `json:"relation_type"`
		Label        string `json:"label"`
	} `json:"relations"`
	Foreshadows []struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		ChapterRef string `json:"chapter_ref"`
		Resolved   bool   `json:"resolved"`
	} `json:"foreshadows"`
}

// parseTagsField 兼容 AI 返回 tags 为 string 或 []string 的情况，统一转为逗号分隔字符串
func parseTagsField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// 尝试解析为 []string
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return strings.Join(arr, ",")
	}
	// 尝试解析为 string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// ParseExtractResult 解析 AI 提取总览结果并写入知识库和关系表
func (s *OverviewService) ParseExtractResult(novelID uint, taskID uint) error {
	task, err := s.aiTaskDAO.GetTask(context.Background(), taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}
	if task.Status != model.TaskStatusCompleted {
		return fmt.Errorf("task is not completed")
	}

	// 解析 AI 返回的 JSON
	var taskResult struct {
		Content string `json:"content"`
	}
	result := strings.TrimSpace(task.Result)
	if err := json.Unmarshal([]byte(result), &taskResult); err == nil && taskResult.Content != "" {
		result = taskResult.Content
	}
	// 尝试提取 JSON 对象块
	if idx := strings.Index(result, "{"); idx >= 0 {
		if end := strings.LastIndex(result, "}"); end > idx {
			result = result[idx : end+1]
		}
	}

	var extracted extractedOverview
	if err := json.Unmarshal([]byte(result), &extracted); err != nil {
		return fmt.Errorf("failed to parse extract result: %w, raw(100)=%s", err, truncate(result, 100))
	}

	log.Printf("[overview] ParseExtractResult: novelID=%d taskID=%d plotlines=%d characters=%d foreshadows=%d relations=%d",
		novelID, taskID, len(extracted.Plotlines), len(extracted.Characters), len(extracted.Foreshadows), len(extracted.Relations))

	// 清除旧数据（幂等：重复调用不会产生重复数据）
	for _, cat := range []string{model.KnowledgeCategoryPlotline, model.KnowledgeCategoryCharacter, model.KnowledgeCategoryForeshadow} {
		_ = s.knowledgeDAO.DeleteByNovelAndCategory(novelID, cat)
	}
	_ = s.overviewDAO.DeleteRelationsByNovel(novelID)

	// 写入情节线
	for _, p := range extracted.Plotlines {
		k := &model.NovelKnowledge{
			NovelID:    novelID,
			Category:   model.KnowledgeCategoryPlotline,
			Title:      p.Title,
			Content:    p.Content,
			ChapterRef: p.ChapterRef,
			SortOrder:  p.SortOrder,
			Status:     model.KnowledgeStatusConfirmed,
		}
		if err := s.knowledgeDAO.Create(k); err != nil {
			log.Printf("[overview] create plotline failed: %v", err)
		}
	}

	// 写入人物（同时建立 name→ID 映射，供关系写入使用）
	charNameToID := make(map[string]uint)
	charNormToID := make(map[string]uint) // 归一化名称映射（去标点空格）
	for _, ch := range extracted.Characters {
		// tags 兼容 string 和 []string 两种 AI 返回格式
		tags := parseTagsField(ch.Tags)
		k := &model.NovelKnowledge{
			NovelID:    novelID,
			Category:   model.KnowledgeCategoryCharacter,
			Title:      ch.Title,
			Content:    ch.Content,
			ChapterRef: ch.ChapterRef,
			Tags:       tags,
			Status:     model.KnowledgeStatusConfirmed,
		}
		if err := s.knowledgeDAO.Create(k); err != nil {
			log.Printf("[overview] create character failed: %v", err)
		} else {
			charNameToID[ch.Title] = k.ID
			charNormToID[normalizeCharName(ch.Title)] = k.ID
		}
	}

	// 写入伏笔
	for _, f := range extracted.Foreshadows {
		k := &model.NovelKnowledge{
			NovelID:    novelID,
			Category:   model.KnowledgeCategoryForeshadow,
			Title:      f.Title,
			Content:    f.Content,
			ChapterRef: f.ChapterRef,
			Resolved:   f.Resolved,
			Status:     model.KnowledgeStatusConfirmed,
		}
		if err := s.knowledgeDAO.Create(k); err != nil {
			log.Printf("[overview] create foreshadow failed: %v", err)
		}
	}

	// 写入人物关系（需要通过人物名称匹配 ID）
	for _, r := range extracted.Relations {
		fromName := strings.TrimSpace(r.From)
		toName := strings.TrimSpace(r.To)
		// 先精确匹配，再归一化匹配
		fromID, fromOK := charNameToID[fromName]
		if !fromOK {
			fromID, fromOK = charNormToID[normalizeCharName(fromName)]
		}
		toID, toOK := charNameToID[toName]
		if !toOK {
			toID, toOK = charNormToID[normalizeCharName(toName)]
		}
		if !fromOK || !toOK {
			log.Printf("[overview] skip relation: from=%q(ok=%v) to=%q(ok=%v)", r.From, fromOK, r.To, toOK)
			continue
		}
		relType := r.RelationType
		if !model.ValidRelationTypes[relType] {
			relType = model.RelationTypeCustom
		}
		rel := &model.NovelCharacterRelation{
			NovelID:         novelID,
			FromKnowledgeID: fromID,
			ToKnowledgeID:   toID,
			RelationType:    relType,
			Label:           r.Label,
		}
		if err := s.overviewDAO.CreateRelation(rel); err != nil {
			log.Printf("[overview] create relation failed: %v", err)
		}
	}

	log.Printf("[overview] ParseExtractResult done: novelID=%d", novelID)

	return nil
}

// ========== 内部方法 ==========

// buildChapterIndex 构建章节摘要索引（只含标题+概要截取200字，节省 token）
func buildChapterIndex(chapters []model.Chapter) string {
	var sb strings.Builder
	for _, ch := range chapters {
		summary := ch.Summary
		if utf8.RuneCountInString(summary) > 200 {
			runes := []rune(summary)
			summary = string(runes[:200]) + "..."
		}
		sb.WriteString(fmt.Sprintf("[Ch%d] %s: %s\n", ch.SortOrder, ch.Title, summary))
	}
	return sb.String()
}

// buildRevisionDiff 将前端变更列表序列化为结构化 Diff 文本
func buildRevisionDiff(changes []OverviewChange) string {
	var sb strings.Builder
	sb.WriteString("【变更列表】\n")
	for i, c := range changes {
		dataJSON, _ := json.Marshal(c.Data)
		sb.WriteString(fmt.Sprintf("%d. [%s] %s: %s\n", i+1, sanitizePromptInput(c.Type), sanitizePromptInput(c.Action), string(dataJSON)))
	}
	return sb.String()
}

// sanitizePromptInput 清理用户输入中可能干扰 Prompt 结构的内容
// 移除常见的 Prompt 注入标记，限制长度
func sanitizePromptInput(s string) string {
	// 移除可能被用于 Prompt 注入的分隔符
	replacer := strings.NewReplacer(
		"【", "[", "】", "]",
		"```", "'''",
		"<|", "&lt;|", "|>", "|&gt;",
		"<<", "&lt;&lt;", ">>", "&gt;&gt;",
	)
	s = replacer.Replace(s)
	// 限制单个字段最大长度
	if utf8.RuneCountInString(s) > 500 {
		runes := []rune(s)
		s = string(runes[:500]) + "..."
	}
	return s
}

// buildExtractOverviewPrompt 构建总览提取 Prompt
func buildExtractOverviewPrompt(novel *model.Novel, chapterIndex string) string {
	title := sanitizePromptInput(novel.Title)
	desc := sanitizePromptInput(novel.Description)
	return fmt.Sprintf(`你是一个小说结构分析助手。请从以下章节概要中提取小说的宏观结构信息。

【小说】%s
【简介】%s
【章节概要索引】
%s

请从以上章节概要中提取：
1. 主要情节线（plotline）：关键事件及涉及章节编号
2. 人物列表（character）：姓名、简要描述、首次出场章节编号
3. 人物关系（relation）：人物间关系类型（ally/enemy/mentor/lover/family/rival）及描述
4. 伏笔/反转（foreshadow）：埋设章节编号、是否已揭示（resolved: true/false）

以 JSON 格式返回，结构如下：
{
  "plotlines": [{"title": "...", "content": "...", "chapter_ref": "1,3,5", "sort_order": 1}],
  "characters": [{"title": "姓名", "content": "描述", "chapter_ref": "1", "tags": "标签"}],
  "relations": [{"from": "人物A姓名", "to": "人物B姓名", "relation_type": "ally", "label": "描述"}],
  "foreshadows": [{"title": "...", "content": "...", "chapter_ref": "2", "resolved": false}]
}

仅提取明确出现在概要中的信息，不要推测。`, title, desc, chapterIndex)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// normalizeCharName 归一化人物名称：去除标点、括号、空格等，只保留字母/数字/汉字
func normalizeCharName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
