package content_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
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
			From:      &telegram.User{ID: 20, FirstName: "User"},
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

func TestFiltersSupportQuotedTriggersStopAllAndContentFillings(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100733, Type: "supergroup", Title: "Formatting"}

	for idx, cmd := range []string{
		"/setrules No spam here",
		"/filter \"buy now\" Hello {fullname} from {chatname}",
		"/filter ping Pong %%% Still here %%% Online",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 50),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", cmd, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 10,
		Message: &telegram.Message{
			MessageID: 60,
			From:      &telegram.User{ID: 20, FirstName: "Filter", LastName: "User"},
			Chat:      chat,
			Text:      "please BUY NOW",
		},
	}); err != nil {
		t.Fatalf("quoted filter trigger failed: %v", err)
	}
	quoted := h.Client.Messages[len(h.Client.Messages)-1]
	if quoted.Text != "Hello Filter User from Formatting" {
		t.Fatalf("expected fillings in quoted filter response, got %q", quoted.Text)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 11,
		Message: &telegram.Message{
			MessageID: 61,
			From:      &telegram.User{ID: 20, FirstName: "Filter", LastName: "User"},
			Chat:      chat,
			Text:      "ping",
		},
	}); err != nil {
		t.Fatalf("random filter trigger failed: %v", err)
	}
	randomReply := h.Client.Messages[len(h.Client.Messages)-1].Text
	if randomReply != "Pong" && randomReply != "Still here" && randomReply != "Online" {
		t.Fatalf("expected one configured random reply, got %q", randomReply)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 12,
		Message: &telegram.Message{
			MessageID: 62,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/stopall",
		},
	}); err != nil {
		t.Fatalf("stopall failed: %v", err)
	}

	filters, err := h.Store.ListFilters(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list filters failed: %v", err)
	}
	if len(filters) != 0 {
		t.Fatalf("expected stopall to remove all filters, got %+v", filters)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 13,
		Message: &telegram.Message{
			MessageID: 63,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/save greet Welcome {mention} %%% Read {rules}",
		},
	}); err != nil {
		t.Fatalf("save note with fillings failed: %v", err)
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 14,
		Message: &telegram.Message{
			MessageID: 64,
			From:      &telegram.User{ID: 20, FirstName: "Filter", Username: "filteruser"},
			Chat:      chat,
			Text:      "/get greet",
		},
	}); err != nil {
		t.Fatalf("get filled note failed: %v", err)
	}
	noteText := h.Client.Messages[len(h.Client.Messages)-1].Text
	if !strings.Contains(noteText, "@filteruser") && !strings.Contains(noteText, "Read No spam here") {
		t.Fatalf("expected note fillings or random content to resolve, got %q", noteText)
	}
}

func TestStopAllRequiresCreatorOrBotOwner(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100734, Type: "supergroup", Title: "StopAll"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:              telegram.User{ID: 2, FirstName: "Admin"},
			Status:            "administrator",
			CanDeleteMessages: true,
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/filter hello Hi",
		},
	}); err != nil {
		t.Fatalf("setup filter failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 2, FirstName: "Admin"},
			Chat:      chat,
			Text:      "/stopall",
		},
	}); err == nil || !strings.Contains(err.Error(), "chat creator rights required") {
		t.Fatalf("expected non-creator stopall to fail, got %v", err)
	}

	filters, err := h.Store.ListFilters(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list filters failed: %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("expected filter to remain after denied stopall, got %+v", filters)
	}
}

func TestFilterRoseStyleExamplesThatAreLive(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100735, Type: "supergroup", Title: "Rose Filters"}

	commands := []string{
		"/filter (multi1, multi2, \"multi three\") Multi hello",
		"/filter \"exact:hi\" Exact hello",
		"/filter \"prefix:go\" Prefix hello",
		"/filter adminonly Admin hello {admin}",
		"/filter useronly User hello {user}",
		"/filter raw Hello {first}",
		"/filter magic Watch out {first}! {replytag}",
		"/filter secret Hidden {protect} {nonotif}",
	}
	for idx, cmd := range commands {
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

	filters, err := h.Store.ListFilters(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list filters failed: %v", err)
	}
	if len(filters) != 10 {
		t.Fatalf("expected multi-trigger command to create 10 filters total, got %d: %+v", len(filters), filters)
	}

	testCases := []struct {
		text        string
		user        telegram.User
		replyTo     *telegram.Message
		want        string
		wantReply   int64
		wantSilent  bool
		wantProtect bool
	}{
		{text: "multi three", user: telegram.User{ID: 20, FirstName: "Member"}, want: "Multi hello"},
		{text: "say hi", user: telegram.User{ID: 20, FirstName: "Member"}, want: ""},
		{text: "hi", user: telegram.User{ID: 20, FirstName: "Member"}, want: "Exact hello"},
		{text: "say go", user: telegram.User{ID: 20, FirstName: "Member"}, want: ""},
		{text: "go now", user: telegram.User{ID: 20, FirstName: "Member"}, want: "Prefix hello"},
		{text: "adminonly", user: telegram.User{ID: 20, FirstName: "Member"}, want: ""},
		{text: "adminonly", user: telegram.User{ID: 1, FirstName: "Owner"}, want: "Admin hello"},
		{text: "useronly", user: telegram.User{ID: 1, FirstName: "Owner"}, want: ""},
		{text: "useronly force", user: telegram.User{ID: 1, FirstName: "Owner"}, want: "User hello"},
		{text: "raw noformat", user: telegram.User{ID: 20, FirstName: "Member"}, want: "Hello {first}"},
		{text: "magic", user: telegram.User{ID: 20, FirstName: "Member"}, replyTo: &telegram.Message{MessageID: 500, From: &telegram.User{ID: 21, FirstName: "Target"}, Chat: chat}, want: "Watch out Target!"},
		{text: "secret", user: telegram.User{ID: 20, FirstName: "Member"}, want: "Hidden", wantSilent: true, wantProtect: true},
	}

	for idx, tc := range testCases {
		before := len(h.Client.Messages)
		messageID := int64(100 + idx)
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(100 + idx),
			Message: &telegram.Message{
				MessageID:      messageID,
				From:           &tc.user,
				Chat:           chat,
				Text:           tc.text,
				ReplyToMessage: tc.replyTo,
			},
		}); err != nil {
			t.Fatalf("filter trigger %q failed: %v", tc.text, err)
		}
		if tc.want == "" {
			if len(h.Client.Messages) != before {
				t.Fatalf("expected %q not to trigger, got new message %+v", tc.text, h.Client.Messages[len(h.Client.Messages)-1])
			}
			continue
		}
		if len(h.Client.Messages) != before+1 {
			t.Fatalf("expected %q to trigger exactly once, got %d new messages", tc.text, len(h.Client.Messages)-before)
		}
		got := h.Client.Messages[len(h.Client.Messages)-1]
		if got.Text != tc.want {
			t.Fatalf("expected %q to render %q, got %q", tc.text, tc.want, got.Text)
		}
		if got.Options.ReplyToMessageID != messageID {
			t.Fatalf("expected filter reply to message %d, got %+v", messageID, got.Options)
		}
		if got.Options.DisableNotification != tc.wantSilent || got.Options.ProtectContent != tc.wantProtect {
			t.Fatalf("unexpected filter send options for %q: %+v", tc.text, got.Options)
		}
	}
}
