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
	if !strings.Contains(startMessage.Text, "Hey there! My name is Sukoon") {
		t.Fatalf("expected /start response to introduce Sukoon, got %q", startMessage.Text)
	}
	if !strings.Contains(startMessage.Text, "Use /help") || !strings.Contains(startMessage.Text, "/privacy") {
		t.Fatalf("expected /start response to guide help and privacy, got %q", startMessage.Text)
	}
	if !strings.Contains(startMessage.Text, "https://t.me/VivaanUpdates") {
		t.Fatalf("expected /start response to include clickable support channel link, got %q", startMessage.Text)
	}
	if startMessage.Options.ReplyToMessageID != 10 {
		t.Fatalf("expected /start response to reply to message 10, got %+v", startMessage.Options)
	}
	if len(h.Client.AdminLookups) != 0 {
		t.Fatalf("expected private /start to skip admin lookups, got %+v", h.Client.AdminLookups)
	}
	if startMessage.Options.ParseMode != "HTML" {
		t.Fatalf("expected /start response to use HTML parse mode, got %+v", startMessage.Options)
	}
	if !startMessage.Options.DisableWebPagePreview {
		t.Fatalf("expected /start response to disable web page preview, got %+v", startMessage.Options)
	}
	startMarkup := requireMarkup(t, startMessage)
	assertButton(t, startMarkup, 0, 0, "Add me to your chat!", "", serviceutil.BotAddGroupLink(h.Bot.Username))
	assertButton(t, startMarkup, 0, 1, "Get your own Sukoon", "ux:start:clone", "")
	assertNoButtonText(t, startMarkup, "Help")
	assertNoButtonText(t, startMarkup, "Admin")
	assertNoButtonText(t, startMarkup, "Close")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 50, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	if len(h.Client.Messages) != 2 {
		t.Fatalf("expected /help to send one additional message, got %+v", h.Client.Messages)
	}
	helpMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(helpMessage.Text, "Sukoon Help") {
		t.Fatalf("expected help landing page, got %q", helpMessage.Text)
	}
	if !strings.Contains(helpMessage.Text, "/donate: Gives you info on how to support me and my creator.") {
		t.Fatalf("expected updated helpful commands copy, got %q", helpMessage.Text)
	}
	if !strings.Contains(helpMessage.Text, "https://t.me/VivaanUpdates") || !strings.Contains(helpMessage.Text, serviceutil.WebsiteURL) {
		t.Fatalf("expected help landing page to link support and website, got %q", helpMessage.Text)
	}
	if helpMessage.Options.ReplyToMessageID != 11 {
		t.Fatalf("expected /help response to reply to message 11, got %+v", helpMessage.Options)
	}
	if helpMessage.Options.ParseMode != "HTML" {
		t.Fatalf("expected /help response to use HTML parse mode, got %+v", helpMessage.Options)
	}
	if !helpMessage.Options.DisableWebPagePreview {
		t.Fatalf("expected /help response to disable web previews, got %+v", helpMessage.Options)
	}
	helpMarkup := requireMarkup(t, helpMessage)
	assertButton(t, helpMarkup, 0, 0, "Admin", "ux:help:admin", "")
	assertButton(t, helpMarkup, 0, 1, "Antiflood", "ux:help:antiflood", "")
	assertButton(t, helpMarkup, 0, 2, "AntiRaid", "ux:help:antiraid", "")
	assertButton(t, helpMarkup, 1, 0, "Approval", "ux:help:approval", "")
	assertButton(t, helpMarkup, 1, 1, "Bans", "ux:help:bans", "")
	assertButton(t, helpMarkup, 1, 2, "Blocklists", "ux:help:blocklists", "")
	assertButton(t, helpMarkup, 2, 0, "CAPTCHA", "ux:help:captcha", "")
	assertButton(t, helpMarkup, 2, 1, "Clean Commands", "ux:help:cleancommands", "")
	assertButton(t, helpMarkup, 2, 2, "Clean Service", "ux:help:cleanservice", "")
	assertButton(t, helpMarkup, 3, 0, "Connections", "ux:help:connections", "")
	assertButton(t, helpMarkup, 3, 1, "Disabling", "ux:help:disabling", "")
	assertButton(t, helpMarkup, 3, 2, "Locks", "ux:help:locks", "")
	assertButton(t, helpMarkup, 4, 0, "Federations", "ux:help:federations", "")
	assertButton(t, helpMarkup, 4, 1, "Filters", "ux:help:filters", "")
	assertButton(t, helpMarkup, 4, 2, "Formatting", "ux:help:formatting", "")
	assertButton(t, helpMarkup, 5, 0, "Greetings", "ux:help:greetings", "")
	assertButton(t, helpMarkup, 5, 1, "Import/Export", "ux:help:importexport", "")
	assertButton(t, helpMarkup, 5, 2, "Languages", "ux:help:languages", "")
	assertButton(t, helpMarkup, 6, 0, "Log Channels", "ux:help:logchannels", "")
	assertButton(t, helpMarkup, 6, 1, "Misc", "ux:help:misc", "")
	assertButton(t, helpMarkup, 6, 2, "Notes", "ux:help:notes", "")
	assertButton(t, helpMarkup, 7, 0, "Pin", "ux:help:pin", "")
	assertButton(t, helpMarkup, 7, 1, "Privacy", "ux:help:privacy", "")
	assertButton(t, helpMarkup, 7, 2, "Purges", "ux:help:purges", "")
	assertButton(t, helpMarkup, 8, 0, "Reports", "ux:help:reports", "")
	assertButton(t, helpMarkup, 8, 1, "Rules", "ux:help:rules", "")
	assertButton(t, helpMarkup, 8, 2, "Topics", "ux:help:topics", "")
	assertButton(t, helpMarkup, 9, 0, "Warnings", "ux:help:warnings", "")
	assertButton(t, helpMarkup, 9, 1, "AntiAbuse", "ux:help:antiabuse", "")
	assertButton(t, helpMarkup, 9, 2, "Bio Links", "ux:help:biolinks", "")
	assertButton(t, helpMarkup, 10, 0, "⭐ Custom Instances", "ux:help:custominstances", "")
	assertButton(t, helpMarkup, 10, 1, "🔎 Docs Website", "", serviceutil.WebsiteURL)
	assertNoButtonText(t, helpMarkup, "Home")
	assertNoButtonText(t, helpMarkup, "Close")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-blocklists",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: helpMessage.MessageID,
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
	assertNoButtonText(t, sectionMarkup, "Home")
	assertNoButtonText(t, sectionMarkup, "Close")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 4,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-blocklists-examples",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: helpMessage.MessageID,
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
	assertNoButtonText(t, examplesMarkup, "Home")
	assertNoButtonText(t, examplesMarkup, "Close")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 5,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-blocklists-back",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: helpMessage.MessageID,
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

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 6,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-home",
			From: telegram.User{ID: 50, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: helpMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:start:home",
		},
	}); err != nil {
		t.Fatalf("home callback failed: %v", err)
	}

	homeMessage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(homeMessage.Text, "Hey there! My name is Sukoon") {
		t.Fatalf("expected home callback to restore the start page, got %q", homeMessage.Text)
	}
	if homeMessage.Options.ParseMode != "HTML" {
		t.Fatalf("expected home callback to restore start page with HTML parse mode, got %+v", homeMessage.Options)
	}
	if !homeMessage.Options.DisableWebPagePreview {
		t.Fatalf("expected home callback to disable web page preview, got %+v", homeMessage.Options)
	}
	homeMarkup := requireEditedMarkup(t, homeMessage)
	assertButton(t, homeMarkup, 0, 0, "Add me to your chat!", "", serviceutil.BotAddGroupLink(h.Bot.Username))
	assertButton(t, homeMarkup, 0, 1, "Get your own Sukoon", "ux:start:clone", "")

	if len(h.Client.CallbackAnswers) != 4 {
		t.Fatalf("expected callback answers for menu navigation, got %+v", h.Client.CallbackAnswers)
	}
}

func TestDonateCommandSendsSupportImage(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 53, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 40,
			From:      &telegram.User{ID: 53, FirstName: "User"},
			Chat:      chat,
			Text:      "/donate",
		},
	}); err != nil {
		t.Fatalf("donate failed: %v", err)
	}

	if len(h.Client.Photos) != 1 {
		t.Fatalf("expected one donate photo, got %+v", h.Client.Photos)
	}
	photo := h.Client.Photos[0]
	if photo.ChatID != chat.ID {
		t.Fatalf("expected donate photo in chat %d, got %+v", chat.ID, photo)
	}
	if photo.Photo != "https://files.catbox.moe/25hv2j.jpg" {
		t.Fatalf("expected donate photo URL, got %+v", photo)
	}
	if photo.Options.Caption != "Hey, thanks for wanting to donate! Sukoon is entirely run by volunteers, so this means a lot.\nWe accept only UPI as donation method." {
		t.Fatalf("expected donate caption, got %+v", photo.Options)
	}
	if photo.Options.ReplyToMessageID != 40 {
		t.Fatalf("expected donate photo to reply to the command message, got %+v", photo.Options)
	}
}

func TestStartCloneGuideUsesInPlaceCallbackUX(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 52, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 30,
			From:      &telegram.User{ID: 52, FirstName: "User"},
			Chat:      chat,
			Text:      "/start",
		},
	}); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	startMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-start-clone",
			From: telegram.User{ID: 52, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:start:clone",
		},
	}); err != nil {
		t.Fatalf("clone callback failed: %v", err)
	}

	if len(h.Client.Messages) != 1 {
		t.Fatalf("expected clone guide callback to edit in place, got %+v", h.Client.Messages)
	}
	cloneGuide := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if cloneGuide.MessageID != startMessage.MessageID {
		t.Fatalf("expected clone guide to edit the start message, got %+v", cloneGuide)
	}
	if !strings.Contains(cloneGuide.Text, "Get your own Sukoon") || !strings.Contains(cloneGuide.Text, "/clone <bot_token>") {
		t.Fatalf("expected clone guide instructions, got %q", cloneGuide.Text)
	}
	if !strings.Contains(cloneGuide.Text, "@BotFather") || strings.Contains(cloneGuide.Text, "PUBLIC_WEBHOOK_BASE_URL") {
		t.Fatalf("expected BotFather mention without server-internal webhook note, got %q", cloneGuide.Text)
	}
	if strings.Contains(cloneGuide.Text, "/clone sync <clone>") || strings.Contains(cloneGuide.Text, "/clones") {
		t.Fatalf("expected clone guide to avoid operator-heavy commands, got %q", cloneGuide.Text)
	}
	cloneMarkup := requireEditedMarkup(t, cloneGuide)
	assertButton(t, cloneMarkup, 0, 0, "Back", "ux:start:home", "")
	assertButton(t, cloneMarkup, 0, 1, "Close", "ux:close", "")
	assertNoButtonText(t, cloneMarkup, "Open BotFather")
	assertNoButtonText(t, cloneMarkup, "Website")
	assertNoButtonText(t, cloneMarkup, "Help")
	assertNoButtonText(t, cloneMarkup, "Add to Group")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-start-home",
			From: telegram.User{ID: 52, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: startMessage.MessageID,
				Chat:      chat,
			},
			Data: "ux:start:home",
		},
	}); err != nil {
		t.Fatalf("start home callback failed: %v", err)
	}

	home := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(home.Text, "Hey there! My name is Sukoon") {
		t.Fatalf("expected back to restore the start landing page, got %q", home.Text)
	}
	if len(h.Client.CallbackAnswers) != 2 {
		t.Fatalf("expected callback answers for clone guide navigation, got %+v", h.Client.CallbackAnswers)
	}
}

func TestHelpNavigationSupportsNestedHelpBatchPages(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 51, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 20,
			From:      &telegram.User{ID: 51, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	sequence := []struct {
		updateID int64
		data     string
		want     string
	}{
		{2, "ux:help:federations", "Federations"},
		{3, "ux:help:federations_owner", "Federation Owner Commands"},
		{4, "ux:help:federations", "Federations"},
		{5, "ux:help:filters", "Filters"},
		{6, "ux:help:filters_examples", "Filter Example Usage"},
		{7, "ux:help:filters", "Filters"},
		{8, "ux:help:formatting", "Formatting"},
		{9, "ux:help:formatting_markdown", "Markdown Formatting"},
		{10, "ux:help:formatting_buttons", "Buttons"},
		{11, "ux:help:formatting", "Formatting"},
		{12, "ux:help:disabling", "Disabling"},
		{13, "ux:help:connections", "Connections"},
		{14, "ux:help:antiraid", "AntiRaid"},
		{15, "ux:help:antiabuse", "AntiAbuse"},
		{16, "ux:help:biolinks", "Bio Links"},
		{17, "ux:help:custominstances", "Custom Instances"},
	}
	for _, step := range sequence {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: step.updateID,
			CallbackQuery: &telegram.CallbackQuery{
				ID:   step.data,
				From: telegram.User{ID: 51, FirstName: "User"},
				Message: &telegram.Message{
					MessageID: root.MessageID,
					Chat:      chat,
				},
				Data: step.data,
			},
		}); err != nil {
			t.Fatalf("callback %q failed: %v", step.data, err)
		}
		last := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
		if last.MessageID != root.MessageID {
			t.Fatalf("expected in-place edit for %q, got %+v", step.data, last)
		}
		if !strings.Contains(last.Text, step.want) {
			t.Fatalf("expected page %q to contain %q, got %q", step.data, step.want, last.Text)
		}
	}

	if !strings.Contains(h.Client.EditedMessages[1].Text, "/renamefed") || !strings.Contains(h.Client.EditedMessages[1].Text, "/fedtransfer") {
		t.Fatalf("expected federation owner page to list live owner commands, got %q", h.Client.EditedMessages[1].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[4].Text, "/filter \"buy now\"") || !strings.Contains(h.Client.EditedMessages[4].Text, "/stop hello | buy now") {
		t.Fatalf("expected filter examples page, got %q", h.Client.EditedMessages[4].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[7].Text, "full markdown helper set") {
		t.Fatalf("expected truthful markdown page, got %q", h.Client.EditedMessages[7].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[8].Text, "buttonurl:https://misssukoon.vercel.app/") {
		t.Fatalf("expected buttons page to show live syntax, got %q", h.Client.EditedMessages[8].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[10].Text, "/disabledel [on|off]") {
		t.Fatalf("expected disabling page, got %q", h.Client.EditedMessages[10].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[11].Text, "does not expose remote chat connections") {
		t.Fatalf("expected truthful connections placeholder, got %q", h.Client.EditedMessages[11].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[12].Text, "does not expose dedicated antiraid commands yet") {
		t.Fatalf("expected truthful antiraid placeholder, got %q", h.Client.EditedMessages[12].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[13].Text, "/antiabuse <on|off>") {
		t.Fatalf("expected antiabuse page, got %q", h.Client.EditedMessages[13].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[14].Text, "/freelist") || !strings.Contains(h.Client.EditedMessages[14].Text, "Approved users and freed users bypass") {
		t.Fatalf("expected bio links page, got %q", h.Client.EditedMessages[14].Text)
	}
	if !strings.Contains(h.Client.EditedMessages[15].Text, "/mybot") || !strings.Contains(h.Client.EditedMessages[15].Text, "one active clone") {
		t.Fatalf("expected custom instances page, got %q", h.Client.EditedMessages[15].Text)
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
