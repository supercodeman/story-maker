// server/internal/dao/character.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// CharacterDAO 角色数据访问层
type CharacterDAO struct {
	db *gorm.DB
}

// NewCharacterDAO 创建 CharacterDAO 实例
func NewCharacterDAO() *CharacterDAO {
	return &CharacterDAO{db: model.DB}
}

// Create 创建角色
func (d *CharacterDAO) Create(ch *model.Character) error {
	return d.db.Create(ch).Error
}

// GetByID 根据 ID 获取角色
func (d *CharacterDAO) GetByID(id uint) (*model.Character, error) {
	var ch model.Character
	err := d.db.First(&ch, id).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// ListByPortfolioID 获取作品集下的所有角色
func (d *CharacterDAO) ListByPortfolioID(portfolioID uint) ([]model.Character, error) {
	var characters []model.Character
	err := d.db.Where("portfolio_id = ?", portfolioID).
		Order("created_at DESC").
		Find(&characters).Error
	return characters, err
}

// Update 更新角色
func (d *CharacterDAO) Update(ch *model.Character) error {
	return d.db.Save(ch).Error
}

// Delete 删除角色
func (d *CharacterDAO) Delete(id uint) error {
	return d.db.Delete(&model.Character{}, id).Error
}
