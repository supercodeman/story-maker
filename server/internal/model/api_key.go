// server/internal/model/api_key.go
package model

import "time"

// APIKey 用户 API Key 管理表，支持用户自有 Key 和平台默认 Key
type APIKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Provider  string    `gorm:"size:50" json:"provider"` // kimi, claude, copilot
	KeyValue  string    `gorm:"size:500" json:"-"`       // 加密存储，不返回给前端
	IsDefault bool      `json:"is_default"`              // 是否为该 Provider 的默认 Key
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Provider 常量
const (
	ProviderKimi     = "kimi"
	ProviderClaude   = "claude"
	ProviderCopilot  = "copilot"
	ProviderZhipu    = "zhipu"
	ProviderQwen     = "qwen"
	ProviderDeepseek = "deepseek"
)
