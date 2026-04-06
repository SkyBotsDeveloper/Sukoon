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
