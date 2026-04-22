// server/internal/service/fact_retriever.go
package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
	"ai-curton/server/internal/vectordb"
)

// FactRetriever 事实召回器，从 Milvus 检索相关事实并格式化注入 prompt
type FactRetriever struct {
	factDAO       *dao.NovelFactDAO
	novelDAO      *dao.NovelDAO
	milvus        *vectordb.MilvusClient
	dispatcher    *agent.Dispatcher
	modelRegistry *ModelRegistryService
}

// NewFactRetriever 创建事实召回器
func NewFactRetriever(factDAO *dao.NovelFactDAO, novelDAO *dao.NovelDAO, milvus *vectordb.MilvusClient, dispatcher *agent.Dispatcher) *FactRetriever {
	return &FactRetriever{
		factDAO:    factDAO,
		novelDAO:   novelDAO,
		milvus:     milvus,
		dispatcher: dispatcher,
	}
}

// SetModelRegistry 注入模型注册服务
func (r *FactRetriever) SetModelRegistry(mr *ModelRegistryService) {
	r.modelRegistry = mr
}

// Retrieve 检索与当前章节相关的动态记忆，返回格式化文本
func (r *FactRetriever) Retrieve(ctx context.Context, novelID uint, chapter *model.Chapter) string {
	if r.milvus == nil {
		log.Printf("[fact-retriever] Milvus 未启用，跳过召回")
		return ""
	}

	log.Printf("[fact-retriever] 开始召回 novel_id=%d chapter_id=%d(%s)", novelID, chapter.ID, chapter.Title)

	// 1. 构建查询文本
	queryText := r.buildQueryText(chapter)
	if queryText == "" {
		log.Printf("[fact-retriever] 查询文本为空，跳过召回")
		return ""
	}
	log.Printf("[fact-retriever] 查询文本: %s", truncateStr(queryText, 100))

	// 2. 生成查询向量
	queryVec, err := r.getQueryEmbedding(ctx, queryText)
	if err != nil {
		log.Printf("[fact-retriever] 生成查询向量失败: %v", err)
		return ""
	}
	log.Printf("[fact-retriever] 查询向量生成成功, dim=%d", len(queryVec))

	// 3. Milvus 向量检索
	results, err := r.milvus.SearchByNovel(ctx, int64(novelID), queryVec, 20, nil)
	if err != nil {
		log.Printf("[fact-retriever] Milvus 检索失败: %v", err)
		return ""
	}
	if len(results) == 0 {
		log.Printf("[fact-retriever] Milvus 检索结果为空")
		return ""
	}
	log.Printf("[fact-retriever] Milvus 返回 %d 条候选", len(results))

	// 4. 回查 MySQL 获取完整事实，过滤已取代的记录
	factIDs := make([]uint, 0, len(results))
	for _, sr := range results {
		factIDs = append(factIDs, uint(sr.FactID))
	}

	facts, err := r.factDAO.GetByIDs(factIDs)
	if err != nil {
		log.Printf("[fact-retriever] 查询事实详情失败: %v", err)
		return ""
	}
	log.Printf("[fact-retriever] MySQL 回查到 %d 条事实", len(facts))

	// 过滤已取代的事实
	var activeFacts []model.NovelMemoryFact
	for _, f := range facts {
		if !f.IsSuperseded {
			activeFacts = append(activeFacts, f)
		}
	}

	if len(activeFacts) == 0 {
		log.Printf("[fact-retriever] 过滤后无有效事实")
		return ""
	}
	log.Printf("[fact-retriever] 过滤后 %d 条有效事实 (去除 %d 条已取代)", len(activeFacts), len(facts)-len(activeFacts))

	// 5. 按 fact_type 分组格式化
	formatted := r.formatFacts(activeFacts)
	log.Printf("[fact-retriever] 小说 %d 召回 %d 条动态记忆, 格式化长度=%d", novelID, len(activeFacts), len([]rune(formatted)))
	return formatted
}

// buildQueryText 构建检索查询文本
func (r *FactRetriever) buildQueryText(chapter *model.Chapter) string {
	parts := []string{}

	if chapter.Title != "" {
		parts = append(parts, chapter.Title)
	}
	if chapter.Summary != "" {
		parts = append(parts, chapter.Summary)
	}

	// 取正文前 200 字作为补充上下文
	if chapter.Content != "" {
		runes := []rune(chapter.Content)
		if len(runes) > 200 {
			parts = append(parts, string(runes[:200]))
		} else {
			parts = append(parts, chapter.Content)
		}
	}

	return strings.Join(parts, " ")
}

// getQueryEmbedding 生成查询文本的 Embedding 向量（含降级链）
func (r *FactRetriever) getQueryEmbedding(ctx context.Context, text string) ([]float32, error) {
	// 构建 Embedding 降级链
	var providers []string
	if r.modelRegistry != nil {
		defaultEmb := r.modelRegistry.GetDefaultModel(model.CapEmbedding)
		providers = append([]string{defaultEmb}, r.modelRegistry.GetFallbackProviders(0, defaultEmb, model.CapEmbedding)...)
	} else {
		providers = []string{"qwen", "zhipu"}
	}

	var lastErr error
	for _, providerName := range providers {
		provider, err := r.dispatcher.GetProviderWithKey(ctx, providerName)
		if err != nil {
			log.Printf("[fact-retriever] 获取 %s Provider 失败: %v", providerName, err)
			lastErr = err
			continue
		}

		resp, err := provider.Embedding(ctx, &agent.EmbeddingRequest{
			Texts: []string{text},
		})
		if err != nil {
			log.Printf("[fact-retriever] %s Embedding 失败: %v", providerName, err)
			lastErr = err
			continue
		}

		if len(resp.Vectors) == 0 {
			lastErr = fmt.Errorf("Embedding 返回空向量")
			continue
		}

		return float64ToFloat32(resp.Vectors[0]), nil
	}

	return nil, fmt.Errorf("Embedding 所有 Provider 均失败: %w", lastErr)
}

// formatFacts 按事实类型分组格式化为 prompt 注入文本
func (r *FactRetriever) formatFacts(facts []model.NovelMemoryFact) string {
	// 按 fact_type 分组
	grouped := make(map[string][]model.NovelMemoryFact)
	for _, f := range facts {
		grouped[f.FactType] = append(grouped[f.FactType], f)
	}

	// 按固定顺序输出
	typeOrder := []string{
		model.FactTypeCharacterTrait,
		model.FactTypeRelationshipChange,
		model.FactTypeForeshadow,
		model.FactTypePlotEvent,
		model.FactTypeWorldviewRule,
	}

	var sections []string
	for _, ft := range typeOrder {
		items, ok := grouped[ft]
		if !ok || len(items) == 0 {
			continue
		}

		label := model.FactTypeLabel[ft]
		var lines []string
		for _, item := range items {
			line := fmt.Sprintf("- %s：%s", item.Title, item.Content)
			if item.ChapterID > 0 {
				line += fmt.Sprintf("（来源章节ID:%d）", item.ChapterID)
			}
			lines = append(lines, line)
		}

		section := fmt.Sprintf("【动态记忆 - %s】\n%s", label, strings.Join(lines, "\n"))
		sections = append(sections, section)
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n")
}

// truncateStr 截断字符串用于日志输出
func truncateStr(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
