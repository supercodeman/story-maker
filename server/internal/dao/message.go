// server/internal/dao/message.go
package dao

import (
	"context"

	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// MessageDAO 消息数据访问层
type MessageDAO struct {
	db *gorm.DB
}

func NewMessageDAO(db *gorm.DB) *MessageDAO {
	return &MessageDAO{db: db}
}

func (d *MessageDAO) Create(ctx context.Context, msg *model.Message) error {
	return d.db.WithContext(ctx).Create(msg).Error
}

func (d *MessageDAO) ListByConversation(ctx context.Context, convID uint, limit, offset int) ([]*model.Message, error) {
	var msgs []*model.Message
	query := d.db.WithContext(ctx).Where("conversation_id = ?", convID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	err := query.Find(&msgs).Error
	return msgs, err
}

// GetRecentMessages 获取最近 N 条消息（按时间倒序取，再正序返回）
func (d *MessageDAO) GetRecentMessages(ctx context.Context, convID uint, limit int) ([]*model.Message, error) {
	var msgs []*model.Message
	err := d.db.WithContext(ctx).Where("conversation_id = ?", convID).
		Order("created_at DESC").Limit(limit).Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	// 反转为正序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (d *MessageDAO) CountByConversation(ctx context.Context, convID uint) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Message{}).Where("conversation_id = ?", convID).Count(&count).Error
	return count, err
}

func (d *MessageDAO) DeleteBatch(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Message{}).Error
}
