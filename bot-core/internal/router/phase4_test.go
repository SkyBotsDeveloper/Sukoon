package router_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestCleanServiceCoversTitlePhotoAndOther(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100714, Type: "supergroup", Title: "Clean"}

	for idx, command := range []string{
		"/cleanservice title on",
		"/cleanservice photo on",
		"/cleanservice other on",
		"/cleanservice videochat on",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(10 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      command,
			},
		}); err != nil {
			t.Fatalf("enable %q failed: %v", command, err)
		}
	}

	serviceUpdates := []telegram.Update{
		{
			UpdateID: 10,
			Message: &telegram.Message{
				MessageID:    20,
				From:         &telegram.User{ID: 2, FirstName: "User"},
				Chat:         chat,
				NewChatTitle: "New title",
			},
		},
		{
			UpdateID: 11,
			Message: &telegram.Message{
				MessageID:    21,
				From:         &telegram.User{ID: 2, FirstName: "User"},
				Chat:         chat,
				NewChatPhoto: []telegram.PhotoSize{{FileID: "photo"}},
			},
		},
		{
			UpdateID: 12,
			Message: &telegram.Message{
				MessageID:        22,
				From:             &telegram.User{ID: 2, FirstName: "User"},
				Chat:             chat,
				VideoChatStarted: &telegram.VideoChatStarted{},
			},
		},
		{
			UpdateID: 13,
			Message: &telegram.Message{
				MessageID:                     23,
				From:                          &telegram.User{ID: 2, FirstName: "User"},
				Chat:                          chat,
				MessageAutoDeleteTimerChanged: &telegram.MessageAutoDeleteTimerChanged{MessageAutoDeleteTime: 60},
			},
		},
	}

	for _, update := range serviceUpdates {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, update); err != nil {
			t.Fatalf("service update %d failed: %v", update.UpdateID, err)
		}
	}

	if len(h.Client.DeletedMessages) != 4 {
		t.Fatalf("expected 4 cleanservice deletions, got %+v", h.Client.DeletedMessages)
	}
}
