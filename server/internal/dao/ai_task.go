// server/internal/dao/ai_task.go
package dao

import (
	"context"

	"story-maker/server/internal/model"
	"gorm.io/gorm"
)

// AITaskDAO AI 任务数据访问层
type AITaskDAO struct {
	db *gorm.DB
}

// NewAITaskDAO 创建 AITaskDAO 实例
func NewAITaskDAO(db *gorm.DB) *AITaskDAO {
	return &AITaskDAO{db: db}
}

// CreateTask 创建任务
func (d *AITaskDAO) CreateTask(ctx context.Context, task *model.AITask) error {
	return d.db.WithContext(ctx).Create(task).Error
}

// UpdateTask 更新任务
func (d *AITaskDAO) UpdateTask(ctx context.Context, task *model.AITask) error {
	return d.db.WithContext(ctx).Save(task).Error
}

// GetTask 根据 ID 获取任务
func (d *AITaskDAO) GetTask(ctx context.Context, taskID uint) (*model.AITask, error) {
	var task model.AITask
	err := d.db.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasksByUser 获取用户的任务列表
func (d *AITaskDAO) ListTasksByUser(ctx context.Context, userID uint, limit, offset int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64

	query := d.db.WithContext(ctx).Where("user_id = ?", userID)

	if err := query.Model(&model.AITask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

// ListTasksByPortfolio 获取作品集的任务列表，可选按 taskTypes / butlerSessionID / novelID 过滤
func (d *AITaskDAO) ListTasksByPortfolio(ctx context.Context, portfolioID uint, limit, offset int, taskTypes []string, butlerSessionID string, novelID uint) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64

	query := d.db.WithContext(ctx).Where("portfolio_id = ?", portfolioID)
	if len(taskTypes) > 0 {
		query = query.Where("task_type IN ?", taskTypes)
	}
	if butlerSessionID != "" {
		query = query.Where("butler_session_id = ?", butlerSessionID)
	}
	if novelID > 0 {
		query = query.Where("novel_id = ?", novelID)
	}

	if err := query.Model(&model.AITask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

// DeleteTask 删除任务
func (d *AITaskDAO) DeleteTask(ctx context.Context, taskID uint) error {
	return d.db.WithContext(ctx).Delete(&model.AITask{}, taskID).Error
}

// SumTokensByNovel 统计小说的 token 使用总量（仅统计已完成的任务）
func (d *AITaskDAO) SumTokensByNovel(ctx context.Context, novelID uint) (int, error) {
	var total *int
	err := d.db.WithContext(ctx).Model(&model.AITask{}).
		Where("novel_id = ? AND status = ?", novelID, model.TaskStatusCompleted).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// UpdateNovelIDBySessionID 按 butler_session_id 批量回填 novel_id
func (d *AITaskDAO) UpdateNovelIDBySessionID(ctx context.Context, sessionID string, novelID uint) error {
	if sessionID == "" {
		return nil
	}
	return d.db.WithContext(ctx).
		Model(&model.AITask{}).
		Where("butler_session_id = ? AND novel_id = 0", sessionID).
		Update("novel_id", novelID).Error
}

// ListOrphanButlerTasks 查询 portfolio 下 novel_id=0 的管家类型已完成任务（孤立任务）
func (d *AITaskDAO) ListOrphanButlerTasks(ctx context.Context, portfolioID uint, taskTypes []string) ([]*model.AITask, error) {
	var tasks []*model.AITask
	err := d.db.WithContext(ctx).
		Where("portfolio_id = ? AND novel_id = 0 AND status = ? AND task_type IN ?",
			portfolioID, model.TaskStatusCompleted, taskTypes).
		Order("created_at ASC").
		Find(&tasks).Error
	return tasks, err
}

// HasLinkedButlerTasks 检查指定时间范围内是否存在已关联真实小说的管家任务
func (d *AITaskDAO) HasLinkedButlerTasks(ctx context.Context, portfolioID uint, taskTypes []string, after, before interface{}) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.AITask{}).
		Where("portfolio_id = ? AND novel_id > 0 AND status = ? AND task_type IN ? AND created_at >= ? AND created_at <= ?",
			portfolioID, model.TaskStatusCompleted, taskTypes, after, before).
		Count(&count).Error
	return count > 0, err
}

// BatchUpdateNovelID 批量更新任务的 novel_id
func (d *AITaskDAO) BatchUpdateNovelID(ctx context.Context, taskIDs []uint, novelID uint) error {
	if len(taskIDs) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).
		Model(&model.AITask{}).
		Where("id IN ?", taskIDs).
		Update("novel_id", novelID).Error
}

// ResetNovelIDByNovelIDs 将指定 novel_id 列表的任务 novel_id 重置为 0
func (d *AITaskDAO) ResetNovelIDByNovelIDs(ctx context.Context, novelIDs []uint) error {
	if len(novelIDs) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).
		Model(&model.AITask{}).
		Where("novel_id IN ?", novelIDs).
		Update("novel_id", 0).Error
}

// CountPipelineTasksByStatus 统计指定 pipeline + stage 下特定状态的任务数
func (d *AITaskDAO) CountPipelineTasksByStatus(ctx context.Context, pipelineID uint, stage string, statuses []string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.AITask{}).
		Where("pipeline_id = ? AND stage = ? AND status IN ?", pipelineID, stage, statuses).
		Count(&count).Error
	return count, err
}
