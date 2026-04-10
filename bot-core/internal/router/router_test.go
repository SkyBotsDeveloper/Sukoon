package router_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"sukoon/bot-core/internal/permissions"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestRoutesBanCommandToModeration(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	update := telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      telegram.Chat{ID: -100123, Type: "supergroup", Title: "Audit"},
			Text:      "/ban spammer",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Spammer"},
				Chat:      telegram.Chat{ID: -100123, Type: "supergroup"},
			},
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, update); err != nil {
		t.Fatalf("handle update: %v", err)
	}
	if len(h.Client.Bans) != 1 || h.Client.Bans[0].UserID != 20 {
		t.Fatalf("expected one ban for user 20, got %+v", h.Client.Bans)
	}
	if len(h.Client.Messages) != 1 {
		t.Fatalf("expected one confirmation message, got %+v", h.Client.Messages)
	}
	if h.Client.Messages[0].Options.ReplyToMessageID != 10 {
		t.Fatalf("expected confirmation reply to command message 10, got %+v", h.Client.Messages[0].Options)
	}
}

func TestRepeatedGroupCommandsDebounceEnsuresAndRoleLookups(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	update := telegram.Update{
		Message: &telegram.Message{
			From: &telegram.User{ID: 1, FirstName: "Owner"},
			Chat: telegram.Chat{ID: -100124, Type: "supergroup", Title: "Flood"},
			Text: "/flood",
		},
	}

	update.UpdateID = 1
	update.Message.MessageID = 11
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, update); err != nil {
		t.Fatalf("first command failed: %v", err)
	}
	update.UpdateID = 2
	update.Message.MessageID = 12
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, update); err != nil {
		t.Fatalf("second command failed: %v", err)
	}

	if h.Store.EnsureChatCalls != 1 {
		t.Fatalf("expected one EnsureChat call, got %d", h.Store.EnsureChatCalls)
	}
	if h.Store.EnsureUserCalls != 1 {
		t.Fatalf("expected one EnsureUser call, got %d", h.Store.EnsureUserCalls)
	}
	if h.Store.GetBotRolesCalls != 1 {
		t.Fatalf("expected one GetBotRoles call, got %d", h.Store.GetBotRolesCalls)
	}
}

func TestRepeatedAdminCommandsReuseCachedAdminLookup(t *testing.T) {
	store := testsupport.NewMemoryStore()
	state := testsupport.NewMemoryState()
	client := testsupport.NewFakeTelegramClient()
	client.AdminsByChat[-100125] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 99, FirstName: "Admin"},
			Status:             "administrator",
			CanRestrictMembers: true,
			CanDeleteMessages:  true,
		},
	}
	bot := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil))).Bot
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, nil)

	service := permissions.New(store)
	for i := 0; i < 2; i++ {
		_, err := service.Load(context.Background(), bot.ID, 99, -100125, "supergroup", client)
		if err != nil {
			t.Fatalf("load %d failed: %v", i, err)
		}
	}
	_ = state
	if len(client.AdminLookups) != 1 {
		t.Fatalf("expected one admin lookup, got %+v", client.AdminLookups)
	}
}

func TestCleanCommandCategoriesDeleteHandledAndOtherCommands(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100126, Type: "supergroup", Title: "Clean"}

	for idx, text := range []string{
		"/cleancommand user other",
		"/help",
		"/start@otherbot",
		"/keepcommand all",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(20 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", text, err)
		}
	}

	if len(h.Client.DeletedMessages) < 2 {
		t.Fatalf("expected clean command deletes, got %+v", h.Client.DeletedMessages)
	}
	if h.Client.DeletedMessages[0].MessageID != 21 {
		t.Fatalf("expected /help message to be cleaned as a user command, got %+v", h.Client.DeletedMessages)
	}
	if h.Client.DeletedMessages[1].MessageID != 22 {
		t.Fatalf("expected /start@otherbot message to be cleaned as other command, got %+v", h.Client.DeletedMessages)
	}
	if len(h.Client.Messages) == 0 || !strings.Contains(h.Client.Messages[1].Text, "easier to browse in PM") {
		t.Fatalf("expected /help to still respond before delete, got %+v", h.Client.Messages)
	}
}
