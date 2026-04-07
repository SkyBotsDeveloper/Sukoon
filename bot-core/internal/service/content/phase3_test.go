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

func TestNotesAndFiltersListingAndRulesAliases(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100732, Type: "supergroup", Title: "Discovery"}

	for idx, cmd := range []string{
		"/save greet Hello there",
		"/save rulesnote Read the rules",
		"/filter hello Hi",
		"/filter bye Bye",
		"/setwelcome on Welcome {first}",
		"/setgoodbye on Bye {first}",
		"/setrules Be respectful",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 10),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", cmd, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 20,
		Message: &telegram.Message{
			MessageID: 40,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
			Text:      "/notes",
		},
	}); err != nil {
		t.Fatalf("notes failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 21,
		Message: &telegram.Message{
			MessageID: 41,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/filters",
		},
	}); err != nil {
		t.Fatalf("filters failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 22,
		Message: &telegram.Message{
			MessageID: 42,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/resetrules",
		},
	}); err != nil {
		t.Fatalf("resetrules failed: %v", err)
	}

	notesMsg := h.Client.Messages[len(h.Client.Messages)-3]
	if notesMsg.Text != "Saved notes: #greet, #rulesnote" {
		t.Fatalf("unexpected notes listing: %q", notesMsg.Text)
	}

	filtersMsg := h.Client.Messages[len(h.Client.Messages)-2]
	if filtersMsg.Text != "Saved filters: hello, bye" && filtersMsg.Text != "Saved filters: bye, hello" {
		t.Fatalf("unexpected filters listing: %q", filtersMsg.Text)
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if bundle.Settings.WelcomeText == "" || bundle.Settings.GoodbyeText == "" {
		t.Fatalf("expected setwelcome/setgoodbye aliases to update settings, got %+v", bundle.Settings)
	}
	if bundle.Settings.RulesText != "" {
		t.Fatalf("expected resetrules to clear rules, got %q", bundle.Settings.RulesText)
	}
}
