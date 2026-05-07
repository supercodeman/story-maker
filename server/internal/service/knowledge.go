// server/internal/service/knowledge.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// KnowledgeService 知识库业务逻辑层
type KnowledgeService struct {
	knowledgeDAO  *dao.KnowledgeDAO
	relationDAO   *dao.KnowledgeRelationDAO
	novelDAO      *dao.NovelDAO
	aiTaskDAO     *dao.AITaskDAO
	dispatcher    *agent.Dispatcher
	modelRegistry *ModelRegistryService
}

// SetModelRegistry 延迟注入模型注册中心
func (s *KnowledgeService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// NewKnowledgeService 创建 KnowledgeService 实例
func NewKnowledgeService(aiTaskDAO *dao.AITaskDAO, dispatcher *agent.Dispatcher) *KnowledgeService {
	return &KnowledgeService{
		knowledgeDAO: dao.NewKnowledgeDAO(),
		relationDAO:  dao.NewKnowledgeRelationDAO(),
		novelDAO:     dao.NewNovelDAO(),
		aiTaskDAO:    aiTaskDAO,
		dispatcher:   dispatcher,
	}
}

// ========== 请求参数定义 ==========

// CreateKnowledgeRequest 创建知识条目请求
type CreateKnowledgeRequest struct {
	Category   string `json:"category" binding:"required"`
	Title      string `json:"title" binding:"required,max=200"`
	Content    string `json:"content" binding:"required"`
	Tags       string `json:"tags"`
	ChapterRef string `json:"chapter_ref"`
	Priority   int    `json:"priority"`
}

// UpdateKnowledgeRequest 更新知识条目请求
type UpdateKnowledgeRequest struct {
	Category   string `json:"category"`
	Title      string `json:"title" binding:"omitempty,max=200"`
	Content    string `json:"content"`
	Tags       string `json:"tags"`
	ChapterRef string `json:"chapter_ref"`
	Priority   *int   `json:"priority"`
	Status     string `json:"status"`
	Resolved   *bool  `json:"resolved"`
	SortOrder  *int   `json:"sort_order"`
}

// ExtractKnowledgeRequest AI 提取知识请求
type ExtractKnowledgeRequest struct {
	ChapterID uint   `json:"chapter_id" binding:"required"`
	ModelName string `json:"model_name"`
}

// ========== CRUD 操作 ==========

// Create 创建知识条目
func (s *KnowledgeService) Create(novelID uint, req *CreateKnowledgeRequest) (*model.NovelKnowledge, error) {
	if !model.ValidKnowledgeCategories[req.Category] {
		return nil, fmt.Errorf("invalid category: %s", req.Category)
	}

	k := &model.NovelKnowledge{
		NovelID:    novelID,
		Category:   req.Category,
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		ChapterRef: req.ChapterRef,
		Priority:   req.Priority,
		Status:     model.KnowledgeStatusConfirmed,
	}

	if err := s.knowledgeDAO.Create(k); err != nil {
		return nil, err
	}
	return k, nil
}

// Get 获取知识条目
func (s *KnowledgeService) Get(id uint) (*model.NovelKnowledge, error) {
	return s.knowledgeDAO.Get(id)
}

// Update 更新知识条目
func (s *KnowledgeService) Update(id uint, req *UpdateKnowledgeRequest) (*model.NovelKnowledge, error) {
	k, err := s.knowledgeDAO.Get(id)
	if err != nil {
		return nil, err
	}

	if req.Category != "" {
		if !model.ValidKnowledgeCategories[req.Category] {
			return nil, fmt.Errorf("invalid category: %s", req.Category)
		}
		k.Category = req.Category
	}
	if req.Title != "" {
		k.Title = req.Title
	}
	if req.Content != "" {
		k.Content = req.Content
	}
	if req.Tags != "" {
		k.Tags = req.Tags
	}
	if req.ChapterRef != "" {
		k.ChapterRef = req.ChapterRef
	}
	if req.Priority != nil {
		k.Priority = *req.Priority
	}
	if req.Status != "" {
		if !model.ValidKnowledgeStatuses[req.Status] {
			return nil, fmt.Errorf("invalid status: %s", req.Status)
		}
		k.Status = req.Status
	}
	if req.Resolved != nil {
		k.Resolved = *req.Resolved
	}
	if req.SortOrder != nil {
		k.SortOrder = *req.SortOrder
	}

	if err := s.knowledgeDAO.Update(k); err != nil {
		return nil, err
	}
	return k, nil
}

// Delete 删除知识条目
func (s *KnowledgeService) Delete(id uint) error {
	return s.knowledgeDAO.Delete(id)
}

// List 获取小说的知识条目列表
func (s *KnowledgeService) List(novelID uint, category, status string) ([]model.NovelKnowledge, error) {
	if category != "" {
		return s.knowledgeDAO.ListByNovelAndCategory(novelID, category)
	}
	if status != "" {
		return s.knowledgeDAO.ListByNovelAndStatus(novelID, status)
	}
	return s.knowledgeDAO.ListByNovel(novelID)
}

// Confirm 确认待审核条目
func (s *KnowledgeService) Confirm(id uint) error {
	return s.knowledgeDAO.ConfirmPending(id)
}

// BatchConfirm 批量确认小说下所有待审核条目
func (s *KnowledgeService) BatchConfirm(novelID uint) error {
	return s.knowledgeDAO.BatchConfirmByNovel(novelID)
}

// Search 按关键词搜索知识条目
func (s *KnowledgeService) Search(novelID uint, keyword string) ([]model.NovelKnowledge, error) {
	return s.knowledgeDAO.SearchByTags(novelID, keyword)
}

// ========== AI 知识提取 ==========

// ExtractFromChapter 从章节内容中 AI 提取知识条目
func (s *KnowledgeService) ExtractFromChapter(ctx context.Context, userID, novelID uint, req *ExtractKnowledgeRequest) (uint, error) {
	chapter, err := s.novelDAO.GetChapter(req.ChapterID)
	if err != nil {
		return 0, fmt.Errorf("chapter not found: %w", err)
	}
	if chapter.NovelID != novelID {
		return 0, fmt.Errorf("chapter does not belong to this novel")
	}

	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return 0, fmt.Errorf("novel not found: %w", err)
	}

	// 构建提取 Prompt
	prompt := s.buildExtractPrompt(novel, chapter)

	modelName := req.ModelName
	if modelName == "" {
		if s.modelRegistry != nil {
			modelName = s.modelRegistry.GetDefaultModel(model.CapTextGen)
		} else {
			modelName = "zhipu"
		}
	}

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: novel.PortfolioID,
		TaskType:    model.TaskTypeKnowledgeExtract,
		ModelName:   modelName,
		Prompt:      prompt,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// ParseExtractResult 解析 AI 提取结果并写入知识库
func (s *KnowledgeService) ParseExtractResult(novelID, chapterID, taskID uint) ([]model.NovelKnowledge, error) {
	task, err := s.aiTaskDAO.GetTask(context.Background(), taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	if task.Status != model.TaskStatusCompleted {
		return nil, fmt.Errorf("task is not completed")
	}

	// 解析 AI 返回的 JSON 数组
	var extracted []struct {
		Category string          `json:"category"`
		Title    string          `json:"title"`
		Content  string          `json:"content"`
		Tags     json.RawMessage `json:"tags"`
	}

	// task.Result 是 executor 返回的 JSON，格式为 {"content":"...AI文本..."}
	// 先提取 content 字段
	var taskResult struct {
		Content string `json:"content"`
	}
	result := strings.TrimSpace(task.Result)
	if err := json.Unmarshal([]byte(result), &taskResult); err == nil && taskResult.Content != "" {
		result = taskResult.Content
	}
	// 尝试提取 JSON 数组块（AI 可能返回 markdown 包裹的 JSON）
	if idx := strings.Index(result, "["); idx >= 0 {
		if end := strings.LastIndex(result, "]"); end > idx {
			result = result[idx : end+1]
		}
	}

	if err := json.Unmarshal([]byte(result), &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse extract result: %w", err)
	}

	chapterRef := fmt.Sprintf("%d", chapterID)
	var items []model.NovelKnowledge
	for _, e := range extracted {
		if !model.ValidKnowledgeCategories[e.Category] {
			e.Category = model.KnowledgeCategoryCustom
		}
		items = append(items, model.NovelKnowledge{
			NovelID:    novelID,
			Category:   e.Category,
			Title:      e.Title,
			Content:    e.Content,
			Tags:       parseTagsField(e.Tags),
			ChapterRef: chapterRef,
			Status:     model.KnowledgeStatusPending,
		})
	}

	if len(items) > 0 {
		if err := s.knowledgeDAO.BatchCreate(items); err != nil {
			return nil, err
		}
	}

	return items, nil
}

// ========== 上下文构建（供 NovelService 调用） ==========

// BuildKnowledgeContext 构建知识库上下文字符串，用于注入 Prompt
// maxChars 限制总字数，避免 Token 浪费
func (s *KnowledgeService) BuildKnowledgeContext(novelID uint, maxChars int) (string, string, string) {
	items, err := s.knowledgeDAO.ListConfirmedByNovel(novelID)
	if err != nil || len(items) == 0 {
		return "", "", ""
	}

	if maxChars <= 0 {
		maxChars = 3000
	}

	// 按类别分组
	grouped := make(map[string][]model.NovelKnowledge)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}

	var fullContext strings.Builder
	var characters strings.Builder
	var worldview strings.Builder
	totalChars := 0

	// 按优先级顺序构建：人物 > 世界观 > 伏笔 > 剧情 > 文风 > 自定义
	categoryOrder := []string{
		model.KnowledgeCategoryCharacter,
		model.KnowledgeCategoryWorldview,
		model.KnowledgeCategoryForeshadow,
		model.KnowledgeCategoryPlotline,
		model.KnowledgeCategoryStyle,
		model.KnowledgeCategoryCustom,
	}

	for _, cat := range categoryOrder {
		catItems, ok := grouped[cat]
		if !ok || len(catItems) == 0 {
			continue
		}

		label := model.KnowledgeCategoryLabel[cat]
		section := fmt.Sprintf("【%s】\n", label)

		for _, item := range catItems {
			entry := fmt.Sprintf("- %s：%s\n", item.Title, item.Content)
			if totalChars+len([]rune(section))+len([]rune(entry)) > maxChars {
				break
			}
			section += entry
			totalChars += len([]rune(entry))

			// 同时填充分类字段
			switch cat {
			case model.KnowledgeCategoryCharacter:
				characters.WriteString(entry)
			case model.KnowledgeCategoryWorldview:
				worldview.WriteString(entry)
			}
		}

		fullContext.WriteString(section)
		fullContext.WriteString("\n")

		if totalChars >= maxChars {
			break
		}
	}

	// 追加实体关系边输出
	relations, err := s.relationDAO.ListByNovel(novelID)
	if err == nil && len(relations) > 0 {
		// 构建实体 ID → 标题的映射
		entityTitles := make(map[uint]string)
		for _, item := range items {
			entityTitles[item.ID] = item.Title
		}

		var relSection strings.Builder
		relSection.WriteString("【实体关系】\n")
		relChars := 0
		for _, rel := range relations {
			fromTitle := entityTitles[rel.FromEntityID]
			toTitle := entityTitles[rel.ToEntityID]
			if fromTitle == "" || toTitle == "" {
				continue
			}
			entry := fmt.Sprintf("- %s → %s [%s]", fromTitle, toTitle, rel.RelationType)
			if rel.Description != "" {
				entry += "：" + rel.Description
			}
			entry += "\n"
			entryChars := len([]rune(entry))
			if totalChars+relChars+entryChars > maxChars {
				break
			}
			relSection.WriteString(entry)
			relChars += entryChars
		}
		if relChars > 0 {
			fullContext.WriteString("\n")
			fullContext.WriteString(relSection.String())
		}
	}

	return fullContext.String(), characters.String(), worldview.String()
}

// ========== 内部方法 ==========

// buildExtractPrompt 构建知识提取的 Prompt
func (s *KnowledgeService) buildExtractPrompt(novel *model.Novel, chapter *model.Chapter) string {
	return fmt.Sprintf(`你是一个小说知识提取助手。请从以下章节内容中提取关键知识条目。

【小说】%s
【小说简介】%s
【章节】%s
【章节概要】%s
【章节内容】
%s

请提取以下类别的知识条目：
- character：人物信息（姓名、外貌、性格、能力、关系）
- worldview：世界观设定（地理、历史、体系、规则）
- plotline：剧情线索（关键事件、转折点）
- foreshadow：伏笔（暗示、未解之谜）
- style：文风特征（叙事手法、语言风格）

请以 JSON 数组格式返回，每个条目包含 category、title、content、tags 字段。
仅提取明确出现在文本中的信息，不要推测。
如果某个类别没有可提取的内容，跳过即可。

示例格式：
[
  {"category": "character", "title": "张三", "content": "男，25岁，剑修，性格沉稳", "tags": "主角,剑修"},
  {"category": "worldview", "title": "灵气体系", "content": "分为九品，每品三阶", "tags": "修炼,体系"}
]`,
		novel.Title, novel.Description,
		chapter.Title, chapter.Summary, chapter.Content)
}

// extractedRelation AI 提取的关系结构
type extractedRelation struct {
	FromEntity   string `json:"from_entity"`
	ToEntity     string `json:"to_entity"`
	RelationType string `json:"relation_type"`
	Description  string `json:"description"`
}

// ExtractRelations 从章节内容中提取实体关系
func (s *KnowledgeService) ExtractRelations(ctx context.Context, novelID, chapterID uint, modelName string) error {
	// 获取章节信息
	chapter, err := s.novelDAO.GetChapter(chapterID)
	if err != nil {
		return fmt.Errorf("获取章节失败: %w", err)
	}
	if strings.TrimSpace(chapter.Content) == "" {
		return nil
	}

	// 获取小说的所有知识实体（用于匹配）
	entities, err := s.knowledgeDAO.ListConfirmedByNovel(novelID)
	if err != nil {
		return fmt.Errorf("获取知识实体失败: %w", err)
	}
	if len(entities) == 0 {
		return nil
	}

	// 构建实体名称列表
	var entityNames []string
	entityMap := make(map[string]uint) // title → ID
	for _, e := range entities {
		entityNames = append(entityNames, e.Title)
		entityMap[e.Title] = e.ID
	}

	if modelName == "" {
		if s.modelRegistry != nil {
			modelName = s.modelRegistry.GetDefaultModel(model.CapTextGen)
		} else {
			modelName = "qwen"
		}
	}
	provider, err := s.dispatcher.GetProviderWithKey(ctx, modelName)
	if err != nil {
		return fmt.Errorf("获取 AI Provider 失败: %w", err)
	}

	// 截取章节内容
	content := chapter.Content
	runes := []rune(content)
	if len(runes) > 5000 {
		content = string(runes[:5000])
	}

	prompt := fmt.Sprintf(`你是一个小说知识图谱构建助手。请从以下章节内容中提取实体之间的关系。

已知实体列表：%s

【章节内容】
%s

请提取实体之间的关系，以 JSON 数组格式返回：
[
  {"from_entity": "实体A名称", "to_entity": "实体B名称", "relation_type": "关系类型", "description": "关系描述"}
]

可用的关系类型：
- master_of：师徒关系（A是B的师父）
- enemy_of：敌对关系
- ally_of：盟友关系
- family_of：亲属关系
- located_in：位于（A位于B）
- member_of：隶属（A是B的成员）
- created_by：创造（A由B创造）
- evolves_to：进化/升级（A进化为B）

仅提取明确出现在文本中的关系，不要推测。实体名称必须与已知实体列表中的名称完全匹配。
如果没有可提取的关系，返回空数组 []。`,
		strings.Join(entityNames, "、"), content)

	resp, err := provider.GenerateText(ctx, &agent.TextRequest{
		Prompt:      prompt,
		MaxTokens:   2048,
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("AI 提取关系失败: %w", err)
	}

	// 解析 JSON
	var relations []extractedRelation
	respContent := strings.TrimSpace(resp.Content)
	// 尝试提取 JSON 数组
	startIdx := strings.Index(respContent, "[")
	endIdx := strings.LastIndex(respContent, "]")
	if startIdx >= 0 && endIdx > startIdx {
		respContent = respContent[startIdx : endIdx+1]
	}
	if err := json.Unmarshal([]byte(respContent), &relations); err != nil {
		log.Printf("[knowledge] 解析关系 JSON 失败: %v", err)
		return nil
	}

	// 转换并保存
	var dbRels []model.KnowledgeRelation
	for _, rel := range relations {
		fromID, fromOK := entityMap[rel.FromEntity]
		toID, toOK := entityMap[rel.ToEntity]
		if !fromOK || !toOK {
			continue
		}
		if !model.ValidKnowledgeRelationTypes[rel.RelationType] {
			continue
		}
		dbRels = append(dbRels, model.KnowledgeRelation{
			NovelID:      novelID,
			FromEntityID: fromID,
			ToEntityID:   toID,
			RelationType: rel.RelationType,
			Description:  rel.Description,
			ChapterRef:   fmt.Sprintf("第%d章", chapter.SortOrder),
		})
	}

	if len(dbRels) > 0 {
		if err := s.relationDAO.BatchCreate(dbRels); err != nil {
			return fmt.Errorf("保存关系失败: %w", err)
		}
		log.Printf("[knowledge] 从章节 %d 提取到 %d 条实体关系", chapterID, len(dbRels))
	}

	return nil
}

// GetEntityGraph 给定实体 ID，返回 N 跳内的关联实体和关系
func (s *KnowledgeService) GetEntityGraph(novelID, entityID uint, depth int) ([]model.NovelKnowledge, []model.KnowledgeRelation, error) {
	if depth <= 0 {
		depth = 1
	}

	// 获取关系
	relations, err := s.relationDAO.ListByEntity(entityID, depth)
	if err != nil {
		return nil, nil, fmt.Errorf("查询关系失败: %w", err)
	}

	// 收集所有涉及的实体 ID
	entityIDs := map[uint]bool{entityID: true}
	for _, rel := range relations {
		entityIDs[rel.FromEntityID] = true
		entityIDs[rel.ToEntityID] = true
	}

	// 批量获取实体
	var idList []uint
	for id := range entityIDs {
		idList = append(idList, id)
	}

	var entities []model.NovelKnowledge
	for _, id := range idList {
		entity, err := s.knowledgeDAO.Get(id)
		if err != nil {
			continue
		}
		if entity.NovelID == novelID {
			entities = append(entities, *entity)
		}
	}

	return entities, relations, nil
}

// GenerateQuestions 基于知识点生成面试题
func (s *KnowledgeService) GenerateQuestions(ctx context.Context, userID, pointID uint, count int, difficulty string) (string, error) {
	// 获取知识点详情
	knowledge, err := s.knowledgeDAO.Get(pointID)
	if err != nil {
		return "", fmt.Errorf("获取知识点失败: %w", err)
	}

	// 构造prompt
	prompt := fmt.Sprintf(`你是一位资深的技术面试官。请基于以下知识点生成 %d 道面试题。

知识点标题：%s
知识点内容：%s
难度要求：%s

要求：
1. 题目要有深度，考察对知识点的理解和应用能力
2. 答案要详细，包含原理、场景、最佳实践
3. 每道题包含：question（问题）、answer（答案）、difficulty（难度）、tags（标签，逗号分隔）
4. 返回格式为JSON数组，不要包含任何其他文字

示例格式：
[
  {
    "question": "问题内容",
    "answer": "详细答案",
    "difficulty": "%s",
    "tags": "标签1, 标签2, 标签3"
  }
]`, count, knowledge.Title, knowledge.Content, difficulty, difficulty)

	// 选择模型
	modelName := "qwen"

	// 创建AI任务
	task := &model.AITask{
		UserID:      userID,
		TaskType:    "text",
		ModelName:   modelName,
		Prompt:      prompt,
		Status:      model.TaskStatusPending,
		PortfolioID: 0,
	}

	if err := s.aiTaskDAO.CreateTask(ctx, task); err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}

	// 同步执行AI任务
	_, err = s.dispatcher.ExecuteSingle(ctx, task)
	if err != nil {
		return "", fmt.Errorf("AI生成失败: %w", err)
	}

	// 重新加载任务获取结果
	task, err = s.aiTaskDAO.GetTask(ctx, task.ID)
	if err != nil {
		return "", fmt.Errorf("获取任务结果失败: %w", err)
	}

	if task.Status != model.TaskStatusCompleted {
		return "", fmt.Errorf("任务执行失败: %s", task.ErrorMsg)
	}

	return task.Result, nil
}
