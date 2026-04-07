package utility

import (
	"fmt"
	"strings"

	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

const (
	callbackStartHome  = "ux:start:home"
	callbackStartClone = "ux:start:clone"
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
	helpCleanService       = "cleanservice"
	helpConnections        = "connections"
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
	helpGreetings          = "greetings"
	helpImportExport       = "importexport"
	helpLanguages          = "languages"
	helpLocks              = "locks"
	helpLockDescriptions   = "locks_descriptions"
	helpLockExamples       = "locks_examples"
	helpLogChannels        = "logchannels"
	helpMisc               = "misc"
	helpNotes              = "notes"
	helpPin                = "pin"
	helpPrivacy            = "privacy"
	helpPurges             = "purges"
	helpReports            = "reports"
	helpRules              = "rules"
	helpTopics             = "topics"
	helpWarnings           = "warnings"
	helpAntiRaid           = "antiraid"
	helpAntiAbuse          = "antiabuse"
	helpBioLinks           = "biolinks"
	helpCustomInstances    = "custominstances"
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
	helpCleanService: {
		Title: "Clean Service",
		Lines: []string{
			"Clean service removes join/leave and other service messages after Telegram posts them.",
			"",
			"/cleanservice <on|off|join|leave|pin|title|photo|other|videochat|all> [on|off]",
			"/nocleanservice <join|leave|pin|title|photo|other|videochat|all>",
			"/cleanservicetypes",
			"",
			"Examples:",
			"/cleanservice on",
			"/cleanservice join on",
			"/nocleanservice pin",
		},
	},
	helpConnections: {
		Title: "Connections",
		Lines: []string{
			"Connections let admins manage one chat from another place in Rose-style setups.",
			"",
			"Live runtime status: Sukoon does not expose remote chat connections in production yet.",
			"",
			"Planned surface:",
			"/connect <chatid/username>",
			"/disconnect",
			"/reconnect",
			"/connection",
			"",
			"This button is here so the help tree stays complete while the safe connection runtime is still deferred.",
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
	helpGreetings: {
		Title: "Greetings",
		Lines: []string{
			"Greetings control join and leave messages with the same stored-content features used in notes and filters.",
			"",
			"/welcome [on|off] [text]",
			"/setwelcome [on|off] [text]",
			"/goodbye [on|off] [text]",
			"/setgoodbye [on|off] [text]",
			"",
			"Fillings, buttons, and random-content separators are supported in greeting text.",
		},
	},
	helpImportExport: {
		Title: "Import / Export",
		Lines: []string{
			"This section is reserved for moving settings between chats or backups.",
			"",
			"Live runtime status: dedicated import/export commands are not exposed yet.",
			"",
			"Sukoon keeps this section visible so the merged help tree stays familiar while the safe transfer workflow is still deferred.",
		},
	},
	helpLanguages: {
		Title: "Languages",
		Lines: []string{
			"Language controls choose the active bot language for the current chat.",
			"",
			"/language",
			"/setlang <language_code>",
			"",
			"Current runtime ships a shared localization layer, but not every response string has translated variants yet.",
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
	helpMisc: {
		Title: "Misc",
		Lines: []string{
			"General utility commands that do not belong to one moderation family.",
			"",
			"/start",
			"/help",
			"/afk [reason]",
			"",
			"Use /privacy for data controls and /mybot for clone controls.",
		},
	},
	helpNotes: {
		Title: "Notes",
		Lines: []string{
			"Notes store reusable replies for quick retrieval with #name or /get name.",
			"",
			"/save <name> <text>",
			"/notes, /saved",
			"/get <name>",
			"/clear <name>",
			"",
			"Notes support buttons, fillings, and random-content separators.",
		},
	},
	helpPin: {
		Title: "Pin",
		Lines: []string{
			"Pin tools help admins highlight important messages.",
			"",
			"/pin",
			"/unpin [reply|message_id]",
			"/unpinall",
			"",
			"Reply to a message for /pin and /unpin when possible.",
		},
	},
	helpPrivacy: {
		Title: "Privacy",
		Lines: []string{
			"Privacy controls let users inspect or remove eligible personal data stored by the bot.",
			"",
			"/privacy",
			"/mydata",
			"/forgetme confirm",
			"",
			"Use these in PM when possible so exports and deletion prompts stay private.",
		},
	},
	helpPurges: {
		Title: "Purges",
		Lines: []string{
			"Purges remove batches of chat history without doing heavy work in the webhook path.",
			"",
			"/purge <count>",
			"/purge (reply to a message)",
			"/del (reply to a message)",
			"",
			"Large purge ranges are job-backed behind the scenes so the bot still responds quickly.",
		},
	},
	helpReports: {
		Title: "Reports",
		Lines: []string{
			"Reports let members flag a replied message to admins through the configured log channel.",
			"",
			"/reports [on|off]",
			"/report [reason] (reply to a message)",
			"",
			"A working log channel should be configured for reports to be useful.",
		},
	},
	helpRules: {
		Title: "Rules",
		Lines: []string{
			"Rules keep one canonical rules text per chat with group-friendly and PM-friendly delivery.",
			"",
			"/setrules <text>",
			"/rules",
			"/resetrules",
			"",
			"Stored rules support buttons, fillings, and random-content separators.",
		},
	},
	helpTopics: {
		Title: "Topics",
		Lines: []string{
			"This section is reserved for forum-topic aware moderation helpers.",
			"",
			"Live runtime status: dedicated topic commands are not exposed yet.",
			"",
			"The placeholder stays in the menu so Sukoon's help surface can grow without reshuffling everything later.",
		},
	},
	helpWarnings: {
		Title: "Warnings",
		Lines: []string{
			"Warnings build progressive discipline before the configured action fires.",
			"",
			"/warn <reason>",
			"/warns [reply|user]",
			"/resetwarns [reply|user]",
			"/setwarnlimit <number>",
			"/setwarnmode <mute|kick|ban>",
			"",
			"Example:",
			"/setwarnlimit 3",
			"/setwarnmode mute",
		},
	},
	helpAntiRaid: {
		Title: "AntiRaid",
		Lines: []string{
			"AntiRaid is meant for coordinated join-wave protection and emergency lockdown style flows.",
			"",
			"Live runtime status: Sukoon does not expose dedicated antiraid commands yet.",
			"",
			"Current protection coverage comes from /captcha, /flood, /lock, /cleanservice, and federation/global moderation.",
		},
	},
	helpAntiAbuse: {
		Title: "AntiAbuse",
		Lines: []string{
			"AntiAbuse targets a narrowed set of real abusive slurs instead of broad false-positive word lists.",
			"",
			"/antiabuse",
			"/antiabuse <on|off> [warn|delete_warn|mute|kick|ban]",
			"",
			"Admins, owners, sudo users, and approved users are not hit by the matcher.",
		},
	},
	helpBioLinks: {
		Title: "Bio Links",
		Lines: []string{
			"Bio Links checks user bios for invite-style handles and link spam without punishing normal messages.",
			"",
			"/antibio",
			"/antibio <on|off> [kick|ban|mute]",
			"/free <reply|user|username>",
			"/unfree <reply|user|username>",
			"/freelist",
			"",
			"Approved users and freed users bypass AntiBio checks.",
		},
	},
	helpCustomInstances: {
		Title: "Custom Instances",
		Lines: []string{
			"Custom instances let an owner or sudo user attach one private Sukoon clone to their account.",
			"",
			"/clone <bot_token>",
			"/mybot",
			"/rmclone <clone>",
			"/clones",
			"",
			"Each account can keep only one active clone at a time.",
		},
	},
}

func helpCallback(page string) string {
	return callbackHelpPrefix + page
}

func startLandingText() string {
	return strings.Join([]string{
		"Hey there! My name is Sukoon - I'm here to help you manage your groups! Use /help to find out how to use me to my full potential.",
		"",
		"Join my <a href=\"https://t.me/VivaanUpdates\">support channel</a> to get information on all the latest updates.",
		"",
		"Check /privacy to view the privacy policy, and interact with your data.",
	}, "\n")
}

func helpLandingText() string {
	return strings.Join([]string{
		"Sukoon Help",
		"",
		"Hey! I'm Sukoon, a fast moderation bot for groups and private communities.",
		"",
		"Browse the merged Rose-style, Group Help-style, AntiAbuse, and Bio Links sections below.",
		"",
		"Helpful commands:",
		"- /start: open the landing page",
		"- /help: browse this help menu",
		"- /privacy: review data controls",
		"- /mybot: manage your Sukoon clone",
		"",
		"Need updates or support? Join the <a href=\"https://t.me/VivaanUpdates\">support channel</a> or visit the <a href=\"" + serviceutil.WebsiteURL + "\">website</a>.",
		"",
		"Only real Sukoon commands are listed inside each page. Sections still being filled out say so clearly.",
	}, "\n")
}

func helpSectionOptions(section string) (string, bool) {
	if section == helpRoot {
		return "HTML", true
	}
	return "", false
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
			{Text: "Add me to your chat!", URL: serviceutil.BotAddGroupLink(username)},
			{Text: "Get your own Sukoon", CallbackData: callbackStartClone},
		},
	)
}

func cloneLandingText() string {
	return strings.Join([]string{
		"Get your own Sukoon",
		"",
		"If you want a private Sukoon instance for your own groups, create a bot in @BotFather first and then attach it to this runtime.",
		"",
		"Quick flow:",
		"1. Open @BotFather and use /newbot.",
		"2. Copy the bot token BotFather gives you.",
		"3. Run /clone <bot_token> from an owner or sudo account.",
		"4. Start using your clone in your groups.",
		"5. Use /mybot later if you want to restart or remove it.",
		"",
		"Each account can create only one Sukoon clone.",
		"If an old clone token was revoked, Sukoon clears the stale clone entry when you create a replacement.",
	}, "\n")
}

func cloneLandingMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: callbackStartHome},
			{Text: "Close", CallbackData: callbackClose},
		},
	)
}

func helpLandingMarkup(username string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Admin", CallbackData: helpCallback(helpAdmin)},
			{Text: "Antiflood", CallbackData: helpCallback(helpAntiflood)},
			{Text: "AntiRaid", CallbackData: helpCallback(helpAntiRaid)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Approval", CallbackData: helpCallback(helpApproval)},
			{Text: "Bans", CallbackData: helpCallback(helpBans)},
			{Text: "Blocklists", CallbackData: helpCallback(helpBlocklists)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "CAPTCHA", CallbackData: helpCallback(helpCaptcha)},
			{Text: "Clean Commands", CallbackData: helpCallback(helpCleanCommands)},
			{Text: "Clean Service", CallbackData: helpCallback(helpCleanService)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Connections", CallbackData: helpCallback(helpConnections)},
			{Text: "Disabling", CallbackData: helpCallback(helpDisabling)},
			{Text: "Locks", CallbackData: helpCallback(helpLocks)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Federations", CallbackData: helpCallback(helpFederations)},
			{Text: "Filters", CallbackData: helpCallback(helpFilters)},
			{Text: "Formatting", CallbackData: helpCallback(helpFormatting)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Greetings", CallbackData: helpCallback(helpGreetings)},
			{Text: "Import/Export", CallbackData: helpCallback(helpImportExport)},
			{Text: "Languages", CallbackData: helpCallback(helpLanguages)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Log Channels", CallbackData: helpCallback(helpLogChannels)},
			{Text: "Misc", CallbackData: helpCallback(helpMisc)},
			{Text: "Notes", CallbackData: helpCallback(helpNotes)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Pin", CallbackData: helpCallback(helpPin)},
			{Text: "Privacy", CallbackData: helpCallback(helpPrivacy)},
			{Text: "Purges", CallbackData: helpCallback(helpPurges)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Reports", CallbackData: helpCallback(helpReports)},
			{Text: "Rules", CallbackData: helpCallback(helpRules)},
			{Text: "Topics", CallbackData: helpCallback(helpTopics)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Warnings", CallbackData: helpCallback(helpWarnings)},
			{Text: "AntiAbuse", CallbackData: helpCallback(helpAntiAbuse)},
			{Text: "Bio Links", CallbackData: helpCallback(helpBioLinks)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Custom Instances", CallbackData: helpCallback(helpCustomInstances)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Docs Website", URL: serviceutil.WebsiteURL},
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
	case "antiraid", "raid", "raidtime", "raidactiontime", "autoantiraid":
		return helpAntiRaid
	case "approval", "approve", "approvals", "approved", "unapprove", "unapproveall":
		return helpApproval
	case "bans", "ban", "moderation", "mute", "kick", "kickme":
		return helpBans
	case "warningsonly", "warning", "warn", "warns", "resetwarns", "setwarnlimit", "setwarnmode":
		return helpWarnings
	case "blocklist", "blocklists", "rmblocklist", "rmbl", "unblocklistall":
		return helpBlocklists
	case "blocklists_examples", "blocklistexamples", "blocklist_examples":
		return helpBlocklistExamples
	case "captcha", "captchas", "captchamode", "captchakick", "captchakicktime":
		return helpCaptcha
	case "clean", "cleanup", "cleancommands", "cleancommand", "keepcommand", "cleancommandtypes":
		return helpCleanCommands
	case "cleanservice", "nocleanservice", "cleanservicetypes":
		return helpCleanService
	case "connections", "connect", "disconnect", "reconnect", "connection":
		return helpConnections
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
	case "greetings", "greeting", "welcome", "setwelcome", "goodbye", "setgoodbye":
		return helpGreetings
	case "importexport", "import", "export":
		return helpImportExport
	case "languages", "language", "setlang", "lang":
		return helpLanguages
	case "locks", "lock", "unlock", "locktypes":
		return helpLocks
	case "locks_descriptions", "lockdescriptions", "lock_descriptions":
		return helpLockDescriptions
	case "locks_examples", "lockexamples", "lock_examples":
		return helpLockExamples
	case "log", "logs", "logchannel", "logchannels", "setlog", "unsetlog", "nolog", "logcategories":
		return helpLogChannels
	case "misc", "utility", "utilities", "start", "afk":
		return helpMisc
	case "notesfilters", "notes", "saved", "save", "get", "clear":
		return helpNotes
	case "pin", "unpin", "unpinall":
		return helpPin
	case "privacy", "mydata", "forgetme":
		return helpPrivacy
	case "purge", "purges", "del":
		return helpPurges
	case "reports", "report":
		return helpReports
	case "ruleswelcome", "rules", "setrules", "resetrules":
		return helpRules
	case "topics", "topic":
		return helpTopics
	case "antiabuse", "abuse":
		return helpAntiAbuse
	case "antibio", "bio", "biolinks", "free", "unfree", "freelist":
		return helpBioLinks
	case "clones", "clone", "mybot", "mybots", "custominstances", "custom":
		return helpCustomInstances
	case "owner", "protection", "security", "spam":
		return helpRoot
	default:
		return ""
	}
}
