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

func ListSkills(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "0"))

	query := database.DB.Where("user_id = ?", userUUID)

	var total int64
	query.Model(&models.Skill{}).Count(&total)

	var skills []models.Skill
	if pageSize > 0 {
		query = query.Order("sort_order ASC, created_at ASC").
			Offset((page - 1) * pageSize).
			Limit(pageSize)
	} else {
		query = query.Order("sort_order ASC, created_at ASC")
	}
	if err := query.Find(&skills).Error; err != nil {
		log.Printf("查询技能列表失败: %v", err)
	}

	list := []models.SkillResponse{}
	for _, s := range skills {
		list = append(list, models.SkillResponse{
			ID:          s.ID.String(),
			Name:        s.Name,
			Description: s.Description,
			IsActive:    s.IsActive,
			SortOrder:   s.SortOrder,
		})
	}

	utils.SuccessList(c, list, total)
}

func CreateSkill(c *gin.Context) {
	var req models.SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	skill := models.Skill{
		UserID:      userUUID,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		SortOrder:   0,
	}

	if err := database.DB.Create(&skill).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建技能失败")
		return
	}

	utils.SuccessWithMsg(c, models.SkillResponse{
		ID:          skill.ID.String(),
		Name:        skill.Name,
		Description: skill.Description,
		IsActive:    skill.IsActive,
		SortOrder:   skill.SortOrder,
	}, "创建成功")
}

func UpdateSkill(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	var skill models.Skill
	if err := database.DB.Where("id = ? AND user_id = ?", id, userUUID).First(&skill).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "技能不存在")
		return
	}

	var req models.SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	skill.Name = req.Name
	skill.Description = req.Description
	if req.IsActive != nil {
		skill.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		skill.SortOrder = *req.SortOrder
	}

	if err := database.DB.Save(&skill).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "更新失败")
		return
	}

	utils.SuccessWithMsg(c, nil, "更新成功")
}

func DeleteSkill(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	result := database.DB.Where("id = ? AND user_id = ?", id, userUUID).
		Delete(&models.Skill{})

	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "技能不存在")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}
