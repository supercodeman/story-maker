// server/internal/model/prompt_template.go
package model

import "time"

// PromptTemplate Prompt 模板表，支持系统默认和小说级自定义
type PromptTemplate struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NovelID    uint      `gorm:"not null;default:0;uniqueIndex:uk_novel_action_type" json:"novel_id"` // 0=系统默认
	Action     string    `gorm:"size:30;not null;uniqueIndex:uk_novel_action_type" json:"action"`      // summary_polish/polish/expand/continue
	PromptType string    `gorm:"size:20;not null;default:user;uniqueIndex:uk_novel_action_type" json:"prompt_type"` // system/user
	Name       string    `gorm:"size:100;not null" json:"name"`
	Content    string    `gorm:"type:text;not null" json:"content"` // Go text/template 语法
	IsDefault  bool      `gorm:"not null;default:false" json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PromptTemplateData 模板渲染时传入的数据
type PromptTemplateData struct {
	NovelTitle       string
	NovelDescription string
	ChapterTitle     string
	ChapterSummary   string
	ChapterContent   string
	PrevSummaries    string // 已格式化的前文各章概要
	PrevContent      string // 前一章末尾内容（截取 2000 字）
	WordCount        int    // 当前字数
	TargetWords      int    // 扩写目标字数
	SelectedText     string // 用户选中的文本片段，为空表示全文操作

	// 知识库上下文（RAG Phase 1）
	KnowledgeContext string // 根据当前章节相关性筛选后的知识条目（已格式化）
	Characters       string // 本章涉及的人物档案
	WorldviewNotes   string // 相关世界观设定

	// 大纲生成专用字段
	Setting        string // 世界观/设定
	Background     string // 背景信息
	Plot           string // 剧情思路
	ChapterNum     int    // 期望章节数
	PrevChapters   string // 前文章节概要（已格式化）
	NextChapters   string // 后续章节概要（已格式化）
	UserInstruction string // 用户自定义指令

	// 写作风格（全局 + 场景预设合并后的格式化文本）
	WritingStyle string

	// 增强大纲：剧情结构模板 + 爆款拆解 + 多轮迭代
	StructureSkeleton string // 剧情结构模板骨架文本
	HitAnalysisRef    string // 爆款拆解参考文本
	PrevOutline       string // 上一轮大纲结果（多轮迭代）
	UserFeedback      string // 用户反馈（多轮迭代）

	// 历史审核问题上下文（注入 ChapterAIAction，帮助 AI 规避已知问题）
	ReviewContext string

	// 润色方向预设
	PolishMode            string // 润色方向标识（dialogue/pacing/sensory/emotion/trim）
	PolishModeInstruction string // 润色方向对应的指令文本
}
