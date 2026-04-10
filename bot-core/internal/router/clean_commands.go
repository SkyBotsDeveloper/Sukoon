package router

import (
	"strings"

	"sukoon/bot-core/internal/domain"
)

var cleanUserCommands = map[string]struct{}{
	"start":    {},
	"help":     {},
	"donate":   {},
	"privacy":  {},
	"mydata":   {},
	"forgetme": {},
	"report":   {},
	"approval": {},
	"warns":    {},
	"kickme":   {},
	"get":      {},
	"notes":    {},
	"saved":    {},
	"rules":    {},
	"afk":      {},
}

func shouldCleanHandledCommand(name string, settings domain.ChatSettings) bool {
	if !settings.CleanCommands {
		return false
	}
	category := "admin"
	if _, ok := cleanUserCommands[name]; ok {
		category = "user"
	}
	return settings.CleanCommandCategoryEnabled(category)
}

func shouldCleanUnhandledCommandMessage(text string, settings domain.ChatSettings) bool {
	if !settings.CleanCommandCategoryEnabled("other") {
		return false
	}
	text = strings.TrimSpace(text)
	return text != "" && strings.HasPrefix(text, "/")
}
