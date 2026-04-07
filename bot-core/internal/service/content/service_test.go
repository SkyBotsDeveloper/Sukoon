package content_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"sukoon/bot-core/internal/serviceutil"
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

func TestRulesSurfaceUsesButtonsInGroupAndCallbackFlow(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100501, Type: "supergroup", Title: "Rules"}

	if err := h.Store.SetRules(context.Background(), h.Bot.ID, chat.ID, "Be respectful."); err != nil {
		t.Fatalf("set rules failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 20,
			From:      &telegram.User{ID: 30, FirstName: "Member"},
			Chat:      chat,
			Text:      "/rules",
		},
	}); err != nil {
		t.Fatalf("rules command failed: %v", err)
	}

	groupPrompt := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(groupPrompt.Text, "Open them in PM") {
		t.Fatalf("expected group rules guidance, got %q", groupPrompt.Text)
	}
	groupMarkup := requireRulesMarkup(t, groupPrompt)
	assertRulesButton(t, groupMarkup, 0, 0, "Open PM", "", serviceutil.BotDeepLink(h.Bot.Username, "rules_-100501"))
	assertRulesButton(t, groupMarkup, 0, 1, "Show Here", "ux:rules:show", "")
	assertRulesButton(t, groupMarkup, 1, 0, "Help", "", serviceutil.BotDeepLink(h.Bot.Username, "help_main"))
	assertRulesButton(t, groupMarkup, 1, 1, "Website", "", serviceutil.WebsiteURL)

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-rules-show",
			From: telegram.User{ID: 30, FirstName: "Member"},
			Message: &telegram.Message{
				MessageID: groupPrompt.MessageID,
				Chat:      chat,
			},
			Data: "ux:rules:show",
		},
	}); err != nil {
		t.Fatalf("rules callback failed: %v", err)
	}

	if len(h.Client.EditedMessages) == 0 {
		t.Fatalf("expected rules callback to edit in place, got %+v", h.Client.EditedMessages)
	}
	shownRules := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if shownRules.MessageID != groupPrompt.MessageID {
		t.Fatalf("expected rules callback to edit message %d, got %+v", groupPrompt.MessageID, shownRules)
	}
	if !strings.Contains(shownRules.Text, "Rules for Rules") || !strings.Contains(shownRules.Text, "Be respectful.") {
		t.Fatalf("expected shown rules message, got %q", shownRules.Text)
	}
	shownMarkup := requireEditedRulesMarkup(t, shownRules)
	assertRulesButton(t, shownMarkup, 0, 0, "Open PM", "", serviceutil.BotDeepLink(h.Bot.Username, "rules_-100501"))
	assertRulesButton(t, shownMarkup, 0, 1, "Help", "", serviceutil.BotDeepLink(h.Bot.Username, "help_main"))
	assertRulesButton(t, shownMarkup, 1, 0, "Website", "", serviceutil.WebsiteURL)
	assertRulesButton(t, shownMarkup, 1, 1, "Close", "ux:close", "")
}

func requireRulesMarkup(t *testing.T, msg testsupport.SentMessage) *telegram.InlineKeyboardMarkup {
	t.Helper()
	if msg.Options.ReplyMarkup == nil {
		t.Fatalf("expected rules reply markup, got %+v", msg.Options)
	}
	return msg.Options.ReplyMarkup
}

func requireEditedRulesMarkup(t *testing.T, msg testsupport.EditedMessage) *telegram.InlineKeyboardMarkup {
	t.Helper()
	if msg.Options.ReplyMarkup == nil {
		t.Fatalf("expected rules reply markup, got %+v", msg.Options)
	}
	return msg.Options.ReplyMarkup
}

func assertRulesButton(t *testing.T, markup *telegram.InlineKeyboardMarkup, row int, col int, text string, callbackData string, url string) {
	t.Helper()
	if len(markup.InlineKeyboard) <= row || len(markup.InlineKeyboard[row]) <= col {
		t.Fatalf("expected button at row %d col %d, got %+v", row, col, markup.InlineKeyboard)
	}
	button := markup.InlineKeyboard[row][col]
	if button.Text != text || button.CallbackData != callbackData || button.URL != url {
		t.Fatalf("unexpected button at row %d col %d: got %+v", row, col, button)
	}
}
