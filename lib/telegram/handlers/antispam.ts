import type { CommandContext, TelegramMessage } from "../types"
import { supabase, isApproved, isUserAdmin, parseTime } from "../utils"
import { bot as defaultBot } from "../bot"

// ===========================================
// IN-MEMORY FLOOD TRACKING (for speed)
// ===========================================

interface FloodTracker {
  count: number
  lastMessage: number
  messages: number[] // message IDs to delete
}

// chatId -> userId -> FloodTracker
const floodCache = new Map<number, Map<number, FloodTracker>>()

// chatId -> settings
const floodSettingsCache = new Map<
  number,
  {
    limit: number
    mode: string
    time: number
    timer: number
    deleteMessages: boolean
    lastFetch: number
  }
>()

const SETTINGS_CACHE_TTL = 60000 // 1 minute cache for settings

// Clean old entries periodically
setInterval(() => {
  const now = Date.now()
  for (const [chatId, users] of floodCache.entries()) {
    for (const [userId, tracker] of users.entries()) {
      if (now - tracker.lastMessage > 30000) {
        // 30 seconds
        users.delete(userId)
      }
    }
    if (users.size === 0) {
      floodCache.delete(chatId)
    }
  }
}, 30000)

// ===========================================
// LOCK TYPES
// ===========================================

const LOCK_TYPES = [
  "all",
  "media",
  "sticker",
  "gif",
  "url",
  "forward",
  "bot",
  "game",
  "voice",
  "video",
  "document",
  "photo",
  "audio",
  "command",
  "email",
  "location",
  "contact",
  "inline",
  "button",
  "emoji",
  "poll",
  "invitelink",
  "rtl",
  "arabic",
  "chinese",
  "anonchannel",
]

// ===========================================
// LOCK COMMANDS
// ===========================================

export async function handleLock(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to lock things.")
    return
  }

  const lockType = ctx.args[0]?.toLowerCase()

  if (!lockType) {
    await ctx.bot.sendMessage(ctx.chat.id, `Please specify what to lock.\nAvailable: ${LOCK_TYPES.join(", ")}`)
    return
  }

  if (!LOCK_TYPES.includes(lockType)) {
    await ctx.bot.sendMessage(ctx.chat.id, `Invalid lock type.\nAvailable: ${LOCK_TYPES.join(", ")}`)
    return
  }

  await supabase.from("locks").upsert(
    {
      chat_id: ctx.chat.id,
      lock_type: lockType,
    },
    { onConflict: "chat_id,lock_type" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Locked ${lockType} for this chat.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleUnlock(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to unlock things.")
    return
  }

  const lockType = ctx.args[0]?.toLowerCase()

  if (!lockType) {
    await ctx.bot.sendMessage(ctx.chat.id, `Please specify what to unlock.\nAvailable: ${LOCK_TYPES.join(", ")}`)
    return
  }

  if (lockType === "all") {
    await supabase.from("locks").delete().eq("chat_id", ctx.chat.id)
    await ctx.bot.sendMessage(ctx.chat.id, "All locks have been removed.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await supabase.from("locks").delete().eq("chat_id", ctx.chat.id).eq("lock_type", lockType)

  await ctx.bot.sendMessage(ctx.chat.id, `Unlocked ${lockType} for this chat.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleLocks(ctx: CommandContext): Promise<void> {
  const { data: locks } = await supabase.from("locks").select("lock_type").eq("chat_id", ctx.chat.id)

  if (!locks || locks.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No locks are currently active in this chat.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const lockList = locks.map((l) => l.lock_type).join(", ")
  await ctx.bot.sendMessage(ctx.chat.id, `Currently locked:\n${lockList}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// ===========================================
// BLOCKLIST COMMANDS
// ===========================================

export async function handleBlocklist(ctx: CommandContext): Promise<void> {
  const { data: blocklist } = await supabase
    .from("blocklists")
    .select("trigger_word, match_mode")
    .eq("chat_id", ctx.chat.id)

  if (!blocklist || blocklist.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No blocklist triggers in this chat.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  let text = "Current blocklist triggers:\n\n"
  blocklist.forEach((item) => {
    text += `- ${item.trigger_word} (${item.match_mode})\n`
  })

  await ctx.bot.sendMessage(ctx.chat.id, text, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleAddBlocklist(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to add blocklist triggers.")
    return
  }

  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a word/phrase to blocklist.")
    return
  }

  const trigger = ctx.args.join(" ").toLowerCase()

  await supabase.from("blocklists").upsert(
    {
      chat_id: ctx.chat.id,
      trigger_word: trigger,
      match_mode: "word",
    },
    { onConflict: "chat_id,trigger_word" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Added "${trigger}" to the blocklist.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleUnBlocklist(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to remove blocklist triggers.")
    return
  }

  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a word/phrase to remove.")
    return
  }

  const trigger = ctx.args.join(" ").toLowerCase()

  const { error } = await supabase.from("blocklists").delete().eq("chat_id", ctx.chat.id).eq("trigger_word", trigger)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to remove from blocklist.")
    return
  }

  await ctx.bot.sendMessage(ctx.chat.id, `Removed "${trigger}" from the blocklist.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleBlocklistMode(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set blocklist mode.")
    return
  }

  const mode = ctx.args[0]?.toLowerCase()
  const validModes = ["delete", "warn", "kick", "ban", "tban", "mute", "tmute"]

  if (!mode || !validModes.includes(mode)) {
    await ctx.bot.sendMessage(ctx.chat.id, `Please provide a valid mode: ${validModes.join(", ")}`)
    return
  }

  let blocklistTime = 0
  if (mode === "tban" || mode === "tmute") {
    const timeArg = ctx.args[1]
    if (timeArg) {
      blocklistTime = parseTime(timeArg) || 0
    }
  }

  await supabase.from("blocklist_settings").upsert(
    {
      chat_id: ctx.chat.id,
      blocklist_mode: mode,
      blocklist_time: blocklistTime,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Blocklist mode set to ${mode}${blocklistTime ? ` (${ctx.args[1]})` : ""}.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// ===========================================
// ANTIFLOOD COMMANDS
// ===========================================

export async function handleSetFlood(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set flood settings.")
    return
  }

  const limit = Number.parseInt(ctx.args[0])

  if (isNaN(limit)) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a number for the flood limit (0 to disable).")
    return
  }

  await supabase.from("antiflood_settings").upsert(
    {
      chat_id: ctx.chat.id,
      flood_limit: limit,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  floodSettingsCache.delete(ctx.chat.id)

  if (limit === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Antiflood has been disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else {
    await ctx.bot.sendMessage(ctx.chat.id, `Antiflood limit set to ${limit} messages.`, {
      reply_to_message_id: ctx.message.message_id,
    })
  }
}

export async function handleFlood(ctx: CommandContext): Promise<void> {
  const { data: settings } = await supabase.from("antiflood_settings").select("*").eq("chat_id", ctx.chat.id).single()

  if (!settings || settings.flood_limit === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Antiflood is currently disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `Antiflood settings:\n` +
      `- Limit: ${settings.flood_limit} messages\n` +
      `- Mode: ${settings.flood_mode || "mute"}\n` +
      `- Timer: ${settings.flood_timer || 0}s\n` +
      `- Delete messages: ${settings.delete_flood_messages ? "Yes" : "No"}`,
    { reply_to_message_id: ctx.message.message_id },
  )
}

export async function handleSetFloodMode(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set flood mode.")
    return
  }

  const mode = ctx.args[0]?.toLowerCase()
  const validModes = ["ban", "kick", "mute", "tban", "tmute"]

  if (!mode || !validModes.includes(mode)) {
    await ctx.bot.sendMessage(ctx.chat.id, `Please provide a valid mode: ${validModes.join(", ")}`)
    return
  }

  let floodTime = 0
  if (mode === "tban" || mode === "tmute") {
    const timeArg = ctx.args[1]
    if (timeArg) {
      floodTime = parseTime(timeArg) || 0
    }
  }

  await supabase.from("antiflood_settings").upsert(
    {
      chat_id: ctx.chat.id,
      flood_mode: mode,
      flood_time: floodTime,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  floodSettingsCache.delete(ctx.chat.id)

  await ctx.bot.sendMessage(ctx.chat.id, `Flood mode set to ${mode}${floodTime ? ` (${ctx.args[1]})` : ""}.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleClearFlood(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to change flood settings.")
    return
  }

  const setting = ctx.args[0]?.toLowerCase()

  if (setting === "on" || setting === "yes") {
    await supabase.from("antiflood_settings").upsert(
      {
        chat_id: ctx.chat.id,
        delete_flood_messages: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    floodSettingsCache.delete(ctx.chat.id)
    await ctx.bot.sendMessage(ctx.chat.id, "Flood messages will now be deleted.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else if (setting === "off" || setting === "no") {
    await supabase.from("antiflood_settings").upsert(
      {
        chat_id: ctx.chat.id,
        delete_flood_messages: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    floodSettingsCache.delete(ctx.chat.id)
    await ctx.bot.sendMessage(ctx.chat.id, "Flood messages will no longer be deleted.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else {
    const { data } = await supabase
      .from("antiflood_settings")
      .select("delete_flood_messages")
      .eq("chat_id", ctx.chat.id)
      .maybeSingle()
    const status = data?.delete_flood_messages ? "enabled" : "disabled"
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Flood message deletion is currently ${status}.\nUsage: /clearflood on|off`,
      { reply_to_message_id: ctx.message.message_id },
    )
  }
}

// ===========================================
// SPAM CHECKING FUNCTIONS (OPTIMIZED)
// ===========================================

export async function checkLocks(message: TelegramMessage): Promise<boolean> {
  if (!message.from) return false

  const chatId = message.chat.id
  const userId = message.from.id

  // Skip for admins and approved users (parallel check)
  const [isAdmin, approved] = await Promise.all([isUserAdmin(chatId, userId), isApproved(chatId, userId)])

  if (isAdmin || approved) return false

  const { data: locks } = await supabase.from("locks").select("lock_type").eq("chat_id", chatId)

  if (!locks || locks.length === 0) return false

  const lockTypes = new Set(locks.map((l) => l.lock_type))

  // Check various lock types
  if (lockTypes.has("all")) return true
  if (
    lockTypes.has("media") &&
    (message.photo ||
      message.video ||
      message.document ||
      message.audio ||
      message.voice ||
      message.sticker ||
      message.animation)
  )
    return true
  if (lockTypes.has("sticker") && message.sticker) return true
  if (lockTypes.has("gif") && message.animation) return true
  if (lockTypes.has("photo") && message.photo) return true
  if (lockTypes.has("video") && message.video) return true
  if (lockTypes.has("document") && message.document) return true
  if (lockTypes.has("audio") && message.audio) return true
  if (lockTypes.has("voice") && message.voice) return true
  if (lockTypes.has("poll") && message.poll) return true
  if (lockTypes.has("contact") && message.contact) return true
  if (lockTypes.has("location") && message.location) return true
  if (lockTypes.has("forward") && (message.forward_from || message.forward_from_chat)) return true
  if (lockTypes.has("inline") && message.via_bot) return true
  if (lockTypes.has("anonchannel") && message.sender_chat) return true

  const text = message.text || message.caption || ""

  if (lockTypes.has("url") && /https?:\/\/[^\s]+/.test(text)) return true
  if (lockTypes.has("invitelink") && /t\.me\/[^\s]+|telegram\.me\/[^\s]+/.test(text)) return true
  if (lockTypes.has("email") && /[^\s]+@[^\s]+\.[^\s]+/.test(text)) return true
  if (lockTypes.has("command") && text.startsWith("/")) return true
  if (lockTypes.has("rtl") && /[\u0591-\u07FF\uFB1D-\uFDFD\uFE70-\uFEFC]/.test(text)) return true
  if (lockTypes.has("arabic") && /[\u0600-\u06FF]/.test(text)) return true
  if (lockTypes.has("chinese") && /[\u4E00-\u9FFF]/.test(text)) return true

  return false
}

export async function checkBlocklist(
  message: TelegramMessage,
): Promise<{ matched: boolean; action?: string; time?: number }> {
  if (!message.chat || message.chat.type === "private") return { matched: false }
  if (!message.from) return { matched: false }

  const chatId = message.chat.id
  const userId = message.from.id
  const text = (message.text || message.caption || "").toLowerCase()

  if (!text) return { matched: false }

  const { data: blocklist } = await supabase.from("blocklists").select("trigger_word, match_mode").eq("chat_id", chatId)

  if (!blocklist || blocklist.length === 0) return { matched: false }

  console.log("[v0] Checking blocklist for chat:", chatId, "text:", text.substring(0, 50))

  for (const item of blocklist) {
    let matched = false

    if (item.match_mode === "word") {
      const regex = new RegExp(`\\b${item.trigger_word}\\b`, "i")
      matched = regex.test(text)
    } else if (item.match_mode === "contains") {
      matched = text.includes(item.trigger_word)
    } else if (item.match_mode === "regex") {
      try {
        matched = new RegExp(item.trigger_word, "i").test(text)
      } catch {
        // Invalid regex
      }
    }

    if (matched) {
      console.log("[v0] Blocklist matched:", item.trigger_word)
      const { data: settings } = await supabase
        .from("blocklist_settings")
        .select("blocklist_mode, blocklist_time")
        .eq("chat_id", chatId)
        .single()

      return {
        matched: true,
        action: settings?.blocklist_mode || "delete",
        time: settings?.blocklist_time || 0,
      }
    }
  }

  return { matched: false }
}

async function getFloodSettings(chatId: number) {
  const cached = floodSettingsCache.get(chatId)
  const now = Date.now()

  if (cached && now - cached.lastFetch < SETTINGS_CACHE_TTL) {
    return cached
  }

  const { data: settings } = await supabase.from("antiflood_settings").select("*").eq("chat_id", chatId).maybeSingle()

  if (!settings || settings.flood_limit === 0) {
    floodSettingsCache.set(chatId, {
      limit: 0,
      mode: "mute",
      time: 0,
      timer: 0,
      deleteMessages: false,
      lastFetch: now,
    })
    return null
  }

  const result = {
    limit: settings.flood_limit,
    mode: settings.flood_mode || "mute",
    time: settings.flood_time || 0,
    timer: settings.flood_timer || 0,
    deleteMessages: settings.delete_flood_messages || false,
    lastFetch: now,
  }

  floodSettingsCache.set(chatId, result)
  return result
}

export async function checkFlood(
  message: TelegramMessage,
): Promise<{ flooded: boolean; action?: string; time?: number; messageIds?: number[] }> {
  if (!message.from) return { flooded: false }

  const chatId = message.chat.id
  const userId = message.from.id

  // Skip for admins and approved users (parallel check for speed)
  const [isAdmin, approved] = await Promise.all([isUserAdmin(chatId, userId), isApproved(chatId, userId)])

  if (isAdmin || approved) return { flooded: false }

  // Get settings (cached)
  const settings = await getFloodSettings(chatId)
  if (!settings || settings.limit === 0) return { flooded: false }

  const now = Date.now()

  // Get or create chat's flood tracking
  if (!floodCache.has(chatId)) {
    floodCache.set(chatId, new Map())
  }
  const chatFlood = floodCache.get(chatId)!

  // Get or create user's flood tracking
  let tracker = chatFlood.get(userId)

  if (!tracker) {
    tracker = { count: 1, lastMessage: now, messages: [message.message_id] }
    chatFlood.set(userId, tracker)
    return { flooded: false }
  }

  // Check if timer exceeded (reset if too much time passed)
  const timeDiff = (now - tracker.lastMessage) / 1000
  if (settings.timer > 0 && timeDiff > settings.timer) {
    tracker.count = 1
    tracker.lastMessage = now
    tracker.messages = [message.message_id]
    return { flooded: false }
  }

  // Increment count
  tracker.count++
  tracker.lastMessage = now
  tracker.messages.push(message.message_id)

  // Keep only recent messages
  if (tracker.messages.length > settings.limit + 5) {
    tracker.messages = tracker.messages.slice(-settings.limit - 5)
  }

  // Check if flooded
  if (tracker.count >= settings.limit) {
    const messageIds = settings.deleteMessages ? [...tracker.messages] : []

    // Reset tracker
    chatFlood.delete(userId)

    return {
      flooded: true,
      action: settings.mode,
      time: settings.time,
      messageIds,
    }
  }

  return { flooded: false }
}

// Execute antiflood/blocklist action (uses defaultBot for background operations)
export async function executeAction(
  chatId: number,
  userId: number,
  action: string,
  time = 0,
  reason = "",
): Promise<void> {
  try {
    switch (action) {
      case "ban":
        await defaultBot.banChatMember(chatId, userId)
        break
      case "kick":
        await defaultBot.banChatMember(chatId, userId)
        await defaultBot.unbanChatMember(chatId, userId)
        break
      case "mute":
        await defaultBot.restrictChatMember(chatId, userId, { can_send_messages: false })
        break
      case "tban":
        await defaultBot.banChatMember(chatId, userId, {
          until_date: Math.floor(Date.now() / 1000) + (time || 3600),
        })
        break
      case "tmute":
        await defaultBot.restrictChatMember(
          chatId,
          userId,
          { can_send_messages: false },
          { until_date: Math.floor(Date.now() / 1000) + (time || 3600) },
        )
        break
      case "warn":
        await supabase.from("warnings").insert({
          chat_id: chatId,
          user_id: userId,
          reason,
        })
        break
    }
  } catch (error) {
    console.error("[v0] Failed to execute action:", error)
  }
}

export async function deleteFloodMessages(chatId: number, messageIds: number[]): Promise<void> {
  if (!messageIds.length) return

  // Delete messages in parallel (faster)
  await Promise.allSettled(messageIds.map((msgId) => defaultBot.deleteMessage(chatId, msgId).catch(() => {})))
}
