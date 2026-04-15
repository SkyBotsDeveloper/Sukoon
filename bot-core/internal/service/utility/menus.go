package utility

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

const (
	callbackStartHome  = "ux:start:home"
	callbackStartClone = "ux:start:clone"
	callbackHelpPrefix = "ux:help:"
	callbackHelpCtx    = "ux:helpctx:"
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
	helpSilentPower        = "silentpower"
	helpExtra              = "extra"
	helpAntiRaid           = "antiraid"
	helpAntiAbuse          = "antiabuse"
	helpBioCheck           = "biocheck"
	helpCustomInstances    = "custominstances"
)

type helpPage struct {
	Title string
	Lines []string
}

var inlineSlashCommandPattern = regexp.MustCompile(`/[A-Za-z][A-Za-z0-9_]*`)
var inlinePlaceholderPattern = regexp.MustCompile(`\{[A-Za-z][A-Za-z0-9_:]*\}`)

var helpPages = map[string]helpPage{
	helpAdmin: {
		Title: "Admin",
		Lines: []string{
			"Make it easy to promote and demote users with the admin module!",
			"",
			"Admin commands:",
			"- /promote <reply/username/userid>: Promote a user.",
			"- /demote <reply/username/userid>: Demote a user.",
			"- /adminlist: List the admins in the current chat. (/admins also works.)",
			"- /admincache: Refresh Sukoon's admin lookup for this chat.",
			"- /anonadmin <yes/no/on/off>: Allow anonymous admins to use admin commands without permission verification. Not recommended.",
			"- /adminerror <yes/no/on/off>: Send error messages when normal users use admin commands. Default: on.",
			"",
			"Sukoon maps Telegram admin permissions to bot actions so admins cannot escalate beyond what Telegram already allows.",
			"Promoted users receive the overlap of the caller's Telegram admin rights, minus add-admins permission.",
			"Anonymous admins stay hidden from /adminlist to preserve anonymity.",
			"Use /admincache to force a fresh Telegram admin lookup if permissions were changed recently.",
		},
	},
	helpAntiflood: {
		Title: "Antiflood",
		Lines: []string{
			"You know how sometimes, people join, send 100 messages, and ruin your chat? With antiflood, that happens no more!",
			"",
			"Antiflood allows Sukoon to take action on users that send too many messages in a row or inside a timed window. Actions are: ban/mute/kick/tban/tmute.",
			"",
			"Admin commands:",
			"- /flood: Get the current antiflood settings.",
			"- /setflood <number/off/no>: Set the number of consecutive messages that trigger antiflood. Set to 0, off, or no to disable.",
			"- /setfloodtimer <count> <duration>: Set timed antiflood. Set to off or no to disable.",
			"- /floodmode <action type>: Choose which action to take on a flooder. Possible actions: ban/mute/kick/tban/tmute.",
			"- /clearflood <yes/no/on/off>: Whether to delete the full triggered flood set instead of only the messages after the limit.",
			"",
			"Examples:",
			"- /setflood 7",
			"- /setflood off",
			"- /setfloodtimer 10 30s",
			"- /setfloodtimer off",
			"- /floodmode mute",
			"- /floodmode tban 3d",
		},
	},
	helpApproval: {
		Title: "Approval",
		Lines: []string{
			"Sometimes, you might trust a user not to send unwanted content.",
			"Maybe not enough to make them admin, but you might be ok with locks, blocklists, and antiflood not applying to them.",
			"",
			"That's what approvals are for - approve trustworthy users to allow them to send.",
			"",
			"User commands:",
			"- /approval: Check a user's approval status in this chat.",
			"",
			"Admin commands:",
			"- /approve: Approve a user. Locks, blocklists, and antiflood won't apply to them anymore.",
			"- /unapprove: Unapprove a user. They will now be subject to locks, blocklists, and antiflood again.",
			"- /approved: List all approved users.",
			"- /unapproveall: Unapprove ALL users in a chat. This cannot be undone.",
		},
	},
	helpBans: {
		Title: "Bans",
		Lines: []string{
			"Some people need to be publicly banned; spammers, annoyances, or just trolls.",
			"",
			"This module allows you to do that easily, by exposing some common actions, so everyone will see!",
			"",
			"User commands:",
			"- /kickme: Users that use this, kick themselves.",
			"",
			"Admin commands:",
			"- /ban: Ban a user.",
			"- /dban: Ban a user by reply, and delete their message.",
			"- /sban: Silently ban a user, and delete your command message.",
			"- /tban: Temporarily ban a user. Example time values: 4m = 4 minutes, 3h = 3 hours, 6d = 6 days, 5w = 5 weeks.",
			"- /unban: Unban a user.",
			"- /mute: Mute a user.",
			"- /dmute: Mute a user by reply, and delete their message.",
			"- /smute: Silently mute a user, and delete your command message.",
			"- /tmute: Temporarily mute a user. Example time values: 4m = 4 minutes, 3h = 3 hours, 6d = 6 days, 5w = 5 weeks.",
			"- /unmute: Unmute a user.",
			"- /kick: Kick a user.",
			"- /dkick: Kick a user by reply, and delete their message.",
			"- /skick: Silently kick a user, and delete your command message.",
			"",
			"Examples:",
			"- Mute the user with username '@username' for two hours.",
			"-> /tmute @username 2h",
			"",
			"- Silently ban the replied user for spamming.",
			"-> /sban spam links",
		},
	},
	helpBlocklists: {
		Title: "Blocklists",
		Lines: []string{
			"Want to stop people asking stupid questions? or ban anyone saying censored words? Blocklists is the module for you!",
			"",
			"From blocking rude words, filenames/extensions, to specific emoji, everything is possible.",
			"",
			"Admin commands:",
			"- /addblocklist <blocklist trigger><reason>: Add a blocklist trigger. You can blocklist an entire sentence by putting it in quotes.",
			"- /rmblocklist <blocklist trigger>: Remove a blocklist trigger.",
			"- /unblocklistall: Remove all blocklist triggers - chat creator only.",
			"- /blocklist: List all blocklisted items.",
			"- /blocklistmode <blocklist mode>: Set the desired action to take when someone says a blocklisted item. Available: nothing/ban/mute/kick/warn/tban/tmute.",
			"- /blocklistdelete <yes/no/on/off>: Set whether blocklisted messages should be deleted. Default: on.",
			"- /setblocklistreason <reason>: Set the default blocklist reason to warn people with.",
			"- /resetblocklistreason: Reset the default blocklist reason to default - nothing.",
			"",
			"Top tip:",
			"Blocklists allow you to use some modifiers to match unknown characters.",
			"- ? matches a single occurrence of any non-whitespace character.",
			"- * matches any number of any non-whitespace character.",
			"- ** matches any number of any character (including spaces).",
		},
	},
	helpBlocklistExamples: {
		Title: "Blocklist Command Examples",
		Lines: []string{
			"If you're still curious as to how blocklists work, here are some examples you can copy.",
			"",
			"Example blocklist commands:",
			"- Automatically warn users who say blocklisted words:",
			"-> /blocklistmode warn",
			"",
			"- Override the blocklist mode for a single filter. Users that say 'boo' will get muted for 6 hours, instead of the default blocklist action:",
			"-> /addblocklist boo Don't scare the ghosts! {tmute 6h}",
			"",
			"- Add a full sentence to the blocklist. This would delete any message containing 'the admins suck':",
			"-> /addblocklist \"the admins suck\" Respect your admins!",
			"",
			"- Add multiple blocklist entries at once by separating them with commas inside brackets:",
			"-> /addblocklist (hi, hey, hello) Stop saying hello!",
			"",
			"- Stop any bit.ly links followed by exactly three characters:",
			"-> /addblocklist \"bit.ly/???\" We dont like 3 letter shorteners!",
			"",
			"- Stop any bit.ly links using the * shortcut to match any character:",
			"-> /addblocklist \"bit.ly/*\" We dont like shorteners!",
			"",
			"- Stop \"follow me on X\", as well as \"follow on X\", by using the ** syntax to block any number of words:",
			"-> /addblocklist \"follow ** X\" No promoting X accounts!",
			"",
			"- Stop people sending zip files, by blocklisting file:*.zip:",
			"-> /addblocklist \"file:*.zip\" zip files are not allowed here.",
			"",
			"- Stop people using the @gif inline bot by adding inline:@gif:",
			"-> /addblocklist \"inline:@gif\" The gif bot is not allowed here.",
			"",
			"- Stop forwards from a channel by adding forward:@channelusername:",
			"-> /addblocklist \"forward:@botnews\" The bot news channel is not allowed here.",
			"",
			"- Stop messages that exactly match the blocklist entry:",
			"-> /addblocklist \"exact:hi\" This will delete messages that just say 'hi', and not 'hi there'",
			"",
			"- Stop messages that start with a certain prefix:",
			"-> /addblocklist \"prefix:hi\" This will delete messages that start with 'hi', but not 'say hi'",
			"",
			"- Stop messages containing visually similar words, for example using mixtures of scripts:",
			"-> /addblocklist \"lookalike:bot\" This will delete messages that contain 'bot', but also 'вот' (cyrillic script)",
			"",
			"- Stop any 🖕 emoji, or any stickers related to it:",
			"-> /addblocklist 🖕 This emoji is not allowed here.",
			"",
			"- To blocklist a stickerpack, simply reply to a sticker with your addblocklist command:",
			"-> (replying to a sticker) /addblocklist",
			"",
			"- To blocklist a stickerpack and assign a reason, use the stickerpack:<> syntax:",
			"-> (replying to a sticker) /addblocklist stickerpack:<> These stickers are banned!",
			"",
			"- To stop a single blocklist item from deleting messages:",
			"-> /addblocklist test {nodel} {warn} No talking about tests here, don't do it again!",
			"",
			"- If you've disabled blocklist deletion, but you want to configure some items to still delete:",
			"-> /addblocklist boop {del} {ban} No b words here!",
		},
	},
	helpCaptcha: {
		Title: "CAPTCHA",
		Lines: []string{
			"Some chats get a lot of users joining just to spam. This could be because they're trolls, or part of a spam network.",
			"",
			"To slow them down, you could try enabling CAPTCHAs. New users joining your chat will be required to complete a test to confirm that they're real people.",
			"",
			"Admin commands:",
			"- /captcha <yes/no/on/off>: Enable or disable CAPTCHAs. Welcome messages must be enabled first.",
			"- /captchamode <button/math/text/text2>: Choose which CAPTCHA type to use for your chat.",
			"- /captcharules <yes/no/on/off>: Require new users to accept the rules before they can speak.",
			"- /captchamutetime <Xw/d/h/m>: Automatically unmute unsolved users after a delay. Use off to disable.",
			"- /captchakick <yes/no/on/off>: Kick users that still have not solved the CAPTCHA.",
			"- /captchakicktime <Xw/d/h/m>: Set the time after which unsolved users are kicked.",
			"- /setcaptchatext <text>: Customise the CAPTCHA button text.",
			"- /resetcaptchatext: Reset the CAPTCHA button text to the default value.",
			"",
			"Examples:",
			"- Enable CAPTCHAs",
			"-> /captcha on",
			"",
			"- Change the CAPTCHA mode to text.",
			"-> /captchamode text",
			"",
			"- Enable CAPTCHA rules, forcing users to read the rules before being allowed to speak.",
			"-> /captcharules on",
			"",
			"- Disable captcha mute time; users will stay muted until they solve the captcha.",
			"-> /captchamutetime off",
			"",
			"Note: CAPTCHAs only run while welcome messages are enabled. PM solve modes are live; join-request-specific delivery is still deferred.",
		},
	},
	helpCleanCommands: {
		Title: "Clean Commands",
		Lines: []string{
			"Keep your chat clean by cleaning up commands from both users and admins!",
			"",
			"This module allows you to delete certain command categories, for both users and admins, to ensure your chat is kept clean.",
			"For example, you might choose to delete all user commands; this will stop users from accidentally pressing on blue-text commands in other people's messages.",
			"",
			"Available options are:",
			"- all: Delete ALL commands sent to the group.",
			"- admin: Delete any admin-only commands sent to the group (eg /ban, /mute, or any settings changes).",
			"- user: Delete any user commands sent to the group (eg /get, /rules, or /report). These commands will also be cleaned when admins use them.",
			"- other: Delete any commands which aren't recognised as being valid Sukoon commands.",
			"",
			"Admin commands:",
			"- /cleancommand <type>: Select which command types to delete.",
			"- /keepcommand <type>: Select which command types to stop deleting.",
			"- /cleancommandtypes: List the different command types which can be cleaned.",
			"",
			"Examples:",
			"- Delete all commands, but still respond to them:",
			"-> /cleancommand all",
			"",
			"- Delete all users commands (but still respond), as well as commands for other bots:",
			"-> /cleancommand user other",
			"",
			"- Stop deleting all commands:",
			"-> /keepcommand all",
			"",
			"Note: If you are looking to stop your users from using any commands altogether, and don't want Sukoon to reply to them, have a look at the locks module instead. You may also want to set up log channels, to track the settings changes that your admins are making!",
		},
	},
	helpCleanService: {
		Title: "Clean Service",
		Lines: []string{
			"Clean up automated Telegram service messages! The available categories are:",
			"",
			"- all: All service messages.",
			"- join: When a new user joins, or is added. Eg: 'X joined the chat'.",
			"- leave: When a user leaves, or is removed. Eg: 'X left the chat'.",
			"- other: Miscellaneous items; such as chat boosts, successful telegram payments, proximity alerts, webapp messages, message auto deletion changes, or checklist updates.",
			"- photo: When chat photos or chat backgrounds are changed.",
			"- pin: When a new message is pinned. Eg: 'X pinned a message'.",
			"- title: When chat or topic titles are changed.",
			"- videochat: When a video chat action occurs - eg starting, ending, scheduling, or adding members to the call.",
			"",
			"Admin commands:",
			"- /cleanservice <type/yes/no/on/off>: Select which service messages to delete.",
			"- /keepservice <type>: Select which service messages to stop deleting.",
			"- /nocleanservice <type>: Same as /keepservice.",
			"- /cleanservicetypes: List all the available service messages, with a brief explanation.",
			"",
			"Examples:",
			"- Stop all Telegram service messages:",
			"-> /cleanservice all",
			"",
			"- Stop Telegram's 'X joined the chat' messages:",
			"-> /cleanservice join",
			"",
			"- Keep Telegram's 'X pinned a message' messages:",
			"-> /keepservice pin",
		},
	},
	helpConnections: {
		Title: "Connections",
		Lines: []string{
			"Sometimes, you just want to add some notes and filters to a group chat, but you don't want everyone to see; this is where connections come in.",
			"",
			"Connections allow you to connect to a chat's database, and add things to it without the chat knowing about it.",
			"For obvious reasons, you need to be an admin to edit connected chat data; members can view public data.",
			"",
			"Admin commands:",
			"- /connect <chatid/username>: Connect to the specified chat, allowing you to view/edit contents.",
			"- /disconnect: Disconnect from the current chat.",
			"- /reconnect: Reconnect to the previously connected chat.",
			"- /connection: See information about the currently connected chat.",
			"",
			"Tips:",
			"- Connect to a chat by ID:",
			"-> /connect -1001235155926",
			"",
			"- Connect to a chat by username:",
			"-> /connect @SukoonSupportChat",
			"",
			"- When in a group, connect to the current chat:",
			"-> /connect",
			"",
			"- When in private, list recently connected chats:",
			"-> /connect",
			"",
			"You can retrieve the chat id from Telegram chat info or an id helper. Supergroup ids are usually negative.",
		},
	},
	helpDisabling: {
		Title: "Disabling",
		Lines: []string{
			"Not everyone wants every feature Sukoon offers. Some commands are best left unused; to avoid spam and abuse.",
			"",
			"This allows you to disable some commonly used commands, so no one can use them. It'll also allow you to autodelete them, stopping people from bluetexting.",
			"",
			"Admin commands:",
			"- /disable <commandname>: Stop users from using commandname in this group.",
			"- /enable <commandname>: Allow users to use commandname in this group.",
			"- /disableable: List all disableable commands.",
			"- /disabledel <yes/no/on/off>: Delete disabled commands when used by non-admins.",
			"- /disableadmin <yes/no/on/off>: Stop admins from using disabled commands too.",
			"- /disabled: List the disabled commands in this chat.",
			"",
			"Examples:",
			"- Stop people from using the info command:",
			"-> /disable info",
			"",
			"- Enable the info command:",
			"-> /enable info",
			"",
			"- Disable all commands listed in /disableable:",
			"-> /disable all",
			"",
			"- Delete disabled commands that get used:",
			"-> /disabledel on",
			"",
			"- Make sure that disabled commands are also disabled for admins:",
			"-> /disableadmin on",
			"",
			"Note:",
			"By default, disabling a command only disables it for non-admins. To stop admins from using disabled commands too, check the /disableadmin toggle.",
			"Disabled commands are still accessible through the /connect feature.",
		},
	},
	helpFederations: {
		Title: "Federations",
		Lines: []string{
			"Ah, group management. It's all fun and games, until you start getting spammers in, and you need to ban them. Then you need to start banning more, and more, and it gets painful.",
			"But then you have multiple groups, and you don't want these spammers in any of your groups - how can you deal? Do you have to ban them manually, in all your groups?",
			"",
			"No more! With federations, you can make a ban in one chat overlap to all your other chats.",
			"You can even appoint federation admins, so that your trustworthiest admins can ban across all the chats that you want to protect.",
			"",
			"Open a federation command section below.",
		},
	},
	helpFederationsAdmin: {
		Title: "Fed Admin Commands",
		Lines: []string{
			"The following is the list of all fed admin commands. To run these, you have to be a federation admin in the current federation.",
			"",
			"Commands:",
			"- /fban <reply/username/mention/userid> <reason>: Ban a user from the current chat's federation.",
			"- /unfban <reply/username/mention/userid>: Unban a user from the current chat's federation.",
			"- /feddemoteme <fedID>: Demote yourself from a fed.",
			"- /myfeds: List all feds you are an admin in.",
			"",
			"Examples:",
			"-> /fban @spammer raid spam",
			"-> /unfban 123456789",
			"-> /feddemoteme main",
		},
	},
	helpFederationsOwner: {
		Title: "Federation Owner Commands",
		Lines: []string{
			"These are the list of available fed owner commands. To run these, you have to own the current federation.",
			"",
			"Owner Commands:",
			"- /newfed <fedname>: Create a new federation. Only one federation per user.",
			"- /renamefed <fedname>: Rename your federation.",
			"- /delfed: Delete your federation and its stored federation data.",
			"- /fedtransfer <reply/username/mention/userid>: Transfer your federation to another user.",
			"- /fedpromote <reply/username/mention/userid>: Promote a user to fed admin in your fed.",
			"- /feddemote <reply/username/mention/userid>: Demote a federation admin in your fed.",
			"- /fednotif <yes/no/on/off>: Whether to receive PM notifications of federation actions.",
			"- /fedreason <yes/no/on/off>: Whether fedbans should require a reason.",
			"- /subfed <FedID>: Subscribe your federation to another. Users banned in the subscribed fed will also be banned in this one.",
			"- /unsubfed <FedID>: Unsubscribe your federation from another.",
			"- /fedexport <csv/minicsv/json/human>: Export the list of currently banned users. Default output is CSV.",
			"- /fedimport <overwrite/keep> <csv/minicsv/json/human>: Import a list of banned users.",
			"- /setfedlog: Set the current chat as the federation log.",
			"- /unsetfedlog: Unset the federation log.",
			"- /setfedlang <language>: Change the federation log language label.",
			"",
			"Note:",
			"Subscriptions do not change your own banlist. They only inherit bans from the subscribed federation.",
		},
	},
	helpFederationsUser: {
		Title: "Federation User Commands",
		Lines: []string{
			"These commands do not require you to be admin of a federation. They are for looking up federation information, checking fbans, or linking a chat.",
			"",
			"Commands:",
			"- /fedinfo <FedID>: Information about a federation.",
			"- /fedadmins <FedID>: List the admins in a federation.",
			"- /fedsubs <FedID>: List all federations your federation is subscribed to.",
			"- /joinfed <FedID>: Join the current chat to a federation. A chat can only join one federation.",
			"- /leavefed: Leave the current federation.",
			"- /fedstat: List all federations that you have been banned in.",
			"- /fedstat <user ID>: List all federations that a user has been banned in.",
			"- /fedstat <FedID>: Give information about your ban in a federation.",
			"- /fedstat <user ID> <FedID>: Give information about a user's ban in a federation.",
			"- /chatfed: Information about the federation the current chat is in.",
			"- /quietfed <yes/no/on/off>: Whether to send notifications when fedbanned users join the chat.",
			"",
			"Examples:",
			"-> /fedinfo main",
			"-> /fedstat 123456789 main",
			"-> /quietfed on",
		},
	},
	helpFilters: {
		Title: "Filters",
		Lines: []string{
			"Make your chat more lively with filters; the bot will reply to certain words.",
			"",
			"Filters are case-insensitive. Every time someone says your trigger words, Sukoon will reply with your configured response. You can use this to create your own lightweight commands if desired.",
			"",
			"Commands:",
			"- /filter <trigger> <reply>: Every time someone says trigger, the bot will reply with your sentence. For multiple word filters, quote the trigger.",
			"- /filters: List all chat filters.",
			"- /stop <trigger>: Stop the bot from replying to trigger.",
			"- /stopall: Stop ALL filters in the current chat. This cannot be undone.",
			"",
			"Quoted triggers are supported for multi-word phrases. Open the example and formatting pages below for live syntax.",
		},
	},
	helpFilterExamples: {
		Title: "Filter Example Usage",
		Lines: []string{
			"Filters can seem quite complicated; so here are some examples, so you can get some inspiration.",
			"",
			"Examples:",
			"- Set a filter:",
			"-> /filter hello Hello there! How are you?",
			"",
			"- Set a filter which uses the user's name through fillings:",
			"-> /filter hello Hello there {first}! How are you?",
			"",
			"- Set a filter on a sentence:",
			"-> /filter \"hello friend\" Hello back! Long time no see!",
			"",
			"- Set multiple filters at once by wrapping triggers in brackets and separating with commas:",
			"-> /filter (hi, hey, hello, \"hi there\") Hello back! Long time no see!",
			"",
			"- Set a filter that can only be used by admins:",
			"-> /filter \"trigger\" This filter won't happen if a normal user says it {admin}",
			"",
			"- Or, set a filter that can only be used by users:",
			"-> /filter \"trigger\" Admins won't trigger this {user}",
			"",
			"- Set a filter that only triggers if the entire message matches the filter:",
			"-> /filter \"exact:hi\" This will only match 'hi', and not 'hi there'",
			"",
			"- Set a filter that only triggers if the message starts with the trigger:",
			"-> /filter \"prefix:hi\" This will match 'hi', and 'hi there', but NOT 'Say hi'",
			"",
			"- If an admin wants to force a {user} filter to reply:",
			"-> trigger force",
			"",
			"- To get the unformatted version of a filter, say the trigger followed by noformat:",
			"-> trigger noformat",
			"",
			"- To save a protected filter, which can't be forwarded:",
			"-> /filter \"example\" This filter can't be forwarded {protect}",
			"",
			"- If you want the filter to reply to the person you replied to, instead of you:",
			"-> /filter \"magic\" Watch out for wizards! {replytag}",
			"",
			"- To save a file, image, gif, or any other attachment, simply reply to it with:",
			"-> /filter trigger",
			"",
			"- To set a filter which replies with a random answer from a preset list:",
			"-> /filter test Answer one %%% Answer two",
			"",
			"- To set a filter which gives a different reply to admins and users:",
			"-> /filter test Only admins see this {admin} %%% Only users see this {user}",
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
			"You can format your message using bold, italics, underline, and much more. Go ahead and experiment!",
			"",
			"Supported markdown:",
			"- `code words`: Backticks are used for monospace fonts. Shows as: code words.",
			"- _italic words_: Underscores are used for italic fonts. Shows as: italic words.",
			"- *bold words*: Asterisks are used for bold fonts. Shows as: bold words.",
			"- ~strikethrough~: Tildes are used for strikethrough. Shows as: strikethrough.",
			"- __underline__: Double underscores are used for underlines. Shows as: underline. NOTE: Some clients try to be smart and interpret it as italic. In that case, try to use your app's built-in formatting.",
			"- ||spoiler||: Double vertical bars are used for spoilers. Shows as: spoiler.",
			"- ```shell",
			"echo \"hi\"```: Triple backticks are used for codeblocks. You can also specify the code language in the first line (here, shell). Shows as: echo \"hi\".",
			"- > quote: You can quote a line by prefixing it with >. Shows as: quote.",
			"- **> first line",
			"> second",
			"> third",
			"> hidden||: You can create a multiline quote by starting a quote with **>, and ending it with ||. This will show the first three lines, and then collapse the rest.",
			"- [hyperlink](misssukoon.vercel.app): This is the formatting used for hyperlinks. Shows as: hyperlink (https://misssukoon.vercel.app/).",
			"- [My button](buttonurl://misssukoon.vercel.app): This is the formatting used for creating buttons. This example will create a button named My button which opens misssukoon.vercel.app when clicked.",
			"If you would like to send buttons on the same row, use the :same formatting. EG:",
			"[button 1](buttonurl://example.com)",
			"[button 2](buttonurl://example.com:same)",
			"[button 3](buttonurl://example.com)",
			"This will show button 1 and 2 on the same line, with 3 underneath.",
			"Use the Sukoon docs website to help with the button syntax: https://misssukoon.vercel.app/",
			"- [Styled button](buttonurl#primary://misssukoon.vercel.app): Styled button syntax is accepted for compatibility. Telegram bots cannot force client-side button colours, so styles are treated as normal URL buttons.",
			"- [note button](buttonurl://#notename): This syntax creates a button which links to a note. When clicked, the user is redirected to the bot's PM to see the note.",
		},
	},
	helpFormattingFillings: {
		Title: "Fillings",
		Lines: []string{
			"You can customise stored messages with contextual data. For example, you can mention a user by name in a welcome message, note, or filter.",
			"",
			"Supported fillings:",
			"- {first}: The user's first name.",
			"- {last}: The user's last name.",
			"- {fullname}: The user's full name.",
			"- {username}: The user's username. If they don't have one, Sukoon falls back to their display name.",
			"- {mention}: Uses @username when available, otherwise the user's display name.",
			"- {id}: The user's ID.",
			"- {chatname}: The chat's name.",
			"- {rules}: Create a button to the current chat's rules on a new row.",
			"- {rules:same}: Create a rules button on the same row as the previous buttons.",
			"- {admin}: Filter response is available to admins only.",
			"- {user}: Filter response is available to normal users only.",
			"- {replytag}: Filter fillings target the replied-to user when available.",
			"- {preview}: Enable web previews for stored text replies.",
			"- {preview:top}: Enable the preview and place it above the message text when Telegram allows it.",
			"- {nonotif}: Sends the filter reply without notification.",
			"- {protect}: Sends the filter reply as protected content.",
			"- {mediaspoiler}: Marks saved photo/video/animation filter replies as spoiler media.",
			"",
			"Example usages:",
			"- Save a filter using the user's name.",
			"-> /filter test {first} triggered this filter.",
			"",
			"- Add a rules button to a note.",
			"-> /save info Press the button to read the chat rules! {rules}",
			"",
			"- Mention a user in the welcome message.",
			"-> /setwelcome on Welcome {mention} to {chatname}!",
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
			"One of Telegram's popular features is the ability to add buttons to welcome messages, notes, or filters.",
			"",
			"Simple buttons:",
			"- The following syntax creates a button called Google, which opens google.com.",
			"-> [Google](buttonurl://google.com)",
			"",
			"Buttons on the same line:",
			"- Add :same to place a button on the previous row.",
			"-> [Google](buttonurl://google.com)",
			"-> [Bing](buttonurl://bing.com:same)",
			"",
			"Note buttons:",
			"- Note buttons open an existing saved note in PM. Save the note first.",
			"-> [First note](buttonurl://#my_note)",
			"-> [Second note](buttonurl://#second_note:same)",
			"",
			"Advanced example:",
			"-> [Google](buttonurl://google.com)",
			"-> [Bing](buttonurl://bing.com:same)",
			"-> [Other search engines](buttonurl://#search_engines)",
			"",
			"Online docs and generator:",
			"https://misssukoon.vercel.app/",
			"",
			"Remember that buttons need to be saved in Sukoon to be used; you can't send them directly from your account.",
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
			"Do stickers annoy you? or want to avoid people sharing links? or pictures? You're in the right place!",
			"",
			"The locks module allows you to lock away some common items in the Telegram world; the bot will automatically delete them!",
			"",
			"Admin commands:",
			"- /lock <item(s)>: Lock one or more items. Now, only admins can use this type!",
			"- /unlock <item(s)>: Unlock one or more items. Everyone can use this type again!",
			"- /locks: List currently locked items.",
			"- /lockwarns <yes/no/on/off>: Enable or disable whether a user should be warned when using a locked item.",
			"- /locktypes: Show the list of all lockable items.",
			"- /allowlist <url/id/command/@username(s)>: Allowlist a URL, group ID, channel @, bot @, command, or stickerpack link to stop it being deleted by relevant locks. Separate entries with a space. If no arguments are given, returns the current allowlist.",
			"- /rmallowlist <url/id/@channelname(s)>: Remove one or more allowlist items.",
			"- /rmallowlistall: Remove all allowlisted items.",
		},
	},
	helpLockDescriptions: {
		Title: "Lock Descriptions",
		Lines: []string{
			"There are lots of different locks, and some of them might not be super clear; this section aims to explain each kind of supported lock.",
			"",
			"Types:",
			"- all: Every supported lock type at once.",
			"- album: Media albums such as grouped photos.",
			"- anonchannel: Messages sent through anonymous channels.",
			"- audio: Audio media messages.",
			"- botlink: Messages containing links or usernames to Telegram bots.",
			"- button: Messages which contain inline buttons.",
			"- cashtag: Messages containing cash tags such as '$CASH'.",
			"- cjk: Messages containing Chinese, Japanese, or Korean characters.",
			"- command: Messages that start with a Telegram command.",
			"- contact: Contact media messages.",
			"- cyrillic: Messages containing Cyrillic characters.",
			"- document: Document media messages.",
			"- email: Messages which contain emails.",
			"- emoji: Messages containing emoji.",
			"- emojicustom: Messages containing custom Telegram emoji.",
			"- emojigame: Telegram mini games like dice, football, or darts.",
			"- emojionly: Messages which contain only emoji.",
			"- externalreply: Replies to messages from other chats.",
			"- forward / forwarduser / forwardbot / forwardchannel / forwardstory: Forwarded messages and their typed variants.",
			"- game: Bot API game messages.",
			"- gif: GIF media messages.",
			"- inline: Messages sent through inline bots, like @gif or @pic.",
			"- invitelink: Messages containing Telegram group or channel links.",
			"- location: Location messages.",
			"- phone: Messages containing phone numbers.",
			"- photo: Messages containing a photo.",
			"- poll: Poll messages.",
			"- rtl: Messages containing right-to-left characters.",
			"- spoiler: Messages containing Telegram spoiler entities.",
			"- sticker / stickeranimated / stickerpremium: Sticker messages and their animated/premium variants.",
			"- text: Messages containing text or captions.",
			"- url: Messages containing website links.",
			"- video: Video media messages.",
			"- videonote: Telegram video notes.",
			"- voice: Voice messages.",
			"- zalgo: Messages containing excessive formatting characters.",
			"",
			"Comment, checklist, and bot-add locks are still deferred until the runtime has clean Telegram coverage for them.",
		},
	},
	helpLockExamples: {
		Title: "Example Commands",
		Lines: []string{
			"Locks are a powerful tool, with lots of different options. So here are a few examples to get you started and familiar with how to use them.",
			"",
			"Examples:",
			"- Stop all users from sending stickers with:",
			"-> /lock sticker",
			"",
			"- You can lock/unlock multiple items by chaining them:",
			"-> /lock sticker photo gif video",
			"",
			"- Want a harsher punishment for certain actions? Set a custom lock action for it! Separate the types from your reason with ###:",
			"-> /lock invitelink ### no promoting other chats {ban}",
			"",
			"- Reset the custom lock action and reason for a single item:",
			"-> /lock emoji ###",
			"",
			"- Reset all custom lock actions and reasons; remember to unlock again after:",
			"-> /lock all ###",
			"",
			"- List all locks at once:",
			"-> /locks list",
			"",
			"- To allow forwards from a specific channel, use its username or ID:",
			"-> /allowlist @channelusername",
			"",
			"- If you've locked stickers but want to allow a specific sticker pack, allowlist the pack share link:",
			"-> /allowlist t.me/addstickers/Pinup_Girl",
			"",
			"- Emoji-pack allowlisting is still deferred in Sukoon's current runtime.",
		},
	},
	helpLogChannels: {
		Title: "Log Channels",
		Lines: []string{
			"Recent actions are nice, but they don't help you log every action taken by the bot. This is why you need log channels!",
			"",
			"Log channels can help you keep track of exactly what the other admins are doing. Bans, mutes, warns, approvals, reports, and automated actions can all be tracked there.",
			"",
			"Setting a log channel is done by the following steps:",
			"- Add Sukoon to your channel, as an admin.",
			"- Send /setlog to your channel.",
			"- Forward the /setlog command to the group you wish to be logged.",
			"- Congrats! all done :)",
			"",
			"Admin commands:",
			"- /logchannel: Get the name of the current log channel.",
			"- /setlog: Set the log channel for the current chat.",
			"- /unsetlog: Unset the log channel for the current chat.",
			"- /log <category>: Enable a log category - actions of that type will now be logged.",
			"- /nolog <category>: Disable a log category - actions of that type will no longer be logged.",
			"- /logcategories: List all supported categories, with information on what they refer to.",
		},
	},
	helpMisc: {
		Title: "Misc",
		Lines: []string{
			"General utility commands that do not belong to one moderation family.",
			"",
			"/start",
			"/help",
			"/donate",
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
	helpSilentPower: {
		Title: "Silent Power",
		Lines: []string{
			"Silent power lets trusted helpers act quietly without giving them full Telegram admin rights.",
			"",
			"/mods",
			"/mod <reply|user>",
			"/unmod <reply|user>",
			"/muter <reply|user>",
			"/unmuter <reply|user>",
			"/sban, /smute, /skick",
			"",
			"Only a group admin with promote-members permission can grant or remove mod power.",
			"Use /mods to review the current silent-power list.",
		},
	},
	helpExtra: {
		Title: "Extra",
		Lines: []string{
			"Extra collects smaller helper surfaces that do not need a full category tree of their own.",
			"",
			"/donate",
			"/afk [reason]",
			"/mybot",
			"/language",
			"",
			"Use Misc, Privacy, and Custom Instances for the deeper command details.",
		},
	},
	helpAntiRaid: {
		Title: "AntiRaid",
		Lines: []string{
			"Some people on telegram find it entertaining to \"raid\" chats. During a raid, hundreds of users join a chat to spam.",
			"",
			"The antiraid module allows you to quickly stop anyone from joining when such a raid is happening.",
			"All new joins will be temporarily banned for the next few hours, allowing you to wait out the spam attack until the trolls stop.",
			"",
			"Admin commands:",
			"- /antiraid <optional time/off/no>: Toggle antiraid. All new joins will be temporarily banned for the next few hours.",
			"- /raidtime <time>: View or set the desired antiraid duration. Default 6h.",
			"- /raidactiontime <time>: View or set the time for antiraid to tempban users for. Default 1h.",
			"- /autoantiraid <number/off/no>: Set the number of joins per minute after which to enable automatic antiraid. Set to 0, off, or no to disable.",
			"",
			"Examples:",
			"- Enable antiraid for 3 hours:",
			"-> /antiraid 3h",
			"",
			"- Disable antiraid:",
			"-> /antiraid off",
			"",
			"- Automatically enable antiraid if over 15 users join in under a minute:",
			"-> /autoantiraid 15",
			"",
			"- Disable automatic antiraid:",
			"-> /autoantiraid off",
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
	helpBioCheck: {
		Title: "Bio Check",
		Lines: []string{
			"Bio Check scans user bios for invite-style handles and link spam without punishing normal messages.",
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

func helpContextCallback(parent string, page string) string {
	return callbackHelpCtx + parent + ":" + page
}

func startLandingText() string {
	return strings.Join([]string{
		"Hey there! My name is Sukoon - I'm here to help you manage your groups! Use <code>/help</code> to find out how to use me to my full potential.",
		"",
		"Join my <a href=\"https://t.me/VivaanUpdates\">support channel</a> to get information on all the latest updates.",
		"",
		"Check <code>/privacy</code> to view the privacy policy, and interact with your data.",
	}, "\n")
}

func helpLandingText() string {
	return strings.Join([]string{
		"Sukoon Help",
		"",
		"Hey! I'm Sukoon, a fast moderation bot for groups and private communities.",
		"",
		"Browse the moderation, protection, AntiAbuse, and Bio Check sections below.",
		"",
		"Helpful commands:",
		"- <code>/start</code>: Starts me! You've probably already used this.",
		"- <code>/help</code>: Sends this message; I'll tell you more about myself!",
		"- <code>/donate</code>: Gives you info on how to support me and my creator.",
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
	switch section {
	case helpBlocklistExamples, helpLockExamples:
		return "HTML", true
	}
	return "HTML", false
}

func helpPageText(section string) string {
	if section == helpRoot {
		return helpLandingText()
	}
	page, ok := helpPages[section]
	if !ok {
		return helpLandingText()
	}
	lines := []string{"<b>" + html.EscapeString(page.Title) + "</b>", ""}
	for _, line := range page.Lines {
		lines = append(lines, formatHelpLine(line))
	}
	return strings.Join(lines, "\n")
}

func formatHelpLine(line string) string {
	if line == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(line, "-> "):
		return "-> " + helpCode(line[3:])
	case strings.HasPrefix(line, "/"):
		return helpCode(line)
	case strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}"):
		return helpCode(line)
	case strings.HasPrefix(line, "[") && strings.Contains(line, "]("):
		return helpCode(line)
	case strings.HasPrefix(line, "- "):
		return formatHelpBullet(line[2:])
	default:
		return formatInlineCode(html.EscapeString(line))
	}
}

func formatHelpBullet(body string) string {
	if body == "" {
		return "- "
	}
	if idx := strings.Index(body, ":"); idx > 0 {
		head := strings.TrimSpace(body[:idx])
		tail := body[idx+1:]
		if shouldCodeWrap(head) {
			return "- " + helpCode(head) + ":" + formatInlineCode(html.EscapeString(tail))
		}
	}
	if shouldCodeWrap(strings.TrimSpace(body)) {
		return "- " + helpCode(strings.TrimSpace(body))
	}
	return "- " + formatInlineCode(html.EscapeString(body))
}

func shouldCodeWrap(value string) bool {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return false
	case strings.HasPrefix(value, "/"):
		return true
	case strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}"):
		return true
	case strings.HasPrefix(value, "[") && strings.Contains(value, "]("):
		return true
	case strings.Contains(value, " / "):
		return true
	case !strings.Contains(value, " "):
		return true
	default:
		return false
	}
}

func helpCode(value string) string {
	return "<code>" + html.EscapeString(value) + "</code>"
}

func formatInlineCode(value string) string {
	value = inlineSlashCommandPattern.ReplaceAllStringFunc(value, func(match string) string {
		return "<code>" + match + "</code>"
	})
	value = inlinePlaceholderPattern.ReplaceAllStringFunc(value, func(match string) string {
		return "<code>" + match + "</code>"
	})
	return strings.ReplaceAll(value, "%%%", "<code>%%%</code>")
}

func privacyText() string {
	return strings.Join([]string{
		"Privacy",
		"",
		"Sukoon stores only the operational data it needs for moderation, automation, safety, and owner-requested workflows.",
		"",
		"Use <code>/mydata</code> to export your stored data and <code>/forgetme confirm</code> to delete eligible personal data for this bot instance.",
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
		"1. Open @BotFather and use <code>/newbot</code>.",
		"2. Copy the bot token BotFather gives you.",
		"3. Run <code>/clone &lt;bot_token&gt;</code> from an owner or sudo account.",
		"4. Start using your clone in your groups.",
		"5. Use <code>/mybot</code> later if you want to restart or remove it.",
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
			{Text: "Silent Power", CallbackData: helpCallback(helpSilentPower)},
			{Text: "Extra", CallbackData: helpCallback(helpExtra)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Bio Check", CallbackData: helpCallback(helpBioCheck)},
			{Text: "AntiAbuse", CallbackData: helpCallback(helpAntiAbuse)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "⭐ Custom Instances", CallbackData: helpCallback(helpCustomInstances)},
			{Text: "📚 Docs Website", URL: serviceutil.WebsiteURL},
		},
	)
}

func helpSectionMarkup(page string, username string, parent string) *telegram.InlineKeyboardMarkup {
	if parent == "" {
		parent = helpRoot
	}
	switch page {
	case helpAdmin:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpAntiflood:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpApproval:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpCaptcha:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpCleanService:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpConnections:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpDisabling:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpBans:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpAntiRaid:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpBlocklists:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Blocklist Command Examples", CallbackData: helpCallback(helpBlocklistExamples)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpFederations:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Fed Admin Commands", CallbackData: helpCallback(helpFederationsAdmin)},
				{Text: "Federation Owner Commands", CallbackData: helpCallback(helpFederationsOwner)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "User Commands", CallbackData: helpCallback(helpFederationsUser)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
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
			},
		)
	case helpCleanCommands:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Locks", CallbackData: helpContextCallback(helpCleanCommands, helpLocks)},
				{Text: "Log Channels", CallbackData: helpContextCallback(helpCleanCommands, helpLogChannels)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
		)
	case helpLocks:
		lockExamplesCallback := helpCallback(helpLockExamples)
		lockDescriptionsCallback := helpCallback(helpLockDescriptions)
		lockBackCallback := callbackHelpMain
		if parent == helpCleanCommands {
			lockExamplesCallback = helpContextCallback(helpCleanCommands, helpLockExamples)
			lockDescriptionsCallback = helpContextCallback(helpCleanCommands, helpLockDescriptions)
			lockBackCallback = helpCallback(helpCleanCommands)
		}
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Example Commands", CallbackData: lockExamplesCallback},
				{Text: "Lock descriptions", CallbackData: lockDescriptionsCallback},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: lockBackCallback},
			},
		)
	case helpLogChannels:
		logBackCallback := callbackHelpMain
		if parent == helpCleanCommands {
			logBackCallback = helpCallback(helpCleanCommands)
		}
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: logBackCallback},
			},
		)
	case helpBlocklistExamples:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: helpCallback(helpBlocklists)},
			},
		)
	case helpFederationsAdmin, helpFederationsOwner, helpFederationsUser:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: helpCallback(helpFederations)},
			},
		)
	case helpFilterExamples:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Fillings", CallbackData: helpContextCallback(helpFilterExamples, helpFormattingFillings)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: helpCallback(helpFilters)},
			},
		)
	case helpFormattingMarkdown:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Buttons", CallbackData: helpContextCallback(helpFormattingMarkdown, helpFormattingButtons)},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: helpCallback(helpFormatting)},
			},
		)
	case helpFormattingFillings:
		parentCallback := helpCallback(helpFormatting)
		if parent == helpFilterExamples {
			parentCallback = helpCallback(helpFilterExamples)
		}
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: parentCallback},
			},
		)
	case helpFormattingRandom:
		return helpSubsectionMarkup(username, helpFormatting)
	case helpFormattingButtons:
		parentCallback := helpCallback(helpFormatting)
		if parent == helpFormattingMarkdown {
			parentCallback = helpCallback(helpFormattingMarkdown)
		}
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: parentCallback},
			},
		)
	case helpLockDescriptions, helpLockExamples:
		lockParentCallback := helpCallback(helpLocks)
		if parent == helpCleanCommands {
			lockParentCallback = helpContextCallback(helpCleanCommands, helpLocks)
		}
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: lockParentCallback},
			},
		)
	default:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: callbackHelpMain},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Website", URL: serviceutil.WebsiteURL},
				{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
			},
		)
	}
}

func helpSubsectionMarkup(username string, parent string) *telegram.InlineKeyboardMarkup {
	return serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Back", CallbackData: helpCallback(parent)},
		},
		[]telegram.InlineKeyboardButton{
			{Text: "Website", URL: serviceutil.WebsiteURL},
			{Text: "Add to Group", URL: serviceutil.BotAddGroupLink(username)},
		},
	)
}

func helpMarkupWithParent(section string, username string, parent string) *telegram.InlineKeyboardMarkup {
	if section == helpRoot {
		return helpLandingMarkup(username)
	}
	return helpSectionMarkup(section, username, parent)
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
	value = strings.Join(strings.Fields(value), " ")
	switch value {
	case "", "main", "home", "root":
		return helpRoot
	case "admin", "admins", "adminlist", "mods", "mod", "staff", "promote", "demote", "admincache", "anonadmin", "adminerror":
		return helpAdmin
	case "antiflood", "flood", "setflood", "setfloodtimer", "setfloodmode", "floodmode", "clearflood":
		return helpAntiflood
	case "antiraid", "raid", "raidtime", "raidactiontime", "autoantiraid":
		return helpAntiRaid
	case "approval", "approve", "approvals", "approved", "unapprove", "unapproveall":
		return helpApproval
	case "bans", "ban", "moderation", "mute", "kick", "kickme":
		return helpBans
	case "warningsonly", "warning", "warn", "warns", "resetwarns", "setwarnlimit", "setwarnmode":
		return helpWarnings
	case "blocklist", "blocklists", "rmblocklist", "rmbl", "unblocklistall", "blocklistmode", "blocklistdelete", "setblocklistreason", "resetblocklistreason":
		return helpBlocklists
	case "blocklists_examples", "blocklistexamples", "blocklist_examples":
		return helpBlocklistExamples
	case "captcha", "captchas", "captchamode", "captcharules", "captchamutetime", "captchakick", "captchakicktime", "setcaptchatext", "resetcaptchatext":
		return helpCaptcha
	case "clean", "cleanup", "clean commands", "cleancommands", "cleancommand", "keepcommand", "cleancommandtypes":
		return helpCleanCommands
	case "clean service", "clean services", "cleanservice", "keepservice", "nocleanservice", "cleanservicetypes":
		return helpCleanService
	case "connections", "connect", "disconnect", "reconnect", "connection":
		return helpConnections
	case "disable", "enable", "disabled", "disableable", "disabledel", "disableadmin", "disabling":
		return helpDisabling
	case "federation", "federations", "fed":
		return helpFederations
	case "fedadmincommands", "federations_admin", "federation_admin", "fed_admin", "fban", "unfban", "feddemoteme", "myfeds":
		return helpFederationsAdmin
	case "fedownercommands", "federations_owner", "federation_owner", "fed_owner", "newfed", "renamefed", "delfed", "fedtransfer", "fedpromote", "feddemote", "fednotif", "fedreason", "subfed", "unsubfed", "fedexport", "fedimport", "setfedlog", "unsetfedlog", "setfedlang":
		return helpFederationsOwner
	case "fedusercommands", "federations_user", "federation_user", "fed_user", "fedinfo", "fedadmins", "fedsubs", "joinfed", "leavefed", "fedstat", "chatfed", "quietfed":
		return helpFederationsUser
	case "filters", "filter", "stop", "stopall":
		return helpFilters
	case "filters_examples", "filterexamples", "filter_examples", "exampleusage", "filter example", "filter examples", "filters example", "filters examples", "fiter example", "fiter examples":
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
	case "locks", "lock", "unlock", "locktypes", "lockwarns", "allowlist", "rmallowlist", "rmallowlistall":
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
	case "silentpower", "silent", "muter", "unmuter", "sban", "smute", "skick":
		return helpSilentPower
	case "extra":
		return helpExtra
	case "antiabuse", "abuse":
		return helpAntiAbuse
	case "antibio", "bio", "biocheck", "biolinks", "free", "unfree", "freelist":
		return helpBioCheck
	case "clones", "clone", "mybot", "mybots", "custominstances", "custom":
		return helpCustomInstances
	case "owner", "protection", "security", "spam":
		return helpRoot
	default:
		return ""
	}
}
