package clones_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

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
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

type tokenFactory struct {
	defaultClient *testsupport.FakeTelegramClient
	byToken       map[string]*testsupport.FakeTelegramClient
}

func (f tokenFactory) ForBot(bot domain.BotInstance) telegram.Client {
	if client, ok := f.byToken[bot.TelegramToken]; ok {
		return client
	}
	return f.defaultClient
}

func TestCloneLifecycleCreateListAndRemove(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	primaryClient := testsupport.NewFakeTelegramClient()
	cloneClient := testsupport.NewFakeTelegramClient()
	cloneClient.Me = telegram.User{ID: 200, FirstName: "Clone", Username: "clonebot"}

	factory := tokenFactory{
		defaultClient: primaryClient,
		byToken: map[string]*testsupport.FakeTelegramClient{
			"clone-token": cloneClient,
		},
	}

	bot := domain.BotInstance{
		ID:            "bot-1",
		Slug:          "primary",
		DisplayName:   "Sukoon",
		TelegramToken: "primary-token",
		WebhookKey:    "hook-1",
		WebhookSecret: "secret-1",
		Username:      "sukoon_bot",
		IsPrimary:     true,
	}
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, []int64{1})

	jobService := jobs.New(store, factory, logger)
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
		ownerservice.New(jobService),
		federationservice.New(jobService),
		clonesservice.New(store, factory, "https://example.test", logger),
		antiabuseservice.New(),
		antibioservice.New(),
		utilityservice.New(),
		logger,
	)

	chat := telegram.Chat{ID: -100790, Type: "supergroup", Title: "Clones"}
	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/clone clone-token",
		},
	}); err != nil {
		t.Fatalf("clone creation failed: %v", err)
	}

	owned, err := store.ListOwnedBots(context.Background(), 1)
	if err != nil {
		t.Fatalf("list owned bots failed: %v", err)
	}
	if len(owned) != 2 {
		t.Fatalf("expected primary bot plus clone, got %+v", owned)
	}
	if len(cloneClient.Webhooks) != 1 {
		t.Fatalf("expected clone webhook registration, got %+v", cloneClient.Webhooks)
	}

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/rmclone @clonebot",
		},
	}); err != nil {
		t.Fatalf("clone removal failed: %v", err)
	}

	owned, err = store.ListOwnedBots(context.Background(), 1)
	if err != nil {
		t.Fatalf("list owned bots failed: %v", err)
	}
	if len(owned) != 1 {
		t.Fatalf("expected clone removal to leave only primary bot, got %+v", owned)
	}
}
