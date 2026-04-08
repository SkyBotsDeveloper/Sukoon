//go:build integration

package redis_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"sukoon/bot-core/internal/domain"
	redisstate "sukoon/bot-core/internal/state/redis"
)

func TestStoreTracksFloodAndLeaseState(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}

	password := os.Getenv("TEST_REDIS_PASSWORD")
	db := 0
	if raw := os.Getenv("TEST_REDIS_DB"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("invalid TEST_REDIS_DB: %v", err)
		}
		db = parsed
	}

	store := redisstate.New(addr, password, db)
	defer store.Close()

	ctx := context.Background()
	bot := domain.BotInstance{ID: "bot_primary", Slug: "primary", WebhookKey: "hook-test"}
	if err := store.CacheBot(ctx, bot, time.Minute); err != nil {
		t.Fatalf("CacheBot() error = %v", err)
	}
	cached, ok, err := store.GetCachedBot(ctx, bot.WebhookKey)
	if err != nil {
		t.Fatalf("GetCachedBot() error = %v", err)
	}
	if !ok || cached.ID != bot.ID {
		t.Fatalf("GetCachedBot() = (%+v, %v), want cached bot", cached, ok)
	}

	result, err := store.TrackFlood(ctx, bot.ID, -100123, 2002, 1, time.Minute)
	if err != nil {
		t.Fatalf("TrackFlood(first) error = %v", err)
	}
	if result.TimedCount != 1 || result.ConsecutiveCount != 1 {
		t.Fatalf("TrackFlood(first) = %+v, want timed=1 consecutive=1", result)
	}
	result, err = store.TrackFlood(ctx, bot.ID, -100123, 2002, 2, time.Minute)
	if err != nil {
		t.Fatalf("TrackFlood(second) error = %v", err)
	}
	if result.TimedCount != 2 || result.ConsecutiveCount != 2 {
		t.Fatalf("TrackFlood(second) = %+v, want timed=2 consecutive=2", result)
	}
	if err := store.ClearFlood(ctx, bot.ID, -100123, 2002); err != nil {
		t.Fatalf("ClearFlood() error = %v", err)
	}

	joins, err := store.TrackJoinBurst(ctx, bot.ID, -100123, 3003, time.Minute)
	if err != nil {
		t.Fatalf("TrackJoinBurst(first) error = %v", err)
	}
	if joins != 1 {
		t.Fatalf("TrackJoinBurst(first) count = %d, want 1", joins)
	}
	joins, err = store.TrackJoinBurst(ctx, bot.ID, -100123, 3004, time.Minute)
	if err != nil {
		t.Fatalf("TrackJoinBurst(second) error = %v", err)
	}
	if joins != 2 {
		t.Fatalf("TrackJoinBurst(second) count = %d, want 2", joins)
	}

	acquired, err := store.AcquireLease(ctx, "integration:lease", time.Minute)
	if err != nil {
		t.Fatalf("AcquireLease(first) error = %v", err)
	}
	if !acquired {
		t.Fatal("expected first lease acquisition to succeed")
	}
	acquired, err = store.AcquireLease(ctx, "integration:lease", time.Minute)
	if err != nil {
		t.Fatalf("AcquireLease(second) error = %v", err)
	}
	if acquired {
		t.Fatal("expected second lease acquisition to fail while lease is active")
	}
}
