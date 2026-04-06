package antibio_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestFreeSupportsUsername(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100750, Type: "supergroup", Title: "Free"}
	if err := h.Store.EnsureUser(context.Background(), telegram.User{ID: 20, Username: "freeuser", FirstName: "Free"}); err != nil {
		t.Fatalf("ensure user failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/free @freeuser",
		},
	}); err != nil {
		t.Fatalf("free by username failed: %v", err)
	}

	exempt, err := h.Store.IsAntiBioExempt(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("anti bio exemption lookup failed: %v", err)
	}
	if !exempt {
		t.Fatalf("expected anti bio exemption for username target")
	}
}

func TestAntiBioApprovalBypassAndBioEnforcement(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100751, Type: "supergroup", Title: "Bio"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/antibio on ban",
		},
	}); err != nil {
		t.Fatalf("enable antibio failed: %v", err)
	}

	h.Client.ChatsByID[20] = telegram.Chat{ID: 20, Bio: "https://spam.example"}
	if err := h.Store.SetApproval(context.Background(), h.Bot.ID, chat.ID, 20, 1, true); err != nil {
		t.Fatalf("set approval failed: %v", err)
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "Approved"},
			Chat:      chat,
			Text:      "hello",
		},
	}); err != nil {
		t.Fatalf("approved message failed: %v", err)
	}
	if len(h.Client.Bans) != 0 {
		t.Fatalf("expected approved user to bypass antibio, got bans %+v", h.Client.Bans)
	}

	h2 := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	h2.Client.ChatsByID[21] = telegram.Chat{ID: 21, Bio: "t.me/spamchannel"}
	if err := h2.Router.HandleUpdate(context.Background(), h2.Bot, h2.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/antibio on ban",
		},
	}); err != nil {
		t.Fatalf("enable antibio failed: %v", err)
	}
	if err := h2.Router.HandleUpdate(context.Background(), h2.Bot, h2.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 21, FirstName: "Spammer"},
			Chat:      chat,
			Text:      "hello",
		},
	}); err != nil {
		t.Fatalf("bio violating message failed: %v", err)
	}
	if len(h2.Client.Bans) != 1 || h2.Client.Bans[0].UserID != 21 {
		t.Fatalf("expected antibio ban for user 21, got %+v", h2.Client.Bans)
	}
}
