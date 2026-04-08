package permissions_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"sukoon/bot-core/internal/permissions"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestOwnerGetsElevatedPermissions(t *testing.T) {
	store := testsupport.NewMemoryStore()
	client := testsupport.NewFakeTelegramClient()
	_, _ = store.UpsertPrimaryBot(context.Background(), testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil))).Bot, []int64{1})

	service := permissions.New(store)
	perms, err := service.Load(context.Background(), "bot-1", 1, -100123, "supergroup", client)
	if err != nil {
		t.Fatalf("load permissions: %v", err)
	}
	if !perms.IsOwner || !perms.CanRestrictMembers {
		t.Fatalf("expected owner permissions, got %+v", perms)
	}
}

func TestChatAdminPermissionsUseTelegramAdmins(t *testing.T) {
	store := testsupport.NewMemoryStore()
	client := testsupport.NewFakeTelegramClient()
	bot := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil))).Bot
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, nil)
	client.AdminsByChat[-100123] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 99},
			Status:             "administrator",
			CanRestrictMembers: true,
			CanDeleteMessages:  true,
		},
	}

	service := permissions.New(store)
	perms, err := service.Load(context.Background(), bot.ID, 99, -100123, "supergroup", client)
	if err != nil {
		t.Fatalf("load permissions: %v", err)
	}
	if !perms.IsChatAdmin || !perms.CanDeleteMessages || !perms.CanRestrictMembers {
		t.Fatalf("expected admin permissions, got %+v", perms)
	}
}

func TestChatAdminPermissionsReuseCachedAdminList(t *testing.T) {
	store := testsupport.NewMemoryStore()
	client := testsupport.NewFakeTelegramClient()
	bot := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil))).Bot
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, nil)
	client.AdminsByChat[-100123] = []telegram.ChatAdministrator{
		{
			User:               telegram.User{ID: 99},
			Status:             "administrator",
			CanRestrictMembers: true,
			CanDeleteMessages:  true,
		},
	}

	service := permissions.New(store)
	for i := 0; i < 2; i++ {
		perms, err := service.Load(context.Background(), bot.ID, 99, -100123, "supergroup", client)
		if err != nil {
			t.Fatalf("load permissions %d: %v", i, err)
		}
		if !perms.IsChatAdmin {
			t.Fatalf("expected admin permissions on iteration %d, got %+v", i, perms)
		}
	}
	if len(client.AdminLookups) != 1 {
		t.Fatalf("expected one admin lookup with cache reuse, got %+v", client.AdminLookups)
	}
}

func TestPrivateChatsSkipTelegramAdminLookup(t *testing.T) {
	store := testsupport.NewMemoryStore()
	client := testsupport.NewFakeTelegramClient()
	bot := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil))).Bot
	_, _ = store.UpsertPrimaryBot(context.Background(), bot, nil)

	service := permissions.New(store)
	perms, err := service.Load(context.Background(), bot.ID, 99, 99, "private", client)
	if err != nil {
		t.Fatalf("load permissions: %v", err)
	}
	if perms.IsChatAdmin || perms.CanDeleteMessages || perms.CanRestrictMembers {
		t.Fatalf("expected private chat without elevated permissions, got %+v", perms)
	}
	if len(client.AdminLookups) != 0 {
		t.Fatalf("expected no admin lookups in private chat, got %v", client.AdminLookups)
	}
}
