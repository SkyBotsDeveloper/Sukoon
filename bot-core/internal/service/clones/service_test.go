package clones_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
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
	permissionService := permissions.New(store)
	router := router.New(
		store,
		state,
		permissionService,
		moderationservice.New(),
		adminservice.New(jobService, permissionService),
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

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/clone clone-token",
		},
	}); err != nil {
		t.Fatalf("clone recreation after rmclone failed: %v", err)
	}

	owned, err = store.ListOwnedBots(context.Background(), 1)
	if err != nil {
		t.Fatalf("list owned bots after recreation failed: %v", err)
	}
	if len(owned) != 2 {
		t.Fatalf("expected primary bot plus recreated clone, got %+v", owned)
	}
}

func TestCloneCreationIsLimitedToOneClonePerOwner(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	primaryClient := testsupport.NewFakeTelegramClient()
	cloneClientA := testsupport.NewFakeTelegramClient()
	cloneClientA.Me = telegram.User{ID: 200, FirstName: "Clone", Username: "clonebotone"}
	cloneClientB := testsupport.NewFakeTelegramClient()
	cloneClientB.Me = telegram.User{ID: 201, FirstName: "Clone", Username: "clonebottwo"}

	factory := tokenFactory{
		defaultClient: primaryClient,
		byToken: map[string]*testsupport.FakeTelegramClient{
			"clone-token-a": cloneClientA,
			"clone-token-b": cloneClientB,
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
	permissionService := permissions.New(store)
	router := router.New(
		store,
		state,
		permissionService,
		moderationservice.New(),
		adminservice.New(jobService, permissionService),
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
			Text:      "/clone clone-token-a",
		},
	}); err != nil {
		t.Fatalf("first clone creation failed: %v", err)
	}

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/clone clone-token-b",
		},
	}); err != nil {
		t.Fatalf("second clone creation attempt failed unexpectedly: %v", err)
	}

	owned, err := store.ListOwnedBots(context.Background(), 1)
	if err != nil {
		t.Fatalf("list owned bots failed: %v", err)
	}
	if len(owned) != 2 {
		t.Fatalf("expected primary bot plus one clone only, got %+v", owned)
	}
	if len(cloneClientA.Webhooks) != 1 {
		t.Fatalf("expected first clone webhook registration, got %+v", cloneClientA.Webhooks)
	}
	if len(cloneClientB.Webhooks) != 0 {
		t.Fatalf("expected second clone to be rejected before webhook registration, got %+v", cloneClientB.Webhooks)
	}
	last := primaryClient.Messages[len(primaryClient.Messages)-1]
	if last.Text != "Only one Sukoon clone is allowed per account. Remove your existing clone with /rmclone before creating another." {
		t.Fatalf("expected clone limit warning, got %q", last.Text)
	}
}

func TestRevokedCloneIsAutoRemovedBeforeReplacement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	primaryClient := testsupport.NewFakeTelegramClient()
	cloneClientA := testsupport.NewFakeTelegramClient()
	cloneClientA.Me = telegram.User{ID: 200, FirstName: "Clone", Username: "clonebotone"}
	cloneClientB := testsupport.NewFakeTelegramClient()
	cloneClientB.Me = telegram.User{ID: 201, FirstName: "Clone", Username: "clonebottwo"}

	factory := tokenFactory{
		defaultClient: primaryClient,
		byToken: map[string]*testsupport.FakeTelegramClient{
			"clone-token-a": cloneClientA,
			"clone-token-b": cloneClientB,
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
	permissionService := permissions.New(store)
	router := router.New(
		store,
		state,
		permissionService,
		moderationservice.New(),
		adminservice.New(jobService, permissionService),
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
			Text:      "/clone clone-token-a",
		},
	}); err != nil {
		t.Fatalf("first clone creation failed: %v", err)
	}

	cloneClientA.GetMeError = errors.New("unauthorized")

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/clone clone-token-b",
		},
	}); err != nil {
		t.Fatalf("replacement clone creation failed: %v", err)
	}

	owned, err := store.ListOwnedBots(context.Background(), 1)
	if err != nil {
		t.Fatalf("list owned bots failed: %v", err)
	}
	if len(owned) != 2 {
		t.Fatalf("expected primary bot plus replacement clone, got %+v", owned)
	}
	if len(cloneClientB.Webhooks) != 1 {
		t.Fatalf("expected replacement clone webhook registration, got %+v", cloneClientB.Webhooks)
	}
	if len(cloneClientA.Webhooks) != 1 {
		t.Fatalf("expected original clone webhook history to remain at one create call, got %+v", cloneClientA.Webhooks)
	}
	last := primaryClient.Messages[len(primaryClient.Messages)-1]
	if last.Text != "Clone created: @clonebottwo" {
		t.Fatalf("expected replacement clone success message, got %q", last.Text)
	}
}

func TestMyBotListsOwnedCloneAndSupportsRestartAndDelete(t *testing.T) {
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
	permissionService := permissions.New(store)
	router := router.New(
		store,
		state,
		permissionService,
		moderationservice.New(),
		adminservice.New(jobService, permissionService),
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

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/mybot",
		},
	}); err != nil {
		t.Fatalf("mybot failed: %v", err)
	}

	menu := primaryClient.Messages[len(primaryClient.Messages)-1]
	if !strings.Contains(menu.Text, "Your Sukoon Bots") {
		t.Fatalf("expected mybot menu text, got %q", menu.Text)
	}
	if menu.Options.ReplyMarkup == nil || len(menu.Options.ReplyMarkup.InlineKeyboard) < 2 {
		t.Fatalf("expected inline mybot menu, got %+v", menu.Options)
	}

	cloneID := ""
	owned, err := store.ListOwnedBots(context.Background(), 1)
	if err != nil {
		t.Fatalf("list owned bots failed: %v", err)
	}
	for _, ownedBot := range owned {
		if !ownedBot.IsPrimary {
			cloneID = ownedBot.ID
			break
		}
	}
	if cloneID == "" {
		t.Fatal("expected owned clone id")
	}

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-select-clone",
			From: telegram.User{ID: 1, FirstName: "Owner"},
			Message: &telegram.Message{
				MessageID: menu.MessageID,
				Chat:      chat,
			},
			Data: "clone:select:" + cloneID,
		},
	}); err != nil {
		t.Fatalf("clone select callback failed: %v", err)
	}

	manage := primaryClient.EditedMessages[len(primaryClient.EditedMessages)-1]
	if !strings.Contains(manage.Text, "Manage Your Sukoon Clone") || !strings.Contains(manage.Text, "@clonebot") {
		t.Fatalf("expected manage clone page, got %q", manage.Text)
	}

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 4,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-restart-clone",
			From: telegram.User{ID: 1, FirstName: "Owner"},
			Message: &telegram.Message{
				MessageID: menu.MessageID,
				Chat:      chat,
			},
			Data: "clone:restart:" + cloneID,
		},
	}); err != nil {
		t.Fatalf("clone restart callback failed: %v", err)
	}

	restarted := primaryClient.EditedMessages[len(primaryClient.EditedMessages)-1]
	if !strings.Contains(restarted.Text, "Clone restarted successfully.") {
		t.Fatalf("expected restart confirmation, got %q", restarted.Text)
	}
	if len(cloneClient.Webhooks) != 2 {
		t.Fatalf("expected clone restart to resync webhook, got %+v", cloneClient.Webhooks)
	}

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 5,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-delete-clone",
			From: telegram.User{ID: 1, FirstName: "Owner"},
			Message: &telegram.Message{
				MessageID: menu.MessageID,
				Chat:      chat,
			},
			Data: "clone:delete:" + cloneID,
		},
	}); err != nil {
		t.Fatalf("clone delete callback failed: %v", err)
	}

	confirm := primaryClient.EditedMessages[len(primaryClient.EditedMessages)-1]
	if !strings.Contains(confirm.Text, "Delete This Clone?") {
		t.Fatalf("expected delete confirmation page, got %q", confirm.Text)
	}

	if err := router.HandleUpdate(context.Background(), bot, primaryClient, telegram.Update{
		UpdateID: 6,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-confirm-delete-clone",
			From: telegram.User{ID: 1, FirstName: "Owner"},
			Message: &telegram.Message{
				MessageID: menu.MessageID,
				Chat:      chat,
			},
			Data: "clone:confirm:" + cloneID,
		},
	}); err != nil {
		t.Fatalf("clone confirm delete callback failed: %v", err)
	}

	afterDelete := primaryClient.EditedMessages[len(primaryClient.EditedMessages)-1]
	if !strings.Contains(afterDelete.Text, "Removed @clonebot.") || !strings.Contains(afterDelete.Text, "No Sukoon clone is linked to your account right now.") {
		t.Fatalf("expected mybot list after delete, got %q", afterDelete.Text)
	}
	if cloneClient.DeletedWebhooks != 1 {
		t.Fatalf("expected clone delete to delete webhook, got %d", cloneClient.DeletedWebhooks)
	}
}

func TestMyBotShowsEmptyStateWithoutClone(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	primaryClient := testsupport.NewFakeTelegramClient()

	factory := tokenFactory{
		defaultClient: primaryClient,
		byToken:       map[string]*testsupport.FakeTelegramClient{},
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
	permissionService := permissions.New(store)
	router := router.New(
		store,
		state,
		permissionService,
		moderationservice.New(),
		adminservice.New(jobService, permissionService),
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
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/mybot",
		},
	}); err != nil {
		t.Fatalf("mybot empty state failed: %v", err)
	}

	menu := primaryClient.Messages[len(primaryClient.Messages)-1]
	if !strings.Contains(menu.Text, "No Sukoon clone is linked to your account right now.") {
		t.Fatalf("expected empty mybot state, got %q", menu.Text)
	}
}
