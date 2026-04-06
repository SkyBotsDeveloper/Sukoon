package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/observability"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/router"
	"sukoon/bot-core/internal/telegram"
)

type Processor struct {
	store   persistence.Store
	factory telegram.Factory
	router  *router.Router
	logger  *slog.Logger
	metrics observability.Metrics
}

func New(store persistence.Store, factory telegram.Factory, router *router.Router, logger *slog.Logger) *Processor {
	return NewWithMetrics(store, factory, router, logger, observability.NewNoopMetrics())
}

func NewWithMetrics(store persistence.Store, factory telegram.Factory, router *router.Router, logger *slog.Logger, metrics observability.Metrics) *Processor {
	return &Processor{
		store:   store,
		factory: factory,
		router:  router,
		logger:  logger,
		metrics: metrics,
	}
}

func (p *Processor) Process(ctx context.Context, job domain.QueuedUpdate) error {
	start := time.Now()
	bot, err := p.store.GetBotByID(ctx, job.BotID)
	if err != nil {
		return fmt.Errorf("load bot: %w", err)
	}

	var update telegram.Update
	if err := json.Unmarshal(job.Payload, &update); err != nil {
		return fmt.Errorf("decode update: %w", err)
	}

	client := p.factory.ForBot(bot)
	updateLogger := p.logger.With(
		"bot_id", bot.ID,
		"update_queue_id", job.ID,
		"update_id", job.UpdateID,
		"chat_id", updateChatID(update),
	)
	err = p.router.HandleUpdate(ctx, bot, client, update)
	duration := time.Since(start)
	if err != nil {
		p.metrics.IncCounter("update_failures_total", "bot_id", bot.ID)
		p.metrics.ObserveDuration("update_duration", duration, "bot_id", bot.ID, "status", "error")
		updateLogger.Error("update failed", "duration_ms", duration.Milliseconds(), "error", err)
		return err
	}
	p.metrics.IncCounter("updates_processed_total", "bot_id", bot.ID)
	p.metrics.ObserveDuration("update_duration", duration, "bot_id", bot.ID, "status", "ok")
	updateLogger.Info("update processed", "duration_ms", duration.Milliseconds())
	return nil
}

func updateChatID(update telegram.Update) int64 {
	switch {
	case update.Message != nil:
		return update.Message.Chat.ID
	case update.EditedMessage != nil:
		return update.EditedMessage.Chat.ID
	case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
		return update.CallbackQuery.Message.Chat.ID
	default:
		return 0
	}
}
