// server/internal/dao/ai_workflow.go
package dao

import (
	"context"
	"time"

	"ai-curton/server/internal/model"
	"gorm.io/gorm"
)

// AIWorkflowDAO 工作流数据访问层
type AIWorkflowDAO struct {
	db *gorm.DB
}

// NewAIWorkflowDAO 创建 AIWorkflowDAO 实例
func NewAIWorkflowDAO(db *gorm.DB) *AIWorkflowDAO {
	return &AIWorkflowDAO{db: db}
}

// Create 创建工作流
func (d *AIWorkflowDAO) Create(ctx context.Context, workflow *model.AIWorkflow) error {
	return d.db.WithContext(ctx).Create(workflow).Error
}

// Update 更新工作流
func (d *AIWorkflowDAO) Update(ctx context.Context, workflow *model.AIWorkflow) error {
	return d.db.WithContext(ctx).Save(workflow).Error
}

// Get 根据 ID 获取工作流
func (d *AIWorkflowDAO) Get(ctx context.Context, id uint) (*model.AIWorkflow, error) {
	var wf model.AIWorkflow
	err := d.db.WithContext(ctx).First(&wf, id).Error
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

// ListByUser 获取用户的工作流列表
func (d *AIWorkflowDAO) ListByUser(ctx context.Context, userID uint, page, pageSize int) ([]model.AIWorkflow, int64, error) {
	var workflows []model.AIWorkflow
	var total int64

	query := d.db.WithContext(ctx).Where("user_id = ?", userID)

	if err := query.Model(&model.AIWorkflow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&workflows).Error

	return workflows, total, err
}

// CreateNode 创建工作流节点
func (d *AIWorkflowDAO) CreateNode(ctx context.Context, node *model.AIWorkflowNode) error {
	return d.db.WithContext(ctx).Create(node).Error
}

// UpdateNode 更新工作流节点
func (d *AIWorkflowDAO) UpdateNode(ctx context.Context, node *model.AIWorkflowNode) error {
	return d.db.WithContext(ctx).Save(node).Error
}

// GetNodesByWorkflow 获取工作流的所有节点
func (d *AIWorkflowDAO) GetNodesByWorkflow(ctx context.Context, workflowID uint) ([]model.AIWorkflowNode, error) {
	var nodes []model.AIWorkflowNode
	err := d.db.WithContext(ctx).Where("workflow_id = ?", workflowID).
		Order("id ASC").
		Find(&nodes).Error
	return nodes, err
}

// GetNode 根据 workflowID 和 nodeID 获取节点
func (d *AIWorkflowDAO) GetNode(ctx context.Context, workflowID uint, nodeID string) (*model.AIWorkflowNode, error) {
	var node model.AIWorkflowNode
	err := d.db.WithContext(ctx).
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// IncrCompletedNodes 原子递增已完成节点数，避免并发竞态
func (d *AIWorkflowDAO) IncrCompletedNodes(ctx context.Context, workflowID uint) error {
	return d.db.WithContext(ctx).
		Model(&model.AIWorkflow{}).
		Where("id = ?", workflowID).
		UpdateColumn("completed_nodes", gorm.Expr("completed_nodes + 1")).
		Error
}

// ListStale 查询卡住的工作流（状态为 pending/running 且超过指定时间未更新）
func (d *AIWorkflowDAO) ListStale(ctx context.Context, staleBefore time.Time) ([]model.AIWorkflow, error) {
	var workflows []model.AIWorkflow
	err := d.db.WithContext(ctx).
		Where("status IN ? AND updated_at < ?",
			[]string{model.WorkflowStatusPending, model.WorkflowStatusRunning},
			staleBefore,
		).
		Find(&workflows).Error
	return workflows, err
}

// ListActiveByNovel 查询指定小说下活跃的工作流（pending/running）
func (d *AIWorkflowDAO) ListActiveByNovel(ctx context.Context, novelID uint) ([]model.AIWorkflow, error) {
	var workflows []model.AIWorkflow
	err := d.db.WithContext(ctx).
		Where("novel_id = ? AND status IN ?", novelID, []string{model.WorkflowStatusPending, model.WorkflowStatusRunning}).
		Order("created_at ASC").
		Find(&workflows).Error
	return workflows, err
}

// ResetNodes 将工作流的所有未完成节点重置为 pending
func (d *AIWorkflowDAO) ResetNodes(ctx context.Context, workflowID uint) error {
	return d.db.WithContext(ctx).
		Model(&model.AIWorkflowNode{}).
		Where("workflow_id = ? AND status NOT IN ?", workflowID, []string{"completed"}).
		Updates(map[string]interface{}{
			"status":    model.WorkflowStatusPending,
			"error_msg": "",
		}).Error
}

// DeleteNodes 删除工作流的所有节点（恢复重跑时使用，避免旧节点状态残留）
func (d *AIWorkflowDAO) DeleteNodes(ctx context.Context, workflowID uint) error {
	return d.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Delete(&model.AIWorkflowNode{}).Error
}
