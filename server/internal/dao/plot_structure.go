// server/internal/dao/plot_structure.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// PlotStructureDAO 剧情结构模板数据访问层
type PlotStructureDAO struct {
	db *gorm.DB
}

// NewPlotStructureDAO 创建 PlotStructureDAO 实例
func NewPlotStructureDAO() *PlotStructureDAO {
	return &PlotStructureDAO{db: model.DB}
}

// Create 创建模板
func (d *PlotStructureDAO) Create(tpl *model.PlotStructureTemplate) error {
	return d.db.Create(tpl).Error
}

// Get 根据 ID 获取模板
func (d *PlotStructureDAO) Get(id uint) (*model.PlotStructureTemplate, error) {
	var tpl model.PlotStructureTemplate
	err := d.db.First(&tpl, id).Error
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}

// Update 更新模板
func (d *PlotStructureDAO) Update(tpl *model.PlotStructureTemplate) error {
	return d.db.Save(tpl).Error
}

// Delete 删除模板
func (d *PlotStructureDAO) Delete(id uint) error {
	return d.db.Delete(&model.PlotStructureTemplate{}, id).Error
}

// List 获取系统模板 + 指定用户的自定义模板
func (d *PlotStructureDAO) List(userID uint) ([]model.PlotStructureTemplate, error) {
	var templates []model.PlotStructureTemplate
	err := d.db.Where("is_system = ? OR user_id = ?", true, userID).
		Order("is_system DESC, id ASC").
		Find(&templates).Error
	return templates, err
}

// FirstOrCreateByName 按名称查找或创建（用于种子数据）
func (d *PlotStructureDAO) FirstOrCreateByName(tpl *model.PlotStructureTemplate) error {
	return d.db.Where("name = ? AND is_system = ?", tpl.Name, true).
		FirstOrCreate(tpl).Error
}
