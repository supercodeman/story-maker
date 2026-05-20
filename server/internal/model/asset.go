// server/internal/model/asset.go
package model

import "time"

// Asset 资源表，存储生成的图片、音频、视频等文件
type Asset struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PortfolioID uint      `gorm:"index;not null" json:"portfolio_id"`
	Type        string    `gorm:"size:20;not null" json:"type"`       // image, text, script, audio, video
	FilePath    string    `gorm:"size:500;not null" json:"file_path"` // 存储路径
	Metadata    string    `gorm:"type:json" json:"metadata"`          // 生成参数、提示词等
	Duration    float64   `gorm:"default:0" json:"duration"`          // 音频/视频时长（秒）
	ChapterID   *uint     `gorm:"index" json:"chapter_id,omitempty"`  // 关联章节ID
	Role        string    `gorm:"size:50;default:''" json:"role"`     // 角色标记：character_ref 等
	CreatedBy   uint      `gorm:"index;not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// AssetType 资源类型枚举
const (
	AssetTypeImage  = "image"
	AssetTypeText   = "text"
	AssetTypeScript = "script"
	AssetTypeAudio  = "audio"
	AssetTypeVideo  = "video"
)

// ValidAssetTypes 合法的资源类型白名单
var ValidAssetTypes = map[string]bool{
	AssetTypeImage:  true,
	AssetTypeText:   true,
	AssetTypeScript: true,
	AssetTypeAudio:  true,
	AssetTypeVideo:  true,
}

// AllowedUploadMIMETypes 允许上传的 MIME 类型白名单
var AllowedUploadMIMETypes = map[string]bool{
	"image/jpeg":  true,
	"image/png":   true,
	"image/webp":  true,
	"image/gif":   true,
	"audio/mpeg":  true,
	"audio/mp3":   true,
	"video/mp4":   true,
	"video/webm":  true,
}

// MaxUploadSize 最大上传文件大小：20MB
const MaxUploadSize = 20 * 1024 * 1024
