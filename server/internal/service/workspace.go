// server/internal/service/workspace.go
package service

import (
	"errors"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// WorkspaceService 工作空间业务逻辑层
type WorkspaceService struct {
	dao *dao.WorkspaceDAO
}

// NewWorkspaceService 创建 WorkspaceService 实例
func NewWorkspaceService() *WorkspaceService {
	return &WorkspaceService{dao: dao.NewWorkspaceDAO()}
}

// CreateWorkspaceRequest 创建工作空间请求参数
type CreateWorkspaceRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Type        string `json:"type" binding:"required,oneof=personal team"`
	Description string `json:"description"`
}

// UpdateWorkspaceRequest 更新工作空间请求参数
type UpdateWorkspaceRequest struct {
	Name        string `json:"name" binding:"omitempty,max=100"`
	Description string `json:"description"`
}

// AddMemberRequest 添加成员请求参数
type AddMemberRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=editor viewer"`
}

// Create 创建工作空间，自动将创建者添加为 owner 成员
func (s *WorkspaceService) Create(userID uint, req *CreateWorkspaceRequest) (*model.Workspace, error) {
	if !model.ValidWorkspaceTypes[req.Type] {
		return nil, errors.New("invalid workspace type")
	}

	ws := &model.Workspace{
		Name:        req.Name,
		Type:        req.Type,
		OwnerID:     userID,
		Description: req.Description,
	}

	if err := s.dao.Create(ws); err != nil {
		return nil, err
	}

	// 自动将创建者添加为 owner 成员（遵循 SRP：创建即关联）
	member := &model.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      userID,
		Role:        model.WorkspaceRoleOwner,
	}
	if err := s.dao.AddMember(member); err != nil {
		return nil, err
	}

	return ws, nil
}

// GetByID 获取工作空间详情（需校验用户是否为成员）
func (s *WorkspaceService) GetByID(workspaceID, userID uint) (*model.Workspace, error) {
	// 先校验用户是否为成员
	_, err := s.dao.GetMember(workspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return s.dao.GetByID(workspaceID)
}

// List 获取用户所属的所有工作空间
func (s *WorkspaceService) List(userID uint) ([]model.Workspace, error) {
	return s.dao.ListByUserID(userID)
}

// Update 更新工作空间（需 owner 权限）
func (s *WorkspaceService) Update(workspaceID, userID uint, req *UpdateWorkspaceRequest) (*model.Workspace, error) {
	ok, err := s.dao.CheckPermission(workspaceID, userID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return nil, errors.New("access denied: owner permission required")
	}

	ws, err := s.dao.GetByID(workspaceID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		ws.Name = req.Name
	}
	// Description 允许清空，所以始终更新
	ws.Description = req.Description

	if err := s.dao.Update(ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// Delete 删除工作空间（需 owner 权限）
func (s *WorkspaceService) Delete(workspaceID, userID uint) error {
	ok, err := s.dao.CheckPermission(workspaceID, userID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return errors.New("access denied: owner permission required")
	}
	return s.dao.Delete(workspaceID)
}

// GetMembers 获取工作空间成员列表（需成员身份）
func (s *WorkspaceService) GetMembers(workspaceID, userID uint) ([]model.WorkspaceMember, error) {
	_, err := s.dao.GetMember(workspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}
	return s.dao.GetMembers(workspaceID)
}

// AddMember 添加工作空间成员（需 owner 权限）
func (s *WorkspaceService) AddMember(workspaceID, operatorID uint, req *AddMemberRequest) error {
	ok, err := s.dao.CheckPermission(workspaceID, operatorID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return errors.New("access denied: owner permission required")
	}

	if !model.ValidWorkspaceRoles[req.Role] || req.Role == model.WorkspaceRoleOwner {
		return errors.New("invalid role: only editor or viewer allowed")
	}

	member := &model.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      req.UserID,
		Role:        req.Role,
	}
	return s.dao.AddMember(member)
}

// RemoveMember 移除工作空间成员（需 owner 权限，不能移除自己）
func (s *WorkspaceService) RemoveMember(workspaceID, operatorID, targetUserID uint) error {
	if operatorID == targetUserID {
		return errors.New("cannot remove yourself from workspace")
	}

	ok, err := s.dao.CheckPermission(workspaceID, operatorID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return errors.New("access denied: owner permission required")
	}

	return s.dao.RemoveMember(workspaceID, targetUserID)
}
