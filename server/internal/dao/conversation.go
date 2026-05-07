// server/internal/dao/conversation.go
package dao

import (
	"context"

	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// ConversationDAO 会话数据访问层
type ConversationDAO struct {
	db *gorm.DB
}

func NewConversationDAO(db *gorm.DB) *ConversationDAO {
	return &ConversationDAO{db: db}
}

func (d *ConversationDAO) Create(ctx context.Context, conv *model.Conversation) error {
	return d.db.WithContext(ctx).Create(conv).Error
}

func (d *ConversationDAO) GetByID(ctx context.Context, id uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := d.db.WithContext(ctx).First(&conv, id).Error
	return &conv, err
}

func (d *ConversationDAO) Update(ctx context.Context, conv *model.Conversation) error {
	return d.db.WithContext(ctx).Save(conv).Error
}

func (d *ConversationDAO) ListByUser(ctx context.Context, userID uint, portfolioID *uint, limit, offset int) ([]*model.Conversation, int64, error) {
	var convs []*model.Conversation
	var total int64

	query := d.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, model.ConversationStatusActive)
	if portfolioID != nil {
		query = query.Where("portfolio_id = ?", *portfolioID)
	}

	if err := query.Model(&model.Conversation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&convs).Error
	return convs, total, err
}

func (d *ConversationDAO) Archive(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).
		Update("status", model.ConversationStatusArchived).Error
}
