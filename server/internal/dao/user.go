// server/internal/dao/user.go
package dao

import (
	"time"

	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// UserDAO 用户数据访问对象
type UserDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建 UserDAO 实例
func NewUserDAO() *UserDAO {
	return &UserDAO{db: model.DB}
}

// CreateUser 创建用户记录
func (d *UserDAO) CreateUser(user *model.User) error {
	return d.db.Create(user).Error
}

// GetUserByEmail 根据邮箱查询用户
func (d *UserDAO) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := d.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 根据 ID 查询用户
func (d *UserDAO) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	err := d.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息（仅更新非零值字段）
func (d *UserDAO) UpdateUser(user *model.User) error {
	return d.db.Model(user).Updates(map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
	}).Error
}

// GetUserByUsername 根据用户名查询用户
func (d *UserDAO) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := d.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers 查询所有用户列表
func (d *UserDAO) ListUsers() ([]model.User, error) {
	var users []model.User
	err := d.db.Order("id ASC").Find(&users).Error
	return users, err
}

// UpdateRole 更新用户角色字段
func (d *UserDAO) UpdateRole(id uint, role string) error {
	return d.db.Model(&model.User{}).Where("id = ?", id).Update("role", role).Error
}

// UpdateWriterLevel 更新用户写手等级及解锁信息
func (d *UserDAO) UpdateWriterLevel(id uint, level, source string) error {
	now := time.Now()
	return d.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"writer_level":   level,
		"level_unlock_at": &now,
		"level_source":   source,
	}).Error
}

// UpdateWriterStats 更新用户创作统计
func (d *UserDAO) UpdateWriterStats(id uint, wordDelta int64, chapterDelta int) error {
	return d.db.Model(&model.User{}).Where("id = ?", id).
		UpdateColumns(map[string]interface{}{
			"total_word_count": gorm.Expr("total_word_count + ?", wordDelta),
			"total_chapters":   gorm.Expr("total_chapters + ?", chapterDelta),
		}).Error
}

// IncrCompletedNovels 完本数 +1
func (d *UserDAO) IncrCompletedNovels(id uint) error {
	return d.db.Model(&model.User{}).Where("id = ?", id).
		UpdateColumn("completed_novels", gorm.Expr("completed_novels + 1")).Error
}

// UpdateViewMode 更新视图模式
func (d *UserDAO) UpdateViewMode(id uint, mode string) error {
	return d.db.Model(&model.User{}).Where("id = ?", id).Update("view_mode", mode).Error
}

// EnsureAllUsersAdmin 确保所有用户都是 admin + 大神写手（幂等，启动时调用）
func (d *UserDAO) EnsureAllUsersAdmin() error {
	return d.db.Model(&model.User{}).Where("role != ? OR writer_level != ? OR view_mode != ?", "admin", model.WriterLevelAdvanced, model.ViewModeAdvanced).Updates(map[string]interface{}{
		"role":         "admin",
		"writer_level": model.WriterLevelAdvanced,
		"view_mode":    model.ViewModeAdvanced,
		"level_source": model.LevelSourceAdmin,
	}).Error
}
