// server/internal/middleware/admin.go
package middleware

import (
	"net/http"

	"ai-curton/server/internal/model"

	"github.com/gin-gonic/gin"
)

// RequireAdmin 管理员权限中间件，实时查数据库确认 role == "admin"
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "User not authenticated",
			})
			return
		}

		var user model.User
		if err := model.DB.Select("role").First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Admin access required",
			})
			return
		}

		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Admin access required",
			})
			return
		}
		c.Next()
	}
}
