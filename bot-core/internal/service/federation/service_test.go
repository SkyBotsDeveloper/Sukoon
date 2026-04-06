package federation_test

import (
	"context"
	"io"
	"log/slog"
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
