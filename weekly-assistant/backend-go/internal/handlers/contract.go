package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func UploadContract(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Failed to parse form: %v", err)
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请提供上传文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" && ext != ".txt" {
		log.Printf("Invalid file type: %s", ext)
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "仅支持 .pdf .doc .docx 格式")
		return
	}

	if header.Size > 20*1024*1024 {
		log.Printf("Invalid file size: %d", header.Size)
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "文件大小不能超过 20MB")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "读取文件失败")
		return
	}

	fileUUID := uuid.New().String()
	ctx := context.Background()

	if err := services.UploadContractFile(ctx, fileUUID, data); err != nil {
		log.Printf("Failed to upload contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "文件上传至存储失败")
		return
	}

	cf := models.ContractFile{
		ID:       uuid.New(),
		UserID:   userUUID,
		FileName: header.Filename,
		FileSize: header.Size,
		FileUUID: fileUUID,
		Bucket:   services.GetOssBucket(),
		FileType: ext,
		Status:   "parsed",
	}
	if err := database.DB.Create(&cf).Error; err != nil {
		log.Printf("Failed to save contract file record: %v", err)
		services.DeleteContractFile(ctx, fileUUID)
		utils.ErrorWithMsg(c, utils.CodeServerError, "保存文件记录失败")
		return
	}

	utils.Success(c, gin.H{
		"id":        cf.ID.String(),
		"name":      cf.FileName,
		"size":      formatFileSize(cf.FileSize),
		"status":    cf.Status,
		"file_uuid": cf.FileUUID,
	})
}

func DeleteContractFile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	id := c.Param("id")
	var cf models.ContractFile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userUUID).First(&cf).Error; err != nil {
		log.Printf("Failed to find contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件不存在")
		return
	}

	if cf.FileUUID != "" {
		services.DeleteContractFile(context.Background(), cf.FileUUID)
	}

	database.DB.Delete(&cf)
	utils.SuccessWithMsg(c, nil, "删除成功")
}

func GetContractText(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	id := c.Param("id")
	var cf models.ContractFile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userUUID).First(&cf).Error; err != nil {
		log.Printf("Failed to find contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件不存在")
		return
	}

	if cf.FileUUID == "" {
		log.Printf("Invalid contract file: %s", cf.FileUUID)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件存储信息缺失")
		return
	}

	data, err := services.DownloadContractFile(context.Background(), cf.FileUUID)
	if err != nil {
		log.Printf("Failed to download contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "获取文件内容失败")
		return
	}

	text := extractText(cf.FileName, data)

	utils.Success(c, text)
}

func StartReview(c *gin.Context) {
	user, isExist := c.Get("user")
	if !isExist {
		utils.ErrorWithMsg(c, utils.CodeUnauthorized, "用户未登录")
		return
	}
	userName := "unknown"
	switch user := user.(type) {
	case models.User:
		userName = user.Username
	default:
		utils.ErrorWithMsg(c, utils.CodeUnauthorized, "用户未登录")
		return
	}

	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	var req struct {
		FileIDs      []string `json:"file_ids"`
		ContractType string   `json:"contract_type"`
		Position     string   `json:"position"`
		Standards    []string `json:"standards"`
		CustomType   string   `json:"custom_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}
	if len(req.FileIDs) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请至少选择一份合同文件")
		return
	}
	if req.ContractType == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请选择合同类型")
		return
	}
	if req.Position == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请选择审查立场")
		return
	}
	if len(req.Standards) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请选择至少一项审查标准")
		return
	}

	var files []models.ContractFile
	database.DB.Where("id IN ? AND user_id = ?", req.FileIDs, userUUID).Find(&files)
	if len(files) == 0 {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "选中的文件不存在")
		return
	}

	fileIDsJSON, _ := json.Marshal(req.FileIDs)
	standardsJSON, _ := json.Marshal(req.Standards)
	positionLabel := positionLabel(req.Position)
	contractTypeLabel := contractTypeLabel(req.ContractType, req.CustomType)
	standardsLabel := standardsLabel(req.Standards)
	ruleNames := ruleNameList()
	totalRules := len(ruleNames)
	review := models.ContractReview{
		UserID:            userUUID,
		FileName:          files[0].FileName,
		FileIDs:           string(fileIDsJSON),
		ContractType:      req.ContractType,
		ContractTypeLabel: contractTypeLabel,
		Position:          req.Position,
		PositionLabel:     positionLabel,
		Standards:         string(standardsJSON),
		StandardsLabel:    standardsLabel,
		Status:            "running",
		Progress:          0,
		HighRisk:          0,
		MediumRisk:        0,
		LowRisk:           0,
		TotalRules:        totalRules,
		Reviewer:          userName,
	}

	if err := database.DB.Create(&review).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "创建审查任务失败")
		return
	}

	go runReviewEngine(&review, files, req.Position, req.ContractType, ruleNames)

	utils.Success(c, gin.H{
		"task_id":   review.ID.String(),
		"report_id": review.ID.String(),
	})
}

func GetReviewProgress(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	taskID := c.Param("taskId")
	var review models.ContractReview
	if err := database.DB.Where("id = ? AND user_id = ?", taskID, userUUID).First(&review).Error; err != nil {
		log.Printf("Failed to find contract review: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "审查任务不存在")
		return
	}

	utils.Success(c, gin.H{
		"percent":      review.Progress,
		"current_rule": review.CurrentRule,
		"high_risk":    review.HighRisk,
		"medium_risk":  review.MediumRisk,
		"low_risk":     review.LowRisk,
		"status":       review.Status,
	})
}

func GetReviewReport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	reportID := c.Param("reportId")
	var review models.ContractReview
	if err := database.DB.Where("id = ? AND user_id = ?", reportID, userUUID).First(&review).Error; err != nil {
		log.Printf("Failed to find contract review: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "报告不存在")
		return
	}

	var items []models.ContractReviewItem
	database.DB.Where("review_id = ?", review.ID).Order("sort_order ASC").Find(&items)

	itemList := make([]gin.H, 0)
	for _, item := range items {
		itemList = append(itemList, gin.H{
			"id":            item.ID.String(),
			"level":         item.Level,
			"section":       item.Section,
			"rule_name":     item.RuleName,
			"description":   item.Description,
			"suggestion":    item.Suggestion,
			"law_ref":       item.LawRef,
			"original_text": item.OriginalText,
			"status":        item.Status,
			"comment":       item.Comment,
		})
	}

	riskStats := gin.H{
		"high":   review.HighRisk,
		"medium": review.MediumRisk,
		"low":    review.LowRisk,
	}

	utils.Success(c, gin.H{
		"id":                  review.ID.String(),
		"file_name":           review.FileName,
		"contract_type":       review.ContractType,
		"contract_type_label": review.ContractTypeLabel,
		"position":            review.Position,
		"position_label":      review.PositionLabel,
		"standards_label":     review.StandardsLabel,
		"status":              review.Status,
		"conclusion":          review.Conclusion,
		"total_rules":         review.TotalRules,
		"risk_stats":          riskStats,
		"review_time":         review.ReviewTime,
		"reviewer":            review.Reviewer,
		"items":               itemList,
	})
}

func UpdateReviewItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	reportID := c.Param("reportId")
	itemID := c.Param("itemId")

	var review models.ContractReview
	if err := database.DB.Where("id = ? AND user_id = ?", reportID, userUUID).First(&review).Error; err != nil {
		log.Printf("Failed to find contract review: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "报告不存在")
		return
	}

	var item models.ContractReviewItem
	if err := database.DB.Where("id = ? AND review_id = ?", itemID, review.ID).First(&item).Error; err != nil {
		log.Printf("Failed to find contract review item: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "审查项不存在")
		return
	}

	var req struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Comment != "" {
		updates["comment"] = req.Comment
	}
	if len(updates) > 0 {
		database.DB.Model(&item).Updates(updates)
	}

	utils.SuccessWithMsg(c, nil, "更新成功")
}

func GetHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "15"))
	keyword := c.Query("keyword")
	contractType := c.Query("contract_type")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 15
	}

	query := database.DB.Model(&models.ContractReview{}).
		Where("user_id = ?", userUUID)

	if keyword != "" {
		query = query.Where("file_name ILIKE ?", "%"+keyword+"%")
	}
	if contractType != "" {
		query = query.Where("contract_type = ?", contractType)
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var total int64
	query.Count(&total)

	var reviews []models.ContractReview
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&reviews)

	list := make([]gin.H, 0)
	for _, r := range reviews {
		riskStats := gin.H{
			"high":   r.HighRisk,
			"medium": r.MediumRisk,
			"low":    r.LowRisk,
		}
		list = append(list, gin.H{
			"id":                  r.ID.String(),
			"file_name":           r.FileName,
			"contract_type":       r.ContractType,
			"contract_type_label": r.ContractTypeLabel,
			"reviewer":            r.Reviewer,
			"review_time":         r.ReviewTime,
			"risk_stats":          riskStats,
			"total_risks":         r.HighRisk + r.MediumRisk + r.LowRisk,
			"conclusion":          r.Conclusion,
			"status":              r.Status,
		})
	}

	utils.SuccessPage(c, list, total, page, pageSize)
}

func DeleteHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	reportID := c.Param("reportId")
	var review models.ContractReview
	if err := database.DB.Where("id = ? AND user_id = ?", reportID, userUUID).First(&review).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "记录不存在")
		return
	}

	database.DB.Where("review_id = ?", review.ID).Delete(&models.ContractReviewItem{})
	database.DB.Delete(&review)
	utils.SuccessWithMsg(c, nil, "删除成功")
}

func ExportReport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	reportID := c.Param("reportId")
	format := c.Query("format")

	var review models.ContractReview
	if err := database.DB.Where("id = ? AND user_id = ?", reportID, userUUID).First(&review).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "报告不存在")
		return
	}

	var items []models.ContractReviewItem
	database.DB.Where("review_id = ?", review.ID).Order("sort_order ASC").Find(&items)

	ext := "txt"
	contentType := "text/plain; charset=utf-8"
	switch format {
	case "word":
		ext = "docx"
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "pdf":
		ext = "pdf"
		contentType = "application/pdf"
	case "excel":
		ext = "xlsx"
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	var buf bytes.Buffer
	writeReportContent(&buf, &review, items, format)

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="审查报告_%s.%s"`, review.FileName, ext))
	c.String(200, buf.String())
}

// --- helpers ---

func extractText(filename string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return string(data)
	case ".docx":
		return extractDocxText(data)
	case ".doc":
		return extractDocText(data)
	case ".pdf":
		return extractPDFText(data)
	default:
		return string(data)
	}
}

func extractDocxText(data []byte) string {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Printf("Failed to read docx file: %v", err)
		return ""
	}
	var parts []string
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				log.Printf("Failed to open docx file: %v", err)
				continue
			}
			defer rc.Close()
			xmlData, _ := io.ReadAll(rc)
			text := extractXMLText(string(xmlData))
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func extractXMLText(xml string) string {
	var result strings.Builder
	inTag := false
	for _, r := range xml {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func extractDocText(data []byte) string {
	if len(data) < 8 {
		log.Printf("File is too small to be a .doc file")
		return extractPrintableStrings(data)
	}

	// try zip first — some .doc files are actually .docx
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		log.Printf("Found OLE2 magic, assuming .docx")
		return extractDocxText(data)
	}

	// OLE2 magic: D0 CF 11 E0 A1 B1 1A E1
	isOLE2 := data[0] == 0xD0 && data[1] == 0xCF &&
		data[2] == 0x11 && data[3] == 0xE0 &&
		data[4] == 0xA1 && data[5] == 0xB1 &&
		data[6] == 0x1A && data[7] == 0xE1

	if !isOLE2 {
		log.Printf("Not an OLE2 file")
		return extractPrintableStrings(data)
	}

	var result strings.Builder

	textUTF16 := extractUTF16LEText(data)
	if textUTF16 != "" {
		result.WriteString(textUTF16)
		result.WriteString("\n")
	}

	textASCII := extractPrintableStrings(data)
	if textASCII != "" {
		result.WriteString(textASCII)
	}

	combined := result.String()
	if len(combined) < 20 {
		return string(data)
	}
	return combined
}

func extractUTF16LEText(data []byte) string {
	var result strings.Builder
	runes := make([]rune, 0, len(data)/16)

	for i := 0; i < len(data)-1; i += 2 {
		low := uint16(data[i])
		high := uint16(data[i+1])

		switch {
		case high == 0 && low >= 0x20 && low <= 0x7E:
			runes = append(runes, rune(low))
		case high == 0 && low == 0:
			if len(runes) >= 4 {
				result.WriteString(string(runes))
				result.WriteRune('\n')
			}
			runes = runes[:0]
		case high == 0 && (low == 0x0D || low == 0x0A):
			if len(runes) >= 4 {
				result.WriteString(string(runes))
				result.WriteRune('\n')
			}
			runes = runes[:0]
		default:
			if len(runes) >= 4 {
				result.WriteString(string(runes))
				result.WriteRune('\n')
			}
			runes = runes[:0]
		}
	}
	if len(runes) >= 4 {
		result.WriteString(string(runes))
	}

	return result.String()
}

func extractPrintableStrings(data []byte) string {
	var result strings.Builder
	var buf []byte

	for _, b := range data {
		if b >= 0x20 && b <= 0x7E {
			buf = append(buf, b)
		} else if b == '\n' || b == '\r' || b == '\t' {
			if len(buf) > 0 {
				buf = append(buf, b)
			}
		} else {
			if len(buf) >= 4 {
				if result.Len() > 0 {
					result.WriteRune('\n')
				}
				result.Write(buf)
			}
			buf = buf[:0]
		}
	}
	if len(buf) >= 4 {
		if result.Len() > 0 {
			result.WriteRune('\n')
		}
		result.Write(buf)
	}

	return result.String()
}

func extractPDFText(data []byte) string {
	content := string(data)
	var result strings.Builder
	inText := false
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "BT") {
			inText = true
			continue
		}
		if strings.HasPrefix(trimmed, "ET") {
			inText = false
			continue
		}
		if inText {
			clean := extractPDFTextFragment(trimmed)
			if clean != "" {
				result.WriteString(clean + " ")
			}
		}
	}
	if result.Len() < 20 {
		return stripNonPrintable(content)
	}
	return result.String()
}

func extractPDFTextFragment(s string) string {
	if strings.Contains(s, "Tj") || strings.Contains(s, "TJ") {
		var result strings.Builder
		inParen := false
		for _, r := range s {
			if r == '(' {
				inParen = true
			} else if r == ')' {
				inParen = false
				result.WriteRune(' ')
			} else if inParen {
				result.WriteRune(r)
			}
		}
		return result.String()
	}
	return ""
}

func stripNonPrintable(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 32 && r <= 126 || r == '\n' || r == '\t' || r == '\r' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func runReviewEngine(review *models.ContractReview, files []models.ContractFile, position, contractType string, ruleNames []string) {
	totalRules := len(ruleNames)

	var allText string
	for _, f := range files {
		if f.FileUUID != "" {
			data, err := services.DownloadContractFile(context.Background(), f.FileUUID)
			if err == nil {
				text := extractText(f.FileName, data)
				allText += text + "\n"
			}
		}
	}

	if allText == "" {
		database.DB.Model(review).Updates(map[string]interface{}{
			"status":   "failed",
			"progress": 0,
		})
		return
	}

	systemPrompt := buildReviewSystemPrompt(position, contractType)

	llm := services.NewLLMService()

	for i, name := range ruleNames {
		progress := int(math.Round(float64(i+1) / float64(totalRules) * 100))
		database.DB.Model(review).Updates(map[string]interface{}{
			"progress":     progress,
			"current_rule": name,
		})

		rulePrompt := fmt.Sprintf(`作为合同审查专家，请对以下合同文本进行审查。

审查规则：%s
审查立场：%s
合同类型：%s

合同文本：
%s

请基于上述规则审查合同。如果发现相关问题，输出JSON对象，包含以下字段：
- level: "high" 或 "medium" 或 "low"
- section: 条款编号（如 "7"）
- rule_name: "%s"
- description: 问题描述
- suggestion: 修改建议
- law_ref: 法律依据（如相关法条）
- original_text: 原文片段（从合同中摘录）

如果未发现问题，输出 null。只输出JSON，不要其他文字。`, allText, positionLabel(position), contractTypeLabel(contractType, ""), allText, name)

		result, err := llm.GenerateLlmWithPrompt(systemPrompt, rulePrompt)
		if err != nil {
			log.Printf("Failed to contract review for rule %s: %s", name, err)
			continue
		}

		log.Printf("Contract review result for rule %s: %s", name, result)

		result = strings.TrimSpace(result)
		result = strings.TrimPrefix(result, "```json")
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)

		if result == "" || result == "null" {
			log.Printf("No problem found for rule %s", name)
			continue
		}

		var itemDatas []struct {
			Level        string `json:"level"`
			Section      string `json:"section"`
			RuleName     string `json:"rule_name"`
			Description  string `json:"description"`
			Suggestion   string `json:"suggestion"`
			LawRef       string `json:"law_ref"`
			OriginalText string `json:"original_text"`
		}

		if err := json.Unmarshal([]byte(result), &itemDatas); err != nil {
			log.Printf("Failed to parse contract review for rule %s: %s", name, err)
			continue
		}
		for _, itemData := range itemDatas {
			if itemData.Description == "" {
				log.Printf("No description found for rule %s", name)
				continue
			}
			if itemData.Level == "" {
				itemData.Level = "medium"
			}
			if itemData.RuleName == "" {
				itemData.RuleName = name
			}

			item := models.ContractReviewItem{
				ReviewID:     review.ID,
				Level:        itemData.Level,
				Section:      itemData.Section,
				RuleName:     itemData.RuleName,
				Description:  itemData.Description,
				Suggestion:   itemData.Suggestion,
				LawRef:       itemData.LawRef,
				OriginalText: itemData.OriginalText,
				Status:       "open",
				SortOrder:    i,
			}
			database.DB.Create(&item)

			switch itemData.Level {
			case "high":
				review.HighRisk++
			case "medium":
				review.MediumRisk++
			case "low":
				review.LowRisk++
			}
		}

	}

	conclusion := generateConclusion(review.HighRisk, review.MediumRisk)
	reviewTime := time.Now().Format("2006-01-02 15:04")

	database.DB.Model(review).Updates(map[string]interface{}{
		"status":      "completed",
		"progress":    100,
		"high_risk":   review.HighRisk,
		"medium_risk": review.MediumRisk,
		"low_risk":    review.LowRisk,
		"conclusion":  conclusion,
		"review_time": reviewTime,
	})
}

func buildReviewSystemPrompt(position, contractType string) string {
	return fmt.Sprintf(`你是一位资深的合同审查法律专家，精通中国民法典及相关法律法规。

当前审查配置：
- 审查立场：%s
- 合同类型：%s

请根据审查立场和合同类型，仔细审查合同条款，发现潜在的法律风险和商业风险。

审查规则包括：
1. 违约金比例是否超过法定上限
2. 是否仅单方约定违约责任
3. 违约金计算标准是否明确
4. 是否约定了免责条款
5. 管辖条款是否缺失
6. 知识产权归属是否模糊
7. 保密期限是否缺失
8. 付款条款是否合理
9. 验收条款是否明确
10. 争议解决方式是否明确
11. 合同解除条件是否合理
12. 不可抗力条款是否完整
13. 通知送达条款是否完备
14. 文本表述是否有歧义
15. 必备条款是否缺失
16. 权利义务是否对等
17. 赔偿上限是否合理
18. 续约/终止条款是否明确
19. 数据保护条款是否缺失
20. 适用法律是否明确

对于每条规则，如果发现问题，输出JSON格式结果。如果未发现问题，输出null。`, positionLabel(position), contractTypeLabel(contractType, ""))
}

func ruleNameList() []string {
	return []string{
		"违约金比例超限检测",
		"单方违约责任检测",
		"违约计算标准检测",
		"免责条款完整性检测",
		"管辖条款缺失检测",
		"知识产权归属检测",
		"保密期限缺失检测",
		"付款条款合理性检测",
		"验收条款明确性检测",
		"争议解决方式检测",
		"合同解除条件检测",
		"不可抗力条款检测",
		"通知送达条款检测",
		"文本表述歧义检测",
		"必备条款完整性检测",
		"权利义务对等性检测",
		"赔偿上限合理性检测",
		"续约终止条款检测",
		"数据保护条款检测",
		"适用法律明确性检测",
	}
}

func generateConclusion(high, medium int) string {
	if high >= 3 {
		return "建议不通过"
	}
	if high >= 1 || medium >= 3 {
		return "建议有条件通过"
	}
	return "建议通过"
}

func positionLabel(pos string) string {
	switch pos {
	case "party_a":
		return "甲方立场"
	case "party_b":
		return "乙方立场"
	case "neutral":
		return "中立立场"
	default:
		return pos
	}
}

func contractTypeLabel(ct, custom string) string {
	types := map[string]string{
		"purchase":            "买卖合同",
		"lease":               "租赁合同",
		"service":             "服务合同",
		"labor":               "劳动合同",
		"investment":          "投融资合同",
		"engineering":         "工程合同",
		"ip":                  "知识产权合同",
		"other":               "其他",
		"equipment":           "设备采购",
		"raw_material":        "原材料采购",
		"service_procurement": "服务采购",
		"framework":           "框架协议",
		"housing":             "房屋租赁",
		"equipment_lease":     "设备租赁",
		"venue":               "场地租赁",
		"tech_service":        "技术服务",
		"consulting":          "咨询服务",
		"property":            "物业服务",
		"transport":           "运输服务",
		"employment":          "劳动合同",
		"dispatch":            "劳务派遣",
		"nda":                 "保密协议",
		"non_compete":         "竞业限制",
		"equity_transfer":     "股权转让",
		"capital_increase":    "增资协议",
		"loan":                "借款合同",
		"guarantee":           "担保合同",
		"construction":        "施工总包",
		"subcontract":         "分包合同",
		"survey_design":       "勘察设计",
		"supervision":         "监理合同",
		"patent_license":      "专利许可",
		"trademark":           "商标转让",
		"copyright":           "版权授权",
		"tech_dev":            "技术开发",
	}
	if ct == "other" && custom != "" {
		return custom
	}
	if label, ok := types[ct]; ok {
		return label
	}
	return ct
}

func standardsLabel(standards []string) string {
	labels := map[string]string{
		"internal": "内部合规标准",
		"legal":    "法律法规标准",
		"industry": "行业标准",
		"custom":   "自定义标准",
	}
	result := make([]string, 0)
	for _, s := range standards {
		if label, ok := labels[s]; ok {
			result = append(result, label)
		} else {
			result = append(result, s)
		}
	}
	return strings.Join(result, "、")
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

func writeReportContent(w io.Writer, review *models.ContractReview, items []models.ContractReviewItem, format string) {
	if format == "excel" {
		writeExcelReport(w, review, items)
		return
	}
	writeTextReport(w, review, items)
}

func writeTextReport(w io.Writer, review *models.ContractReview, items []models.ContractReviewItem) {
	fmt.Fprintf(w, "合同审查报告\n")
	fmt.Fprintf(w, "==============================\n\n")
	fmt.Fprintf(w, "合同名称：%s\n", review.FileName)
	fmt.Fprintf(w, "合同类型：%s\n", review.ContractTypeLabel)
	fmt.Fprintf(w, "审查立场：%s\n", review.PositionLabel)
	fmt.Fprintf(w, "审查标准：%s\n", review.StandardsLabel)
	fmt.Fprintf(w, "审查时间：%s\n", review.ReviewTime)
	fmt.Fprintf(w, "审查人：%s\n", review.Reviewer)
	fmt.Fprintf(w, "综合评级：%s\n\n", review.Conclusion)

	fmt.Fprintf(w, "风险统计：高风险 %d 项 / 中风险 %d 项 / 低风险 %d 项\n\n", review.HighRisk, review.MediumRisk, review.LowRisk)
	fmt.Fprintf(w, "==============================\n")
	fmt.Fprintf(w, "审查明细\n")
	fmt.Fprintf(w, "==============================\n\n")

	for i, item := range items {
		fmt.Fprintf(w, "%d. [%s] ", i+1, strings.ToUpper(item.Level))
		if item.Section != "" {
			fmt.Fprintf(w, "第%s条 ", item.Section)
		}
		fmt.Fprintf(w, "%s\n", item.RuleName)
		fmt.Fprintf(w, "   问题：%s\n", item.Description)
		if item.Suggestion != "" {
			fmt.Fprintf(w, "   建议：%s\n", item.Suggestion)
		}
		if item.LawRef != "" {
			fmt.Fprintf(w, "   依据：%s\n", item.LawRef)
		}
		if item.OriginalText != "" {
			fmt.Fprintf(w, "   原文：%s\n", item.OriginalText)
		}
		fmt.Fprintf(w, "   状态：%s\n\n", item.Status)
	}
}

func writeExcelReport(w io.Writer, review *models.ContractReview, items []models.ContractReviewItem) {
	fmt.Fprintf(w, "合同名称\t合同类型\t审查立场\t审查时间\t综合评级\n")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n\n", review.FileName, review.ContractTypeLabel, review.PositionLabel, review.ReviewTime, review.Conclusion)
	fmt.Fprintf(w, "风险等级\t条款\t规则名称\t问题描述\t修改建议\t法律依据\t原文\t状态\n")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", item.Level, item.Section, item.RuleName, item.Description, item.Suggestion, item.LawRef, item.OriginalText, item.Comment)
	}
}
