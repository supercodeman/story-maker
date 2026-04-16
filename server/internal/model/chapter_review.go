// server/internal/model/chapter_review.go
package model

import "time"

// ChapterReview 章节审核评分记录
type ChapterReview struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	WorkflowID           uint      `gorm:"index;not null" json:"workflow_id"`
	NovelID              uint      `gorm:"index" json:"novel_id"`
	ChapterID            uint      `gorm:"index" json:"chapter_id"`
	Round                int       `gorm:"not null;default:1" json:"round"`
	OverallScore         int       `json:"overall_score"`
	Passed               bool      `json:"passed"`
	CharacterScore       int       `json:"character_score"`
	CharacterIssues      string    `gorm:"type:text" json:"character_issues"`
	PlotScore            int       `json:"plot_score"`
	PlotIssues           string    `gorm:"type:text" json:"plot_issues"`
	WorldviewScore       int       `json:"worldview_score"`
	WorldviewIssues      string    `gorm:"type:text" json:"worldview_issues"`
	NarrativeScore       int       `json:"narrative_score"`
	NarrativeIssues      string    `gorm:"type:text" json:"narrative_issues"`
	ContinuityScore      int       `json:"continuity_score"`
	ContinuityIssues     string    `gorm:"type:text" json:"continuity_issues"`
	FormattingScore      int       `json:"formatting_score"`
	FormattingIssues     string    `gorm:"type:text" json:"formatting_issues"`
	AIArtifactsScore     int       `json:"ai_artifacts_score"`
	AIArtifactsIssues    string    `gorm:"type:text" json:"ai_artifacts_issues"`
	RevisionInstructions string    `gorm:"type:text" json:"revision_instructions"`
	CreatedAt            time.Time `json:"created_at"`
}
