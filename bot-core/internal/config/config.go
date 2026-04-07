package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                 string
	AppMode                string
	AppAddr                string
	PublicWebhookBaseURL   string
	DatabaseURL            string
	RedisAddr              string
	RedisPassword          string
	RedisDB                int
	WorkerConcurrency      int
	WorkerPollInterval     time.Duration
	TelegramBaseURL        string
	TelegramRequestTimeout time.Duration
	TelegramMaxRetries     int
	TelegramInitialBackoff time.Duration
	BotOwnerUserIDs        []int64
	PrimaryBot             PrimaryBotConfig
}

type PrimaryBotConfig struct {
	Slug          string
	DisplayName   string
	Token         string
	WebhookKey    string
	WebhookSecret string
	Username      string
}

func LoadFromEnv() (Config, error) {
	appAddr := getEnv("APP_ADDR", "")
	if appAddr == "" {
		if port := getEnv("PORT", ""); port != "" {
			appAddr = ":" + port
		} else {
			appAddr = ":8080"
		}
	}

	redisAddr := getEnv("REDIS_ADDR", "127.0.0.1:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	if rawRedisURL := getEnv("REDIS_URL", ""); rawRedisURL != "" {
		if parsed, err := url.Parse(rawRedisURL); err == nil {
			if parsed.Host != "" {
				redisAddr = parsed.Host
			}
			if parsed.User != nil {
				if password, ok := parsed.User.Password(); ok {
					redisPassword = password
				}
			}
			if parsed.Path != "" && parsed.Path != "/" {
				if db, err := strconv.Atoi(strings.TrimPrefix(parsed.Path, "/")); err == nil {
					redisDB = db
				}
			}
		}
	}

	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppMode:                getEnv("APP_MODE", "all"),
		AppAddr:                appAddr,
		PublicWebhookBaseURL:   strings.TrimRight(getEnv("PUBLIC_WEBHOOK_BASE_URL", ""), "/"),
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		RedisAddr:              redisAddr,
		RedisPassword:          redisPassword,
		RedisDB:                redisDB,
		WorkerConcurrency:      getEnvInt("WORKER_CONCURRENCY", 4),
		WorkerPollInterval:     getEnvDuration("WORKER_POLL_INTERVAL", 100*time.Millisecond),
		TelegramBaseURL:        getEnv("TELEGRAM_BASE_URL", "https://api.telegram.org"),
		TelegramRequestTimeout: getEnvDuration("TELEGRAM_REQUEST_TIMEOUT", 5*time.Second),
		TelegramMaxRetries:     getEnvInt("TELEGRAM_MAX_RETRIES", 3),
		TelegramInitialBackoff: getEnvDuration("TELEGRAM_INITIAL_BACKOFF", 750*time.Millisecond),
		BotOwnerUserIDs:        getEnvInt64List("BOT_OWNER_USER_IDS"),
		PrimaryBot: PrimaryBotConfig{
			Slug:          getEnv("PRIMARY_BOT_SLUG", "primary"),
			DisplayName:   getEnv("PRIMARY_BOT_DISPLAY_NAME", "Sukoon"),
			Token:         getEnv("PRIMARY_BOT_TOKEN", ""),
			WebhookKey:    getEnv("PRIMARY_BOT_WEBHOOK_KEY", ""),
			WebhookSecret: getEnv("PRIMARY_BOT_WEBHOOK_SECRET", ""),
			Username:      getEnv("PRIMARY_BOT_USERNAME", ""),
		},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	switch cfg.AppMode {
	case "all", "web", "worker":
	default:
		return Config{}, fmt.Errorf("APP_MODE must be one of all, web, worker")
	}
	if cfg.PrimaryBot.Token != "" {
		if cfg.PrimaryBot.WebhookKey == "" {
			return Config{}, fmt.Errorf("PRIMARY_BOT_WEBHOOK_KEY is required when PRIMARY_BOT_TOKEN is set")
		}
		if cfg.PrimaryBot.WebhookSecret == "" {
			return Config{}, fmt.Errorf("PRIMARY_BOT_WEBHOOK_SECRET is required when PRIMARY_BOT_TOKEN is set")
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt64List(key string) []int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			result = append(result, parsed)
		}
	}
	return result
}
