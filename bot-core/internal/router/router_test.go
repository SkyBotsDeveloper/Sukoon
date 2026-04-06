package router_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

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
