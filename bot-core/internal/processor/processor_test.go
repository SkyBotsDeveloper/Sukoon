package processor_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/permissions"
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
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

type multiFactory struct {
	clients map[string]*testsupport.FakeTelegramClient
}

func (m multiFactory) ForBot(bot domain.BotInstance) telegram.Client {
	return m.clients[bot.ID]
}

func TestProcessorUsesCorrectBotClientForCloneIsolation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	client1 := testsupport.NewFakeTelegramClient()
	client2 := testsupport.NewFakeTelegramClient()

	bot1 := domain.BotInstance{ID: "bot-1", Slug: "primary", TelegramToken: "token1", WebhookKey: "hook1", WebhookSecret: "secret1", Username: "bot1"}
	bot2 := domain.BotInstance{ID: "bot-2", Slug: "clone", TelegramToken: "token2", WebhookKey: "hook2", WebhookSecret: "secret2", Username: "bot2"}
	_, _ = store.UpsertPrimaryBot(context.Background(), bot1, nil)
	_, _ = store.UpsertPrimaryBot(context.Background(), bot2, nil)
	client2.AdminsByChat[-100600] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 30},
			Status:             "administrator",
			CanRestrictMembers: true,
		},
	}

	factory := multiFactory{clients: map[string]*testsupport.FakeTelegramClient{
		"bot-1": client1,
		"bot-2": client2,
	}}
	jobService := jobs.New(store, factory, logger)
	ownerService := ownerservice.New(jobService)
	federationService := federationservice.New(jobService)
	cloneService := clonesservice.New(store, factory, "https://example.test", logger)
	antiabuseService := antiabuseservice.New()
	antibioService := antibioservice.New()
	utilityService := utilityservice.New()

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
	processor := processor.New(store, factory, router, logger)

	payload := []byte(`{"update_id":1,"message":{"message_id":10,"from":{"id":30,"first_name":"Admin"},"chat":{"id":-100600,"type":"supergroup","title":"Clone"},"text":"/ban","reply_to_message":{"message_id":9,"from":{"id":40,"first_name":"Target"},"chat":{"id":-100600,"type":"supergroup"}}}}`)
	if _, err := store.EnqueueUpdate(context.Background(), "bot-2", 1, payload); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	jobs, err := store.ClaimPendingUpdates(context.Background(), "test", 1)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("claim updates failed: %v jobs=%d", err, len(jobs))
	}
	if err := processor.Process(context.Background(), jobs[0]); err != nil {
		t.Fatalf("process failed: %v", err)
	}
	if len(client1.Bans) != 0 {
		t.Fatalf("expected primary bot client to stay untouched, got %+v", client1.Bans)
	}
	if len(client2.Bans) != 1 || client2.Bans[0].UserID != 40 {
		t.Fatalf("expected clone bot client to ban target, got %+v", client2.Bans)
	}
}
