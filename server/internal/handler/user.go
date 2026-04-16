// server/internal/handler/user.go
package handler

import (
	"strconv"

	"ai-curton/server/internal/middleware"
	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户信息相关请求处理
type UserHandler struct {
	authService *service.AuthService
}

// NewUserHandler 创建 UserHandler 实例
func NewUserHandler() *UserHandler {
	return &UserHandler{
		authService: service.NewAuthService(),
	}
}

// GetProfile 获取当前用户个人信息
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, user)
}

// UpdateProfile 更新当前用户个人信息
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	user, err := h.authService.UpdateProfile(userID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, user)
}

// AdminListUsers 管理员获取所有用户列表
// GET /api/v1/admin/users
func (h *UserHandler) AdminListUsers(c *gin.Context) {
	users, err := h.authService.ListUsers()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, users)
}

// UpdateRoleRequest 更新角色请求参数
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// AdminUpdateRole 管理员更新用户角色
// PUT /api/v1/admin/users/:uid/role
func (h *UserHandler) AdminUpdateRole(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	if err := h.authService.UpdateUserRole(uint(uid), req.Role); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "Role updated successfully"})
}
