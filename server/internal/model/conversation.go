// server/internal/model/conversation.go
package model

import "time"

// Conversation 会话表，管理一次连贯的对话生命周期
type Conversation struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	PortfolioID  uint      `gorm:"index" json:"portfolio_id"`
	Title        string    `gorm:"size:200" json:"title"`
	ModelName    string    `gorm:"size:50" json:"model_name"`
	Summary      string    `gorm:"type:text" json:"summary"`
	MessageCount int       `json:"message_count"`
	Status       string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 会话状态常量
const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
)

// Message 消息表，记录每一轮对话
type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index" json:"conversation_id"`
	Role           string    `gorm:"size:20" json:"role"` // user, assistant, system
	Content        string    `gorm:"type:text" json:"content"`
	TokenCount     int       `json:"token_count"`
	TaskID         *uint     `json:"task_id"` // 关联的 AITask（可选）
	CreatedAt      time.Time `json:"created_at"`
}
