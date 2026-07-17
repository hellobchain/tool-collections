package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/auth"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorWithMsg(c, utils.CodeUnauthorized, "未提供token")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorWithMsg(c, utils.CodeUnauthorized, "token格式错误")
			c.Abort()
			return
		}

		userID, err := auth.ParseToken(parts[1])
		if err != nil {
			utils.ErrorWithMsg(c, utils.CodeUnauthorized, "无效或过期的token")
			c.Abort()
			return
		}

		// 查询用户
		var user models.User
		if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
			utils.ErrorWithMsg(c, utils.CodeUnauthorized, "用户不存在")
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Set("user", user)
		c.Next()
	}
}

// GetUserID 从context获取当前用户ID
func GetUserID(c *gin.Context) string {
	return c.GetString("user_id")
}
