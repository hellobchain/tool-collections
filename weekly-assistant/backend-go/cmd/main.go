package main

import (
	"fmt"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/database"
	"github.com/hellobchain/weekly-assistant/internal/handlers"
	"github.com/hellobchain/weekly-assistant/internal/middleware"
)

func main() {
	// 加载配置
	config.LoadConfig()

	// 初始化数据库
	database.InitDB()

	// 创建Gin引擎（不用 gin.Default()，避免重复日志）
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 请求入参日志
	r.Use(middleware.RequestLogger())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 认证路由（无需token）
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/refresh", handlers.RefreshToken)
	}

	// 业务路由（需要token）
	apiGroup := r.Group("/api")
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
	}

	// 启动服务
	log.Printf("Server starting on port %d", config.AppConfig.Port)
	r.Run(":" + fmt.Sprintf("%d", config.AppConfig.Port))
}
