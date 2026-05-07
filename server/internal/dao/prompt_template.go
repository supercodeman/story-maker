// server/internal/dao/prompt_template.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PromptTemplateDAO Prompt 模板数据访问层
type PromptTemplateDAO struct {
	db *gorm.DB
}

// NewPromptTemplateDAO 创建 PromptTemplateDAO 实例
func NewPromptTemplateDAO() *PromptTemplateDAO {
	return &PromptTemplateDAO{db: model.DB}
}

// GetTemplate 获取模板：优先小说级，fallback 到系统默认（novel_id=0）
func (d *PromptTemplateDAO) GetTemplate(novelID uint, action, promptType string) (*model.PromptTemplate, error) {
	var tpl model.PromptTemplate
	// 先查小说级自定义
	err := d.db.Where("novel_id = ? AND action = ? AND prompt_type = ?", novelID, action, promptType).
		First(&tpl).Error
	if err == nil {
		return &tpl, nil
	}
	// fallback 到系统默认
	err = d.db.Where("novel_id = 0 AND action = ? AND prompt_type = ?", action, promptType).
		First(&tpl).Error
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}

// GetDefault 获取系统默认模板（novel_id=0）
func (d *PromptTemplateDAO) GetDefault(action, promptType string) (*model.PromptTemplate, error) {
	var tpl model.PromptTemplate
	err := d.db.Where("novel_id = 0 AND action = ? AND prompt_type = ?", action, promptType).
		First(&tpl).Error
	return &tpl, err
}

// GetCustom 获取小说级自定义模板
func (d *PromptTemplateDAO) GetCustom(novelID uint, action, promptType string) (*model.PromptTemplate, error) {
	var tpl model.PromptTemplate
	err := d.db.Where("novel_id = ? AND action = ? AND prompt_type = ?", novelID, action, promptType).
		First(&tpl).Error
	return &tpl, err
}

// ListByNovel 列出小说的所有自定义模板
func (d *PromptTemplateDAO) ListByNovel(novelID uint) ([]model.PromptTemplate, error) {
	var templates []model.PromptTemplate
	err := d.db.Where("novel_id = ?", novelID).
		Order("action ASC, prompt_type ASC").
		Find(&templates).Error
	return templates, err
}

// ListDefaults 列出所有系统默认模板
func (d *PromptTemplateDAO) ListDefaults() ([]model.PromptTemplate, error) {
	var templates []model.PromptTemplate
	err := d.db.Where("novel_id = 0 AND is_default = ?", true).
		Order("action ASC, prompt_type ASC").
		Find(&templates).Error
	return templates, err
}

// Upsert 创建或更新模板（基于 novel_id + action + prompt_type 唯一索引）
func (d *PromptTemplateDAO) Upsert(tpl *model.PromptTemplate) error {
	return d.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "novel_id"}, {Name: "action"}, {Name: "prompt_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "content", "updated_at"}),
	}).Create(tpl).Error
}

// Delete 删除模板（不允许删除 is_default=true 的系统默认模板）
func (d *PromptTemplateDAO) Delete(id uint) error {
	return d.db.Where("id = ? AND is_default = ?", id, false).Delete(&model.PromptTemplate{}).Error
}

// SeedDefaults 初始化默认模板（不存在则插入，已存在则更新 content）
func (d *PromptTemplateDAO) SeedDefaults(templates []model.PromptTemplate) error {
	for i := range templates {
		templates[i].IsDefault = true
		templates[i].NovelID = 0

		var existing model.PromptTemplate
		err := d.db.Where("novel_id = 0 AND action = ? AND prompt_type = ?", templates[i].Action, templates[i].PromptType).
			First(&existing).Error
		if err != nil {
			// 不存在，插入
			if err := d.db.Create(&templates[i]).Error; err != nil {
				return err
			}
		} else {
			// 已存在，更新 content 和 name
			if err := d.db.Model(&existing).Updates(map[string]interface{}{
				"content": templates[i].Content,
				"name":    templates[i].Name,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
