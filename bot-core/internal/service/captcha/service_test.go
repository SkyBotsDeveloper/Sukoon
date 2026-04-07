package captcha_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestCaptchaJoinAndCallbackFlow(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100400, Type: "supergroup", Title: "Captcha"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/captcha on",
		},
	}); err != nil {
		t.Fatalf("enable captcha failed: %v", err)
	}

	joinUpdate := telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID:      11,
			From:           &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:           chat,
			NewChatMembers: []telegram.User{{ID: 20, FirstName: "NewUser"}},
		},
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, joinUpdate); err != nil {
		t.Fatalf("join update failed: %v", err)
	}

	challenge, ok := h.Store.PendingCaptchaForUser(h.Bot.ID, chat.ID, 20)
	if !ok {
		t.Fatalf("expected pending captcha challenge")
	}
	if len(h.Client.Restrictions) == 0 || h.Client.Restrictions[0].UserID != 20 {
		t.Fatalf("expected initial restriction for captcha")
	}

	callbackUpdate := telegram.Update{
		UpdateID: 3,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-1",
			From: telegram.User{ID: 20, FirstName: "NewUser"},
			Message: &telegram.Message{
				MessageID: challenge.MessageID,
				Chat:      chat,
			},
			Data: "captcha:" + challenge.ID + ":" + challenge.Answer,
		},
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, callbackUpdate); err != nil {
		t.Fatalf("captcha callback failed: %v", err)
	}
	if len(h.Client.CallbackAnswers) == 0 {
		t.Fatalf("expected callback answer")
	}
	if last := h.Client.Restrictions[len(h.Client.Restrictions)-1]; !last.Permissions.CanSendMessages {
		t.Fatalf("expected final unrestrict call, got %+v", last)
	}
}

func TestCaptchaControlCommands(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100401, Type: "supergroup", Title: "Captcha Controls"}

	for idx, text := range []string{
		"/captchamode",
		"/captchamode button",
		"/captchakick mute",
		"/captchakicktime 90",
		"/captcha on",
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
	if !bundle.Captcha.Enabled || bundle.Captcha.Mode != "button" || bundle.Captcha.FailureAction != "mute" || bundle.Captcha.TimeoutSeconds != 90 {
		t.Fatalf("unexpected captcha settings: %+v", bundle.Captcha)
	}
}
