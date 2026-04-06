package content_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestFilterBulkRemoval(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100730, Type: "supergroup", Title: "Filters"}

	for idx, cmd := range []string{"/filter hello Hi", "/filter bye Bye"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 10),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("setup filter %q failed: %v", cmd, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 20,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/stop hello | bye",
		},
	}); err != nil {
		t.Fatalf("bulk stop failed: %v", err)
	}

	filters, err := h.Store.ListFilters(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list filters failed: %v", err)
	}
	if len(filters) != 0 {
		t.Fatalf("expected all filters removed, got %+v", filters)
	}
}

func TestSaveNoteParsesButtonsAndRows(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100731, Type: "supergroup", Title: "Notes"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/save greet Hello there\n[Docs](buttonurl:https://example.com) [Ping](button:ping)\n[More](buttonurl:https://example.org)",
		},
	}); err != nil {
		t.Fatalf("save note failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
			Text:      "/get greet",
		},
	}); err != nil {
		t.Fatalf("get note failed: %v", err)
	}

	last := h.Client.Messages[len(h.Client.Messages)-1]
	if last.Options.ReplyMarkup == nil || len(last.Options.ReplyMarkup.InlineKeyboard) != 2 {
		t.Fatalf("expected two button rows, got %+v", last.Options.ReplyMarkup)
	}
	if len(last.Options.ReplyMarkup.InlineKeyboard[0]) != 2 || len(last.Options.ReplyMarkup.InlineKeyboard[1]) != 1 {
		t.Fatalf("unexpected button row layout: %+v", last.Options.ReplyMarkup.InlineKeyboard)
	}
}
