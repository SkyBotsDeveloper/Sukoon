package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/testsupport"
)

func TestProcessJobsRetriesBeforeDeadLetter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	client := testsupport.NewFakeTelegramClient()
	factory := testsupport.StaticClientFactory{Client: client}
	jobService := jobs.New(store, factory, logger)

	bot := domain.BotInstance{ID: "bot-1", Slug: "primary", TelegramToken: "token", WebhookKey: "hook", WebhookSecret: "secret", Username: "bot"}
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, []int64{1})
	_ = store.CreateJob(context.Background(), domain.Job{
		ID:          "job-1",
		BotID:       bot.ID,
		Kind:        jobs.KindBroadcast,
		Status:      "pending",
		PayloadJSON: []byte("{"),
		MaxAttempts: 2,
		AvailableAt: time.Now(),
	})

	service := New(store, nil, jobService, nil, 1, time.Millisecond, logger)
	if worked, err := service.processJobs(context.Background(), "worker-1"); err != nil || !worked {
		t.Fatalf("process jobs failed: worked=%t err=%v", worked, err)
	}

	job, err := store.GetJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("get job failed: %v", err)
	}
	if job.Status != "retry" {
		t.Fatalf("expected retry status after first failure, got %s", job.Status)
	}

	job.AvailableAt = time.Now()
	_ = store.MarkJobRetry(context.Background(), "job-1", 1, job.Error, time.Now())
	if worked, err := service.processJobs(context.Background(), "worker-1"); err != nil || !worked {
		t.Fatalf("process jobs second pass failed: worked=%t err=%v", worked, err)
	}
	job, err = store.GetJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("get job failed: %v", err)
	}
	if job.Status != "dead" {
		t.Fatalf("expected dead status after max attempts, got %s", job.Status)
	}
}
