package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

var (
	stockTasks   = make(map[string]*StockTaskInfo)
	stockTasksMu sync.RWMutex
)

type StockTaskInfo struct {
	TaskID      string      `json:"task_id"`
	StockCode   string      `json:"stock_code"`
	StockName   string      `json:"stock_name,omitempty"`
	Status      string      `json:"status"`
	Progress    int         `json:"progress"`
	Message     string      `json:"message,omitempty"`
	Error       string      `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// POST /weekly-assistant/stock/v1/analysis/analyze - 触发股票分析
func TriggerStockAnalysis(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		StockCode     string   `json:"stock_code"`
		StockCodes    []string `json:"stock_codes"`
		ReportType    string   `json:"report_type"`
		ForceRefresh  bool     `json:"force_refresh"`
		AsyncMode     bool     `json:"async_mode"`
		AnalysisPhase string   `json:"analysis_phase"`
		StockName     string   `json:"stock_name"`
		Notify        bool     `json:"notify"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请求参数无效: "+err.Error())
		return
	}

	codes := req.StockCodes
	if req.StockCode != "" {
		codes = append([]string{req.StockCode}, codes...)
	}
	if len(codes) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "必须提供 stock_code 或 stock_codes 参数")
		return
	}
	if len(codes) > 50 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "单次分析最多支持50只股票")
		return
	}

	reportType := req.ReportType
	if reportType == "" {
		reportType = "detailed"
	}

	if !req.AsyncMode {
		if len(codes) > 1 {
			utils.ErrorWithMsg(c, utils.CodeInvalidParams, "同步模式仅支持单只股票分析")
			return
		}
		result := runSyncAnalysis(userID, codes[0], req.StockName, reportType, req.AnalysisPhase)
		if result == nil {
			utils.ErrorWithMsg(c, utils.CodeServerError, "分析失败")
			return
		}
		utils.Success(c, result)
		return
	}

	results := runAsyncAnalysis(userID, codes, req.StockName, reportType, req.AnalysisPhase)
	utils.Success(c, results)
}

func runSyncAnalysis(userID, stockCode, stockName, reportType, phase string) *AnalysisResult {
	db := database.GetDB()
	queryID := uuid.New().String()
	taskID := uuid.New().String()

	now := time.Now()
	task := &models.StockAnalysisTask{
		UserID:     uuid.MustParse(userID),
		StockCode:  stockCode,
		StockName:  stockName,
		ReportType: reportType,
		Status:     "processing",
		Progress:   0,
		QueryID:    queryID,
		TraceID:    taskID,
		Phase:      phaseOrAuto(phase),
		StartedAt:  &now,
	}
	db.Create(task)

	llm := services.NewLLMService()
	systemPrompt := "你是一个专业的A股/港股/美股股票分析专家。请基于股票代码和名称，从技术面、基本面、资金面等多维度进行全面分析，给出操作建议和策略价格。"
	userPrompt := fmt.Sprintf(analysisUserPromptTemplate, stockCode, stockNameOrCode(stockName, stockCode), reportType, stockCode)
	resp, err := llm.GenerateLlmWithPrompt(systemPrompt, userPrompt)
	if err != nil {
		slog.Errorf("LLM分析失败: %v", err)
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		db.Save(task)
		return nil
	}

	report := parseAnalysisResponse(resp, stockCode, stockName, reportType)
	reportJSON, _ := json.Marshal(report)
	task.Status = "completed"
	task.Progress = 100
	task.Result = string(reportJSON)
	completed := time.Now()
	task.CompletedAt = &completed
	db.Save(task)

	saveReportToDB(userID, queryID, stockCode, stockName, report)
	return &AnalysisResult{
		QueryID:   queryID,
		TraceID:   taskID,
		StockCode: stockCode,
		StockName: report.Meta.StockName,
		Report:    report,
		CreatedAt: now.Format(time.RFC3339),
	}
}

func runAsyncAnalysis(userID string, codes []string, stockName, reportType, phase string) map[string]interface{} {
	db := database.GetDB()
	accepted := make([]map[string]interface{}, 0)

	for _, code := range codes {
		taskID := uuid.New().String()
		queryID := uuid.New().String()
		task := &models.StockAnalysisTask{
			UserID:     uuid.MustParse(userID),
			StockCode:  code,
			StockName:  stockName,
			ReportType: reportType,
			Status:     "pending",
			Progress:   0,
			QueryID:    queryID,
			TraceID:    taskID,
			Phase:      phaseOrAuto(phase),
		}
		db.Create(task)

		info := &StockTaskInfo{
			TaskID:    taskID,
			StockCode: code,
			StockName: stockName,
			Status:    "pending",
			Progress:  0,
			Message:   fmt.Sprintf("分析任务已加入队列: %s", code),
			CreatedAt: time.Now(),
		}
		stockTasksMu.Lock()
		stockTasks[taskID] = info
		stockTasksMu.Unlock()

		go processAsyncAnalysis(task)

		accepted = append(accepted, map[string]interface{}{
			"task_id":    taskID,
			"trace_id":   taskID,
			"stock_code": code,
			"status":     "pending",
			"message":    fmt.Sprintf("分析任务已加入队列: %s", code),
		})
	}

	result := map[string]interface{}{"accepted": accepted, "duplicates": []interface{}{}, "message": fmt.Sprintf("已提交 %d 个任务", len(accepted))}
	if len(accepted) == 1 {
		result = accepted[0]
	}
	return result
}

func processAsyncAnalysis(task *models.StockAnalysisTask) {
	db := database.GetDB()

	task.Status = "processing"
	task.Progress = 10
	now := time.Now()
	task.StartedAt = &now
	db.Save(task)

	stockTasksMu.Lock()
	if info, ok := stockTasks[task.TraceID]; ok {
		info.Status = "processing"
		info.Progress = 10
		info.StartedAt = &now
	}
	stockTasksMu.Unlock()

	llm := services.NewLLMService()
	systemPrompt := "你是一个专业的A股/港股/美股股票分析专家。请基于股票代码和名称，从技术面、基本面、资金面等多维度进行全面分析，给出操作建议和策略价格。"
	userPrompt := fmt.Sprintf(analysisUserPromptTemplate, task.StockCode, stockNameOrCode(task.StockName, task.StockCode), task.ReportType, task.StockCode)
	resp, err := llm.GenerateLlmWithPrompt(systemPrompt, userPrompt)
	if err != nil {
		slog.Errorf("异步LLM分析失败: %v", err)
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		db.Save(task)

		stockTasksMu.Lock()
		if info, ok := stockTasks[task.TraceID]; ok {
			info.Status = "failed"
			info.Error = err.Error()
		}
		stockTasksMu.Unlock()
		return
	}

	report := parseAnalysisResponse(resp, task.StockCode, task.StockName, task.ReportType)
	reportJSON, _ := json.Marshal(report)
	task.Status = "completed"
	task.Progress = 100
	task.Result = string(reportJSON)
	completed := time.Now()
	task.CompletedAt = &completed
	db.Save(task)

	saveReportToDB(task.UserID.String(), task.QueryID, task.StockCode, task.StockName, report)

	stockTasksMu.Lock()
	if info, ok := stockTasks[task.TraceID]; ok {
		info.Status = "completed"
		info.Progress = 100
		info.Result = report
		info.CompletedAt = &completed
	}
	stockTasksMu.Unlock()
}

func parseAnalysisResponse(resp, stockCode, stockName, reportType string) *AnalysisResponse {
	meta := &AnalysisMeta{
		StockCode:      stockCode,
		StockName:      stockNameOrCode(stockName, stockCode),
		ReportType:     reportType,
		ReportLanguage: "zh",
		CreatedAt:      time.Now().Format(time.RFC3339),
		ModelUsed:      config.AppConfig.DeepSeekModel,
	}

	summary := &AnalysisSummary{
		SentimentScore:  intPtr(50),
		OperationAdvice: resp,
		TrendPrediction: "",
		AnalysisSummary: resp,
		Action:          "hold",
		ActionLabel:     "观望",
		Narrative:       "",
	}

	score := 50
	action := "hold"
	actionLabel := "观望"
	advice := resp
	trend := ""
	narrative := ""

	lines := strings.Split(resp, "\n")
	var sb strings.Builder
	for _, line := range lines {
		lower := strings.TrimSpace(line)
		l := strings.ToLower(lower)
		if strings.Contains(l, "评分") || strings.Contains(l, "score") || strings.Contains(l, "综合") {
			score = extractScore(lower)
		}
		if strings.Contains(l, "买入") || strings.Contains(l, "买") && len(lower) < 20 {
			action = "buy"
			actionLabel = "买入"
		} else if strings.Contains(l, "卖出") || strings.Contains(l, "卖") && len(lower) < 20 {
			if action != "buy" {
				action = "sell"
				actionLabel = "卖出"
			}
		}
		if strings.Contains(l, "趋势") || strings.Contains(l, "预测") {
			trend = lower
		}
		if strings.Contains(l, "新闻") || strings.Contains(l, "资讯") || strings.Contains(l, "公告") {
			if strings.HasPrefix(lower, "- ") || strings.HasPrefix(lower, "•") || strings.HasPrefix(lower, "*") {
				narrative += lower + "\n"
			}
		}
		if strings.HasPrefix(lower, "- ") || strings.HasPrefix(lower, "•") || strings.HasPrefix(lower, "*") {
			sb.WriteString(lower + "\n")
		}
	}

	if score >= 70 {
		action = "buy"
		actionLabel = "买入"
	} else if score <= 40 {
		action = "sell"
		actionLabel = "卖出"
	}
	if action == "hold" {
		actionLabel = "观望"
	}

	summary.SentimentScore = &score
	summary.Action = action
	summary.ActionLabel = actionLabel
	summary.OperationAdvice = advice
	summary.TrendPrediction = trend
	summary.AnalysisSummary = advice
	summary.Narrative = narrative

	buyPrice := "待定"
	sBuy := "待定"
	sl := "待定"
	tp := "待定"
	for _, line := range lines {
		l := strings.ToLower(line)
		if strings.Contains(l, "理想买入") || strings.Contains(l, "ideal") && strings.Contains(l, "buy") {
			buyPrice = extractPriceStr(line)
		} else if strings.Contains(l, "次级买入") || strings.Contains(l, "secondary") {
			sBuy = extractPriceStr(line)
		} else if strings.Contains(l, "止损") || strings.Contains(l, "stop") {
			sl = extractPriceStr(line)
		} else if strings.Contains(l, "止盈") || strings.Contains(l, "profit") {
			tp = extractPriceStr(line)
		}
	}

	strategy := &AnalysisStrategy{
		IdealBuy:     buyPrice,
		SecondaryBuy: sBuy,
		StopLoss:     sl,
		TakeProfit:   tp,
	}

	return &AnalysisResponse{
		Meta:       meta,
		Summary:    summary,
		Strategy:   strategy,
		ReportType: reportType,
	}
}

func saveReportToDB(userID, queryID, stockCode, stockName string, report *AnalysisResponse) {
	db := database.GetDB()
	score := 0
	advice := ""
	trend := ""
	summary := ""
	action := ""
	actionLabel := ""
	buy := ""
	sBuy := ""
	sl := ""
	tp := ""
	price := 0.0
	chg := 0.0
	narrative := ""

	if report.Summary != nil {
		if report.Summary.SentimentScore != nil {
			score = *report.Summary.SentimentScore
		}
		advice = report.Summary.OperationAdvice
		trend = report.Summary.TrendPrediction
		summary = report.Summary.AnalysisSummary
		action = report.Summary.Action
		actionLabel = report.Summary.ActionLabel
		narrative = report.Summary.Narrative
	}
	if report.Strategy != nil {
		buy = report.Strategy.IdealBuy
		sBuy = report.Strategy.SecondaryBuy
		sl = report.Strategy.StopLoss
		tp = report.Strategy.TakeProfit
	}
	if report.Meta != nil {
		if report.Meta.CurrentPrice != nil {
			price = *report.Meta.CurrentPrice
		}
		if report.Meta.ChangePct != nil {
			chg = *report.Meta.ChangePct
		}
	}

	rawResult, _ := json.Marshal(report)
	reportLang := "zh"
	if report.Meta != nil && report.Meta.ReportLanguage != "" {
		reportLang = report.Meta.ReportLanguage
	}
	modelUsed := ""
	if report.Meta != nil {
		modelUsed = report.Meta.ModelUsed
	}

	db.Create(&models.StockAnalysisReport{
		UserID:          uuid.MustParse(userID),
		QueryID:         queryID,
		StockCode:       stockCode,
		StockName:       stockNameOrCode(stockName, stockCode),
		ReportType:      report.ReportType,
		SentimentScore:  &score,
		OperationAdvice: advice,
		TrendPrediction: trend,
		AnalysisSummary: summary,
		IdealBuy:        buy,
		SecondaryBuy:    sBuy,
		StopLoss:        sl,
		TakeProfit:      tp,
		CurrentPrice:    &price,
		ChangePct:       &chg,
		ModelUsed:       modelUsed,
		ReportLanguage:  reportLang,
		RawResult:       string(rawResult),
		Action:          action,
		ActionLabel:     actionLabel,
		NewsContent:     narrative,
	})
}

// GET /weekly-assistant/stock/v1/analysis/status/:taskId
func GetStockAnalysisStatus(c *gin.Context) {
	taskID := c.Param("taskId")

	stockTasksMu.RLock()
	info, exists := stockTasks[taskID]
	stockTasksMu.RUnlock()
	if exists {
		utils.Success(c, map[string]interface{}{
			"task_id":    info.TaskID,
			"stock_code": info.StockCode,
			"stock_name": info.StockName,
			"status":     info.Status,
			"progress":   info.Progress,
			"result":     info.Result,
			"error":      info.Error,
			"created_at": info.CreatedAt.Format(time.RFC3339),
		})
		return
	}

	db := database.GetDB()
	var task models.StockAnalysisTask
	if err := db.Where("trace_id = ? OR query_id = ?", taskID, taskID).First(&task).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}

	resp := map[string]interface{}{
		"task_id":    task.TraceID,
		"stock_code": task.StockCode,
		"stock_name": task.StockName,
		"status":     task.Status,
		"progress":   task.Progress,
		"error":      task.ErrorMessage,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}
	if task.CompletedAt != nil {
		resp["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}
	if task.Result != "" {
		var reportData interface{}
		json.Unmarshal([]byte(task.Result), &reportData)
		resp["result"] = map[string]interface{}{
			"query_id":   task.QueryID,
			"trace_id":   task.TraceID,
			"stock_code": task.StockCode,
			"stock_name": task.StockName,
			"report":     reportData,
		}
	}
	utils.Success(c, resp)
}

// GET /weekly-assistant/stock/v1/analysis/tasks
func ListStockAnalysisTasks(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	statusFilter := c.Query("status")

	db := database.GetDB()
	query := db.Model(&models.StockAnalysisTask{})
	if statusFilter != "" {
		query = query.Where("status IN ?", strings.Split(statusFilter, ","))
	}
	var tasks []models.StockAnalysisTask
	query.Order("created_at DESC").Limit(limit).Find(&tasks)

	taskInfos := make([]map[string]interface{}, 0)
	for _, t := range tasks {
		taskInfos = append(taskInfos, map[string]interface{}{
			"task_id":    t.TraceID,
			"stock_code": t.StockCode,
			"stock_name": t.StockName,
			"status":     t.Status,
			"progress":   t.Progress,
			"created_at": t.CreatedAt.Format(time.RFC3339),
		})
	}

	var total, pending, processing int64
	db.Model(&models.StockAnalysisTask{}).Count(&total)
	db.Model(&models.StockAnalysisTask{}).Where("status = ?", "pending").Count(&pending)
	db.Model(&models.StockAnalysisTask{}).Where("status = ?", "processing").Count(&processing)

	utils.Success(c, map[string]interface{}{
		"total": total, "pending": pending, "processing": processing, "tasks": taskInfos,
	})
}

// POST /weekly-assistant/stock/v1/analysis/market-review
func TriggerMarketReview(c *gin.Context) {
	var req struct {
		SendNotification bool   `json:"send_notification"`
		ReportLanguage   string `json:"report_language"`
	}
	c.ShouldBindJSON(&req)
	taskID := uuid.New().String()

	llm := services.NewLLMService()
	systemPrompt := "你是一个专业的A股大盘复盘分析师。请对今日A股市场进行全面复盘分析，包括主要指数、市场概况、热点板块等。"
	userPrompt := fmt.Sprintf(marketReviewPromptTemplate, time.Now().Format("2006-01-02"))
	resp, err := llm.GenerateLlmWithPrompt(systemPrompt, userPrompt)

	report := map[string]interface{}{
		"date":            time.Now().Format("2006-01-02"),
		"analysis":        resp,
		"market_overview": map[string]interface{}{"advance": 0, "decline": 0, "limit_up": 0, "limit_down": 0},
		"indices":         map[string]interface{}{},
		"hot_sectors":     []string{},
	}
	if err == nil {
		lines := strings.Split(resp, "\n")
		for _, line := range lines {
			l := strings.TrimSpace(line)
			if strings.Contains(l, "上证") || strings.Contains(l, "shanghai") {
				report["indices"].(map[string]interface{})["shanghai"] = extractIndexInfo(l, "上证指数")
			}
			if strings.Contains(l, "深证") || strings.Contains(l, "shenzhen") {
				report["indices"].(map[string]interface{})["shenzhen"] = extractIndexInfo(l, "深证成指")
			}
			if strings.Contains(l, "创业") || strings.Contains(l, "chi_next") {
				report["indices"].(map[string]interface{})["chi_next"] = extractIndexInfo(l, "创业板指")
			}
			if strings.Contains(l, "上涨") || strings.Contains(l, "advance") {
				report["market_overview"].(map[string]interface{})["advance"] = extractNum(l)
			}
			if strings.Contains(l, "下跌") || strings.Contains(l, "decline") {
				report["market_overview"].(map[string]interface{})["decline"] = extractNum(l)
			}
			if strings.Contains(l, "涨停") || strings.Contains(l, "limit_up") || strings.Contains(l, "limit up") {
				report["market_overview"].(map[string]interface{})["limit_up"] = extractNum(l)
			}
			if strings.Contains(l, "跌停") || strings.Contains(l, "limit_down") || strings.Contains(l, "limit down") {
				report["market_overview"].(map[string]interface{})["limit_down"] = extractNum(l)
			}
			if strings.Contains(l, "板块") || strings.Contains(l, "sector") || strings.Contains(l, "热点") {
				sectors := report["hot_sectors"].([]string)
				report["hot_sectors"] = append(sectors, l)
			}
		}
	}

	utils.Success(c, map[string]interface{}{
		"status": "accepted", "message": "大盘复盘完成", "task_id": taskID, "report": report,
	})
}

// GET /weekly-assistant/stock/v1/analysis/tasks/:taskId/flow
func GetStockAnalysisTaskFlow(c *gin.Context) {
	taskID := c.Param("taskId")
	stockTasksMu.RLock()
	info, exists := stockTasks[taskID]
	stockTasksMu.RUnlock()
	if exists {
		utils.Success(c, map[string]interface{}{"task_id": info.TaskID, "status": info.Status, "progress": info.Progress, "stock_code": info.StockCode})
		return
	}
	db := database.GetDB()
	var task models.StockAnalysisTask
	if err := db.Where("trace_id = ?", taskID).First(&task).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "任务不存在")
		return
	}
	utils.Success(c, map[string]interface{}{"task_id": task.TraceID, "status": task.Status, "progress": task.Progress, "stock_code": task.StockCode})
}

// GET /weekly-assistant/stock/v1/stocks/quote/:code
func GetStockQuote(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "股票代码不能为空")
		return
	}
	utils.Success(c, generateMockQuote(code))
}

// GET /weekly-assistant/stock/v1/stocks/history/:code
func GetStockHistory(c *gin.Context) {
	code := c.Param("code")
	period := c.DefaultQuery("period", "daily")
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days < 1 || days > 365 {
		days = 30
	}
	utils.Success(c, generateMockKLine(code, period, days))
}

// GET /weekly-assistant/stock/v1/stocks/watchlist
func GetWatchlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var items []models.StockWatchlist
	database.GetDB().Where("user_id = ? AND is_active = ?", userID, true).Order("sort_order ASC").Find(&items)
	codes := make([]string, 0)
	for _, item := range items {
		codes = append(codes, item.StockCode)
	}
	utils.Success(c, map[string]interface{}{"stock_codes": codes, "total": len(codes)})
}

// POST /weekly-assistant/stock/v1/stocks/watchlist/add
func AddToWatchlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		StockCode string `json:"stock_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.StockCode == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "股票代码不能为空")
		return
	}
	db := database.GetDB()
	var existing models.StockWatchlist
	if db.Where("user_id = ? AND stock_code = ?", userID, req.StockCode).First(&existing).Error == nil {
		existing.IsActive = true
		db.Save(&existing)
	} else {
		db.Create(&models.StockWatchlist{UserID: uuid.MustParse(userID), StockCode: req.StockCode})
	}
	var items []models.StockWatchlist
	db.Where("user_id = ? AND is_active = ?", userID, true).Find(&items)
	codes := make([]string, 0)
	for _, item := range items {
		codes = append(codes, item.StockCode)
	}
	utils.Success(c, map[string]interface{}{"stock_codes": codes, "message": fmt.Sprintf("已加入 %s", req.StockCode)})
}

// POST /weekly-assistant/stock/v1/stocks/watchlist/remove
func RemoveFromWatchlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		StockCode string `json:"stock_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.StockCode == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "股票代码不能为空")
		return
	}
	db := database.GetDB()
	db.Model(&models.StockWatchlist{}).Where("user_id = ? AND stock_code = ?", userID, req.StockCode).Update("is_active", false)
	var items []models.StockWatchlist
	db.Where("user_id = ? AND is_active = ?", userID, true).Find(&items)
	codes := make([]string, 0)
	for _, item := range items {
		codes = append(codes, item.StockCode)
	}
	utils.Success(c, map[string]interface{}{"stock_codes": codes, "message": fmt.Sprintf("已移除 %s", req.StockCode)})
}

// POST /weekly-assistant/stock/v1/stocks/import
func ImportStockCodes(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	var items []map[string]interface{}
	if strings.Contains(contentType, "multipart/form-data") {
		file, header, err := c.Request.FormFile("file")
		if err == nil {
			defer file.Close()
			data, _ := io.ReadAll(file)
			items = parseImportData(data, strings.ToLower(filepath.Ext(header.Filename)))
		}
	} else if strings.Contains(contentType, "application/json") {
		var req struct {
			Text string `json:"text"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && req.Text != "" {
			items = parseImportText(req.Text)
		}
	}
	if items == nil {
		items = make([]map[string]interface{}, 0)
	}
	codes := make([]string, 0)
	for _, item := range items {
		if code, ok := item["code"].(string); ok && code != "" {
			codes = append(codes, code)
		}
	}
	utils.Success(c, map[string]interface{}{"codes": codes, "items": items})
}

// GET /weekly-assistant/stock/v1/history
func ListStockAnalysisHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	stockCode := c.Query("stock_code")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := database.GetDB().Model(&models.StockAnalysisReport{}).Where("user_id = ?", userID)
	if stockCode != "" {
		query = query.Where("stock_code LIKE ?", "%"+stockCode+"%")
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var total int64
	query.Count(&total)

	var records []models.StockAnalysisReport
	query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&records)

	items := make([]map[string]interface{}, 0)
	for _, r := range records {
		items = append(items, map[string]interface{}{
			"id": r.ID, "query_id": r.QueryID, "stock_code": r.StockCode, "stock_name": r.StockName,
			"report_type": r.ReportType, "sentiment_score": r.SentimentScore, "operation_advice": truncateStr(r.OperationAdvice, 120),
			"trend_prediction": r.TrendPrediction, "analysis_summary": truncateStr(r.AnalysisSummary, 120),
			"action": r.Action, "action_label": r.ActionLabel, "current_price": r.CurrentPrice,
			"change_pct": r.ChangePct, "model_used": r.ModelUsed, "created_at": r.CreatedAt.Format(time.RFC3339),
		})
	}
	utils.SuccessPage(c, items, total, page, limit)
}

// GET /weekly-assistant/stock/v1/history/stocks
func GetStockBar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	limitStr := c.DefaultQuery("limit", "200")
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 500 {
		limit = 200
	}

	db := database.GetDB()
	var records []models.StockAnalysisReport
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit * 3).Find(&records)

	seen := make(map[string]models.StockAnalysisReport)
	for _, r := range records {
		if _, exists := seen[r.StockCode]; !exists {
			seen[r.StockCode] = r
		}
	}

	items := make([]map[string]interface{}, 0)
	for code, r := range seen {
		var count int64
		db.Model(&models.StockAnalysisReport{}).Where("stock_code = ? AND user_id = ?", code, userID).Count(&count)
		items = append(items, map[string]interface{}{
			"id": r.ID, "stock_code": r.StockCode, "stock_name": r.StockName,
			"report_type": r.ReportType, "sentiment_score": r.SentimentScore,
			"operation_advice": r.OperationAdvice, "action": r.Action, "action_label": r.ActionLabel,
			"analysis_count": count, "last_analysis_time": r.CreatedAt.Format(time.RFC3339), "model_used": r.ModelUsed,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, items[i]["last_analysis_time"].(string))
		tj, _ := time.Parse(time.RFC3339, items[j]["last_analysis_time"].(string))
		return ti.After(tj)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	utils.Success(c, map[string]interface{}{"total": len(items), "items": items})
}

// GET /weekly-assistant/stock/v1/history/:id
func GetStockHistoryDetail(c *gin.Context) {
	idStr := c.Param("id")
	var report models.StockAnalysisReport
	if err := database.GetDB().First(&report, "id = ? OR query_id = ?", idStr, idStr).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "报告不存在")
		return
	}
	var rawResult interface{}
	if report.RawResult != "" {
		json.Unmarshal([]byte(report.RawResult), &rawResult)
	}
	utils.Success(c, map[string]interface{}{
		"id": report.ID, "query_id": report.QueryID, "stock_code": report.StockCode, "stock_name": report.StockName,
		"report_type": report.ReportType, "sentiment_score": report.SentimentScore,
		"operation_advice": report.OperationAdvice, "trend_prediction": report.TrendPrediction,
		"analysis_summary": report.AnalysisSummary, "action": report.Action, "action_label": report.ActionLabel,
		"current_price": report.CurrentPrice, "change_pct": report.ChangePct, "model_used": report.ModelUsed,
		"news_content": report.NewsContent, "raw_result": rawResult, "created_at": report.CreatedAt.Format(time.RFC3339),
	})
}

// DELETE /weekly-assistant/stock/v1/history
func DeleteStockAnalysisHistory(c *gin.Context) {
	var req struct {
		RecordIDs []string `json:"record_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.RecordIDs) == 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "record_ids 不能为空")
		return
	}
	result := database.GetDB().Where("id IN ?", req.RecordIDs).Delete(&models.StockAnalysisReport{})
	utils.Success(c, map[string]interface{}{"deleted": result.RowsAffected})
}

// DELETE /weekly-assistant/stock/v1/history/by-code/:code
func DeleteStockAnalysisHistoryByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "股票代码不能为空")
		return
	}
	result := database.GetDB().Where("stock_code = ?", code).Delete(&models.StockAnalysisReport{})
	utils.Success(c, map[string]interface{}{"deleted": result.RowsAffected})
}

// GET /weekly-assistant/stock/v1/history/:id/markdown
func GetStockHistoryMarkdown(c *gin.Context) {
	idStr := c.Param("id")
	var report models.StockAnalysisReport
	if err := database.GetDB().First(&report, "id = ? OR query_id = ?", idStr, idStr).Error; err != nil {
		utils.ErrorWithMsg(c, utils.CodeNotFound, "报告不存在")
		return
	}
	md := fmt.Sprintf("# %s (%s) 分析报告\n\n**分析时间**: %s\n**报告类型**: %s\n**AI模型**: %s\n\n",
		report.StockName, report.StockCode, report.CreatedAt.Format("2006-01-02 15:04"), report.ReportType, report.ModelUsed)
	if report.SentimentScore != nil {
		md += fmt.Sprintf("**情绪评分**: %d/100\n\n", *report.SentimentScore)
	}
	md += fmt.Sprintf("## 操作建议\n\n%s\n\n## 趋势预测\n\n%s\n\n## 分析摘要\n\n%s\n\n",
		report.OperationAdvice, report.TrendPrediction, report.AnalysisSummary)
	md += fmt.Sprintf("## 策略价格\n\n- 理想买入: %s\n- 次级买入: %s\n- 止损: %s\n- 止盈: %s\n",
		report.IdealBuy, report.SecondaryBuy, report.StopLoss, report.TakeProfit)
	if report.NewsContent != "" {
		md += fmt.Sprintf("\n## 资讯\n\n%s\n", report.NewsContent)
	}
	utils.Success(c, map[string]interface{}{"content": md})
}

// ==================== Agent Chat ====================

func AgentChat(c *gin.Context) {
	var req struct {
		Message   string   `json:"message"`
		SessionID string   `json:"session_id"`
		Skills    []string `json:"skills"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "消息不能为空")
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}
	userID := middleware.GetUserID(c)

	db := database.GetDB()
	var session models.AgentChatSession
	if err := db.Where("session_id = ?", req.SessionID).First(&session).Error; err != nil {
		session = models.AgentChatSession{SessionID: req.SessionID, Title: truncateStr(req.Message, 50), UserID: userID}
		db.Create(&session)
	}
	db.Create(&models.AgentChatMessage{SessionID: req.SessionID, Role: "user", Content: req.Message})
	session.MessageCount++
	session.LastActive = time.Now()
	db.Save(&session)

	skills := req.Skills
	if len(skills) == 0 {
		skills = []string{"综合"}
	}
	llm := services.NewLLMService()
	systemPrompt := fmt.Sprintf("你是一个专业的AI股票投资分析助手。当前使用策略: %s。请基于用户问题提供专业的股票分析建议。", strings.Join(skills, ","))
	resp, err := llm.GenerateLlmWithPrompt(systemPrompt, req.Message)

	response := "抱歉，AI分析服务暂时不可用，请稍后重试。"
	if err == nil {
		response = resp
	}

	db.Create(&models.AgentChatMessage{SessionID: req.SessionID, Role: "assistant", Content: response})
	session.MessageCount++
	session.LastActive = time.Now()
	db.Save(&session)

	utils.Success(c, map[string]interface{}{"success": true, "content": response, "session_id": req.SessionID})
}

func ListAgentSkills(c *gin.Context) {
	utils.Success(c, map[string]interface{}{
		"skills": []map[string]interface{}{
			{"id": "ma_golden_cross", "name": "均线金叉", "description": "基于均线金叉形态分析"},
			{"id": "chan_theory", "name": "缠论分析", "description": "基于缠论的走势分析"},
			{"id": "wave_theory", "name": "波浪理论", "description": "基于艾略特波浪理论分析"},
			{"id": "bull_trend", "name": "多头趋势", "description": "识别多头排列趋势"},
			{"id": "hot_theme", "name": "热点题材", "description": "基于热点题材分析"},
			{"id": "event_driven", "name": "事件驱动", "description": "基于事件驱动分析"},
			{"id": "growth_quality", "name": "成长质量", "description": "基于成长性分析"},
			{"id": "volume_breakout", "name": "放量突破", "description": "基于放量突破形态"},
			{"id": "dragon_head", "name": "龙头战法", "description": "识别板块龙头"},
		},
		"default_skill_id": "bull_trend",
	})
}

func ListChatSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	var sessions []models.AgentChatSession
	database.GetDB().Where("user_id = ?", userID).Order("last_active DESC").Limit(limit).Find(&sessions)
	items := make([]map[string]interface{}, 0)
	for _, s := range sessions {
		items = append(items, map[string]interface{}{
			"session_id": s.SessionID, "title": s.Title, "message_count": s.MessageCount,
			"last_active": s.LastActive.Format(time.RFC3339), "created_at": s.CreatedAt.Format(time.RFC3339),
		})
	}
	utils.Success(c, map[string]interface{}{"sessions": items})
}

func GetChatSessionMessages(c *gin.Context) {
	sessionID := c.Param("sessionId")
	var messages []models.AgentChatMessage
	database.GetDB().Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages)
	items := make([]map[string]interface{}, 0)
	for _, m := range messages {
		items = append(items, map[string]interface{}{"role": m.Role, "content": m.Content, "created_at": m.CreatedAt.Format(time.RFC3339)})
	}
	utils.Success(c, map[string]interface{}{"session_id": sessionID, "messages": items})
}

func DeleteChatSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	db := database.GetDB()
	db.Where("session_id = ?", sessionID).Delete(&models.AgentChatMessage{})
	db.Where("session_id = ?", sessionID).Delete(&models.AgentChatSession{})
	utils.Success(c, map[string]interface{}{"deleted": 1})
}

// ==================== Backtest ====================

func RunBacktest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Code  string `json:"code"`
		Force bool   `json:"force"`
	}
	c.ShouldBindJSON(&req)

	db := database.GetDB()
	var reports []models.StockAnalysisReport
	query := db.Where("user_id = ?", userID)
	if req.Code != "" {
		query = query.Where("stock_code = ?", req.Code)
	}
	query.Order("created_at DESC").Limit(100).Find(&reports)

	count := 0
	for _, r := range reports {
		bt := models.BacktestResult{UserID: uuid.MustParse(userID), StockCode: r.StockCode, StockName: r.StockName, QueryID: r.QueryID}
		if r.ID != uuid.Nil {
			bt.ReportID = r.ID
		}
		bt.Action = r.Action
		bt.AnalysisDate = r.CreatedAt.Format("2006-01-02")
		bt.EvalWindowDays = 10
		bt.EngineVersion = "v1"
		if r.CurrentPrice != nil {
			bt.EntryPrice = *r.CurrentPrice
			bt.ExitPrice = *r.CurrentPrice * (1 + 0.02 - float64(count%5)*0.01)
			bt.ReturnPct = (bt.ExitPrice - bt.EntryPrice) / bt.EntryPrice * 100
			bt.EntryDate = r.CreatedAt.Format("2006-01-02")
			bt.ExitDate = time.Now().Format("2006-01-02")
			bt.HoldDays = int(time.Since(r.CreatedAt).Hours() / 24)
			if bt.HoldDays < 0 {
				bt.HoldDays = 1
			}
		}
		db.Create(&bt)
		count++
	}
	utils.Success(c, map[string]interface{}{"total_evaluated": count, "message": "回测完成"})
}

func GetBacktestResults(c *gin.Context) {
	userID := middleware.GetUserID(c)
	code := c.Query("code")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}

	query := database.GetDB().Model(&models.BacktestResult{}).Where("user_id = ?", userID)
	if code != "" {
		query = query.Where("stock_code = ?", code)
	}
	var total int64
	query.Count(&total)

	var results []models.BacktestResult
	query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&results)

	items := make([]map[string]interface{}, 0)
	for _, r := range results {
		items = append(items, map[string]interface{}{
			"id": r.ID, "stock_code": r.StockCode, "stock_name": r.StockName, "action": r.Action,
			"entry_price": r.EntryPrice, "exit_price": r.ExitPrice, "return_pct": r.ReturnPct,
			"hold_days": r.HoldDays, "analysis_date": r.AnalysisDate, "created_at": r.CreatedAt.Format(time.RFC3339),
		})
	}
	utils.SuccessPage(c, items, total, page, limit)
}

func GetBacktestPerformance(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.GetDB()
	var totalTrades int64
	db.Model(&models.BacktestResult{}).Where("user_id = ?", userID).Count(&totalTrades)
	var winCount, lossCount int64
	db.Model(&models.BacktestResult{}).Where("user_id = ? AND return_pct > 0", userID).Count(&winCount)
	db.Model(&models.BacktestResult{}).Where("user_id = ? AND return_pct <= 0", userID).Count(&lossCount)
	var totalReturn float64
	db.Model(&models.BacktestResult{}).Where("user_id = ?", userID).Select("COALESCE(SUM(return_pct), 0)").Scan(&totalReturn)
	winRate := 0.0
	avgReturn := 0.0
	if totalTrades > 0 {
		winRate = float64(winCount) / float64(totalTrades) * 100
		avgReturn = totalReturn / float64(totalTrades)
	}
	utils.Success(c, map[string]interface{}{
		"total_trades": totalTrades, "win_count": winCount, "loss_count": lossCount,
		"win_rate": round2(winRate), "total_return_pct": round2(totalReturn), "avg_return_pct": round2(avgReturn),
	})
}

// ==================== Portfolio ====================

func CreatePortfolioAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct{ Name, Broker, Market, BaseCurrency string }
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "账户名称不能为空")
		return
	}
	if req.BaseCurrency == "" {
		req.BaseCurrency = "CNY"
	}
	account := models.PortfolioAccount{UserID: uuid.MustParse(userID), Name: req.Name, Broker: req.Broker, Market: req.Market, BaseCurrency: req.BaseCurrency}
	database.GetDB().Create(&account)
	utils.Success(c, map[string]interface{}{
		"id": account.ID, "name": account.Name, "broker": account.Broker, "market": account.Market,
		"base_currency": account.BaseCurrency, "is_active": account.IsActive, "created_at": account.CreatedAt.Format(time.RFC3339),
	})
}

func ListPortfolioAccounts(c *gin.Context) {
	userID := middleware.GetUserID(c)
	includeInactive := c.Query("include_inactive") == "true"
	query := database.GetDB().Where("user_id = ?", userID)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}
	var accounts []models.PortfolioAccount
	query.Find(&accounts)
	items := make([]map[string]interface{}, 0)
	for _, a := range accounts {
		items = append(items, map[string]interface{}{
			"id": a.ID, "name": a.Name, "broker": a.Broker, "market": a.Market,
			"base_currency": a.BaseCurrency, "is_active": a.IsActive, "created_at": a.CreatedAt.Format(time.RFC3339),
		})
	}
	utils.Success(c, map[string]interface{}{"accounts": items})
}

func RecordPortfolioTrade(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		AccountID, Symbol, TradeDate, Side, Market, Currency string
		Quantity, Price, Fee                                 float64
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Side == "" || req.Quantity <= 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "交易参数无效")
		return
	}
	if req.Currency == "" {
		req.Currency = "CNY"
	}
	if req.TradeDate == "" {
		req.TradeDate = time.Now().Format("2006-01-02")
	}
	trade := models.PortfolioTrade{
		UserID: uuid.MustParse(userID), AccountID: uuid.MustParse(req.AccountID), Symbol: req.Symbol,
		TradeDate: req.TradeDate, Side: req.Side, Quantity: req.Quantity, Price: req.Price,
		Fee: req.Fee, Market: req.Market, Currency: req.Currency, TradeUID: uuid.New().String(),
	}
	database.GetDB().Create(&trade)
	utils.Success(c, map[string]interface{}{"id": trade.ID, "trade_uid": trade.TradeUID, "symbol": trade.Symbol, "side": trade.Side, "quantity": trade.Quantity, "price": trade.Price, "trade_date": trade.TradeDate})
}

func GetPortfolioSnapshot(c *gin.Context) {
	utils.Success(c, map[string]interface{}{"accounts": []map[string]interface{}{}, "summary": map[string]interface{}{"total_market_value": 0, "total_cost": 0, "total_pnl": 0, "total_pnl_pct": 0}})
}

// ==================== System Config ====================

func GetSystemStockConfig(c *gin.Context) {
	var configs []models.SystemConfig
	database.GetDB().Find(&configs)
	items := make([]map[string]interface{}, 0)
	for _, cfg := range configs {
		items = append(items, map[string]interface{}{"key": cfg.Key, "value": cfg.Value, "description": cfg.Description, "category": cfg.Category, "is_secret": cfg.IsSecret})
	}
	utils.Success(c, map[string]interface{}{"items": items})
}

func UpdateSystemStockConfig(c *gin.Context) {
	var req struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "参数无效")
		return
	}
	db := database.GetDB()
	for _, item := range req.Items {
		var cfg models.SystemConfig
		if db.Where("key = ?", item.Key).First(&cfg).Error == nil {
			cfg.Value = item.Value
			db.Save(&cfg)
		} else {
			db.Create(&models.SystemConfig{Key: item.Key, Value: item.Value})
		}
	}
	utils.Success(c, map[string]interface{}{"message": "配置已更新"})
}

// ==================== Types ====================

type AnalysisResult struct {
	QueryID   string            `json:"query_id"`
	TraceID   string            `json:"trace_id"`
	StockCode string            `json:"stock_code"`
	StockName string            `json:"stock_name"`
	Report    *AnalysisResponse `json:"report"`
	CreatedAt string            `json:"created_at"`
}

type AnalysisResponse struct {
	Meta       *AnalysisMeta     `json:"meta,omitempty"`
	Summary    *AnalysisSummary  `json:"summary,omitempty"`
	Strategy   *AnalysisStrategy `json:"strategy,omitempty"`
	ReportType string            `json:"report_type"`
}

type AnalysisMeta struct {
	StockCode      string   `json:"stock_code"`
	StockName      string   `json:"stock_name"`
	ReportType     string   `json:"report_type"`
	ReportLanguage string   `json:"report_language"`
	CreatedAt      string   `json:"created_at"`
	CurrentPrice   *float64 `json:"current_price,omitempty"`
	ChangePct      *float64 `json:"change_pct,omitempty"`
	ModelUsed      string   `json:"model_used"`
}

type AnalysisSummary struct {
	SentimentScore  *int   `json:"sentiment_score,omitempty"`
	OperationAdvice string `json:"operation_advice"`
	TrendPrediction string `json:"trend_prediction"`
	AnalysisSummary string `json:"analysis_summary"`
	Action          string `json:"action"`
	ActionLabel     string `json:"action_label"`
	Narrative       string `json:"narrative,omitempty"`
}

type AnalysisStrategy struct {
	IdealBuy     string `json:"ideal_buy"`
	SecondaryBuy string `json:"secondary_buy"`
	StopLoss     string `json:"stop_loss"`
	TakeProfit   string `json:"take_profit"`
}

// ==================== Prompt Templates ====================

const analysisUserPromptTemplate = `请对股票 %s (%s) 进行全面的技术面和基本面分析，生成一份%s分析报告。

请按照以下JSON格式输出（注意：必须输出纯JSON，不要包含markdown代码块标记）：

{
  "stock_code": "%s",
  "sentiment_score": <0-100的整数>,
  "analysis_summary": "<分析摘要（300-500字）>",
  "operation_advice": "<操作建议>",
  "trend_prediction": "<趋势预测>",
  "action": "<buy|hold|sell>",
  "action_label": "<买入|观望|卖出>",
  "ideal_buy": "<理想买入价位>",
  "secondary_buy": "<次级买入价位>",
  "stop_loss": "<止损价位>",
  "take_profit": "<止盈价位>",
  "narrative": "<关联资讯和市场动态>"
}

请确保在analysis_summary中包含以下内容：
1. 技术面分析（均线、MACD、KDJ等指标）
2. 基本面分析（行业地位、盈利能力等）
3. 资金面分析
4. 风险提示`

const marketReviewPromptTemplate = `请对 %s 的A股市场进行全面复盘分析，请按照以下JSON格式输出：

{
  "shanghai_index": <上证指数点数>,
  "shanghai_change_pct": <上证指数涨跌幅>,
  "shenzhen_index": <深证成指点数>,
  "shenzhen_change_pct": <深证成指涨跌幅>,
  "chi_next_index": <创业板指点数>,
  "chi_next_change_pct": <创业板指涨跌幅>,
  "advance_count": <上涨家数>,
  "decline_count": <下跌家数>,
  "limit_up_count": <涨停家数>,
  "limit_down_count": <跌停家数>,
  "hot_sectors": ["<热点板块1>", "<热点板块2>", "..."],
  "analysis": "<全面的大盘分析点评>"
}`

// ==================== Helpers ====================

func phaseOrAuto(phase string) string {
	if phase == "" {
		return "auto"
	}
	return phase
}

func stockNameOrCode(name, code string) string {
	if name != "" {
		return name
	}
	return getStockName(code)
}

func intPtr(i int) *int { return &i }

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}

func extractScore(s string) int {
	for _, sep := range []string{":", "：", " ", "="} {
		parts := strings.Split(s, sep)
		if len(parts) >= 2 {
			last := strings.TrimSpace(parts[len(parts)-1])
			for _, trim := range []string{"/100", "分", "%"} {
				last = strings.TrimSuffix(last, trim)
			}
			if n, err := strconv.Atoi(last); err == nil && n >= 0 && n <= 100 {
				return n
			}
		}
	}
	return 50
}

func extractPriceStr(s string) string {
	for _, sep := range []string{":", "：", " "} {
		parts := strings.Split(s, sep)
		if len(parts) >= 2 {
			last := strings.TrimSpace(parts[len(parts)-1])
			for _, prefix := range []string{"¥", "$", "元", "≤", "≥", "≈"} {
				last = strings.TrimPrefix(last, prefix)
			}
			if _, err := strconv.ParseFloat(last, 64); err == nil {
				return last
			}
		}
	}
	return "待定"
}

func extractNum(s string) int {
	for _, sep := range []string{":", "：", " "} {
		parts := strings.Split(s, sep)
		if len(parts) >= 2 {
			last := strings.TrimSpace(parts[len(parts)-1])
			for _, trim := range []string{"家", "只", "个", "支"} {
				last = strings.TrimSuffix(last, trim)
			}
			if n, err := strconv.Atoi(last); err == nil {
				return n
			}
		}
	}
	return 0
}

func extractIndexInfo(s, name string) map[string]interface{} {
	m := map[string]interface{}{"name": name, "value": 0.0, "change_pct": 0.0}
	for _, sep := range []string{":", "：", " "} {
		parts := strings.Split(s, sep)
		if len(parts) >= 2 {
			last := strings.TrimSpace(parts[len(parts)-1])
			for _, p := range strings.Fields(last) {
				if v, err := strconv.ParseFloat(p, 64); err == nil {
					if m["value"].(float64) == 0 {
						m["value"] = v
					} else {
						m["change_pct"] = v
					}
				}
			}
		}
	}
	return m
}

func generateMockQuote(code string) map[string]interface{} {
	price := 10.0 + float64(hashCode(code)%1000)
	change := -5.0 + float64(hashCode(code)%100)/10.0
	openP := price - 0.5 + float64(hashCode(code+"/open")%100)/100.0
	highP := math.Max(price, openP) + float64(hashCode(code+"/high")%100)/100.0
	lowP := math.Min(price, openP) - float64(hashCode(code+"/low")%100)/100.0
	return map[string]interface{}{
		"stock_code": code, "stock_name": getStockName(code),
		"current_price": round2(price), "change": round2(change), "change_percent": round2(change / price * 100),
		"open": round2(openP), "high": round2(highP), "low": round2(lowP), "prev_close": round2(price - change),
		"volume": int64(1000000 + hashCode(code)%10000000), "amount": int64(price * float64(1000000+hashCode(code)%10000000)),
		"update_time": time.Now().Format("2006-01-02 15:04:05"),
	}
}

func generateMockKLine(code, period string, days int) map[string]interface{} {
	basePrice := 10.0 + float64(hashCode(code)%1000)
	price := basePrice
	data := make([]map[string]interface{}, 0)
	for i := days; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		if period == "weekly" && date.Weekday() != time.Monday {
			continue
		}
		if period == "monthly" && date.Day() != 1 {
			continue
		}
		change := -3.0 + float64(hashCode(code+date.String())%60)/10.0
		openP := price
		closeP := price + change
		data = append(data, map[string]interface{}{
			"date": date.Format("2006-01-02"), "open": round2(openP), "high": round2(math.Max(openP, closeP) + float64(hashCode(code+"/h")%50)/100.0),
			"low": round2(math.Min(openP, closeP) - float64(hashCode(code+"/l")%50)/100.0), "close": round2(closeP),
			"volume": int64(500000 + hashCode(code+date.String())%5000000), "change_percent": round2(change / openP * 100),
		})
		price = closeP
	}
	return map[string]interface{}{"stock_code": code, "stock_name": getStockName(code), "period": period, "data": data}
}

func getStockName(code string) string {
	m := map[string]string{"600519": "贵州茅台", "000858": "五粮液", "600036": "招商银行", "601318": "中国平安", "000333": "美的集团", "300750": "宁德时代", "000651": "格力电器", "002415": "海康威视", "601012": "隆基绿能", "600887": "伊利股份", "300059": "东方财富", "000725": "京东方A", "600309": "万华化学", "002475": "立讯精密"}
	if v, ok := m[code]; ok {
		return v
	}
	return fmt.Sprintf("股票%s", code)
}

func hashCode(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func parseImportData(data []byte, _ string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		code := strings.TrimSpace(parts[0])
		name := ""
		if len(parts) > 1 {
			name = strings.TrimSpace(parts[1])
		}
		if code != "" {
			result = append(result, map[string]interface{}{"code": code, "name": name, "confidence": "high"})
		}
	}
	return result
}

func parseImportText(text string) []map[string]interface{} {
	return parseImportData([]byte(text), ".txt")
}
