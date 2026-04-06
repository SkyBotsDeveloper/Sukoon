package config

import "testing"

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

func TestLoadFromEnvRejectsInvalidAppMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/sukoon?sslmode=disable")
	t.Setenv("APP_MODE", "invalid")

	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid APP_MODE")
	}
}
