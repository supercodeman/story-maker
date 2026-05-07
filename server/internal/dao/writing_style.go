// server/internal/dao/writing_style.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WritingStyleDAO 写作风格数据访问层
type WritingStyleDAO struct {
	db *gorm.DB
}

// NewWritingStyleDAO 创建 WritingStyleDAO 实例
func NewWritingStyleDAO() *WritingStyleDAO {
	return &WritingStyleDAO{db: model.DB}
}

// ========== WritingStyle CRUD ==========

// GetByNovelID 根据小说 ID 获取写作风格配置
func (d *WritingStyleDAO) GetByNovelID(novelID uint) (*model.WritingStyle, error) {
	var style model.WritingStyle
	err := d.db.Where("novel_id = ?", novelID).First(&style).Error
	if err != nil {
		return nil, err
	}
	return &style, nil
}

// Upsert 创建或更新写作风格（基于 novel_id 唯一索引）
func (d *WritingStyleDAO) Upsert(style *model.WritingStyle) error {
	return d.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "novel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"narrative_voice", "tone", "language_level", "reference_authors", "forbidden_patterns", "custom_rules", "updated_at"}),
	}).Create(style).Error
}

// Delete 删除写作风格配置
func (d *WritingStyleDAO) Delete(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.WritingStyle{}).Error
}

// ========== ScenePreset CRUD ==========

// ListPresets 获取小说下的所有场景预设
func (d *WritingStyleDAO) ListPresets(novelID uint) ([]model.ScenePreset, error) {
	var presets []model.ScenePreset
	err := d.db.Where("novel_id = ?", novelID).
		Order("created_at ASC").
		Find(&presets).Error
	return presets, err
}

// GetPreset 根据 ID 获取场景预设
func (d *WritingStyleDAO) GetPreset(id uint) (*model.ScenePreset, error) {
	var preset model.ScenePreset
	err := d.db.First(&preset, id).Error
	if err != nil {
		return nil, err
	}
	return &preset, nil
}

// CreatePreset 创建场景预设
func (d *WritingStyleDAO) CreatePreset(preset *model.ScenePreset) error {
	return d.db.Create(preset).Error
}

// UpdatePreset 更新场景预设
func (d *WritingStyleDAO) UpdatePreset(preset *model.ScenePreset) error {
	return d.db.Save(preset).Error
}

// DeletePreset 删除场景预设
func (d *WritingStyleDAO) DeletePreset(id uint) error {
	return d.db.Delete(&model.ScenePreset{}, id).Error
}

// UpdateBoundUserStyleID 更新小说写作风格的绑定用户风格ID
func (d *WritingStyleDAO) UpdateBoundUserStyleID(novelID uint, userStyleID *uint) error {
	return d.db.Model(&model.WritingStyle{}).
		Where("novel_id = ?", novelID).
		Update("bound_user_style_id", userStyleID).Error
}
