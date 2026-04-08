package moderation_test

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

func TestWarnLimitAndModeEscalateToBan(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100200, Type: "supergroup", Title: "Warns"}

	for _, text := range []string{"/setwarnlimit 2", "/setwarnmode ban"} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(len(h.Client.Messages) + 1),
			Message: &telegram.Message{
				MessageID: int64(len(h.Client.Messages) + 10),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	for i := 0; i < 2; i++ {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(100 + i),
			Message: &telegram.Message{
				MessageID: int64(1000 + i),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      "/warn rude",
				ReplyToMessage: &telegram.Message{
					MessageID: 99,
					From:      &telegram.User{ID: 20, FirstName: "Target"},
					Chat:      chat,
				},
			},
		}); err != nil {
			t.Fatalf("warn failed: %v", err)
		}
	}

	if len(h.Client.Bans) != 1 || h.Client.Bans[0].UserID != 20 {
		t.Fatalf("expected warn escalation ban, got %+v", h.Client.Bans)
	}
}

func TestBanAliasesAndKickMe(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100201, Type: "supergroup", Title: "Ban Aliases"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 20,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/dban spam links",
			ReplyToMessage: &telegram.Message{
				MessageID: 19,
				From:      &telegram.User{ID: 20, FirstName: "Target"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("dban failed: %v", err)
	}

	if len(h.Client.Bans) == 0 || h.Client.Bans[0].UserID != 20 {
		t.Fatalf("expected dban to ban target, got %+v", h.Client.Bans)
	}
	if len(h.Client.DeletedMessages) == 0 || h.Client.DeletedMessages[0].MessageID != 19 {
		t.Fatalf("expected dban to delete the replied message, got %+v", h.Client.DeletedMessages)
	}

	messageCount := len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 21,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/sban spam links",
			ReplyToMessage: &telegram.Message{
				MessageID: 18,
				From:      &telegram.User{ID: 21, FirstName: "Target2"},
				Chat:      chat,
			},
		},
	}); err != nil {
		t.Fatalf("sban failed: %v", err)
	}

	if len(h.Client.Bans) < 2 || h.Client.Bans[1].UserID != 21 {
		t.Fatalf("expected sban to ban second target, got %+v", h.Client.Bans)
	}
	if len(h.Client.DeletedMessages) < 3 {
		t.Fatalf("expected sban to delete both the replied and command messages, got %+v", h.Client.DeletedMessages)
	}
	if h.Client.DeletedMessages[1].MessageID != 18 || h.Client.DeletedMessages[2].MessageID != 21 {
		t.Fatalf("expected sban cleanup order reply then command, got %+v", h.Client.DeletedMessages)
	}
	if len(h.Client.Messages) != messageCount {
		t.Fatalf("expected sban to stay silent, got %+v", h.Client.Messages)
	}

	messageCount = len(h.Client.Messages)
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 22,
			From:      &telegram.User{ID: 30, FirstName: "SelfKick"},
			Chat:      chat,
			Text:      "/kickme",
		},
	}); err != nil {
		t.Fatalf("kickme failed: %v", err)
	}

	if len(h.Client.Bans) < 3 || h.Client.Bans[2].UserID != 30 {
		t.Fatalf("expected kickme to ban actor temporarily, got %+v", h.Client.Bans)
	}
	if len(h.Client.Unbans) == 0 || h.Client.Unbans[len(h.Client.Unbans)-1].UserID != 30 {
		t.Fatalf("expected kickme to unban actor after temporary kick, got %+v", h.Client.Unbans)
	}
	if len(h.Client.Messages) != messageCount+1 {
		t.Fatalf("expected kickme confirmation message, got %+v", h.Client.Messages)
	}
}

func TestDeleteByReplyVariantsRequireReplyAndDeleteTargetMessage(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100202, Type: "supergroup", Title: "Delete Variants"}

	for _, tc := range []struct {
		updateID   int64
		text       string
		targetID   int64
		replyMsgID int64
		expectBan  bool
		expectKick bool
		expectMute bool
	}{
		{updateID: 1, text: "/dmute flood", targetID: 30, replyMsgID: 1001, expectMute: true},
		{updateID: 2, text: "/dkick flood", targetID: 31, replyMsgID: 1002, expectKick: true},
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: tc.updateID,
			Message: &telegram.Message{
				MessageID: 2000 + tc.updateID,
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      tc.text,
				ReplyToMessage: &telegram.Message{
					MessageID: tc.replyMsgID,
					From:      &telegram.User{ID: tc.targetID, FirstName: "Target"},
					Chat:      chat,
				},
			},
		}); err != nil {
			t.Fatalf("%s failed: %v", tc.text, err)
		}
	}

	if len(h.Client.Restrictions) == 0 || h.Client.Restrictions[0].UserID != 30 || h.Client.Restrictions[0].Permissions.CanSendMessages {
		t.Fatalf("expected dmute to mute the replied target, got %+v", h.Client.Restrictions)
	}
	if len(h.Client.Bans) == 0 || h.Client.Bans[0].UserID != 31 {
		t.Fatalf("expected dkick to ban target temporarily, got %+v", h.Client.Bans)
	}
	if len(h.Client.Unbans) == 0 || h.Client.Unbans[0].UserID != 31 {
		t.Fatalf("expected dkick to unban target after kick, got %+v", h.Client.Unbans)
	}
	if len(h.Client.DeletedMessages) < 2 || h.Client.DeletedMessages[0].MessageID != 1001 || h.Client.DeletedMessages[1].MessageID != 1002 {
		t.Fatalf("expected delete variants to remove replied messages, got %+v", h.Client.DeletedMessages)
	}

	for _, text := range []string{"/dban 50", "/dmute 50", "/dkick 50"} {
		err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: time.Now().UnixNano(),
			Message: &telegram.Message{
				MessageID: 9000,
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		})
		if err == nil || !strings.Contains(err.Error(), "reply to a user's message") {
			t.Fatalf("expected %s to require reply, got %v", text, err)
		}
	}
}

func TestTimedModerationSupportsDaysAndWeeks(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100203, Type: "supergroup", Title: "Timed Moderation"}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 301,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/tban 40 6d cooldown",
		},
	}); err != nil {
		t.Fatalf("tban failed: %v", err)
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 302,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/tmute 41 5w cooldown",
		},
	}); err != nil {
		t.Fatalf("tmute failed: %v", err)
	}

	if len(h.Client.Bans) == 0 || h.Client.Bans[0].Until == nil {
		t.Fatalf("expected tban to set an until date, got %+v", h.Client.Bans)
	}
	if len(h.Client.Restrictions) == 0 || h.Client.Restrictions[0].Until == nil {
		t.Fatalf("expected tmute to set an until date, got %+v", h.Client.Restrictions)
	}

	banDuration := h.Client.Bans[0].Until.Sub(time.Now())
	if banDuration < (6*24*time.Hour-time.Hour) || banDuration > (6*24*time.Hour+time.Hour) {
		t.Fatalf("expected tban duration near 6d, got %s", banDuration)
	}
	muteDuration := h.Client.Restrictions[0].Until.Sub(time.Now())
	if muteDuration < (5*7*24*time.Hour-time.Hour) || muteDuration > (5*7*24*time.Hour+time.Hour) {
		t.Fatalf("expected tmute duration near 5w, got %s", muteDuration)
	}

	if got := h.Client.Messages[len(h.Client.Messages)-2].Text; !strings.Contains(got, "for 6d") {
		t.Fatalf("expected tban response to humanize 6d, got %q", got)
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "for 5w") {
		t.Fatalf("expected tmute response to humanize 5w, got %q", got)
	}
}
