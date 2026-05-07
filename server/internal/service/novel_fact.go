// server/internal/service/novel_fact.go
package service

import (
	"context"
	"fmt"
	"log"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
	"story-maker/server/internal/vectordb"
)

// NovelFactService 小说动态记忆事实服务（统一门面）
type NovelFactService struct {
	factDAO   *dao.NovelFactDAO
	novelDAO  *dao.NovelDAO
	collector *FactCollector
	retriever *FactRetriever
}

// NewNovelFactService 创建 NovelFactService 实例
func NewNovelFactService(milvus *vectordb.MilvusClient, dispatcher *agent.Dispatcher) *NovelFactService {
	factDAO := dao.NewNovelFactDAO()
	novelDAO := dao.NewNovelDAO()

	var collector *FactCollector
	var retriever *FactRetriever

	if milvus != nil {
		collector = NewFactCollector(factDAO, novelDAO, milvus, dispatcher)
		retriever = NewFactRetriever(factDAO, novelDAO, milvus, dispatcher)
	} else {
		log.Println("[novel-fact] Milvus 未启用，动态记忆功能降级为仅 MySQL 存储")
		collector = NewFactCollector(factDAO, novelDAO, nil, dispatcher)
	}

	return &NovelFactService{
		factDAO:   factDAO,
		novelDAO:  novelDAO,
		collector: collector,
		retriever: retriever,
	}
}

// SetModelRegistry 注入模型注册中心（传递给子组件）
func (s *NovelFactService) SetModelRegistry(mr *ModelRegistryService) {
	if s.collector != nil {
		s.collector.SetModelRegistry(mr)
	}
	if s.retriever != nil {
		s.retriever.SetModelRegistry(mr)
	}
}

// CollectFromChapter 异步从章节采集事实（供 UpdateChapter 调用）
func (s *NovelFactService) CollectFromChapter(ctx context.Context, novel *model.Novel, chapter *model.Chapter, userID uint) {
	if s.collector == nil {
		return
	}

	log.Printf("[novel-fact] === 采集链路开始 === novel_id=%d chapter_id=%d(%s) user_id=%d", novel.ID, chapter.ID, chapter.Title, userID)

	// 检查是否需要冷启动
	count, err := s.factDAO.CountByNovel(novel.ID)
	if err == nil && count == 0 {
		log.Printf("[novel-fact] 小说 %d 无事实记录，触发冷启动", novel.ID)
		if err := s.collector.ColdStart(ctx, novel, userID); err != nil {
			log.Printf("[novel-fact] 冷启动失败: %v", err)
		}
	} else {
		log.Printf("[novel-fact] 小说 %d 已有 %d 条事实，跳过冷启动", novel.ID, count)
	}

	// 采集当前章节
	s.collector.CollectFromChapter(ctx, novel, chapter, userID)
	log.Printf("[novel-fact] === 采集链路结束 === novel_id=%d chapter_id=%d", novel.ID, chapter.ID)
}

// FullColdStart 全量冷启动：清除旧数据后对所有章节做全量事实采集
func (s *NovelFactService) FullColdStart(ctx context.Context, novelID, userID uint) error {
	if s.collector == nil {
		return fmt.Errorf("fact collector 未初始化")
	}

	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return fmt.Errorf("查询小说失败: %w", err)
	}

	return s.collector.FullColdStart(ctx, novel, userID)
}

// Retrieve 检索与当前章节相关的动态记忆（供 buildTemplateData 调用）
func (s *NovelFactService) Retrieve(ctx context.Context, novelID uint, chapter *model.Chapter) string {
	if s.retriever == nil {
		return ""
	}
	log.Printf("[novel-fact] === 召回链路开始 === novel_id=%d chapter_id=%d(%s)", novelID, chapter.ID, chapter.Title)
	result := s.retriever.Retrieve(ctx, novelID, chapter)
	log.Printf("[novel-fact] === 召回链路结束 === novel_id=%d 召回内容长度=%d", novelID, len([]rune(result)))
	return result
}

// ========== 用户 CRUD 接口 ==========

// CreateFactRequest 用户手动创建事实请求
type CreateFactRequest struct {
	FactType string `json:"fact_type" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

// UpdateFactRequest 用户更新事实请求（部分更新）
type UpdateFactRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

// ListFacts 获取小说未被取代的事实列表，支持按 fact_type 过滤
func (s *NovelFactService) ListFacts(novelID uint, factType string) ([]model.NovelMemoryFact, error) {
	return s.factDAO.ListByNovelActiveWithFilter(novelID, factType)
}

// GetFact 按 ID 获取单条事实详情
func (s *NovelFactService) GetFact(factID uint) (*model.NovelMemoryFact, error) {
	return s.factDAO.GetByID(factID)
}

// CreateFact 用户手动创建事实，ChapterID=0 表示非章节来源
func (s *NovelFactService) CreateFact(ctx context.Context, novelID, userID uint, req *CreateFactRequest) (*model.NovelMemoryFact, error) {
	// 校验 fact_type 白名单
	if !model.ValidFactTypes[req.FactType] {
		return nil, fmt.Errorf("invalid fact_type: %s", req.FactType)
	}

	fact := &model.NovelMemoryFact{
		NovelID:   novelID,
		UserID:    userID,
		ChapterID: 0, // 用户手动创建，非章节来源
		FactType:  req.FactType,
		Title:     req.Title,
		Content:   req.Content,
		Version:   1,
	}
	if err := s.factDAO.Create(fact); err != nil {
		return nil, fmt.Errorf("create fact failed: %w", err)
	}

	// 生成向量并存入 Milvus
	if s.collector != nil {
		s.collector.EmbedAndStore(ctx, []*model.NovelMemoryFact{fact})
	}

	return fact, nil
}

// UpdateFact 更新事实：创建新版本 + Supersede 旧版本 + 删除旧向量 + 生成新向量
func (s *NovelFactService) UpdateFact(ctx context.Context, factID, userID uint, req *UpdateFactRequest) (*model.NovelMemoryFact, error) {
	old, err := s.factDAO.GetByID(factID)
	if err != nil {
		return nil, fmt.Errorf("fact not found: %w", err)
	}
	if old.IsSuperseded {
		return nil, fmt.Errorf("fact already superseded")
	}

	// 合并更新字段
	newTitle := old.Title
	newContent := old.Content
	if req.Title != nil {
		newTitle = *req.Title
	}
	if req.Content != nil {
		newContent = *req.Content
	}

	// 创建新版本
	newFact := &model.NovelMemoryFact{
		NovelID:   old.NovelID,
		UserID:    userID,
		ChapterID: old.ChapterID,
		FactType:  old.FactType,
		Title:     newTitle,
		Content:   newContent,
		Version:   old.Version + 1,
	}
	if err := s.factDAO.Create(newFact); err != nil {
		return nil, fmt.Errorf("create new version failed: %w", err)
	}

	// Supersede 旧版本
	_ = s.factDAO.Supersede(old.ID, newFact.ID)

	// 删除旧向量 + 生成新向量
	if s.collector != nil {
		if old.MilvusID > 0 {
			_ = s.collector.milvus.DeleteByFactIDs(ctx, []int64{int64(old.ID)})
		}
		s.collector.EmbedAndStore(ctx, []*model.NovelMemoryFact{newFact})
	}

	return newFact, nil
}

// DeleteFact 删除事实：Supersede 标记 + 删除 Milvus 向量
func (s *NovelFactService) DeleteFact(ctx context.Context, factID uint) error {
	fact, err := s.factDAO.GetByID(factID)
	if err != nil {
		return fmt.Errorf("fact not found: %w", err)
	}
	if fact.IsSuperseded {
		return fmt.Errorf("fact already superseded")
	}

	// Supersede 标记（superseded_by=0 表示被用户删除）
	_ = s.factDAO.Supersede(fact.ID, 0)

	// 删除 Milvus 向量
	if s.collector != nil && s.collector.milvus != nil && fact.MilvusID > 0 {
		_ = s.collector.milvus.DeleteByFactIDs(ctx, []int64{int64(fact.ID)})
	}

	return nil
}
