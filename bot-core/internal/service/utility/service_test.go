package utility_test

import (
	"context"
	"html"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

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
	startRendered := renderedText(startMessage.Text)
	if !strings.Contains(startRendered, "Hey there! My name is Sukoon") {
		t.Fatalf("expected /start response to introduce Sukoon, got %q", startMessage.Text)
	}
	if !strings.Contains(startRendered, "Use /help") || !strings.Contains(startRendered, "/privacy") {
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
	helpRendered := renderedText(helpMessage.Text)
	if !strings.Contains(helpRendered, "Sukoon Help") {
		t.Fatalf("expected help landing page, got %q", helpMessage.Text)
	}
	if !strings.Contains(helpRendered, "/donate: Gives you info on how to support me and my creator.") {
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
	assertButton(t, helpMarkup, 9, 1, "Silent Power", "ux:help:silentpower", "")
	assertButton(t, helpMarkup, 9, 2, "Extra", "ux:help:extra", "")
	assertButton(t, helpMarkup, 10, 0, "Bio Check", "ux:help:biocheck", "")
	assertButton(t, helpMarkup, 10, 1, "AntiAbuse", "ux:help:antiabuse", "")
	assertButton(t, helpMarkup, 11, 0, "⭐ Custom Instances", "ux:help:custominstances", "")
	assertButton(t, helpMarkup, 11, 1, "📚 Docs Website", "", serviceutil.WebsiteURL)
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
	sectionRendered := renderedText(sectionMessage.Text)
	if !strings.Contains(sectionRendered, "/blocklistmode <blocklist mode>") || !strings.Contains(sectionRendered, "/blocklistdelete <yes/no/on/off>") || !strings.Contains(sectionRendered, "/setblocklistreason <reason>") {
		t.Fatalf("expected blocklist help page, got %q", sectionMessage.Text)
	}
	sectionMarkup := requireEditedMarkup(t, sectionMessage)
	assertButton(t, sectionMarkup, 0, 0, "Blocklist Command Examples", "ux:help:blocklists_examples", "")
	assertButton(t, sectionMarkup, 1, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, sectionMarkup, "Website")
	assertNoButtonText(t, sectionMarkup, "Add to Group")
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
	examplesRendered := renderedText(examplesMessage.Text)
	if !strings.Contains(examplesRendered, "/blocklistmode warn") || !strings.Contains(examplesRendered, "\"bit.ly/???\"") || !strings.Contains(examplesRendered, "stickerpack:<>") {
		t.Fatalf("expected blocklist examples help page, got %q", examplesMessage.Text)
	}
	examplesMarkup := requireEditedMarkup(t, examplesMessage)
	assertButton(t, examplesMarkup, 0, 0, "Back", "ux:help:blocklists", "")
	if !examplesMessage.Options.DisableWebPagePreview {
		t.Fatalf("expected blocklist examples help page to disable web previews, got %+v", examplesMessage.Options)
	}
	assertNoButtonText(t, examplesMarkup, "Website")
	assertNoButtonText(t, examplesMarkup, "Add to Group")
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
	backRendered := renderedText(backMessage.Text)
	if !strings.Contains(backRendered, "Blocklists") || !strings.Contains(backRendered, "/blocklistmode <blocklist mode>") {
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
	homeRendered := renderedText(homeMessage.Text)
	if !strings.Contains(homeRendered, "Hey there! My name is Sukoon") {
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

func TestStartHelpCallbacksUseFastPathWithoutHeavyRuntimeLoads(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 501, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 100,
			From:      &telegram.User{ID: 501, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	if h.Store.LoadBundleCalls != 0 || h.Store.EnsureChatCalls != 0 || h.Store.EnsureUserCalls != 0 || h.Store.GetBotRolesCalls != 0 {
		t.Fatalf("expected /help fast path to skip heavy runtime work, got bundle=%d ensureChat=%d ensureUser=%d botRoles=%d",
			h.Store.LoadBundleCalls, h.Store.EnsureChatCalls, h.Store.EnsureUserCalls, h.Store.GetBotRolesCalls)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-fast-help-admin",
			From: telegram.User{ID: 501, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:admin",
		},
	}); err != nil {
		t.Fatalf("help callback failed: %v", err)
	}

	if h.Store.LoadBundleCalls != 0 || h.Store.EnsureChatCalls != 0 || h.Store.EnsureUserCalls != 0 || h.Store.GetBotRolesCalls != 0 {
		t.Fatalf("expected help callback fast path to skip heavy runtime work, got bundle=%d ensureChat=%d ensureUser=%d botRoles=%d",
			h.Store.LoadBundleCalls, h.Store.EnsureChatCalls, h.Store.EnsureUserCalls, h.Store.GetBotRolesCalls)
	}
	if len(h.Client.CallbackAnswers) != 1 || h.Client.CallbackAnswers[0].ID != "cb-fast-help-admin" {
		t.Fatalf("expected immediate callback ack, got %+v", h.Client.CallbackAnswers)
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
	cloneRendered := renderedText(cloneGuide.Text)
	if !strings.Contains(cloneRendered, "Get your own Sukoon") || !strings.Contains(cloneRendered, "/clone <bot_token>") {
		t.Fatalf("expected clone guide instructions, got %q", cloneGuide.Text)
	}
	if !strings.Contains(cloneRendered, "@BotFather") || strings.Contains(cloneRendered, "PUBLIC_WEBHOOK_BASE_URL") {
		t.Fatalf("expected BotFather mention without server-internal webhook note, got %q", cloneGuide.Text)
	}
	if strings.Contains(cloneRendered, "/clone sync <clone>") || strings.Contains(cloneRendered, "/clones") {
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
	if !strings.Contains(renderedText(home.Text), "Hey there! My name is Sukoon") {
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
		{15, "ux:help:silentpower", "Silent Power"},
		{16, "ux:help:extra", "Extra"},
		{17, "ux:help:antiabuse", "AntiAbuse"},
		{18, "ux:help:biocheck", "Bio Check"},
		{19, "ux:help:custominstances", "Custom Instances"},
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
		if !strings.Contains(renderedText(last.Text), step.want) {
			t.Fatalf("expected page %q to contain %q, got %q", step.data, step.want, last.Text)
		}
	}

	if !strings.Contains(renderedText(h.Client.EditedMessages[1].Text), "/renamefed") || !strings.Contains(renderedText(h.Client.EditedMessages[1].Text), "/fedtransfer") {
		t.Fatalf("expected federation owner page to list live owner commands, got %q", h.Client.EditedMessages[1].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[4].Text), "/filter \"buy now\"") || !strings.Contains(renderedText(h.Client.EditedMessages[4].Text), "/stop hello | buy now") {
		t.Fatalf("expected filter examples page, got %q", h.Client.EditedMessages[4].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[7].Text), "full markdown helper set") {
		t.Fatalf("expected truthful markdown page, got %q", h.Client.EditedMessages[7].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[8].Text), "buttonurl:https://misssukoon.vercel.app/") {
		t.Fatalf("expected buttons page to show live syntax, got %q", h.Client.EditedMessages[8].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[10].Text), "/disabledel [on|off]") {
		t.Fatalf("expected disabling page, got %q", h.Client.EditedMessages[10].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[11].Text), "does not expose remote chat connections") {
		t.Fatalf("expected truthful connections placeholder, got %q", h.Client.EditedMessages[11].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[12].Text), "/raidtime <time>") || !strings.Contains(renderedText(h.Client.EditedMessages[12].Text), "/autoantiraid <number/off/no>") {
		t.Fatalf("expected live antiraid help page, got %q", h.Client.EditedMessages[12].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[13].Text), "/mods") || !strings.Contains(renderedText(h.Client.EditedMessages[13].Text), "/muter") {
		t.Fatalf("expected silent power page, got %q", h.Client.EditedMessages[13].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[14].Text), "/donate") || !strings.Contains(renderedText(h.Client.EditedMessages[14].Text), "/mybot") {
		t.Fatalf("expected extra page, got %q", h.Client.EditedMessages[14].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[15].Text), "/antiabuse <on|off>") {
		t.Fatalf("expected antiabuse page, got %q", h.Client.EditedMessages[15].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[16].Text), "/freelist") || !strings.Contains(renderedText(h.Client.EditedMessages[16].Text), "Approved users and freed users bypass") {
		t.Fatalf("expected bio check page, got %q", h.Client.EditedMessages[16].Text)
	}
	if !strings.Contains(renderedText(h.Client.EditedMessages[17].Text), "/mybot") || !strings.Contains(renderedText(h.Client.EditedMessages[17].Text), "one active clone") {
		t.Fatalf("expected custom instances page, got %q", h.Client.EditedMessages[17].Text)
	}
}

func TestAdminHelpPageUsesBackOnlyMarkup(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 54, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 60,
			From:      &telegram.User{ID: 54, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-admin",
			From: telegram.User{ID: 54, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:admin",
		},
	}); err != nil {
		t.Fatalf("admin help callback failed: %v", err)
	}

	adminPage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	adminRendered := renderedText(adminPage.Text)
	if !strings.Contains(adminRendered, "/promote <reply/username/userid>") || !strings.Contains(adminRendered, "/anonadmin <yes/no/on/off>") || !strings.Contains(adminRendered, "/adminerror <yes/no/on/off>") {
		t.Fatalf("expected admin help page copy, got %q", adminPage.Text)
	}
	adminMarkup := requireEditedMarkup(t, adminPage)
	assertButton(t, adminMarkup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, adminMarkup, "Website")
	assertNoButtonText(t, adminMarkup, "Add to Group")
}

func TestAntifloodHelpPageUsesBackOnlyMarkup(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 55, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 61,
			From:      &telegram.User{ID: 55, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-antiflood",
			From: telegram.User{ID: 55, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:antiflood",
		},
	}); err != nil {
		t.Fatalf("antiflood help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "/setfloodtimer <count> <duration>") || !strings.Contains(pageRendered, "/clearflood <yes/no/on/off>") || !strings.Contains(pageRendered, "/floodmode tban 3d") {
		t.Fatalf("expected antiflood help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")
}

func TestApprovalHelpPageUsesBackOnlyMarkup(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 57, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 63,
			From:      &telegram.User{ID: 57, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-approval",
			From: telegram.User{ID: 57, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:approval",
		},
	}); err != nil {
		t.Fatalf("approval help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "User commands:") || !strings.Contains(pageRendered, "/approval") || !strings.Contains(pageRendered, "/unapproveall") {
		t.Fatalf("expected approval help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")
}

func TestCaptchaHelpPageUsesBackOnlyMarkup(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 59, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 65,
			From:      &telegram.User{ID: 59, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-captcha",
			From: telegram.User{ID: 59, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:captcha",
		},
	}); err != nil {
		t.Fatalf("captcha help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "/captcharules <yes/no/on/off>") || !strings.Contains(pageRendered, "/captchamutetime <Xw/d/h/m>") || !strings.Contains(pageRendered, "/setcaptchatext <text>") {
		t.Fatalf("expected captcha help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")
}

func TestBansHelpPageUsesBackOnlyMarkup(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 58, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 64,
			From:      &telegram.User{ID: 58, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-bans",
			From: telegram.User{ID: 58, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:bans",
		},
	}); err != nil {
		t.Fatalf("bans help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "/kickme") || !strings.Contains(pageRendered, "/dban") || !strings.Contains(pageRendered, "/tmute") || !strings.Contains(pageRendered, "4m = 4 minutes") {
		t.Fatalf("expected bans help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")
}

func TestAntiRaidHelpPageUsesBackOnlyMarkup(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 56, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 62,
			From:      &telegram.User{ID: 56, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-antiraid",
			From: telegram.User{ID: 56, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:antiraid",
		},
	}); err != nil {
		t.Fatalf("antiraid help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "/antiraid <optional time/off/no>") || !strings.Contains(pageRendered, "/raidactiontime <time>") || !strings.Contains(pageRendered, "-> /autoantiraid off") {
		t.Fatalf("expected antiraid help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")
}

func TestLocksHelpPageUsesExamplesDescriptionsAndBack(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 66, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 70,
			From:      &telegram.User{ID: 66, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-locks",
			From: telegram.User{ID: 66, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:locks",
		},
	}); err != nil {
		t.Fatalf("locks help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "/lockwarns <yes/no/on/off>") || !strings.Contains(pageRendered, "/allowlist <url/id/command/@username(s)>") || !strings.Contains(pageRendered, "/rmallowlistall") {
		t.Fatalf("expected locks help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Example Commands", "ux:help:locks_examples", "")
	assertButton(t, markup, 0, 1, "Lock descriptions", "ux:help:locks_descriptions", "")
	assertButton(t, markup, 1, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-locks-examples",
			From: telegram.User{ID: 66, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:locks_examples",
		},
	}); err != nil {
		t.Fatalf("locks examples callback failed: %v", err)
	}

	examples := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	examplesRendered := renderedText(examples.Text)
	if !strings.Contains(examplesRendered, "/lock invitelink ### no promoting other chats {ban}") || !strings.Contains(examplesRendered, "/allowlist t.me/addstickers/Pinup_Girl") {
		t.Fatalf("expected lock examples copy, got %q", examples.Text)
	}
	if !examples.Options.DisableWebPagePreview {
		t.Fatalf("expected lock examples page to disable previews, got %+v", examples.Options)
	}
	exampleMarkup := requireEditedMarkup(t, examples)
	assertButton(t, exampleMarkup, 0, 0, "Back", "ux:help:locks", "")
	assertNoButtonText(t, exampleMarkup, "Website")
	assertNoButtonText(t, exampleMarkup, "Add to Group")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 4,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-locks-descriptions",
			From: telegram.User{ID: 66, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:locks_descriptions",
		},
	}); err != nil {
		t.Fatalf("locks descriptions callback failed: %v", err)
	}

	descriptions := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	descriptionsRendered := renderedText(descriptions.Text)
	if !strings.Contains(descriptionsRendered, "- invitelink: Messages containing Telegram group or channel links.") || !strings.Contains(descriptionsRendered, "- sticker / stickeranimated / stickerpremium:") {
		t.Fatalf("expected lock descriptions copy, got %q", descriptions.Text)
	}
	descriptionMarkup := requireEditedMarkup(t, descriptions)
	assertButton(t, descriptionMarkup, 0, 0, "Back", "ux:help:locks", "")
	assertNoButtonText(t, descriptionMarkup, "Website")
	assertNoButtonText(t, descriptionMarkup, "Add to Group")
}

func TestCleanCommandsHelpPageLinksToContextualLocksAndLogChannels(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 67, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 71,
			From:      &telegram.User{ID: 67, FirstName: "User"},
			Chat:      chat,
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	root := h.Client.Messages[len(h.Client.Messages)-1]
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-cleancommands",
			From: telegram.User{ID: 67, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:help:cleancommands",
		},
	}); err != nil {
		t.Fatalf("cleancommands help callback failed: %v", err)
	}

	page := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	pageRendered := renderedText(page.Text)
	if !strings.Contains(pageRendered, "/cleancommand <type>") || !strings.Contains(pageRendered, "/keepcommand <type>") || !strings.Contains(pageRendered, "/cleancommand user other") {
		t.Fatalf("expected clean commands help copy, got %q", page.Text)
	}
	markup := requireEditedMarkup(t, page)
	assertButton(t, markup, 0, 0, "Locks", "ux:helpctx:cleancommands:locks", "")
	assertButton(t, markup, 0, 1, "Log Channels", "ux:helpctx:cleancommands:logchannels", "")
	assertButton(t, markup, 1, 0, "Back", "ux:help:root", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-cleancommands-locks",
			From: telegram.User{ID: 67, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:helpctx:cleancommands:locks",
		},
	}); err != nil {
		t.Fatalf("cleancommands locks callback failed: %v", err)
	}

	locksPage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	locksMarkup := requireEditedMarkup(t, locksPage)
	assertButton(t, locksMarkup, 0, 0, "Example Commands", "ux:helpctx:cleancommands:locks_examples", "")
	assertButton(t, locksMarkup, 0, 1, "Lock descriptions", "ux:helpctx:cleancommands:locks_descriptions", "")
	assertButton(t, locksMarkup, 1, 0, "Back", "ux:help:cleancommands", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 4,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-cleancommands-locks-examples",
			From: telegram.User{ID: 67, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:helpctx:cleancommands:locks_examples",
		},
	}); err != nil {
		t.Fatalf("cleancommands locks examples callback failed: %v", err)
	}

	locksExamplesPage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	locksExamplesMarkup := requireEditedMarkup(t, locksExamplesPage)
	assertButton(t, locksExamplesMarkup, 0, 0, "Back", "ux:helpctx:cleancommands:locks", "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 5,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-cleancommands-logchannels",
			From: telegram.User{ID: 67, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: root.MessageID,
				Chat:      chat,
			},
			Data: "ux:helpctx:cleancommands:logchannels",
		},
	}); err != nil {
		t.Fatalf("cleancommands log channels callback failed: %v", err)
	}

	logPage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	logRendered := renderedText(logPage.Text)
	if !strings.Contains(logRendered, "/setlog") || !strings.Contains(logRendered, "/logcategories") {
		t.Fatalf("expected log channels help copy, got %q", logPage.Text)
	}
	logMarkup := requireEditedMarkup(t, logPage)
	assertButton(t, logMarkup, 0, 0, "Back", "ux:help:cleancommands", "")
}

func TestCleanServiceHelpPageUsesBackOnlyLayout(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 61, Type: "private"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-help-cleanservice",
			From: telegram.User{ID: 61, FirstName: "User"},
			Message: &telegram.Message{
				MessageID: 100,
				Chat:      chat,
			},
			Data: "ux:help:cleanservice",
		},
	}); err != nil {
		t.Fatalf("cleanservice help callback failed: %v", err)
	}

	edited := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	rendered := renderedText(edited.Text)
	if !strings.Contains(rendered, "/keepservice <type>") || !strings.Contains(rendered, "/cleanservicetypes") {
		t.Fatalf("expected cleanservice help page content, got %q", edited.Text)
	}
	markup := requireEditedMarkup(t, edited)
	assertButton(t, markup, 0, 0, "Back", "ux:help:root", "")
	assertNoButtonText(t, markup, "Website")
	assertNoButtonText(t, markup, "Add to Group")
	assertNoButtonText(t, markup, "Home")
	assertNoButtonText(t, markup, "Close")
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

func renderedText(value string) string {
	return html.UnescapeString(htmlTagPattern.ReplaceAllString(value, ""))
}
