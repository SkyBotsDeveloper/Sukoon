package jobs_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestBroadcastJobTracksProgressAndPartialFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	client := testsupport.NewFakeTelegramClient()
	client.SendErrors[-100802] = context.DeadlineExceeded
	factory := testsupport.StaticClientFactory{Client: client}
	service := jobs.New(store, factory, logger)

	bot := domain.BotInstance{ID: "bot-1", Slug: "primary", TelegramToken: "token", WebhookKey: "hook", WebhookSecret: "secret", Username: "bot"}
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, []int64{1})
	for _, chat := range []telegram.Chat{
		{ID: -100801, Type: "supergroup", Title: "One"},
		{ID: -100802, Type: "supergroup", Title: "Two"},
	} {
		if err := store.EnsureChat(context.Background(), bot.ID, chat); err != nil {
			t.Fatalf("ensure chat failed: %v", err)
		}
	}

	job, err := service.Enqueue(context.Background(), bot.ID, jobs.KindBroadcast, 1, -100900, jobs.BroadcastPayload{Mode: "chats", Text: "hello"}, 2)
	if err != nil {
		t.Fatalf("enqueue job failed: %v", err)
	}
	summary, err := service.Process(context.Background(), job)
	if err != nil {
		t.Fatalf("process job failed: %v", err)
	}
	if !strings.Contains(summary, "failed=1") {
		t.Fatalf("expected partial failure summary, got %q", summary)
	}
	stored, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job failed: %v", err)
	}
	if stored.Progress != 2 || stored.Total != 2 {
		t.Fatalf("expected stored job progress 2/2, got %d/%d", stored.Progress, stored.Total)
	}
}
