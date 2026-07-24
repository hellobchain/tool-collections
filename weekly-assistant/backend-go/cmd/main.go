package main

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/handlers"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
	"github.com/hellobchain/weekly-assistant/internal/middleware/ginlog"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/wswlog/wlogging"
)

var slog = wlogging.MustGetLoggerWithoutName()

func main() {
	// 加载配置
	config.LoadConfig()

	// 初始化数据库
	database.InitDB()

	// 创建Gin引擎
	r := gin.New()
	r.HandleMethodNotAllowed = true
	gin.SetMode(config.AppConfig.GinMode)
	r.Use(gin.Recovery())
	r.Use(ginlog.Logger()) // 设置路由日志
	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 健康检查
	r.GET("/weekly-assistant/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 认证路由（无需token）
	authGroup := r.Group("/weekly-assistant/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/refresh", handlers.RefreshToken)
	}

	// 业务路由（需要token）
	apiGroup := r.Group("/weekly-assistant")
	apiGroup.Use(middleware.AuthMiddleware())
	{
		// Week
		apiGroup.GET("/week/status", handlers.GetWeekStatus)
		apiGroup.GET("/week/history", handlers.GetWeekHistory)
		apiGroup.GET("/week/history/weeks/export", handlers.ExportWeekHistory)
		apiGroup.GET("/week/summary", handlers.GenerateSummary)
		apiGroup.GET("/week/summaries", handlers.ListSummaries)
		apiGroup.GET("/week/history/summaries/export", handlers.ExportSummariesHistory)
		apiGroup.DELETE("/week/summary/:id", handlers.DeleteSummaryReport)
		apiGroup.POST("/week/carryover/confirm", handlers.ConfirmCarryover)
		apiGroup.POST("/week/generate", handlers.GenerateDraft)
		apiGroup.POST("/week/generate-stream", handlers.GenerateDraftStream)
		apiGroup.POST("/week/finalize", handlers.FinalizeWeek)
		apiGroup.DELETE("/week/report/:id", handlers.DeleteWeekReport)

		// Fragments
		apiGroup.GET("/fragments", handlers.ListFragments)
		apiGroup.POST("/fragments", handlers.AddFragment)
		apiGroup.DELETE("/fragments/:id", handlers.DeleteFragment)

		// GitLab
		apiGroup.POST("/gitlab/commits", handlers.GitLabCommits)
		apiGroup.GET("/git-projects", handlers.ListGitProjects)
		apiGroup.POST("/git-projects", handlers.CreateGitProject)
		apiGroup.PUT("/git-projects/:id", handlers.UpdateGitProject)
		apiGroup.DELETE("/git-projects/:id", handlers.DeleteGitProject)

		// User
		apiGroup.POST("/user/change-password", handlers.ChangePassword)

		// Prompts
		apiGroup.GET("/prompts", handlers.GetPromptTemplates)
		apiGroup.GET("/prompts/:id", handlers.GetPromptTemplate)
		apiGroup.POST("/prompts", handlers.CreatePromptTemplate)
		apiGroup.PUT("/prompts/:id", handlers.UpdatePromptTemplate)
		apiGroup.DELETE("/prompts/:id", handlers.DeletePromptTemplate)

		// Skills
		apiGroup.GET("/skills", handlers.ListSkills)
		apiGroup.POST("/skills", handlers.CreateSkill)
		apiGroup.PUT("/skills/:id", handlers.UpdateSkill)
		apiGroup.DELETE("/skills/:id", handlers.DeleteSkill)

		// Contract Review
		apiGroup.POST("/contract/v1/upload", handlers.UploadContract)
		apiGroup.DELETE("/contract/v1/files/:id", handlers.DeleteContractFile)
		apiGroup.GET("/contract/v1/files/:id/text", handlers.GetContractText)
		apiGroup.POST("/contract/v1/review", handlers.StartReview)
		apiGroup.GET("/contract/v1/review/:taskId/progress", handlers.GetReviewProgress)
		apiGroup.GET("/contract/v1/report/:reportId", handlers.GetReviewReport)
		apiGroup.PUT("/contract/v1/report/:reportId/items/:itemId", handlers.UpdateReviewItem)
		apiGroup.GET("/contract/v1/history", handlers.GetHistory)
		apiGroup.DELETE("/contract/v1/history/:reportId", handlers.DeleteHistory)
		apiGroup.GET("/contract/v1/report/:reportId/export", handlers.ExportReport)

		// Contract Extract
		apiGroup.POST("/contract/v1/extract/start", handlers.StartExtract)
		apiGroup.GET("/contract/v1/extract/:taskId/progress", handlers.GetExtractProgress)
		apiGroup.GET("/contract/v1/extract/:taskId/result", handlers.GetExtractResult)
		apiGroup.PUT("/contract/v1/extract/result/:resultId/cell", handlers.UpdateExtractCell)
		apiGroup.GET("/contract/v1/extract/history", handlers.GetExtractHistory)
		apiGroup.DELETE("/contract/v1/extract/:taskId", handlers.DeleteExtractTask)
		apiGroup.GET("/contract/v1/extract/:taskId/export", handlers.ExportExtractResult)

		// Contract Draft (static routes before :param routes)
		apiGroup.GET("/contract/v1/draft/history", handlers.GetDraftHistory)
		apiGroup.GET("/contract/v1/draft/history/:draftId", handlers.GetDraftDetail)
		apiGroup.DELETE("/contract/v1/draft/history/:draftId", handlers.DeleteDraft)
		apiGroup.POST("/contract/v1/draft/generate", handlers.StartDraftGenerate)
		apiGroup.GET("/contract/v1/draft/:taskId/progress", handlers.GetDraftProgress)
		apiGroup.GET("/contract/v1/draft/:taskId/result", handlers.GetDraftResult)
		apiGroup.GET("/contract/v1/draft/:taskId/download", handlers.DownloadDraft)

		// Markitdown - convert files to Markdown
		apiGroup.POST("/markitdown/v1/convert", handlers.MarkitdownConvert)

		// Doc Clean - remove comments and accept track changes
		apiGroup.POST("/doc-clean/v1/clean", handlers.DocClean)

		// JSON Compare
		apiGroup.POST("/json-compare/v1/compare", handlers.JsonCompare)

		// Document Convert
		apiGroup.POST("/doc-convert/v1/convert", handlers.DocConvert)

		// Document Type Detection
		apiGroup.POST("/doc-type/v1/detect", handlers.DetectDocType)

		// JSON Tools
		apiGroup.POST("/json-tool/v1/lang-convert", handlers.JsonLangConvert)

		// JSONL Stream
		apiGroup.POST("/jsonl-read/v1/data/stream", handlers.StreamJSONL)

		// JSONL Convert to Docx
		apiGroup.POST("/jsonl-read/v1/data/convert-to-docx", handlers.ConvertJSONLToDocx)

		// ===================== Stock Analysis =====================
		// Analysis
		apiGroup.POST("/stock/v1/analysis/analyze", handlers.TriggerStockAnalysis)
		apiGroup.GET("/stock/v1/analysis/status/:taskId", handlers.GetStockAnalysisStatus)
		apiGroup.GET("/stock/v1/analysis/tasks", handlers.ListStockAnalysisTasks)
		apiGroup.POST("/stock/v1/analysis/market-review", handlers.TriggerMarketReview)
		apiGroup.GET("/stock/v1/analysis/tasks/:taskId/flow", handlers.GetStockAnalysisTaskFlow)

		// Stock Data
		apiGroup.GET("/stock/v1/stocks/quote/:code", handlers.GetStockQuote)
		apiGroup.GET("/stock/v1/stocks/history/:code", handlers.GetStockHistory)
		apiGroup.GET("/stock/v1/stocks/watchlist", handlers.GetWatchlist)
		apiGroup.POST("/stock/v1/stocks/watchlist/add", handlers.AddToWatchlist)
		apiGroup.POST("/stock/v1/stocks/watchlist/remove", handlers.RemoveFromWatchlist)
		apiGroup.POST("/stock/v1/stocks/import", handlers.ImportStockCodes)

		// History
		apiGroup.GET("/stock/v1/history", handlers.ListStockAnalysisHistory)
		apiGroup.GET("/stock/v1/history/stocks", handlers.GetStockBar)
		apiGroup.GET("/stock/v1/history/:id", handlers.GetStockHistoryDetail)
		apiGroup.DELETE("/stock/v1/history", handlers.DeleteStockAnalysisHistory)
		apiGroup.DELETE("/stock/v1/history/by-code/:code", handlers.DeleteStockAnalysisHistoryByCode)
		apiGroup.GET("/stock/v1/history/:id/markdown", handlers.GetStockHistoryMarkdown)

		// Agent Chat
		apiGroup.POST("/stock/v1/agent/chat", handlers.AgentChat)
		apiGroup.GET("/stock/v1/agent/skills", handlers.ListAgentSkills)
		apiGroup.GET("/stock/v1/agent/chat/sessions", handlers.ListChatSessions)
		apiGroup.GET("/stock/v1/agent/chat/sessions/:sessionId", handlers.GetChatSessionMessages)
		apiGroup.DELETE("/stock/v1/agent/chat/sessions/:sessionId", handlers.DeleteChatSession)

		// Portfolio
		apiGroup.POST("/stock/v1/portfolio/accounts", handlers.CreatePortfolioAccount)
		apiGroup.GET("/stock/v1/portfolio/accounts", handlers.ListPortfolioAccounts)
		apiGroup.POST("/stock/v1/portfolio/trades", handlers.RecordPortfolioTrade)
		apiGroup.GET("/stock/v1/portfolio/snapshot", handlers.GetPortfolioSnapshot)

		// Backtest
		apiGroup.POST("/stock/v1/backtest/run", handlers.RunBacktest)
		apiGroup.GET("/stock/v1/backtest/results", handlers.GetBacktestResults)
		apiGroup.GET("/stock/v1/backtest/performance", handlers.GetBacktestPerformance)

		// System Config
		apiGroup.GET("/stock/v1/system/config", handlers.GetSystemStockConfig)
		apiGroup.PUT("/stock/v1/system/config", handlers.UpdateSystemStockConfig)
	}

	// 启动周报自动生成定时任务
	services.StartAutoWeeklyScheduler()

	// 启动服务
	slog.Infof("Server starting on port %d", config.AppConfig.Port)
	r.Run(":" + fmt.Sprintf("%d", config.AppConfig.Port))
}
