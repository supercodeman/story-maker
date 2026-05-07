// server/internal/dao/portfolio.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// PortfolioDAO 作品集数据访问层
type PortfolioDAO struct {
	db *gorm.DB
}

// NewPortfolioDAO 创建 PortfolioDAO 实例
func NewPortfolioDAO() *PortfolioDAO {
	return &PortfolioDAO{db: model.DB}
}

// Create 创建作品集
func (d *PortfolioDAO) Create(p *model.Portfolio) error {
	return d.db.Create(p).Error
}

// GetByID 根据 ID 获取作品集
func (d *PortfolioDAO) GetByID(id uint) (*model.Portfolio, error) {
	var p model.Portfolio
	err := d.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListByWorkspaceID 获取工作空间下的所有作品集
func (d *PortfolioDAO) ListByWorkspaceID(workspaceID uint) ([]model.Portfolio, error) {
	var portfolios []model.Portfolio
	err := d.db.Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&portfolios).Error
	return portfolios, err
}

// Update 更新作品集
func (d *PortfolioDAO) Update(p *model.Portfolio) error {
	return d.db.Save(p).Error
}

// Delete 删除作品集
func (d *PortfolioDAO) Delete(id uint) error {
	return d.db.Delete(&model.Portfolio{}, id).Error
}
