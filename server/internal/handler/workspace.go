// server/internal/handler/workspace.go
package handler

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/service"

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
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, workspaces)
}

// Create 创建工作空间
// POST /api/v1/workspaces
func (h *WorkspaceHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ws, err := h.svc.Create(userID, &req)
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, ws)
}

// Get 获取工作空间详情
// GET /api/v1/workspaces/:id
func (h *WorkspaceHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace id")
		return
	}

	ws, err := h.svc.GetByID(uint(wsID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, ws)
}

// Update 更新工作空间
// PUT /api/v1/workspaces/:id
func (h *WorkspaceHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace id")
		return
	}

	var req service.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ws, err := h.svc.Update(uint(wsID), userID, &req)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, ws)
}

// Delete 删除工作空间
// DELETE /api/v1/workspaces/:id
func (h *WorkspaceHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace id")
		return
	}

	if err := h.svc.Delete(uint(wsID), userID); err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, gin.H{"message": "workspace deleted"})
}

// GetMembers 获取工作空间成员列表
// GET /api/v1/workspaces/:id/members
func (h *WorkspaceHandler) GetMembers(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace id")
		return
	}

	members, err := h.svc.GetMembers(uint(wsID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, members)
}

// AddMember 添加工作空间成员
// POST /api/v1/workspaces/:id/members
func (h *WorkspaceHandler) AddMember(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace id")
		return
	}

	var req service.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.AddMember(uint(wsID), userID, &req); err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, gin.H{"message": "member added"})
}

// RemoveMember 移除工作空间成员
// DELETE /api/v1/workspaces/:id/members/:user_id
func (h *WorkspaceHandler) RemoveMember(c *gin.Context) {
	operatorID := c.GetUint("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace id")
		return
	}
	targetUserID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}

	if err := h.svc.RemoveMember(uint(wsID), operatorID, uint(targetUserID)); err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, gin.H{"message": "member removed"})
}
