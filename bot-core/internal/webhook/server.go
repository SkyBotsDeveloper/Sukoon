package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/observability"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/state"
)

type Server struct {
	store   persistence.Store
	state   state.Store
	logger  *slog.Logger
	metrics observability.Metrics
}

func New(store persistence.Store, state state.Store, logger *slog.Logger) *Server {
	return NewWithMetrics(store, state, logger, observability.NewNoopMetrics())
}

func NewWithMetrics(store persistence.Store, state state.Store, logger *slog.Logger, metrics observability.Metrics) *Server {
	return &Server{store: store, state: state, logger: logger, metrics: metrics}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/webhook/", s.handleWebhook)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.metrics.IncCounter("webhook_rejected_total", "reason", "method")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	webhookKey := strings.TrimPrefix(r.URL.Path, "/webhook/")
	if webhookKey == "" {
		s.metrics.IncCounter("webhook_rejected_total", "reason", "missing_key")
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "missing webhook key"})
		return
	}

	ctx := r.Context()
	bot, err := s.resolveBot(ctx, webhookKey)
	if err != nil {
		s.logger.Error("bot resolution failed", "webhook_key", webhookKey, "error", err)
		s.metrics.IncCounter("webhook_rejected_total", "reason", "unknown_bot")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unknown bot"})
		return
	}

	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != bot.WebhookSecret {
		s.metrics.IncCounter("webhook_rejected_total", "reason", "secret")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid secret"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		s.metrics.IncCounter("webhook_rejected_total", "reason", "body")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	var payload struct {
		UpdateID int64 `json:"update_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.UpdateID == 0 {
		s.metrics.IncCounter("webhook_rejected_total", "reason", "payload")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid telegram update"})
		return
	}

	enqueued, err := s.store.EnqueueUpdate(ctx, bot.ID, payload.UpdateID, body)
	if err != nil {
		s.logger.Error("enqueue failed", "bot_id", bot.ID, "update_id", payload.UpdateID, "error", err)
		s.metrics.IncCounter("webhook_enqueue_failures_total", "bot_id", bot.ID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "enqueue failed"})
		return
	}
	if enqueued {
		s.metrics.IncCounter("webhook_accepted_total", "bot_id", bot.ID)
	} else {
		s.metrics.IncCounter("webhook_duplicates_total", "bot_id", bot.ID)
	}
	s.metrics.ObserveDuration("webhook_ack_duration", time.Since(start), "bot_id", bot.ID)
	s.logger.Info("webhook accepted", "bot_id", bot.ID, "update_id", payload.UpdateID, "duplicate", !enqueued, "duration_ms", time.Since(start).Milliseconds())

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":    "accepted",
		"duplicate": !enqueued,
	})
}

func (s *Server) resolveBot(ctx context.Context, webhookKey string) (domain.BotInstance, error) {
	if bot, ok, err := s.state.GetCachedBot(ctx, webhookKey); err == nil && ok {
		return bot, nil
	}
	bot, err := s.store.ResolveBotByWebhookKey(ctx, webhookKey)
	if err != nil {
		return domain.BotInstance{}, err
	}
	_ = s.state.CacheBot(ctx, bot, 10*time.Minute)
	return bot, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
