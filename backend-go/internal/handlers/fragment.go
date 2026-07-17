package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/constants"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func AddFragment(c *gin.Context) {
	var req models.AddFragmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	now := time.Now()
	if req.Date != "" {
		t, err := services.ParseDate(constants.DateFormatDate, req.Date)
		if err == nil {
			now = t
		}
	}
	weekStart := services.GetWeekStart(now)

	occurredAt := req.OccurredAt
	if occurredAt == nil {
		occurredAt = &now
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "无效的用户ID")
		return
	}
	fragment := models.Fragment{
		UserID:     userUUID,
		WeekStart:  weekStart,
		Content:    req.Content,
		Source:     "manual",
		OccurredAt: occurredAt,
		IsCarried:  false,
	}

	if err := database.DB.Create(&fragment).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "添加碎片失败")
		return
	}

	utils.Success(c, gin.H{
		"id":          fragment.ID.String(),
		"content":     fragment.Content,
		"date":        services.FormatDate(*occurredAt),
		"week_start":  services.FormatDate(weekStart),
		"occurred_at": fragment.OccurredAt,
		"is_carried":  fragment.IsCarried,
	})
}

func ListFragments(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.ListFragmentsQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	if req.Page < constants.DefaultPage {
		req.Page = constants.DefaultPage
	}
	if req.PageSize < constants.DefaultPageSize || req.PageSize > constants.FragmentMaxSize {
		req.PageSize = constants.FragmentPageSize
	}

	query := database.DB.Where("user_id = ?", userID)
	query = query.Where("week_start >= ?", services.GetWeekStart(time.Now()))

	if req.Date != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, req.Date); err == nil {
			weekStart := services.GetWeekStart(t)
			query = query.Where("week_start = ?", weekStart)
		}
	} else if req.WeekStart != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, req.WeekStart); err == nil {
			query = query.Where("week_start = ?", services.GetWeekStart(t))
		}
	}

	var total int64
	query.Model(&models.Fragment{}).Count(&total)

	var fragments []models.Fragment
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("occurred_at DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&fragments).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "查询碎片失败")
		return
	}

	list := []models.FragmentResponse{}
	for _, f := range fragments {
		dateStr := ""
		if f.OccurredAt != nil {
			dateStr = services.FormatDate(*f.OccurredAt)
		}
		list = append(list, models.FragmentResponse{
			ID:         f.ID.String(),
			Content:    f.Content,
			Date:       dateStr,
			OccurredAt: f.OccurredAt,
			IsCarried:  f.IsCarried,
		})
	}
	utils.SuccessPage(c, list, total, req.Page, req.PageSize)
}

func DeleteFragment(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Fragment{})
	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "碎片不存在")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
