package handlers

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func GetPromptTemplates(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "0"))
	promptType := c.DefaultQuery("prompt_type", "")

	query := database.DB.Where("user_id = ? OR user_id IS NULL", userUUID)

	var total int64
	query.Model(&models.PromptTemplate{}).Count(&total)

	if pageSize > 0 {
		query = query.Order("sort_order ASC, created_at ASC").
			Offset((page - 1) * pageSize).
			Limit(pageSize)
	} else {
		query = query.Order("sort_order ASC, created_at ASC")
	}

	if promptType != "" {
		query = query.Where("prompt_type = ?", promptType)
	}

	var templates []models.PromptTemplate
	if err := query.Find(&templates).Error; err != nil {
		log.Printf("查询模板列表失败: %v", err)
	}

	responses := []models.PromptTemplateResponse{}
	for _, t := range templates {
		isSystem := t.UserID == nil
		responses = append(responses, models.PromptTemplateResponse{
			ID:                 t.ID.String(),
			Name:               t.Name,
			Category:           t.Category,
			PromptType:         t.PromptType,
			SystemPrompt:       t.SystemPrompt,
			UserPromptTemplate: t.UserPromptTemplate,
			Description:        t.Description,
			IsActive:           t.IsActive,
			SortOrder:          t.SortOrder,
			IsSystem:           isSystem,
		})
	}

	utils.SuccessList(c, responses, total)
}

func GetPromptTemplate(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	var template models.PromptTemplate
	err := database.DB.Where("id = ? AND (user_id = ? OR user_id IS NULL)", id, userUUID).
		First(&template).Error

	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "模板不存在")
		return
	}

	utils.Success(c, models.PromptTemplateResponse{
		ID:                 template.ID.String(),
		Name:               template.Name,
		Category:           template.Category,
		PromptType:         template.PromptType,
		SystemPrompt:       template.SystemPrompt,
		UserPromptTemplate: template.UserPromptTemplate,
		Description:        template.Description,
		IsActive:           template.IsActive,
		SortOrder:          template.SortOrder,
		IsSystem:           template.UserID == nil,
	})
}

func CreatePromptTemplate(c *gin.Context) {
	var req models.CreatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	promptType := req.PromptType
	if promptType == "" {
		promptType = "weekly"
	}

	template := models.PromptTemplate{
		UserID:             &userUUID,
		Name:               req.Name,
		Category:           "custom",
		PromptType:         promptType,
		SystemPrompt:       req.SystemPrompt,
		UserPromptTemplate: req.UserPromptTemplate,
		Description:        req.Description,
		IsActive:           true,
		SortOrder:          100,
	}

	if err := database.DB.Create(&template).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建模板失败")
		return
	}

	utils.SuccessWithMsg(c, gin.H{"id": template.ID.String()}, "创建成功")
}

func UpdatePromptTemplate(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	var template models.PromptTemplate
	if err := database.DB.Where("id = ? AND user_id = ?", id, userUUID).First(&template).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "模板不存在或无权修改")
		return
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	if req.SystemPrompt != "" {
		template.SystemPrompt = req.SystemPrompt
	}
	if req.UserPromptTemplate != "" {
		template.UserPromptTemplate = req.UserPromptTemplate
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		template.SortOrder = *req.SortOrder
	}
	if req.PromptType != "" {
		template.PromptType = req.PromptType
	}

	if err := database.DB.Save(&template).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "更新失败")
		return
	}

	utils.SuccessWithMsg(c, nil, "更新成功")
}

func DeletePromptTemplate(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	result := database.DB.Where("id = ? AND user_id = ?", id, userUUID).
		Delete(&models.PromptTemplate{})

	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "模板不存在或无权删除")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}
