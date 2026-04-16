// server/internal/model/character.go
package model

import "time"

// Character 角色模型表，用于人物一致性管理
// ReferenceImages 和 Attributes 使用 JSON 字符串存储，保持灵活性
type Character struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PortfolioID     uint      `gorm:"index;not null" json:"portfolio_id"`
	Name            string    `gorm:"size:100;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	ReferenceImages string    `gorm:"type:json" json:"reference_images"` // JSON 数组：参考图路径列表
	LoraPath        string    `gorm:"size:500" json:"lora_path"`         // LoRA 模型路径（预留）
	Attributes      string    `gorm:"type:json" json:"attributes"`       // JSON 对象：角色属性（发型、服装等）
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
