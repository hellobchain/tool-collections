package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hellobchain/weekly-assistant/internal/auth"
	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/wswlog/wlogging"
)

var slog = wlogging.MustGetLoggerWithoutName()
var DB *gorm.DB

func InitDB() {
	cfg := config.AppConfig
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: NewSqlLogger(slog, logger.LogLevel(cfg.DBLogLevel)),
	})
	if err != nil {
		slog.Fatal("Failed to connect to database:", err)
	}

	// 自动迁移
	err = DB.AutoMigrate(
		&models.User{},
		&models.WeeklyReport{},
		&models.Fragment{},
		&models.Goal{},
		&models.PromptTemplate{},
		&models.Skill{},
		&models.GitProject{},
		&models.Summary{},
		&models.ContractFile{},
		&models.ContractReview{},
		&models.ContractReviewItem{},
		&models.ContractDraft{},
		&models.ContractExtractTask{},
		&models.ContractExtractResult{},
		&models.StockAnalysisTask{},
		&models.StockAnalysisReport{},
		&models.StockWatchlist{},
		&models.PortfolioAccount{},
		&models.PortfolioTrade{},
		&models.PortfolioCashLedger{},
		&models.PortfolioCorporateAction{},
		&models.AgentChatSession{},
		&models.AgentChatMessage{},
		&models.SystemConfig{},
		&models.BacktestResult{},
	)
	if err != nil {
		slog.Fatal("Failed to migrate database:", err)
	}

	slog.Infof("Database connected and migrated successfully")
	InitUser()
	InitPromptTemplates()
}

func GetDB() *gorm.DB {
	return DB
}

// 初始化用户信息
func InitUser() {
	if config.AppConfig.InitUser.Username == "" {
		slog.Warn("InitUser config not set, skipping user initialization")
		return
	}
	if !DB.Migrator().HasTable(&models.User{}) {
		slog.Fatal("Table 'users' does not exist. Initializing user...")
	}
	cfg := config.AppConfig
	// 判断用户是否存在
	var count int64
	err := DB.Where("username = ?", cfg.InitUser.Username).Find(&models.User{}).Count(&count).Error
	if err != nil {
		slog.Fatal("Failed to check user existence:", err)
	}
	if count > 0 {
		slog.Info("User already exists. Skipping initialization.")
		return
	}
	slog.Info("Initializing user...")
	// 加密密码
	hashedPwd, err := auth.HashPassword(cfg.InitUser.Password)
	if err != nil {
		slog.Fatal("Failed to hash password:", err)
	}
	user := models.User{
		Email:        cfg.InitUser.Email,
		PasswordHash: hashedPwd,
		Username:     cfg.InitUser.Username,
	}
	if err := DB.Create(&user).Error; err != nil {
		slog.Fatal("Failed to create user:", err)
	}
	slog.Info("User initialized successfully")
}

func InitPromptTemplates() {
	if !DB.Migrator().HasTable(&models.PromptTemplate{}) {
		slog.Fatal("Table 'prompt_templates' does not exist. Initializing prompt templates...")
	}

	var count int64
	DB.Model(&models.PromptTemplate{}).Where("user_id IS NULL").Count(&count)
	if count > 0 {
		slog.Info("System prompt templates already exist. Skipping initialization.")
		return
	}

	templates := []models.PromptTemplate{
		{
			Name:               "标准专业风",
			Category:           "default",
			PromptType:         "weekly",
			SystemPrompt:       "你是一个专业的周报撰写助手，擅长把零散信息组织成结构化、有重点的工作汇报。",
			UserPromptTemplate: `请根据以下碎片和继承事项，生成一份周报。输出必须包含三个平行板块，每个板块用"### "开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（从"攻克、突破、解决"等角度组织本周的关键产出，每条用"- "开头）\n\n### 协作\n（从"推动、对齐、拉通"等角度组织常规事项与协作工作，每条用"- "开头）\n\n### 稳健\n（从"保障、规范、优化"等角度组织风险控制与稳定性工作，每条用"- "开头）\n\n要求：\n1. 每个板块至少2-3条\n2. 语言专业、客观，每条包含动作+对象+结果\n3. 禁止出现"可能""大概""似乎"等弱化词\n4. 如果继承项不为空，在"攻坚"板块顶部插入一条"承接上周遗留：{内容}"`,
			Description:        "通用场景，适合大多数情况",
			SortOrder:          1,
		},
		{
			Name:               "数据驱动风",
			Category:           "default",
			PromptType:         "weekly",
			SystemPrompt:       "你是一个数据驱动的周报撰写专家，擅长用数据量化工作成果。",
			UserPromptTemplate: `请根据以下碎片和继承事项，生成一份数据驱动的周报。输出必须包含三个平行板块，每个板块用"### "开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（从"攻克、突破、解决"等角度，用数据量化本周关键产出，每条用"- "开头）\n\n### 协作\n（从"推动、对齐、拉通"等角度，用数据量化协作成果，每条用"- "开头）\n\n### 稳健\n（从"保障、规范、优化"等角度，用数据量化稳定性工作，每条用"- "开头）\n\n要求：\n1. 每项产出必须包含量化指标（如：响应时间降低30%、处理工单50个）\n2. 如果数据缺失，用"[待补充数据]"占位\n3. 每个板块至少2-3条`,
			Description:        "适合数据型老板，强调KPI和指标",
			SortOrder:          2,
		},
		{
			Name:               "故事叙述风",
			Category:           "default",
			PromptType:         "weekly",
			SystemPrompt:       "你是一个善于讲故事的技术专家，能用项目背景和业务价值串联工作成果。",
			UserPromptTemplate: `请根据以下碎片和继承事项，生成一份有故事感的周报。输出必须包含三个平行板块，每个板块用"### "开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（从"攻克、突破、解决"角度讲述本周关键产出的故事，每条用"- "开头）\n\n### 协作\n（从"推动、对齐、拉通"角度讲述协作工作的故事，每条用"- "开头）\n\n### 稳健\n（从"保障、规范、优化"角度讲述稳定性工作的故事，每条用"- "开头）\n\n要求：\n1. 每项工作都要回答"为什么做这件事"（背景/痛点）\n2. 描述"怎么做的"（过程/挑战）\n3. 说明"结果如何"（影响/价值）\n4. 每个板块至少2-3条`,
			Description:        "适合产品/业务型老板，强调价值",
			SortOrder:          3,
		},
		{
			Name:               "极简风",
			Category:           "default",
			PromptType:         "weekly",
			SystemPrompt:       "你是一个高效的周报助手，只输出核心信息，不废话。",
			UserPromptTemplate: `请根据以下碎片和继承事项，生成一份极简周报。输出必须包含三个平行板块，每个板块用"### "开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（关键产出，每条用"- "开头）\n\n### 协作\n（协作事项，每条用"- "开头）\n\n### 稳健\n（风险与稳定性，每条用"- "开头）\n\n要求：\n1. 每个板块只保留最核心的1-2条\n2. 每条不超过20个字\n3. 去掉所有修饰词和客套话`,
			Description:        "适合非常忙碌的老板，快速阅读",
			SortOrder:          4,
		},
		{
			Name:               "技术深度风",
			Category:           "default",
			PromptType:         "weekly",
			SystemPrompt:       "你是一个技术专家，擅长从技术视角剖析工作，强调技术难点和解决方案的优雅性。",
			UserPromptTemplate: `请根据以下碎片和继承事项，生成一份技术深度的周报。输出必须包含三个平行板块，每个板块用"### "开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（从技术突破角度描述关键产出，每条用"- "开头）\n\n### 协作\n（从技术协同角度描述协作工作，每条用"- "开头）\n\n### 稳健\n（从系统稳定性、技术规范角度描述保障工作，每条用"- "开头）\n\n要求：\n1. 重点描述技术难点和挑战\n2. 说明采用的解决方案和技术选型理由\n3. 分享踩过的坑和学到的经验\n4. 适当使用技术术语，体现专业性\n5. 每个板块至少2-3条`,
			Description:        "适合技术型老板，强调技术深度",
			SortOrder:          5,
		},
		{
			Name:               "向上汇报风",
			Category:           "default",
			PromptType:         "weekly",
			SystemPrompt:       "你是一个资深管理者，擅长从战略高度提炼工作价值，向上汇报时强调业务影响和资源需求。",
			UserPromptTemplate: `请根据以下碎片和继承事项，生成一份向上汇报风格的周报。输出必须包含三个平行板块，每个板块用"### "开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（从战略突破角度提炼关键产出，每条用"- "开头）\n\n### 协作\n（从跨部门协同角度提炼协作成果，每条用"- "开头）\n\n### 稳健\n（从风险控制角度提炼保障工作，每条用"- "开头）\n\n要求：\n1. 每项工作都要关联到季度OKR或部门目标\n2. 突出风险和需要领导决策的事项\n3. 用"战略意义""关键里程碑""资源缺口"等管理语言\n4. 格式要正式，适合转发给高层\n5. 每个板块至少2-3条`,
			Description:        "适合高层汇报，强调战略价值",
			SortOrder:          6,
		},
	}
	DB.CreateInBatches(templates, len(templates))
	slog.Info("System prompt templates initialized successfully")
}
