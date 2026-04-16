// server/internal/service/portfolio.go
package service

import (
	"errors"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// PortfolioService 作品集业务逻辑层
type PortfolioService struct {
	portfolioDAO *dao.PortfolioDAO
	workspaceDAO *dao.WorkspaceDAO
}

// NewPortfolioService 创建 PortfolioService 实例
func NewPortfolioService() *PortfolioService {
	return &PortfolioService{
		portfolioDAO: dao.NewPortfolioDAO(),
		workspaceDAO: dao.NewWorkspaceDAO(),
	}
}

// CreatePortfolioRequest 创建作品集请求参数
type CreatePortfolioRequest struct {
	WorkspaceID uint   `json:"workspace_id" binding:"required"`
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description"`
}

// UpdatePortfolioRequest 更新作品集请求参数
type UpdatePortfolioRequest struct {
	Name        string `json:"name" binding:"omitempty,max=100"`
	Description string `json:"description"`
	CoverImage  string `json:"cover_image"`
	Status      string `json:"status" binding:"omitempty,oneof=draft published"`
}

// Create 创建作品集（需 editor 及以上权限）
func (s *PortfolioService) Create(userID uint, req *CreatePortfolioRequest) (*model.Portfolio, error) {
	// 校验用户对工作空间的编辑权限
	ok, err := s.workspaceDAO.CheckPermission(req.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return nil, errors.New("access denied: editor permission required")
	}

	p := &model.Portfolio{
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		Description: req.Description,
		Status:      model.PortfolioStatusDraft,
	}

	if err := s.portfolioDAO.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetByID 获取作品集详情（需工作空间成员身份）
func (s *PortfolioService) GetByID(portfolioID, userID uint) (*model.Portfolio, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, err
	}

	// 校验用户是否为该作品集所属工作空间的成员
	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return p, nil
}

// List 获取工作空间下的作品集列表（需工作空间成员身份）
func (s *PortfolioService) List(workspaceID, userID uint) ([]model.Portfolio, error) {
	_, err := s.workspaceDAO.GetMember(workspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return s.portfolioDAO.ListByWorkspaceID(workspaceID)
}

// Update 更新作品集（需 editor 及以上权限）
func (s *PortfolioService) Update(portfolioID, userID uint, req *UpdatePortfolioRequest) (*model.Portfolio, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, err
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return nil, errors.New("access denied: editor permission required")
	}

	if req.Name != "" {
		p.Name = req.Name
	}
	p.Description = req.Description
	if req.CoverImage != "" {
		p.CoverImage = req.CoverImage
	}
	if req.Status != "" && model.ValidPortfolioStatuses[req.Status] {
		p.Status = req.Status
	}

	if err := s.portfolioDAO.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete 删除作品集（需 owner 权限）
func (s *PortfolioService) Delete(portfolioID, userID uint) error {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return err
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return errors.New("access denied: owner permission required")
	}

	return s.portfolioDAO.Delete(portfolioID)
}
