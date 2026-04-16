// server/internal/dao/writing_memory.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// WritingMemoryDAO 写作记忆数据访问层
type WritingMemoryDAO struct {
	db *gorm.DB
}

// NewWritingMemoryDAO 创建 WritingMemoryDAO 实例
func NewWritingMemoryDAO() *WritingMemoryDAO {
	return &WritingMemoryDAO{db: model.DB}
}

// ========== WritingMemory CRUD ==========

// Create 创建记忆
func (d *WritingMemoryDAO) Create(memory *model.WritingMemory) error {
	return d.db.Create(memory).Error
}

// Get 根据 ID 获取记忆
func (d *WritingMemoryDAO) Get(id uint) (*model.WritingMemory, error) {
	var memory model.WritingMemory
	err := d.db.First(&memory, id).Error
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// Update 更新记忆
func (d *WritingMemoryDAO) Update(memory *model.WritingMemory) error {
	return d.db.Save(memory).Error
}

// Delete 删除记忆及关联数据
func (d *WritingMemoryDAO) Delete(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("memory_id = ?", id).Delete(&model.WritingMemoryVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Where("memory_id = ?", id).Delete(&model.MemoryEmbedding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("memory_id = ?", id).Delete(&model.NovelMemoryBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("memory_id = ?", id).Delete(&model.MemoryGenre{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.WritingMemory{}, id).Error
	})
}

// ListByUser 获取用户的记忆列表
func (d *WritingMemoryDAO) ListByUser(userID uint, category string) ([]model.WritingMemory, error) {
	var memories []model.WritingMemory
	q := d.db.Where("user_id = ?", userID)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	err := q.Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetBySampleHash 根据样本哈希查找（防重复）
func (d *WritingMemoryDAO) GetBySampleHash(hash string) (*model.WritingMemory, error) {
	var memory model.WritingMemory
	err := d.db.Where("sample_hash = ?", hash).First(&memory).Error
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// UpdateStatus 更新记忆状态
func (d *WritingMemoryDAO) UpdateStatus(id uint, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == model.MemoryStatusPublished {
		updates["is_public"] = true
	}
	if status == model.MemoryStatusArchived {
		updates["is_public"] = false
	}
	return d.db.Model(&model.WritingMemory{}).Where("id = ?", id).Updates(updates).Error
}

// IncrSalesCount 增加销量
func (d *WritingMemoryDAO) IncrSalesCount(id uint) error {
	return d.db.Model(&model.WritingMemory{}).Where("id = ?", id).
		UpdateColumn("sales_count", gorm.Expr("sales_count + 1")).Error
}

// UpdateRating 更新评分
func (d *WritingMemoryDAO) UpdateRating(id uint, avgRating float64, ratingCount int) error {
	return d.db.Model(&model.WritingMemory{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"avg_rating":   avgRating,
			"rating_count": ratingCount,
		}).Error
}

// ========== WritingMemoryVersion ==========

// CreateVersion 创建版本记录
func (d *WritingMemoryDAO) CreateVersion(version *model.WritingMemoryVersion) error {
	return d.db.Create(version).Error
}

// ListVersions 获取记忆的版本列表
func (d *WritingMemoryDAO) ListVersions(memoryID uint) ([]model.WritingMemoryVersion, error) {
	var versions []model.WritingMemoryVersion
	err := d.db.Where("memory_id = ?", memoryID).Order("version DESC").Find(&versions).Error
	return versions, err
}

// ========== MemoryEmbedding ==========

// BatchCreateEmbeddings 批量创建 Embedding
func (d *WritingMemoryDAO) BatchCreateEmbeddings(embeddings []model.MemoryEmbedding) error {
	return d.db.CreateInBatches(embeddings, 100).Error
}

// DeleteEmbeddings 删除记忆的所有 Embedding
func (d *WritingMemoryDAO) DeleteEmbeddings(memoryID uint) error {
	return d.db.Where("memory_id = ?", memoryID).Delete(&model.MemoryEmbedding{}).Error
}

// ListEmbeddings 获取记忆的所有 Embedding
func (d *WritingMemoryDAO) ListEmbeddings(memoryID uint) ([]model.MemoryEmbedding, error) {
	var embeddings []model.MemoryEmbedding
	err := d.db.Where("memory_id = ?", memoryID).Order("chunk_idx ASC").Find(&embeddings).Error
	return embeddings, err
}

// ========== NovelMemoryBinding ==========

// UpsertBinding 创建或更新绑定（每个小说每个类别最多一个）
func (d *WritingMemoryDAO) UpsertBinding(binding *model.NovelMemoryBinding) error {
	// 先删除同小说同类别的旧绑定
	d.db.Where("novel_id = ? AND category = ?", binding.NovelID, binding.Category).
		Delete(&model.NovelMemoryBinding{})
	return d.db.Create(binding).Error
}

// DeleteBinding 删除绑定
func (d *WritingMemoryDAO) DeleteBinding(novelID uint, category string) error {
	return d.db.Where("novel_id = ? AND category = ?", novelID, category).
		Delete(&model.NovelMemoryBinding{}).Error
}

// ListBindingsByNovel 获取小说的所有记忆绑定
func (d *WritingMemoryDAO) ListBindingsByNovel(novelID uint) ([]model.NovelMemoryBinding, error) {
	var bindings []model.NovelMemoryBinding
	err := d.db.Where("novel_id = ?", novelID).Find(&bindings).Error
	return bindings, err
}

// ========== 市场查询 ==========

// ListPublished 获取已上架的记忆列表（市场浏览，支持赛道筛选）
func (d *WritingMemoryDAO) ListPublished(category string, keyword string, orderBy string, page, pageSize int, memoryIDs []uint) ([]model.WritingMemory, int64, error) {
	var memories []model.WritingMemory
	var total int64

	q := d.db.Where("is_public = ? AND status = ?", true, model.MemoryStatusPublished)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if keyword != "" {
		q = q.Where("title LIKE ? OR tags LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	// 赛道筛选
	if len(memoryIDs) > 0 {
		q = q.Where("id IN ?", memoryIDs)
	}

	if err := q.Model(&model.WritingMemory{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序白名单
	switch orderBy {
	case "sales":
		q = q.Order("sales_count DESC")
	case "rating":
		q = q.Order("avg_rating DESC")
	case "newest":
		q = q.Order("created_at DESC")
	default:
		q = q.Order("sales_count DESC")
	}

	offset := (page - 1) * pageSize
	err := q.Offset(offset).Limit(pageSize).Find(&memories).Error
	return memories, total, err
}

// ListAccessible 获取用户可用的记忆（自己的 + 已购买的）
func (d *WritingMemoryDAO) ListAccessible(userID uint, category string) ([]model.WritingMemory, error) {
	var memories []model.WritingMemory
	q := d.db.Where("user_id = ?", userID)
	if category != "" {
		q = q.Where("category = ?", category)
	}

	// 自己的记忆
	if err := q.Find(&memories).Error; err != nil {
		return nil, err
	}

	// 已购买的记忆
	var licensedIDs []uint
	d.db.Model(&model.MemoryLicense{}).Where("user_id = ?", userID).Pluck("memory_id", &licensedIDs)
	if len(licensedIDs) > 0 {
		var licensed []model.WritingMemory
		lq := d.db.Where("id IN ?", licensedIDs)
		if category != "" {
			lq = lq.Where("category = ?", category)
		}
		if err := lq.Find(&licensed).Error; err == nil {
			memories = append(memories, licensed...)
		}
	}

	return memories, nil
}

// UpdateExtractStatus 更新提取状态
func (d *WritingMemoryDAO) UpdateExtractStatus(id uint, status, errMsg string, workflowID uint) error {
	updates := map[string]interface{}{
		"extract_status": status,
		"extract_error":  errMsg,
	}
	if workflowID > 0 {
		updates["extract_workflow_id"] = workflowID
	}
	return d.db.Model(&model.WritingMemory{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateAfterExtract 提取完成后更新记忆全部字段
func (d *WritingMemoryDAO) UpdateAfterExtract(memory *model.WritingMemory) error {
	return d.db.Model(memory).Updates(map[string]interface{}{
		"features":            memory.Features,
		"prompt_tpl":          memory.PromptTpl,
		"anchor_texts":        memory.AnchorTexts,
		"preview_text":        memory.PreviewText,
		"quality":             memory.Quality,
		"quality_detail":      memory.QualityDetail,
		"quality_grade":       memory.QualityGrade,
		"extract_status":      memory.ExtractStatus,
		"extract_error":       memory.ExtractError,
	}).Error
}

// ListByStatus 按状态查询记忆列表（管理员用）
func (d *WritingMemoryDAO) ListByStatus(status string) ([]model.WritingMemory, error) {
	var memories []model.WritingMemory
	err := d.db.Where("status = ?", status).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// ListByExtractStatus 按提取状态查询记忆
func (d *WritingMemoryDAO) ListByExtractStatus(status string) ([]model.WritingMemory, error) {
	var memories []model.WritingMemory
	err := d.db.Where("extract_status = ?", status).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// UpdateReviewResult 更新审核结果
func (d *WritingMemoryDAO) UpdateReviewResult(id uint, status, reason string, wfID uint) error {
	updates := map[string]interface{}{
		"status":        status,
		"review_reason": reason,
	}
	if wfID > 0 {
		updates["review_workflow_id"] = wfID
	}
	if status == model.MemoryStatusPublished {
		updates["is_public"] = true
	}
	return d.db.Model(&model.WritingMemory{}).Where("id = ?", id).Updates(updates).Error
}
