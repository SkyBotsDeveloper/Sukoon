package migrations_test

import (
	"strings"
	"testing"

	"sukoon/bot-core/migrations"
)

func TestCanonicalSchemaContainsCriticalTables(t *testing.T) {
	body, err := migrations.Files.ReadFile("0001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS bot_instances",
		"CREATE TABLE IF NOT EXISTS chat_settings",
		"CREATE TABLE IF NOT EXISTS moderation_settings",
		"CREATE TABLE IF NOT EXISTS captcha_settings",
		"CREATE TABLE IF NOT EXISTS captcha_challenges",
		"CREATE TABLE IF NOT EXISTS telegram_updates",
		"CREATE TABLE IF NOT EXISTS jobs",
		"CREATE TABLE IF NOT EXISTS federations",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected migration to contain %q", fragment)
		}
	}
}
