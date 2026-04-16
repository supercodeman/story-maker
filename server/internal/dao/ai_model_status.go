// server/internal/dao/ai_model_status.go
package dao

import (
	"context"
	"time"

	"ai-curton/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AIModelStatusDAO 模型状态数据访问层
type AIModelStatusDAO struct {
	db *gorm.DB
}

// NewAIModelStatusDAO 创建 AIModelStatusDAO 实例
func NewAIModelStatusDAO(db *gorm.DB) *AIModelStatusDAO {
	return &AIModelStatusDAO{db: db}
}

// Upsert 按 provider+model_name+capability 唯一键 upsert
func (d *AIModelStatusDAO) Upsert(ctx context.Context, status *model.AIModelStatus) error {
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "model_name"}, {Name: "capability"}},
			DoUpdates: clause.AssignmentColumns([]string{"is_available", "last_check", "last_error", "latency_ms", "updated_at"}),
		}).
		Create(status).Error
}

// GetByCapability 按能力查询所有模型状态
func (d *AIModelStatusDAO) GetByCapability(ctx context.Context, capability string) ([]*model.AIModelStatus, error) {
	var list []*model.AIModelStatus
	err := d.db.WithContext(ctx).
		Where("capability = ?", capability).
		Order("provider ASC, model_name ASC").
		Find(&list).Error
	return list, err
}

// GetByProvider 按 Provider 查询所有模型状态
func (d *AIModelStatusDAO) GetByProvider(ctx context.Context, provider string) ([]*model.AIModelStatus, error) {
	var list []*model.AIModelStatus
	err := d.db.WithContext(ctx).
		Where("provider = ?", provider).
		Find(&list).Error
	return list, err
}

// GetAll 查询所有模型状态
func (d *AIModelStatusDAO) GetAll(ctx context.Context) ([]*model.AIModelStatus, error) {
	var list []*model.AIModelStatus
	err := d.db.WithContext(ctx).
		Order("provider ASC, model_name ASC, capability ASC").
		Find(&list).Error
	return list, err
}

// Delete 按 ID 删除模型状态记录
func (d *AIModelStatusDAO) Delete(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.AIModelStatus{}, id).Error
}

// UpdateAvailability 更新模型可用性
func (d *AIModelStatusDAO) UpdateAvailability(ctx context.Context, provider, modelName, capability string, available bool, latencyMs int, lastError string) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.AIModelStatus{}).
		Where("provider = ? AND model_name = ? AND capability = ?", provider, modelName, capability).
		Updates(map[string]interface{}{
			"is_available": available,
			"latency_ms":   latencyMs,
			"last_error":   lastError,
			"last_check":   &now,
		}).Error
}
