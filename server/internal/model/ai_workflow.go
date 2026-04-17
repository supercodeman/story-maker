// server/internal/model/ai_workflow.go
package model

import "time"

// AIWorkflow 编排工作流主表
type AIWorkflow struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index" json:"user_id"`
	PortfolioID    uint      `gorm:"index" json:"portfolio_id"`
	NovelID        uint      `gorm:"index;default:0" json:"novel_id"`
	WorkflowType   string    `gorm:"size:50" json:"workflow_type"`
	Status         string    `gorm:"size:20;default:pending" json:"status"`
	GraphJSON      string    `gorm:"type:text" json:"graph_json"`
	InitialContext string    `gorm:"type:text" json:"initial_context"`
	ResultJSON     string    `gorm:"type:text" json:"result_json"`
	ErrorMsg       string    `gorm:"type:text" json:"error_msg"`
	TotalNodes     int       `json:"total_nodes"`
	CompletedNodes int       `json:"completed_nodes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AIWorkflowNode 工作流节点表
type AIWorkflowNode struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	WorkflowID uint      `gorm:"index" json:"workflow_id"`
	NodeID     string    `gorm:"size:100" json:"node_id"`
	TaskID     uint      `json:"task_id"`
	Status     string    `gorm:"size:20;default:pending" json:"status"`
	ResultJSON string    `gorm:"type:text" json:"result_json"`
	ErrorMsg   string    `gorm:"type:text" json:"error_msg"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// 工作流状态常量
const (
	WorkflowStatusPending              = "pending"
	WorkflowStatusRunning              = "running"
	WorkflowStatusCompleted            = "completed"
	WorkflowStatusCompletedWithWarning = "completed_with_warning" // 循环耗尽但继续使用最后结果
	WorkflowStatusFailed               = "failed"
	WorkflowStatusCancelled            = "cancelled"
)

// 工作流类型常量
const (
	WorkflowTypeFullChapter       = "full_chapter"
	WorkflowTypeBatchExpand       = "batch_expand"
	WorkflowTypeNovelRevision     = "novel_revision"
	WorkflowTypeNovelRevisionExec = "novel_revision_execute"
	WorkflowTypeMemoryExtract     = "memory_extract"
	WorkflowTypeMemoryReview      = "memory_review"
	WorkflowTypeHitAnalysis       = "hit_analysis"
	WorkflowTypeOpeningChapter    = "opening_chapter"
)
