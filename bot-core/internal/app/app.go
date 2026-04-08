package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"sukoon/bot-core/internal/config"
	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/observability"
	"sukoon/bot-core/internal/permissions"
	"sukoon/bot-core/internal/persistence/postgres"
	"sukoon/bot-core/internal/processor"
	"sukoon/bot-core/internal/router"
	adminservice "sukoon/bot-core/internal/service/admin"
	afkservice "sukoon/bot-core/internal/service/afk"
	antiabuseservice "sukoon/bot-core/internal/service/antiabuse"
	antibioservice "sukoon/bot-core/internal/service/antibio"
	antispamservice "sukoon/bot-core/internal/service/antispam"
	captchaservice "sukoon/bot-core/internal/service/captcha"
	clonesservice "sukoon/bot-core/internal/service/clones"
	contentservice "sukoon/bot-core/internal/service/content"
	federationservice "sukoon/bot-core/internal/service/federation"
	moderationservice "sukoon/bot-core/internal/service/moderation"
	ownerservice "sukoon/bot-core/internal/service/owner"
	utilityservice "sukoon/bot-core/internal/service/utility"
	redisstate "sukoon/bot-core/internal/state/redis"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
	"sukoon/bot-core/internal/webhook"
	"sukoon/bot-core/internal/worker"
)

type Runtime struct {
	HTTPServer *http.Server
	Worker     *worker.Service
	closeFn    func()
}

func (r *Runtime) Close() {
	if r.closeFn != nil {
		r.closeFn()
	}
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Runtime, error) {
	store, err := postgres.New(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		store.Close()
		return nil, err
	}

	state := redisstate.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if cfg.PrimaryBot.Token != "" {
		_, err := store.UpsertPrimaryBot(ctx, domain.BotInstance{
			ID:            util.RandomID(16),
			Slug:          cfg.PrimaryBot.Slug,
			DisplayName:   cfg.PrimaryBot.DisplayName,
			TelegramToken: cfg.PrimaryBot.Token,
			WebhookKey:    cfg.PrimaryBot.WebhookKey,
			WebhookSecret: cfg.PrimaryBot.WebhookSecret,
			Username:      cfg.PrimaryBot.Username,
			IsPrimary:     true,
		}, cfg.BotOwnerUserIDs)
		if err != nil {
			state.Close()
			store.Close()
			return nil, fmt.Errorf("upsert primary bot: %w", err)
		}
	}

	factory := telegram.NewHTTPFactory(cfg.TelegramBaseURL, cfg.TelegramRequestTimeout, cfg.TelegramMaxRetries, cfg.TelegramInitialBackoff, logger)
	metrics := observability.NewNoopMetrics()
	jobService := jobs.New(store, factory, logger)
	permissionService := permissions.New(store)
	moderation := moderationservice.New()
	admin := adminservice.New(jobService, permissionService)
	antispam := antispamservice.New()
	content := contentservice.New()
	captcha := captchaservice.New(store, factory, logger)
	afk := afkservice.New()
	owner := ownerservice.New(jobService)
	federation := federationservice.New(jobService)
	clones := clonesservice.New(store, factory, cfg.PublicWebhookBaseURL, logger)
	antiabuse := antiabuseservice.New()
	antibio := antibioservice.New()
	utility := utilityservice.New()

	router := router.New(store, state, permissionService, moderation, admin, antispam, content, captcha, afk, owner, federation, clones, antiabuse, antibio, utility, logger)
	processor := processor.NewWithMetrics(store, factory, router, logger, metrics)
	webhookServer := webhook.NewWithMetrics(store, state, logger, metrics)
	var httpServer *http.Server
	if cfg.AppMode == "all" || cfg.AppMode == "web" {
		httpServer = &http.Server{
			Addr:    cfg.AppAddr,
			Handler: webhookServer.Handler(),
		}
	}
	var workerService *worker.Service
	if cfg.AppMode == "all" || cfg.AppMode == "worker" {
		workerService = worker.NewWithMetrics(store, processor, jobService, captcha, cfg.WorkerConcurrency, cfg.WorkerPollInterval, logger, metrics)
	}

	return &Runtime{
		HTTPServer: httpServer,
		Worker:     workerService,
		closeFn: func() {
			_ = state.Close()
			store.Close()
		},
	}, nil
}
