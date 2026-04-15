package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	AppEnv                  string
	AppMode                 string
	AppAddr                 string
	PublicWebhookBaseURL    string
	DatabaseURL             string
	MigrateDatabaseURL      string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseMaxConnIdleTime time.Duration
	RedisAddr               string
	RedisPassword           string
	RedisDB                 int
	WorkerConcurrency       int
	WorkerPollInterval      time.Duration
	TelegramBaseURL         string
	TelegramRequestTimeout  time.Duration
	TelegramMaxRetries      int
	TelegramInitialBackoff  time.Duration
	BotOwnerUserIDs         []int64
	PrimaryBot              PrimaryBotConfig
}

type PrimaryBotConfig struct {
	Slug          string
	DisplayName   string
	Token         string
	WebhookKey    string
	WebhookSecret string
	Username      string
}

func (c Config) EffectiveMigrateDatabaseURL() string {
	if c.MigrateDatabaseURL != "" {
		return c.MigrateDatabaseURL
	}
	return c.DatabaseURL
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

	redisDB, err := getEnvIntStrict("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	workerConcurrency, err := getEnvIntStrict("WORKER_CONCURRENCY", 4)
	if err != nil {
		return Config{}, err
	}
	workerPollInterval, err := getEnvDurationStrict("WORKER_POLL_INTERVAL", 100*time.Millisecond)
	if err != nil {
		return Config{}, err
	}
	telegramRequestTimeout, err := getEnvDurationStrict("TELEGRAM_REQUEST_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	telegramInitialBackoff, err := getEnvDurationStrict("TELEGRAM_INITIAL_BACKOFF", 750*time.Millisecond)
	if err != nil {
		return Config{}, err
	}
	telegramMaxRetries, err := getEnvIntStrict("TELEGRAM_MAX_RETRIES", 3)
	if err != nil {
		return Config{}, err
	}
	databaseMaxConns, err := getEnvInt32Strict("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	databaseMinConns, err := getEnvInt32Strict("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return Config{}, err
	}
	databaseMaxConnLifetime, err := getEnvDurationStrict("DATABASE_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return Config{}, err
	}
	databaseMaxConnIdleTime, err := getEnvDurationStrict("DATABASE_MAX_CONN_IDLE_TIME", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	redisAddr, redisPassword, redisDB, err := resolveRedisConfig(
		getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		getEnv("REDIS_PASSWORD", ""),
		redisDB,
		getEnv("REDIS_URL", ""),
	)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		AppMode:                 getEnv("APP_MODE", "all"),
		AppAddr:                 appAddr,
		PublicWebhookBaseURL:    strings.TrimRight(getEnv("PUBLIC_WEBHOOK_BASE_URL", ""), "/"),
		DatabaseURL:             getEnv("DATABASE_URL", ""),
		MigrateDatabaseURL:      getEnv("MIGRATE_DATABASE_URL", ""),
		DatabaseMaxConns:        databaseMaxConns,
		DatabaseMinConns:        databaseMinConns,
		DatabaseMaxConnLifetime: databaseMaxConnLifetime,
		DatabaseMaxConnIdleTime: databaseMaxConnIdleTime,
		RedisAddr:               redisAddr,
		RedisPassword:           redisPassword,
		RedisDB:                 redisDB,
		WorkerConcurrency:       workerConcurrency,
		WorkerPollInterval:      workerPollInterval,
		TelegramBaseURL:         getEnv("TELEGRAM_BASE_URL", "https://api.telegram.org"),
		TelegramRequestTimeout:  telegramRequestTimeout,
		TelegramMaxRetries:      telegramMaxRetries,
		TelegramInitialBackoff:  telegramInitialBackoff,
		BotOwnerUserIDs:         getEnvInt64List("BOT_OWNER_USER_IDS"),
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
	if _, err := pgxpool.ParseConfig(cfg.DatabaseURL); err != nil {
		return Config{}, fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	if cfg.MigrateDatabaseURL != "" {
		if _, err := pgxpool.ParseConfig(cfg.MigrateDatabaseURL); err != nil {
			return Config{}, fmt.Errorf("MIGRATE_DATABASE_URL is invalid: %w", err)
		}
	}
	switch cfg.AppMode {
	case "all", "web", "worker":
	default:
		return Config{}, fmt.Errorf("APP_MODE must be one of all, web, worker")
	}
	if cfg.WorkerConcurrency <= 0 {
		return Config{}, fmt.Errorf("WORKER_CONCURRENCY must be greater than 0")
	}
	if cfg.WorkerPollInterval <= 0 {
		return Config{}, fmt.Errorf("WORKER_POLL_INTERVAL must be greater than 0")
	}
	if cfg.DatabaseMaxConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than 0")
	}
	if cfg.DatabaseMinConns < 0 {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must be 0 or greater")
	}
	if cfg.DatabaseMaxConns < cfg.DatabaseMinConns {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than or equal to DATABASE_MIN_CONNS")
	}
	if cfg.DatabaseMaxConnLifetime <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONN_LIFETIME must be greater than 0")
	}
	if cfg.DatabaseMaxConnIdleTime <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONN_IDLE_TIME must be greater than 0")
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

func resolveRedisConfig(defaultAddr, defaultPassword string, defaultDB int, rawRedisURL string) (string, string, int, error) {
	redisAddr := defaultAddr
	redisPassword := defaultPassword
	if rawRedisURL == "" {
		if _, _, err := net.SplitHostPort(redisAddr); err != nil {
			return "", "", 0, fmt.Errorf("REDIS_ADDR must be in host:port form: %w", err)
		}
		return redisAddr, redisPassword, defaultDB, nil
	}

	parsed, err := url.Parse(rawRedisURL)
	if err != nil {
		return "", "", 0, fmt.Errorf("REDIS_URL is invalid: %w", err)
	}
	if parsed.Scheme != "redis" && parsed.Scheme != "rediss" {
		return "", "", 0, fmt.Errorf("REDIS_URL must use redis:// or rediss://")
	}
	if parsed.Host == "" {
		return "", "", 0, fmt.Errorf("REDIS_URL must include a host")
	}
	redisAddr = parsed.Host
	if parsed.User != nil {
		if password, ok := parsed.User.Password(); ok {
			redisPassword = password
		}
	}
	if parsed.Path != "" && parsed.Path != "/" {
		db, err := strconv.Atoi(strings.TrimPrefix(parsed.Path, "/"))
		if err != nil {
			return "", "", 0, fmt.Errorf("REDIS_URL database index is invalid: %w", err)
		}
		defaultDB = db
	}
	if _, _, err := net.SplitHostPort(redisAddr); err != nil {
		return "", "", 0, fmt.Errorf("REDIS_URL host must be in host:port form: %w", err)
	}
	return redisAddr, redisPassword, defaultDB, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvIntStrict(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func getEnvInt32Strict(key string, fallback int32) (int32, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a 32-bit integer: %w", key, err)
	}
	return int32(parsed), nil
}

func getEnvDurationStrict(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return parsed, nil
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
