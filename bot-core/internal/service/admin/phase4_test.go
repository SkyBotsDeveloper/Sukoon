package admin_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestMuterCanMuteButCannotBan(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100712, Type: "supergroup", Title: "Muters"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/muter",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Muter"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("grant muter failed: %v", err)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 11,
			From:      &telegram.User{ID: 20, FirstName: "Muter"},
			Chat:      chat,
			Text:      "/mute",
			ReplyToMessage: &telegram.Message{
				MessageID: 8,
				From:      &telegram.User{ID: 30, FirstName: "Target"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("mute by muter failed: %v", err)
	}

	if len(h.Client.Restrictions) != 1 || h.Client.Restrictions[0].UserID != 30 {
		t.Fatalf("expected muter to apply one restriction, got %+v", h.Client.Restrictions)
	}

	err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 20, FirstName: "Muter"},
			Chat:      chat,
			Text:      "/ban",
			ReplyToMessage: &telegram.Message{
				MessageID: 7,
				From:      &telegram.User{ID: 31, FirstName: "Target2"},
				Chat:      chat,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected muter ban attempt to fail")
	}
}

func TestPinAndUnpinCommands(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100713, Type: "supergroup", Title: "Pins"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:           telegram.User{ID: 10, FirstName: "Pinner"},
			Status:         "administrator",
			CanPinMessages: true,
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 15,
			From:      &telegram.User{ID: 10, FirstName: "Pinner"},
			Chat:      chat,
			Text:      "/pin quiet",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Pinned"},
				Chat:      chat,
				Text:      "Important",
			},
		},
	}); err != nil {
		t.Fatalf("pin failed: %v", err)
	}
	if len(h.Client.PinnedMessages) != 1 || h.Client.PinnedMessages[0].MessageID != 9 || !h.Client.PinnedMessages[0].DisableNotification {
		t.Fatalf("expected quiet pin of replied message, got %+v", h.Client.PinnedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 16,
			From:      &telegram.User{ID: 10, FirstName: "Pinner"},
			Chat:      chat,
			Text:      "/unpin",
			ReplyToMessage: &telegram.Message{
				MessageID: 9,
				From:      &telegram.User{ID: 20, FirstName: "Pinned"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("unpin failed: %v", err)
	}
	if len(h.Client.UnpinnedMessages) != 1 || h.Client.UnpinnedMessages[0].MessageID == nil || *h.Client.UnpinnedMessages[0].MessageID != 9 {
		t.Fatalf("expected specific unpin of message 9, got %+v", h.Client.UnpinnedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 17,
			From:      &telegram.User{ID: 10, FirstName: "Pinner"},
			Chat:      chat,
			Text:      "/unpinall",
		},
	}); err != nil {
		t.Fatalf("unpinall failed: %v", err)
	}
	if len(h.Client.UnpinAllChats) != 1 || h.Client.UnpinAllChats[0] != chat.ID {
		t.Fatalf("expected unpinall for chat %d, got %+v", chat.ID, h.Client.UnpinAllChats)
	}
}

func TestModsRepliesToTriggerMessage(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100714, Type: "supergroup", Title: "Silent Staff"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 22,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/mods",
		},
	}); err != nil {
		t.Fatalf("mods failed: %v", err)
	}

	if len(h.Client.Messages) == 0 {
		t.Fatalf("expected mods response message")
	}
	last := h.Client.Messages[len(h.Client.Messages)-1]
	if last.Options.ReplyToMessageID != 22 {
		t.Fatalf("expected /mods response to reply to message 22, got %+v", last.Options)
	}
}

func TestAdminsListsVisibleChatAdmins(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100715, Type: "supergroup", Title: "Admins"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:   telegram.User{ID: 1, FirstName: "Owner", Username: "owner"},
			Status: "creator",
		},
		{
			User:              telegram.User{ID: 2, FirstName: "Mod", Username: "mod"},
			Status:            "administrator",
			CanDeleteMessages: true,
		},
		{
			User:        telegram.User{ID: 3, FirstName: "Anon"},
			Status:      "administrator",
			IsAnonymous: true,
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 30,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
			Text:      "/admins",
		},
	}); err != nil {
		t.Fatalf("admins failed: %v", err)
	}

	if len(h.Client.Messages) == 0 {
		t.Fatalf("expected admins response")
	}
	last := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(last.Text, "@owner [owner]") || !strings.Contains(last.Text, "@mod") || !strings.Contains(last.Text, "Anonymous admin") {
		t.Fatalf("expected admin list in response, got %q", last.Text)
	}
}

func TestApprovalStatusUnapproveAllAndLogAliases(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100716, Type: "supergroup", Title: "Approvals"}
	if err := h.Store.EnsureUser(context.Background(), telegram.User{ID: 20, Username: "target", FirstName: "Target"}); err != nil {
		t.Fatalf("ensure user failed: %v", err)
	}

	for idx, text := range []string{
		"/approve @target",
		"/approval @target",
		"/setlog -100999",
		"/logcategories",
		"/nolog",
		"/unapproveall",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(50 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", text, err)
		}
	}

	if got := h.Client.Messages[1].Text; !strings.Contains(got, "is approved") {
		t.Fatalf("expected approval status response, got %q", got)
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if bundle.Settings.LogChannelID != nil {
		t.Fatalf("expected nolog to clear log channel, got %+v", bundle.Settings.LogChannelID)
	}
	if got := h.Client.Messages[3].Text; !strings.Contains(got, "Current log categories") {
		t.Fatalf("expected logcategories response, got %q", got)
	}

	approved, err := h.Store.IsApproved(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("approval lookup failed: %v", err)
	}
	if approved {
		t.Fatalf("expected unapproveall to clear approvals")
	}
}

func TestCleanCommandAliases(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100717, Type: "supergroup", Title: "Clean Commands"}

	for idx, text := range []string{
		"/cleancommand on",
		"/cleancommandtypes",
		"/keepcommand",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(80 + idx),
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
	if bundle.Settings.CleanCommands {
		t.Fatalf("expected keepcommand to disable clean commands")
	}
	if got := h.Client.Messages[1].Text; !strings.Contains(got, "Clean command types") {
		t.Fatalf("expected cleancommandtypes response, got %q", got)
	}
}
