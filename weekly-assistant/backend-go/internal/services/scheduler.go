package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/constants"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/wswlog/wlogging"
)

var slog = wlogging.MustGetLoggerWithoutName()

// StartAutoWeeklyScheduler 启动周报自动生成定时任务
// 根据配置的 CronSchedule 自动为所有用户执行：生成草稿 → 定稿归档
func StartAutoWeeklyScheduler() {
	if !config.AppConfig.SchedulerEnable {
		slog.Infof("auto weekly scheduler disabled")
		return
	}
	cronExpr := config.AppConfig.CronSchedule
	s := gocron.NewScheduler(time.Local)
	_, err := s.Cron(cronExpr).Do(processAllUsers)
	if err != nil {
		slog.Errorf("failed to register cron job: %v", err)
		return
	}
	s.StartAsync()
	slog.Infof("auto weekly scheduler started (cron: %s)", cronExpr)
}

func processAllUsers() {
	slog.Infof("starting auto weekly report generation")

	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		slog.Errorf("failed to list users: %v", err)
		return
	}

	for _, user := range users {
		userID := user.ID.String()
		if err := autoGenerateAndFinalize(userID); err != nil {
			slog.Errorf("user %s (%s): %v", userID, user.Username, err)
		}
	}
	slog.Infof("auto weekly report generation completed")
}

func autoGenerateAndFinalize(userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %v", err)
	}

	weekStart := GetWeekStart(time.Now())

	// 1. 检查该周是否已归档
	var existing models.WeeklyReport
	if err := database.DB.Where("user_id = ? AND week_start = ?", userUUID, weekStart).
		First(&existing).Error; err == nil {
		slog.Infof("user %s: week %s already finalized, skipping", userID, FormatDate(weekStart))
		return nil
	}

	// 2. 查询本周碎片
	var fragments []models.Fragment
	if err := database.DB.Where("user_id = ? AND week_start = ?", userUUID, weekStart).
		Order("occurred_at ASC").
		Find(&fragments).Error; err != nil {
		return fmt.Errorf("query fragments: %v", err)
	}
	if len(fragments) == 0 {
		slog.Infof("user %s: no fragments for week %s, skipping", userID, FormatDate(weekStart))
		return nil
	}

	fragMaps := make([]map[string]interface{}, len(fragments))
	for i, f := range fragments {
		fragMaps[i] = map[string]interface{}{"content": f.Content}
	}

	// 3. 获取上周报告的继承事项
	lastWeek := weekStart.AddDate(0, 0, -7)
	var lastReport models.WeeklyReport
	carryover := []map[string]interface{}{}
	if err := database.DB.Where("user_id = ? AND week_start = ?", userUUID, lastWeek).
		First(&lastReport).Error; err == nil {
		var items []models.CarryoverResponse
		if err := json.Unmarshal([]byte(lastReport.CarryoverFromPrev), &items); err == nil {
			for _, item := range items {
				carryover = append(carryover, map[string]interface{}{
					"id":      item.ID,
					"content": item.Content,
				})
			}
		}
	}

	// 4. 加载默认提示词模板
	var template models.PromptTemplate
	narrativeType := "稳健"
	if err := database.DB.Where("user_id IS NULL AND category = 'default'").
		Order("sort_order ASC").
		First(&template).Error; err != nil {
		template = models.PromptTemplate{
			SystemPrompt:       constants.DefaultWeeklyDraftSystemPrompt,
			UserPromptTemplate: constants.DefaultWeeklyDraftUserPrompt,
		}
	}

	// 5. 构建提示词
	engine := NewPromptEngine()
	userPrompt := engine.RenderPrompt(template.UserPromptTemplate, fragMaps, carryover, narrativeType)

	systemPrompt := template.SystemPrompt + buildSkillsContextForScheduler(userID)

	// 6. 调用 LLM 生成草稿
	llm := NewLLMService()
	draft, err := llm.GenerateLlmWithPrompt(systemPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("LLM generate draft: %v", err)
	}
	if strings.TrimSpace(draft) == "" {
		return fmt.Errorf("LLM returned empty draft")
	}

	slog.Infof("user %s: draft generated (%d chars)", userID, len(draft))

	// 7. 定稿归档
	carryoverFromFragments := []models.CarryoverResponse{}
	var unfinalizedFragments []models.Fragment
	if err := database.DB.Where("user_id = ? AND week_start = ? AND is_carried = ?", userUUID, lastWeek, true).
		Find(&unfinalizedFragments).Error; err != nil {
		slog.Errorf("query carryover fragments: %v", err)
	}
	for _, f := range unfinalizedFragments {
		carryoverFromFragments = append(carryoverFromFragments, models.CarryoverResponse{
			ID:      f.ID.String(),
			Content: f.Content,
		})
	}
	carryoverJSON, _ := json.Marshal(carryoverFromFragments)

	nextPlans := llm.ExtractNextWeekPlan(GetISOWeekNumber(weekStart), draft)
	planJSON, _ := json.Marshal(nextPlans)

	report := models.WeeklyReport{
		UserID:            userUUID,
		WeekStart:         weekStart,
		Content:           draft,
		NarrativeType:     narrativeType,
		CarryoverFromPrev: string(carryoverJSON),
		NextWeekPlan:      string(planJSON),
	}
	if err := database.DB.Create(&report).Error; err != nil {
		return fmt.Errorf("finalize report: %v", err)
	}

	slog.Infof("user %s: week %s auto-generated and archived successfully", userID, FormatDate(weekStart))
	return nil
}

// buildSkillsContextForScheduler 构建技能上下文（同 week.go 中逻辑，但使用 uuid.UUID）
func buildSkillsContextForScheduler(userID string) string {
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
