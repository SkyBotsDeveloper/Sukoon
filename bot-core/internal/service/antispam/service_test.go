package antispam_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestAntifloodMutesAfterConfiguredLimit(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100300, Type: "supergroup", Title: "Flood"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/setflood 2",
		},
	}); err != nil {
		t.Fatalf("setflood failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(10 + i),
			Message: &telegram.Message{
				MessageID: int64(20 + i),
				From:      &telegram.User{ID: 20, FirstName: "Flooder"},
				Chat:      chat,
				Text:      "hello",
			},
		}); err != nil {
			t.Fatalf("flood message %d failed: %v", i, err)
		}
	}

	if len(h.Client.Restrictions) == 0 {
		t.Fatalf("expected antiflood restriction")
	}
}
