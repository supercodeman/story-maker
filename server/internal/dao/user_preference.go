// server/internal/dao/user_preference.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserPreferenceDAO 用户偏好摘要数据访问层
type UserPreferenceDAO struct {
	db *gorm.DB
}

// NewUserPreferenceDAO 创建 UserPreferenceDAO 实例
func NewUserPreferenceDAO() *UserPreferenceDAO {
	return &UserPreferenceDAO{db: model.DB}
}

// GetByUserNovel 根据用户ID和小说ID获取偏好
func (d *UserPreferenceDAO) GetByUserNovel(userID, novelID uint) (*model.UserPreference, error) {
	var pref model.UserPreference
	err := d.db.Where("user_id = ? AND novel_id = ?", userID, novelID).First(&pref).Error
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// Upsert 创建或更新偏好（基于 user_id + novel_id 唯一索引）
func (d *UserPreferenceDAO) Upsert(pref *model.UserPreference) error {
	return d.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "novel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"vocab_profile", "style_profile", "narrative_profile",
			"ai_feedback_profile", "prompt_summary", "event_count", "version", "updated_at",
		}),
	}).Create(pref).Error
}
