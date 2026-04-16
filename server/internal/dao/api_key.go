// server/internal/dao/api_key.go
package dao

import (
	"context"

	"ai-curton/server/internal/model"
	"gorm.io/gorm"
)

// APIKeyDAO API Key 数据访问层
type APIKeyDAO struct {
	db *gorm.DB
}

// NewAPIKeyDAO 创建 APIKeyDAO 实例
func NewAPIKeyDAO(db *gorm.DB) *APIKeyDAO {
	return &APIKeyDAO{db: db}
}

// CreateKey 创建 API Key
func (d *APIKeyDAO) CreateKey(ctx context.Context, key *model.APIKey) error {
	return d.db.WithContext(ctx).Create(key).Error
}

// GetKeys 获取用户的所有 API Key
func (d *APIKeyDAO) GetKeys(ctx context.Context, userID uint) ([]*model.APIKey, error) {
	var keys []*model.APIKey
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

// GetKeyByID 根据 ID 获取 API Key
func (d *APIKeyDAO) GetKeyByID(ctx context.Context, keyID uint) (*model.APIKey, error) {
	var key model.APIKey
	err := d.db.WithContext(ctx).First(&key, keyID).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// UpdateKey 更新 API Key
func (d *APIKeyDAO) UpdateKey(ctx context.Context, key *model.APIKey) error {
	return d.db.WithContext(ctx).Save(key).Error
}

// DeleteKey 删除 API Key
func (d *APIKeyDAO) DeleteKey(ctx context.Context, keyID uint) error {
	return d.db.WithContext(ctx).Delete(&model.APIKey{}, keyID).Error
}

// GetUserKey 获取用户指定 Provider 的 API Key（优先 is_default=true）
func (d *APIKeyDAO) GetUserKey(ctx context.Context, userID uint, provider string) (*model.APIKey, error) {
	var key model.APIKey
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Order("is_default DESC").
		First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}
