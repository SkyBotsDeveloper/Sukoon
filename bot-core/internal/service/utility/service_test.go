package utility_test

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

func TestLanguageSelectionUpdatesChatLanguage(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100760, Type: "supergroup", Title: "Lang"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/setlang hi",
		},
	}); err != nil {
		t.Fatalf("setlang failed: %v", err)
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if bundle.Settings.Language != "hi" {
		t.Fatalf("expected chat language hi, got %s", bundle.Settings.Language)
	}
}

func TestPrivacyExportAndDelete(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	groupChat := telegram.Chat{ID: -100761, Type: "supergroup", Title: "Privacy"}
	privateChat := telegram.Chat{ID: 20, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      groupChat,
			Text:      "/afk busy",
		},
	}); err != nil {
		t.Fatalf("set afk failed: %v", err)
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      privateChat,
			Text:      "/mydata",
		},
	}); err != nil {
		t.Fatalf("mydata failed: %v", err)
	}
	if !strings.Contains(h.Client.Messages[len(h.Client.Messages)-1].Text, "busy") {
		t.Fatalf("expected privacy export to include AFK payload, got %q", h.Client.Messages[len(h.Client.Messages)-1].Text)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      privateChat,
			Text:      "/forgetme confirm",
		},
	}); err != nil {
		t.Fatalf("forgetme failed: %v", err)
	}

	state, err := h.Store.GetAFK(context.Background(), h.Bot.ID, 20)
	if err != nil {
		t.Fatalf("get afk failed: %v", err)
	}
	if state.UserID != 0 {
		t.Fatalf("expected AFK state to be removed, got %+v", state)
	}
}

func TestStartAndHelpCommandsRenderPolishedUX(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 50, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 50, FirstName: "User"},
			Chat:      chat,
			Text:      "/start",
		},
	}); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	startMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(startMessage.Text, "Fast moderation") {
		t.Fatalf("expected /start response to introduce Sukoon, got %q", startMessage.Text)
	}
	if startMessage.Options.ReplyToMessageID != 10 {
		t.Fatalf("expected /start response to reply to message 10, got %+v", startMessage.Options)
	}
	startMarkup := requireMarkup(t, startMessage)
	assertButton(t, startMarkup, 0, 0, "Help", "ux:help:main", "")
	assertButton(t, startMarkup, 0, 1, "Website", "", serviceutil.WebsiteURL)
	assertButton(t, startMarkup, 1, 0, "Add to Group", "", serviceutil.BotAddGroupLink(h.Bot.Username))
	assertButton(t, startMarkup, 1, 1, "Privacy", "ux:privacy", "")
	assertButton(t, startMarkup, 2, 0, "Close", "ux:close", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-home",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:main",
		},
	}); err != nil {
		t.Fatalf("help callback failed: %v", err)
	}

	helpMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(helpMessage.Text, "Sukoon Help") {
		t.Fatalf("expected help landing page, got %q", helpMessage.Text)
	}
	if len(h.Client.DeletedMessages) == 0 || h.Client.DeletedMessages[0].MessageID != startMessage.MessageID {
		t.Fatalf("expected callback navigation to delete the previous menu, got %+v", h.Client.DeletedMessages)
	}
	helpMarkup := requireMarkup(t, helpMessage)
	assertButton(t, helpMarkup, 0, 0, "Moderation", "ux:help:moderation", "")
	assertButton(t, helpMarkup, 0, 1, "Admin", "ux:help:admin", "")
	assertButton(t, helpMarkup, 4, 0, "Back", "ux:start:home", "")
	assertButton(t, helpMarkup, 4, 1, "Close", "ux:close", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-moderation",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: helpMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:moderation",
		},
	}); err != nil {
		t.Fatalf("help moderation callback failed: %v", err)
	}

	moderationMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(moderationMessage.Text, "/ban") || !strings.Contains(moderationMessage.Text, "/warn") {
		t.Fatalf("expected moderation help page, got %q", moderationMessage.Text)
	}
	moderationMarkup := requireMarkup(t, moderationMessage)
	assertButton(t, moderationMarkup, 0, 0, "Back", "ux:help:main", "")
	assertButton(t, moderationMarkup, 0, 1, "Close", "ux:close", "")

	messageCount := len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 4,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-close",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: moderationMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:close",
		},
	}); err != nil {
		t.Fatalf("close callback failed: %v", err)
	}
	if len(h.Client.Messages) != messageCount {
		t.Fatalf("expected close callback to avoid sending a new message, got %+v", h.Client.Messages)
	}
	if len(h.Client.CallbackAnswers) != 3 {
		t.Fatalf("expected callback answers for menu navigation, got %+v", h.Client.CallbackAnswers)
	}
}

func TestGroupPMGuidanceUsesButtonsForHelpAndPrivacy(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	group := telegram.Chat{ID: -100990, Type: "supergroup", Title: "Help"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      group,
			Text:      "/help admin",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	helpMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(helpMessage.Text, "easier to browse in PM") {
		t.Fatalf("expected PM guidance for group help, got %q", helpMessage.Text)
	}
	if helpMessage.Options.ReplyToMessageID != 11 {
		t.Fatalf("expected help guidance to reply to message 11, got %+v", helpMessage.Options)
	}
	helpMarkup := requireMarkup(t, helpMessage)
	assertButton(t, helpMarkup, 0, 0, "Open PM", "", serviceutil.BotURL(h.Bot.Username))
	assertButton(t, helpMarkup, 0, 1, "Help", "", serviceutil.BotDeepLink(h.Bot.Username, "help_admin"))
	assertButton(t, helpMarkup, 1, 0, "Website", "", serviceutil.WebsiteURL)
	assertButton(t, helpMarkup, 1, 1, "Close", "ux:close", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      group,
			Text:      "/mydata",
		},
	}); err != nil {
		t.Fatalf("mydata guidance failed: %v", err)
	}

	myDataMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(myDataMessage.Text, "Use /mydata in PM") {
		t.Fatalf("expected PM guidance for /mydata, got %q", myDataMessage.Text)
	}
	myDataMarkup := requireMarkup(t, myDataMessage)
	assertButton(t, myDataMarkup, 0, 1, "Help", "", serviceutil.BotDeepLink(h.Bot.Username, "privacy"))
}

func requireMarkup(t *testing.T, msg testsupport.SentMessage) *telegram.InlineKeyboardMarkup {
	t.Helper()
	if msg.Options.ReplyMarkup == nil {
		t.Fatalf("expected inline keyboard markup, got %+v", msg.Options)
	}
	return msg.Options.ReplyMarkup
}

func assertButton(t *testing.T, markup *telegram.InlineKeyboardMarkup, row int, col int, text string, callbackData string, url string) {
	t.Helper()
	if len(markup.InlineKeyboard) <= row || len(markup.InlineKeyboard[row]) <= col {
		t.Fatalf("expected button at row %d col %d, got %+v", row, col, markup.InlineKeyboard)
	}
	button := markup.InlineKeyboard[row][col]
	if button.Text != text || button.CallbackData != callbackData || button.URL != url {
		t.Fatalf("unexpected button at row %d col %d: got %+v", row, col, button)
	}
}
