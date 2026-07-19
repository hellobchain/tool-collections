package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
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

type draftTask struct {
	ID           string
	UserID       string
	FileID       string
	FileName     string
	Requirements string
	TemplateText string
	Status       string
	Progress     int
	CurrentStep  string
	Content      string
	ChangeLog    string
	GeneratedAt  string
	Error        string
	CreatedAt    time.Time
}

var (
	draftTasks   = make(map[string]*draftTask)
	draftTasksMu sync.Mutex
)

func StartDraftGenerate(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		FileID       string `json:"file_id"`
		Requirements string `json:"requirements"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}
	if req.FileID == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请上传合同模板")
		return
	}
	if strings.TrimSpace(req.Requirements) == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请输入合同需求")
		return
	}

	var cf models.ContractFile
	if err := database.DB.Where("id = ? AND user_id = ?", req.FileID, userID).First(&cf).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "模板文件不存在")
		return
	}

	// Create DB record immediately so history can show real-time progress
	userUUID, _ := uuid.Parse(userID)
	draftRecord := models.ContractDraft{
		UserID:       userUUID,
		FileName:     cf.FileName,
		FileID:       req.FileID,
		Requirements: req.Requirements,
		Status:       constants.ContractDraftStatusGenerating,
		Progress:     0,
	}
	if err := database.DB.Create(&draftRecord).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建草稿记录失败")
		return
	}

	task := &draftTask{
		ID:           draftRecord.ID.String(),
		UserID:       userID,
		FileID:       req.FileID,
		FileName:     cf.FileName,
		Requirements: req.Requirements,
		Status:       constants.ContractDraftStatusGenerating,
		CreatedAt:    time.Now(),
	}

	draftTasksMu.Lock()
	draftTasks[task.ID] = task
	draftTasksMu.Unlock()

	go runDraftAgent(task)

	utils.Success(c, gin.H{"task_id": task.ID})
}

func GetDraftProgress(c *gin.Context) {
	taskID := c.Param("taskId")

	// Try in-memory task first (real-time during generation)
	draftTasksMu.Lock()
	task, ok := draftTasks[taskID]
	draftTasksMu.Unlock()

	if ok {
		utils.Success(c, gin.H{
			"percent":      task.Progress,
			"current_step": task.CurrentStep,
			"status":       task.Status,
		})
		return
	}

	// Fallback to DB (for completed/failed/historical)
	var d models.ContractDraft
	if err := database.DB.Where("id = ?", taskID).First(&d).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}

	utils.Success(c, gin.H{
		"percent":      d.Progress,
		"current_step": constants.ContractDraftStatusPendingDesc,
		"status":       d.Status,
	})
}

func GetDraftResult(c *gin.Context) {
	taskID := c.Param("taskId")
	draftTasksMu.Lock()
	task, ok := draftTasks[taskID]
	draftTasksMu.Unlock()

	if !ok {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}
	if task.Status != constants.ContractDraftStatusCompleted && task.Status != constants.ContractDraftStatusFailed {
		utils.ErrorWithMsg(c, utils.CodeError, "任务尚未完成")
		return
	}

	utils.Success(c, gin.H{
		"id":           task.ID,
		"content":      task.Content,
		"change_log":   task.ChangeLog,
		"generated_at": task.GeneratedAt,
		"file_name":    task.FileName,
	})
}

func DownloadDraft(c *gin.Context) {
	taskID := c.Param("taskId")
	draftTasksMu.Lock()
	task, ok := draftTasks[taskID]
	draftTasksMu.Unlock()

	if !ok {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}
	if task.Status != constants.ContractDraftStatusCompleted && task.Status != constants.ContractDraftStatusFailed {
		utils.ErrorWithMsg(c, utils.CodeError, "任务尚未完成")
		return
	}

	content := fmt.Sprintf("合同草案\n==============================\n\n生成时间：%s\n模板：%s\n\n%s\n\n条款变更说明\n==============================\n\n%s",
		task.GeneratedAt, task.FileName, task.Content, task.ChangeLog)

	trueFileName := fmt.Sprintf("合同草案_%s.docx", strings.TrimSuffix(task.FileName, ".docx"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%s`, utils.PercentEncode(trueFileName)))
	c.Data(200, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte(content))
}

// ===== 历史记录 =====

func GetDraftHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "15"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 15
	}

	var total int64
	database.DB.Model(&models.ContractDraft{}).Where("user_id = ?", userUUID).Count(&total)

	var drafts []models.ContractDraft
	database.DB.Where("user_id = ?", userUUID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&drafts)

	list := make([]gin.H, 0)
	for _, d := range drafts {
		list = append(list, gin.H{
			"id":           d.ID.String(),
			"file_name":    d.FileName,
			"requirements": truncateText(d.Requirements, 100),
			"generated_at": d.GeneratedAt.Format("2006-01-02 15:04"),
			"content_len":  len(d.Content),
			"status":       d.Status,
			"progress":     d.Progress,
		})
	}

	utils.SuccessPage(c, list, total, page, pageSize)
}

func GetDraftDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	draftID := c.Param("draftId")
	var d models.ContractDraft
	if err := database.DB.Where("id = ? AND user_id = ?", draftID, userUUID).First(&d).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "记录不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":           d.ID.String(),
		"file_name":    d.FileName,
		"requirements": d.Requirements,
		"content":      d.Content,
		"change_log":   d.ChangeLog,
		"generated_at": d.GeneratedAt.Format("2006-01-02 15:04"),
		"status":       d.Status,
		"progress":     d.Progress,
	})
}

func DeleteDraft(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	draftID := c.Param("draftId")
	result := database.DB.Where("id = ? AND user_id = ?", draftID, userUUID).Delete(&models.ContractDraft{})
	if result.RowsAffected == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "记录不存在")
		return
	}

	utils.SuccessWithMsg(c, nil, "删除成功")
}

func truncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// runDraftAgent runs the LLM agent pipeline for contract drafting
func runDraftAgent(task *draftTask) {
	draftTasksMu.Lock()
	task.Status = constants.ContractDraftStatusGenerating
	draftTasksMu.Unlock()

	llm := services.NewLLMService()

	updateDB := func(pct int, status, step string) {
		draftTasksMu.Lock()
		task.Progress = pct
		task.Status = status
		task.CurrentStep = step
		draftTasksMu.Unlock()
		userUUID, _ := uuid.Parse(task.UserID)
		database.DB.Model(&models.ContractDraft{}).Where("id = ? AND user_id = ?", task.ID, userUUID).Updates(map[string]interface{}{
			"progress": pct,
			"status":   status,
		})
	}

	// Step 1: Load template text
	setStep := updateDB
	setStep(5, constants.ContractDraftStatusGeneratingDesc, "正在读取模板文件...")
	var cf models.ContractFile
	if err := database.DB.Where("id = ?", task.FileID).First(&cf).Error; err != nil {
		failTask(task, "模板文件读取失败")
		return
	}

	templateText := ""
	if cf.FileSavePath != "" {
		data, err := services.DownloadContractFile(context.Background(), cf.FileSavePath)
		if err == nil {
			templateText = extractText(cf.FileName, data)
		}
	}
	if templateText == "" {
		failTask(task, "无法读取模板内容")
		return
	}
	task.TemplateText = templateText

	// Step 2: Analyze template structure
	setStep(15, constants.ContractDraftStatusGeneratingDesc, "正在分析模板结构...")
	analysisPrompt := fmt.Sprintf(`你是一个合同模板分析专家。请分析以下合同模板，提取其结构信息。

合同模板内容：
%s

请分析并输出JSON格式结果，包含以下字段：
1. title: 合同标题
2. sections: 章节/条款列表，每个包含 section_name(条款名称)和content(原文内容)
3. variables: 模板中可替换的变量列表，每个包含 name(变量名)和 description(变量说明)
4. optional_clauses: 可选条款列表（如保密、违约责任方式等）

只输出JSON，不要其他文字。`, templateText)

	analysisResult, err := llm.GenerateLlmWithPrompt("你是一个合同模板分析专家。请准确分析合同模板结构。", analysisPrompt)
	if err != nil {
		failTask(task, fmt.Sprintf("模板分析失败: %v", err))
		return
	}
	analysisResult = cleanJSON(analysisResult)

	// Step 3: Understand requirements
	setStep(35, constants.ContractDraftStatusGeneratingDesc, "正在理解用户需求...")
	reqPrompt := fmt.Sprintf(`你是一个合同需求分析专家。请将用户的自然语言需求转化为结构化参数，用于填充合同模板。

用户需求：
%s

模板分析结果：
%s

请输出JSON格式的结构化参数，包含合同双方信息、金额、付款条款、交付条款、特殊要求等。
只输出JSON，不要其他文字。`, task.Requirements, analysisResult)

	paramsResult, err := llm.GenerateLlmWithPrompt("你是一个合同需求分析专家。请准确提取合同参数。", reqPrompt)
	if err != nil {
		failTask(task, fmt.Sprintf("需求解析失败: %v", err))
		return
	}
	paramsResult = cleanJSON(paramsResult)

	// Step 4: Generate draft
	setStep(55, constants.ContractDraftStatusGeneratingDesc, "正在生成合同草案...")
	genPrompt := fmt.Sprintf(`你是一个合同起草专家。请根据以下信息生成一份完整的合同草案。

模板原文：
%s

用户需求：
%s

结构化参数：
%s

要求：
1. 基于模板结构，将参数填充到对应位置
2. 根据用户需求，必要时增删或改写条款
3. 对缺失的必要条款进行智能补全
4. 对可通过 %s 的违约金条款附加风险提示
5. 输出格式需保留清晰的章节和条款编号

请直接输出完整的合同草案文本。`, templateText, task.Requirements, paramsResult, "超过30%")

	draftContent, err := llm.GenerateLlmWithPrompt("你是一个专业的合同起草专家，精通中国民法典及相关法律法规。", genPrompt)
	if err != nil {
		failTask(task, fmt.Sprintf("合同生成失败: %v", err))
		return
	}

	// Step 5: Generate change log
	setStep(80, constants.ContractDraftStatusGeneratingDesc, "正在生成条款变更说明...")
	clPrompt := fmt.Sprintf(`你是一个合同变更分析专家。请对比原始模板和生成的草案，生成一份条款变更说明。

模板原文：
%s

生成的草案：
%s

请分析并输出变更说明，列出：
1. 新增的条款
2. 删除的条款（如有）
3. 修改的条款
4. 填充的变量值
5. 风险提示

按以下格式输出（Markdown）：
## 条款变更说明
### 新增条款
...
### 修改条款
...
### 填充变量
...
### 风险提示
...`, templateText, draftContent)

	changeLog, err := llm.GenerateLlmWithPrompt("你是一个合同变更分析专家。", clPrompt)
	if err != nil {
		changeLog = "（变更说明生成失败）"
	}

	generatedAt := time.Now()
	draftTasksMu.Lock()
	task.Status = constants.ContractDraftStatusCompleted
	task.Progress = 100
	task.CurrentStep = constants.ContractDraftStatusCompletedDesc
	task.Content = draftContent
	task.ChangeLog = changeLog
	task.GeneratedAt = generatedAt.Format("2006-01-02 15:04")
	draftTasksMu.Unlock()

	// Update DB record (already created in StartDraftGenerate)
	userUUID, _ := uuid.Parse(task.UserID)
	database.DB.Model(&models.ContractDraft{}).Where("id = ? AND user_id = ?", task.ID, userUUID).Updates(map[string]interface{}{
		"content":      draftContent,
		"change_log":   changeLog,
		"status":       constants.ContractDraftStatusCompleted,
		"progress":     100,
		"generated_at": generatedAt,
	})

	slog.Infof("[DraftAgent] Task %s completed: %d chars draft, %d chars changelog", task.ID, len(draftContent), len(changeLog))
}

func failTask(task *draftTask, errMsg string) {
	draftTasksMu.Lock()
	task.Status = constants.ContractDraftStatusFailed
	task.Error = errMsg
	task.Progress = 0
	draftTasksMu.Unlock()

	// Update DB
	userUUID, _ := uuid.Parse(task.UserID)
	database.DB.Model(&models.ContractDraft{}).Where("id = ? AND user_id = ?", task.ID, userUUID).Updates(map[string]interface{}{
		"status":   constants.ContractDraftStatusFailed,
		"progress": 0,
	})

	slog.Errorf("[DraftAgent] Task %s failed: %s", task.ID, errMsg)
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
