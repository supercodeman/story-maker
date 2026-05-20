// server/internal/model/comic_drama.go
package model

import "time"

// ComicDrama 漫剧项目主表
type ComicDrama struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	NovelID    uint      `gorm:"index" json:"novel_id"`
	ChapterID  uint      `gorm:"index" json:"chapter_id"`
	Title      string    `gorm:"size:200" json:"title"`
	Stage      string    `gorm:"size:30;default:draft" json:"stage"`
	StageIndex int       `gorm:"default:0" json:"stage_index"`
	Status     string    `gorm:"size:20;default:pending" json:"status"`
	Config     string    `gorm:"type:text" json:"config"`
	ErrorMsg   string    `gorm:"type:text" json:"error_msg"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const (
	ComicStageDraft      = "draft"
	ComicStageScript     = "script"
	ComicStageStoryboard = "storyboard"
	ComicStageCharRef    = "char_ref"
	ComicStageAudio      = "audio"
	ComicStageMedia      = "media"
	ComicStageCompose    = "compose"
	ComicStageDone       = "done"
)

const (
	ComicStatusPending   = "pending"
	ComicStatusRunning   = "running"
	ComicStatusPaused    = "paused"
	ComicStatusCompleted = "completed"
	ComicStatusFailed    = "failed"
)

// ComicScript 漫剧剧本段落
type ComicScript struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ComicDramaID uint      `gorm:"index" json:"comic_drama_id"`
	SeqNo        int       `gorm:"default:0" json:"seq_no"`
	SceneDesc    string    `gorm:"type:text" json:"scene_desc"`
	Dialogue     string    `gorm:"type:text" json:"dialogue"`
	Emotion      string    `gorm:"size:50" json:"emotion"`
	MediaType    string    `gorm:"size:20" json:"media_type"`
	Duration     float64   `gorm:"default:0" json:"duration"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Storyboard 分镜
type Storyboard struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ComicDramaID  uint      `gorm:"index" json:"comic_drama_id"`
	ComicScriptID uint      `gorm:"index" json:"comic_script_id"`
	SeqNo         int       `gorm:"default:0" json:"seq_no"`
	FrameDesc     string    `gorm:"type:text" json:"frame_desc"`
	CameraAngle   string    `gorm:"size:50" json:"camera_angle"`
	Characters    string    `gorm:"type:text" json:"characters"`
	AudioURL      string    `gorm:"size:500" json:"audio_url"`
	MediaURL      string    `gorm:"size:500" json:"media_url"`
	Status        string    `gorm:"size:20;default:pending" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CharacterRef 角色定妆照
type CharacterRef struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ComicDramaID uint      `gorm:"index" json:"comic_drama_id"`
	CharacterID  uint      `gorm:"index" json:"character_id"`
	Name         string    `gorm:"size:100" json:"name"`
	RefImageURL  string    `gorm:"size:500" json:"ref_image_url"`
	StylePrompt  string    `gorm:"type:text" json:"style_prompt"`
	Status       string    `gorm:"size:20;default:pending" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
