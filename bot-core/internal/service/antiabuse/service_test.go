package antiabuse_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestAntiAbuseTargetsRealAbuseWordsOnly(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100740, Type: "supergroup", Title: "Abuse"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/antiabuse on warn",
		},
	}); err != nil {
		t.Fatalf("enable antiabuse failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
			Text:      "You are a bastard",
		},
	}); err != nil {
		t.Fatalf("abusive message failed: %v", err)
	}

	if len(h.Client.DeletedMessages) == 0 {
		t.Fatalf("expected antiabuse to delete abusive message")
	}
	count, err := h.Store.GetWarnings(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("get warnings failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one warning after antiabuse hit, got %d", count)
	}
}

func TestAntiAbuseAvoidsFalsePositivesAndAdmins(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100741, Type: "supergroup", Title: "Abuse"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 30, FirstName: "Admin"},
			Status:             "administrator",
			CanDeleteMessages:  true,
			CanRestrictMembers: true,
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/antiabuse on warn",
		},
	}); err != nil {
		t.Fatalf("enable antiabuse failed: %v", err)
	}

	for idx, message := range []telegram.Message{
		{MessageID: 11, From: &telegram.User{ID: 20, FirstName: "User"}, Chat: chat, Text: "My dog is cute"},
		{MessageID: 12, From: &telegram.User{ID: 30, FirstName: "Admin"}, Chat: chat, Text: "bastard"},
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 2),
			Message:  &message,
		}); err != nil {
			t.Fatalf("message %d failed: %v", idx, err)
		}
	}

	count, err := h.Store.GetWarnings(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("get warnings failed: %v", err)
	}
	if count != 0 || len(h.Client.DeletedMessages) != 0 {
		t.Fatalf("expected no antiabuse enforcement for false positive/admin, warnings=%d deletions=%+v", count, h.Client.DeletedMessages)
	}
}
