import type { CommandContext, TelegramChat, TelegramUser, TelegramMessage } from "../types"
import { supabase, formatText, parseButtons, escapeHtml } from "../utils"
import { bot as defaultBot } from "../bot"

// ===========================================
// HELPER: Check if user can change group info
// ===========================================
async function canChangeInfo(chatId: number, userId: number): Promise<boolean> {
  try {
    const member = await defaultBot.getChatMember(chatId, userId)
    if (member.status === "creator") return true
    if (member.status === "administrator" && member.can_change_info) return true
    return false
  } catch {
    return false
  }
}

// ===========================================
// HELPER: Get media info from message
// ===========================================
function getMediaInfo(message: TelegramMessage): { type: string; fileId: string } | null {
  if (message.photo && message.photo.length > 0) {
    return { type: "photo", fileId: message.photo[message.photo.length - 1].file_id }
  }
  if (message.video) {
    return { type: "video", fileId: message.video.file_id }
  }
  if (message.document) {
    return { type: "document", fileId: message.document.file_id }
  }
  if (message.audio) {
    return { type: "audio", fileId: message.audio.file_id }
  }
  if (message.voice) {
    return { type: "voice", fileId: message.voice.file_id }
  }
  if (message.video_note) {
    return { type: "video_note", fileId: message.video_note.file_id }
  }
  if (message.sticker) {
    return { type: "sticker", fileId: message.sticker.file_id }
  }
  if (message.animation) {
    return { type: "animation", fileId: message.animation.file_id }
  }
  return null
}

// ===========================================
// NOTES COMMANDS
// ===========================================

export async function handleSave(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to save notes.")
    return
  }

  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a note name.")
    return
  }

  const noteName = ctx.args[0].toLowerCase().replace("#", "")
  let noteContent = ctx.args.slice(1).join(" ")
  let noteType = "text"
  let fileId: string | undefined

  // Check for reply with media
  if (ctx.replyToMessage) {
    const media = getMediaInfo(ctx.replyToMessage)
    if (media) {
      noteType = media.type
      fileId = media.fileId
      noteContent = noteContent || ctx.replyToMessage.caption || ""
    } else if (ctx.replyToMessage.text && !noteContent) {
      noteContent = ctx.replyToMessage.text
    }
  }

  if (!noteContent && !fileId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide note content or reply to a message.")
    return
  }

  const { text: cleanText, buttons } = parseButtons(noteContent)

  await supabase.from("notes").upsert(
    {
      chat_id: ctx.chat.id,
      note_name: noteName,
      note_content: cleanText,
      note_type: noteType,
      file_id: fileId,
      buttons: buttons.length > 0 ? buttons : null,
      created_by: ctx.user.id,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id,note_name" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Note "${noteName}" has been saved!`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleGet(ctx: CommandContext): Promise<void> {
  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a note name.")
    return
  }

  const noteName = ctx.args[0].toLowerCase().replace("#", "")
  await sendNote(ctx.chat.id, noteName, ctx.user, ctx.chat, ctx.message.message_id)
}

export async function handleNotes(ctx: CommandContext): Promise<void> {
  const { data: notes } = await supabase.from("notes").select("note_name").eq("chat_id", ctx.chat.id)

  if (!notes || notes.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No notes saved in this chat.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const noteList = notes.map((n) => `- #${n.note_name}`).join("\n")
  await ctx.bot.sendMessage(ctx.chat.id, `Notes in this chat:\n${noteList}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleClear(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to clear notes.")
    return
  }

  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a note name to clear.")
    return
  }

  const noteName = ctx.args[0].toLowerCase().replace("#", "")

  const { error } = await supabase.from("notes").delete().eq("chat_id", ctx.chat.id).eq("note_name", noteName)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to clear note.")
    return
  }

  await ctx.bot.sendMessage(ctx.chat.id, `Note "${noteName}" has been cleared!`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// Send note - uses defaultBot for background
export async function sendNote(
  chatId: number,
  noteName: string,
  user: TelegramUser,
  chat: TelegramChat,
  replyToMessageId?: number,
): Promise<void> {
  const { data: note } = await supabase
    .from("notes")
    .select("*")
    .eq("chat_id", chatId)
    .eq("note_name", noteName.toLowerCase())
    .single()

  if (!note) return

  const formattedText = formatText(note.note_content || "", user, chat)

  const options: Record<string, unknown> = {
    parse_mode: note.parse_mode || "HTML",
    reply_to_message_id: replyToMessageId,
  }

  if (note.buttons && Array.isArray(note.buttons) && note.buttons.length > 0) {
    options.reply_markup = { inline_keyboard: note.buttons }
  }

  try {
    switch (note.note_type) {
      case "photo":
        await defaultBot.sendPhoto(chatId, note.file_id, { caption: formattedText, ...options })
        break
      case "video":
        await defaultBot.sendVideo(chatId, note.file_id, { caption: formattedText, ...options })
        break
      case "document":
        await defaultBot.sendDocument(chatId, note.file_id, { caption: formattedText, ...options })
        break
      case "audio":
        await defaultBot.sendAudio(chatId, note.file_id, { caption: formattedText, ...options })
        break
      case "voice":
        await defaultBot.sendVoice(chatId, note.file_id, options)
        break
      case "video_note":
        await defaultBot.sendVideoNote(chatId, note.file_id, options)
        break
      case "sticker":
        await defaultBot.sendSticker(chatId, note.file_id)
        break
      case "animation":
        await defaultBot.sendAnimation(chatId, note.file_id, { caption: formattedText, ...options })
        break
      default:
        await defaultBot.sendMessage(chatId, formattedText, options)
    }
  } catch (error) {
    console.error("[v0] Error sending note:", error)
  }
}

// ===========================================
// FILTER COMMANDS
// ===========================================

export async function handleFilter(ctx: CommandContext): Promise<void> {
  // Check if user is owner or has can_change_info permission
  const hasPermission = await canChangeInfo(ctx.chat.id, ctx.user.id)
  if (!hasPermission) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You need to be the group owner or have 'Change Group Info' permission to add filters.",
    )
    return
  }

  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Usage: /filter <keyword> <reply text>\nOr reply to a message with /filter <keyword>",
    )
    return
  }

  const keyword = ctx.args[0].toLowerCase()
  let replyText = ctx.args.slice(1).join(" ")
  let replyType = "text"
  let fileId: string | undefined

  // Check for reply with media - supports ALL media types
  if (ctx.replyToMessage) {
    const media = getMediaInfo(ctx.replyToMessage)
    if (media) {
      replyType = media.type
      fileId = media.fileId
      replyText = replyText || ctx.replyToMessage.caption || ""
    } else if (ctx.replyToMessage.text && !replyText) {
      replyText = ctx.replyToMessage.text
    }
  }

  // For media-only filters (like stickers), we don't need text
  if (!replyText && !fileId) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Please provide filter reply content or reply to a message (text, photo, video, sticker, audio, document, voice, video note, or animation).",
    )
    return
  }

  const { text: cleanText, buttons } = parseButtons(replyText)

  await supabase.from("filters").upsert(
    {
      chat_id: ctx.chat.id,
      keyword,
      reply_text: cleanText || null,
      reply_type: replyType,
      file_id: fileId || null,
      buttons: buttons.length > 0 ? buttons : null,
      created_by: ctx.user.id,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id,keyword" },
  )

  const mediaLabel = replyType !== "text" ? ` (${replyType})` : ""
  await ctx.bot.sendMessage(ctx.chat.id, `Filter "${keyword}" has been added!${mediaLabel}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleStop(ctx: CommandContext): Promise<void> {
  // Check if user is owner or has can_change_info permission
  const hasPermission = await canChangeInfo(ctx.chat.id, ctx.user.id)
  if (!hasPermission) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You need to be the group owner or have 'Change Group Info' permission to remove filters.",
    )
    return
  }

  if (ctx.args.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a filter keyword to remove.")
    return
  }

  const keyword = ctx.args[0].toLowerCase()

  const { data: existing } = await supabase
    .from("filters")
    .select("id")
    .eq("chat_id", ctx.chat.id)
    .eq("keyword", keyword)
    .maybeSingle()

  if (!existing) {
    await ctx.bot.sendMessage(ctx.chat.id, `No filter found for "${keyword}".`, {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await supabase.from("filters").delete().eq("chat_id", ctx.chat.id).eq("keyword", keyword)

  await ctx.bot.sendMessage(ctx.chat.id, `Filter "${keyword}" has been removed!`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleFilters(ctx: CommandContext): Promise<void> {
  const { data: filters } = await supabase.from("filters").select("keyword, reply_type").eq("chat_id", ctx.chat.id)

  if (!filters || filters.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No filters in this chat.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const filterList = filters
    .map((f) => {
      const typeLabel = f.reply_type !== "text" ? ` [${f.reply_type}]` : ""
      return `- ${f.keyword}${typeLabel}`
    })
    .join("\n")

  await ctx.bot.sendMessage(ctx.chat.id, `Filters in this chat:\n${filterList}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// Filter cache for performance
const filterCache = new Map<
  number,
  {
    filters: Array<{
      keyword: string
      reply_text: string | null
      reply_type: string
      file_id: string | null
      buttons: unknown
      parse_mode: string | null
    }>
    timestamp: number
  }
>()
const FILTER_CACHE_TTL = 60000 // 60 seconds

// Check and trigger filters - uses defaultBot, optimized with caching
export async function checkFilters(message: { chat: { id: number }; text?: string; caption?: string }): Promise<void> {
  const text = (message.text || message.caption || "").toLowerCase()
  if (!text) return

  const chatId = message.chat.id
  let filters: Array<{
    keyword: string
    reply_text: string | null
    reply_type: string
    file_id: string | null
    buttons: unknown
    parse_mode: string | null
  }>

  // Check cache first
  const cached = filterCache.get(chatId)
  if (cached && Date.now() - cached.timestamp < FILTER_CACHE_TTL) {
    filters = cached.filters
  } else {
    const { data } = await supabase
      .from("filters")
      .select("keyword, reply_text, reply_type, file_id, buttons, parse_mode")
      .eq("chat_id", chatId)
    filters = data || []
    filterCache.set(chatId, { filters, timestamp: Date.now() })
  }

  if (filters.length === 0) return

  for (const filter of filters) {
    // Check if keyword exists in message (word boundary check for better matching)
    const keywordLower = filter.keyword.toLowerCase()
    const regex = new RegExp(
      `(^|\\s|[^a-z0-9])${keywordLower.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}($|\\s|[^a-z0-9])`,
      "i",
    )

    if (regex.test(text) || text === keywordLower) {
      const options: Record<string, unknown> = {
        parse_mode: filter.parse_mode || "HTML",
      }

      if (filter.buttons && Array.isArray(filter.buttons) && filter.buttons.length > 0) {
        options.reply_markup = { inline_keyboard: filter.buttons }
      }

      try {
        switch (filter.reply_type) {
          case "photo":
            if (filter.file_id) {
              await defaultBot.sendPhoto(chatId, filter.file_id, {
                caption: filter.reply_text || undefined,
                ...options,
              })
            }
            break
          case "video":
            if (filter.file_id) {
              await defaultBot.sendVideo(chatId, filter.file_id, {
                caption: filter.reply_text || undefined,
                ...options,
              })
            }
            break
          case "document":
            if (filter.file_id) {
              await defaultBot.sendDocument(chatId, filter.file_id, {
                caption: filter.reply_text || undefined,
                ...options,
              })
            }
            break
          case "audio":
            if (filter.file_id) {
              await defaultBot.sendAudio(chatId, filter.file_id, {
                caption: filter.reply_text || undefined,
                ...options,
              })
            }
            break
          case "voice":
            if (filter.file_id) {
              await defaultBot.sendVoice(chatId, filter.file_id, options)
            }
            break
          case "video_note":
            if (filter.file_id) {
              await defaultBot.sendVideoNote(chatId, filter.file_id, options)
            }
            break
          case "sticker":
            if (filter.file_id) {
              await defaultBot.sendSticker(chatId, filter.file_id)
            }
            break
          case "animation":
            if (filter.file_id) {
              await defaultBot.sendAnimation(chatId, filter.file_id, {
                caption: filter.reply_text || undefined,
                ...options,
              })
            }
            break
          default:
            if (filter.reply_text) {
              await defaultBot.sendMessage(chatId, filter.reply_text, options)
            }
        }
      } catch (error) {
        console.error("[v0] Error sending filter reply:", error)
      }
      break // Only trigger first matching filter
    }
  }
}

// Invalidate filter cache when filters change
export function invalidateFilterCache(chatId: number): void {
  filterCache.delete(chatId)
}

// ===========================================
// WELCOME/GOODBYE COMMANDS
// ===========================================

export async function handleWelcome(ctx: CommandContext): Promise<void> {
  if (ctx.args[0]?.toLowerCase() === "on") {
    if (!ctx.isAdmin) {
      await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin.")
      return
    }
    await supabase.from("greetings").upsert(
      {
        chat_id: ctx.chat.id,
        welcome_enabled: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Welcome messages enabled!", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (ctx.args[0]?.toLowerCase() === "off") {
    if (!ctx.isAdmin) {
      await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin.")
      return
    }
    await supabase.from("greetings").upsert(
      {
        chat_id: ctx.chat.id,
        welcome_enabled: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Welcome messages disabled!", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  // Show current welcome message
  const { data: greetings } = await supabase.from("greetings").select("*").eq("chat_id", ctx.chat.id).single()

  if (!greetings || !greetings.welcome_text) {
    await ctx.bot.sendMessage(ctx.chat.id, "No custom welcome message set. Using default.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `Current welcome message:\n\n${greetings.welcome_text}\n\nEnabled: ${greetings.welcome_enabled ? "Yes" : "No"}`,
    { reply_to_message_id: ctx.message.message_id },
  )
}

export async function handleSetWelcome(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set welcome message.")
    return
  }

  let welcomeText = ctx.args.join(" ")
  let welcomeType = "text"
  let fileId: string | undefined

  if (ctx.replyToMessage) {
    const media = getMediaInfo(ctx.replyToMessage)
    if (media) {
      welcomeType = media.type
      fileId = media.fileId
      welcomeText = welcomeText || ctx.replyToMessage.caption || ""
    } else if (ctx.replyToMessage.text && !welcomeText) {
      welcomeText = ctx.replyToMessage.text
    }
  }

  if (!welcomeText && !fileId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a welcome message or reply to a message.")
    return
  }

  const { text: cleanText, buttons } = parseButtons(welcomeText)

  await supabase.from("greetings").upsert(
    {
      chat_id: ctx.chat.id,
      welcome_text: cleanText,
      welcome_type: welcomeType,
      welcome_file_id: fileId,
      welcome_buttons: buttons.length > 0 ? buttons : null,
      welcome_enabled: true,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, "Welcome message has been set!", {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleGoodbye(ctx: CommandContext): Promise<void> {
  if (ctx.args[0]?.toLowerCase() === "on") {
    if (!ctx.isAdmin) {
      await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin.")
      return
    }
    await supabase.from("greetings").upsert(
      {
        chat_id: ctx.chat.id,
        goodbye_enabled: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Goodbye messages enabled!", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (ctx.args[0]?.toLowerCase() === "off") {
    if (!ctx.isAdmin) {
      await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin.")
      return
    }
    await supabase.from("greetings").upsert(
      {
        chat_id: ctx.chat.id,
        goodbye_enabled: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Goodbye messages disabled!", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  // Show current goodbye message
  const { data: greetings } = await supabase.from("greetings").select("*").eq("chat_id", ctx.chat.id).single()

  if (!greetings || !greetings.goodbye_text) {
    await ctx.bot.sendMessage(ctx.chat.id, "No custom goodbye message set.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `Current goodbye message:\n\n${greetings.goodbye_text}\n\nEnabled: ${greetings.goodbye_enabled ? "Yes" : "No"}`,
    { reply_to_message_id: ctx.message.message_id },
  )
}

export async function handleSetGoodbye(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set goodbye message.")
    return
  }

  let goodbyeText = ctx.args.join(" ")
  let goodbyeType = "text"
  let fileId: string | undefined

  if (ctx.replyToMessage) {
    const media = getMediaInfo(ctx.replyToMessage)
    if (media) {
      goodbyeType = media.type
      fileId = media.fileId
      goodbyeText = goodbyeText || ctx.replyToMessage.caption || ""
    } else if (ctx.replyToMessage.text && !goodbyeText) {
      goodbyeText = ctx.replyToMessage.text
    }
  }

  if (!goodbyeText && !fileId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a goodbye message or reply to a message.")
    return
  }

  const { text: cleanText, buttons } = parseButtons(goodbyeText)

  await supabase.from("greetings").upsert(
    {
      chat_id: ctx.chat.id,
      goodbye_text: cleanText,
      goodbye_type: goodbyeType,
      goodbye_file_id: fileId,
      goodbye_buttons: buttons.length > 0 ? buttons : null,
      goodbye_enabled: true,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, "Goodbye message has been set!", {
    reply_to_message_id: ctx.message.message_id,
  })
}

// Send welcome message - uses defaultBot
export async function sendWelcome(chatId: number, user: TelegramUser, chat: TelegramChat): Promise<void> {
  const { data: greetings } = await supabase.from("greetings").select("*").eq("chat_id", chatId).maybeSingle()

  if (greetings?.welcome_enabled === false) return

  const DEFAULT_WELCOME_MESSAGE = `Hey {mention}! Welcome to {chatname}!

I'm Sukoon, and I'm here to help manage this group. Feel free to check out /help to see what I can do!

Enjoy your stay!`
  const welcomeText = greetings?.welcome_text || DEFAULT_WELCOME_MESSAGE

  const formattedText = formatText(welcomeText, user, chat)

  const options: Record<string, unknown> = {
    parse_mode: greetings?.welcome_parse_mode || "HTML",
  }

  if (greetings?.welcome_buttons && Array.isArray(greetings.welcome_buttons) && greetings.welcome_buttons.length > 0) {
    options.reply_markup = { inline_keyboard: greetings.welcome_buttons }
  }

  try {
    if (greetings?.welcome_type && greetings.welcome_type !== "text" && greetings.welcome_file_id) {
      switch (greetings.welcome_type) {
        case "photo":
          await defaultBot.sendPhoto(chatId, greetings.welcome_file_id, { caption: formattedText, ...options })
          break
        case "video":
          await defaultBot.sendVideo(chatId, greetings.welcome_file_id, { caption: formattedText, ...options })
          break
        case "sticker":
          await defaultBot.sendSticker(chatId, greetings.welcome_file_id)
          break
        case "animation":
          await defaultBot.sendAnimation(chatId, greetings.welcome_file_id, { caption: formattedText, ...options })
          break
        case "video_note":
          await defaultBot.sendVideoNote(chatId, greetings.welcome_file_id, options)
          break
        default:
          await defaultBot.sendMessage(chatId, formattedText, options)
      }
    } else {
      await defaultBot.sendMessage(chatId, formattedText, options)
    }
  } catch (error) {
    console.error("[v0] Error sending welcome:", error)
  }
}

// Send goodbye message - uses defaultBot
export async function sendGoodbye(chatId: number, user: TelegramUser, chat: TelegramChat): Promise<void> {
  const { data: greetings } = await supabase.from("greetings").select("*").eq("chat_id", chatId).single()

  if (!greetings?.goodbye_enabled) return

  const goodbyeText = greetings.goodbye_text || `Goodbye ${escapeHtml(user.first_name)}! We'll miss you.`

  const formattedText = formatText(goodbyeText, user, chat)

  const options: Record<string, unknown> = {
    parse_mode: greetings.goodbye_parse_mode || "HTML",
  }

  if (greetings.goodbye_buttons && Array.isArray(greetings.goodbye_buttons) && greetings.goodbye_buttons.length > 0) {
    options.reply_markup = { inline_keyboard: greetings.goodbye_buttons }
  }

  try {
    switch (greetings.goodbye_type) {
      case "photo":
        await defaultBot.sendPhoto(chatId, greetings.goodbye_file_id, { caption: formattedText, ...options })
        break
      case "video":
        await defaultBot.sendVideo(chatId, greetings.goodbye_file_id, { caption: formattedText, ...options })
        break
      case "sticker":
        await defaultBot.sendSticker(chatId, greetings.goodbye_file_id)
        break
      case "video_note":
        await defaultBot.sendVideoNote(chatId, greetings.goodbye_file_id, options)
        break
      default:
        await defaultBot.sendMessage(chatId, formattedText, options)
    }
  } catch (error) {
    console.error("[v0] Error sending goodbye:", error)
  }
}

// ===========================================
// RULES COMMANDS
// ===========================================

export async function handleRules(ctx: CommandContext): Promise<void> {
  const { data: settings } = await supabase
    .from("chat_settings")
    .select("rules, private_rules")
    .eq("chat_id", ctx.chat.id)
    .single()

  if (!settings?.rules) {
    await ctx.bot.sendMessage(ctx.chat.id, "No rules have been set for this chat.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (settings.private_rules) {
    try {
      await defaultBot.sendMessage(
        ctx.user.id,
        `Rules for ${escapeHtml(ctx.chat.title || "this chat")}:\n\n${settings.rules}`,
        {
          parse_mode: "HTML",
        },
      )
      await ctx.bot.sendMessage(ctx.chat.id, "I've sent you the rules in PM!", {
        reply_to_message_id: ctx.message.message_id,
      })
    } catch {
      await ctx.bot.sendMessage(ctx.chat.id, "Please start a private chat with me first to receive the rules.", {
        reply_to_message_id: ctx.message.message_id,
      })
    }
  } else {
    await ctx.bot.sendMessage(ctx.chat.id, `Rules for this chat:\n\n${settings.rules}`, {
      reply_to_message_id: ctx.message.message_id,
      parse_mode: "HTML",
    })
  }
}

export async function handleSetRules(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set rules.")
    return
  }

  let rulesText = ctx.args.join(" ")

  if (ctx.replyToMessage?.text && !rulesText) {
    rulesText = ctx.replyToMessage.text
  }

  if (!rulesText) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide rules text.")
    return
  }

  await supabase.from("chat_settings").upsert(
    {
      chat_id: ctx.chat.id,
      rules: rulesText,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, "Rules have been set!", {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleClearRules(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to clear rules.")
    return
  }

  await supabase.from("chat_settings").upsert(
    {
      chat_id: ctx.chat.id,
      rules: null,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, "Rules have been cleared!", {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handlePrivateRules(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin.")
    return
  }

  const mode = ctx.args[0]?.toLowerCase()

  if (mode === "on" || mode === "yes") {
    await supabase.from("chat_settings").upsert(
      {
        chat_id: ctx.chat.id,
        private_rules: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Rules will now be sent in PM.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else if (mode === "off" || mode === "no") {
    await supabase.from("chat_settings").upsert(
      {
        chat_id: ctx.chat.id,
        private_rules: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Rules will now be sent in the group.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else {
    const { data } = await supabase.from("chat_settings").select("private_rules").eq("chat_id", ctx.chat.id).single()
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Private rules: ${data?.private_rules ? "On" : "Off"}\nUsage: /privaterules on/off`,
      {
        reply_to_message_id: ctx.message.message_id,
      },
    )
  }
}

// ===========================================
// REPORT COMMANDS
// ===========================================

export async function handleReport(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command only works in groups.")
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to report it to admins.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const { data: settings } = await supabase
    .from("report_settings")
    .select("reports_enabled")
    .eq("chat_id", ctx.chat.id)
    .single()

  if (settings?.reports_enabled === false) {
    return
  }

  try {
    const admins = await defaultBot.getChatAdministrators(ctx.chat.id)
    const adminMentions = admins
      .filter((a: { user: TelegramUser }) => !a.user.is_bot)
      .map((a: { user: TelegramUser }) => `<a href="tg://user?id=${a.user.id}">${escapeHtml(a.user.first_name)}</a>`)
      .join(", ")

    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Reported to admins: ${adminMentions}\n\nReported by: ${escapeHtml(ctx.user.first_name)}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.replyToMessage.message_id,
      },
    )
  } catch (error) {
    console.error("[v0] Error handling report:", error)
  }
}

export async function handleReports(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin.")
    return
  }

  const mode = ctx.args[0]?.toLowerCase()

  if (mode === "on" || mode === "yes") {
    await supabase.from("report_settings").upsert(
      {
        chat_id: ctx.chat.id,
        reports_enabled: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Reports are now enabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else if (mode === "off" || mode === "no") {
    await supabase.from("report_settings").upsert(
      {
        chat_id: ctx.chat.id,
        reports_enabled: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await ctx.bot.sendMessage(ctx.chat.id, "Reports are now disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else {
    const { data } = await supabase
      .from("report_settings")
      .select("reports_enabled")
      .eq("chat_id", ctx.chat.id)
      .single()
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Reports: ${data?.reports_enabled !== false ? "On" : "Off"}\nUsage: /reports on/off`,
      {
        reply_to_message_id: ctx.message.message_id,
      },
    )
  }
}
