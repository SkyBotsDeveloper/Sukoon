package admin_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestMuterCanMuteButCannotBan(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100712, Type: "supergroup", Title: "Muters"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/muter",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Muter"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("grant muter failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "Muter"},
			Chat:      chat,
			Text:      "/mute",
			ReplyToMessage: &telegram.Message{
				MessageID: 8,
				From:      &telegram.User{ID: 30, FirstName: "Target"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("mute by muter failed: %v", err)
	}

	if len(h.Client.Restrictions) != 1 || h.Client.Restrictions[0].UserID != 30 {
		t.Fatalf("expected muter to apply one restriction, got %+v", h.Client.Restrictions)
	}

	err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 20, FirstName: "Muter"},
			Chat:      chat,
			Text:      "/ban",
			ReplyToMessage: &telegram.Message{
				MessageID: 7,
				From:      &telegram.User{ID: 31, FirstName: "Target2"},
				Chat:      chat,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected muter ban attempt to fail")
	}
}

func TestPinAndUnpinCommands(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100713, Type: "supergroup", Title: "Pins"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:           telegram.User{ID: 10, FirstName: "Pinner"},
			Status:         "administrator",
			CanPinMessages: true,
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 15,
			From:      &telegram.User{ID: 10, FirstName: "Pinner"},
			Chat:      chat,
			Text:      "/pin quiet",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Pinned"},
				Chat:      chat,
				Text:      "Important",
			},
		},
	}); err != nil {
		t.Fatalf("pin failed: %v", err)
	}
	if len(h.Client.PinnedMessages) != 1 || h.Client.PinnedMessages[0].MessageID != 9 || !h.Client.PinnedMessages[0].DisableNotification {
		t.Fatalf("expected quiet pin of replied message, got %+v", h.Client.PinnedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 16,
			From:      &telegram.User{ID: 10, FirstName: "Pinner"},
			Chat:      chat,
			Text:      "/unpin",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Pinned"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("unpin failed: %v", err)
	}
	if len(h.Client.UnpinnedMessages) != 1 || h.Client.UnpinnedMessages[0].MessageID == nil || *h.Client.UnpinnedMessages[0].MessageID != 9 {
		t.Fatalf("expected specific unpin of message 9, got %+v", h.Client.UnpinnedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 17,
			From:      &telegram.User{ID: 10, FirstName: "Pinner"},
			Chat:      chat,
			Text:      "/unpinall",
		},
	}); err != nil {
		t.Fatalf("unpinall failed: %v", err)
	}
	if len(h.Client.UnpinAllChats) != 1 || h.Client.UnpinAllChats[0] != chat.ID {
		t.Fatalf("expected unpinall for chat %d, got %+v", chat.ID, h.Client.UnpinAllChats)
	}
}
