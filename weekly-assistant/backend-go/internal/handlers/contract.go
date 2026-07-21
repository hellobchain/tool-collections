package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/constants"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
	"github.com/hellobchain/wswlog/wlogging"
)

var slog = wlogging.MustGetLoggerWithoutName()

func UploadContract(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userUUID, _ := uuid.Parse(userID)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		slog.Errorf("Failed to parse form: %v", err)
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请提供上传文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" && ext != ".txt" {
		slog.Errorf("Invalid file type: %s", ext)
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "仅支持 .pdf .doc .docx 格式")
		return
	}

	if header.Size > constants.MAX_FILE_SIZE {
		slog.Errorf("Invalid file size: %d", header.Size)
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "文件大小不能超过 20MB")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		slog.Errorf("Failed to read file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "读取文件失败")
		return
	}

	fileUUID := uuid.New().String()
	dateStr := time.Now().Format("2006-01-02")
	fileName := header.Filename
	fileSavePath := filepath.Join(dateStr, fileUUID, fileName)
	ctx := context.Background()

	if err := services.UploadContractFile(ctx, fileSavePath, data); err != nil {
		slog.Errorf("Failed to upload contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "文件上传至存储失败")
		return
	}

	cf := models.ContractFile{
		UserID:       userUUID,
		FileName:     fileName,
		FileSize:     header.Size,
		FileUUID:     fileUUID,
		FileSavePath: fileSavePath,
		Bucket:       services.GetOssBucket(),
		FileType:     ext,
		Status:       constants.ContractDraftStatusParsed,
	}
	if err := database.DB.Create(&cf).Error; err != nil {
		slog.Errorf("Failed to save contract file record: %v", err)
		services.DeleteContractFile(ctx, fileSavePath)
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
		slog.Errorf("Failed to find contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件不存在")
		return
	}

	if cf.FileSavePath != "" {
		services.DeleteContractFile(context.Background(), cf.FileSavePath)
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
		slog.Errorf("Failed to find contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件不存在")
		return
	}

	if cf.FileSavePath == "" {
		slog.Errorf("Invalid contract file: %s", cf.FileSavePath)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "文件存储信息缺失")
		return
	}

	data, err := services.DownloadContractFile(context.Background(), cf.FileSavePath)
	if err != nil {
		slog.Errorf("Failed to download contract file: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "获取文件内容失败")
		return
	}

	text, err := extractText(cf.FileName, data)
	if err != nil {
		slog.Errorf("Failed to extract text: %v", err)
		utils.ErrorWithMsg(c, utils.CodeServerError, "获取文件内容失败")
		return
	}

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
	reviewStartTime := time.Now().Format(constants.DateFormatTimeHHMMSS)
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
		Status:            constants.ContractReviewStatusRunning,
		Progress:          0,
		HighRisk:          0,
		MediumRisk:        0,
		LowRisk:           0,
		TotalRules:        totalRules,
		Reviewer:          userName,
		ReviewStartTime:   reviewStartTime,
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
		slog.Errorf("Failed to find contract review: %v", err)
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
		slog.Errorf("Failed to find contract review: %v", err)
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
		"review_start_time":   review.ReviewStartTime,
		"review_end_time":     review.ReviewEndTime,
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
		slog.Errorf("Failed to find contract review: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, "报告不存在")
		return
	}

	var item models.ContractReviewItem
	if err := database.DB.Where("id = ? AND review_id = ?", itemID, review.ID).First(&item).Error; err != nil {
		slog.Errorf("Failed to find contract review item: %v", err)
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
	reviewStartDateStart := c.Query("review_start_date_start")
	reviewStartDateEnd := c.Query("review_start_date_end")
	reviewEndDateStart := c.Query("review_end_date_start")
	reviewEndDateEnd := c.Query("review_end_date_end")

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
	if reviewStartDateStart != "" {
		query = query.Where("review_start_time >= ?", reviewStartDateStart+" 00:00:00")
	}
	if reviewStartDateEnd != "" {
		query = query.Where("review_start_time <= ?", reviewStartDateEnd+" 23:59:59")
	}
	if reviewEndDateStart != "" {
		query = query.Where("review_end_time >= ?", reviewEndDateStart+" 00:00:00")
	}
	if reviewEndDateEnd != "" {
		query = query.Where("review_end_time <= ?", reviewEndDateEnd+" 23:59:59")
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
			"review_start_time":   r.ReviewStartTime,
			"review_end_time":     r.ReviewEndTime,
			"risk_stats":          riskStats,
			"total_risks":         r.HighRisk + r.MediumRisk + r.LowRisk,
			"conclusion":          r.Conclusion,
			"status":              r.Status,
			"progress":            r.Progress,
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
	trueFileName := strings.TrimSuffix(review.FileName, filepath.Ext(review.FileName))
	trueFileName = fmt.Sprintf("审查报告_%s.%s", trueFileName, ext)
	c.Writer.WriteHeader(http.StatusOK)
	c.Header("Accept-charset", "utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%s`, utils.PercentEncode(trueFileName)))
	c.Header("Content-Type", contentType)
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.String(200, buf.String())
}

// --- helpers ---

func extractText(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return string(data), nil
	case ".docx":
		// return extractDocxText(data)
		return services.DocConvertText(data, filename)
	case ".doc":
		// return extractDocText(data)
		return services.DocConvertText(data, filename)
	case ".pdf":
		// return extractPDFText(data)
		return services.DocConvertText(data, filename)
	default:
		return string(data), nil
	}
}

func extractDocxText(data []byte) string {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		slog.Errorf("Failed to read docx file: %v", err)
		return ""
	}
	var parts []string
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				slog.Errorf("Failed to open docx file: %v", err)
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

type oleReader struct {
	data    []byte
	secSize int
	secCnt  int
}

const (
	ole2FreeSec = -1
	ole2EndSec  = -2
)

func oleSecID(data []byte, offset int) int {
	u := uint32(data[offset]) | uint32(data[offset+1])<<8 | uint32(data[offset+2])<<16 | uint32(data[offset+3])<<24
	return int(int32(u))
}

func hasRootEntry(dirData []byte) bool {
	if len(dirData) < 128 {
		return false
	}
	rootName := []byte{'R', 0, 'o', 0, 'o', 0, 't', 0, ' ', 0, 'E', 0, 'n', 0, 't', 0, 'r', 0, 'y', 0}
	for i := 0; i < min(10, len(dirData)-20); i += 128 {
		match := true
		for j := 0; j < len(rootName); j++ {
			if dirData[i+j] != rootName[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (r *oleReader) findDirectoryByContent(sat []int) []byte {
	// Scan all sector chains for one that starts with "Root Entry"
	visited := make([]bool, r.secCnt)
	for start := 0; start < r.secCnt; start++ {
		if visited[start] || sat[start] == -1 || sat[start] == -3 || sat[start] == -4 {
			continue
		}
		// Read the first sector of this chain
		sec := r.sector(start)
		if sec == nil {
			continue
		}
		if hasRootEntry(sec) {
			// This chain is the directory — read all sectors in it
			var allSectors []byte
			sid := start
			for sid >= 0 && sid != ole2EndSec && sid < r.secCnt {
				if sid >= len(visited) || visited[sid] {
					break
				}
				visited[sid] = true
				sec := r.sector(sid)
				if sec == nil {
					break
				}
				allSectors = append(allSectors, sec...)
				sid = sat[sid]
			}
			return allSectors
		}
		// Mark this chain as visited
		sid := start
		for sid >= 0 && sid != ole2EndSec && sid < r.secCnt {
			visited[sid] = true
			sid = sat[sid]
		}
	}
	return nil
}

func (r *oleReader) sector(secID int) []byte {
	off := (secID + 1) * r.secSize
	if off < 0 || off+int(r.secSize) > len(r.data) {
		return nil
	}
	return r.data[off : off+r.secSize]
}

func (r *oleReader) readSAT() []int {
	// DIFAT: first 109 entries are in the header at offset 76
	var secIDs []int
	for i := 0; i < 109; i++ {
		sid := oleSecID(r.data, 76+i*4)
		if sid == ole2FreeSec {
			continue
		}
		if sid == ole2EndSec {
			break
		}
		secIDs = append(secIDs, sid)
	}
	// additional DIFAT sectors
	nextDif := oleSecID(r.data, 44)
	cntDif := oleSecID(r.data, 48)
	for i := 0; i < cntDif && nextDif != ole2EndSec && nextDif >= 0; i++ {
		sec := r.sector(nextDif)
		if sec == nil {
			break
		}
		for j := 0; j < (r.secSize/4)-1; j++ {
			sid := oleSecID(sec, j*4)
			if sid == ole2EndSec || sid == ole2FreeSec {
				break
			}
			secIDs = append(secIDs, sid)
		}
		nextDif = oleSecID(sec, r.secSize-4)
	}

	// Build SAT: read each FAT sector
	sat := make([]int, r.secCnt)
	idx := 0
	for _, fatSid := range secIDs {
		sec := r.sector(fatSid)
		if sec == nil {
			break
		}
		for j := 0; j < r.secSize/4 && idx < r.secCnt; j++ {
			sat[idx] = oleSecID(sec, j*4)
			idx++
		}
	}
	return sat
}

func (r *oleReader) readStreamData(startSecID, size int, sat []int) []byte {
	if startSecID < 0 || size <= 0 {
		return nil
	}
	// Mini Stream cutoff is at header offset 32
	miniCutoff := int(uint32(r.data[32]) | uint32(r.data[33])<<8 | uint32(r.data[34])<<16 | uint32(r.data[35])<<24)
	if miniCutoff == 0 {
		miniCutoff = 4096 // default
	}

	// If stream size < mini cutoff, use Mini Stream (Root Entry data)
	if size < miniCutoff {
		// Find Root Entry and read its data as Mini Stream
		satCopy := sat
		entries := r.readDirEntries(satCopy)
		for _, e := range entries {
			if e.name == "Root Entry" && e.startSec >= 0 && e.size > 0 {
				mData := r.readFromSectors(e.startSec, e.size, sat)
				if len(mData) > 0 && startSecID*64+size <= len(mData) {
					// In Mini Stream, startSecID is a mini-sector index (64 bytes each)
					start := startSecID * 64
					return mData[start : start+size]
				}
				break
			}
		}
	}
	return r.readFromSectors(startSecID, size, sat)
}

func (r *oleReader) readFromSectors(startSecID, size int, sat []int) []byte {
	if startSecID < 0 || size <= 0 {
		return nil
	}
	out := make([]byte, 0, size)
	sid := startSecID
	for sid >= 0 && sid != ole2EndSec && len(out) < size {
		sec := r.sector(sid)
		if sec == nil {
			break
		}
		remain := size - len(out)
		if remain >= r.secSize {
			out = append(out, sec...)
		} else {
			out = append(out, sec[:remain]...)
		}
		if sid < 0 || sid >= len(sat) {
			break
		}
		sid = sat[sid]
	}
	return out
}

type oleDirEntry struct {
	name     string
	objType  byte
	startSec int
	size     int
	child    int
}

func (r *oleReader) readDirEntries(sat []int) []oleDirEntry {
	// Try reading directory from the standard offset (28 in OLE2 header)
	dirSecID := oleSecID(r.data, 28)
	dirData := r.readStreamData(dirSecID, r.secSize*100, sat)

	// Fallback: some Word-created .doc files put the directory at the DIFAT sector
	// or at a chain that starts with "Root Entry"
	if len(dirData) < 128 || !hasRootEntry(dirData) {
		// Try searching all valid chains for Root Entry
		dirData = r.findDirectoryByContent(sat)
	}
	if len(dirData) == 0 {
		return nil
	}

	entries := make([]oleDirEntry, 0, len(dirData)/128)
	for i := 0; i+127 < len(dirData); i += 128 {
		// name is UTF-16LE at offset 0, max 64 bytes (32 chars)
		nameLen := int(dirData[i+64]) | int(dirData[i+65])<<8
		if nameLen < 2 {
			break
		}
		nameBytes := dirData[i : i+nameLen-2] // exclude null terminator
		name := decodeUTF16LE(nameBytes)

		objType := dirData[i+66]
		startSec := int(uint32(dirData[i+116]) | uint32(dirData[i+117])<<8 | uint32(dirData[i+118])<<16 | uint32(dirData[i+119])<<24)
		size := int(uint32(dirData[i+120]) | uint32(dirData[i+121])<<8 | uint32(dirData[i+122])<<16 | uint32(dirData[i+123])<<24)
		child := int(uint32(dirData[i+76]) | uint32(dirData[i+77])<<8 | uint32(dirData[i+78])<<16 | uint32(dirData[i+79])<<24)

		entries = append(entries, oleDirEntry{
			name:     name,
			objType:  objType,
			startSec: startSec,
			size:     size,
			child:    child,
		})
	}
	return entries
}

func decodeUTF16LE(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	runes := make([]rune, 0, len(b)/2)
	for i := 0; i < len(b)-1; i += 2 {
		low := uint16(b[i])
		high := uint16(b[i+1])
		r := rune(high)<<8 | rune(low)
		if r == 0 {
			break
		}
		runes = append(runes, r)
	}
	return string(runes)
}

// ============== .doc text extraction ==============

func extractDocText(data []byte) string {
	if len(data) < 64 {
		return extractPrintableStrings(data)
	}

	// Try zip first — some .doc files are actually .docx
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		return extractDocxText(data)
	}

	// Strategy 1: collect all OLE2 stream data, then scan for encodings
	if isOLE2(data) {
		streamData := collectOLE2Streams(data)
		if len(streamData) > 0 {
			if t := tryDecode(streamData); t != "" {
				return t
			}
		}
	}

	// Strategy 2: scan raw file directly for UTF-16LE or GBK text
	if t := tryDecode(data); t != "" {
		return t
	}

	return extractPrintableStrings(data)
}

func isOLE2(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0xD0 && data[1] == 0xCF &&
		data[2] == 0x11 && data[3] == 0xE0 &&
		data[4] == 0xA1 && data[5] == 0xB1 &&
		data[6] == 0x1A && data[7] == 0xE1
}

func collectOLE2Streams(data []byte) []byte {
	if len(data) < 512 || !isOLE2(data) {
		return nil
	}
	secPower := int(data[30])
	secSize := 1 << uint(secPower)
	if secSize < 64 || secSize > 4096 {
		return nil
	}
	secCnt := (len(data) + secSize - 1) / secSize
	rd := &oleReader{data: data, secSize: secSize, secCnt: secCnt}
	sat := rd.readSAT()
	if len(sat) == 0 {
		return nil
	}
	entries := rd.readDirEntries(sat)
	if len(entries) == 0 {
		return nil
	}
	var combined []byte
	// Prioritize WordDocument stream
	for _, e := range entries {
		if e.objType == 2 && e.name == "WordDocument" && e.size > 0 && e.startSec >= 0 {
			d := rd.readStreamData(e.startSec, e.size, sat)
			if len(d) > 0 {
				combined = append(combined, d...)
			}
			break
		}
	}
	// Then add other streams
	for _, e := range entries {
		if e.objType == 2 && e.name != "WordDocument" && e.size > 0 && e.startSec >= 0 {
			d := rd.readStreamData(e.startSec, e.size, sat)
			if len(d) > 0 {
				combined = append(combined, d...)
			}
		}
	}
	return combined
}

// tryDecode scans data as UTF-16LE, collects ALL viable segments >= 6 bytes,
// concatenates in document order. Catches table cells (separated by 0x07)
// which are tiny 3-5 char segments.
func tryDecode(data []byte) string {
	type seg struct{ start, end int }
	const minSeg = 6
	var segs []seg
	cur := -1

	for i := 0; i < len(data)-1; i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		if isTextRune(r) {
			if cur < 0 {
				cur = i
			}
		} else {
			if cur >= 0 && i-cur >= minSeg {
				// Merge with previous if gap is 1 char (0x07 cell mark)
				if len(segs) > 0 && cur-segs[len(segs)-1].end <= 4 {
					segs[len(segs)-1].end = i
				} else {
					segs = append(segs, seg{cur, i})
				}
			}
			cur = -1
		}
	}
	if cur >= 0 && len(data)-cur >= minSeg {
		segs = append(segs, seg{cur, len(data)})
	}
	if len(segs) == 0 {
		return ""
	}

	var buf strings.Builder
	for _, s := range segs {
		raw := data[s.start:s.end]
		for i := 0; i < len(raw)-1; i += 2 {
			r := rune(raw[i]) | rune(raw[i+1])<<8
			if r == 0 || r == '\r' || r == '\n' {
				buf.WriteRune('\n')
			} else {
				buf.WriteRune(r)
			}
		}
		buf.WriteRune('\n')
	}
	return strings.TrimSpace(buf.String())
}

func isTextRune(r rune) bool {
	switch {
	case r >= 0x20 && r <= 0x7E: // ASCII printable
		return true
	case r >= 0x4E00 && r <= 0x9FFF: // CJK
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	case r >= 0x2E80 && r <= 0x2EFF:
		return true
	case r >= 0x3000 && r <= 0x303F:
		return true
	case r >= 0xFF00 && r <= 0xFFEF:
		return true
	case r >= 0x2000 && r <= 0x206F:
		return true
	case r >= 0xFE30 && r <= 0xFE4F:
		return true
	case r >= 0x00A0 && r <= 0x00FF:
		return true
	case r >= 0x0100 && r <= 0x024F:
		return true
	case r >= 0x0370 && r <= 0x03FF:
		return true
	case r >= 0x0400 && r <= 0x04FF:
		return true
	case r == 0x0A || r == 0x0D:
		return true
	default:
		return false
	}
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
		if f.FileSavePath != "" {
			data, err := services.DownloadContractFile(context.Background(), f.FileSavePath)
			if err == nil {
				text, err := extractText(f.FileName, data)
				if err != nil {
					slog.Error("Failed to extract text from file", "file_id", f.ID, "error", err)
					break
				}
				allText += text + "\n"
			}
		}
	}

	if allText == "" {
		database.DB.Model(review).Updates(map[string]interface{}{
			"status":   constants.ContractReviewStatusFailed,
			"progress": 0,
		})
		return
	}

	systemPrompt := buildReviewSystemPrompt(position, contractType)

	llm := services.NewLLMService()

	for i, name := range ruleNames {
		progress := int(math.Round(float64(i+1) / float64(totalRules) * 100))
		// Prevent progress from exceeding 100%
		if progress >= 100 {
			progress = 99
		}
		database.DB.Model(review).Updates(map[string]interface{}{
			"progress":     progress,
			"current_rule": name,
			"high_risk":    review.HighRisk,
			"medium_risk":  review.MediumRisk,
			"low_risk":     review.LowRisk,
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
			slog.Errorf("Failed to contract review for rule %s: %s", name, err)
			continue
		}

		slog.Infof("Contract review result for rule %s: %s", name, result)

		result = strings.TrimSpace(result)
		result = strings.TrimPrefix(result, "```json")
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)

		if result == "" || result == "null" {
			slog.Infof("No problem found for rule %s", name)
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
			slog.Errorf("Failed to parse contract review for rule %s: %s", name, err)
			continue
		}
		for _, itemData := range itemDatas {
			if itemData.Description == "" {
				slog.Errorf("No description found for rule %s", name)
				continue
			}
			if itemData.Level == "" {
				itemData.Level = "medium"
			}
			if itemData.RuleName == "" {
				itemData.RuleName = name
			}
			itemData.Level = strings.ToLower(itemData.Level)
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
	reviewEndTime := time.Now().Format(constants.DateFormatTimeHHMMSS)

	database.DB.Model(review).Updates(map[string]interface{}{
		"status":          constants.ContractReviewStatusCompleted,
		"progress":        100,
		"high_risk":       review.HighRisk,
		"medium_risk":     review.MediumRisk,
		"low_risk":        review.LowRisk,
		"conclusion":      conclusion,
		"review_end_time": reviewEndTime,
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
	fmt.Fprintf(w, "审查开始时间：%s\n", review.ReviewStartTime)
	fmt.Fprintf(w, "审查结束时间：%s\n", review.ReviewEndTime)
	fmt.Fprintf(w, "审查人：%s\n", review.Reviewer)
	fmt.Fprintf(w, "综合评级：%s\n\n", review.Conclusion)

	fmt.Fprintf(w, "风险统计：高风险 %d 项 / 中风险 %d 项 / 低风险 %d 项\n\n", review.HighRisk, review.MediumRisk, review.LowRisk)
	fmt.Fprintf(w, "==============================\n")
	fmt.Fprintf(w, "审查明细\n")
	fmt.Fprintf(w, "==============================\n\n")

	for i, item := range items {
		levelLabel := toLevelLabel(item.Level)
		fmt.Fprintf(w, "%d. [%s] ", i+1, levelLabel)
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
		fmt.Fprintf(w, "   状态：%s\n\n", toStatusLabel(item.Status))
	}
}

func writeExcelReport(w io.Writer, review *models.ContractReview, items []models.ContractReviewItem) {
	fmt.Fprintf(w, "合同名称\t合同类型\t审查立场\t审查开始时间\t审查结束时间\t综合评级\n")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n\n", review.FileName, review.ContractTypeLabel, review.PositionLabel, review.ReviewStartTime, review.ReviewEndTime, review.Conclusion)
	fmt.Fprintf(w, "风险等级\t条款\t规则名称\t问题描述\t修改建议\t法律依据\t原文\t状态\n")
	for _, item := range items {
		levelLabel := toLevelLabel(item.Level)
		statusLabel := toStatusLabel(item.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", levelLabel, item.Section, item.RuleName, item.Description, item.Suggestion, item.LawRef, item.OriginalText, statusLabel)
	}
}

func toLevelLabel(level string) string {
	levelLabel := strings.ToLower(level)
	switch levelLabel {
	case constants.RiskLevelHigh:
		levelLabel = constants.RiskLevelHighDesc
	case constants.RiskLevelMedium:
		levelLabel = constants.RiskLevelMediumDesc
	case constants.RiskLevelLow:
		levelLabel = constants.RiskLevelLowDesc
	}
	return levelLabel
}

func toStatusLabel(status string) string {
	statusLabel := strings.ToLower(status)
	switch statusLabel {
	case constants.ContractReviewItemStatusOpen:
		statusLabel = constants.ContractReviewItemStatusOpenDesc
	case constants.ContractReviewItemStatusIgnore:
		statusLabel = constants.ContractReviewItemStatusIgnoreDesc
	case constants.ContractReviewItemStatusFixed:
		statusLabel = constants.ContractReviewItemStatusFixedDesc
	case constants.ContractReviewItemStatusDone:
		statusLabel = constants.ContractReviewItemStatusDoneDesc
	}
	return "-"
}
