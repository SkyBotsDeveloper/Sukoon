package commands_test

import (
	"testing"

	"sukoon/bot-core/internal/commands"
)

func TestParseCommandForMatchingBot(t *testing.T) {
	parsed, ok := commands.Parse("/ban@Sukoon_Bot 123 spam", "sukoon_bot")
	if !ok {
		t.Fatalf("expected command to parse")
	}
	if parsed.Name != "ban" {
		t.Fatalf("expected ban, got %q", parsed.Name)
	}
	if parsed.RawArgs != "123 spam" {
		t.Fatalf("unexpected raw args: %q", parsed.RawArgs)
	}
}

func TestParseRejectsDifferentBotUsername(t *testing.T) {
	if _, ok := commands.Parse("/ban@otherbot 123", "sukoon_bot"); ok {
		t.Fatalf("expected parse to reject different bot username")
	}
}
