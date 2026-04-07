package federation_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

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
