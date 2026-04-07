package utility

import (
	"fmt"
	"strings"

	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

const (
	callbackStartHome  = "ux:start:home"
	callbackHelpPrefix = "ux:help:"
	callbackHelpMain   = callbackHelpPrefix + "root"
	callbackPrivacy    = "ux:privacy"
	callbackClose      = "ux:close"
	callbackRulesShow  = "ux:rules:show"

	helpRoot              = "root"
	helpAdmin             = "admin"
	helpAntiflood         = "antiflood"
	helpApproval          = "approval"
	helpBans              = "bans"
	helpBlocklists        = "blocklists"
	helpBlocklistExamples = "blocklists_examples"
	helpCaptcha           = "captcha"
	helpCleanCommands     = "cleancommands"
	helpLocks             = "locks"
	helpLockDescriptions  = "locks_descriptions"
	helpLockExamples      = "locks_examples"
	helpLogChannels       = "logchannels"
)

type helpPage struct {
	Title string
	Lines []string
}

var helpPages = map[string]helpPage{
	helpAdmin: {
		Title: "Admin",
		Lines: []string{
			"Admin visibility and Sukoon staff workflows.",
			"",
			"/admins, /adminlist",
			"/mods, /mod, /unmod, /muter, /unmuter",
			"",
			"Sukoon loads Telegram admin permissions live on each update, so there is no separate /admincache refresh command in the current build.",
			"Telegram promote/demote and anonymous-admin controls are still deferred.",
		},
	},
	helpAntiflood: {
		Title: "Antiflood",
		Lines: []string{
			"Antiflood tracks bursty users and applies the configured action once they exceed the limit inside the active window.",
			"",
			"/flood",
			"/setflood <count|off>",
			"/setfloodtimer <seconds>",
			"/floodmode [mute|kick|ban]",
			"/clearflood [reply|user]",
			"",
			"Examples:",
			"/setflood 6",
			"/setfloodtimer 10",
			"/floodmode mute",
		},
	},
	helpApproval: {
		Title: "Approval",
		Lines: []string{
			"Approvals mark trusted members so they bypass selected protections without weakening the whole chat.",
			"",
			"/approval <reply|user>",
			"/approve <reply|user>",
			"/unapprove <reply|user>",
			"/approved",
			"/unapproveall",
			"",
			"Approved users currently bypass blocklist and antibio enforcement.",
		},
	},
	helpBans: {
		Title: "Bans And Mutes",
		Lines: []string{
			"Direct moderation actions. Reply to the target user's message whenever possible.",
			"",
			"/kickme",
			"/ban, /dban, /sban, /tban, /unban",
			"/mute, /dmute, /smute, /tmute, /unmute",
			"/kick, /dkick, /skick",
			"",
			"Examples:",
			"/ban spam links",
			"/tban @user 24h repeated abuse",
			"/tmute @user 30m cooldown",
		},
	},
	helpBlocklists: {
		Title: "Blocklists",
		Lines: []string{
			"Blocklists delete matching content automatically. The current build supports word, phrase, and regex patterns with approved-user bypass.",
			"",
			"/addblocklist <word|phrase|regex> <pattern>",
			"/rmbl <pattern> or /rmblocklist <pattern>",
			"/unblocklistall",
			"/blocklist",
			"",
			"Open the examples page below for live command syntax.",
			"Rose-style blocklist mode, delete toggles, and custom blocklist reasons are still deferred.",
		},
	},
	helpBlocklistExamples: {
		Title: "Blocklist Command Examples",
		Lines: []string{
			"Examples for the current Sukoon parser:",
			"",
			"/addblocklist word spam",
			"/addblocklist phrase buy now",
			"/addblocklist regex (?i)free\\s+crypto",
			"/rmblocklist spam | buy now",
			"/unblocklistall",
			"",
			"Bulk removal accepts the same pipe-separated syntax already used elsewhere in Sukoon.",
		},
	},
	helpCaptcha: {
		Title: "CAPTCHA",
		Lines: []string{
			"New members can be challenge-restricted until they solve the current button-based captcha.",
			"",
			"/captcha [on|off]",
			"/captchamode [button]",
			"/captchakick [kick|mute|ban]",
			"/captchakicktime [seconds]",
			"",
			"Current captcha mode is button-only. Rose-style custom captcha text, extra rules text, and mute-duration variants are still deferred.",
		},
	},
	helpCleanCommands: {
		Title: "Clean Commands",
		Lines: []string{
			"Command cleanup removes handled command messages after Sukoon responds, reducing admin clutter.",
			"",
			"/cleancommands [on|off]",
			"/cleancommand [on|off]",
			"/keepcommand",
			"/cleancommandtypes",
			"",
			"Examples:",
			"/cleancommand on",
			"/keepcommand",
		},
	},
	helpLocks: {
		Title: "Locks",
		Lines: []string{
			"Locks delete matching content types automatically for non-admin users.",
			"",
			"/lock <type>",
			"/unlock <type>",
			"/locks",
			"/locktypes",
			"",
			"Open the descriptions and examples pages below for the live lock set.",
			"Warn-mode locks and allowlist commands are still deferred.",
		},
	},
	helpLockDescriptions: {
		Title: "Lock Descriptions",
		Lines: []string{
			"Supported lock types in the current build:",
			"",
			"links: deletes messages containing URLs or t.me links",
			"forwards: deletes forwarded messages",
			"media: deletes photos, videos, documents, and animations",
			"sticker: deletes stickers",
			"gif: deletes animation/GIF posts",
			"",
			"Aliases like url, urls, forward, stickers, gifs, and animations map to the same canonical lock types.",
		},
	},
	helpLockExamples: {
		Title: "Lock Examples",
		Lines: []string{
			"Example commands for the supported lock set:",
			"",
			"/lock links",
			"/lock forwards",
			"/lock media",
			"/lock gifs",
			"/unlock sticker",
			"/locks",
		},
	},
	helpLogChannels: {
		Title: "Log Channels",
		Lines: []string{
			"Log channels receive moderation and protection events outside the main chat.",
			"",
			"/logchannel [chat_id|off]",
			"/setlog <chat_id>",
			"/unsetlog",
			"/log [chat_id|off]",
			"/nolog",
			"/logcategories",
			"",
			"Reports also use the configured log channel when /reports is enabled.",
		},
	},
}

func helpCallback(page string) string {
	return callbackHelpPrefix + page
}

func startLandingText() string {
	return strings.Join([]string{
		"Sukoon",
		"",
		"Fast moderation, cleaner group management, and safer admin workflows for Telegram groups.",
		"",
		"Use the buttons below to browse live help sections, open the website, or add Sukoon to another group.",
	}, "\n")
}

func helpLandingText() string {
	return strings.Join([]string{
		"Sukoon Help",
		"",
		"Browse the currently implemented Rose-style help sections below.",
		"",
		"Only live Sukoon commands are shown. Help navigation edits the same message to avoid spam.",
	}, "\n")
}

func helpPageText(section string) string {
	if section == helpRoot {
		return helpLandingText()
	}
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
			{Text: "Admin", CallbackData: helpCallback(helpAdmin)},
			{Text: "Bans", CallbackData: helpCallback(helpBans)},
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
			{Text: "Admin", CallbackData: helpCallback(helpAdmin)},
			{Text: "Approval", CallbackData: helpCallback(helpApproval)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Bans", CallbackData: helpCallback(helpBans)},
			{Text: "Antiflood", CallbackData: helpCallback(helpAntiflood)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Blocklists", CallbackData: helpCallback(helpBlocklists)},
			{Text: "CAPTCHA", CallbackData: helpCallback(helpCaptcha)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Clean Commands", CallbackData: helpCallback(helpCleanCommands)},
			{Text: "Locks", CallbackData: helpCallback(helpLocks)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Log Channels", CallbackData: helpCallback(helpLogChannels)},
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

func helpSectionMarkup(page string, username string) *telegram.InlineKeyboardMarkup {
	switch page {
	case helpBlocklists:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Examples", CallbackData: helpCallback(helpBlocklistExamples)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
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
	case helpLocks:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Lock Descriptions", CallbackData: helpCallback(helpLockDescriptions)},
				{Text: "Example Commands", CallbackData: helpCallback(helpLockExamples)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
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
	case helpBlocklistExamples:
		return helpSubsectionMarkup(username, helpBlocklists)
	case helpLockDescriptions, helpLockExamples:
		return helpSubsectionMarkup(username, helpLocks)
	default:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
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
}

func helpSubsectionMarkup(username string, parent string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: helpCallback(parent)},
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
			{Text: "Help", CallbackData: callbackHelpMain},
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
			{Text: "Home", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func normalizeHelpSection(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "main", "home", "root":
		return helpRoot
	case "admin", "admins", "adminlist", "mods", "mod", "silent", "staff", "promote", "demote", "admincache", "anonadmin", "adminerror":
		return helpAdmin
	case "antiflood", "flood", "setflood", "setfloodtimer", "floodmode", "clearflood":
		return helpAntiflood
	case "approval", "approve", "approvals", "approved", "unapprove", "unapproveall":
		return helpApproval
	case "bans", "ban", "moderation", "mute", "kick", "kickme", "warn", "warnings":
		return helpBans
	case "protection", "security", "spam":
		return helpRoot
	case "blocklist", "blocklists", "rmblocklist", "rmbl", "unblocklistall":
		return helpBlocklists
	case "blocklists_examples", "blocklistexamples", "blocklist_examples":
		return helpBlocklistExamples
	case "captcha", "captchas", "captchamode", "captchakick", "captchakicktime":
		return helpCaptcha
	case "clean", "cleanup", "cleancommands", "cleancommand", "keepcommand", "cleancommandtypes":
		return helpCleanCommands
	case "locks", "lock", "unlock", "locktypes":
		return helpLocks
	case "locks_descriptions", "lockdescriptions", "lock_descriptions":
		return helpLockDescriptions
	case "locks_examples", "lockexamples", "lock_examples":
		return helpLockExamples
	case "log", "logs", "logchannel", "logchannels", "setlog", "unsetlog", "nolog", "logcategories":
		return helpLogChannels
	case "notesfilters", "notes", "filters", "ruleswelcome", "rules", "utility", "privacy", "owner", "federation", "clones":
		return helpRoot
	default:
		return ""
	}
}
