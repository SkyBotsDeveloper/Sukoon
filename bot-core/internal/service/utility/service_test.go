package utility_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

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
	chat := telegram.Chat{ID: -100761, Type: "supergroup", Title: "Privacy"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
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
			Chat:      chat,
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
			Chat:      chat,
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

func TestStartAndHelpCommandsRespond(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 50, FirstName: "User"},
			Chat:      telegram.Chat{ID: 50, Type: "private"},
			Text:      "/start",
		},
	}); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	startMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(startMessage.Text, "Welcome to Sukoon") {
		t.Fatalf("expected /start response to introduce Sukoon, got %q", startMessage.Text)
	}
	if startMessage.Options.ReplyToMessageID != 10 {
		t.Fatalf("expected /start response to reply to message 10, got %+v", startMessage.Options)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      telegram.Chat{ID: -100990, Type: "supergroup", Title: "Help"},
			Text:      "/help",
		},
	}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	helpMessage := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(helpMessage.Text, "/ban") || !strings.Contains(helpMessage.Text, "/reports") || !strings.Contains(helpMessage.Text, "/rules") {
		t.Fatalf("expected /help response to contain command guidance, got %q", helpMessage.Text)
	}
	if helpMessage.Options.ReplyToMessageID != 11 {
		t.Fatalf("expected /help response to reply to message 11, got %+v", helpMessage.Options)
	}
}
