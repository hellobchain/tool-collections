package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func ListGitProjects(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "0"))
	keyword := c.Query("keyword")

	query := database.DB.Where("user_id = ?", userUUID)
	if keyword != "" {
		query = query.Where("project_name ILIKE ? OR project_id ILIKE ? OR base_url ILIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Model(&models.GitProject{}).Count(&total)

	var projects []models.GitProject
	if pageSize > 0 {
		query = query.Order("created_at DESC").
			Offset((page - 1) * pageSize).
			Limit(pageSize)
	} else {
		query = query.Order("created_at DESC")
	}
	if err := query.Find(&projects).Error; err != nil {
		slog.Errorf("查询项目列表失败: %v", err)
	}

	items := []models.GitProjectResponse{}
	for _, p := range projects {
		items = append(items, models.GitProjectResponse{
			ID:          p.ID.String(),
			ProjectID:   p.ProjectID,
			ProjectName: p.ProjectName,
			BaseURL:     p.BaseURL,
			Token:       p.Token,
			Branch:      p.Branch,
		})
	}

	utils.SuccessList(c, items, total)
}

func CreateGitProject(c *gin.Context) {
	var req models.GitProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	branch := req.Branch
	if branch == "" {
		branch = "master"
	}

	project := models.GitProject{
		UserID:      userUUID,
		ProjectID:   req.ProjectID,
		ProjectName: req.ProjectName,
		BaseURL:     req.BaseURL,
		Token:       req.Token,
		Branch:      branch,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建项目失败")
		return
	}

	utils.SuccessWithMsg(c, models.GitProjectResponse{
		ID:          project.ID.String(),
		ProjectID:   project.ProjectID,
		ProjectName: project.ProjectName,
		BaseURL:     project.BaseURL,
		Token:       project.Token,
		Branch:      project.Branch,
	}, "创建成功")
}

func UpdateGitProject(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	var project models.GitProject
	if err := database.DB.Where("id = ? AND user_id = ?", id, userUUID).First(&project).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "项目不存在")
		return
	}

	var req models.GitProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	project.ProjectID = req.ProjectID
	project.ProjectName = req.ProjectName
	project.BaseURL = req.BaseURL
	project.Token = req.Token
	if req.Branch != "" {
		project.Branch = req.Branch
	}

	if err := database.DB.Save(&project).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "更新失败")
		return
	}
	utils.SuccessWithMsg(c, nil, "更新成功")
}

func DeleteGitProject(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	result := database.DB.Where("id = ? AND user_id = ?", id, userUUID).
		Delete(&models.GitProject{})

	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "项目不存在")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}
