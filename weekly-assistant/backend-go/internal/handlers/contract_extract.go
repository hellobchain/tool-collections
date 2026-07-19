package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"

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

type extractTask struct {
	ID       string
	UserID   string
	Fields   []models.ExtractFieldConfig
	Files    []fileInfo
	Status   string
	Progress int
	Step     string
}

type fileInfo struct {
	ID   string
	Name string
}

var (
	extractTasks   = make(map[string]*extractTask)
	extractTasksMu sync.Mutex
)

func StartExtract(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		TaskName string                      `json:"task_name"`
		FileIDs  []string                    `json:"file_ids"`
		Fields   []models.ExtractFieldConfig `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}
	if len(req.FileIDs) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请上传合同文件")
		return
	}
	if len(req.Fields) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请配置提取字段")
		return
	}

	// Validate files
	var files []models.ContractFile
	database.DB.Where("id IN ? AND user_id = ?", req.FileIDs, userID).Find(&files)
	if len(files) == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件不存在")
		return
	}

	userUUID, _ := uuid.Parse(userID)
	fileIDsJSON, _ := json.Marshal(req.FileIDs)
	fileNames := make([]string, len(files))
	finfos := make([]fileInfo, len(files))
	for i, f := range files {
		fileNames[i] = f.FileName
		finfos[i] = fileInfo{ID: f.ID.String(), Name: f.FileName}
	}
	fileNamesJSON, _ := json.Marshal(fileNames)
	fieldsJSON, _ := json.Marshal(req.Fields)

	// Create task in DB
	task := models.ContractExtractTask{
		UserID:     userUUID,
		TaskName:   req.TaskName,
		FileIDs:    string(fileIDsJSON),
		FileNames:  string(fileNamesJSON),
		Fields:     string(fieldsJSON),
		Status:     constants.ContractExtractStatusExtracting,
		Progress:   0,
		TotalFiles: len(files),
	}
	if err := database.DB.Create(&task).Error; err != nil {
		slog.Errorf("[StartExtract] Failed to create task: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建任务失败")
		return
	}

	// Create result records for each file
	for _, f := range files {
		database.DB.Create(&models.ContractExtractResult{
			TaskID:   task.ID,
			FileID:   f.ID.String(),
			FileName: f.FileName,
			Status:   constants.ContractExtractStatusPending,
		})
	}

	taskID := task.ID.String()
	et := &extractTask{
		ID:     taskID,
		UserID: userID,
		Fields: req.Fields,
		Files:  finfos,
		Status: constants.ContractExtractStatusExtracting,
	}
	extractTasksMu.Lock()
	extractTasks[taskID] = et
	extractTasksMu.Unlock()

	go runExtractAgent(et, task.ID)

	utils.Success(c, gin.H{"task_id": taskID})
}

func GetExtractProgress(c *gin.Context) {
	taskID := c.Param("taskId")

	extractTasksMu.Lock()
	et, ok := extractTasks[taskID]
	extractTasksMu.Unlock()

	if ok {
		utils.Success(c, gin.H{
			"percent": et.Progress,
			"step":    et.Step,
			"status":  et.Status,
		})
		return
	}

	var t models.ContractExtractTask
	if err := database.DB.Where("id = ?", taskID).First(&t).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}
	utils.Success(c, gin.H{
		"percent": t.Progress,
		"step":    "",
		"status":  t.Status,
	})
}

func GetExtractResult(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	taskID := c.Param("taskId")
	var task models.ContractExtractTask
	if err := database.DB.Where("id = ? AND user_id = ?", taskID, userUUID).First(&task).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}

	var fields []models.ExtractFieldConfig
	json.Unmarshal([]byte(task.Fields), &fields)

	var results []models.ContractExtractResult
	database.DB.Where("task_id = ?", task.ID).Find(&results)

	resultList := make([]gin.H, 0)
	for _, r := range results {
		var data map[string]interface{}
		json.Unmarshal([]byte(r.Results), &data)
		resultList = append(resultList, gin.H{
			"id":        r.ID.String(),
			"file_id":   r.FileID,
			"file_name": r.FileName,
			"data":      data,
			"status":    r.Status,
			"error_msg": r.ErrorMsg,
		})
	}

	utils.Success(c, gin.H{
		"task_id":   task.ID.String(),
		"task_name": task.TaskName,
		"fields":    fields,
		"results":   resultList,
		"status":    task.Status,
		"progress":  task.Progress,
	})
}

func UpdateExtractCell(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	resultID := c.Param("resultId")
	var req struct {
		Field string      `json:"field"`
		Value interface{} `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	var result models.ContractExtractResult
	if err := database.DB.Where("id = ?", resultID).First(&result).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "结果不存在")
		return
	}

	// Verify task belongs to user
	var task models.ContractExtractTask
	if err := database.DB.Where("id = ? AND user_id = ?", result.TaskID, userUUID).First(&task).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}

	var data map[string]interface{}
	json.Unmarshal([]byte(result.Results), &data)
	if data == nil {
		data = make(map[string]interface{})
	}
	data[req.Field] = req.Value
	updated, _ := json.Marshal(data)
	database.DB.Model(&result).Update("results", string(updated))

	utils.SuccessWithMsg(c, nil, "更新成功")
}

func GetExtractHistory(c *gin.Context) {
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
	database.DB.Model(&models.ContractExtractTask{}).Where("user_id = ?", userUUID).Count(&total)

	var tasks []models.ContractExtractTask
	database.DB.Where("user_id = ?", userUUID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tasks)

	list := make([]gin.H, 0)
	for _, t := range tasks {
		var fileNames []string
		json.Unmarshal([]byte(t.FileNames), &fileNames)
		var fields []models.ExtractFieldConfig
		json.Unmarshal([]byte(t.Fields), &fields)
		list = append(list, gin.H{
			"id":          t.ID.String(),
			"task_name":   t.TaskName,
			"file_count":  len(fileNames),
			"field_count": len(fields),
			"status":      t.Status,
			"progress":    t.Progress,
			"created_at":  t.CreatedAt.Format(constants.DateFormatTimeHHMMSS),
		})
	}
	utils.SuccessPage(c, list, total, page, pageSize)
}

func DeleteExtractTask(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	taskID := c.Param("taskId")
	var task models.ContractExtractTask
	if err := database.DB.Where("id = ? AND user_id = ?", taskID, userUUID).First(&task).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}

	database.DB.Where("task_id = ?", task.ID).Delete(&models.ContractExtractResult{})
	database.DB.Delete(&task)
	utils.SuccessWithMsg(c, nil, "删除成功")
}

func ExportExtractResult(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	taskID := c.Param("taskId")
	var task models.ContractExtractTask
	if err := database.DB.Where("id = ? AND user_id = ?", taskID, userUUID).First(&task).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}

	var fields []models.ExtractFieldConfig
	json.Unmarshal([]byte(task.Fields), &fields)

	var results []models.ContractExtractResult
	database.DB.Where("task_id = ?", task.ID).Find(&results)

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	// Header
	f.SetCellValue(sheet, "A1", "文件名")
	for i, fld := range fields {
		col := string(rune('B' + i))
		f.SetCellValue(sheet, col+"1", fld.Name)
	}

	// Data rows
	for ri, r := range results {
		row := ri + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.FileName)
		var data map[string]interface{}
		json.Unmarshal([]byte(r.Results), &data)
		for fi, fld := range fields {
			col := excelize.ToAlphaString(fi + 1) // B, C, D...
			if v, ok := data[fld.Name]; ok && v != nil {
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), v)
			}
		}
	}

	// Auto-width columns
	lastCol := excelize.ToAlphaString(len(fields))
	f.SetColWidth(sheet, "A", lastCol, 30)

	trueFileName := fmt.Sprintf("提取结果_%s.xlsx", task.TaskName)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%s`, utils.PercentEncode(trueFileName)))
	buf, _ := f.WriteToBuffer()
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// runExtractAgent runs LLM extraction for each file
func runExtractAgent(et *extractTask, taskDBID uuid.UUID) {
	llm := services.NewLLMService()
	updateDB := func(pct int, status, step string) {
		et.Status = status
		et.Progress = pct
		et.Step = step
		database.DB.Model(&models.ContractExtractTask{}).Where("id = ?", taskDBID).
			Updates(map[string]interface{}{"progress": pct, "status": status})
	}

	fieldsJSON, _ := json.MarshalIndent(et.Fields, "", "  ")

	for idx, fi := range et.Files {
		pct := (idx * 100) / len(et.Files)
		updateDB(pct, constants.ContractExtractStatusExtracting, fmt.Sprintf("正在提取：%s", fi.Name))

		// Load file text
		var cf models.ContractFile
		if err := database.DB.Where("id = ?", fi.ID).First(&cf).Error; err != nil {
			database.DB.Model(&models.ContractExtractResult{}).Where("task_id = ? AND file_id = ?", taskDBID, fi.ID).
				Updates(map[string]interface{}{"status": "失败", "error_msg": "文件不存在"})
			continue
		}

		docText := ""
		if cf.FileSavePath != "" {
			data, err := services.DownloadContractFile(context.Background(), cf.FileSavePath)
			if err == nil {
				docText = extractText(cf.FileName, data)
			}
		}
		if docText == "" {
			database.DB.Model(&models.ContractExtractResult{}).Where("task_id = ? AND file_id = ?", taskDBID, fi.ID).
				Updates(map[string]interface{}{"status": "失败", "error_msg": "无法读取文档内容"})
			continue
		}

		// Truncate if too long
		runes := []rune(docText)
		if len(runes) > 30000 {
			docText = string(runes[:30000])
		}

		// Build prompt
		prompt := fmt.Sprintf(`你是一个合同信息提取专家。请从以下合同文本中提取指定字段的信息。

提取字段（以JSON格式输出，字段名作为key）：
%s

要求：
1. 每个字段输出对应的值，如果找不到则输出null
2. 金额字段请只输出数字
3. 日期字段请输出YYYY-MM-DD格式
4. 如字段描述中有枚举值，请从中选择
5. 只输出JSON，不要其他文字

合同文本：
%s`, string(fieldsJSON), docText)

		var resultData string
		var resultErr string
		success := false

		for attempt := 0; attempt < 3; attempt++ {
			resp, err := llm.GenerateLlmWithPrompt("你是一个合同信息提取专家，请严格按照要求输出JSON。", prompt)
			if err != nil {
				resultErr = fmt.Sprintf("LLM调用失败: %v", err)
				continue
			}
			resp = cleanJSON(resp)
			if json.Valid([]byte(resp)) {
				resultData = resp
				success = true
				break
			}
			resultErr = fmt.Sprintf("LLM返回格式错误，第%d次重试", attempt+1)
		}

		status := constants.ContractExtractStatusCompleted
		if !success {
			status = constants.ContractExtractStatusFailed
			resultData = "{}"
		}

		database.DB.Model(&models.ContractExtractResult{}).Where("task_id = ? AND file_id = ?", taskDBID, fi.ID).
			Updates(map[string]interface{}{
				"results":   resultData,
				"status":    status,
				"error_msg": resultErr,
			})
	}

	updateDB(100, constants.ContractExtractStatusCompleted, "提取完成")
	database.DB.Model(&models.ContractExtractTask{}).Where("id = ?", taskDBID).
		Update("done_files", len(et.Files))
	log.Printf("[ExtractAgent] Task %s completed: %d files", taskDBID, len(et.Files))
}
