package utility

import (
	"fmt"
	"strings"

	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

const (
	callbackStartHome        = "ux:start:home"
	callbackHelpMain         = "ux:help:main"
	callbackHelpModeration   = "ux:help:moderation"
	callbackHelpWarnings     = "ux:help:warnings"
	callbackHelpApprovals    = "ux:help:approvals"
	callbackHelpAdmin        = "ux:help:admin"
	callbackHelpCleanup      = "ux:help:cleanup"
	callbackHelpProtection   = "ux:help:protection"
	callbackHelpNotesFilters = "ux:help:notesfilters"
	callbackHelpRulesWelcome = "ux:help:ruleswelcome"
	callbackHelpUtility      = "ux:help:utility"
	callbackHelpOwner        = "ux:help:owner"
	callbackHelpFederation   = "ux:help:federation"
	callbackHelpClones       = "ux:help:clones"
	callbackPrivacy          = "ux:privacy"
	callbackClose            = "ux:close"
	callbackRulesShow        = "ux:rules:show"
)

type helpPage struct {
	Title    string
	Callback string
	Lines    []string
}

var helpPages = map[string]helpPage{
	"moderation": {
		Title:    "Moderation",
		Callback: callbackHelpModeration,
		Lines: []string{
			"Direct actions for handling problem users.",
			"",
			"/ban, /unban, /tban",
			"/mute, /unmute, /tmute, /smute, /dmute",
			"/kick, /dkick, /skick",
			"",
			"Best workflow: reply to the target user's message before running the command.",
		},
	},
	"warnings": {
		Title:    "Warnings",
		Callback: callbackHelpWarnings,
		Lines: []string{
			"Warning-based moderation and escalation policy.",
			"",
			"/warn, /warns, /resetwarns",
			"/setwarnlimit",
			"/setwarnmode",
			"",
			"Warn mode supports mute, kick, or ban after the configured limit.",
		},
	},
	"approvals": {
		Title:    "Approvals",
		Callback: callbackHelpApprovals,
		Lines: []string{
			"Allow trusted users to bypass selected protection checks.",
			"",
			"/approve, /unapprove, /approved",
			"",
			"Approved users bypass blocklist and antibio checks in the current build.",
		},
	},
	"admin": {
		Title:    "Admin",
		Callback: callbackHelpAdmin,
		Lines: []string{
			"Core admin control surfaces and group visibility tools.",
			"",
			"/admins, /adminlist",
			"/disable, /enable, /disabled",
			"/logchannel",
			"/reports, /report",
			"/mod, /unmod, /muter, /unmuter, /mods",
			"",
			"Silent mod powers are internal Sukoon roles, not Telegram admin promotion.",
		},
	},
	"cleanup": {
		Title:    "Cleanup",
		Callback: callbackHelpCleanup,
		Lines: []string{
			"Chat cleanup, service cleanup, and pin management.",
			"",
			"/cleancommands",
			"/cleanservice, /nocleanservice, /cleanservicetypes",
			"/purge, /del",
			"/pin, /unpin, /unpinall",
		},
	},
	"protection": {
		Title:    "Protection",
		Callback: callbackHelpProtection,
		Lines: []string{
			"Automated protection, anti-spam, and verification.",
			"",
			"/lock, /unlock, /locks",
			"/addblocklist, /rmbl, /blocklist",
			"/setflood, /setfloodmode",
			"/captcha",
			"/antiabuse",
			"/antibio, /free, /unfree, /freelist",
		},
	},
	"notesfilters": {
		Title:    "Notes And Filters",
		Callback: callbackHelpNotesFilters,
		Lines: []string{
			"Saved responses, note retrieval, and trigger-based replies.",
			"",
			"/save, /notes, /saved",
			"/get, /clear",
			"/filter, /filters, /stop",
			"",
			"Structured inline buttons are supported in saved notes and filter responses.",
		},
	},
	"ruleswelcome": {
		Title:    "Rules And Greetings",
		Callback: callbackHelpRulesWelcome,
		Lines: []string{
			"Rules, welcome flows, and goodbye messaging.",
			"",
			"/setrules, /resetrules, /rules",
			"/welcome, /setwelcome",
			"/goodbye, /setgoodbye",
			"",
			"Group /rules prompts users toward PM while still offering a show-here button.",
		},
	},
	"utility": {
		Title:    "Utility And Privacy",
		Callback: callbackHelpUtility,
		Lines: []string{
			"Start/help entry points, language, AFK, and personal-data controls.",
			"",
			"/start, /help",
			"/setlang, /language",
			"/privacy, /mydata, /forgetme confirm",
			"/afk",
			"",
			"Privacy-sensitive flows are PM-first by design.",
		},
	},
	"owner": {
		Title:    "Owner And Global",
		Callback: callbackHelpOwner,
		Lines: []string{
			"Owner or sudo tooling with durable job-backed fanout where required.",
			"",
			"/broadcast",
			"/stats",
			"/gban, /ungban",
			"/bluser, /unbluser, /blchat, /unblchat",
			"/addsudo, /rmsudo",
		},
	},
	"federation": {
		Title:    "Federation",
		Callback: callbackHelpFederation,
		Lines: []string{
			"Federation lifecycle, admin assignment, and federation-wide bans.",
			"",
			"/newfed, /delfed",
			"/joinfed, /leavefed",
			"/fedinfo, /fedadmins, /myfeds",
			"/fedpromote, /feddemote",
			"/fban, /unfban, /fedtransfer",
		},
	},
	"clones": {
		Title:    "Clones",
		Callback: callbackHelpClones,
		Lines: []string{
			"Operator flows for bot-instance lifecycle and webhook sync.",
			"",
			"/clone",
			"/clone sync",
			"/clones",
			"/rmclone",
		},
	},
}

func startLandingText() string {
	return strings.Join([]string{
		"Sukoon",
		"",
		"Fast moderation, cleaner group management, and safer admin workflows for Telegram groups.",
		"",
		"Use the buttons below to browse live command sections, open the website, or add Sukoon to another group.",
	}, "\n")
}

func helpLandingText() string {
	return strings.Join([]string{
		"Sukoon Help",
		"",
		"Browse implemented command families using the sections below.",
		"",
		"Moderation works best by replying to the target user. PM-first flows are used where privacy or cleaner navigation matters.",
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
			{Text: "Moderation", CallbackData: callbackHelpModeration},
			{Text: "Protection", CallbackData: callbackHelpProtection},
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
			{Text: "Warnings", CallbackData: callbackHelpWarnings},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Approvals", CallbackData: callbackHelpApprovals},
			{Text: "Admin", CallbackData: callbackHelpAdmin},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Cleanup", CallbackData: callbackHelpCleanup},
			{Text: "Protection", CallbackData: callbackHelpProtection},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Notes & Filters", CallbackData: callbackHelpNotesFilters},
			{Text: "Rules & Welcome", CallbackData: callbackHelpRulesWelcome},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Utility", CallbackData: callbackHelpUtility},
			{Text: "Owner", CallbackData: callbackHelpOwner},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Federation", CallbackData: callbackHelpFederation},
			{Text: "Clones", CallbackData: callbackHelpClones},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Home", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func helpSectionMarkup(username string, backCallback string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: backCallback},
			{Text: "Home", CallbackData: callbackStartHome},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func privacyMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Help", CallbackData: callbackHelpUtility},
			{Text: "Website", URL: serviceutil.WebsiteURL},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Home", CallbackData: callbackStartHome},
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
			{Text: "Help", URL: serviceutil.BotDeepLink(username, "help_ruleswelcome")},
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
			{Text: "Help", URL: serviceutil.BotDeepLink(username, "help_ruleswelcome")},
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
			{Text: "Help", CallbackData: callbackHelpRulesWelcome},
			{Text: "Website", URL: serviceutil.WebsiteURL},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Home", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func normalizeHelpSection(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "mod", "moderation", "ban", "bans", "mute", "mutes", "kick", "kicks":
		return "moderation"
	case "warn", "warnings", "warns":
		return "warnings"
	case "approve", "approvals", "approved":
		return "approvals"
	case "admin", "admins", "report", "reports", "log", "logs", "staff", "silent":
		return "admin"
	case "cleanup", "clean", "purge", "pin", "pins":
		return "cleanup"
	case "protection", "antispam", "security", "locks", "lock", "blocklist", "flood", "captcha", "antiabuse", "antibio":
		return "protection"
	case "content", "notes", "note", "filters", "filter", "saved":
		return "notesfilters"
	case "rules", "welcome", "goodbye", "greetings":
		return "ruleswelcome"
	case "utility", "info", "language", "privacy", "afk", "start", "help":
		return "utility"
	case "owner", "sudo", "global", "broadcast", "stats":
		return "owner"
	case "federation", "fed", "feds":
		return "federation"
	case "clone", "clones", "instance", "instances":
		return "clones"
	default:
		return ""
	}
}
