// server/internal/model/ai_model_status.go
package model

import "time"

// ========== 能力常量 ==========

const (
	CapTextGen    = "text_gen"
	CapTextPolish = "text_polish"
	CapEmbedding  = "embedding"
	CapImageGen   = "image_gen"
	CapImageEdit  = "image_edit"
)

// ========== 模型状态表 ==========

// AIModelStatus 模型可用性状态（DB 持久化）
type AIModelStatus struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Provider    string     `gorm:"size:50;uniqueIndex:uk_pmc" json:"provider"`
	ModelName   string     `gorm:"size:100;uniqueIndex:uk_pmc;default:''" json:"model_name"`
	Capability  string     `gorm:"size:50;uniqueIndex:uk_pmc" json:"capability"`
	IsAvailable bool       `gorm:"default:true" json:"is_available"`
	LastCheck   *time.Time `json:"last_check"`
	LastError   string     `gorm:"size:500;default:''" json:"last_error"`
	LatencyMs   int        `gorm:"default:0" json:"latency_ms"`
	Priority    int        `gorm:"default:0" json:"priority"` // 降级优先级，数字越小越优先
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ========== 静态元数据 ==========

// ModelMeta 单个模型元数据
type ModelMeta struct {
	ModelName   string
	DisplayName string
}

// ProviderMeta Provider 静态元数据
type ProviderMeta struct {
	Provider     string
	DisplayName  string
	Models       []ModelMeta
	Capabilities []string
	Priority     int // 默认排序优先级，数字越小优先级越高
}

// DefaultProviders 所有 Provider 的静态注册表（作为 seed 数据源）
var DefaultProviders = []ProviderMeta{
	{
		Provider:     "qwen",
		DisplayName:  "通义千问",
		Priority:     1,
		Capabilities: []string{CapTextGen, CapTextPolish, CapEmbedding},
		Models: []ModelMeta{
			//{ModelName: "qwen-long", DisplayName: "Qwen Long"},
			{ModelName: "qwen-max", DisplayName: "Qwen Max"},
			{ModelName: "deepseek-v3.2", DisplayName: "deepseek-v3.2"},
			{ModelName: "deepseek-v3", DisplayName: "deepseek-v3"},
		},
	},
	{
		Provider:     "zhipu",
		DisplayName:  "智谱 AI",
		Priority:     2,
		Capabilities: []string{CapTextGen, CapTextPolish, CapEmbedding},
		Models: []ModelMeta{
			{ModelName: "glm-4.5-flash", DisplayName: "GLM-4.5 Flash"},
			{ModelName: "glm-4.7-flash", DisplayName: "GLM-4.7 Flash"},
		},
	},
	{
		Provider:     "deepseek",
		DisplayName:  "DeepSeek",
		Priority:     3,
		Capabilities: []string{CapTextGen, CapTextPolish},
		Models: []ModelMeta{
			{ModelName: "deepseek-r1:7b", DisplayName: "DeepSeek Chat"},
		},
	},
	{
		Provider:     "kimi",
		DisplayName:  "Kimi",
		Priority:     4,
		Capabilities: []string{CapTextGen, CapTextPolish},
		Models: []ModelMeta{
			{ModelName: "", DisplayName: "默认"},
		},
	},
}
