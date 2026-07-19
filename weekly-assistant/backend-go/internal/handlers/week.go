package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/constants"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func resolveWeekStart(reqWeekStart string) time.Time {
	if reqWeekStart != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, reqWeekStart); err == nil {
			return services.GetWeekStart(t)
		}
	}
	return services.GetWeekStart(time.Now())
}

func buildSkillsContext(userID string) string {
	userUUID, _ := uuid.Parse(userID)
	var skills []models.Skill
	database.DB.Where("user_id = ? AND is_active = ?", userUUID, true).
		Order("sort_order ASC").Find(&skills)
	if len(skills) == 0 {
		return ""
	}
	b := strings.Builder{}
	b.WriteString("\n\n【个人技能/角色】\n")
	for _, s := range skills {
		b.WriteString(fmt.Sprintf("- %s：%s\n", s.Name, s.Description))
	}
	b.WriteString("请将这些技能/角色融入到周报内容中，体现个人技术特长和职责定位。")
	return b.String()
}

// === 共享草稿准备逻辑 ===

type draftContext struct {
	fragments  []map[string]interface{}
	carryover  []map[string]interface{}
	template   models.PromptTemplate
	userPrompt string
	weekStart  time.Time
}

func prepareDraft(c *gin.Context, req models.GenerateDraftRequest) (*draftContext, bool) {
	userID := middleware.GetUserID(c)
	weekStart := resolveWeekStart(req.WeekStart)

	if req.NarrativeType == "" {
		req.NarrativeType = "攻坚"
	}

	var template models.PromptTemplate
	if req.TemplateID != "" {
		err := database.DB.Where("id = ? AND (user_id = ? OR user_id IS NULL)", req.TemplateID, userID).
			First(&template).Error
		if err != nil {
			utils.ErrorWithMsg(c, utils.CodeInvalidParams, "指定的模板不存在")
			return nil, false
		}
	} else {
		err := database.DB.Where("user_id IS NULL AND category = 'default'").
			Order("sort_order ASC").
			First(&template).Error
		if err != nil {
			template = models.PromptTemplate{
				SystemPrompt:       constants.DefaultWeeklyDraftSystemPrompt,
				UserPromptTemplate: constants.DefaultWeeklyDraftUserPrompt,
			}
		}
	}

	var fragments []models.Fragment
	if err := database.DB.Where("user_id = ? AND week_start = ?", userID, weekStart).
		Order("occurred_at ASC").
		Find(&fragments).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "查询碎片失败")
		return nil, false
	}
	if len(fragments) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "没有碎片，无法生成草稿")
		return nil, false
	}
	fragMaps := make([]map[string]interface{}, len(fragments))
	for i, f := range fragments {
		fragMaps[i] = map[string]interface{}{"content": f.Content}
	}

	lastWeek := weekStart.AddDate(0, 0, -7)
	var lastReport models.WeeklyReport
	carryover := []map[string]interface{}{}
	if database.DB.Where("user_id = ? AND week_start = ?", userID, lastWeek).
		First(&lastReport).Error == nil {
		var items []models.CarryoverResponse
		if err := json.Unmarshal([]byte(lastReport.CarryoverFromPrev), &items); err != nil {
			slog.Errorf("解析继承数据失败: %v", err)
		}
		for _, item := range items {
			carryover = append(carryover, map[string]interface{}{
				"id":      item.ID,
				"content": item.Content,
			})
		}
	}

	engine := services.NewPromptEngine()
	userPrompt := engine.RenderPrompt(template.UserPromptTemplate, fragMaps, carryover, req.NarrativeType)

	return &draftContext{
		fragments:  fragMaps,
		carryover:  carryover,
		template:   template,
		userPrompt: userPrompt,
		weekStart:  weekStart,
	}, true
}

// GetWeekStatus 获取本周状态
func GetWeekStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	weekStart := resolveWeekStart(c.Query("week_start"))

	var fragments []models.Fragment
	if err := database.DB.Where("user_id = ? AND week_start = ?", userID, weekStart).
		Order("occurred_at ASC").
		Find(&fragments).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "查询碎片失败")
		return
	}

	var report models.WeeklyReport
	isFinalized := database.DB.Where("user_id = ? AND week_start = ?", userID, weekStart).
		First(&report).Error == nil

	nextWeekPlan := []models.CarryoverResponse{}
	if isFinalized && report.NextWeekPlan != "" {
		if err := json.Unmarshal([]byte(report.NextWeekPlan), &nextWeekPlan); err != nil {
			slog.Errorf("解析下周计划失败: %v", err)
		}
	}

	lastWeek := weekStart.AddDate(0, 0, -7)
	var lastReport models.WeeklyReport
	carryover := []models.CarryoverResponse{}
	if database.DB.Where("user_id = ? AND week_start = ?", userID, lastWeek).
		First(&lastReport).Error == nil {
		if err := json.Unmarshal([]byte(lastReport.CarryoverFromPrev), &carryover); err != nil {
			slog.Errorf("解析继承数据失败: %v", err)
		}
	}

	var carriedCount int64
	if err := database.DB.Model(&models.Fragment{}).
		Where("user_id = ? AND week_start = ? AND is_carried = true", userID, weekStart).
		Count(&carriedCount).Error; err != nil {
		slog.Errorf("查询已继承数量失败: %v", err)
	}
	hasLastReport := database.DB.Where("user_id = ? AND week_start = ?", userID, lastWeek).
		First(&lastReport).Error == nil
	isConfirmed := !hasLastReport || carriedCount > 0

	fragResponses := make([]models.FragmentResponse, len(fragments))
	for i, f := range fragments {
		dateStr := ""
		if f.OccurredAt != nil {
			dateStr = services.FormatDate(*f.OccurredAt)
		}
		fragResponses[i] = models.FragmentResponse{
			ID:         f.ID.String(),
			Content:    f.Content,
			Date:       dateStr,
			OccurredAt: f.OccurredAt,
			IsCarried:  f.IsCarried,
		}
	}

	utils.Success(c, gin.H{
		"week_start":             services.FormatDate(weekStart),
		"week_end":               services.FormatDate(services.GetWeekEnd(weekStart)),
		"week_number":            services.GetISOWeekNumber(weekStart),
		"fragments":              fragResponses,
		"carryover":              carryover,
		"is_carryover_confirmed": isConfirmed,
		"next_week_plan":         nextWeekPlan,
		"is_finalized":           isFinalized,
	})
}

func ConfirmCarryover(c *gin.Context) {
	var req models.ConfirmCarryoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	weekStart := resolveWeekStart("")
	lastWeek := weekStart.AddDate(0, 0, -7)

	var lastReport models.WeeklyReport
	if database.DB.Where("user_id = ? AND week_start = ?", userID, lastWeek).
		First(&lastReport).Error != nil {
		utils.SuccessWithMsg(c, nil, "无继承事项")
		return
	}

	var carryoverItems []models.CarryoverResponse
	if err := json.Unmarshal([]byte(lastReport.CarryoverFromPrev), &carryoverItems); err != nil {
		slog.Errorf("解析继承数据失败: %v", err)
	}

	keptMap := make(map[string]bool, len(req.KeptIDs))
	for _, id := range req.KeptIDs {
		keptMap[id] = true
	}

	for _, item := range carryoverItems {
		if keptMap[item.ID] {
			userUUID, err := parseUUID(userID)
			if err != nil {
				slog.Errorf("解析用户ID失败: %v", err)
				continue
			}
			fragment := models.Fragment{
				UserID:    userUUID,
				WeekStart: weekStart,
				Content:   item.Content,
				Source:    "carryover",
				IsCarried: true,
			}
			if err := database.DB.Create(&fragment).Error; err != nil {
				slog.Errorf("创建继承碎片失败: %v", err)
			}
		}
	}

	utils.SuccessWithMsg(c, nil, "确认成功")
}

func GenerateDraft(c *gin.Context) {
	var req models.GenerateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	ctx, ok := prepareDraft(c, req)
	if !ok {
		return
	}

	userID := middleware.GetUserID(c)
	llm := services.NewLLMService()
	systemPrompt := ctx.template.SystemPrompt + buildSkillsContext(userID)
	draft, err := llm.GenerateLlmWithPrompt(systemPrompt, ctx.userPrompt)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "生成草稿失败")
		return
	}

	utils.Success(c, gin.H{
		"content":    draft,
		"week_start": services.FormatDate(ctx.weekStart),
	})
}

func GenerateDraftStream(c *gin.Context) {
	var req models.GenerateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	ctx, ok := prepareDraft(c, req)
	if !ok {
		return
	}

	userID := middleware.GetUserID(c)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch := make(chan string, 64)
	reqCtx := c.Request.Context()
	go func() {
		llm := services.NewLLMService()
		sysPrompt := ctx.template.SystemPrompt + buildSkillsContext(userID)
		select {
		case <-reqCtx.Done():
			return
		default:
			llm.GenerateDraftStream(sysPrompt, ctx.userPrompt, ch)
		}
	}()

	flusher, flushOk := c.Writer.(http.Flusher)
	done := false
	for !done {
		select {
		case chunk, ok := <-ch:
			if !ok {
				done = true
				continue
			}
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
			if flushOk {
				flusher.Flush()
			}
		case <-reqCtx.Done():
			done = true
		}
	}
	_, _ = fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	if flushOk {
		flusher.Flush()
	}
}

func GetWeekHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.HistoryQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 200 {
		req.PageSize = 20
	}

	query := database.DB.Where("user_id = ?", userID)

	if req.WeekStart != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, req.WeekStart); err == nil {
			query = query.Where("week_start >= ?", t)
		}
	}
	if req.WeekEnd != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, req.WeekEnd); err == nil {
			query = query.Where("week_start <= ?", t)
		}
	}

	var total int64
	query.Model(&models.WeeklyReport{}).Count(&total)

	var reports []models.WeeklyReport
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("week_start DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&reports).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "查询历史失败")
		return
	}

	items := make([]models.HistoryItemResponse, len(reports))
	for i, r := range reports {
		items[i] = models.HistoryItemResponse{
			ID:            r.ID.String(),
			WeekStart:     services.FormatDate(r.WeekStart),
			Content:       r.Content,
			NarrativeType: r.NarrativeType,
			CreatedAt:     r.CreatedAt.Format(constants.DateFormatTimeHHMMSS),
		}
	}

	utils.SuccessPage(c, items, total, req.Page, req.PageSize)
}

func FinalizeWeek(c *gin.Context) {
	var req models.FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	weekStart := resolveWeekStart(req.WeekStart)
	lastWeek := weekStart.AddDate(0, 0, -7)
	carryoverFromFragments := []models.CarryoverResponse{}
	var unfinalizedFragments []models.Fragment
	if err := database.DB.Where("user_id = ? AND week_start = ? AND is_carried = ?", userID, lastWeek, true).
		Find(&unfinalizedFragments).Error; err != nil {
		slog.Errorf("查询未继承碎片失败: %v", err)
	}
	for _, f := range unfinalizedFragments {
		carryoverFromFragments = append(carryoverFromFragments, models.CarryoverResponse{
			ID:      f.ID.String(),
			Content: f.Content,
		})
	}
	carryoverJSON, _ := json.Marshal(carryoverFromFragments)

	llm := services.NewLLMService()
	nextPlans := llm.ExtractNextWeekPlan(services.GetISOWeekNumber(weekStart), req.Content)
	planJSON, _ := json.Marshal(nextPlans)

	narrativeType := req.NarrativeType
	if narrativeType == "" {
		narrativeType = "攻坚"
	}

	var existing models.WeeklyReport
	result := database.DB.Where("user_id = ? AND week_start = ?", userID, weekStart).
		First(&existing)

	if result.Error == nil {
		existing.Content = req.Content
		existing.NarrativeType = narrativeType
		existing.CarryoverFromPrev = string(carryoverJSON)
		existing.NextWeekPlan = string(planJSON)
		if err := database.DB.Save(&existing).Error; err != nil {
			utils.ErrorWithMsg(c, utils.CodeServerError, "归档失败")
			return
		}
	} else {
		userUUID, err := parseUUID(userID)
		if err != nil {
			utils.ErrorWithMsg(c, utils.CodeServerError, "无效的用户ID")
			return
		}
		report := models.WeeklyReport{
			UserID:            userUUID,
			WeekStart:         weekStart,
			Content:           req.Content,
			NarrativeType:     narrativeType,
			CarryoverFromPrev: string(carryoverJSON),
			NextWeekPlan:      string(planJSON),
		}
		if err := database.DB.Create(&report).Error; err != nil {
			utils.ErrorWithMsg(c, utils.CodeServerError, "归档失败")
			return
		}
	}

	utils.SuccessWithMsg(c, nil, "归档成功")
}

func ExportWeekHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	weekStart := c.Query("week_start")
	weekEnd := c.Query("week_end")

	query := database.DB.Where("user_id = ?", userID)

	if weekStart != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, weekStart); err == nil {
			query = query.Where("week_start >= ?", t)
		}
	}
	if weekEnd != "" {
		if t, err := services.ParseDate(constants.DateFormatDate, weekEnd); err == nil {
			query = query.Where("week_start <= ?", t)
		}
	}

	var reports []models.WeeklyReport
	if err := query.Order("week_start DESC").Limit(200).Find(&reports).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "查询历史失败")
		return
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "周次")
	f.SetCellValue(sheet, "B1", "开始日期")
	f.SetCellValue(sheet, "C1", "叙事类型")
	f.SetCellValue(sheet, "D1", "周报内容")
	f.SetCellValue(sheet, "E1", "归档时间")

	for i, r := range reports {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), services.GetISOWeekNumber(r.WeekStart))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), services.FormatDate(r.WeekStart))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.NarrativeType)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.Content)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.CreatedAt.Format(constants.DateFormatTimeHHMMSS))
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=weekly_reports.xlsx")
	if err := f.Write(c.Writer); err != nil {
		slog.Errorf("导出周报 Excel 失败: %v", err)
	}
}

func DeleteWeekReport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.WeeklyReport{})
	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "周报不存在")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}

func ListSummaries(c *gin.Context) {
	userID := middleware.GetUserID(c)
	query := database.DB.Where("user_id = ?", userID)
	summaryType := c.Query("type")
	if summaryType == "quarter" || summaryType == "year" {
		query = query.Where("period_type = ?", summaryType)
	}
	var summaries []models.Summary
	if err := query.Order("created_at DESC").Find(&summaries).Error; err != nil {
		slog.Errorf("查询汇总列表失败: %v", err)
	}

	type summaryItem struct {
		ID          string `json:"id"`
		PeriodType  string `json:"period_type"`
		PeriodValue string `json:"period_value"`
		Content     string `json:"content"`
		CreatedAt   string `json:"created_at"`
	}
	quarter := make([]summaryItem, 0)
	year := make([]summaryItem, 0)
	for _, s := range summaries {
		item := summaryItem{
			ID:          s.ID.String(),
			PeriodType:  s.PeriodType,
			PeriodValue: s.PeriodValue,
			Content:     s.Content,
			CreatedAt:   s.CreatedAt.Format(constants.DateFormatTimeHHMMSS),
		}
		if s.PeriodType == "quarter" {
			quarter = append(quarter, item)
		} else {
			year = append(year, item)
		}
	}

	utils.Success(c, gin.H{"quarter": quarter, "year": year})
}

func GenerateSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	periodType := c.Query("type")
	periodValue := c.Query("value")

	if periodType == "" || periodValue == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "缺少 type 或 value 参数")
		return
	}

	var periodLabel string
	var startDate, endDate string

	switch periodType {
	case "quarter":
		yr := periodValue[:4]
		q := periodValue[5:]
		switch q {
		case "Q1":
			startDate = yr + "-01-01"
			endDate = yr + "-03-31"
		case "Q2":
			startDate = yr + "-04-01"
			endDate = yr + "-06-30"
		case "Q3":
			startDate = yr + "-07-01"
			endDate = yr + "-09-30"
		case "Q4":
			startDate = yr + "-10-01"
			endDate = yr + "-12-31"
		default:
			utils.ErrorWithMsg(c, utils.CodeInvalidParams, "非法季度值")
			return
		}
		periodLabel = periodValue
	case "year":
		startDate = periodValue + "-01-01"
		endDate = periodValue + "-12-31"
		periodLabel = periodValue + "年度"
	default:
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "type 只能是 quarter 或 year")
		return
	}

	start, _ := services.ParseDate(constants.DateFormatDate, startDate)
	end, _ := services.ParseDate(constants.DateFormatDate, endDate)

	var reports []models.WeeklyReport
	if err := database.DB.Where("user_id = ? AND week_start >= ? AND week_start <= ?", userID, start, end).
		Order("week_start ASC").
		Find(&reports).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "查询失败")
		return
	}

	if len(reports) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "该周期内暂无周报记录")
		return
	}

	reportMaps := make([]map[string]string, len(reports))
	for i, r := range reports {
		reportMaps[i] = map[string]string{
			"week_start": services.FormatDate(r.WeekStart),
			"content":    r.Content,
		}
	}

	llm := services.NewLLMService()
	summary, err := llm.GenerateSummary(periodLabel, reportMaps)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "生成失败")
		return
	}

	var existing models.Summary
	if err := database.DB.Where("user_id = ? AND period_type = ? AND period_value = ?", userID, periodType, periodValue).
		First(&existing).Error; err == nil {
		existing.Content = summary
		if err := database.DB.Save(&existing).Error; err != nil {
			slog.Errorf("保存汇总失败: %v", err)
		}
	} else {
		userUUID, err := parseUUID(userID)
		if err != nil {
			utils.ErrorWithMsg(c, utils.CodeServerError, "无效的用户ID")
			return
		}
		if err := database.DB.Create(&models.Summary{
			UserID:      userUUID,
			PeriodType:  periodType,
			PeriodValue: periodValue,
			Content:     summary,
		}).Error; err != nil {
			slog.Errorf("创建汇总失败: %v", err)
		}
	}

	utils.Success(c, gin.H{
		"summary":    summary,
		"total":      len(reports),
		"period":     periodLabel,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

func ExportSummariesHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	query := database.DB.Where("user_id = ?", userID)
	summaryType := c.Query("type")
	if summaryType != "quarter" && summaryType != "year" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "type 只能是 quarter 或 year")
		return
	}
	query = query.Where("period_type = ?", summaryType)

	var summaries []models.Summary
	if err := query.Order("created_at DESC").Find(&summaries).Error; err != nil {
		slog.Errorf("查询汇总历史失败: %v", err)
	}

	fileName := ""
	f := excelize.NewFile()
	sheet := "Sheet1"
	if summaryType == "quarter" {
		f.SetCellValue(sheet, "A1", "季度")
		f.SetCellValue(sheet, "B1", "季度报内容")
		fileName = "quarter_history.xlsx"
	} else {
		f.SetCellValue(sheet, "A1", "年度")
		f.SetCellValue(sheet, "B1", "年度报内容")
		fileName = "year_history.xlsx"
	}
	f.SetCellValue(sheet, "C1", "归档时间")

	for i, s := range summaries {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), s.PeriodValue)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), s.Content)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), s.CreatedAt.Format(constants.DateFormatTimeHHMMSS))
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	if err := f.Write(c.Writer); err != nil {
		slog.Errorf("导出汇总 Excel 失败: %v", err)
	}
}

func DeleteSummaryReport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Summary{})
	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "季度/年度报不存在")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}
