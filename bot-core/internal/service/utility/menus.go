package utility

import (
	"fmt"
	"strings"

	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

const (
	callbackStartHome      = "ux:start:home"
	callbackHelpMain       = "ux:help:main"
	callbackHelpModeration = "ux:help:moderation"
	callbackHelpAdmin      = "ux:help:admin"
	callbackHelpProtection = "ux:help:protection"
	callbackHelpContent    = "ux:help:content"
	callbackHelpUtility    = "ux:help:utility"
	callbackHelpAdvanced   = "ux:help:advanced"
	callbackPrivacy        = "ux:privacy"
	callbackClose          = "ux:close"
	callbackRulesShow      = "ux:rules:show"
)

type helpPage struct {
	Title string
	Lines []string
}

var helpPages = map[string]helpPage{
	"moderation": {
		Title: "Moderation",
		Lines: []string{
			"Core actions for dealing with problem users.",
			"",
			"/ban, /unban, /tban",
			"/mute, /unmute, /tmute, /smute, /dmute",
			"/kick, /dkick, /skick",
			"/warn, /warns, /resetwarns",
			"/setwarnlimit, /setwarnmode",
			"",
			"Tip: reply to the target user's message for the cleanest workflow.",
		},
	},
	"admin": {
		Title: "Admin",
		Lines: []string{
			"Chat configuration and day-to-day admin workflows.",
			"",
			"/approve, /unapprove, /approved",
			"/disable, /enable, /disabled",
			"/logchannel",
			"/reports, /report",
			"/cleancommands",
			"/cleanservice, /nocleanservice, /cleanservicetypes",
			"/purge, /del",
			"/pin, /unpin, /unpinall",
			"/mod, /unmod, /muter, /unmuter, /mods",
		},
	},
	"protection": {
		Title: "Protection",
		Lines: []string{
			"Automated moderation and safety controls.",
			"",
			"/lock, /unlock, /locks",
			"/addblocklist, /rmbl, /blocklist",
			"/setflood, /setfloodmode",
			"/captcha",
			"/antiabuse",
			"/antibio, /free, /unfree, /freelist",
		},
	},
	"content": {
		Title: "Content",
		Lines: []string{
			"Saved content, rules, and member-facing group messages.",
			"",
			"/save, /get, /clear",
			"/filter, /stop",
			"/welcome, /goodbye",
			"/setrules, /rules",
			"",
			"Notes and filters already support structured inline buttons.",
		},
	},
	"utility": {
		Title: "Utility",
		Lines: []string{
			"Personal and informational commands.",
			"",
			"/start, /help",
			"/setlang, /language",
			"/privacy, /mydata, /forgetme confirm",
			"/afk",
		},
	},
	"advanced": {
		Title: "Advanced",
		Lines: []string{
			"Owner and multi-chat tooling. Some commands require owner or sudo access.",
			"",
			"/broadcast, /stats",
			"/gban, /ungban",
			"/bluser, /unbluser, /blchat, /unblchat",
			"/addsudo, /rmsudo",
			"/newfed, /delfed, /joinfed, /leavefed",
			"/fedinfo, /fedadmins, /myfeds",
			"/fedpromote, /feddemote, /fban, /unfban, /fedtransfer",
			"/clone, /clone sync, /clones, /rmclone",
		},
	},
}

func startLandingText() string {
	return strings.Join([]string{
		"Sukoon",
		"",
		"Fast moderation, cleaner group management, and safer admin workflows for Telegram groups.",
		"",
		"Browse the real command set, open the website, manage your privacy settings, or add Sukoon to another group.",
	}, "\n")
}

func helpLandingText() string {
	return strings.Join([]string{
		"Sukoon Help",
		"",
		"Choose a category to browse the commands that Sukoon actually supports in this build.",
		"",
		"Main moderation and admin actions work best by replying to the target user's message inside a group.",
	}, "\n")
}

func helpPageText(section string) string {
	page, ok := helpPages[section]
	if !ok {
		return helpLandingText()
	}
	lines := append([]string{page.Title, ""}, page.Lines...)
	return strings.Join(lines, "\n")
}

func privacyText() string {
	return strings.Join([]string{
		"Privacy",
		"",
		"Sukoon stores only the operational data it needs for moderation, automation, safety, and owner-requested workflows.",
		"",
		"Use /mydata to export your stored data and /forgetme confirm to delete eligible personal data for this bot instance.",
	}, "\n")
}

func rulesText(chatTitle string, rules string) string {
	header := "Rules"
	if strings.TrimSpace(chatTitle) != "" {
		header = fmt.Sprintf("Rules for %s", chatTitle)
	}
	return strings.Join([]string{header, "", strings.TrimSpace(rules)}, "\n")
}

func startLandingMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Help", CallbackData: callbackHelpMain},
			{Text: "Website", URL: serviceutil.WebsiteURL},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
			{Text: "Privacy", CallbackData: callbackPrivacy},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func helpLandingMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Moderation", CallbackData: callbackHelpModeration},
			{Text: "Admin", CallbackData: callbackHelpAdmin},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Protection", CallbackData: callbackHelpProtection},
			{Text: "Content", CallbackData: callbackHelpContent},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Utility", CallbackData: callbackHelpUtility},
			{Text: "Advanced", CallbackData: callbackHelpAdvanced},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func helpSectionMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: callbackHelpMain},
			{Text: "Close", CallbackData: callbackClose},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
		},
	)
}

func privacyMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Help", CallbackData: callbackHelpMain},
			{Text: "Website", URL: serviceutil.WebsiteURL},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func pmGuidanceMarkup(username string, payload string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Open PM", URL: serviceutil.BotURL(username)},
			{Text: "Help", URL: serviceutil.BotDeepLink(username, payload)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func rulesGroupMarkup(username string, chatID int64) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Open PM", URL: serviceutil.BotDeepLink(username, fmt.Sprintf("rules_%d", chatID))},
			{Text: "Show Here", CallbackData: callbackRulesShow},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Help", URL: serviceutil.BotDeepLink(username, "help_main")},
			{Text: "Website", URL: serviceutil.WebsiteURL},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func rulesShownHereMarkup(username string, chatID int64) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Open PM", URL: serviceutil.BotDeepLink(username, fmt.Sprintf("rules_%d", chatID))},
			{Text: "Help", URL: serviceutil.BotDeepLink(username, "help_main")},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func rulesPMMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Help", CallbackData: callbackHelpMain},
			{Text: "Website", URL: serviceutil.WebsiteURL},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func normalizeHelpSection(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "mod", "moderation":
		return "moderation"
	case "admin", "admins":
		return "admin"
	case "protection", "antispam", "security":
		return "protection"
	case "content", "notes", "filters", "rules":
		return "content"
	case "utility", "info":
		return "utility"
	case "advanced", "owner", "federation", "clone", "clones":
		return "advanced"
	default:
		return ""
	}
}
