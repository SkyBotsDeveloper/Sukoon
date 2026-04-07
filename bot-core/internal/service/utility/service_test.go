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
	assertButton(t, startMarkup, 0, 0, "Help", "ux:help:root", "")
	assertButton(t, startMarkup, 0, 1, "Website", "", serviceutil.WebsiteURL)
	assertButton(t, startMarkup, 1, 0, "Admin", "ux:help:admin", "")
	assertButton(t, startMarkup, 1, 1, "Bans", "ux:help:bans", "")
	assertButton(t, startMarkup, 2, 0, "Add to Group", "", serviceutil.BotAddGroupLink(h.Bot.Username))
	assertButton(t, startMarkup, 2, 1, "Privacy", "ux:privacy", "")
	assertButton(t, startMarkup, 3, 0, "Close", "ux:close", "")

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

	if len(h.Client.Messages) != 1 {
		t.Fatalf("expected callback navigation to edit in place, got messages %+v", h.Client.Messages)
	}
	if len(h.Client.EditedMessages) == 0 {
		t.Fatalf("expected edited help message, got %+v", h.Client.EditedMessages)
	}
	helpMessage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(helpMessage.Text, "Sukoon Help") {
		t.Fatalf("expected help landing page, got %q", helpMessage.Text)
	}
	if helpMessage.MessageID != startMessage.MessageID {
		t.Fatalf("expected in-place help edit of message %d, got %+v", startMessage.MessageID, helpMessage)
	}
	helpMarkup := requireEditedMarkup(t, helpMessage)
	assertButton(t, helpMarkup, 0, 0, "Admin", "ux:help:admin", "")
	assertButton(t, helpMarkup, 0, 1, "Approval", "ux:help:approval", "")
	assertButton(t, helpMarkup, 1, 0, "Bans", "ux:help:bans", "")
	assertButton(t, helpMarkup, 1, 1, "Antiflood", "ux:help:antiflood", "")
	assertButton(t, helpMarkup, 2, 0, "Blocklists", "ux:help:blocklists", "")
	assertButton(t, helpMarkup, 2, 1, "CAPTCHA", "ux:help:captcha", "")
	assertButton(t, helpMarkup, 3, 0, "Clean Commands", "ux:help:cleancommands", "")
	assertButton(t, helpMarkup, 3, 1, "Locks", "ux:help:locks", "")
	assertButton(t, helpMarkup, 4, 0, "Log Channels", "ux:help:logchannels", "")
	assertButton(t, helpMarkup, 6, 0, "Home", "ux:start:home", "")
	assertButton(t, helpMarkup, 6, 1, "Close", "ux:close", "")
	assertNoButtonText(t, helpMarkup, "AntiRaid")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-blocklists",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:blocklists",
		},
	}); err != nil {
		t.Fatalf("help blocklists callback failed: %v", err)
	}

	sectionMessage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(sectionMessage.Text, "/addblocklist") || !strings.Contains(sectionMessage.Text, "/unblocklistall") {
		t.Fatalf("expected blocklist help page, got %q", sectionMessage.Text)
	}
	sectionMarkup := requireEditedMarkup(t, sectionMessage)
	assertButton(t, sectionMarkup, 0, 0, "Examples", "ux:help:blocklists_examples", "")
	assertButton(t, sectionMarkup, 1, 0, "Back", "ux:help:root", "")
	assertButton(t, sectionMarkup, 1, 1, "Home", "ux:start:home", "")
	assertButton(t, sectionMarkup, 3, 0, "Close", "ux:close", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 4,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-blocklists-examples",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:blocklists_examples",
		},
	}); err != nil {
		t.Fatalf("help blocklist examples callback failed: %v", err)
	}

	examplesMessage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(examplesMessage.Text, "/addblocklist regex") || !strings.Contains(examplesMessage.Text, "/rmblocklist spam | buy now") {
		t.Fatalf("expected blocklist examples help page, got %q", examplesMessage.Text)
	}
	examplesMarkup := requireEditedMarkup(t, examplesMessage)
	assertButton(t, examplesMarkup, 0, 0, "Back", "ux:help:blocklists", "")
	assertButton(t, examplesMarkup, 0, 1, "Home", "ux:start:home", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 5,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-blocklists-back",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:blocklists",
		},
	}); err != nil {
		t.Fatalf("blocklists back callback failed: %v", err)
	}

	backMessage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(backMessage.Text, "Blocklists") || !strings.Contains(backMessage.Text, "/unblocklistall") {
		t.Fatalf("expected blocklists page after back, got %q", backMessage.Text)
	}

	messageCount := len(h.Client.Messages)
	editCount := len(h.Client.EditedMessages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 6,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-close",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
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
	if len(h.Client.EditedMessages) != editCount {
		t.Fatalf("expected close callback to avoid extra edits, got %+v", h.Client.EditedMessages)
	}
	if len(h.Client.DeletedMessages) != 1 || h.Client.DeletedMessages[0].MessageID != startMessage.MessageID {
		t.Fatalf("expected close callback to delete the menu message, got %+v", h.Client.DeletedMessages)
	}
	if len(h.Client.CallbackAnswers) != 5 {
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

func requireEditedMarkup(t *testing.T, msg testsupport.EditedMessage) *telegram.InlineKeyboardMarkup {
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

func assertNoButtonText(t *testing.T, markup *telegram.InlineKeyboardMarkup, text string) {
	t.Helper()
	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			if button.Text == text {
				t.Fatalf("unexpected button %q in markup %+v", text, markup.InlineKeyboard)
			}
		}
	}
}
