package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/testsupport"
	"sukoon/bot-core/internal/webhook"
)

func TestWebhookSecretValidationAndIdempotency(t *testing.T) {
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := webhook.New(store, state, logger)

	bot := domain.BotInstance{
		ID:            "bot-1",
		Slug:          "primary",
		DisplayName:   "Sukoon",
		TelegramToken: "token",
		WebhookKey:    "hook-1",
		WebhookSecret: "secret-1",
	}
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, nil)

	payload, _ := json.Marshal(map[string]any{"update_id": 55})

	req := httptest.NewRequest(http.MethodPost, "/webhook/hook-1", bytes.NewReader(payload))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong secret, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/webhook/hook-1", bytes.NewReader(payload))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-1")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for accepted update, got %d", rec.Code)
	}
	if store.QueuedUpdateCount() != 1 {
		t.Fatalf("expected one queued update, got %d", store.QueuedUpdateCount())
	}

	req = httptest.NewRequest(http.MethodPost, "/webhook/hook-1", bytes.NewReader(payload))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-1")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate update to still return 202, got %d", rec.Code)
	}
	if store.QueuedUpdateCount() != 1 {
		t.Fatalf("expected duplicate to stay deduped, got %d", store.QueuedUpdateCount())
	}
}
