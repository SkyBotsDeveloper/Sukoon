package owner_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestBroadcastQueuesDurableJob(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100770, Type: "supergroup", Title: "Owner"}

	for _, c := range []telegram.Chat{
		{ID: -100771, Type: "supergroup", Title: "A"},
		{ID: -100772, Type: "supergroup", Title: "B"},
	} {
		if err := h.Store.EnsureChat(context.Background(), h.Bot.ID, c); err != nil {
			t.Fatalf("ensure chat failed: %v", err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/broadcast chats Hello all",
		},
	}); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	jobsList, err := h.Store.ListRecentJobs(context.Background(), h.Bot.ID, 5)
	if err != nil {
		t.Fatalf("list jobs failed: %v", err)
	}
	if len(jobsList) == 0 || jobsList[0].Kind != jobs.KindBroadcast {
		t.Fatalf("expected queued broadcast job, got %+v", jobsList)
	}
}

func TestGlobalBlacklistUserIsEnforced(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100773, Type: "supergroup", Title: "Owner"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/bluser",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Blacklisted"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("bluser failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "Blacklisted"},
			Chat:      chat,
			Text:      "hello",
		},
	}); err != nil {
		t.Fatalf("blacklisted user message failed: %v", err)
	}

	if len(h.Client.Bans) == 0 || h.Client.Bans[0].UserID != 20 {
		t.Fatalf("expected global blacklist enforcement ban, got %+v", h.Client.Bans)
	}
}
