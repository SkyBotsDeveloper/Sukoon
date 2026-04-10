package antispam_test

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
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

func TestAntiRaidCommandsAndJoinEnforcement(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100303, Type: "supergroup", Title: "Raid Room"}

	for idx, text := range []string{
		"/raidtime 6h",
		"/raidactiontime 3h",
		"/autoantiraid 15",
		"/antiraid 3h",
		"/antiraid",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(100 + idx),
			Message: &telegram.Message{
				MessageID: int64(1000 + idx),
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
	if bundle.AntiRaid.RaidDurationSeconds != 6*60*60 || bundle.AntiRaid.ActionDurationSeconds != 3*60*60 || bundle.AntiRaid.AutoThreshold != 15 {
		t.Fatalf("expected configured antiraid settings, got %+v", bundle.AntiRaid)
	}
	if bundle.AntiRaid.EnabledUntil == nil {
		t.Fatalf("expected antiraid to be enabled")
	}
	if got := h.Client.Messages[len(h.Client.Messages)-1].Text; !strings.Contains(got, "AntiRaid settings:") || !strings.Contains(got, "Tempban duration: 3h") {
		t.Fatalf("unexpected /antiraid status response: %q", got)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 200,
		Message: &telegram.Message{
			MessageID: 2000,
			From:      &telegram.User{ID: 30, FirstName: "Joiner"},
			Chat:      chat,
			NewChatMembers: []telegram.User{
				{ID: 30, FirstName: "Joiner"},
			},
		},
	}); err != nil {
		t.Fatalf("join handling failed: %v", err)
	}

	if len(h.Client.Bans) == 0 {
		t.Fatalf("expected antiraid ban for new join")
	}
	if h.Client.Bans[len(h.Client.Bans)-1].UserID != 30 {
		t.Fatalf("expected antiraid ban target 30, got %+v", h.Client.Bans[len(h.Client.Bans)-1])
	}
	if h.Client.Bans[len(h.Client.Bans)-1].Until == nil {
		t.Fatalf("expected temporary antiraid ban")
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 201,
		Message: &telegram.Message{
			MessageID: 2001,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/antiraid off",
		},
	}); err != nil {
		t.Fatalf("disable antiraid failed: %v", err)
	}

	bundle, err = h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle after disable failed: %v", err)
	}
	if bundle.AntiRaid.EnabledUntil != nil {
		t.Fatalf("expected antiraid to be disabled, got %+v", bundle.AntiRaid)
	}
}

func TestAutoAntiRaidTriggersAfterJoinBurst(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100304, Type: "supergroup", Title: "Auto Raid"}

	for idx, text := range []string{
		"/raidtime 2h",
		"/raidactiontime 1h",
		"/autoantiraid 2",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(300 + idx),
			Message: &telegram.Message{
				MessageID: int64(3000 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	for i, userID := range []int64{41, 42, 43} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(400 + i),
			Message: &telegram.Message{
				MessageID: int64(4000 + i),
				From:      &telegram.User{ID: userID, FirstName: "Joiner"},
				Chat:      chat,
				NewChatMembers: []telegram.User{
					{ID: userID, FirstName: "Joiner"},
				},
			},
		}); err != nil {
			t.Fatalf("join %d failed: %v", i, err)
		}
	}

	if len(h.Client.Bans) == 0 {
		t.Fatalf("expected auto antiraid ban")
	}
	if h.Client.Bans[len(h.Client.Bans)-1].UserID != 43 {
		t.Fatalf("expected third join to trigger auto antiraid, got %+v", h.Client.Bans)
	}
	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if bundle.AntiRaid.EnabledUntil == nil {
		t.Fatalf("expected auto antiraid to become active")
	}
	if !strings.Contains(h.Client.Messages[len(h.Client.Messages)-1].Text, "AntiRaid auto-enabled") {
		t.Fatalf("expected auto antiraid notification, got %q", h.Client.Messages[len(h.Client.Messages)-1].Text)
	}
}

func TestLockCommandsSupportModesAllowlistAndListing(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100305, Type: "supergroup", Title: "Locks"}

	for idx, text := range []string{
		"/lockwarns on",
		"/lock sticker photo gif video",
		"/lock invitelink ### no promoting other chats {ban}",
		"/allowlist @channelusername t.me/addstickers/FriendlyPack /start",
		"/locks list",
		"/locktypes",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(500 + idx),
			Message: &telegram.Message{
				MessageID: int64(5000 + idx),
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
	if !bundle.Settings.LockWarns {
		t.Fatalf("expected lockwarns to persist")
	}
	for _, lockType := range []string{"sticker", "photo", "gif", "video"} {
		if _, ok := bundle.Locks[lockType]; !ok {
			t.Fatalf("expected %s lock to be active, got %+v", lockType, bundle.Locks)
		}
	}
	inviteLock, ok := bundle.Locks["invitelink"]
	if !ok {
		t.Fatalf("expected invitelink lock to exist, got %+v", bundle.Locks)
	}
	if inviteLock.Action != "ban" || inviteLock.Reason != "no promoting other chats" {
		t.Fatalf("expected custom invitelink lock action/reason, got %+v", inviteLock)
	}
	if !slices.Contains(bundle.LockAllowlist, "@channelusername") || !slices.Contains(bundle.LockAllowlist, "stickerpack:friendlypack") || !slices.Contains(bundle.LockAllowlist, "/start") {
		t.Fatalf("expected allowlist entries to persist, got %+v", bundle.LockAllowlist)
	}

	locksStatus := h.Client.Messages[4].Text
	if !strings.Contains(locksStatus, "- Lock warnings: on") || !strings.Contains(locksStatus, "- invitelink: on (ban) - no promoting other chats") {
		t.Fatalf("unexpected /locks list output: %q", locksStatus)
	}
	lockTypes := h.Client.Messages[5].Text
	if !strings.Contains(lockTypes, "stickerpremium") || !strings.Contains(lockTypes, "url") || !strings.Contains(lockTypes, "all") {
		t.Fatalf("unexpected /locktypes output: %q", lockTypes)
	}
}

func TestLockWarningsAndAllowlistsAffectEnforcement(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100306, Type: "supergroup", Title: "Lock Match"}

	for idx, text := range []string{
		"/lockwarns on",
		"/lock url",
		"/allowlist example.com",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(600 + idx),
			Message: &telegram.Message{
				MessageID: int64(6000 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	deletedBefore := len(h.Client.DeletedMessages)
	messagesBefore := len(h.Client.Messages)

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 700,
		Message: &telegram.Message{
			MessageID: 7000,
			From:      &telegram.User{ID: 20, FirstName: "Allowed"},
			Chat:      chat,
			Text:      "read https://example.com/docs first",
		},
	}); err != nil {
		t.Fatalf("allowed url message failed: %v", err)
	}
	if len(h.Client.DeletedMessages) != deletedBefore || len(h.Client.Messages) != messagesBefore {
		t.Fatalf("expected allowlisted url to bypass locks, deleted=%+v messages=%+v", h.Client.DeletedMessages, h.Client.Messages)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 701,
		Message: &telegram.Message{
			MessageID: 7001,
			From:      &telegram.User{ID: 20, FirstName: "Blocked"},
			Chat:      chat,
			Text:      "read https://bad.example right now",
		},
	}); err != nil {
		t.Fatalf("blocked url message failed: %v", err)
	}
	if len(h.Client.DeletedMessages) != deletedBefore+1 || h.Client.DeletedMessages[len(h.Client.DeletedMessages)-1].MessageID != 7001 {
		t.Fatalf("expected bad url to be deleted, got %+v", h.Client.DeletedMessages)
	}
	lastMessage := h.Client.Messages[len(h.Client.Messages)-1].Text
	if !strings.Contains(lastMessage, "1 warning") || !strings.Contains(lastMessage, "Locked content: url") {
		t.Fatalf("expected lock warning response, got %q", lastMessage)
	}
}
