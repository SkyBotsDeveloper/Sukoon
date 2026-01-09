import type { TelegramBot } from "../bot"
import type { CommandContext, InlineKeyboardMarkup, TelegramCallbackQuery } from "../types"
import { supabase } from "../utils"
import { BOT_USERNAME } from "../constants"
import { languages, languageInfo, setChatLanguage, getChatLanguage } from "../langs"
import { OWNER_ID } from "../constants"

// ===========================================
// MISC COMMANDS
// ===========================================

const BOT_NAME = "Sukoon"
const SUPPORT_GROUP = "https://t.me/VivaanSupport"
const UPDATES_CHANNEL = "https://t.me/VivaanUpdates"
const DOCS_URL = "https://misssukoon.vercel.app"

const HELP_TEXTS: Record<string, string> = {
  admin: `<b>Admin Commands</b>

• /adminlist - List all admins in the chat
• /promote [user] - Promote a user to admin
• /demote [user] - Demote an admin
• /title [title] - Set custom admin title
• /setgtitle [title] - Set group title
• /setgpic - Set group photo (reply to image)
• /delgpic - Delete group photo
• /setgdesc [text] - Set group description
• /setsticker [name] - Set group sticker pack
• /delsticker - Delete group sticker pack
• /invitelink - Get invite link
• /admincache - Refresh admin cache`,

  antiflood: `<b>Antiflood Settings</b>

Antiflood allows you to take action on users who send more than X messages in a row.

• /flood - Get current antiflood settings
• /setflood [number] - Set flood limit (0 to disable)
• /setfloodmode [action] - Set action (ban/kick/mute/tban/tmute)
• /clearflood [on/off] - Enable/disable flood clearing

Actions:
• ban - Ban the user
• kick - Kick the user
• mute - Mute the user
• tban [time] - Temporarily ban
• tmute [time] - Temporarily mute`,

  antiraid: `<b>AntiRaid Settings</b>

AntiRaid helps protect your group from mass join attacks.

• /antiraid [on/off] - Enable/disable antiraid mode
• /raidtime [time] - Set how long antiraid lasts
• /raidactiontime [time] - Set action duration for raiders
• /raidmode [action] - Set action (ban/kick/mute)

When enabled, all new members joining during a raid will be actioned automatically.`,

  approval: `<b>Approval System</b>

Approve users to bypass certain restrictions.

• /approve [user] - Approve a user
• /unapprove [user] - Unapprove a user
• /approved - List all approved users
• /approval [user] - Check if a user is approved

Approved users:
• Can send messages during locks
• Bypass antiflood`,

  bans: `<b>Ban Commands</b>

• /ban [user] [reason] - Ban a user
• /sban [user] - Silently ban (delete command)
• /dban [user] - Ban and delete message
• /tban [user] [time] - Temporarily ban
• /unban [user] - Unban a user
• /banme - Ban yourself
• /kickme - Kick yourself

Time format: 4m = 4 minutes, 3h = 3 hours, 2d = 2 days`,

  blocklist: `<b>Blocklist</b>

Block certain words/phrases from being sent.

• /blocklist - View blocked words
• /addblock [word] - Add word to blocklist
• /unblock [word] - Remove from blocklist
• /blocklistmode [action] - Set action (delete/kick/ban/mute/tban/tmute)

Aliases: /addblocklist, /addblacklist, /rmblocklist, /rmblacklist, /blacklist`,

  captcha: `<b>CAPTCHA Settings</b>

Require new members to verify they're human.

• /captcha [on/off] - Enable/disable CAPTCHA
• /captchamode [mode] - Set mode (button/math/text)
• /captchatime [time] - Time to solve CAPTCHA
• /captchatimeout [time] - Timeout duration

Modes:
• button - Click a button to verify
• math - Solve a simple math problem
• text - Type the shown text`,

  cleancommands: `<b>Clean Commands</b>

Automatically delete command messages.

• /cleancommands [on/off] - Toggle command cleaning
• /cleancommand [cmd] [on/off] - Toggle for specific command

When enabled, bot commands like /mute etc. will be automatically deleted to keep chat clean.`,

  cleanservice: `<b>Clean Service Messages</b>

Automatically delete service messages.

• /cleanservice [on/off] - Toggle all service cleaning
• /cleanwelcome [on/off] - Clean welcome messages
• /cleangoodbye [on/off] - Clean goodbye messages

Service messages include:
• User joined/left notifications
• Pinned message notifications
• Group photo changes`,

  connections: `<b>Connections</b>

Connect to a group to manage it from PM.

• /connect [chat_id] - Connect to a group
• /disconnect - Disconnect from group
• /connection - View current connection
• /allowconnect [on/off] - Allow users to connect

When connected, you can use admin commands in PM to manage the connected group.`,

  disabling: `<b>Disabling Commands</b>

Disable certain commands in your group.

• /disable [command] - Disable a command
• /enable [command] - Enable a command
• /disabled - List disabled commands
• /disableable - List commands that can be disabled
• /disabledel [on/off] - Delete disabled command messages`,

  federation: `<b>Federation Commands</b>

Create and manage federations to ban users across multiple groups.

<b>Owner Commands:</b>
• /newfed [name] - Create a federation
• /delfed - Delete your federation
• /fedtransfer [user] - Transfer ownership
• /renamefed [name] - Rename federation

<b>Admin Commands:</b>
• /fban [user] [reason] - Federation ban
• /unfban [user] - Remove federation ban
• /fedpromote [user] - Promote fed admin
• /feddemote [user] - Demote fed admin

<b>General:</b>
• /joinfed [fed_id] - Join a federation
• /leavefed - Leave current federation
• /fedinfo [fed_id] - Federation info
• /fedadmins - List fed admins
• /fbanlist - List all fed bans
• /myfeds - List your federations`,

  filters: `<b>Filters</b>

Set up automatic replies to keywords.

• /filter [keyword] [reply] - Create a filter
• /filters - List all filters
• /stop [keyword] - Remove a filter

Filters support formatting:
• {first} - User's first name
• {last} - User's last name
• {fullname} - Full name
• {username} - @username
• {id} - User ID
• {chatname} - Chat name`,

  formatting: `<b>Text Formatting</b>

Sukoon supports markdown and HTML formatting.

<b>Markdown:</b>
• *bold* - <b>bold</b>
• _italic_ - <i>italic</i>
• \`code\` - <code>code</code>
• [text](url) - hyperlink
• [text](buttonurl://url) - button

<b>HTML:</b>
• &lt;b&gt;bold&lt;/b&gt;
• &lt;i&gt;italic&lt;/i&gt;
• &lt;code&gt;code&lt;/code&gt;
• &lt;a href="url"&gt;text&lt;/a&gt;`,

  greetings: `<b>Greetings</b>

Set welcome and goodbye messages.

• /welcome [on/off/text] - Set welcome message
• /goodbye [on/off/text] - Set goodbye message
• /setwelcome [text] - Set welcome text
• /setgoodbye [text] - Set goodbye text
• /resetwelcome - Reset to default
• /resetgoodbye - Reset to default
• /welcomemute [on/off] - Mute new users
• /welcomemutetime [time] - Mute duration

Variables: {first}, {last}, {fullname}, {username}, {id}, {chatname}, {mention}`,

  importexport: `<b>Import/Export</b>

Backup and restore your group settings.

• /export - Export all group settings
• /import - Import settings (reply to file)

Exports include:
• Notes
• Filters
• Welcome/Goodbye
• Rules
• Blocklist
• Locks
• Antiflood settings`,

  languages: `<b>Languages</b>

Change the bot's language.

• /language - View current language
• /setlang [code] - Set language

Available languages:
• en - English
• hi - Hindi
• es - Spanish
• pt - Portuguese
• ru - Russian
• ar - Arabic
• id - Indonesian
• tr - Turkish`,

  locks: `<b>Lock Settings</b>

Lock various message types.

• /lock [type] - Lock a type
• /unlock [type] - Unlock a type
• /locks - View current locks
• /locktypes - List all lock types

Lock types:
• all - Lock everything
• media - All media
• photo, video, audio, voice
• sticker, gif, document
• url, forward, game
• location, contact, poll`,

  logchannels: `<b>Log Channels</b>

Send all admin actions to a log channel.

• /setlog - Set current channel as log
• /unsetlog - Remove log channel
• /logchannel - View current log channel

Logged actions:
• Bans/Unbans
• Mutes/Unmutes
• Kicks
• Warns
• Promotions/Demotions`,

  misc: `<b>Miscellaneous Commands</b>

• /id - Get user/chat ID
• /info [user] - Get user info
• /markdownhelp - Formatting help
• /paste - Paste text to bin
• /ping - Check bot latency
• /stats - Bot statistics
• /donate - Support the bot`,

  notes: `<b>Notes</b>

Save and retrieve notes.

• /save [name] [text] - Save a note
• /get [name] - Get a note
• #notename - Same as /get
• /notes - List all notes
• /clear [name] - Delete a note
• /clearall - Delete all notes
• /privatenotes [on/off] - Send notes in PM`,

  pin: `<b>Pin Commands</b>

• /pin - Pin the replied message
• /unpin - Unpin current pinned message
• /unpinall - Unpin all messages
• /pinned - Get the pinned message
• /permapin [text] - Pin a text message

Pin options:
• /pin loud - Pin with notification
• /pin silent - Pin without notification`,

  privacy: `<b>Privacy Settings</b>

• /privacy - View privacy settings
• /setprivacy [on/off] - Toggle privacy mode
• /deletemydata - Delete your data from bot

Privacy mode hides:
• Your admin actions in logs
• Your data from /info command`,

  purges: `<b>Purge Commands</b>

Delete multiple messages at once.

• /purge - Delete messages from replied to current
• /spurge - Silent purge (no confirmation)
• /del - Delete single message
• /purgefrom - Start purge point
• /purgeto - End purge and delete

Note: Bot can only delete messages less than 48 hours old.`,

  reports: `<b>Reports</b>

Allow users to report rule-breakers.

• /report [reason] - Report a user
• @admin - Same as report
• /reports [on/off] - Toggle reports
• /reportformat [text] - Custom report format

Admins will receive a notification when someone is reported.`,

  rules: `<b>Rules</b>

Set and manage group rules.

• /rules - View group rules
• /setrules [text] - Set rules
• /clearrules - Clear rules
• /privaterules [on/off] - Send rules in PM
• /resetrules - Reset to default`,

  topics: `<b>Topics</b>

Manage forum topics (for forum groups).

• /newtopic [name] - Create new topic
• /closetopic - Close current topic
• /reopentopic - Reopen topic
• /deletetopic - Delete topic
• /topics - List all topics`,

  warnings: `<b>Warnings System</b>

• /warn [user] [reason] - Warn a user
• /dwarn [user] - Warn and delete message
• /swarn [user] - Silent warn
• /warns [user] - Check user warnings
• /resetwarns [user] - Reset warnings
• /rmwarn [user] - Remove last warn
• /setwarnlimit [number] - Set warn limit
• /setwarnmode [action] - Action at limit (ban/kick/mute/tban/tmute)
• /warnlimit - View current limit`,

  custom_instances: `<b>⭐ Custom Instances</b>

Create your own clone of Sukoon with your own bot token!

<b>How to clone:</b>
1. Go to BotFather and create a new bot
2. Copy the bot token
3. Use: /clone YOUR_TOKEN

Your bot will have all the same features!`,

  silentpower: `<b>Silent Power</b>

Give users moderation powers without showing them as admin.

<b>Commands:</b>
• /mod [user] - Give full mod powers (ban, mute, kick, warn)
• /unmod [user] - Remove mod powers
• /muter [user] - Give only mute power
• /unmuter [user] - Remove mute power
• /mods - List all silent mods in the chat

<b>Note:</b> Only group owner and admins with "Add New Admins" permission can manage silent powers.

<b>Mod Abilities:</b>
• Can ban/kick/mute users
• Can warn users
• Won't appear in admin list
• Can use all moderation commands`,

  extra: `<b>Extra Commands</b>

<b>AFK System:</b>
• /afk [reason] - Set yourself as AFK (away from keyboard)
• /brb [reason] - Same as /afk

When you're AFK and someone mentions or replies to you, the bot will notify them that you're away. Your AFK status is automatically removed when you send a message.

<b>More features coming soon!</b>`,

  biocheck: `<b>Bio Check Settings</b>

Detect and block users who have links in their Telegram bio.

• /antibio [on/off] - Enable/disable bio link checking
• /free [user] - Exempt user from bio checks
• /unfree [user] - Remove user from exemption
• /freelist - List exempt users

When enabled:
• Users with links in their bio will have messages deleted
• They receive a warning to remove links from bio
• Admins and exempt users bypass this check
• Detects @usernames, t.me, http links, and more

This is group-specific - exemptions only work in the group where they were granted.`,

  antiabuse: `<b>Antiabuse Settings</b>

Protect your group from abusive language in multiple languages including Hinglish, Hindi, English, Urdu, Tamil, Telugu, Bengali, and Punjabi.

• /antiabuse - Check current status
• /antiabuse on - Enable abuse detection
• /antiabuse off - Disable abuse detection

When enabled:
• Messages containing abuse words are deleted instantly
• User receives a warning message
• Admins and owners are exempt

The system detects:
• Common abuse words in 8+ languages
• Variations with numbers/symbols (like ch0d, f*ck)
• Transliterated Hindi/Urdu abuse`,
}

// Function to get the start keyboard
function getStartKeyboard(): InlineKeyboardMarkup {
  return {
    inline_keyboard: [
      [
        { text: "Add me to your chat!", url: `https://t.me/${BOT_USERNAME}?startgroup=true` },
        { text: "⭐ Get your own Sukoon", callback_data: "start_custom_instances" },
      ],
    ],
  }
}

// Function to get the back button that returns to start
function getStartBackKeyboard(): InlineKeyboardMarkup {
  return {
    inline_keyboard: [[{ text: "« Back", callback_data: "back_to_start" }]],
  }
}

function getHelpKeyboard(topic: string): InlineKeyboardMarkup {
  return {
    inline_keyboard: [[{ text: "« Back", callback_data: "help_back" }]],
  }
}

export async function handleStart(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type !== "private") {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Hey! I'm ${BOT_NAME}, a group management bot. Add me to your group and make me admin to get started!`,
      { reply_to_message_id: ctx.message.message_id },
    )
    return
  }

  // Check for start parameters (e.g., connect_chatid, help_topic)
  if (ctx.args.length > 0) {
    const param = ctx.args[0]
    if (param.startsWith("connect_")) {
      const chatId = param.replace("connect_", "")
      ctx.args = [chatId]
      const { handleConnect } = await import("./admin")
      await handleConnect(ctx)
      return
    }
    // Handle help deep links
    if (param.startsWith("help_")) {
      const topic = param.replace("help_", "")
      const helpText = HELP_TEXTS[topic]
      const keyboard = getHelpKeyboard(topic)
      await ctx.bot.sendMessage(ctx.chat.id, helpText, {
        parse_mode: "HTML",
        reply_markup: keyboard,
        disable_web_page_preview: true,
      })
      return
    }
    // Handle rules deep link
    if (param.startsWith("rules_")) {
      const chatId = param.replace("rules_", "")
      const { data: settings } = await supabase.from("chat_settings").select("rules").eq("chat_id", chatId).single()

      if (settings?.rules) {
        await ctx.bot.sendMessage(ctx.chat.id, `<b>Rules:</b>\n\n${settings.rules}`, { parse_mode: "HTML" })
      } else {
        await ctx.bot.sendMessage(ctx.chat.id, "No rules set for this chat.")
      }
      return
    }
  }

  const welcomeText = `Hey there! My name is ${BOT_NAME} - I'm here to help you manage your groups! Use /help to find out how to use me to my full potential.

Join my <a href="${UPDATES_CHANNEL}">news channel</a> to get information on all the latest updates.

Check /privacy to view the privacy policy, and interact with your data.`

  const keyboard = getStartKeyboard()

  await ctx.bot.sendMessage(ctx.chat.id, welcomeText, {
    parse_mode: "HTML",
    reply_markup: keyboard,
    disable_web_page_preview: true,
  })
}

export async function handleHelp(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type !== "private") {
    const botInfo = await ctx.bot.getMe()
    await ctx.bot.sendMessage(ctx.chat.id, "Contact me in PM for help!", {
      reply_markup: {
        inline_keyboard: [[{ text: "Help", url: `https://t.me/${botInfo.username}?start=help_main` }]],
      },
    })
    return
  }

  const topic = ctx.args[0]?.toLowerCase()
  if (topic) {
    const helpText = HELP_TEXTS[topic]
    const keyboard = getHelpKeyboard(topic)
    await ctx.bot.sendMessage(ctx.chat.id, helpText, {
      parse_mode: "HTML",
      reply_markup: keyboard,
      disable_web_page_preview: true,
    })
    return
  }

  const keyboard = getMainHelpKeyboard()
  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>${BOT_NAME} Help</b>\n\nSelect a category below to learn about my features:`,
    { parse_mode: "HTML", reply_markup: keyboard },
  )
}

function getMainHelpKeyboard(): InlineKeyboardMarkup {
  return {
    inline_keyboard: [
      [
        { text: "Admin", callback_data: "help_admin" },
        { text: "Antiflood", callback_data: "help_antiflood" },
        { text: "AntiRaid", callback_data: "help_antiraid" },
      ],
      [
        { text: "Approval", callback_data: "help_approval" },
        { text: "Bans", callback_data: "help_bans" },
        { text: "Blocklists", callback_data: "help_blocklist" },
      ],
      [
        { text: "CAPTCHA", callback_data: "help_captcha" },
        { text: "Clean Commands", callback_data: "help_cleancommands" },
        { text: "Clean Service", callback_data: "help_cleanservice" },
      ],
      [
        { text: "Connections", callback_data: "help_connections" },
        { text: "Disabling", callback_data: "help_disabling" },
        { text: "Federations", callback_data: "help_federation" },
      ],
      [
        { text: "Filters", callback_data: "help_filters" },
        { text: "Formatting", callback_data: "help_formatting" },
        { text: "Greetings", callback_data: "help_greetings" },
      ],
      [
        { text: "Import/Export", callback_data: "help_importexport" },
        { text: "Languages", callback_data: "help_languages" },
        { text: "Locks", callback_data: "help_locks" },
      ],
      [
        { text: "Log Channels", callback_data: "help_logchannels" },
        { text: "Misc", callback_data: "help_misc" },
        { text: "Notes", callback_data: "help_notes" },
      ],
      [
        { text: "Pin", callback_data: "help_pin" },
        { text: "Privacy", callback_data: "help_privacy" },
        { text: "Purges", callback_data: "help_purges" },
      ],
      [
        { text: "Reports", callback_data: "help_reports" },
        { text: "Rules", callback_data: "help_rules" },
        { text: "Topics", callback_data: "help_topics" },
      ],
      [
        { text: "Warnings", callback_data: "help_warnings" },
        { text: "Silent Power", callback_data: "help_silentpower" },
        { text: "Extra", callback_data: "help_extra" },
      ],
      [{ text: "Bio Check", callback_data: "help_biocheck" }],
      [{ text: "Antiabuse", callback_data: "help_antiabuse" }],
      [{ text: "⭐ Custom Instances", callback_data: "help_custom_instances" }],
      [{ text: "📚 Docs Website", url: DOCS_URL }],
    ],
  }
}

export async function handlePrivacy(ctx: CommandContext): Promise<void> {
  const privacyText = `
<b>🔐 Privacy Policy</b>

<b>Data We Collect:</b>
• User ID, username, first/last name
• Chat ID and chat name
• Messages only when they trigger bot commands

<b>How We Use Data:</b>
• To provide moderation features
• To store your settings and preferences
• To enforce bans across federations

<b>Data Storage:</b>
• All data is stored securely in our database
• We do not share your data with third parties
• You can request data deletion at any time

<b>Your Rights:</b>
• /gdpr - Download your data
• /deldata - Request data deletion
• Contact @MissSukoon_bot for privacy concerns

Use /privacy to view this policy anytime.
`
  await ctx.bot.sendMessage(ctx.chat.id, privacyText, { parse_mode: "HTML" })
}

export async function handleGdpr(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id

  try {
    // Fetch user's data from various tables
    const [userData, warns, approvals, notes] = await Promise.all([
      supabase.from("users").select("*").eq("user_id", userId).maybeSingle(),
      supabase.from("warnings").select("*").eq("user_id", userId),
      supabase.from("approved_users").select("*").eq("user_id", userId),
      supabase.from("notes").select("chat_id, note_name").eq("creator_id", userId),
    ])

    const data = {
      user: userData.data,
      warnings: warns.data || [],
      approvals: approvals.data || [],
      notes_created: notes.data || [],
    }

    const text = `
<b>📋 Your Data (GDPR Request)</b>

<b>User Info:</b>
• ID: <code>${userId}</code>
• Username: ${ctx.user.username ? "@" + ctx.user.username : "Not set"}
• First Name: ${ctx.user.first_name}

<b>Warnings:</b> ${data.warnings.length} total
<b>Approved in:</b> ${data.approvals.length} chats
<b>Notes Created:</b> ${data.notes_created.length}

Use /deldata to request deletion of all your data.
`

    await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
  } catch (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to fetch your data. Please try again later.")
  }
}

export async function handleDelData(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id

  try {
    // Delete user data from various tables
    await Promise.all([
      supabase.from("warnings").delete().eq("user_id", userId),
      supabase.from("approved_users").delete().eq("user_id", userId),
      supabase.from("user_settings").delete().eq("user_id", userId),
    ])

    const text = `
<b>🗑️ Data Deletion Complete</b>

Your personal data has been deleted from our systems:
• Warnings removed
• Approval records removed
• Personal settings removed

Note: Some data may be retained for security purposes (e.g., federation bans).
`

    await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
  } catch (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to delete your data. Please try again later.")
  }
}

export async function handleSetLang(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "Language settings are only available in groups.")
    return
  }

  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to change language settings.")
    return
  }

  const lang = ctx.args[0]?.toLowerCase()

  if (!lang) {
    // Show language selection keyboard
    const buttons = Object.entries(languageInfo).map(([code, info]) => ({
      text: `${info.flag} ${info.name}`,
      callback_data: `setlang_${code}`,
    }))

    // Create rows of 2 buttons each
    const rows: { text: string; callback_data: string }[][] = []
    for (let i = 0; i < buttons.length; i += 2) {
      rows.push(buttons.slice(i, i + 2))
    }

    await ctx.bot.sendMessage(ctx.chat.id, "Select your preferred language:", {
      reply_markup: { inline_keyboard: rows },
    })
    return
  }

  if (!languages[lang]) {
    const available = Object.entries(languageInfo)
      .map(([code, langInfo]) => `${langInfo.flag} ${code} - ${langInfo.name}`)
      .join("\n")
    await ctx.bot.sendMessage(ctx.chat.id, `Invalid language code!\n\nAvailable languages:\n${available}`)
    return
  }

  const success = await setChatLanguage(ctx.chat.id, lang)

  if (!success) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to update language.")
    return
  }

  const info = languageInfo[lang]
  await ctx.bot.sendMessage(ctx.chat.id, `${info.flag} Language has been set to ${info.name}!`)
}

export async function handleLanguage(ctx: CommandContext): Promise<void> {
  if (ctx.args.length > 0) {
    await handleSetLang(ctx)
    return
  }

  const currentLang = await getChatLanguage(ctx.chat.id)
  const info = languageInfo[currentLang] || languageInfo.en

  const available = Object.entries(languageInfo)
    .map(([code, langInfo]) => `${langInfo.flag} ${code} - ${langInfo.name}${code === currentLang ? " ✓" : ""}`)
    .join("\n")

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Language Settings</b>\n\nCurrent language: ${info.flag} ${info.name}\n\n<b>Available languages:</b>\n${available}\n\nUse /setlang [code] to change language.`,
    { parse_mode: "HTML" },
  )
}

export async function handleSetLangCallback(ctx: CommandContext, langCode: string): Promise<void> {
  const success = await setChatLanguage(ctx.chat.id, langCode)

  if (!success) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to update language.")
    return
  }

  const info = languageInfo[langCode]
  await ctx.bot.sendMessage(ctx.chat.id, `${info.flag} Language has been set to ${info.name}!`)
}

export async function handleCallbackQuery(query: TelegramCallbackQuery, botInstance?: TelegramBot): Promise<void> {
  const chatId = query.message?.chat.id
  const messageId = query.message?.message_id
  const data = query.data
  const activeBot = botInstance || (await import("../bot")).bot

  if (!chatId || !messageId || !data) return

  try {
    // Answer callback query first
    await activeBot.answerCallbackQuery(query.id)

    if (data === "back_to_start") {
      const startText = `Hey there! My name is <b>${BOT_NAME}</b> - I'm here to help you manage your groups! Use /help to find out how to use me to my full potential.

Join my <a href="${UPDATES_CHANNEL}">news channel</a> to get information on all the latest updates.

Check /privacy to view the privacy policy, and interact with your data.`

      await activeBot.editMessageText(chatId, messageId, startText, {
        parse_mode: "HTML",
        reply_markup: getStartKeyboard(),
        disable_web_page_preview: true,
      })
      return
    }

    if (data === "help_back") {
      await activeBot.editMessageText(chatId, messageId, getHelpText(), {
        parse_mode: "HTML",
        reply_markup: getMainHelpKeyboard(),
        disable_web_page_preview: true,
      })
      return
    }

    if (data === "start_custom_instances") {
      await activeBot.editMessageText(chatId, messageId, HELP_TEXTS["custom_instances"], {
        parse_mode: "HTML",
        reply_markup: getStartBackKeyboard(),
        disable_web_page_preview: true,
      })
      return
    }

    if (data.startsWith("help_")) {
      const topic = data.replace("help_", "")
      const helpText = HELP_TEXTS[topic]

      if (helpText) {
        await activeBot.editMessageText(chatId, messageId, helpText, {
          parse_mode: "HTML",
          reply_markup: getHelpKeyboard(topic),
          disable_web_page_preview: true,
        })
      }
      return
    }

    if (data.startsWith("setlang_")) {
      const lang = data.replace("setlang_", "")

      if (!languages[lang]) {
        await activeBot.answerCallbackQuery(query.id, { text: "Invalid language!", show_alert: true })
        return
      }

      const success = await setChatLanguage(chatId, lang)

      if (success) {
        const info = languageInfo[lang]
        await activeBot.editMessageText(chatId, messageId, `${info.flag} Language changed to ${info.name}!`, {
          parse_mode: "HTML",
        })
        await activeBot.answerCallbackQuery(query.id, { text: `Language set to ${info.name}!` })
      } else {
        await activeBot.answerCallbackQuery(query.id, { text: "Failed to change language", show_alert: true })
      }
      return
    }

    // Handle other callbacks...
  } catch (error) {
    console.error("Callback query error:", error)
  }
}

function getHelpText(): string {
  return `<b>${BOT_NAME} Help</b>\n\nSelect a category below to learn about my features:`
}

export async function handleId(ctx: CommandContext): Promise<void> {
  let text = ""

  if (ctx.replyToMessage?.from) {
    const user = ctx.replyToMessage.from
    text = `<b>User ID:</b> <code>${user.id}</code>\n`
    text += `<b>First Name:</b> ${user.first_name}\n`
    if (user.username) text += `<b>Username:</b> @${user.username}\n`
  } else {
    text = `<b>Your ID:</b> <code>${ctx.user.id}</code>\n`
    text += `<b>First Name:</b> ${ctx.user.first_name}\n`
    if (ctx.user.username) text += `<b>Username:</b> @${ctx.user.username}\n`
  }

  if (ctx.chat.type !== "private") {
    text += `\n<b>Chat ID:</b> <code>${ctx.chat.id}</code>\n`
    text += `<b>Chat Title:</b> ${ctx.chat.title}`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleStats(ctx: CommandContext): Promise<void> {
  if (!ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command is for sudoers only.")
    return
  }

  const { count: usersCount } = await supabase.from("users").select("*", { count: "exact", head: true })
  const { count: chatsCount } = await supabase.from("chats").select("*", { count: "exact", head: true })
  const { count: fedsCount } = await supabase.from("federations").select("*", { count: "exact", head: true })

  const text = `
<b>${BOT_NAME} Statistics</b>

👥 <b>Users:</b> ${usersCount || 0}
💬 <b>Chats:</b> ${chatsCount || 0}
🌐 <b>Federations:</b> ${fedsCount || 0}
`

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleMarkdown(ctx: CommandContext): Promise<void> {
  const text = HELP_TEXTS.formatting

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleKickme(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command only works in groups.")
    return
  }

  try {
    await ctx.bot.kickChatMember(ctx.chat.id, ctx.user.id)
    await ctx.bot.unbanChatMember(ctx.chat.id, ctx.user.id)
    await ctx.bot.sendMessage(ctx.chat.id, `${ctx.user.first_name} has left the chat!`)
  } catch {
    await ctx.bot.sendMessage(ctx.chat.id, "I couldn't kick you. Maybe I'm not an admin?")
  }
}

// ===========================================
// GLOBAL BAN COMMANDS (Sudoers only)
// ===========================================

export async function handleGban(ctx: CommandContext): Promise<void> {
  // Only owner and sudoers can use gban
  if (ctx.user.id !== OWNER_ID && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command is for bot owner and sudoers only.")
    return
  }

  const { getTargetUser } = await import("../utils")
  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to globally ban.")
    return
  }

  // Can't gban owner or sudoers
  if (target.userId === OWNER_ID) {
    await ctx.bot.sendMessage(ctx.chat.id, "You cannot globally ban the bot owner.")
    return
  }

  const reason = ctx.args.slice(1).join(" ") || "No reason"

  await supabase
    .from("global_bans")
    .upsert({ user_id: target.userId, reason, banned_by: ctx.user.id }, { onConflict: "user_id" })

  // Get all chats where the bot has ban power and ban the user
  const { data: chats } = await supabase.from("chats").select("chat_id")

  let bannedCount = 0
  if (chats) {
    for (const chat of chats) {
      try {
        await ctx.bot.banChatMember(chat.chat_id, target.userId)
        bannedCount++
      } catch {
        // Bot might not have ban power in all chats
      }
    }
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Global Ban</b>\n\nUser: <code>${target.userId}</code>\nReason: ${reason}\nBanned in: ${bannedCount} chats`,
    { parse_mode: "HTML" },
  )
}

export async function handleUngban(ctx: CommandContext): Promise<void> {
  if (ctx.user.id !== OWNER_ID && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command is for bot owner and sudoers only.")
    return
  }

  const { getTargetUser } = await import("../utils")
  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to ungban.")
    return
  }

  await supabase.from("global_bans").delete().eq("user_id", target.userId)

  // Unban from all chats
  const { data: chats } = await supabase.from("chats").select("chat_id")

  let unbannedCount = 0
  if (chats) {
    for (const chat of chats) {
      try {
        await ctx.bot.unbanChatMember(chat.chat_id, target.userId, { only_if_banned: true })
        unbannedCount++
      } catch {
        // Ignore errors
      }
    }
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Global Unban</b>\n\nUser: <code>${target.userId}</code>\nUnbanned in: ${unbannedCount} chats`,
    { parse_mode: "HTML" },
  )
}

export async function handleAddSudo(ctx: CommandContext): Promise<void> {
  // Only owner can add sudoers
  if (ctx.user.id !== OWNER_ID) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the bot owner can add sudoers.")
    return
  }

  const { getTargetUser } = await import("../utils")
  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to add as sudoer.")
    return
  }

  if (target.userId === OWNER_ID) {
    await ctx.bot.sendMessage(ctx.chat.id, "The owner is already a sudoer by default.")
    return
  }

  await supabase.from("sudo_users").upsert({ user_id: target.userId, added_by: ctx.user.id }, { onConflict: "user_id" })

  await ctx.bot.sendMessage(ctx.chat.id, `<b>New Sudoer Added</b>\n\nUser ID: <code>${target.userId}</code>`, {
    parse_mode: "HTML",
  })
}

export async function handleRmSudo(ctx: CommandContext): Promise<void> {
  // Only owner can remove sudoers
  if (ctx.user.id !== OWNER_ID) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the bot owner can remove sudoers.")
    return
  }

  const { getTargetUser } = await import("../utils")
  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to remove from sudoers.")
    return
  }

  await supabase.from("sudo_users").delete().eq("user_id", target.userId)

  await ctx.bot.sendMessage(ctx.chat.id, `<b>Sudoer Removed</b>\n\nUser ID: <code>${target.userId}</code>`, {
    parse_mode: "HTML",
  })
}

export async function handleSudoList(ctx: CommandContext): Promise<void> {
  if (ctx.user.id !== OWNER_ID && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command is for bot owner and sudoers only.")
    return
  }

  const { data: sudoers } = await supabase.from("sudo_users").select("user_id, created_at")

  let text = `<b>Sudoer List</b>\n\n`
  text += `<b>Owner:</b> <code>${OWNER_ID}</code>\n\n`

  if (sudoers && sudoers.length > 0) {
    text += `<b>Sudoers:</b>\n`
    for (const sudo of sudoers) {
      text += `• <code>${sudo.user_id}</code>\n`
    }
  } else {
    text += `No additional sudoers.`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleGbanList(ctx: CommandContext): Promise<void> {
  if (ctx.user.id !== OWNER_ID && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command is for bot owner and sudoers only.")
    return
  }

  const { data: gbans } = await supabase
    .from("global_bans")
    .select("user_id, reason, created_at")
    .order("created_at", { ascending: false })
    .limit(50)

  if (!gbans || gbans.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No globally banned users.")
    return
  }

  let text = `<b>Global Ban List</b>\n\n`
  for (const gban of gbans) {
    text += `• <code>${gban.user_id}</code> - ${gban.reason || "No reason"}\n`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// Check if user is globally banned
export async function checkGlobalBan(userId: number): Promise<boolean> {
  const { data } = await supabase.from("global_bans").select("user_id").eq("user_id", userId).maybeSingle()

  return !!data
}

// Clone handling
export async function handleClone(ctx: CommandContext): Promise<void> {
  // Clone command only works in private chat
  if (ctx.chat.type !== "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "Please use this command in private chat for security reasons.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const token = ctx.args[0]

  if (!token) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `<b>Clone Your Own Sukoon</b>

To create your own instance of ${BOT_NAME}:

1. Go to BotFather on Telegram
2. Send /newbot and follow the instructions
3. Copy the bot token you receive
4. Use: <code>/clone YOUR_BOT_TOKEN</code>

<b>Note:</b> Each user can only create ONE clone. If you already have a clone, use /rmclone first to remove it before creating a new one.`,
      { parse_mode: "HTML" },
    )
    return
  }

  // Delete the message containing the token for security
  try {
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.message.message_id)
  } catch {
    // Ignore if can't delete
  }

  const userId = ctx.user.id.toString()

  // Check if user already has a clone - ONE clone per user limit
  const { data: existingUserClone, error: checkError } = await supabase
    .from("bot_clones")
    .select("bot_username, bot_name, bot_id")
    .eq("user_id", userId)
    .maybeSingle()

  if (checkError) {
    console.error("Error checking existing clone:", checkError)
    await ctx.bot.sendMessage(ctx.chat.id, "An error occurred. Please try again later.")
    return
  }

  if (existingUserClone) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `<b>Clone Limit Reached</b>

You already have a clone: @${existingUserClone.bot_username}

Each user can only have <b>ONE</b> clone to prevent server overload.

To create a new clone, first remove your existing one:
<code>/rmclone @${existingUserClone.bot_username}</code>`,
      { parse_mode: "HTML" },
    )
    return
  }

  // Validate token format (should be like: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz)
  const tokenRegex = /^\d+:[A-Za-z0-9_-]{35,}$/
  if (!tokenRegex.test(token)) {
    await ctx.bot.sendMessage(ctx.chat.id, "Invalid bot token format. Please check and try again.")
    return
  }

  // Send processing message
  const processingMsg = await ctx.bot.sendMessage(ctx.chat.id, "Processing your clone request...")

  // Verify the token by getting bot info
  let botInfo: { id: number; username: string; first_name: string }
  try {
    const response = await fetch(`https://api.telegram.org/bot${token}/getMe`)
    const data = await response.json()

    if (!data.ok) {
      await ctx.bot.editMessageText("Invalid bot token. Please make sure you copied it correctly from BotFather.", {
        chat_id: ctx.chat.id,
        message_id: processingMsg.message_id,
      })
      return
    }
    botInfo = data.result
  } catch {
    await ctx.bot.editMessageText("Failed to verify token. Please try again.", {
      chat_id: ctx.chat.id,
      message_id: processingMsg.message_id,
    })
    return
  }

  // Check if this bot is already cloned by anyone
  const { data: existingBotClone } = await supabase
    .from("bot_clones")
    .select("user_id")
    .eq("bot_id", botInfo.id.toString())
    .maybeSingle()

  if (existingBotClone) {
    await ctx.bot.editMessageText(
      "This bot is already registered as a clone. If it's yours, use /rmclone to remove it first.",
      { chat_id: ctx.chat.id, message_id: processingMsg.message_id },
    )
    return
  }

  // Get base URL
  let baseUrl = process.env.NEXT_PUBLIC_APP_URL || process.env.VERCEL_URL || ""
  baseUrl = baseUrl.replace(/^https?:\/\//, "").replace(/\/$/, "")

  if (!baseUrl) {
    await ctx.bot.editMessageText("Server configuration error. Please contact support.", {
      chat_id: ctx.chat.id,
      message_id: processingMsg.message_id,
    })
    return
  }

  const webhookUrl = `https://${baseUrl}/api/telegram/webhook?token=${encodeURIComponent(token)}`

  // Delete any existing webhook first
  try {
    await fetch(`https://api.telegram.org/bot${token}/deleteWebhook`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ drop_pending_updates: true }),
    })
  } catch {
    // Ignore errors
  }

  // Set new webhook
  const webhookResponse = await fetch(`https://api.telegram.org/bot${token}/setWebhook`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      url: webhookUrl,
      allowed_updates: ["message", "callback_query", "chat_member", "my_chat_member"],
      drop_pending_updates: true,
    }),
  })

  const webhookData = await webhookResponse.json()

  if (!webhookData.ok) {
    await ctx.bot.editMessageText(`Failed to set webhook: ${webhookData.description}`, {
      chat_id: ctx.chat.id,
      message_id: processingMsg.message_id,
    })
    return
  }

  // Save clone to database
  const { error: insertError } = await supabase.from("bot_clones").insert({
    user_id: userId,
    bot_token: token,
    bot_id: botInfo.id.toString(),
    bot_username: botInfo.username,
    bot_name: botInfo.first_name,
  })

  if (insertError) {
    console.error("Database insert error:", insertError)
    // Rollback webhook
    await fetch(`https://api.telegram.org/bot${token}/deleteWebhook`)
    await ctx.bot.editMessageText("Failed to save clone to database. Please try again.", {
      chat_id: ctx.chat.id,
      message_id: processingMsg.message_id,
    })
    return
  }

  // Success message
  await ctx.bot.editMessageText(
    `<b>Clone Successful!</b>

Your bot @${botInfo.username} is now a clone of ${BOT_NAME}!

<b>What's next?</b>
1. Go to your bot @${botInfo.username}
2. Send /start to test it
3. Add it to a group and make it admin

<b>Your Bot Info:</b>
- Name: ${botInfo.first_name}
- Username: @${botInfo.username}
- ID: <code>${botInfo.id}</code>

Use /clones to see your clone.
Use /rmclone @${botInfo.username} to remove it.`,
    { chat_id: ctx.chat.id, message_id: processingMsg.message_id, parse_mode: "HTML" },
  )
}

// Clones list command
export async function handleClones(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id.toString()

  const { data: clones } = await supabase
    .from("bot_clones")
    .select("bot_username, bot_name, bot_id, created_at")
    .eq("user_id", userId)

  if (!clones || clones.length === 0) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have any cloned bots yet.\n\nUse /clone to create your own instance of Sukoon!",
    )
    return
  }

  let text = `<b>Your Cloned Bots</b>\n\n`
  for (const clone of clones) {
    text += `- @${clone.bot_username} (${clone.bot_name})\n  ID: <code>${clone.bot_id}</code>\n\n`
  }

  text += `Total: ${clones.length}/1 clone(s)`

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// Remove clone command
export async function handleRmClone(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id.toString()
  let botUsername = ctx.args[0]

  if (!botUsername) {
    // If no username provided, check if user has a clone and show it
    const { data: userClone } = await supabase
      .from("bot_clones")
      .select("bot_username")
      .eq("user_id", userId)
      .maybeSingle()

    if (userClone) {
      await ctx.bot.sendMessage(ctx.chat.id, `Usage: <code>/rmclone @${userClone.bot_username}</code>`, {
        parse_mode: "HTML",
      })
    } else {
      await ctx.bot.sendMessage(ctx.chat.id, "You don't have any clones to remove.")
    }
    return
  }

  // Remove @ if present
  botUsername = botUsername.replace(/^@/, "")

  // Find the clone
  const { data: clone } = await supabase
    .from("bot_clones")
    .select("bot_token, bot_username, user_id")
    .eq("bot_username", botUsername)
    .maybeSingle()

  if (!clone) {
    await ctx.bot.sendMessage(ctx.chat.id, `No clone found with username @${botUsername}`)
    return
  }

  // Check ownership
  if (clone.user_id.toString() !== userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can only remove your own clones.")
    return
  }

  // Delete webhook
  try {
    await fetch(`https://api.telegram.org/bot${clone.bot_token}/deleteWebhook`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ drop_pending_updates: true }),
    })
  } catch {
    // Ignore errors
  }

  // Delete from database
  const { error } = await supabase.from("bot_clones").delete().eq("bot_username", botUsername).eq("user_id", userId)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to remove clone. Please try again.")
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `Clone @${botUsername} has been removed successfully.\n\nYou can now create a new clone with /clone.`,
  )
}

// Function to get the help buttons
function getHelpButtons(): InlineKeyboardMarkup {
  return {
    inline_keyboard: [
      [
        { text: "Admin", callback_data: "help_admin" },
        { text: "Bans", callback_data: "help_bans" },
        { text: "Mutes", callback_data: "help_mutes" },
      ],
      [
        { text: "Warnings", callback_data: "help_warnings" },
        { text: "Blocklist", callback_data: "help_blocklist" },
        { text: "Locks", callback_data: "help_locks" },
      ],
      [
        { text: "Antiflood", callback_data: "help_antiflood" },
        { text: "AntiRaid", callback_data: "help_antiraid" },
        { text: "Approval", callback_data: "help_approval" },
      ],
      [
        { text: "Notes", callback_data: "help_notes" },
        { text: "Filters", callback_data: "help_filters" },
        { text: "Welcome", callback_data: "help_welcome" },
      ],
      [
        { text: "Rules", callback_data: "help_rules" },
        { text: "Reports", callback_data: "help_reports" },
        { text: "Connections", callback_data: "help_connections" },
      ],
      [
        { text: "Federation", callback_data: "help_federation" },
        { text: "CAPTCHA", callback_data: "help_captcha" },
        { text: "Disabling", callback_data: "help_disabling" },
      ],
      [
        { text: "Clean Cmds", callback_data: "help_cleancommands" },
        { text: "Clean Service", callback_data: "help_cleanservice" },
        { text: "Logging", callback_data: "help_logging" },
      ],
      [
        { text: "Extra", callback_data: "help_extra" },
        { text: "Silent Power", callback_data: "help_silentpower" },
        { text: "BioCheck", callback_data: "help_biocheck" },
      ],
      [{ text: "Antiabuse", callback_data: "help_antiabuse" }],
      [{ text: "Close", callback_data: "help_close" }],
    ],
  }
}
