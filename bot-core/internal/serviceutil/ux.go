package serviceutil

import (
	"net/url"
	"strings"

	"sukoon/bot-core/internal/telegram"
)

const WebsiteURL = "https://misssukoon.vercel.app/"

func BotURL(username string) string {
	return "https://t.me/" + strings.TrimPrefix(strings.TrimSpace(username), "@")
}

func BotDeepLink(username string, payload string) string {
	link := BotURL(username)
	if strings.TrimSpace(payload) == "" {
		return link
	}
	return link + "?start=" + url.QueryEscape(payload)
}

func BotAddGroupLink(username string) string {
	return BotURL(username) + "?startgroup=true"
}

func Markup(rows ...[]telegram.InlineKeyboardButton) *telegram.InlineKeyboardMarkup {
	keyboard := make([][]telegram.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		keyboard = append(keyboard, row)
	}
	if len(keyboard) == 0 {
		return nil
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}
