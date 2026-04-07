package antispam_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestAntifloodMutesAfterConfiguredLimit(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100300, Type: "supergroup", Title: "Flood"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/setflood 2",
		},
	}); err != nil {
		t.Fatalf("setflood failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(10 + i),
			Message: &telegram.Message{
				MessageID: int64(20 + i),
				From:      &telegram.User{ID: 20, FirstName: "Flooder"},
				Chat:      chat,
				Text:      "hello",
			},
		}); err != nil {
			t.Fatalf("flood message %d failed: %v", i, err)
		}
	}

	if len(h.Client.Restrictions) == 0 {
		t.Fatalf("expected antiflood restriction")
	}
}

func TestAntifloodAliasesAndLockTypes(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100301, Type: "supergroup", Title: "Flood Aliases"}

	for idx, text := range []string{
		"/setflood 4",
		"/setfloodtimer 12",
		"/floodmode kick",
		"/flood",
		"/lock gifs",
		"/locktypes",
		"/clearflood",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(30 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", text, err)
		}
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if bundle.Antiflood.WindowSeconds != 12 || bundle.Antiflood.Action != "kick" {
		t.Fatalf("expected updated antiflood settings, got %+v", bundle.Antiflood)
	}
	if _, ok := bundle.Locks["gif"]; !ok {
		t.Fatalf("expected gif lock via alias, got %+v", bundle.Locks)
	}
	if got := h.Client.Messages[3].Text; got != "Antiflood is on: limit=4 window=12s action=kick." {
		t.Fatalf("unexpected /flood status response: %q", got)
	}
	if got := h.Client.Messages[5].Text; got != "Supported lock types: links, forwards, media, sticker, gif." {
		t.Fatalf("unexpected /locktypes response: %q", got)
	}
}
