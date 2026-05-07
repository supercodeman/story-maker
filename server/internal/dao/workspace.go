// server/internal/dao/workspace.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// WorkspaceDAO 工作空间数据访问层
type WorkspaceDAO struct {
	db *gorm.DB
}

// NewWorkspaceDAO 创建 WorkspaceDAO 实例
func NewWorkspaceDAO() *WorkspaceDAO {
	return &WorkspaceDAO{db: model.DB}
}

// Create 创建工作空间
func (d *WorkspaceDAO) Create(ws *model.Workspace) error {
	return d.db.Create(ws).Error
}

// GetByID 根据 ID 获取工作空间
func (d *WorkspaceDAO) GetByID(id uint) (*model.Workspace, error) {
	var ws model.Workspace
	err := d.db.First(&ws, id).Error
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

// ListByUserID 获取用户所属的所有工作空间（通过成员表关联查询）
func (d *WorkspaceDAO) ListByUserID(userID uint) ([]model.Workspace, error) {
	var workspaces []model.Workspace
	err := d.db.
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ?", userID).
		Find(&workspaces).Error
	return workspaces, err
}

// Update 更新工作空间
func (d *WorkspaceDAO) Update(ws *model.Workspace) error {
	return d.db.Save(ws).Error
}

// Delete 删除工作空间（同时删除成员关系）
func (d *WorkspaceDAO) Delete(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workspace_id = ?", id).Delete(&model.WorkspaceMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Workspace{}, id).Error
	})
}

// AddMember 添加工作空间成员
func (d *WorkspaceDAO) AddMember(member *model.WorkspaceMember) error {
	return d.db.Create(member).Error
}

// RemoveMember 移除工作空间成员
func (d *WorkspaceDAO) RemoveMember(workspaceID, userID uint) error {
	return d.db.
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&model.WorkspaceMember{}).Error
}

// GetMembers 获取工作空间所有成员
func (d *WorkspaceDAO) GetMembers(workspaceID uint) ([]model.WorkspaceMember, error) {
	var members []model.WorkspaceMember
	err := d.db.Where("workspace_id = ?", workspaceID).Find(&members).Error
	return members, err
}

// GetMember 获取指定用户在工作空间中的成员记录
func (d *WorkspaceDAO) GetMember(workspaceID, userID uint) (*model.WorkspaceMember, error) {
	var member model.WorkspaceMember
	err := d.db.
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// CheckPermission 检查用户是否拥有指定角色权限
// 权限层级：owner > editor > viewer
func (d *WorkspaceDAO) CheckPermission(workspaceID, userID uint, requiredRole string) (bool, error) {
	member, err := d.GetMember(workspaceID, userID)
	if err != nil {
		return false, err
	}

	// 权限层级映射
	roleLevel := map[string]int{
		model.WorkspaceRoleViewer: 1,
		model.WorkspaceRoleEditor: 2,
		model.WorkspaceRoleOwner:  3,
	}

	return roleLevel[member.Role] >= roleLevel[requiredRole], nil
}
