import type { CommandContext, TelegramMessage } from "../types"
import { supabase, isUserAdmin, isUserOwner } from "../utils"
import { bot as defaultBot, type TelegramBot } from "../bot"

// Link patterns to detect in bio - comprehensive list
const LINK_PATTERNS = [
  /https?:\/\/\S+/i,
  /t\.me\/\S+/i,
  /telegram\.(me|dog)\/\S+/i,
  /@[a-zA-Z][a-zA-Z0-9_]{3,}/i,
  /(bit\.ly|goo\.gl|tinyurl\.com|is\.gd|v\.gd|shorturl\.at)\/\S+/i,
  /wa\.me\/\S+/i,
  /(discord\.(gg|com)|discordapp\.com)\/\S+/i,
  /(instagram\.com|instagr\.am)\/\S+/i,
  /(youtube\.com|youtu\.be)\/\S+/i,
  /(twitter\.com|x\.com)\/\S+/i,
  /(facebook\.com|fb\.(me|com))\/\S+/i,
  /linkedin\.com\/\S+/i,
  /(tiktok\.com|vm\.tiktok\.com)\/\S+/i,
  /whatsapp\.com\/\S+/i,
  /snapchat\.com\/\S+/i,
]

// Check if bio contains links
function bioHasLinks(bio: string): boolean {
  if (!bio || bio.trim() === "") return false
  for (const pattern of LINK_PATTERNS) {
    if (pattern.test(bio)) return true
  }
  return false
}

// Main function to check user's bio - FULLY OPTIMIZED
export async function checkBioLinks(
  message: TelegramMessage,
  bot: TelegramBot = defaultBot,
): Promise<{ violated: boolean }> {
  // Skip private chats immediately
  if (!message.from || message.chat.type === "private") {
    return { violated: false }
  }

  const chatId = message.chat.id
  const userId = message.from.id

  const [settingsResult, freeResult, adminResult, ownerResult, bioResult] = await Promise.all([
    // Check if antibio is enabled
    supabase
      .from("antibio_settings")
      .select("enabled")
      .eq("chat_id", String(chatId))
      .maybeSingle(),
    // Check if user is free
    supabase
      .from("antibio_free_users")
      .select("user_id")
      .eq("chat_id", String(chatId))
      .eq("user_id", String(userId))
      .maybeSingle(),
    // Check if admin
    isUserAdmin(chatId, userId),
    // Check if owner
    isUserOwner(chatId, userId),
    // Get bio from Telegram API (this runs in parallel with DB checks!)
    bot
      .customRequest("getChat", { chat_id: userId })
      .catch(() => null),
  ])

  // If antibio is not enabled, return immediately
  const enabled = settingsResult.data?.enabled ?? false
  if (!enabled) {
    return { violated: false }
  }

  // If user is admin, owner, or free - allow
  if (adminResult || ownerResult || freeResult.data) {
    return { violated: false }
  }

  // Check the bio for links
  const bio = (bioResult as { bio?: string } | null)?.bio
  if (!bio || !bioHasLinks(bio)) {
    return { violated: false }
  }

  // Bio has links - delete message and warn (run in parallel)
  const userName = message.from.first_name
  await Promise.all([
    bot.deleteMessage(chatId, message.message_id).catch(() => {}),
    bot.sendMessage(
      chatId,
      `⚠️ <b>Warning!</b>\n\n${userName}, your bio contains a link which is not allowed in this group.\n\n📝 Please remove the link from your bio, then wait <b>1 minute</b> before sending messages again.\n\n<i>(Telegram takes up to 1 minute to update your bio information)</i>`,
      { parse_mode: "HTML" },
    ),
  ])

  return { violated: true }
}

// /antibio [on/off] - Toggle antibio for the chat
export async function handleAntibio(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  if (!ctx.isAdmin && !ctx.isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.")
    return
  }

  const arg = ctx.args[0]?.toLowerCase()

  if (!arg) {
    const { data } = await supabase.from("antibio_settings").select("enabled").eq("chat_id", ctx.chat.id).single()
    const enabled = data?.enabled ?? false
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `🔍 <b>Anti-Bio Link Status</b>\n\nCurrent: <b>${enabled ? "ON" : "OFF"}</b>\n\nUsage: <code>/antibio on</code> or <code>/antibio off</code>`,
      { parse_mode: "HTML" },
    )
    return
  }

  if (arg !== "on" && arg !== "off") {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: /antibio [on/off]")
    return
  }

  const enabled = arg === "on"

  await supabase
    .from("antibio_settings")
    .upsert({ chat_id: ctx.chat.id, enabled, updated_at: new Date().toISOString() }, { onConflict: "chat_id" })

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `🔍 Anti-Bio Link has been turned <b>${enabled ? "ON" : "OFF"}</b>.\n\n${enabled ? "Users with links in their bio will now have their messages deleted instantly." : "Users can now send messages freely."}`,
    { parse_mode: "HTML" },
  )
}

// /free [user] - Exempt user from antibio checks
export async function handleFree(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  if (!ctx.isAdmin && !ctx.isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.")
    return
  }

  let targetUserId: number | undefined
  let targetName: string | undefined

  if (ctx.replyToMessage?.from) {
    targetUserId = ctx.replyToMessage.from.id
    targetName = ctx.replyToMessage.from.first_name
  } else if (ctx.args[0] && /^\d+$/.test(ctx.args[0])) {
    targetUserId = Number.parseInt(ctx.args[0])
    targetName = `User ${targetUserId}`
  }

  if (!targetUserId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: Reply to a user's message with /free or use /free [user_id]")
    return
  }

  await supabase
    .from("antibio_free_users")
    .upsert(
      { chat_id: ctx.chat.id, user_id: targetUserId, added_by: ctx.user.id, created_at: new Date().toISOString() },
      { onConflict: "chat_id,user_id" },
    )

  await ctx.bot.sendMessage(ctx.chat.id, `✅ ${targetName} is now exempt from anti-bio checks in this group.`)
}

// /unfree [user] - Remove exemption
export async function handleUnfree(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  if (!ctx.isAdmin && !ctx.isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.")
    return
  }

  let targetUserId: number | undefined
  let targetName: string | undefined

  if (ctx.replyToMessage?.from) {
    targetUserId = ctx.replyToMessage.from.id
    targetName = ctx.replyToMessage.from.first_name
  } else if (ctx.args[0] && /^\d+$/.test(ctx.args[0])) {
    targetUserId = Number.parseInt(ctx.args[0])
    targetName = `User ${targetUserId}`
  }

  if (!targetUserId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: Reply to a user's message with /unfree or use /unfree [user_id]")
    return
  }

  await supabase.from("antibio_free_users").delete().eq("chat_id", ctx.chat.id).eq("user_id", targetUserId)

  await ctx.bot.sendMessage(ctx.chat.id, `✅ ${targetName} is no longer exempt from anti-bio checks.`)
}

// /freelist - List exempt users
export async function handleFreeList(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  if (!ctx.isAdmin && !ctx.isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.")
    return
  }

  const { data } = await supabase.from("antibio_free_users").select("user_id").eq("chat_id", ctx.chat.id)

  if (!data?.length) {
    await ctx.bot.sendMessage(ctx.chat.id, "No users are exempt from anti-bio checks in this group.")
    return
  }

  let text = `📋 <b>Anti-Bio Exempt Users (${data.length})</b>\n\n`
  for (const row of data) {
    text += `• <code>${row.user_id}</code>\n`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}
