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

	helpRoot               = "root"
	helpAdmin              = "admin"
	helpAntiflood          = "antiflood"
	helpApproval           = "approval"
	helpBans               = "bans"
	helpBlocklists         = "blocklists"
	helpBlocklistExamples  = "blocklists_examples"
	helpCaptcha            = "captcha"
	helpCleanCommands      = "cleancommands"
	helpDisabling          = "disabling"
	helpFederations        = "federations"
	helpFederationsAdmin   = "federations_admin"
	helpFederationsOwner   = "federations_owner"
	helpFederationsUser    = "federations_user"
	helpFilters            = "filters"
	helpFilterExamples     = "filters_examples"
	helpFormatting         = "formatting"
	helpFormattingMarkdown = "formatting_markdown"
	helpFormattingFillings = "formatting_fillings"
	helpFormattingRandom   = "formatting_random"
	helpFormattingButtons  = "formatting_buttons"
	helpLocks              = "locks"
	helpLockDescriptions   = "locks_descriptions"
	helpLockExamples       = "locks_examples"
	helpLogChannels        = "logchannels"
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
	helpDisabling: {
		Title: "Disabling",
		Lines: []string{
			"Disable command families for normal members without removing the feature from Sukoon entirely.",
			"",
			"/disable <commandname>",
			"/enable <commandname>",
			"/disableable",
			"/disabledel [on|off]",
			"/disableadmin [on|off]",
			"/disabled",
			"",
			"By default, disabled commands affect non-admins only. Enable /disableadmin if chat admins should also be blocked.",
			"Turn on /disabledel to delete disabled command messages instead of replying.",
			"",
			"Examples:",
			"/disable reports",
			"/disabledel on",
			"/disableadmin on",
		},
	},
	helpFederations: {
		Title: "Federations",
		Lines: []string{
			"Federations link multiple chats under one shared moderation namespace.",
			"",
			"Use federation pages below for the live Sukoon command groups: shared bans, federation admins, chat linking, and ownership workflows.",
			"",
			"Only the commands backed by the current Go runtime are shown.",
		},
	},
	helpFederationsAdmin: {
		Title: "Fed Admin Commands",
		Lines: []string{
			"Federation admins manage shared bans across every joined chat.",
			"",
			"/fban <reply|user> [reason]",
			"/unfban <reply|user>",
			"/feddemoteme [federation]",
			"/myfeds",
			"",
			"Use these from a fed-linked chat, or pass a federation short name where the command supports it.",
		},
	},
	helpFederationsOwner: {
		Title: "Federation Owner Commands",
		Lines: []string{
			"Federation owners manage the federation itself and its admin list.",
			"",
			"/newfed <short_name> [display name]",
			"/renamefed <short_name> [display name]",
			"/delfed",
			"/fedtransfer <reply|user>",
			"/fedpromote <reply|user>",
			"/feddemote <reply|user>",
			"",
			"Fed notifications, subscriptions, import/export, fed logs, and fed language settings are still deferred.",
		},
	},
	helpFederationsUser: {
		Title: "Federation User Commands",
		Lines: []string{
			"These commands inspect or link the current chat to a federation.",
			"",
			"/fedinfo",
			"/fedadmins",
			"/joinfed <federation>",
			"/leavefed",
			"/chatfed",
			"",
			"Rose-style quietfed, fedsubs, and fedstat are still deferred in Sukoon.",
		},
	},
	helpFilters: {
		Title: "Filters",
		Lines: []string{
			"Filters auto-reply when incoming text contains the configured trigger. Matching is case-insensitive in the current build.",
			"",
			"/filter <trigger> <reply>",
			"/filters",
			"/stop <trigger>",
			"/stopall",
			"",
			"Quoted triggers are supported for multi-word phrases. Open the example and formatting pages below for live syntax.",
		},
	},
	helpFilterExamples: {
		Title: "Filter Example Usage",
		Lines: []string{
			"Truthful examples for the current Sukoon filter parser:",
			"",
			"/filter hello Hi there",
			"/filter \"buy now\" Sales links are not allowed here.",
			"/filter welcome Welcome {first}",
			"/filter rules Please read {rules}",
			"/filter ping Pong %%% Still here %%% Online",
			"/stop hello | buy now",
			"",
			"Reply-tag filters, protected filters, exact/prefix match toggles, and media-save shortcuts are still deferred.",
		},
	},
	helpFormatting: {
		Title: "Formatting",
		Lines: []string{
			"Sukoon's stored-content formatting is intentionally narrow and safe.",
			"",
			"Current support focuses on real runtime features: button rows, contextual fillings, and random-content separators.",
			"",
			"Open the pages below for the exact syntax Sukoon currently supports.",
		},
	},
	helpFormattingMarkdown: {
		Title: "Markdown Formatting",
		Lines: []string{
			"Sukoon does not expose the full Rose-style markdown helper set in stored content yet.",
			"",
			"Guaranteed syntax today:",
			"- button rows with [Label](buttonurl:https://example.com)",
			"- contextual fillings such as {first} and {chatname}",
			"- random choices using %%%",
			"",
			"Rose-style bold, italics, spoiler, code blocks, quotes, note buttons, and styled buttons are still deferred in the current runtime.",
		},
	},
	helpFormattingFillings: {
		Title: "Fillings",
		Lines: []string{
			"Supported contextual fillings in the current build:",
			"",
			"{first}",
			"{last}",
			"{fullname}",
			"{username}",
			"{mention}",
			"{id}",
			"{chat}",
			"{chatname}",
			"{rules}",
			"{rules:same}",
			"",
			"{mention} renders @username when available, otherwise the user's display name.",
			"Preview/protect/nonotif/mediaspoiler control tags are still deferred.",
		},
	},
	helpFormattingRandom: {
		Title: "Random Content",
		Lines: []string{
			"Use %%% between choices to let Sukoon pick one reply at send time.",
			"",
			"Supported in notes, filters, welcome text, goodbye text, and rules text.",
			"",
			"Examples:",
			"Hello {first} %%% Welcome back {first}",
			"/filter ping Pong %%% Still here %%% Online",
			"/setwelcome on Welcome {first} %%% Glad you're here {first}",
		},
	},
	helpFormattingButtons: {
		Title: "Buttons",
		Lines: []string{
			"Supported button syntax in stored content:",
			"",
			"[Website](buttonurl:https://misssukoon.vercel.app/)",
			"[Docs](buttonurl:https://example.com) [Status](buttonurl:https://example.org)",
			"",
			"Buttons on the same line stay in the same row. Start a new line for a new row.",
			"Callback-style note buttons and Rose-style button styling are still deferred.",
			"",
			"Website: https://misssukoon.vercel.app/",
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
			{Text: "Filters", CallbackData: helpCallback(helpFilters)},
			{Text: "Federations", CallbackData: helpCallback(helpFederations)},
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
			{Text: "Disabling", CallbackData: helpCallback(helpDisabling)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Federations", CallbackData: helpCallback(helpFederations)},
			{Text: "Filters", CallbackData: helpCallback(helpFilters)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Formatting", CallbackData: helpCallback(helpFormatting)},
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
	case helpFederations:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Fed Admin Commands", CallbackData: helpCallback(helpFederationsAdmin)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Federation Owner Commands", CallbackData: helpCallback(helpFederationsOwner)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "User Commands", CallbackData: helpCallback(helpFederationsUser)},
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
	case helpFilters:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Example Usage", CallbackData: helpCallback(helpFilterExamples)},
				{Text: "Formatting", CallbackData: helpCallback(helpFormatting)},
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
	case helpFormatting:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Markdown Formatting", CallbackData: helpCallback(helpFormattingMarkdown)},
				{Text: "Buttons", CallbackData: helpCallback(helpFormattingButtons)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Fillings", CallbackData: helpCallback(helpFormattingFillings)},
				{Text: "Random Content", CallbackData: helpCallback(helpFormattingRandom)},
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
	case helpFederationsAdmin, helpFederationsOwner, helpFederationsUser:
		return helpSubsectionMarkup(username, helpFederations)
	case helpFilterExamples:
		return helpSubsectionMarkup(username, helpFilters)
	case helpFormattingMarkdown:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Buttons", CallbackData: helpCallback(helpFormattingButtons)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: helpCallback(helpFormatting)},
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
	case helpFormattingFillings, helpFormattingRandom, helpFormattingButtons:
		return helpSubsectionMarkup(username, helpFormatting)
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
	case "blocklist", "blocklists", "rmblocklist", "rmbl", "unblocklistall":
		return helpBlocklists
	case "blocklists_examples", "blocklistexamples", "blocklist_examples":
		return helpBlocklistExamples
	case "captcha", "captchas", "captchamode", "captchakick", "captchakicktime":
		return helpCaptcha
	case "clean", "cleanup", "cleancommands", "cleancommand", "keepcommand", "cleancommandtypes":
		return helpCleanCommands
	case "disable", "enable", "disabled", "disableable", "disabledel", "disableadmin", "disabling":
		return helpDisabling
	case "federation", "federations", "fed", "newfed", "renamefed", "delfed", "fedtransfer":
		return helpFederations
	case "fedadmincommands", "federations_admin", "federation_admin", "fed_admin", "fban", "unfban", "feddemoteme", "myfeds":
		return helpFederationsAdmin
	case "fedownercommands", "federations_owner", "federation_owner", "fed_owner", "fedpromote", "feddemote":
		return helpFederationsOwner
	case "fedusercommands", "federations_user", "federation_user", "fed_user", "fedinfo", "fedadmins", "joinfed", "leavefed", "chatfed":
		return helpFederationsUser
	case "filters", "filter", "stop", "stopall":
		return helpFilters
	case "filters_examples", "filterexamples", "filter_examples", "exampleusage":
		return helpFilterExamples
	case "formatting":
		return helpFormatting
	case "formatting_markdown", "markdown", "markdownformatting":
		return helpFormattingMarkdown
	case "formatting_fillings", "fillings", "filling":
		return helpFormattingFillings
	case "formatting_random", "randomcontent", "random":
		return helpFormattingRandom
	case "formatting_buttons", "buttons", "button":
		return helpFormattingButtons
	case "locks", "lock", "unlock", "locktypes":
		return helpLocks
	case "locks_descriptions", "lockdescriptions", "lock_descriptions":
		return helpLockDescriptions
	case "locks_examples", "lockexamples", "lock_examples":
		return helpLockExamples
	case "log", "logs", "logchannel", "logchannels", "setlog", "unsetlog", "nolog", "logcategories":
		return helpLogChannels
	case "connections", "connect", "disconnect", "reconnect", "connection":
		return ""
	case "notesfilters", "notes", "ruleswelcome", "rules", "utility", "privacy", "owner", "clones", "protection", "security", "spam":
		return helpRoot
	default:
		return ""
	}
}
