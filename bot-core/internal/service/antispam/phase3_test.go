package antispam_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

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

func TestBlocklistModeReasonAndDeleteSettings(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100723, Type: "supergroup", Title: "Blocklist Settings"}

	for idx, cmd := range []string{
		"/blocklistmode warn",
		"/blocklistdelete off",
		"/setblocklistreason Keep it civil.",
		"/addblocklist boo",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 100),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", cmd, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 10,
		Message: &telegram.Message{
			MessageID: 110,
			From:      &telegram.User{ID: 20, FirstName: "Target"},
			Chat:      chat,
			Text:      "boo",
		},
	}); err != nil {
		t.Fatalf("blocklisted message failed: %v", err)
	}

	if len(h.Client.DeletedMessages) != 0 {
		t.Fatalf("expected blocklistdelete off to preserve the triggering message, got %+v", h.Client.DeletedMessages)
	}
	warns, err := h.Store.GetWarnings(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("get warnings failed: %v", err)
	}
	if warns != 1 {
		t.Fatalf("expected blocklist warn mode to increment warnings, got %d", warns)
	}
	last := h.Client.Messages[len(h.Client.Messages)-1].Text
	if !strings.Contains(last, "Keep it civil.") {
		t.Fatalf("expected default blocklist reason in warning message, got %q", last)
	}
}

func TestBlocklistSupportsOverridesAndAdvancedMatchers(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100724, Type: "supergroup", Title: "Blocklist Matchers"}

	for idx, cmd := range []string{
		"/addblocklist boo Don't scare the ghosts! {tmute 6h}",
		"/addblocklist (hi, hey, hello) Stop saying hello!",
		"/addblocklist \"bit.ly/???\" We dont like 3 letter shorteners!",
		"/addblocklist \"file:*.zip\" zip files are not allowed here.",
		"/addblocklist \"inline:@gif\" The gif bot is not allowed here.",
		"/addblocklist \"forward:@botnews\" The bot news channel is not allowed here.",
		"/addblocklist \"exact:hi\" Exact hi only.",
		"/addblocklist \"prefix:wave\" Prefix wave only.",
		"/addblocklist \"lookalike:bot\" No lookalikes!",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(idx + 200),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      cmd,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", cmd, err)
		}
	}

	rules, err := h.Store.ListBlocklistRules(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("list blocklist rules failed: %v", err)
	}
	if len(rules) < 11 {
		t.Fatalf("expected grouped blocklist command to expand rules, got %+v", rules)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 20,
		Message: &telegram.Message{
			MessageID: 500,
			From:      &telegram.User{ID: 20, FirstName: "Target"},
			Chat:      chat,
			Text:      "boo",
		},
	}); err != nil {
		t.Fatalf("override blocklist failed: %v", err)
	}
	if len(h.Client.Restrictions) == 0 || h.Client.Restrictions[0].Until == nil {
		t.Fatalf("expected tmute override to restrict target, got %+v", h.Client.Restrictions)
	}
	if duration := h.Client.Restrictions[0].Until.Sub(time.Now()); duration < (6*time.Hour-time.Hour) || duration > (6*time.Hour+time.Hour) {
		t.Fatalf("expected tmute override around 6h, got %s", duration)
	}

	cases := []telegram.Message{
		{MessageID: 501, From: &telegram.User{ID: 21, FirstName: "Target2"}, Chat: chat, Text: "hi"},
		{MessageID: 502, From: &telegram.User{ID: 22, FirstName: "Target3"}, Chat: chat, Text: "visit bit.ly/hey now"},
		{MessageID: 503, From: &telegram.User{ID: 23, FirstName: "Target4"}, Chat: chat, Document: &telegram.Document{FileID: "doc", FileName: "archive.zip"}},
		{MessageID: 504, From: &telegram.User{ID: 24, FirstName: "Target5"}, Chat: chat, Text: "gif please", ViaBot: &telegram.User{ID: 99, Username: "gif"}},
		{MessageID: 505, From: &telegram.User{ID: 25, FirstName: "Target6"}, Chat: chat, Text: "forwarded", ForwardFromChat: &telegram.Chat{ID: -1009, Username: "botnews"}},
		{MessageID: 506, From: &telegram.User{ID: 26, FirstName: "Target7"}, Chat: chat, Text: "wave there"},
		{MessageID: 507, From: &telegram.User{ID: 27, FirstName: "Target8"}, Chat: chat, Text: "say wave"},
		{MessageID: 508, From: &telegram.User{ID: 28, FirstName: "Target9"}, Chat: chat, Text: "вот"},
	}
	for idx, msg := range cases {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(30 + idx),
			Message:  &msg,
		}); err != nil {
			t.Fatalf("matcher case %d failed: %v", idx, err)
		}
	}

	if len(h.Client.DeletedMessages) < 7 {
		t.Fatalf("expected multiple blocklist matches to delete messages, got %+v", h.Client.DeletedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 50,
		Message: &telegram.Message{
			MessageID: 509,
			From:      &telegram.User{ID: 29, FirstName: "Target10"},
			Chat:      chat,
			Text:      "say wave",
		},
	}); err != nil {
		t.Fatalf("non-matching prefix case failed: %v", err)
	}
	if h.Client.DeletedMessages[len(h.Client.DeletedMessages)-1].MessageID == 509 {
		t.Fatalf("expected /prefix:wave not to match 'say wave'")
	}
}

func TestBlocklistStickerpackAndCreatorOnlyRemoveAll(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100725, Type: "supergroup", Title: "Sticker Blocklist"}

	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{User: telegram.User{ID: 1, FirstName: "Owner"}, Status: "creator"},
		{User: telegram.User{ID: 2, FirstName: "Admin"}, Status: "administrator", CanDeleteMessages: true, CanRestrictMembers: true},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 600,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/addblocklist stickerpack:<> These stickers are banned!",
			ReplyToMessage: &telegram.Message{
				MessageID: 599,
				From:      &telegram.User{ID: 30, FirstName: "StickerUser"},
				Chat:      chat,
				Sticker:   &telegram.Sticker{FileID: "sticker", SetName: "funpack_by_test"},
			},
		},
	}); err != nil {
		t.Fatalf("stickerpack add failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 601,
			From:      &telegram.User{ID: 31, FirstName: "StickerTarget"},
			Chat:      chat,
			Sticker:   &telegram.Sticker{FileID: "sticker2", SetName: "funpack_by_test"},
		},
	}); err != nil {
		t.Fatalf("stickerpack enforcement failed: %v", err)
	}
	if len(h.Client.DeletedMessages) == 0 || h.Client.DeletedMessages[len(h.Client.DeletedMessages)-1].MessageID != 601 {
		t.Fatalf("expected stickerpack blocklist to delete matching sticker message, got %+v", h.Client.DeletedMessages)
	}

	err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 602,
			From:      &telegram.User{ID: 2, FirstName: "Admin"},
			Chat:      chat,
			Text:      "/unblocklistall",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "chat creator rights required") {
		t.Fatalf("expected unblocklistall to require creator, got %v", err)
	}
}
