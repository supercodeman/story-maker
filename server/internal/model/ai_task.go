// server/internal/model/ai_task.go
package model

import "time"

// AITask AI 任务表，记录每次 AI 调用的完整生命周期
type AITask struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	PortfolioID      uint      `gorm:"index" json:"portfolio_id"`
	TaskType         string    `gorm:"size:50" json:"task_type"`     // text_gen, image_gen, character_adjust
	ModelName        string    `gorm:"size:50" json:"model_name"`    // kimi, claude, copilot
	Prompt           string    `gorm:"type:longtext" json:"prompt"`
	History          string    `gorm:"type:text" json:"history"`              // 多轮对话历史 JSON
	Status           string    `gorm:"size:20;default:pending" json:"status"` // pending, running, completed, failed, cancelled
	Result           string    `gorm:"type:text" json:"result"`
	ErrorMsg         string    `gorm:"type:text" json:"error_msg"`
	NovelID          uint      `gorm:"index;default:0" json:"novel_id"`          // 关联小说，0 表示非小说任务
	ChapterID        *uint     `gorm:"index" json:"chapter_id,omitempty"`        // 关联章节，多模态任务用于写回 Asset
	ButlerSessionID  string    `gorm:"size:36;index" json:"butler_session_id"`   // 管家会话 ID，串联同一次管家创作的所有任务
	PromptTokens     int       `gorm:"default:0" json:"prompt_tokens"`
	CompletionTokens int       `gorm:"default:0" json:"completion_tokens"`
	TotalTokens      int       `gorm:"default:0" json:"total_tokens"`
	PipelineID       uint      `gorm:"index;default:0" json:"pipeline_id"`
	Stage            string    `gorm:"size:30" json:"stage,omitempty"`
	StageIndex       int       `gorm:"default:0" json:"stage_index"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AITask 状态常量
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

// AITask 任务类型常量
const (
	TaskTypeTextGen         = "text_gen"
	TaskTypeTextPolish      = "text_polish"
	TaskTypeStoryboard      = "storyboard"
	TaskTypeImageGen        = "image_gen"
	TaskTypeImageEdit       = "image_edit"
	TaskTypeCharacterAdjust = "character_adjust"

	// 小说大纲生成任务类型
	TaskTypeOutlineGenerate = "outline_generate"

	// 小说工坊章节 AI 任务类型
	TaskTypeChapterSummaryPolish = "chapter_summary_polish"
	TaskTypeChapterPolish        = "chapter_polish"
	TaskTypeChapterExpand        = "chapter_expand"
	TaskTypeChapterContinue      = "chapter_continue"

	// 大纲页面章节级 AI 操作任务类型
	TaskTypeOutlineTitlePolish        = "outline_title_polish"
	TaskTypeOutlineSummaryPolish      = "outline_summary_polish"
	TaskTypeOutlineSummaryExpand      = "outline_summary_expand"
	TaskTypeOutlineGenerateCharacters = "outline_generate_characters"

	// 知识库 AI 提取任务类型
	TaskTypeKnowledgeExtract = "knowledge_extract"

	// 总览相关任务类型
	TaskTypeRevisionAnalysis = "revision_analysis"
	TaskTypeRevisionPlanning = "revision_planning"
	TaskTypeChapterRevise    = "chapter_revise"
	TaskTypeOverviewExtract  = "overview_extract"

	// 小说管家任务类型
	TaskTypeButlerGenerateTopic      = "butler_generate_topic"
	TaskTypeButlerGenerateStoryline  = "butler_generate_storyline"
	TaskTypeButlerGenerateCharacters = "butler_generate_characters"

	// 小说管家多轮迭代任务类型
	TaskTypeButlerStorylineDraft   = "butler_storyline_draft"   // 故事线草稿生成
	TaskTypeButlerStorylineReview  = "butler_storyline_review"  // 故事线 Review 修改
	TaskTypeButlerCharactersDraft  = "butler_characters_draft"  // 人物设计草稿生成
	TaskTypeButlerCharactersReview = "butler_characters_review" // 人物设计 Review 修改
	TaskTypeButlerOpeningDraft     = "butler_opening_draft"     // 前5章概要精细化
	TaskTypeButlerOpeningReview    = "butler_opening_review"    // 前5章概要审查

	// 记忆提取相关任务类型
	TaskTypeMemoryFeatureExtract   = "memory_feature_extract"
	TaskTypeMemoryEmbeddingGen     = "memory_embedding_gen"
	TaskTypeMemoryPromptCompile    = "memory_prompt_compile"
	TaskTypeMemoryQualityEval      = "memory_quality_eval"
	TaskTypeMemoryReviewQuality    = "memory_review_quality"
	TaskTypeMemoryReviewCompliance = "memory_review_compliance"
	TaskTypeMemoryReviewDecision   = "memory_review_decision"

	// 多模态生成任务类型
	TaskTypeAudioGen = "audio_gen" // 文生音频（TTS）
	TaskTypeVideoGen = "video_gen" // 文生视频

	// 导出任务类型
	TaskTypeExportWord  = "export_word"  // 导出 Word 文档
	TaskTypeExportAudio = "export_audio" // 导出全本音频

	// 世界构建与规划任务类型
	TaskTypeWorldviewGenerate  = "worldview_generate"   // 世界观生成
	TaskTypeWorldviewReview    = "worldview_review"     // 世界观审查
	TaskTypeCharacterGenerate2 = "character_generate"   // 人物设定生成（世界构建阶段）
	TaskTypeCharacterReview    = "character_review"     // 人物设定审查
	TaskTypeRelationGenerate   = "relation_generate"    // 关系设定生成
	TaskTypeRelationReview     = "relation_review"      // 关系设定审查
	TaskTypeForeshadowGenerate = "foreshadow_generate"  // 伏笔设定生成
	TaskTypeForeshadowReview   = "foreshadow_review"    // 伏笔设定审查
	TaskTypePlotGenerate       = "plot_outline_generate" // 剧情大纲生成
	TaskTypePlotReview         = "plot_outline_review"   // 剧情大纲审查

	// 漫剧 Pipeline 任务类型
	TaskTypeComicScript     = "comic_script"
	TaskTypeComicStoryboard = "comic_storyboard"
	TaskTypeComicCharRef    = "comic_char_ref"
	TaskTypeComicAudio      = "comic_audio"
	TaskTypeComicMedia      = "comic_media"
	TaskTypeComicCompose    = "comic_compose"
)
