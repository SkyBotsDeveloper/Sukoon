//go:build integration

package postgres

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/telegram"
)

func TestStoreMigratesAndPersistsCanonicalContracts(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resetDatabase(t, ctx, databaseURL)

	store, err := New(ctx, databaseURL, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	bot, err := store.UpsertPrimaryBot(ctx, domain.BotInstance{
		ID:            "bot_primary",
		Slug:          "primary",
		DisplayName:   "Sukoon",
		TelegramToken: "token",
		WebhookKey:    "hook",
		WebhookSecret: "secret",
		Username:      "sukoonbot",
	}, []int64{1001})
	if err != nil {
		t.Fatalf("UpsertPrimaryBot() error = %v", err)
	}

	if err := store.EnsureUser(ctx, telegram.User{ID: 1001, Username: "owner", FirstName: "Owner"}); err != nil {
		t.Fatalf("EnsureUser(owner) error = %v", err)
	}
	if err := store.EnsureUser(ctx, telegram.User{ID: 2002, Username: "target", FirstName: "Target"}); err != nil {
		t.Fatalf("EnsureUser(target) error = %v", err)
	}
	if err := store.EnsureChat(ctx, bot.ID, telegram.Chat{ID: -100123, Type: "supergroup", Title: "Audit Room"}); err != nil {
		t.Fatalf("EnsureChat() error = %v", err)
	}

	if ok, err := store.EnqueueUpdate(ctx, bot.ID, 42, []byte(`{"update_id":42}`)); err != nil || !ok {
		t.Fatalf("first EnqueueUpdate() = (%v, %v), want (true, nil)", ok, err)
	}
	if ok, err := store.EnqueueUpdate(ctx, bot.ID, 42, []byte(`{"update_id":42}`)); err != nil || ok {
		t.Fatalf("duplicate EnqueueUpdate() = (%v, %v), want (false, nil)", ok, err)
	}

	updates, err := store.ClaimPendingUpdates(ctx, "integration-worker", 5)
	if err != nil {
		t.Fatalf("ClaimPendingUpdates() error = %v", err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 42 {
		t.Fatalf("ClaimPendingUpdates() = %+v, want one update 42", updates)
	}
	if err := store.MarkUpdateCompleted(ctx, updates[0].ID); err != nil {
		t.Fatalf("MarkUpdateCompleted() error = %v", err)
	}

	if err := store.SetApproval(ctx, bot.ID, -100123, 2002, 1001, true, "testing"); err != nil {
		t.Fatalf("SetApproval() error = %v", err)
	}
	approved, err := store.IsApproved(ctx, bot.ID, -100123, 2002)
	if err != nil {
		t.Fatalf("IsApproved() error = %v", err)
	}
	if !approved {
		t.Fatal("expected approval to persist")
	}
	approval, err := store.GetApproval(ctx, bot.ID, -100123, 2002)
	if err != nil {
		t.Fatalf("GetApproval() error = %v", err)
	}
	if approval.Reason != "testing" {
		t.Fatalf("expected approval reason to persist, got %+v", approval)
	}

	antiraidUntil := time.Now().Add(2 * time.Hour)
	if err := store.SetAntiRaidSettings(ctx, domain.AntiRaidSettings{
		BotID:                 bot.ID,
		ChatID:                -100123,
		EnabledUntil:          &antiraidUntil,
		RaidDurationSeconds:   6 * 60 * 60,
		ActionDurationSeconds: 60 * 60,
		AutoThreshold:         12,
	}); err != nil {
		t.Fatalf("SetAntiRaidSettings() error = %v", err)
	}
	bundle, err := store.LoadRuntimeBundle(ctx, bot.ID, -100123)
	if err != nil {
		t.Fatalf("LoadRuntimeBundle() error = %v", err)
	}
	if bundle.AntiRaid.EnabledUntil == nil || bundle.AntiRaid.AutoThreshold != 12 || bundle.AntiRaid.ActionDurationSeconds != 60*60 {
		t.Fatalf("expected antiraid settings to persist, got %+v", bundle.AntiRaid)
	}

	if err := store.SetLockWarns(ctx, bot.ID, -100123, true); err != nil {
		t.Fatalf("SetLockWarns() error = %v", err)
	}
	if err := store.UpsertLock(ctx, domain.LockRule{
		BotID:                 bot.ID,
		ChatID:                -100123,
		LockType:              "invitelink",
		Action:                "ban",
		ActionDurationSeconds: 0,
		Reason:                "No promos",
	}); err != nil {
		t.Fatalf("UpsertLock() error = %v", err)
	}
	if err := store.AddLockAllowlist(ctx, domain.LockAllowlistEntry{
		BotID:  bot.ID,
		ChatID: -100123,
		Item:   "@trustedchannel",
	}); err != nil {
		t.Fatalf("AddLockAllowlist() error = %v", err)
	}

	bundle, err = store.LoadRuntimeBundle(ctx, bot.ID, -100123)
	if err != nil {
		t.Fatalf("LoadRuntimeBundle() after locks error = %v", err)
	}
	if !bundle.Settings.LockWarns {
		t.Fatal("expected lock warnings to persist")
	}
	lock, ok := bundle.Locks["invitelink"]
	if !ok || lock.Action != "ban" || lock.Reason != "No promos" {
		t.Fatalf("expected custom lock to persist, got %+v", bundle.Locks)
	}
	if len(bundle.LockAllowlist) != 1 || bundle.LockAllowlist[0] != "@trustedchannel" {
		t.Fatalf("expected lock allowlist to persist, got %+v", bundle.LockAllowlist)
	}

	job := domain.Job{
		ID:           "job_broadcast",
		BotID:        bot.ID,
		Kind:         "broadcast",
		Status:       "pending",
		RequestedBy:  1001,
		ReportChatID: -100123,
		AvailableAt:  time.Now(),
	}
	if err := store.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	jobs, err := store.ClaimPendingJobs(ctx, "integration-worker", 5)
	if err != nil {
		t.Fatalf("ClaimPendingJobs() error = %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != job.ID {
		t.Fatalf("ClaimPendingJobs() = %+v, want job %q", jobs, job.ID)
	}
	if err := store.UpdateJobProgress(ctx, job.ID, "processing", 1, 3, ""); err != nil {
		t.Fatalf("UpdateJobProgress() error = %v", err)
	}
	if err := store.MarkJobCompleted(ctx, job.ID, "done"); err != nil {
		t.Fatalf("MarkJobCompleted() error = %v", err)
	}

	stats, err := store.GetStats(ctx, bot.ID)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if stats.ChatCount < 1 || stats.UserCount < 2 || stats.JobCount < 1 {
		t.Fatalf("GetStats() = %+v, expected chat/user/job counts to be populated", stats)
	}

	cloneA, err := store.CreateCloneBot(ctx, domain.BotInstance{
		ID:            "clone_a",
		Slug:          "clonea",
		DisplayName:   "Clone A",
		TelegramToken: "clone-token-a",
		WebhookKey:    "clone-hook-a",
		WebhookSecret: "clone-secret-a",
		Username:      "clonea_bot",
	}, 1001)
	if err != nil {
		t.Fatalf("CreateCloneBot(first) error = %v", err)
	}
	if cloneA.ID == "" {
		t.Fatal("expected first clone to persist")
	}
	if _, err := store.CreateCloneBot(ctx, domain.BotInstance{
		ID:            "clone_b",
		Slug:          "cloneb",
		DisplayName:   "Clone B",
		TelegramToken: "clone-token-b",
		WebhookKey:    "clone-hook-b",
		WebhookSecret: "clone-secret-b",
		Username:      "cloneb_bot",
	}, 1001); err != persistence.ErrCloneLimitReached {
		t.Fatalf("CreateCloneBot(second) error = %v, want ErrCloneLimitReached", err)
	}
}

func resetDatabase(t *testing.T, ctx context.Context, databaseURL string) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store, err := New(ctx, databaseURL, logger)
	if err != nil {
		t.Fatalf("New() for reset error = %v", err)
	}
	defer store.Close()

	if _, err := store.pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema error = %v", err)
	}
}
