package moderation_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestWarnLimitAndModeEscalateToBan(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100200, Type: "supergroup", Title: "Warns"}

	for _, text := range []string{"/setwarnlimit 2", "/setwarnmode ban"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(len(h.Client.Messages) + 1),
			Message: &telegram.Message{
				MessageID: int64(len(h.Client.Messages) + 10),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	for i := 0; i < 2; i++ {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(100 + i),
			Message: &telegram.Message{
				MessageID: int64(1000 + i),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      "/warn rude",
				ReplyToMessage: &telegram.Message{
					MessageID: 99,
					From:      &telegram.User{ID: 20, FirstName: "Target"},
					Chat:      chat,
				},
			},
		}); err != nil {
			t.Fatalf("warn failed: %v", err)
		}
	}

	if len(h.Client.Bans) != 1 || h.Client.Bans[0].UserID != 20 {
		t.Fatalf("expected warn escalation ban, got %+v", h.Client.Bans)
	}
}

func TestBanAliasesAndKickMe(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100201, Type: "supergroup", Title: "Ban Aliases"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 20,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/dban spam links",
			ReplyToMessage: &telegram.Message{
				MessageID: 19,
				From:      &telegram.User{ID: 20, FirstName: "Target"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("dban failed: %v", err)
	}

	if len(h.Client.Bans) == 0 || h.Client.Bans[0].UserID != 20 {
		t.Fatalf("expected dban to ban target, got %+v", h.Client.Bans)
	}
	if len(h.Client.DeletedMessages) == 0 || h.Client.DeletedMessages[0].MessageID != 20 {
		t.Fatalf("expected dban to delete the command message, got %+v", h.Client.DeletedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 21,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/sban spam links",
			ReplyToMessage: &telegram.Message{
				MessageID: 18,
				From:      &telegram.User{ID: 21, FirstName: "Target2"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("sban failed: %v", err)
	}

	if len(h.Client.Bans) < 2 || h.Client.Bans[1].UserID != 21 {
		t.Fatalf("expected sban to ban second target, got %+v", h.Client.Bans)
	}

	messageCount := len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 22,
			From:      &telegram.User{ID: 30, FirstName: "SelfKick"},
			Chat:      chat,
			Text:      "/kickme",
		},
	}); err != nil {
		t.Fatalf("kickme failed: %v", err)
	}

	if len(h.Client.Bans) < 3 || h.Client.Bans[2].UserID != 30 {
		t.Fatalf("expected kickme to ban actor temporarily, got %+v", h.Client.Bans)
	}
	if len(h.Client.Unbans) == 0 || h.Client.Unbans[len(h.Client.Unbans)-1].UserID != 30 {
		t.Fatalf("expected kickme to unban actor after temporary kick, got %+v", h.Client.Unbans)
	}
	if len(h.Client.Messages) != messageCount+1 {
		t.Fatalf("expected kickme confirmation message, got %+v", h.Client.Messages)
	}
}
