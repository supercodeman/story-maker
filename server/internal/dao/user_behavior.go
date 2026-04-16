// server/internal/dao/user_behavior.go
package dao

import (
	"time"

	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// UserBehaviorDAO 用户行为事件数据访问层
type UserBehaviorDAO struct {
	db *gorm.DB
}

// NewUserBehaviorDAO 创建 UserBehaviorDAO 实例
func NewUserBehaviorDAO() *UserBehaviorDAO {
	return &UserBehaviorDAO{db: model.DB}
}

// CreateEvent 创建行为事件
func (d *UserBehaviorDAO) CreateEvent(event *model.UserBehaviorEvent) error {
	return d.db.Create(event).Error
}

// ListEventsByUserNovel 查询用户在某小说的行为事件
func (d *UserBehaviorDAO) ListEventsByUserNovel(userID, novelID uint, since time.Time, limit int) ([]model.UserBehaviorEvent, error) {
	var events []model.UserBehaviorEvent
	err := d.db.Where("user_id = ? AND novel_id = ? AND created_at > ?", userID, novelID, since).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// CountEventsByUserNovel 统计用户在某小说的事件数
func (d *UserBehaviorDAO) CountEventsByUserNovel(userID, novelID uint, since time.Time) (int64, error) {
	var count int64
	err := d.db.Model(&model.UserBehaviorEvent{}).
		Where("user_id = ? AND novel_id = ? AND created_at > ?", userID, novelID, since).
		Count(&count).Error
	return count, err
}

// PurgeOldEvents 清理指定时间之前的事件
func (d *UserBehaviorDAO) PurgeOldEvents(before time.Time) error {
	return d.db.Where("created_at < ?", before).Delete(&model.UserBehaviorEvent{}).Error
}
