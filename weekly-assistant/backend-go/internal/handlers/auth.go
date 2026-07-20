package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/auth"
	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func Register(c *gin.Context) {
	// 注册关闭
	if config.AppConfig.CloseRegister {
		utils.ErrorWithMsg(c, utils.CodeServerError, "注册已关闭")
		return
	}
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	var existing models.User
	if err := database.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "用户名已存在")
		return
	}

	hashedPwd, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "密码加密失败")
		return
	}

	user := models.User{
		Username:     req.Username,
		PasswordHash: hashedPwd,
		Email:        req.Email,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建用户失败")
		return
	}

	utils.Success(c, gin.H{"user_id": user.ID.String()})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeUnauthorized, "用户名或密码错误")
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		utils.ErrorWithMsg(c, utils.CodeUnauthorized, "用户名或密码错误")
		return
	}

	token, err := auth.GenerateToken(user.ID.String())
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "生成token失败")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID.String())
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "生成 refresh token 失败")
		return
	}

	utils.Success(c, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":        user.ID.String(),
			"username":  user.Username,
			"full_name": user.FullName,
			"email":     user.Email,
		},
	})
}

func ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "用户不存在")
		return
	}

	if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "原密码错误")
		return
	}

	hashedPwd, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "密码加密失败")
		return
	}

	if err := database.DB.Model(&user).Update("password_hash", hashedPwd).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "修改密码失败")
		return
	}

	utils.SuccessWithMsg(c, nil, "修改成功")
}

func RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID, err := auth.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeUnauthorized, "刷新令牌无效")
		return
	}

	token, err := auth.GenerateToken(userID)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "生成token失败")
		return
	}

	newRefresh, err := auth.GenerateRefreshToken(userID)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "生成 refresh token 失败")
		return
	}
	utils.Success(c, models.TokenResponse{
		Token:        token,
		RefreshToken: newRefresh,
	})
}
