# Plan 2: 核心业务模块 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现工作空间、作品集、角色、资源的完整 CRUD，包含权限校验和文件存储抽象层。

**Architecture:** 基于 Plan 1 的三层架构扩展，每个业务模块独立 model/dao/service/handler，通过权限中间件统一鉴权。

**Tech Stack:** Go, Gin, GORM, MySQL 8.0

---

### Task 1: Workspace 模型

**Files:**
- Create: `server/internal/model/workspace.go`

- [ ] **Step 1: 定义 Workspace 和 WorkspaceMember 模型并注册自动迁移**

```go
// server/internal/model/workspace.go
package model

import "time"

// Workspace 工作空间表，支持个人和团队两种类型
type Workspace struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Type        string    `gorm:"size:20;not null;default:personal" json:"type"` // personal, team
	OwnerID     uint      `gorm:"index;not null" json:"owner_id"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceMember 工作空间成员表，管理用户与工作空间的关联关系
type WorkspaceMember struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"uniqueIndex:idx_ws_user;not null" json:"workspace_id"`
	UserID      uint      `gorm:"uniqueIndex:idx_ws_user;not null" json:"user_id"`
	Role        string    `gorm:"size:20;not null;default:viewer" json:"role"` // owner, editor, viewer
	CreatedAt   time.Time `json:"created_at"`
}

// WorkspaceType 工作空间类型枚举
const (
	WorkspaceTypePersonal = "personal"
	WorkspaceTypeTeam     = "team"
)

// WorkspaceRole 工作空间角色枚举
const (
	WorkspaceRoleOwner  = "owner"
	WorkspaceRoleEditor = "editor"
	WorkspaceRoleViewer = "viewer"
)

// ValidWorkspaceTypes 合法的工作空间类型白名单
var ValidWorkspaceTypes = map[string]bool{
	WorkspaceTypePersonal: true,
	WorkspaceTypeTeam:     true,
}

// ValidWorkspaceRoles 合法的工作空间角色白名单
var ValidWorkspaceRoles = map[string]bool{
	WorkspaceRoleOwner:  true,
	WorkspaceRoleEditor: true,
	WorkspaceRoleViewer: true,
}
```

同时在 `server/internal/model/` 的初始化逻辑中（如 `db.go` 或 `init.go`）注册自动迁移：

```go
// 在 model 初始化函数中追加
DB.AutoMigrate(&Workspace{}, &WorkspaceMember{})
```

- [ ] **Step 2: Commit**

```bash
git add server/internal/model/workspace.go
git commit -m "feat(model): add Workspace and WorkspaceMember models with role/type enums"
```

---

### Task 2: Workspace DAO + Service + Handler

**Files:**
- Create: `server/internal/dao/workspace.go`
- Create: `server/internal/service/workspace.go`
- Create: `server/internal/handler/workspace.go`

- [ ] **Step 1: 实现 Workspace DAO 层**

```go
// server/internal/dao/workspace.go
package dao

import (
	"ai-curton/server/internal/model"

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
```

- [ ] **Step 2: 实现 Workspace Service 层**

```go
// server/internal/service/workspace.go
package service

import (
	"errors"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
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
```

- [ ] **Step 3: 实现 Workspace Handler 层**

```go
// server/internal/handler/workspace.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// WorkspaceHandler 工作空间请求处理层
type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

// NewWorkspaceHandler 创建 WorkspaceHandler 实例
func NewWorkspaceHandler() *WorkspaceHandler {
	return &WorkspaceHandler{svc: service.NewWorkspaceService()}
}

// List 获取当前用户的工作空间列表
// GET /api/v1/workspaces
func (h *WorkspaceHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	workspaces, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workspaces})
}

// Create 创建工作空间
// POST /api/v1/workspaces
func (h *WorkspaceHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ws, err := h.svc.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": ws})
}

// Get 获取工作空间详情
// GET /api/v1/workspaces/:id
func (h *WorkspaceHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	ws, err := h.svc.GetByID(uint(wsID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ws})
}

// Update 更新工作空间
// PUT /api/v1/workspaces/:id
func (h *WorkspaceHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	var req service.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ws, err := h.svc.Update(uint(wsID), userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ws})
}

// Delete 删除工作空间
// DELETE /api/v1/workspaces/:id
func (h *WorkspaceHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	if err := h.svc.Delete(uint(wsID), userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workspace deleted"})
}

// GetMembers 获取工作空间成员列表
// GET /api/v1/workspaces/:id/members
func (h *WorkspaceHandler) GetMembers(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	members, err := h.svc.GetMembers(uint(wsID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": members})
}

// AddMember 添加工作空间成员
// POST /api/v1/workspaces/:id/members
func (h *WorkspaceHandler) AddMember(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	var req service.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.AddMember(uint(wsID), userID, &req); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "member added"})
}

// RemoveMember 移除工作空间成员
// DELETE /api/v1/workspaces/:id/members/:user_id
func (h *WorkspaceHandler) RemoveMember(c *gin.Context) {
	operatorID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}
	targetUserID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.svc.RemoveMember(uint(wsID), operatorID, uint(targetUserID)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}
```

- [ ] **Step 4: Commit**

```bash
git add server/internal/dao/workspace.go server/internal/service/workspace.go server/internal/handler/workspace.go
git commit -m "feat(workspace): implement workspace CRUD with member management (dao/service/handler)"
```

---

### Task 3: 权限中间件

**Files:**
- Create: `server/internal/middleware/permission.go`

- [ ] **Step 1: 实现基于工作空间的权限校验中间件**

```go
// server/internal/middleware/permission.go
package middleware

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

	"github.com/gin-gonic/gin"
)

// RequireWorkspaceMember 校验用户是否为工作空间成员
// 从 URL 参数 :id 中获取 workspace_id，从 JWT context 中获取 user_id
func RequireWorkspaceMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
			c.Abort()
			return
		}

		wsDAO := dao.NewWorkspaceDAO()
		member, err := wsDAO.GetMember(uint(wsID), userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied: not a workspace member"})
			c.Abort()
			return
		}

		// 将成员角色写入 context，供后续 handler 使用
		c.Set("workspace_id", uint(wsID))
		c.Set("workspace_role", member.Role)
		c.Next()
	}
}

// RequireWorkspaceRole 校验用户角色是否满足最低要求
// 权限层级：owner(3) > editor(2) > viewer(1)
// 用法：RequireWorkspaceRole("editor") 表示至少需要 editor 权限
func RequireWorkspaceRole(requiredRole string) gin.HandlerFunc {
	roleLevel := map[string]int{
		model.WorkspaceRoleViewer: 1,
		model.WorkspaceRoleEditor: 2,
		model.WorkspaceRoleOwner:  3,
	}

	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
			c.Abort()
			return
		}

		wsDAO := dao.NewWorkspaceDAO()
		member, err := wsDAO.GetMember(uint(wsID), userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied: not a workspace member"})
			c.Abort()
			return
		}

		if roleLevel[member.Role] < roleLevel[requiredRole] {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied: insufficient permissions"})
			c.Abort()
			return
		}

		c.Set("workspace_id", uint(wsID))
		c.Set("workspace_role", member.Role)
		c.Next()
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add server/internal/middleware/permission.go
git commit -m "feat(middleware): add workspace permission middleware with role-based access control"
```

---

### Task 4: Portfolio 模型 + DAO + Service + Handler

**Files:**
- Create: `server/internal/model/portfolio.go`
- Create: `server/internal/dao/portfolio.go`
- Create: `server/internal/service/portfolio.go`
- Create: `server/internal/handler/portfolio.go`

- [ ] **Step 1: 定义 Portfolio 模型**

```go
// server/internal/model/portfolio.go
package model

import "time"

// Portfolio 作品集表，归属于某个工作空间
type Portfolio struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"index;not null" json:"workspace_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CoverImage  string    `gorm:"size:500" json:"cover_image"`
	Status      string    `gorm:"size:20;not null;default:draft" json:"status"` // draft, published
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PortfolioStatus 作品集状态枚举
const (
	PortfolioStatusDraft     = "draft"
	PortfolioStatusPublished = "published"
)

// ValidPortfolioStatuses 合法的作品集状态白名单
var ValidPortfolioStatuses = map[string]bool{
	PortfolioStatusDraft:     true,
	PortfolioStatusPublished: true,
}
```

同时在 model 初始化中追加自动迁移：

```go
DB.AutoMigrate(&Portfolio{})
```

- [ ] **Step 2: 实现 Portfolio DAO 层**

```go
// server/internal/dao/portfolio.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// PortfolioDAO 作品集数据访问层
type PortfolioDAO struct {
	db *gorm.DB
}

// NewPortfolioDAO 创建 PortfolioDAO 实例
func NewPortfolioDAO() *PortfolioDAO {
	return &PortfolioDAO{db: model.DB}
}

// Create 创建作品集
func (d *PortfolioDAO) Create(p *model.Portfolio) error {
	return d.db.Create(p).Error
}

// GetByID 根据 ID 获取作品集
func (d *PortfolioDAO) GetByID(id uint) (*model.Portfolio, error) {
	var p model.Portfolio
	err := d.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListByWorkspaceID 获取工作空间下的所有作品集
func (d *PortfolioDAO) ListByWorkspaceID(workspaceID uint) ([]model.Portfolio, error) {
	var portfolios []model.Portfolio
	err := d.db.Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&portfolios).Error
	return portfolios, err
}

// Update 更新作品集
func (d *PortfolioDAO) Update(p *model.Portfolio) error {
	return d.db.Save(p).Error
}

// Delete 删除作品集
func (d *PortfolioDAO) Delete(id uint) error {
	return d.db.Delete(&model.Portfolio{}, id).Error
}
```

- [ ] **Step 3: 实现 Portfolio Service 层**

```go
// server/internal/service/portfolio.go
package service

import (
	"errors"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// PortfolioService 作品集业务逻辑层
type PortfolioService struct {
	portfolioDAO  *dao.PortfolioDAO
	workspaceDAO  *dao.WorkspaceDAO
}

// NewPortfolioService 创建 PortfolioService 实例
func NewPortfolioService() *PortfolioService {
	return &PortfolioService{
		portfolioDAO:  dao.NewPortfolioDAO(),
		workspaceDAO:  dao.NewWorkspaceDAO(),
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
```

- [ ] **Step 4: 实现 Portfolio Handler 层**

```go
// server/internal/handler/portfolio.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// PortfolioHandler 作品集请求处理层
type PortfolioHandler struct {
	svc *service.PortfolioService
}

// NewPortfolioHandler 创建 PortfolioHandler 实例
func NewPortfolioHandler() *PortfolioHandler {
	return &PortfolioHandler{svc: service.NewPortfolioService()}
}

// List 获取作品集列表（按 workspace_id 过滤）
// GET /api/v1/portfolios?workspace_id=xxx
func (h *PortfolioHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsIDStr := c.Query("workspace_id")
	if wsIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
		return
	}

	wsID, err := strconv.ParseUint(wsIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	portfolios, err := h.svc.List(uint(wsID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": portfolios})
}

// Create 创建作品集
// POST /api/v1/portfolios
func (h *PortfolioHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.CreatePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": p})
}

// Get 获取作品集详情
// GET /api/v1/portfolios/:id
func (h *PortfolioHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio id"})
		return
	}

	p, err := h.svc.GetByID(uint(pID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": p})
}

// Update 更新作品集
// PUT /api/v1/portfolios/:id
func (h *PortfolioHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio id"})
		return
	}

	var req service.UpdatePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Update(uint(pID), userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": p})
}

// Delete 删除作品集
// DELETE /api/v1/portfolios/:id
func (h *PortfolioHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio id"})
		return
	}

	if err := h.svc.Delete(uint(pID), userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "portfolio deleted"})
}
```

- [ ] **Step 5: Commit**

```bash
git add server/internal/model/portfolio.go server/internal/dao/portfolio.go server/internal/service/portfolio.go server/internal/handler/portfolio.go
git commit -m "feat(portfolio): implement portfolio CRUD with workspace permission checks"
```

---

### Task 5: Character 模型 + DAO + Service + Handler

**Files:**
- Create: `server/internal/model/character.go`
- Create: `server/internal/dao/character.go`
- Create: `server/internal/service/character.go`
- Create: `server/internal/handler/character.go`

- [ ] **Step 1: 定义 Character 模型**

```go
// server/internal/model/character.go
package model

import "time"

// Character 角色模型表，用于人物一致性管理
// ReferenceImages 和 Attributes 使用 JSON 字符串存储，保持灵活性
type Character struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PortfolioID     uint      `gorm:"index;not null" json:"portfolio_id"`
	Name            string    `gorm:"size:100;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	ReferenceImages string    `gorm:"type:json" json:"reference_images"` // JSON 数组：参考图路径列表
	LoraPath        string    `gorm:"size:500" json:"lora_path"`         // LoRA 模型路径（预留）
	Attributes      string    `gorm:"type:json" json:"attributes"`       // JSON 对象：角色属性（发型、服装等）
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

同时在 model 初始化中追加自动迁移：

```go
DB.AutoMigrate(&Character{})
```

- [ ] **Step 2: 实现 Character DAO 层**

```go
// server/internal/dao/character.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// CharacterDAO 角色数据访问层
type CharacterDAO struct {
	db *gorm.DB
}

// NewCharacterDAO 创建 CharacterDAO 实例
func NewCharacterDAO() *CharacterDAO {
	return &CharacterDAO{db: model.DB}
}

// Create 创建角色
func (d *CharacterDAO) Create(ch *model.Character) error {
	return d.db.Create(ch).Error
}

// GetByID 根据 ID 获取角色
func (d *CharacterDAO) GetByID(id uint) (*model.Character, error) {
	var ch model.Character
	err := d.db.First(&ch, id).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// ListByPortfolioID 获取作品集下的所有角色
func (d *CharacterDAO) ListByPortfolioID(portfolioID uint) ([]model.Character, error) {
	var characters []model.Character
	err := d.db.Where("portfolio_id = ?", portfolioID).
		Order("created_at DESC").
		Find(&characters).Error
	return characters, err
}

// Update 更新角色
func (d *CharacterDAO) Update(ch *model.Character) error {
	return d.db.Save(ch).Error
}

// Delete 删除角色
func (d *CharacterDAO) Delete(id uint) error {
	return d.db.Delete(&model.Character{}, id).Error
}
```

- [ ] **Step 3: 实现 Character Service 层**

```go
// server/internal/service/character.go
package service

import (
	"encoding/json"
	"errors"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// CharacterService 角色业务逻辑层
type CharacterService struct {
	characterDAO *dao.CharacterDAO
	portfolioDAO *dao.PortfolioDAO
	workspaceDAO *dao.WorkspaceDAO
}

// NewCharacterService 创建 CharacterService 实例
func NewCharacterService() *CharacterService {
	return &CharacterService{
		characterDAO: dao.NewCharacterDAO(),
		portfolioDAO: dao.NewPortfolioDAO(),
		workspaceDAO: dao.NewWorkspaceDAO(),
	}
}

// CreateCharacterRequest 创建角色请求参数
type CreateCharacterRequest struct {
	Name        string            `json:"name" binding:"required,max=100"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"` // 角色属性键值对
}

// UpdateCharacterRequest 更新角色请求参数
type UpdateCharacterRequest struct {
	Name        string            `json:"name" binding:"omitempty,max=100"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"`
}

// checkPortfolioEditorPermission 校验用户对作品集所属工作空间的编辑权限
// 遵循 DRY 原则：抽取公共权限校验逻辑
func (s *CharacterService) checkPortfolioEditorPermission(portfolioID, userID uint) (*model.Portfolio, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, errors.New("portfolio not found")
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return nil, errors.New("access denied: editor permission required")
	}

	return p, nil
}

// Create 创建角色（需 editor 及以上权限）
func (s *CharacterService) Create(portfolioID, userID uint, req *CreateCharacterRequest) (*model.Character, error) {
	_, err := s.checkPortfolioEditorPermission(portfolioID, userID)
	if err != nil {
		return nil, err
	}

	// 序列化 attributes 为 JSON 字符串
	attrJSON := "{}"
	if req.Attributes != nil {
		b, err := json.Marshal(req.Attributes)
		if err != nil {
			return nil, errors.New("invalid attributes format")
		}
		attrJSON = string(b)
	}

	ch := &model.Character{
		PortfolioID:     portfolioID,
		Name:            req.Name,
		Description:     req.Description,
		ReferenceImages: "[]", // 初始化为空数组
		Attributes:      attrJSON,
	}

	if err := s.characterDAO.Create(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// GetByID 获取角色详情（需工作空间成员身份）
func (s *CharacterService) GetByID(characterID, userID uint) (*model.Character, error) {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return nil, err
	}

	p, err := s.portfolioDAO.GetByID(ch.PortfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return ch, nil
}

// List 获取作品集下的角色列表（需工作空间成员身份）
func (s *CharacterService) List(portfolioID, userID uint) ([]model.Character, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return s.characterDAO.ListByPortfolioID(portfolioID)
}

// Update 更新角色（需 editor 及以上权限）
func (s *CharacterService) Update(characterID, userID uint, req *UpdateCharacterRequest) (*model.Character, error) {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return nil, err
	}

	_, err = s.checkPortfolioEditorPermission(ch.PortfolioID, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		ch.Name = req.Name
	}
	ch.Description = req.Description
	if req.Attributes != nil {
		b, err := json.Marshal(req.Attributes)
		if err != nil {
			return nil, errors.New("invalid attributes format")
		}
		ch.Attributes = string(b)
	}

	if err := s.characterDAO.Update(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// Delete 删除角色（需 owner 权限）
func (s *CharacterService) Delete(characterID, userID uint) error {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return err
	}

	p, err := s.portfolioDAO.GetByID(ch.PortfolioID)
	if err != nil {
		return err
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return errors.New("access denied: owner permission required")
	}

	return s.characterDAO.Delete(characterID)
}

// AddReferenceImage 添加参考图到角色的 ReferenceImages JSON 数组
func (s *CharacterService) AddReferenceImage(characterID, userID uint, imagePath string) (*model.Character, error) {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return nil, err
	}

	_, err = s.checkPortfolioEditorPermission(ch.PortfolioID, userID)
	if err != nil {
		return nil, err
	}

	// 解析现有参考图列表
	var images []string
	if ch.ReferenceImages != "" && ch.ReferenceImages != "[]" {
		if err := json.Unmarshal([]byte(ch.ReferenceImages), &images); err != nil {
			images = []string{}
		}
	}

	// 限制最多 5 张参考图（MVP 阶段约束）
	if len(images) >= 5 {
		return nil, errors.New("maximum 5 reference images allowed")
	}

	images = append(images, imagePath)
	b, _ := json.Marshal(images)
	ch.ReferenceImages = string(b)

	if err := s.characterDAO.Update(ch); err != nil {
		return nil, err
	}
	return ch, nil
}
```

- [ ] **Step 4: 实现 Character Handler 层**

```go
// server/internal/handler/character.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"
	"ai-curton/server/internal/storage"

	"github.com/gin-gonic/gin"
)

// CharacterHandler 角色请求处理层
type CharacterHandler struct {
	svc     *service.CharacterService
	storage storage.Storage
}

// NewCharacterHandler 创建 CharacterHandler 实例
func NewCharacterHandler(store storage.Storage) *CharacterHandler {
	return &CharacterHandler{
		svc:     service.NewCharacterService(),
		storage: store,
	}
}

// List 获取作品集下的角色列表
// GET /api/v1/portfolios/:id/characters
func (h *CharacterHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio id"})
		return
	}

	characters, err := h.svc.List(uint(pID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": characters})
}

// Create 创建角色
// POST /api/v1/portfolios/:id/characters
func (h *CharacterHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio id"})
		return
	}

	var req service.CreateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.svc.Create(uint(pID), userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": ch})
}

// Get 获取角色详情
// GET /api/v1/characters/:id
func (h *CharacterHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid character id"})
		return
	}

	ch, err := h.svc.GetByID(uint(chID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ch})
}

// Update 更新角色
// PUT /api/v1/characters/:id
func (h *CharacterHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid character id"})
		return
	}

	var req service.UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.svc.Update(uint(chID), userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ch})
}

// Delete 删除角色
// DELETE /api/v1/characters/:id
func (h *CharacterHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid character id"})
		return
	}

	if err := h.svc.Delete(uint(chID), userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "character deleted"})
}

// UploadReference 上传角色参考图
// POST /api/v1/characters/:id/reference
func (h *CharacterHandler) UploadReference(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid character id"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// 校验文件大小（20MB 限制）
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 20MB limit"})
		return
	}

	// 校验文件类型
	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only jpg/png/webp/gif files are allowed"})
		return
	}

	// 构造存储路径：characters/{character_id}/{filename}
	storagePath := "characters/" + strconv.FormatUint(chID, 10) + "/" + header.Filename
	url, err := h.storage.Upload(c.Request.Context(), file, storagePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	// 将参考图路径添加到角色记录
	ch, err := h.svc.AddReferenceImage(uint(chID), userID, url)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ch})
}
```

- [ ] **Step 5: Commit**

```bash
git add server/internal/model/character.go server/internal/dao/character.go server/internal/service/character.go server/internal/handler/character.go
git commit -m "feat(character): implement character CRUD with reference image upload"
```

---

### Task 6: 文件存储抽象层

**Files:**
- Create: `server/internal/storage/storage.go`
- Create: `server/internal/storage/local.go`

- [ ] **Step 1: 定义 Storage 接口**

```go
// server/internal/storage/storage.go
package storage

import (
	"context"
	"io"
)

// Storage 文件存储抽象接口
// 遵循 ISP（接口隔离原则）：仅定义文件存储必需的四个操作
// 本期实现 LocalStorage，后续切换 OSS 只需替换实现（OCP 开闭原则）
type Storage interface {
	// Upload 上传文件，返回可访问的 URL 或路径
	Upload(ctx context.Context, file io.Reader, path string) (string, error)

	// Download 下载文件，返回文件内容的 ReadCloser
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(ctx context.Context, path string) error

	// GetURL 获取文件的访问 URL
	GetURL(ctx context.Context, path string) (string, error)
}
```

- [ ] **Step 2: 实现本地文件存储**

```go
// server/internal/storage/local.go
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	basePath string // 本地存储根目录，如 ./uploads
	baseURL  string // 文件访问的 URL 前缀，如 http://localhost:8080/uploads
}

// NewLocalStorage 创建本地存储实例
// basePath: 本地存储根目录
// baseURL: 文件访问的 URL 前缀
func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	// 确保存储根目录存在
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Upload 上传文件到本地磁盘
func (s *LocalStorage) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
	fullPath := filepath.Join(s.basePath, path)

	// 确保目标目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 写入文件内容
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 返回可访问的 URL
	url := s.baseURL + "/" + path
	return url, nil
}

// Download 从本地磁盘读取文件
func (s *LocalStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete 从本地磁盘删除文件
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetURL 获取文件的访问 URL
func (s *LocalStorage) GetURL(ctx context.Context, path string) (string, error) {
	fullPath := filepath.Join(s.basePath, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return s.baseURL + "/" + path, nil
}
```

- [ ] **Step 3: Commit**

```bash
git add server/internal/storage/storage.go server/internal/storage/local.go
git commit -m "feat(storage): add file storage abstraction layer with local filesystem implementation"
```

---

### Task 7: Asset 模型 + DAO + Service + Handler

**Files:**
- Create: `server/internal/model/asset.go`
- Create: `server/internal/dao/asset.go`
- Create: `server/internal/service/asset.go`
- Create: `server/internal/handler/asset.go`

- [ ] **Step 1: 定义 Asset 模型**

```go
// server/internal/model/asset.go
package model

import "time"

// Asset 资源表，存储生成的图片、文本等文件
type Asset struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PortfolioID uint      `gorm:"index;not null" json:"portfolio_id"`
	Type        string    `gorm:"size:20;not null" json:"type"`      // image, text, script
	FilePath    string    `gorm:"size:500;not null" json:"file_path"` // 存储路径
	Metadata    string    `gorm:"type:json" json:"metadata"`          // 生成参数、提示词等
	CreatedBy   uint      `gorm:"index;not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// AssetType 资源类型枚举
const (
	AssetTypeImage  = "image"
	AssetTypeText   = "text"
	AssetTypeScript = "script"
)

// ValidAssetTypes 合法的资源类型白名单
var ValidAssetTypes = map[string]bool{
	AssetTypeImage:  true,
	AssetTypeText:   true,
	AssetTypeScript: true,
}

// AllowedUploadMIMETypes 允许上传的 MIME 类型白名单
var AllowedUploadMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// MaxUploadSize 最大上传文件大小：20MB
const MaxUploadSize = 20 * 1024 * 1024
```

同时在 model 初始化中追加自动迁移：

```go
DB.AutoMigrate(&Asset{})
```

- [ ] **Step 2: 实现 Asset DAO 层**

```go
// server/internal/dao/asset.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// AssetDAO 资源数据访问层
type AssetDAO struct {
	db *gorm.DB
}

// NewAssetDAO 创建 AssetDAO 实例
func NewAssetDAO() *AssetDAO {
	return &AssetDAO{db: model.DB}
}

// Create 创建资源记录
func (d *AssetDAO) Create(a *model.Asset) error {
	return d.db.Create(a).Error
}

// GetByID 根据 ID 获取资源
func (d *AssetDAO) GetByID(id uint) (*model.Asset, error) {
	var a model.Asset
	err := d.db.First(&a, id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByPortfolioID 获取作品集下的所有资源
func (d *AssetDAO) ListByPortfolioID(portfolioID uint) ([]model.Asset, error) {
	var assets []model.Asset
	err := d.db.Where("portfolio_id = ?", portfolioID).
		Order("created_at DESC").
		Find(&assets).Error
	return assets, err
}

// Delete 删除资源记录
func (d *AssetDAO) Delete(id uint) error {
	return d.db.Delete(&model.Asset{}, id).Error
}
```

- [ ] **Step 3: 实现 Asset Service 层**

```go
// server/internal/service/asset.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
	"ai-curton/server/internal/storage"
)

// AssetService 资源业务逻辑层
type AssetService struct {
	assetDAO     *dao.AssetDAO
	portfolioDAO *dao.PortfolioDAO
	workspaceDAO *dao.WorkspaceDAO
	storage      storage.Storage
}

// NewAssetService 创建 AssetService 实例
func NewAssetService(store storage.Storage) *AssetService {
	return &AssetService{
		assetDAO:     dao.NewAssetDAO(),
		portfolioDAO: dao.NewPortfolioDAO(),
		workspaceDAO: dao.NewWorkspaceDAO(),
		storage:      store,
	}
}

// UploadAssetRequest 上传资源请求参数（非 JSON，从 form-data 解析）
type UploadAssetRequest struct {
	PortfolioID uint   // 从 form field 获取
	Type        string // 资源类型
}

// Upload 上传资源文件并创建记录
func (s *AssetService) Upload(ctx context.Context, userID uint, portfolioID uint, assetType string, filename string, file io.Reader, metadata map[string]string) (*model.Asset, error) {
	// 校验资源类型白名单
	if !model.ValidAssetTypes[assetType] {
		return nil, errors.New("invalid asset type")
	}

	// 校验用户对作品集所属工作空间的编辑权限
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, errors.New("portfolio not found")
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return nil, errors.New("access denied: editor permission required")
	}

	// 构造存储路径：assets/{portfolio_id}/{timestamp}_{filename}
	storagePath := "assets/" + uintToStr(portfolioID) + "/" + timePrefix() + "_" + filename
	url, err := s.storage.Upload(ctx, file, storagePath)
	if err != nil {
		return nil, errors.New("failed to upload file")
	}

	// 序列化 metadata
	metaJSON := "{}"
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		metaJSON = string(b)
	}

	asset := &model.Asset{
		PortfolioID: portfolioID,
		Type:        assetType,
		FilePath:    url,
		Metadata:    metaJSON,
		CreatedBy:   userID,
	}

	if err := s.assetDAO.Create(asset); err != nil {
		return nil, err
	}
	return asset, nil
}

// GetByID 获取资源详情（需工作空间成员身份）
func (s *AssetService) GetByID(assetID, userID uint) (*model.Asset, error) {
	a, err := s.assetDAO.GetByID(assetID)
	if err != nil {
		return nil, err
	}

	p, err := s.portfolioDAO.GetByID(a.PortfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return a, nil
}

// List 获取作品集下的资源列表（需工作空间成员身份）
func (s *AssetService) List(portfolioID, userID uint) ([]model.Asset, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return s.assetDAO.ListByPortfolioID(portfolioID)
}

// Delete 删除资源（需 editor 及以上权限，同时删除存储文件）
func (s *AssetService) Delete(ctx context.Context, assetID, userID uint) error {
	a, err := s.assetDAO.GetByID(assetID)
	if err != nil {
		return err
	}

	p, err := s.portfolioDAO.GetByID(a.PortfolioID)
	if err != nil {
		return err
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return errors.New("access denied: editor permission required")
	}

	// 删除存储文件（忽略文件不存在的错误）
	_ = s.storage.Delete(ctx, a.FilePath)

	return s.assetDAO.Delete(assetID)
}

// uintToStr 辅助函数：uint 转字符串
func uintToStr(n uint) string {
	return fmt.Sprintf("%d", n)
}

// timePrefix 辅助函数：生成时间戳前缀
func timePrefix() string {
	return time.Now().Format("20060102150405")
}
```

注意：需要在文件顶部补充 `"fmt"` 导入。完整 import 块：

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
	"ai-curton/server/internal/storage"
)
```

- [ ] **Step 4: 实现 Asset Handler 层**

```go
// server/internal/handler/asset.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/model"
	"ai-curton/server/internal/service"
	"ai-curton/server/internal/storage"

	"github.com/gin-gonic/gin"
)

// AssetHandler 资源请求处理层
type AssetHandler struct {
	svc     *service.AssetService
	storage storage.Storage
}

// NewAssetHandler 创建 AssetHandler 实例
func NewAssetHandler(store storage.Storage) *AssetHandler {
	return &AssetHandler{
		svc:     service.NewAssetService(store),
		storage: store,
	}
}

// List 获取作品集下的资源列表
// GET /api/v1/portfolios/:id/assets
func (h *AssetHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio id"})
		return
	}

	assets, err := h.svc.List(uint(pID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": assets})
}

// Upload 上传资源文件
// POST /api/v1/assets/upload
// Form fields: file(文件), portfolio_id(uint), type(string)
func (h *AssetHandler) Upload(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 解析 portfolio_id
	portfolioIDStr := c.PostForm("portfolio_id")
	if portfolioIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "portfolio_id is required"})
		return
	}
	portfolioID, err := strconv.ParseUint(portfolioIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio_id"})
		return
	}

	// 解析资源类型
	assetType := c.PostForm("type")
	if assetType == "" {
		assetType = model.AssetTypeImage // 默认为图片类型
	}

	// 获取上传文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// 校验文件大小（20MB 限制）
	if header.Size > model.MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 20MB limit"})
		return
	}

	// 校验文件 MIME 类型（白名单校验，防止恶意文件上传）
	contentType := header.Header.Get("Content-Type")
	if !model.AllowedUploadMIMETypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only jpg/png/webp/gif files are allowed"})
		return
	}

	// 从 form 中解析可选的 metadata
	metadata := map[string]string{}
	if m := c.PostForm("metadata"); m != "" {
		// 简单处理：metadata 作为单个字符串存入
		metadata["raw"] = m
	}

	asset, err := h.svc.Upload(c.Request.Context(), userID, uint(portfolioID), assetType, header.Filename, file, metadata)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": asset})
}

// Get 获取资源详情
// GET /api/v1/assets/:id
func (h *AssetHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	aID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	asset, err := h.svc.GetByID(uint(aID), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": asset})
}

// Delete 删除资源
// DELETE /api/v1/assets/:id
func (h *AssetHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	aID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(aID), userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
}
```

- [ ] **Step 5: Commit**

```bash
git add server/internal/model/asset.go server/internal/dao/asset.go server/internal/service/asset.go server/internal/handler/asset.go
git commit -m "feat(asset): implement asset CRUD with file upload validation (20MB limit, MIME whitelist)"
```

---

### Task 8: 路由注册

**Files:**
- Update: `server/cmd/main.go`

- [ ] **Step 1: 在 main.go 中注册所有业务模块路由**

假设 Plan 1 已完成基础框架，main.go 中已有 Gin 引擎初始化和 JWT 中间件。在此基础上追加业务路由：

```go
// server/cmd/main.go
package main

import (
	"log"

	"ai-curton/server/internal/handler"
	"ai-curton/server/internal/middleware"
	"ai-curton/server/internal/model"
	"ai-curton/server/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库连接（假设 Plan 1 已实现 model.InitDB()）
	if err := model.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化文件存储（本地存储）
	store := storage.NewLocalStorage("./uploads", "http://localhost:8080/uploads")

	// 创建 Gin 引擎
	r := gin.Default()

	// 静态文件服务（用于访问上传的文件）
	r.Static("/uploads", "./uploads")

	// API 路由组
	api := r.Group("/api/v1")

	// 认证路由（假设 Plan 1 已实现 AuthHandler）
	// authHandler := handler.NewAuthHandler()
	// api.POST("/auth/register", authHandler.Register)
	// api.POST("/auth/login", authHandler.Login)

	// 需要 JWT 认证的路由（假设 Plan 1 已实现 middleware.AuthRequired()）
	authorized := api.Group("")
	authorized.Use(middleware.AuthRequired())

	// Workspace 路由
	wsHandler := handler.NewWorkspaceHandler()
	{
		authorized.GET("/workspaces", wsHandler.List)
		authorized.POST("/workspaces", wsHandler.Create)
		authorized.GET("/workspaces/:id", wsHandler.Get)
		authorized.PUT("/workspaces/:id", wsHandler.Update)
		authorized.DELETE("/workspaces/:id", wsHandler.Delete)
		authorized.GET("/workspaces/:id/members", wsHandler.GetMembers)
		authorized.POST("/workspaces/:id/members", wsHandler.AddMember)
		authorized.DELETE("/workspaces/:id/members/:user_id", wsHandler.RemoveMember)
	}

	// Portfolio 路由
	portfolioHandler := handler.NewPortfolioHandler()
	{
		authorized.GET("/portfolios", portfolioHandler.List)           // 需要 workspace_id 查询参数
		authorized.POST("/portfolios", portfolioHandler.Create)
		authorized.GET("/portfolios/:id", portfolioHandler.Get)
		authorized.PUT("/portfolios/:id", portfolioHandler.Update)
		authorized.DELETE("/portfolios/:id", portfolioHandler.Delete)
	}

	// Character 路由
	characterHandler := handler.NewCharacterHandler(store)
	{
		authorized.GET("/portfolios/:id/characters", characterHandler.List)
		authorized.POST("/portfolios/:id/characters", characterHandler.Create)
		authorized.GET("/characters/:id", characterHandler.Get)
		authorized.PUT("/characters/:id", characterHandler.Update)
		authorized.DELETE("/characters/:id", characterHandler.Delete)
		authorized.POST("/characters/:id/reference", characterHandler.UploadReference)
	}

	// Asset 路由
	assetHandler := handler.NewAssetHandler(store)
	{
		authorized.GET("/portfolios/:id/assets", assetHandler.List)
		authorized.POST("/assets/upload", assetHandler.Upload)
		authorized.GET("/assets/:id", assetHandler.Get)
		authorized.DELETE("/assets/:id", assetHandler.Delete)
	}

	// 启动服务器
	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

**说明**：
1. 所有业务路由均在 `/api/v1` 前缀下，并通过 `middleware.AuthRequired()` 进行 JWT 认证
2. 文件存储使用本地实现，上传目录为 `./uploads`，通过 `/uploads` 路径提供静态文件访问
3. 路由分组清晰，每个业务模块独立注册
4. CharacterHandler 和 AssetHandler 需要注入 Storage 实例

- [ ] **Step 2: Commit**

```bash
git add server/cmd/main.go
git commit -m "feat(routes): register workspace/portfolio/character/asset routes with JWT auth"
```

---

## 实施检查清单

完成以上 8 个 Task 后，核心业务模块即已就绪。请确认：

- [ ] 所有模型已定义并注册自动迁移（Workspace, WorkspaceMember, Portfolio, Character, Asset）
- [ ] 每个模块的 DAO/Service/Handler 三层架构完整
- [ ] 权限中间件已实现，支持工作空间成员校验和角色校验
- [ ] 文件存储抽象层已实现，本地存储可用
- [ ] 文件上传已实施白名单校验（MIME 类型、文件大小限制）
- [ ] 所有路由已注册到 main.go，使用 JWT 中间件保护
- [ ] 代码遵循 DRY、KISS、SOLID、YAGNI 原则，无单文件超过 500 行

## 后续工作

Plan 2 完成后，可继续实施：
- Plan 3: AI Agent 模块（Kimi 接入、任务异步执行、WebSocket 推送）
- Plan 4: API Key 管理模块
- Plan 5: 前端 Vue 应用

---

**文档版本**: v1.0
**创建日期**: 2026-03-31
**适用项目**: Ai-Curton MVP
