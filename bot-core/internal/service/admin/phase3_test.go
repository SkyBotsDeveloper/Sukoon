package admin_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestApproveSupportsUsername(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100710, Type: "supergroup", Title: "Approve"}
	if err := h.Store.EnsureUser(context.Background(), telegram.User{ID: 20, Username: "target", FirstName: "Target"}); err != nil {
		t.Fatalf("ensure user: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/approve @target",
		},
	}); err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	approved, err := h.Store.IsApproved(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("approval lookup failed: %v", err)
	}
	if !approved {
		t.Fatalf("expected approval to be stored for username target")
	}
}

func TestModGrantRequiresPromotePermission(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100711, Type: "supergroup", Title: "Mods"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 10, FirstName: "Limited"},
			Status:             "administrator",
			CanDeleteMessages:  true,
			CanRestrictMembers: true,
		},
		{
			User:               telegram.User{ID: 11, FirstName: "Promoter"},
			Status:             "administrator",
			CanDeleteMessages:  true,
			CanRestrictMembers: true,
			CanPromoteMembers:  true,
		},
	}

	err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 10, FirstName: "Limited"},
			Chat:      chat,
			Text:      "/mod",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Target"},
				Chat:      chat,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected mod grant without promote permission to fail")
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 11, FirstName: "Promoter"},
			Chat:      chat,
			Text:      "/mod",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Target"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("mod grant with promote permission failed: %v", err)
	}

	roles, err := h.Store.GetChatRoles(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("get chat roles: %v", err)
	}
	if len(roles) == 0 || roles[0] != "mod" {
		t.Fatalf("expected mod role after successful grant, got %+v", roles)
	}
}
