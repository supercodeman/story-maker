// server/internal/middleware/permission.go
package middleware

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"

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
