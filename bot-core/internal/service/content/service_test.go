package content_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestFilterMatchesAndResponds(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100500, Type: "supergroup", Title: "Filters"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/filter hello Hi there",
		},
	}); err != nil {
		t.Fatalf("save filter failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
			Text:      "hello everyone",
		},
	}); err != nil {
		t.Fatalf("filter message failed: %v", err)
	}

	found := false
	for _, sent := range h.Client.Messages {
		if sent.ChatID == chat.ID && sent.Text == "Hi there" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected filter response message")
	}
}
