package federation_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestFederationWorkflowQueuesFedBanJob(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100780, Type: "supergroup", Title: "Fed"}

	for idx, text := range []string{"/newfed main Main Federation", "/joinfed main"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 10),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("federation setup %q failed: %v", text, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 20,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/fedpromote",
			ReplyToMessage: &telegram.Message{
				MessageID: 19,
				From:      &telegram.User{ID: 25, FirstName: "Admin"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("fedpromote failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 4,
		Message: &telegram.Message{
			MessageID: 21,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/fban spamming",
			ReplyToMessage: &telegram.Message{
				MessageID: 18,
				From:      &telegram.User{ID: 30, FirstName: "Target"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("fban failed: %v", err)
	}

	federation, err := h.Store.GetFederationByChat(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load federation by chat failed: %v", err)
	}
	ban, banned, err := h.Store.GetFederationBan(context.Background(), federation.ID, 30)
	if err != nil {
		t.Fatalf("get federation ban failed: %v", err)
	}
	if !banned || ban.UserID != 30 {
		t.Fatalf("expected stored federation ban, got banned=%t ban=%+v", banned, ban)
	}

	jobsList, err := h.Store.ListRecentJobs(context.Background(), h.Bot.ID, 5)
	if err != nil {
		t.Fatalf("list jobs failed: %v", err)
	}
	if len(jobsList) == 0 || jobsList[0].Kind != jobs.KindFederationBan {
		t.Fatalf("expected queued federation ban job, got %+v", jobsList)
	}
}

func TestFederationRenameChatFedAndDemoteMe(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100781, Type: "supergroup", Title: "Fed Tools"}
	if err := h.Store.EnsureUser(context.Background(), telegram.User{ID: 25, FirstName: "Fed", Username: "fedadmin"}); err != nil {
		t.Fatalf("ensure user failed: %v", err)
	}

	for idx, text := range []string{
		"/newfed tools Tools Federation",
		"/joinfed tools",
		"/renamefed ops Operations Federation",
		"/chatfed",
		"/fedpromote @fedadmin",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 40),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", text, err)
		}
	}

	federation, err := h.Store.GetFederationByChat(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load federation by chat failed: %v", err)
	}
	if federation.ShortName != "ops" || federation.DisplayName != "Operations Federation" {
		t.Fatalf("expected renamed federation, got %+v", federation)
	}
	if got := h.Client.Messages[3].Text; !strings.Contains(got, "Operations Federation") || !strings.Contains(got, "(ops)") {
		t.Fatalf("expected /chatfed response to show renamed federation, got %q", got)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 6,
		Message: &telegram.Message{
			MessageID: 46,
			From:      &telegram.User{ID: 25, FirstName: "Fed", Username: "fedadmin"},
			Chat:      chat,
			Text:      "/feddemoteme",
		},
	}); err != nil {
		t.Fatalf("feddemoteme failed: %v", err)
	}

	admins, err := h.Store.ListFederationAdmins(context.Background(), federation.ID)
	if err != nil {
		t.Fatalf("list federation admins failed: %v", err)
	}
	for _, admin := range admins {
		if admin.UserID == 25 {
			t.Fatalf("expected feddemoteme to remove user 25, got %+v", admins)
		}
	}
}

func TestFederationOwnerSettingsImportExportStatsAndSubscriptions(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100782, Type: "supergroup", Title: "Fed Main"}
	sourceChat := telegram.Chat{ID: -100783, Type: "supergroup", Title: "Fed Source"}

	for idx, update := range []struct {
		user telegram.User
		chat telegram.Chat
		text string
	}{
		{telegram.User{ID: 1, FirstName: "Owner"}, chat, "/newfed main Main Federation"},
		{telegram.User{ID: 1, FirstName: "Owner"}, chat, "/joinfed main"},
		{telegram.User{ID: 2, FirstName: "Other"}, sourceChat, "/newfed source Source Federation"},
	} {
		if err := sendFederationCommand(h, int64(idx+100), update.user, update.chat, update.text); err != nil {
			t.Fatalf("command %q failed: %v", update.text, err)
		}
	}

	if err := sendFederationCommand(h, 104, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/newfed second Second Federation"); err != nil {
		t.Fatalf("duplicate newfed command failed: %v", err)
	}
	owned, err := h.Store.ListFederationsForUser(context.Background(), h.Bot.ID, 1)
	if err != nil {
		t.Fatalf("list owner feds failed: %v", err)
	}
	ownerCount := 0
	for _, fed := range owned {
		if fed.OwnerUserID == 1 {
			ownerCount++
		}
	}
	if ownerCount != 1 {
		t.Fatalf("expected one owned federation, got %d: %+v", ownerCount, owned)
	}
	if last := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(last, "already own federation") {
		t.Fatalf("expected duplicate newfed response, got %q", last)
	}

	if err := sendFederationCommand(h, 105, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fedreason on"); err != nil {
		t.Fatalf("fedreason failed: %v", err)
	}
	if err := sendFederationCommand(h, 106, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fban 30"); err == nil || !strings.Contains(err.Error(), "requires a reason") {
		t.Fatalf("expected reason-required fban error, got %v", err)
	}
	if err := sendFederationCommand(h, 107, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fban 30 raid spam"); err != nil {
		t.Fatalf("fban with reason failed: %v", err)
	}
	if err := sendFederationCommand(h, 108, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fedexport human"); err != nil {
		t.Fatalf("fedexport failed: %v", err)
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "- 30: raid spam") {
		t.Fatalf("expected human fedexport, got %q", got)
	}
	if err := sendFederationCommand(h, 109, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fedimport keep minicsv 91,imported spam"); err != nil {
		t.Fatalf("fedimport failed: %v", err)
	}
	if err := sendFederationCommand(h, 110, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fedstat 91 main"); err != nil {
		t.Fatalf("fedstat failed: %v", err)
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "imported spam") {
		t.Fatalf("expected fedstat imported reason, got %q", got)
	}

	sourceFed, err := h.Store.GetFederationByShortName(context.Background(), h.Bot.ID, "source")
	if err != nil {
		t.Fatalf("load source federation failed: %v", err)
	}
	if err := h.Store.SetFederationBan(context.Background(), telegramFedBan(sourceFed.ID, 77, "source ban", 2), true); err != nil {
		t.Fatalf("seed source ban failed: %v", err)
	}
	if err := sendFederationCommand(h, 111, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/subfed source"); err != nil {
		t.Fatalf("subfed failed: %v", err)
	}
	if err := sendFederationCommand(h, 112, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/fedsubs main"); err != nil {
		t.Fatalf("fedsubs failed: %v", err)
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "Source Federation") {
		t.Fatalf("expected fedsubs source federation, got %q", got)
	}
	if err := sendFederationCommand(h, 113, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/quietfed on"); err != nil {
		t.Fatalf("quietfed failed: %v", err)
	}
	messageCount := len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 114,
		Message: &telegram.Message{
			MessageID:      114,
			From:           &telegram.User{ID: 77, FirstName: "Joined"},
			Chat:           chat,
			NewChatMembers: []telegram.User{{ID: 77, FirstName: "Joined"}},
		},
	}); err != nil {
		t.Fatalf("fedbanned join failed: %v", err)
	}
	if len(h.Client.Bans) == 0 || h.Client.Bans[len(h.Client.Bans)-1].UserID != 77 {
		t.Fatalf("expected subscribed fedban to ban user 77, got %+v", h.Client.Bans)
	}
	if len(h.Client.Messages) != messageCount {
		t.Fatalf("expected quietfed to suppress join notification, got new messages %+v", h.Client.Messages[messageCount:])
	}
}

func TestFederationHelpPagesUseNestedBackOnlyButtons(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: 5001, Type: "private", Title: "PM"}

	if err := sendFederationCommand(h, 201, telegram.User{ID: 1, FirstName: "Owner"}, chat, "/help federations"); err != nil {
		t.Fatalf("help federations failed: %v", err)
	}
	markup := h.Client.Messages[len(h.Client.Messages)-1].Options.ReplyMarkup
	if markup == nil || len(markup.InlineKeyboard) != 3 {
		t.Fatalf("expected federation main buttons, got %+v", markup)
	}
	if len(markup.InlineKeyboard[0]) != 2 || markup.InlineKeyboard[0][0].Text != "Fed Admin Commands" || markup.InlineKeyboard[0][1].Text != "Federation Owner Commands" {
		t.Fatalf("expected admin/owner first row, got %+v", markup.InlineKeyboard[0])
	}
	if markup.InlineKeyboard[1][0].Text != "User Commands" || markup.InlineKeyboard[2][0].Text != "Back" {
		t.Fatalf("expected user and back rows, got %+v", markup.InlineKeyboard)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 202,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "fed-owner",
			From: telegram.User{ID: 1, FirstName: "Owner"},
			Message: &telegram.Message{
				MessageID: h.Client.Messages[len(h.Client.Messages)-1].MessageID,
				Chat:      chat,
			},
			Data: "ux:help:federations_owner",
		},
	}); err != nil {
		t.Fatalf("owner help callback failed: %v", err)
	}
	if len(h.Client.EditedMessages) == 0 {
		t.Fatalf("expected callback edit")
	}
	edited := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(edited.Text, "/fedexport") || !strings.Contains(edited.Text, "/setfedlang") {
		t.Fatalf("expected owner command page, got %q", edited.Text)
	}
	if edited.Options.ReplyMarkup == nil || len(edited.Options.ReplyMarkup.InlineKeyboard) != 1 || edited.Options.ReplyMarkup.InlineKeyboard[0][0].Text != "Back" {
		t.Fatalf("expected back-only owner markup, got %+v", edited.Options.ReplyMarkup)
	}
}

func sendFederationCommand(h *testsupport.Harness, updateID int64, user telegram.User, chat telegram.Chat, text string) error {
	return h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: updateID,
		Message: &telegram.Message{
			MessageID: updateID,
			From:      &user,
			Chat:      chat,
			Text:      text,
		},
	})
}

func telegramFedBan(federationID string, userID int64, reason string, bannedBy int64) domain.FederationBan {
	return domain.FederationBan{
		FederationID: federationID,
		UserID:       userID,
		Reason:       reason,
		BannedBy:     bannedBy,
	}
}
