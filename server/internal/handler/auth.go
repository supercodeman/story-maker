// server/internal/handler/auth.go
package handler

import (
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关请求处理
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建 AuthHandler 实例
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(),
	}
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, "Registration successful", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	tokens, err := h.authService.Login(&req)
	if err != nil {
		Unauthorized(c, err.Error())
		return
	}

	Success(c, tokens)
}

// Logout 用户登出（客户端清除 token 即可，服务端预留 token 黑名单扩展点）
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// MVP 阶段：登出由客户端清除本地 token 实现
	// 后续可在此处将 token 加入 Redis 黑名单
	SuccessWithMessage(c, "Logout successful", nil)
}
