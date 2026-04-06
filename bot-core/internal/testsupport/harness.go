package testsupport

import (
	"context"
	"log/slog"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/permissions"
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
)

type Harness struct {
	Store  *MemoryStore
	State  *MemoryState
	Client *FakeTelegramClient
	Router *router.Router
	Bot    domain.BotInstance
}

func NewHarness(logger *slog.Logger) *Harness {
	store := NewMemoryStore()
	state := NewMemoryState()
	client := NewFakeTelegramClient()
	factory := StaticClientFactory{Client: client}
	jobService := jobs.New(store, factory, logger)
	ownerService := ownerservice.New(jobService)
	federationService := federationservice.New(jobService)
	cloneService := clonesservice.New(store, factory, "https://example.test", logger)
	antiabuseService := antiabuseservice.New()
	antibioService := antibioservice.New()
	utilityService := utilityservice.New()

	bot := domain.BotInstance{
		ID:            "bot-1",
		Slug:          "primary",
		DisplayName:   "Sukoon",
		TelegramToken: "token-1",
		WebhookKey:    "hook-1",
		WebhookSecret: "secret-1",
		Username:      "sukoon_bot",
		IsPrimary:     true,
	}
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, []int64{1})

	router := router.New(
		store,
		state,
		permissions.New(store),
		moderationservice.New(),
		adminservice.New(jobService),
		antispamservice.New(),
		contentservice.New(),
		captchaservice.New(store, factory, logger),
		afkservice.New(),
		ownerService,
		federationService,
		cloneService,
		antiabuseService,
		antibioService,
		utilityService,
		logger,
	)

	return &Harness{
		Store:  store,
		State:  state,
		Client: client,
		Router: router,
		Bot:    bot,
	}
}
