import type { TelegramUpdate, CommandContext } from "./types"
import {
  parseCommand,
  isUserAdmin,
  isUserOwner,
  isSudoer,
  ensureUser,
  ensureChat,
  isCommandDisabled,
  isApproved,
} from "./utils"
import { checkLocks, checkBlocklist, checkFlood, executeAction, deleteFloodMessages } from "./handlers/antispam"
import { checkFilters, sendWelcome, sendGoodbye, sendNote } from "./handlers/content"
import { handleCallbackQuery, checkGlobalBan } from "./handlers/misc"
import { checkFedBan } from "./handlers/federation"
import { bot as defaultBot, type TelegramBot } from "./bot" // Import TelegramBot class
import { checkAntiRaid, getRaidActionTime } from "./handlers/antiraid"
import { shouldCleanService } from "./handlers/clean-service"
import { shouldCleanCommand } from "./handlers/clean-commands"
import { checkAndSendCaptcha, handleCaptchaCallback, checkCaptchaAnswer, hasPendingCaptcha } from "./handlers/captcha"
import * as antiraid from "./handlers/antiraid"
import * as cleanCommands from "./handlers/clean-commands"
import * as cleanService from "./handlers/clean-service"
import * as captcha from "./handlers/captcha"
import * as silentpower from "./handlers/silentpower"
import * as owner from "./handlers/owner"
import { isBlacklistedChat, isBlacklistedUser } from "./handlers/owner"
import * as biocheck from "./handlers/biocheck"
import { checkBioLinks } from "./handlers/biocheck"
import * as antiabuse from "./handlers/antiabuse"
import { checkAbuse } from "./handlers/antiabuse"

// Import all handlers
import * as moderation from "./handlers/moderation"
import * as antispam from "./handlers/antispam"
import * as content from "./handlers/content"
import * as federation from "./handlers/federation"
import * as admin from "./handlers/admin"
import * as misc from "./handlers/misc"
import { handleAfk, handleBrb, checkAfkMention, checkAfkReturn } from "./handlers/afk"

// Command registry with admin/user classification
const adminCommands = new Set([
  "ban",
  "dban",
  "sban",
  "tban",
  "unban",
  "mute",
  "dmute",
  "smute",
  "tmute",
  "unmute",
  "kick",
  "dkick",
  "skick",
  "warn",
  "dwarn",
  "resetwarns",
  "setwarnlimit",
  "setwarnmode",
  "blocklist",
  "blacklist",
  "addblocklist",
  "addblacklist",
  "addblock",
  "block", // Added /block as alias
  "rmblocklist",
  "rmblacklist",
  "unblock",
  "rmblock", // Added /rmblock as alias
  "blocklistmode",
  "blacklistmode",
  "lock",
  "unlock",
  "setflood",
  "setfloodmode",
  "floodmode",
  "save",
  "clear",
  "filter",
  "stop",
  "setwelcome",
  "setgoodbye",
  "welcome",
  "goodbye",
  "setrules",
  "clearrules",
  "privaterules",
  "newfed",
  "delfed",
  "joinfed",
  "leavefed",
  "fedpromote",
  "feddemote",
  "renamefed",
  "fedtransfer",
  "quietfed",
  "pin",
  "unpin",
  "unpinall",
  "purge",
  "del",
  "approve",
  "unapprove",
  "disabled",
  "enable",
  "setlog",
  "logchannel",
  "unsetlog",
  "connect",
  "disconnect",
  "antiraid",
  "raidtime",
  "raidactiontime",
  "autoantiraid",
  "cleancommand",
  "keepcommand",
  "cleanservice",
  "keepservice",
  "nocleanservice",
  "captcha",
  "captchamode",
  "captchamutetime",
  "captchakick",
  "promote",
  "demote",
  "title",
  "setgtitle",
  "setgpic",
  "delgpic",
  "setgdesc",
  "setsticker",
  "delsticker",
  "invitelink",
  "clearflood",
  "raidmode",
  "mod",
  "unmod",
  "mods",
  "antibio",
  "free",
  "unfree",
  "freelist",
  "antiabuse",
])

const userCommands = new Set([
  "start",
  "help",
  "id",
  "info",
  "markdown",
  "kickme",
  "warns",
  "notes",
  "saved",
  "get",
  "filters",
  "rules",
  "locks",
  "flood",
  "approved",
  "disabled",
  "reports",
  "fban",
  "unfban",
  "fedinfo",
  "fedadmins",
  "myfeds",
  "fedstat",
  "chatfed",
  "cleancommandtypes",
  "cleanservicetypes",
  "clone",
  "clones",
  "rmclone",
  "privacy",
  "setlang",
  "language",
  "gdpr",
  "deldata",
  "afk",
  "brb",
  "approval",
  "admin", // Add admin command
])

// Command registry
const commands: Record<string, (ctx: CommandContext) => Promise<void>> = {
  // Moderation - Bans
  ban: moderation.handleBan,
  dban: moderation.handleDBan,
  sban: moderation.handleSBan,
  tban: moderation.handleTBan,
  unban: moderation.handleUnban,

  // Moderation - Mutes
  mute: moderation.handleMute,
  dmute: moderation.handleDMute,
  smute: moderation.handleSMute,
  tmute: moderation.handleTMute,
  unmute: moderation.handleUnmute,

  // Moderation - Kicks
  kick: moderation.handleKick,
  dkick: moderation.handleDKick,
  skick: moderation.handleSKick,

  // Moderation - Warnings
  warn: moderation.handleWarn,
  dwarn: moderation.handleDWarn,
  warns: moderation.handleWarns,
  resetwarns: moderation.handleResetWarns,
  setwarnlimit: moderation.handleSetWarnLimit,
  setwarnmode: moderation.handleSetWarnAction,

  // Anti-spam - Blocklist
  blocklist: antispam.handleBlocklist,
  blacklist: antispam.handleBlocklist,
  addblocklist: antispam.handleAddBlocklist,
  addblacklist: antispam.handleAddBlocklist,
  addblock: antispam.handleAddBlocklist,
  block: antispam.handleAddBlocklist, // Added /block as alias
  rmblocklist: antispam.handleUnBlocklist,
  rmblacklist: antispam.handleUnBlocklist,
  unblock: antispam.handleUnBlocklist,
  rmblock: antispam.handleUnBlocklist, // Added /rmblock as alias
  blocklistmode: antispam.handleBlocklistMode,
  blacklistmode: antispam.handleBlocklistMode,

  // Anti-spam - Locks
  lock: antispam.handleLock,
  unlock: antispam.handleUnlock,
  locks: antispam.handleLocks,

  // Anti-spam - Flood
  flood: antispam.handleFlood,
  setflood: antispam.handleSetFlood,
  setfloodmode: antispam.handleSetFloodMode,
  floodmode: antispam.handleSetFloodMode,
  clearflood: antispam.handleClearFlood,

  // Content - Notes
  save: content.handleSave,
  get: content.handleGet,
  notes: content.handleNotes,
  saved: content.handleNotes,
  clear: content.handleClear,

  // Content - Filters
  filter: content.handleFilter,
  stop: content.handleStop,
  filters: content.handleFilters,

  // Content - Welcome
  welcome: content.handleWelcome,
  setwelcome: content.handleSetWelcome,
  goodbye: content.handleGoodbye,
  setgoodbye: content.handleSetGoodbye,

  // Content - Rules
  rules: content.handleRules,
  setrules: content.handleSetRules,
  clearrules: content.handleClearRules,
  privaterules: content.handlePrivateRules,

  // Federation
  newfed: federation.handleNewFed,
  delfed: federation.handleDelFed,
  joinfed: federation.handleJoinFed,
  leavefed: federation.handleLeaveFed,
  fedinfo: federation.handleFedInfo,
  fban: federation.handleFBan,
  unfban: federation.handleUnFBan,
  fedadmins: federation.handleFedAdmins,
  fedpromote: federation.handleFedPromote,
  feddemote: federation.handleFedDemote,
  renamefed: federation.handleRenameFed,
  fedtransfer: federation.handleFedTransfer,
  myfeds: federation.handleMyFeds,
  fedstat: federation.handleFedStat,
  quietfed: federation.handleQuietFed,
  chatfed: federation.handleChatFed,

  // Admin
  adminlist: admin.handleAdmins,
  admins: admin.handleAdmins,
  pin: admin.handlePin,
  unpin: admin.handleUnpin,
  unpinall: admin.handleUnpinAll,
  purge: admin.handlePurge,
  del: admin.handleDel,
  approve: admin.handleApprove,
  unapprove: admin.handleUnapprove,
  approved: admin.handleApproved,
  disable: admin.handleDisable,
  enable: admin.handleEnable,
  disabled: admin.handleDisabled,
  report: admin.handleReport,
  reports: admin.handleReports,
  setlog: admin.handleSetLog,
  logchannel: admin.handleSetLog,
  unsetlog: admin.handleUnsetLog,
  connect: admin.handleConnect,
  disconnect: admin.handleDisconnect,
  info: admin.handleInfo,
  promote: admin.handlePromote,
  demote: admin.handleDemote,
  title: admin.handleTitle,
  setgtitle: admin.handleSetGTitle,
  setgpic: admin.handleSetGPic,
  delgpic: admin.handleDelGPic,
  setgdesc: admin.handleSetGDesc,
  setsticker: admin.handleSetSticker,
  delsticker: admin.handleDelSticker,
  invitelink: admin.handleInviteLink,
  approval: admin.handleApproval,
  admin: admin.handleAdminCall, // Add admin command

  // Misc
  start: misc.handleStart,
  help: misc.handleHelp,
  id: misc.handleId,
  markdown: misc.handleMarkdown,
  kickme: misc.handleKickme,
  clone: misc.handleClone,
  clones: misc.handleClones, // Add clones handler
  rmclone: misc.handleRmClone, // Add rmclone handler
  privacy: misc.handlePrivacy,
  setlang: misc.handleSetLang,
  language: misc.handleLanguage,
  gdpr: misc.handleGdpr,
  deldata: misc.handleDelData,

  // AFK commands
  afk: handleAfk,
  brb: handleBrb,

  // AntiRaid
  antiraid: antiraid.handleAntiRaid,
  raidtime: antiraid.handleRaidTime,
  raidactiontime: antiraid.handleRaidActionTime,
  autoantiraid: antiraid.handleAutoAntiRaid,
  raidmode: antiraid.handleRaidMode,

  // Clean Commands
  cleancommand: cleanCommands.handleCleanCommand,
  keepcommand: cleanCommands.handleKeepCommand,
  cleancommandtypes: cleanCommands.handleCleanCommandTypes,

  // Clean Service
  cleanservice: cleanService.handleCleanService,
  keepservice: cleanService.handleKeepService,
  nocleanservice: cleanService.handleKeepService,
  cleanservicetypes: cleanService.handleCleanServiceTypes,

  // CAPTCHA commands
  captcha: captcha.handleCaptcha,
  captchamode: captcha.handleCaptchaMode,
  captchamutetime: captcha.handleCaptchaMuteTime,
  captchakick: captcha.handleCaptchaKick,

  // Silent Power
  mod: silentpower.handleMod,
  unmod: silentpower.handleUnmod,
  muter: silentpower.handleMuter,
  unmuter: silentpower.handleUnmuter,
  mods: silentpower.handleMods,

  // Biocheck commands
  antibio: biocheck.handleAntibio,
  free: biocheck.handleFree,
  unfree: biocheck.handleUnfree,
  freelist: biocheck.handleFreeList,

  // Antiabuse commands
  antiabuse: antiabuse.handleAntiAbuse,

  gban: owner.handleGban,
  ungban: owner.handleUngban,
  broadcast: owner.handleBroadcast,
  blchat: owner.handleBlChat,
  unblchat: owner.handleUnblChat,
  bluser: owner.handleBlUser,
  unbluser: owner.handleUnblUser,
  stats: owner.handleStats,
  addsudo: owner.handleAddSudo,
  rmsudo: owner.handleRmSudo,
  sudolist: owner.handleSudoList,
  gbanlist: owner.handleGbanList,
  bllist: owner.handleBlList,
}

const adminCache = new Map<string, { isAdmin: boolean; isOwner: boolean; timestamp: number }>()
const ADMIN_CACHE_TTL = 30000 // 30 seconds

async function getCachedAdminStatus(chatId: number, userId: number): Promise<{ isAdmin: boolean; isOwner: boolean }> {
  const key = `${chatId}:${userId}`
  const cached = adminCache.get(key)

  if (cached && Date.now() - cached.timestamp < ADMIN_CACHE_TTL) {
    return { isAdmin: cached.isAdmin, isOwner: cached.isOwner }
  }

  const [isAdmin, isOwner] = await Promise.all([isUserAdmin(chatId, userId), isUserOwner(chatId, userId)])

  adminCache.set(key, { isAdmin, isOwner, timestamp: Date.now() })
  return { isAdmin, isOwner }
}

// Clean old cache entries periodically
setInterval(() => {
  const now = Date.now()
  for (const [key, value] of adminCache.entries()) {
    if (now - value.timestamp > ADMIN_CACHE_TTL) {
      adminCache.delete(key)
    }
  }
}, 60000)

export async function handleUpdate(update: TelegramUpdate, bot: TelegramBot = defaultBot): Promise<void> {
  try {
    // Handle callback queries
    if (update.callback_query) {
      const data = update.callback_query.data

      if (data?.startsWith("captcha_verify_")) {
        const userId = Number.parseInt(data.replace("captcha_verify_", ""))
        if (update.callback_query.from.id === userId && update.callback_query.message) {
          await handleCaptchaCallback(update.callback_query.message.chat.id, userId, update.callback_query.id)
          return
        }
        await bot.answerCallbackQuery(update.callback_query.id, {
          text: "This button is not for you!",
          show_alert: true,
        })
        return
      }

      await handleCallbackQuery(update.callback_query, bot)
      return
    }

    // Handle messages
    const message = update.message || update.edited_message
    if (!message) return

    if (message.chat.type !== "private") {
      if (await isBlacklistedChat(message.chat.id)) {
        try {
          await bot.leaveChat(message.chat.id)
        } catch {}
        return
      }
    }

    if (message.from && (await isBlacklistedUser(message.from.id))) {
      return
    }

    const dbPromises: Promise<void>[] = []
    if (message.from) {
      dbPromises.push(ensureUser(message.from).catch(() => {}))
    }
    if (message.chat.type !== "private") {
      dbPromises.push(ensureChat(message.chat).catch(() => {}))
    }
    Promise.all(dbPromises)

    // Handle new members
    if (message.new_chat_members && message.new_chat_members.length > 0) {
      const antiraidActive = await checkAntiRaid(message.chat.id)

      for (const newMember of message.new_chat_members) {
        // Skip bots
        if (newMember.is_bot) continue

        const [isGbanned, isFedBanned] = await Promise.all([
          checkGlobalBan(newMember.id),
          checkFedBan(message.chat.id, newMember.id),
        ])

        if (isGbanned) {
          try {
            await bot.banChatMember(message.chat.id, newMember.id)
            await bot.sendMessage(
              message.chat.id,
              `User ${newMember.first_name} is globally banned and has been removed.`,
            )
          } catch (e) {
            console.log("[v0] Failed to ban globally banned user:", e)
          }
          continue
        }

        if (isFedBanned) {
          try {
            await bot.banChatMember(message.chat.id, newMember.id)
            await bot.sendMessage(
              message.chat.id,
              `User ${newMember.first_name} is federation banned and has been removed.`,
            )
          } catch (e) {
            console.log("[v0] Failed to ban fed-banned user:", e)
          }
          continue
        }

        if (antiraidActive) {
          try {
            const actionTime = await getRaidActionTime(message.chat.id)
            const untilDate = Math.floor(Date.now() / 1000) + actionTime
            await bot.banChatMember(message.chat.id, newMember.id, { until_date: untilDate })
            await bot.sendMessage(
              message.chat.id,
              `AntiRaid is active! ${newMember.first_name} has been temporarily banned.`,
            )
          } catch (e) {
            console.log("[v0] Failed to ban during antiraid:", e)
          }
          continue
        }

        const captchaSent = await checkAndSendCaptcha(message.chat.id, newMember, message.chat)

        // Only send welcome if CAPTCHA not enabled
        if (!captchaSent) {
          await sendWelcome(message.chat.id, newMember, message.chat)
        }
      }

      // Clean service message if enabled
      if (await shouldCleanService(message.chat.id, "join")) {
        try {
          await bot.deleteMessage(message.chat.id, message.message_id)
        } catch (e) {
          // Ignore
        }
      }
      return
    }

    // Handle left member
    if (message.left_chat_member) {
      await sendGoodbye(message.chat.id, message.left_chat_member, message.chat)

      if (await shouldCleanService(message.chat.id, "leave")) {
        try {
          await bot.deleteMessage(message.chat.id, message.message_id)
        } catch (e) {
          // Ignore
        }
      }
      return
    }

    // Handle pinned message service
    if (message.pinned_message) {
      if (await shouldCleanService(message.chat.id, "pin")) {
        try {
          await bot.deleteMessage(message.chat.id, message.message_id)
        } catch (e) {
          // Ignore
        }
      }
      return
    }

    // Skip if no user
    if (!message.from) {
      return
    }

    if (message.chat.type !== "private" && hasPendingCaptcha(message.chat.id, message.from.id)) {
      const text = message.text || ""
      const answered = await checkCaptchaAnswer(message.chat.id, message.from.id, text)

      // Delete their message regardless
      try {
        await bot.deleteMessage(message.chat.id, message.message_id)
      } catch {
        // Ignore
      }

      if (answered) {
        // Send welcome after successful CAPTCHA
        await sendWelcome(message.chat.id, message.from, message.chat)
      }
      return
    }

    const afkPromise = Promise.all([checkAfkReturn(message, bot), checkAfkMention(message, bot)]).catch((e) =>
      console.log("[v0] AFK check error:", e),
    )

    if (message.chat.type !== "private") {
      const abuseResult = await checkAbuse(message, bot)
      if (abuseResult.violated) {
        return
      }
    }

    if (message.chat.type !== "private") {
      const blocklistResult = await checkBlocklist(message)
      if (blocklistResult.matched) {
        console.log("[v0] Blocklist triggered, deleting message")
        try {
          await bot.deleteMessage(message.chat.id, message.message_id)
        } catch {
          // Ignore
        }
        if (blocklistResult.action && blocklistResult.action !== "delete") {
          await executeAction(
            message.chat.id,
            message.from.id,
            blocklistResult.action,
            blocklistResult.time,
            "Blocklist trigger",
          )
        }
        return
      }
    }

    // Check if user is approved (bypass other restrictions)
    const userApproved = message.chat.type !== "private" && (await isApproved(message.chat.id, message.from.id))

    // Run anti-spam checks (if not approved and not in private chat)
    if (!userApproved && message.chat.type !== "private") {
      const bioResult = await checkBioLinks(message, bot)
      if (bioResult.violated) {
        return
      }

      const lockViolation = await checkLocks(message)

      if (lockViolation) {
        try {
          await bot.deleteMessage(message.chat.id, message.message_id)
        } catch {
          // Ignore
        }
        return
      }

      // Check flood
      const floodResult = await checkFlood(message)
      if (floodResult.flooded) {
        if (floodResult.messageIds && floodResult.messageIds.length > 0) {
          deleteFloodMessages(message.chat.id, floodResult.messageIds)
        }

        await executeAction(
          message.chat.id,
          message.from.id,
          floodResult.action || "mute",
          floodResult.time,
          "Flooding",
        )
        await bot.sendMessage(
          message.chat.id,
          `User ${message.from.first_name} has been ${floodResult.action || "muted"} for flooding.`,
        )
        return
      }
    }

    const text = message.text || message.caption || ""

    // Handle @admin trigger
    if (message.chat.type !== "private" && /^@admin\b/i.test(text)) {
      const args = text
        .replace(/^@admin\s*/i, "")
        .split(/\s+/)
        .filter(Boolean)

      const { isAdmin, isOwner } = await getCachedAdminStatus(message.chat.id, message.from.id)

      const ctx: CommandContext = {
        message,
        chat: message.chat,
        user: message.from,
        args,
        replyToMessage: message.reply_to_message,
        isAdmin,
        isOwner,
        isSudoer: await isSudoer(message.from.id),
        bot: bot,
      }
      await admin.handleAdminCall(ctx)
      return
    }

    // Check for note hashtags (#notename)
    const hashtagMatch = text.match(/^#(\w+)/)
    if (hashtagMatch) {
      await sendNote(message.chat.id, hashtagMatch[1], message.from, message.chat, message.message_id)
    }

    // Check filters (don't wait)
    checkFilters(message).catch((e) => console.log("[v0] Filter check error:", e))

    // Wait for AFK checks to complete
    await afkPromise

    // Parse command
    const parsed = parseCommand(text)
    if (!parsed) {
      return
    }

    const handler = commands[parsed.command]
    if (!handler) {
      if (message.chat.type !== "private" && (await shouldCleanCommand(message.chat.id, "other"))) {
        bot.deleteMessage(message.chat.id, message.message_id).catch(() => {})
      }
      return
    }

    // Check if command is disabled
    if (message.chat.type !== "private") {
      if (await isCommandDisabled(message.chat.id, parsed.command)) {
        return
      }
    }

    const { isAdmin, isOwner } =
      message.chat.type === "private"
        ? { isAdmin: true, isOwner: true }
        : await getCachedAdminStatus(message.chat.id, message.from.id)

    // Build context
    const ctx: CommandContext = {
      message,
      chat: message.chat,
      user: message.from,
      args: parsed.args,
      replyToMessage: message.reply_to_message,
      isAdmin,
      isOwner,
      isSudoer: await isSudoer(message.from.id),
      bot: bot,
    }

    // Execute handler
    await handler(ctx)

    // Clean commands if enabled (don't wait)
    if (message.chat.type !== "private") {
      const isAdminCmd = adminCommands.has(parsed.command)
      const isUserCmd = userCommands.has(parsed.command)

      if (isAdminCmd && (await shouldCleanCommand(message.chat.id, "admin"))) {
        bot.deleteMessage(message.chat.id, message.message_id).catch(() => {})
      } else if (isUserCmd && (await shouldCleanCommand(message.chat.id, "user"))) {
        bot.deleteMessage(message.chat.id, message.message_id).catch(() => {})
      }
    }
  } catch (error) {
    console.error("[v0] Error handling update:", error)
  }
}
