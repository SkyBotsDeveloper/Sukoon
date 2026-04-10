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
	if !strings.Contains(last.Text, "@owner [owner]") || !strings.Contains(last.Text, "@mod") || strings.Contains(last.Text, "Anonymous admin") {
		t.Fatalf("expected admin list in response, got %q", last.Text)
	}
}

func TestPromoteDemoteAdminCacheAdminErrorAndAnonAdmin(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100719, Type: "supergroup", Title: "Admin Controls"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 1, FirstName: "Owner", Username: "owner"},
			Status:             "creator",
			CanDeleteMessages:  true,
			CanRestrictMembers: true,
			CanChangeInfo:      true,
			CanPinMessages:     true,
			CanPromoteMembers:  true,
		},
	}
	if err := h.Store.EnsureUser(context.Background(), telegram.User{ID: 55, Username: "target", FirstName: "Target"}); err != nil {
		t.Fatalf("ensure target failed: %v", err)
	}

	for idx, text := range []string{"/promote @target", "/admincache", "/demote @target", "/adminerror off"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(200 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner", Username: "owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("owner command %q failed: %v", text, err)
		}
	}

	if len(h.Client.Promotions) != 2 {
		t.Fatalf("expected promote and demote API calls, got %+v", h.Client.Promotions)
	}
	if !h.Client.Promotions[0].Permissions.CanDeleteMessages || h.Client.Promotions[0].Permissions.CanPromoteMembers {
		t.Fatalf("expected promote permissions to mirror caller without add-admins, got %+v", h.Client.Promotions[0])
	}
	if h.Client.Promotions[1].Permissions.CanDeleteMessages || h.Client.Promotions[1].Permissions.CanRestrictMembers {
		t.Fatalf("expected demote call to clear permissions, got %+v", h.Client.Promotions[1])
	}
	if got := h.Client.Messages[1].Text; !strings.Contains(got, "Admin cache refreshed") {
		t.Fatalf("expected admincache response, got %q", got)
	}

	beforeSilent := len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 10,
		Message: &telegram.Message{
			MessageID: 250,
			From:      &telegram.User{ID: 90, FirstName: "Member"},
			Chat:      chat,
			Text:      "/promote @target",
		},
	}); err != nil {
		t.Fatalf("member promote with adminerror off failed unexpectedly: %v", err)
	}
	if len(h.Client.Messages) != beforeSilent {
		t.Fatalf("expected adminerror off to suppress member warning, got %+v", h.Client.Messages[beforeSilent:])
	}

	for idx, text := range []string{"/adminerror on", "/anonadmin on"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(20 + idx),
			Message: &telegram.Message{
				MessageID: int64(260 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner", Username: "owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("owner command %q failed: %v", text, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 30,
		Message: &telegram.Message{
			MessageID: 270,
			From:      &telegram.User{ID: 90, FirstName: "Member"},
			Chat:      chat,
			Text:      "/promote @target",
		},
	}); err != nil {
		t.Fatalf("member promote with adminerror on failed unexpectedly: %v", err)
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "You need to be admin to do this.") {
		t.Fatalf("expected adminerror warning, got %q", got)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 40,
		Message: &telegram.Message{
			MessageID: 280,
			SenderChat: &telegram.Chat{
				ID:    chat.ID,
				Type:  chat.Type,
				Title: chat.Title,
			},
			Chat: chat,
			Text: "/admincache",
		},
	}); err != nil {
		t.Fatalf("anonymous admin admincache failed: %v", err)
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "Admin cache refreshed") {
		t.Fatalf("expected anonymous admin to use admincache when anonadmin is on, got %q", got)
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if !bundle.Settings.AdminErrors || !bundle.Settings.AnonAdmins {
		t.Fatalf("expected adminerror and anonadmin settings to persist, got %+v", bundle.Settings)
	}
}

func TestApprovalStatusUnapproveAllAndLogCategories(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100716, Type: "supergroup", Title: "Approvals"}
	logChannel := telegram.Chat{ID: -100999, Type: "channel", Title: "Audit Log", Username: "auditlog"}
	if err := h.Store.EnsureUser(context.Background(), telegram.User{ID: 20, Username: "target", FirstName: "Target"}); err != nil {
		t.Fatalf("ensure user failed: %v", err)
	}
	h.Client.ChatsByID[logChannel.ID] = logChannel
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:   telegram.User{ID: 10, FirstName: "Admin", Username: "admin"},
			Status: "administrator",
		},
		{
			User:   telegram.User{ID: 1, FirstName: "Owner", Username: "owner"},
			Status: "creator",
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 50,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/approve @target testing",
		},
	}); err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID:       51,
			From:            &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:            chat,
			Text:            "/setlog",
			ForwardFromChat: &logChannel,
		},
	}); err != nil {
		t.Fatalf("setlog forward failed: %v", err)
	}
	for idx, text := range []string{
		"/logcategories",
		"/nolog automated",
		"/nolog reports",
		"/log reports",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(3 + idx),
			Message: &telegram.Message{
				MessageID: int64(52 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", text, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 10,
		Message: &telegram.Message{
			MessageID: 60,
			From:      &telegram.User{ID: 55, FirstName: "Member"},
			Chat:      chat,
			Text:      "/approval @target",
		},
	}); err != nil {
		t.Fatalf("approval status failed: %v", err)
	}

	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "is approved") || !strings.Contains(got, "testing") {
		t.Fatalf("expected approval status response, got %q", got)
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if bundle.Settings.LogChannelID == nil || *bundle.Settings.LogChannelID != logChannel.ID {
		t.Fatalf("expected forwarded setlog to configure log channel %d, got %+v", logChannel.ID, bundle.Settings.LogChannelID)
	}
	if bundle.Settings.LogCategoryAutomated {
		t.Fatalf("expected /nolog automated to disable automated logs")
	}
	if !bundle.Settings.LogCategoryReports {
		t.Fatalf("expected /log reports to re-enable reports logs")
	}
	var logCategoriesText string
	for _, msg := range h.Client.Messages {
		if msg.ChatID == chat.ID && strings.Contains(msg.Text, "Log categories:") {
			logCategoriesText = msg.Text
			break
		}
	}
	if !strings.Contains(logCategoriesText, "Log categories:") || !strings.Contains(logCategoriesText, "automated") {
		t.Fatalf("expected logcategories response, got %q", logCategoriesText)
	}

	before := len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 11,
		Message: &telegram.Message{
			MessageID: 61,
			From:      &telegram.User{ID: 10, FirstName: "Admin"},
			Chat:      chat,
			Text:      "/unapproveall",
		},
	}); err != nil {
		t.Fatalf("admin unapproveall failed: %v", err)
	}
	if len(h.Client.Messages) != before+1 || !strings.Contains(h.Client.Messages[len(h.Client.Messages)-1].Text, "Only the chat owner can do this.") {
		t.Fatalf("expected owner-only warning for unapproveall, got %+v", h.Client.Messages[before:])
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 12,
		Message: &telegram.Message{
			MessageID: 62,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/unapproveall",
		},
	}); err != nil {
		t.Fatalf("owner unapproveall failed: %v", err)
	}

	approved, err := h.Store.IsApproved(context.Background(), h.Bot.ID, chat.ID, 20)
	if err != nil {
		t.Fatalf("approval lookup failed: %v", err)
	}
	if approved {
		t.Fatalf("expected unapproveall to clear approvals")
	}
}

func TestCleanCommandTypeSelection(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100717, Type: "supergroup", Title: "Clean Commands"}

	for idx, text := range []string{
		"/cleancommand user other",
		"/cleancommandtypes",
		"/keepcommand other",
		"/keepcommand all",
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
		t.Fatalf("expected keepcommand all to disable all clean command categories")
	}
	if bundle.Settings.CleanCommandUser || bundle.Settings.CleanCommandOther || bundle.Settings.CleanCommandAll {
		t.Fatalf("expected all clean command categories to be disabled, got %+v", bundle.Settings)
	}
	if got := h.Client.Messages[1].Text; !strings.Contains(got, "Clean command types") || !strings.Contains(got, "- user:") {
		t.Fatalf("expected cleancommandtypes response, got %q", got)
	}
}

func TestDisableControlsAffectNonAdminsFirstThenAdmins(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100718, Type: "supergroup", Title: "Disable"}
	h.Client.AdminsByChat[chat.ID] = []telegram.ChatAdministrator{
		{
			User:   telegram.User{ID: 40, FirstName: "Admin"},
			Status: "administrator",
		},
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 100,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/disable report",
		},
	}); err != nil {
		t.Fatalf("disable report failed: %v", err)
	}

	err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 101,
			From:      &telegram.User{ID: 40, FirstName: "Admin"},
			Chat:      chat,
			Text:      "/report",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "reports are disabled") {
		t.Fatalf("expected admin to bypass disabled command before /disableadmin, got %v", err)
	}

	for idx, text := range []string{"/disableadmin on", "/disabledel on", "/disableable"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 3),
			Message: &telegram.Message{
				MessageID: int64(102 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("owner command %q failed: %v", text, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 10,
		Message: &telegram.Message{
			MessageID: 110,
			From:      &telegram.User{ID: 20, FirstName: "User"},
			Chat:      chat,
			Text:      "/report",
		},
	}); err != nil {
		t.Fatalf("disabled report for member failed: %v", err)
	}
	if len(h.Client.DeletedMessages) == 0 || h.Client.DeletedMessages[len(h.Client.DeletedMessages)-1].MessageID != 110 {
		t.Fatalf("expected disabledel to remove member command message, got %+v", h.Client.DeletedMessages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 11,
		Message: &telegram.Message{
			MessageID: 111,
			From:      &telegram.User{ID: 40, FirstName: "Admin"},
			Chat:      chat,
			Text:      "/report",
		},
	}); err != nil {
		t.Fatalf("disabled report for admin failed: %v", err)
	}
	if h.Client.DeletedMessages[len(h.Client.DeletedMessages)-1].MessageID != 111 {
		t.Fatalf("expected /disableadmin to affect admins too, got %+v", h.Client.DeletedMessages)
	}

	last := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(last.Text, "/stopall") || !strings.Contains(last.Text, "/renamefed") {
		t.Fatalf("expected disableable list to include new truthful commands, got %q", last.Text)
	}

	err = h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 12,
		Message: &telegram.Message{
			MessageID: 112,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/disable help",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be disabled") {
		t.Fatalf("expected protected command disable attempt to fail, got %v", err)
	}
}
