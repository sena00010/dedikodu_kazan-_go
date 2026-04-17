package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv                  string
	HTTPAddr                string
	AppName                 string
	JWTSecret               string
	MySQLDSN                string
	RedisAddr               string
	RedisPassword           string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
	RevenueCatWebhookSecret string
	AIProvider              string
	OpenAIKey               string
	OpenAIModel             string
	AnthropicKey            string
	AnthropicModel          string
	GeminiKey               string
	GeminiModel             string
	AIRoomCost              int
	AIRoomDuration          time.Duration
}

func Load() Config {
	return Config{
		AppEnv:                  env("APP_ENV", "local"),
		HTTPAddr:                env("HTTP_ADDR", ":8080"),
		AppName:                 env("PUBLIC_APP_NAME", "Dedikodu Kazani"),
		JWTSecret:               env("JWT_SECRET", "local-dev-secret-change-me"),
		MySQLDSN:                env("MYSQL_DSN", "root:@tcp(127.0.0.1:3307)/dedikodu_kazani?parseTime=true&multiStatements=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"),
		RedisAddr:               env("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:           env("REDIS_PASSWORD", ""),
		FirebaseProjectID:       env("FIREBASE_PROJECT_ID", ""),
		FirebaseCredentialsFile: env("FIREBASE_CREDENTIALS_FILE", ""),
		RevenueCatWebhookSecret: env("REVENUECAT_WEBHOOK_SECRET", ""),
		AIProvider:              env("AI_PROVIDER", "openai"),
		OpenAIKey:               env("OPENAI_API_KEY", ""),
		OpenAIModel:             env("OPENAI_MODEL", "gpt-4o-mini"),
		AnthropicKey:            env("ANTHROPIC_API_KEY", ""),
		AnthropicModel:          env("ANTHROPIC_MODEL", "claude-3-5-haiku-latest"),
		GeminiKey:               env("GEMINI_API_KEY", ""),
		GeminiModel:             env("GEMINI_MODEL", "gemini-1.5-flash"),
		AIRoomCost:              envInt("AI_ROOM_COST", 50),
		AIRoomDuration:          time.Duration(envInt("AI_ROOM_MINUTES", 60)) * time.Minute,
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
