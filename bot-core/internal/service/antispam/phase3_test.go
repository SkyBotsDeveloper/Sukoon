package antispam_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestApprovedUserBypassesBlocklist(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100720, Type: "supergroup", Title: "Blocklist"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/addblocklist word spam",
		},
	}); err != nil {
		t.Fatalf("add blocklist failed: %v", err)
	}
	if err := h.Store.SetApproval(context.Background(), h.Bot.ID, chat.ID, 20, 1, true, ""); err != nil {
		t.Fatalf("set approval failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "Approved"},
			Chat:      chat,
			Text:      "spam",
		},
	}); err != nil {
		t.Fatalf("approved user message failed: %v", err)
	}

	if len(h.Client.DeletedMessages) != 0 {
		t.Fatalf("expected approved user to bypass blocklist, got deletions %+v", h.Client.DeletedMessages)
	}
}

func TestBulkRemoveBlocklistEntries(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100721, Type: "supergroup", Title: "Bulk Blocklist"}

	for idx, cmd := range []string{"/addblocklist word spam", "/addblocklist phrase buy now"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 10),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", cmd, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 30,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/rmbl spam | buy now",
		},
	}); err != nil {
		t.Fatalf("bulk remove failed: %v", err)
	}

	rules, err := h.Store.ListBlocklistRules(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list blocklist rules failed: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected blocklist to be empty after bulk remove, got %+v", rules)
	}
}

func TestBlocklistAliasesAndRemoveAll(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100722, Type: "supergroup", Title: "Blocklist Aliases"}

	for idx, cmd := range []string{"/addblocklist word spam", "/addblocklist phrase buy now", "/rmblocklist spam", "/unblocklistall"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 40),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", cmd, err)
		}
	}

	rules, err := h.Store.ListBlocklistRules(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list blocklist rules failed: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected unblocklistall to clear remaining rules, got %+v", rules)
	}
}
