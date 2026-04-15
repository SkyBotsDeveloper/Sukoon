package config

import (
	"testing"
	"time"
)

func TestLoadFromEnvUsesPortAndRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/sukoon?sslmode=disable")
	t.Setenv("PORT", "9090")
	t.Setenv("APP_MODE", "worker")
	t.Setenv("REDIS_URL", "redis://:secret@cache:6380/4")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.AppAddr != ":9090" {
		t.Fatalf("expected APP_ADDR from PORT, got %q", cfg.AppAddr)
	}
	if cfg.AppMode != "worker" {
		t.Fatalf("expected APP_MODE worker, got %q", cfg.AppMode)
	}
	if cfg.RedisAddr != "cache:6380" {
		t.Fatalf("expected redis addr from REDIS_URL, got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "secret" {
		t.Fatalf("expected redis password from REDIS_URL, got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 4 {
		t.Fatalf("expected redis db from REDIS_URL, got %d", cfg.RedisDB)
	}
}

func TestLoadFromEnvUsesMigrationURLAndPoolOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@runtime-db:5432/sukoon?sslmode=require")
	t.Setenv("MIGRATE_DATABASE_URL", "postgres://user:pass@direct-db:5432/sukoon?sslmode=require")
	t.Setenv("DATABASE_MAX_CONNS", "20")
	t.Setenv("DATABASE_MIN_CONNS", "1")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "2h")
	t.Setenv("DATABASE_MAX_CONN_IDLE_TIME", "30m")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.EffectiveMigrateDatabaseURL() != "postgres://user:pass@direct-db:5432/sukoon?sslmode=require" {
		t.Fatalf("expected migrate URL override, got %q", cfg.EffectiveMigrateDatabaseURL())
	}
	if cfg.DatabaseMaxConns != 20 || cfg.DatabaseMinConns != 1 {
		t.Fatalf("unexpected pool sizes: max=%d min=%d", cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	}
	if cfg.DatabaseMaxConnLifetime != 2*time.Hour {
		t.Fatalf("unexpected max conn lifetime: %s", cfg.DatabaseMaxConnLifetime)
	}
	if cfg.DatabaseMaxConnIdleTime != 30*time.Minute {
		t.Fatalf("unexpected max conn idle time: %s", cfg.DatabaseMaxConnIdleTime)
	}
}

func TestLoadFromEnvRejectsInvalidAppMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/sukoon?sslmode=disable")
	t.Setenv("APP_MODE", "invalid")

	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid APP_MODE")
	}
}

func TestLoadFromEnvRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "not-a-postgres-url")

	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid DATABASE_URL")
	}
}

func TestLoadFromEnvRejectsInvalidRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/sukoon?sslmode=disable")
	t.Setenv("REDIS_URL", "http://cache:6379/0")

	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid REDIS_URL")
	}
}

func TestLoadFromEnvRejectsInvalidRedisAddr(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/sukoon?sslmode=disable")
	t.Setenv("REDIS_ADDR", "cache")

	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid REDIS_ADDR")
	}
}

func TestLoadFromEnvUsesFastDefaultWorkerPollInterval(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/sukoon?sslmode=disable")
	t.Setenv("APP_MODE", "worker")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.WorkerPollInterval != 100*time.Millisecond {
		t.Fatalf("expected default worker poll interval 100ms, got %s", cfg.WorkerPollInterval)
	}
	if cfg.EffectiveMigrateDatabaseURL() != cfg.DatabaseURL {
		t.Fatalf("expected migrate URL to fall back to DATABASE_URL, got %q", cfg.EffectiveMigrateDatabaseURL())
	}
}
