import { supabase } from "../utils"
import type { CommandContext, TelegramMessage } from "../types"
import { mentionUser } from "../utils"
import type { TelegramBot } from "../bot"

// Set user as AFK
export async function setAfk(userId: number, reason?: string): Promise<void> {
  await supabase.from("afk_users").upsert({
    user_id: userId,
    reason: reason || null,
    afk_time: new Date().toISOString(),
  })
}

// Remove AFK status
export async function removeAfk(userId: number): Promise<boolean> {
  const { data } = await supabase.from("afk_users").select("user_id").eq("user_id", userId).maybeSingle()

  if (!data) return false

  await supabase.from("afk_users").delete().eq("user_id", userId)

  return true
}

// Get AFK info
export async function getAfkInfo(userId: number): Promise<{ reason: string | null; afk_time: string } | null> {
  const { data } = await supabase.from("afk_users").select("reason, afk_time").eq("user_id", userId).maybeSingle()

  return data
}

// Check if user is AFK
export async function isAfk(userId: number): Promise<boolean> {
  const { data } = await supabase.from("afk_users").select("user_id").eq("user_id", userId).maybeSingle()

  return !!data
}

// Format time duration
function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) {
    return `${days}d ${hours % 24}h ${minutes % 60}m`
  } else if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  } else {
    return `${seconds}s`
  }
}

// Handle /afk command
export async function handleAfk(ctx: CommandContext): Promise<void> {
  const reason = ctx.args.join(" ") || undefined

  await setAfk(ctx.user.id, reason)

  const userName = mentionUser(ctx.user)
  let text = `${userName} is now AFK.`

  if (reason) {
    text = `${userName} is now AFK.\nReason: ${reason}`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// Handle /brb command (alias for /afk)
export async function handleBrb(ctx: CommandContext): Promise<void> {
  await handleAfk(ctx)
}

// Check AFK when user is mentioned or replied to
export async function checkAfkMention(message: TelegramMessage, bot: TelegramBot): Promise<void> {
  // Skip if in private chat
  if (message.chat.type === "private") return

  const mentionedUsers: number[] = []

  // Check reply
  if (message.reply_to_message?.from && !message.reply_to_message.from.is_bot) {
    mentionedUsers.push(message.reply_to_message.from.id)
  }

  // Check mentions in entities
  if (message.entities) {
    for (const entity of message.entities) {
      if (entity.type === "mention" && message.text) {
        // Get username from text
        const username = message.text.substring(entity.offset + 1, entity.offset + entity.length)
        // Try to find user by username in database
        const { data: userData } = await supabase
          .from("users")
          .select("user_id")
          .eq("username", username.toLowerCase())
          .maybeSingle()

        if (userData) {
          mentionedUsers.push(Number(userData.user_id))
        }
      } else if (entity.type === "text_mention" && entity.user) {
        mentionedUsers.push(entity.user.id)
      }
    }
  }

  // Check each mentioned user
  for (const userId of mentionedUsers) {
    // Skip if mentioning themselves
    if (userId === message.from?.id) continue

    const afkInfo = await getAfkInfo(userId)
    if (afkInfo) {
      const afkTime = new Date(afkInfo.afk_time).getTime()
      const duration = formatDuration(Date.now() - afkTime)

      let text = `This user is currently AFK (away for ${duration}).`
      if (afkInfo.reason) {
        text = `This user is currently AFK (away for ${duration}).\nReason: ${afkInfo.reason}`
      }

      await bot.sendMessage(message.chat.id, text, {
        reply_to_message_id: message.message_id,
      })
    }
  }
}

// Check if user returned from AFK
export async function checkAfkReturn(message: TelegramMessage, bot: TelegramBot): Promise<void> {
  if (!message.from) return

  // Skip /afk command itself
  const text = message.text || ""
  if (text.startsWith("/afk") || text.startsWith("/brb")) return

  // Check if user was AFK
  const wasAfk = await removeAfk(message.from.id)

  if (wasAfk) {
    const userName = mentionUser(message.from)
    await bot.sendMessage(message.chat.id, `${userName} is no longer AFK!`, { parse_mode: "HTML" })
  }
}
