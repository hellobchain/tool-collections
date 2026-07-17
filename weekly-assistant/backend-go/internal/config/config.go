package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBLogLevel int // 1 silent 2 error 3 warn 4 info

	JWTSecret     string
	EncryptionKey string

	DeepSeekAPIKey    string
	DeepSeekBaseURL   string
	DeepSeekModel     string
	DeepSeekMaxTokens int

	Port int

	InitUser InitUser
}

// 初始化用户
type InitUser struct {
	Email    string
	Password string
	Username string
}

var AppConfig *Config

func LoadConfig() {
	envPaths := []string{"configs/.env", ".env"}
	loaded := false
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			loaded = true
			break
		}
	}
	if !loaded {
		log.Println("Warning: no .env file found, using environment variables")
	}

	port, _ := strconv.Atoi(getEnv("PORT", "8000"))
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))

	maxTokens, _ := strconv.Atoi(getEnv("DEEPSEEK_MAX_TOKENS", "4096"))

	dbLogLevel, _ := strconv.Atoi(getEnv("DB_LOG_LEVEL", "1"))

	AppConfig = &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     dbPort,
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "weekly_assistant"),
		DBLogLevel: dbLogLevel,

		JWTSecret:     getEnv("JWT_SECRET", ""),
		EncryptionKey: getEnv("ENCRYPTION_KEY", ""),

		DeepSeekAPIKey:    getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekBaseURL:   getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
		DeepSeekModel:     getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		DeepSeekMaxTokens: maxTokens,

		Port: port,
	}

	if AppConfig.JWTSecret == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err == nil {
			AppConfig.JWTSecret = hex.EncodeToString(key)
		}
		log.Printf("[WARNING] JWT_SECRET 未设置，已自动生成随机密钥。注意：服务重启后所有 token 将失效")
	}

	if AppConfig.EncryptionKey == "" {
		if len(AppConfig.JWTSecret) >= 32 {
			AppConfig.EncryptionKey = AppConfig.JWTSecret[:32]
		} else {
			key := make([]byte, 32)
			rand.Read(key)
			AppConfig.EncryptionKey = hex.EncodeToString(key)
		}
	}

	initEmail := getEnv("INIT_USER_EMAIL", "")
	initPwd := getEnv("INIT_USER_PASSWORD", "")
	initUser := getEnv("INIT_USER_USERNAME", "")

	if initEmail == "" || initPwd == "" || initUser == "" {
		log.Println("[WARNING] 未设置 INIT_USER_EMAIL/PASSWORD/USERNAME，不会创建初始化用户")
	} else {
		AppConfig.InitUser = InitUser{
			Email:    initEmail,
			Password: initPwd,
			Username: initUser,
		}
	}

	// AppConfig 输出（脱敏敏感字段）
	safe := *AppConfig
	if safe.DBPassword != "" {
		safe.DBPassword = "***"
	}
	if safe.JWTSecret != "" {
		safe.JWTSecret = "***"
	}
	if safe.EncryptionKey != "" {
		safe.EncryptionKey = "***"
	}
	if safe.DeepSeekAPIKey != "" {
		safe.DeepSeekAPIKey = "***"
	}
	if safe.InitUser.Password != "" {
		safe.InitUser.Password = "***"
	}
	ret, _ := json.MarshalIndent(safe, "", "\t")
	log.Println("AppConfig:", string(ret))
	log.Println("Config loaded")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
