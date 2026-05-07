// server/internal/dao/user_style.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// UserStyleDAO 用户风格数据访问层
type UserStyleDAO struct {
	db *gorm.DB
}

// NewUserStyleDAO 创建 UserStyleDAO 实例
func NewUserStyleDAO() *UserStyleDAO {
	return &UserStyleDAO{db: model.DB}
}

// ListByUserID 获取用户的所有风格模板
func (d *UserStyleDAO) ListByUserID(userID uint) ([]model.UserStyle, error) {
	var styles []model.UserStyle
	err := d.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&styles).Error
	return styles, err
}

// GetByID 根据 ID 获取风格
func (d *UserStyleDAO) GetByID(id uint) (*model.UserStyle, error) {
	var style model.UserStyle
	err := d.db.First(&style, id).Error
	if err != nil {
		return nil, err
	}
	return &style, nil
}

// Create 创建风格
func (d *UserStyleDAO) Create(style *model.UserStyle) error {
	return d.db.Create(style).Error
}

// Update 更新风格
func (d *UserStyleDAO) Update(style *model.UserStyle) error {
	return d.db.Save(style).Error
}

// Delete 删除风格
func (d *UserStyleDAO) Delete(id uint) error {
	return d.db.Delete(&model.UserStyle{}, id).Error
}
