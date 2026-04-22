// server/internal/service/fact_collector.go
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
	"ai-curton/server/internal/vectordb"
)

// FactCollector 事实采集器，从章节内容中提取结构化事实
type FactCollector struct {
	factDAO       *dao.NovelFactDAO
	novelDAO      *dao.NovelDAO
	milvus        *vectordb.MilvusClient
	dispatcher    *agent.Dispatcher
	knowledgeSvc  *KnowledgeService
	modelRegistry *ModelRegistryService
}

// NewFactCollector 创建事实采集器
func NewFactCollector(factDAO *dao.NovelFactDAO, novelDAO *dao.NovelDAO, milvus *vectordb.MilvusClient, dispatcher *agent.Dispatcher) *FactCollector {
	return &FactCollector{
		factDAO:    factDAO,
		novelDAO:   novelDAO,
		milvus:     milvus,
		dispatcher: dispatcher,
	}
}

// SetKnowledgeSvc 注入知识服务（避免循环依赖）
func (c *FactCollector) SetKnowledgeSvc(svc *KnowledgeService) {
	c.knowledgeSvc = svc
}

// SetModelRegistry 注入模型注册服务
func (c *FactCollector) SetModelRegistry(mr *ModelRegistryService) {
	c.modelRegistry = mr
}

// extractedFact AI 提取的事实结构
type extractedFact struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Source  string `json:"source"`
}

// CollectFromChapter 从章节内容中采集事实（异步调用）
func (c *FactCollector) CollectFromChapter(ctx context.Context, novel *model.Novel, chapter *model.Chapter, userID uint) {
	log.Printf("[fact-collector] 开始从章节 %d(%s) 采集事实", chapter.ID, chapter.Title)

	// 跳过空内容章节
	if strings.TrimSpace(chapter.Content) == "" {
		log.Printf("[fact-collector] 章节 %d 内容为空，跳过采集", chapter.ID)
		return
	}

	// AI 提取事实
	facts, err := c.extractFacts(ctx, novel, chapter)
	if err != nil {
		log.Printf("[fact-collector] AI 提取事实失败: %v", err)
		return
	}
	if len(facts) == 0 {
		log.Printf("[fact-collector] 章节 %d 未提取到事实", chapter.ID)
		return
	}

	log.Printf("[fact-collector] 从章节 %d 提取到 %d 条事实", chapter.ID, len(facts))

	// 去重/合并 + 存储
	var newFacts []*model.NovelMemoryFact
	for _, ef := range facts {
		if !model.ValidFactTypes[ef.Type] {
			continue
		}

		fact, err := c.deduplicateAndSave(novel.ID, userID, chapter.ID, &ef)
		if err != nil {
			log.Printf("[fact-collector] 保存事实失败: %v", err)
			continue
		}
		if fact != nil {
			newFacts = append(newFacts, fact)
		}
	}

	// 批量 Embedding + 插入 Milvus
	if len(newFacts) > 0 {
		c.EmbedAndStore(ctx, newFacts)
	}

	log.Printf("[fact-collector] 章节 %d 采集完成，新增 %d 条事实", chapter.ID, len(newFacts))

	// 异步提取知识图谱实体关系
	if c.knowledgeSvc != nil {
		go func() {
			if err := c.knowledgeSvc.ExtractRelations(ctx, novel.ID, chapter.ID, "qwen"); err != nil {
				log.Printf("[fact-collector] 提取实体关系失败: %v", err)
			}
		}()
	}
}

// ColdStart 冷启动：从已有数据中初始化事实库
func (c *FactCollector) ColdStart(ctx context.Context, novel *model.Novel, userID uint) error {
	log.Printf("[fact-collector] 小说 %d(%s) 冷启动开始", novel.ID, novel.Title)

	var allFacts []*model.NovelMemoryFact

	// 1. 从已有章节概要提取事实
	chapters, err := c.novelDAO.ListChaptersByNovel(novel.ID)
	if err == nil && len(chapters) > 0 {
		chapterFacts, err := c.extractFromChapterSummaries(ctx, novel, chapters)
		if err != nil {
			log.Printf("[fact-collector] 从章节概要提取失败: %v", err)
		} else {
			for _, ef := range chapterFacts {
				fact := &model.NovelMemoryFact{
					NovelID:  novel.ID,
					UserID:   userID,
					FactType: ef.Type,
					Title:    ef.Title,
					Content:  ef.Content,
					Version:  1,
				}
				if err := c.factDAO.Create(fact); err == nil {
					allFacts = append(allFacts, fact)
				}
			}
		}
	}

	// 2. 从知识库条目转换
	knowledgeFacts := c.convertKnowledge(novel.ID, userID)
	allFacts = append(allFacts, knowledgeFacts...)

	// 3. 从人物关系转换
	relationFacts := c.convertRelations(novel.ID, userID)
	allFacts = append(allFacts, relationFacts...)

	// 4. 批量 Embedding + 插入 Milvus
	if len(allFacts) > 0 {
		c.EmbedAndStore(ctx, allFacts)
	}

	log.Printf("[fact-collector] 小说 %d 冷启动完成，共 %d 条事实", novel.ID, len(allFacts))
	return nil
}

// FullColdStart 全量冷启动：清除旧数据后，遍历所有章节正文+知识库+人物关系做全量事实采集
func (c *FactCollector) FullColdStart(ctx context.Context, novel *model.Novel, userID uint) error {
	log.Printf("[fact-collector] ===== 全量冷启动开始 ===== novel_id=%d(%s) user_id=%d", novel.ID, novel.Title, userID)

	// 1. 清除旧数据：先查出所有 factID 用于删除 Milvus 向量
	existingFacts, err := c.factDAO.ListByNovelActive(novel.ID)
	if err != nil {
		log.Printf("[fact-collector] 查询已有事实失败: %v", err)
	}
	if len(existingFacts) > 0 {
		factIDs := make([]int64, len(existingFacts))
		for i, f := range existingFacts {
			factIDs[i] = int64(f.ID)
		}
		// 删除 Milvus 向量
		if c.milvus != nil {
			if err := c.milvus.DeleteByFactIDs(ctx, factIDs); err != nil {
				log.Printf("[fact-collector] 删除 Milvus 向量失败: %v", err)
			} else {
				log.Printf("[fact-collector] 已删除 Milvus 向量 %d 条", len(factIDs))
			}
		}
	}
	// 删除 MySQL 事实记录
	if err := c.factDAO.DeleteByNovel(novel.ID); err != nil {
		log.Printf("[fact-collector] 删除 MySQL 事实失败: %v", err)
		return fmt.Errorf("清除旧事实失败: %w", err)
	}
	log.Printf("[fact-collector] 已清除小说 %d 的所有旧事实", novel.ID)

	var allFacts []*model.NovelMemoryFact

	// 2. 遍历所有章节，逐章从正文提取事实
	chapters, err := c.novelDAO.ListChaptersByNovel(novel.ID)
	if err != nil {
		log.Printf("[fact-collector] 查询章节列表失败: %v", err)
		return fmt.Errorf("查询章节列表失败: %w", err)
	}
	log.Printf("[fact-collector] 共 %d 个章节待采集", len(chapters))

	for i := range chapters {
		ch := &chapters[i]
		if strings.TrimSpace(ch.Content) == "" {
			log.Printf("[fact-collector] 章节 %d(%s) 内容为空，跳过", ch.ID, ch.Title)
			continue
		}

		log.Printf("[fact-collector] 正在采集章节 %d/%d: %s", i+1, len(chapters), ch.Title)
		facts, err := c.extractFacts(ctx, novel, ch)
		if err != nil {
			log.Printf("[fact-collector] 章节 %d 提取失败: %v", ch.ID, err)
			continue
		}
		log.Printf("[fact-collector] 章节 %d 提取到 %d 条事实", ch.ID, len(facts))

		for _, ef := range facts {
			fact, err := c.deduplicateAndSave(novel.ID, userID, ch.ID, &ef)
			if err != nil {
				log.Printf("[fact-collector] 保存事实失败: %v", err)
				continue
			}
			if fact != nil {
				allFacts = append(allFacts, fact)
			}
		}
	}

	// 3. 从知识库条目转换
	knowledgeFacts := c.convertKnowledge(novel.ID, userID)
	allFacts = append(allFacts, knowledgeFacts...)
	log.Printf("[fact-collector] 知识库转换 %d 条事实", len(knowledgeFacts))

	// 4. 从人物关系转换
	relationFacts := c.convertRelations(novel.ID, userID)
	allFacts = append(allFacts, relationFacts...)
	log.Printf("[fact-collector] 人物关系转换 %d 条事实", len(relationFacts))

	// 5. 批量 Embedding + 写入 Milvus
	if len(allFacts) > 0 {
		c.EmbedAndStore(ctx, allFacts)
	}

	log.Printf("[fact-collector] ===== 全量冷启动完成 ===== novel_id=%d 共 %d 条事实", novel.ID, len(allFacts))
	return nil
}

// extractFacts 调用 AI 从章节内容中提取事实
func (c *FactCollector) extractFacts(ctx context.Context, novel *model.Novel, chapter *model.Chapter) ([]extractedFact, error) {
	systemPrompt := `你是一个专业的小说内容分析助手。请从以下章节内容中提取关键事实信息。

提取规则：
1. character_trait：人物的性格特征、能力变化、重要决定
2. plot_event：推动剧情发展的关键事件
3. foreshadow：伏笔的埋设或回收（标注是"埋设"还是"回收"）
4. worldview_rule：世界观规则、力量体系、社会制度
5. relationship_change：人物关系的建立、变化、破裂

每个事实需包含：type、title（实体名）、content（简洁描述，50-100字）、source（原文关键句）

以 JSON 数组格式返回，不要返回其他内容。`

	// 截取章节内容，避免超长
	content := chapter.Content
	runes := []rune(content)
	if len(runes) > 5000 {
		content = string(runes[:5000])
	}

	userPrompt := fmt.Sprintf("小说：%s\n章节：%s\n概要：%s\n\n正文：\n%s",
		novel.Title, chapter.Title, chapter.Summary, content)

	resp, err := c.dispatcher.GenerateTextWithFallback(ctx, "qwen", &agent.TextRequest{
		Prompt:       userPrompt,
		CharacterCtx: systemPrompt,
		MaxTokens:    4096,
		Temperature:  0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %w", err)
	}

	return parseFactsJSON(resp.Content)
}

// extractFromChapterSummaries 从章节概要批量提取事实（冷启动用）
func (c *FactCollector) extractFromChapterSummaries(ctx context.Context, novel *model.Novel, chapters []model.Chapter) ([]extractedFact, error) {
	// 拼接所有章节概要
	var summaryParts []string
	for _, ch := range chapters {
		if ch.Summary != "" {
			summaryParts = append(summaryParts, fmt.Sprintf("【%s】%s", ch.Title, ch.Summary))
		}
	}
	if len(summaryParts) == 0 {
		return nil, nil
	}

	systemPrompt := `你是一个专业的小说内容分析助手。请从以下章节概要中提取关键事实信息。
提取规则同上：character_trait、plot_event、foreshadow、worldview_rule、relationship_change。
每个事实需包含：type、title、content（50-100字）、source。
以 JSON 数组格式返回。`

	summaryText := strings.Join(summaryParts, "\n")
	runes := []rune(summaryText)
	if len(runes) > 5000 {
		summaryText = string(runes[:5000])
	}

	userPrompt := fmt.Sprintf("小说：%s\n描述：%s\n\n章节概要：\n%s",
		novel.Title, novel.Description, summaryText)

	resp, err := c.dispatcher.GenerateTextWithFallback(ctx, "qwen", &agent.TextRequest{
		Prompt:       userPrompt,
		CharacterCtx: systemPrompt,
		MaxTokens:    4096,
		Temperature:  0.3,
	})
	if err != nil {
		return nil, err
	}

	return parseFactsJSON(resp.Content)
}

// convertKnowledge 将知识库条目转换为事实
func (c *FactCollector) convertKnowledge(novelID, userID uint) []*model.NovelMemoryFact {
	var knowledges []model.NovelKnowledge
	model.DB.Where("novel_id = ? AND status = ?", novelID, model.KnowledgeStatusConfirmed).Find(&knowledges)

	// 知识类别 → 事实类型映射
	categoryToFactType := map[string]string{
		model.KnowledgeCategoryCharacter:  model.FactTypeCharacterTrait,
		model.KnowledgeCategoryWorldview:  model.FactTypeWorldviewRule,
		model.KnowledgeCategoryPlotline:   model.FactTypePlotEvent,
		model.KnowledgeCategoryForeshadow: model.FactTypeForeshadow,
	}

	var facts []*model.NovelMemoryFact
	for _, k := range knowledges {
		factType, ok := categoryToFactType[k.Category]
		if !ok {
			continue
		}
		fact := &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: factType,
			Title:    k.Title,
			Content:  k.Content,
			Version:  1,
		}
		if err := c.factDAO.Create(fact); err == nil {
			facts = append(facts, fact)
		}
	}
	return facts
}

// convertRelations 将人物关系转换为事实
func (c *FactCollector) convertRelations(novelID, userID uint) []*model.NovelMemoryFact {
	var relations []model.NovelCharacterRelation
	model.DB.Where("novel_id = ?", novelID).Find(&relations)

	var facts []*model.NovelMemoryFact
	for _, r := range relations {
		// 查询关系两端的人物名称
		var fromKnowledge, toKnowledge model.NovelKnowledge
		if err := model.DB.First(&fromKnowledge, r.FromKnowledgeID).Error; err != nil {
			continue
		}
		if err := model.DB.First(&toKnowledge, r.ToKnowledgeID).Error; err != nil {
			continue
		}

		label := r.Label
		if label == "" {
			label = model.RelationTypeLabel[r.RelationType]
		}

		fact := &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: model.FactTypeRelationshipChange,
			Title:    fmt.Sprintf("%s-%s", fromKnowledge.Title, toKnowledge.Title),
			Content:  fmt.Sprintf("%s与%s的关系：%s", fromKnowledge.Title, toKnowledge.Title, label),
			Version:  1,
		}
		if err := c.factDAO.Create(fact); err == nil {
			facts = append(facts, fact)
		}
	}
	return facts
}

// deduplicateAndSave 去重/合并后保存事实
func (c *FactCollector) deduplicateAndSave(novelID, userID, chapterID uint, ef *extractedFact) (*model.NovelMemoryFact, error) {
	existing, err := c.factDAO.FindByNovelAndTitle(novelID, ef.Type, ef.Title)
	if err == nil && existing != nil {
		// 已有同名事实，检查内容是否有实质变化
		if isSimilarContent(existing.Content, ef.Content) {
			log.Printf("[fact-collector] 去重跳过: type=%s title=%q (内容相似)", ef.Type, ef.Title)
			return nil, nil // 内容相似，跳过
		}

		// 内容有变化，创建新版本并标记旧版本
		log.Printf("[fact-collector] 事实更新: type=%s title=%q v%d→v%d", ef.Type, ef.Title, existing.Version, existing.Version+1)
		newFact := &model.NovelMemoryFact{
			NovelID:    novelID,
			UserID:     userID,
			ChapterID:  chapterID,
			FactType:   ef.Type,
			Title:      ef.Title,
			Content:    ef.Content,
			SourceText: ef.Source,
			Version:    existing.Version + 1,
		}
		if err := c.factDAO.Create(newFact); err != nil {
			return nil, err
		}
		// 标记旧记录为已取代
		_ = c.factDAO.Supersede(existing.ID, newFact.ID)
		// 删除旧向量
		if c.milvus != nil && existing.MilvusID > 0 {
			_ = c.milvus.DeleteByFactIDs(context.Background(), []int64{int64(existing.ID)})
		}
		return newFact, nil
	}

	// 全新事实
	log.Printf("[fact-collector] 新增事实: type=%s title=%q content=%q", ef.Type, ef.Title, ef.Content)
	newFact := &model.NovelMemoryFact{
		NovelID:    novelID,
		UserID:     userID,
		ChapterID:  chapterID,
		FactType:   ef.Type,
		Title:      ef.Title,
		Content:    ef.Content,
		SourceText: ef.Source,
		Version:    1,
	}
	if err := c.factDAO.Create(newFact); err != nil {
		return nil, err
	}
	return newFact, nil
}

// EmbedAndStore 批量生成 Embedding 并存入 Milvus
// 分批调用（每批最多 10 条），qwen 失败时降级到 zhipu
func (c *FactCollector) EmbedAndStore(ctx context.Context, facts []*model.NovelMemoryFact) {
	if c.milvus == nil {
		log.Printf("[fact-collector] Milvus 未启用，跳过向量存储 (%d 条事实仅存 MySQL)", len(facts))
		return
	}

	log.Printf("[fact-collector] 开始 Embedding + Milvus 存储, %d 条事实", len(facts))

	// 收集需要 embedding 的文本
	texts := make([]string, len(facts))
	for i, f := range facts {
		texts[i] = f.Title + "：" + f.Content
	}

	// 分批调用 Embedding（每批最多 10 条，qwen API 限制）
	const batchSize = 10
	allVectors := make([][]float64, 0, len(texts))

	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[start:end]
		log.Printf("[fact-collector] Embedding 批次 %d-%d/%d", start+1, end, len(texts))

		vectors, err := c.embeddingWithFallback(ctx, batch)
		if err != nil {
			log.Printf("[fact-collector] Embedding 批次 %d-%d 全部失败: %v", start+1, end, err)
			return
		}
		allVectors = append(allVectors, vectors...)
	}

	if len(allVectors) != len(facts) {
		log.Printf("[fact-collector] Embedding 数量不匹配: 期望 %d, 实际 %d", len(facts), len(allVectors))
		return
	}

	// 构建 Milvus 插入数据
	var factVectors []vectordb.FactVector
	for i, f := range facts {
		vec32 := float64ToFloat32(allVectors[i])
		factVectors = append(factVectors, vectordb.FactVector{
			FactID:   int64(f.ID),
			NovelID:  int64(f.NovelID),
			FactType: f.FactType,
			Vector:   vec32,
		})
	}

	// 插入 Milvus
	milvusIDs, err := c.milvus.InsertFacts(ctx, factVectors)
	if err != nil {
		log.Printf("[fact-collector] Milvus 插入失败: %v", err)
		return
	}

	// 更新 MySQL 中的 milvus_id
	for i, mid := range milvusIDs {
		_ = c.factDAO.UpdateMilvusID(facts[i].ID, mid)
	}

	log.Printf("[fact-collector] 成功存入 %d 条向量到 Milvus", len(milvusIDs))
}

// embeddingWithFallback 调用 Embedding API，主力模型失败时降级到备选
func (c *FactCollector) embeddingWithFallback(ctx context.Context, texts []string) ([][]float64, error) {
	var providers []string
	if c.modelRegistry != nil {
		defaultEmb := c.modelRegistry.GetDefaultModel(model.CapEmbedding)
		providers = append([]string{defaultEmb}, c.modelRegistry.GetFallbackProviders(0, defaultEmb, model.CapEmbedding)...)
	} else {
		providers = []string{"qwen", "zhipu"}
	}
	var lastErr error

	for _, providerName := range providers {
		provider, err := c.dispatcher.GetProviderWithKey(ctx, providerName)
		if err != nil {
			log.Printf("[fact-collector] 获取 %s Provider 失败: %v, 尝试下一个", providerName, err)
			lastErr = err
			continue
		}

		resp, err := provider.Embedding(ctx, &agent.EmbeddingRequest{Texts: texts})
		if err != nil {
			log.Printf("[fact-collector] %s Embedding 失败: %v, 尝试下一个", providerName, err)
			lastErr = err
			continue
		}

		if len(resp.Vectors) != len(texts) {
			lastErr = fmt.Errorf("%s Embedding 数量不匹配: 期望 %d, 实际 %d", providerName, len(texts), len(resp.Vectors))
			log.Printf("[fact-collector] %v, 尝试下一个", lastErr)
			continue
		}

		log.Printf("[fact-collector] %s Embedding 成功, %d 条向量", providerName, len(resp.Vectors))
		return resp.Vectors, nil
	}

	return nil, fmt.Errorf("所有 Embedding Provider 均失败: %w", lastErr)
}

// parseFactsJSON 解析 AI 返回的 JSON 事实数组
func parseFactsJSON(content string) ([]extractedFact, error) {
	// 尝试提取 JSON 数组（AI 可能返回 markdown 包裹的 JSON）
	content = strings.TrimSpace(content)
	idx := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if idx < 0 || end <= idx {
		// AI 未返回 JSON 数组，跳过而非报错
		log.Printf("[fact-collector] AI 未返回 JSON 数组，跳过。原文前200字: %s", content[:min(200, len(content))])
		return nil, nil
	}
	content = content[idx : end+1]

	var facts []extractedFact
	if err := json.Unmarshal([]byte(content), &facts); err != nil {
		return nil, fmt.Errorf("解析事实 JSON 失败: %w, 原文: %s", err, content[:min(200, len(content))])
	}
	return facts, nil
}

// isSimilarContent 简单判断两段内容是否语义相似（基于字符重叠率）
func isSimilarContent(a, b string) bool {
	if a == b {
		return true
	}
	runesA := []rune(a)
	runesB := []rune(b)
	if len(runesA) == 0 || len(runesB) == 0 {
		return false
	}

	// 计算字符级 Jaccard 相似度
	setA := make(map[rune]bool)
	for _, r := range runesA {
		setA[r] = true
	}
	setB := make(map[rune]bool)
	for _, r := range runesB {
		setB[r] = true
	}

	intersection := 0
	for r := range setA {
		if setB[r] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return true
	}

	similarity := float64(intersection) / float64(union)
	return similarity > 0.9
}

// float64ToFloat32 将 float64 切片转换为 float32
func float64ToFloat32(vec []float64) []float32 {
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = float32(v)
	}
	return result
}
