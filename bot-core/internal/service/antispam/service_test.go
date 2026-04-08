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
	if len(h.Client.DeletedMessages) != 1 || h.Client.DeletedMessages[0].MessageID != 22 {
		t.Fatalf("expected only the triggering flood message to be deleted, got %+v", h.Client.DeletedMessages)
	}
}

func TestAntifloodCommandsSupportTimedModeAndClearFlood(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100301, Type: "supergroup", Title: "Flood Aliases"}

	for idx, text := range []string{
		"/setflood 4",
		"/setfloodtimer 10 30s",
		"/floodmode tban 3d",
		"/flood",
		"/clearflood on",
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
	if bundle.Antiflood.Limit != 4 || bundle.Antiflood.TimedLimit != 10 || bundle.Antiflood.WindowSeconds != 30 {
		t.Fatalf("expected updated antiflood settings, got %+v", bundle.Antiflood)
	}
	if bundle.Antiflood.Action != "tban" || bundle.Antiflood.ActionDurationSeconds != 3*24*60*60 || !bundle.Antiflood.ClearAll {
		t.Fatalf("expected timed antiflood mode and clearflood toggle, got %+v", bundle.Antiflood)
	}
	if got := h.Client.Messages[3].Text; got != "Antiflood settings:\n- Consecutive limit: 4 messages in a row\n- Timed limit: 10 messages in 30s\n- Action: tban 3d\n- Clearflood: off" {
		t.Fatalf("unexpected /flood status response: %q", got)
	}
	if got := h.Client.Messages[4].Text; got != "Clearflood is now on. Sukoon will delete the full triggered flood set." {
		t.Fatalf("unexpected /clearflood response: %q", got)
	}
}

func TestAntifloodClearFloodDeletesFullTriggeredSet(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100302, Type: "supergroup", Title: "Flood Clear"}

	for idx, text := range []string{
		"/setflood 2",
		"/clearflood on",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(40 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	for i := 0; i < 3; i++ {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(20 + i),
			Message: &telegram.Message{
				MessageID: int64(50 + i),
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
	if len(h.Client.DeletedMessages) != 3 {
		t.Fatalf("expected full flood set deletion, got %+v", h.Client.DeletedMessages)
	}
	for idx, deleted := range h.Client.DeletedMessages {
		if want := int64(50 + idx); deleted.MessageID != want {
			t.Fatalf("deleted message %d = %d, want %d", idx, deleted.MessageID, want)
		}
	}
}
