import { createClient } from "@supabase/supabase-js"
import type { TelegramMessage, TelegramUser, TelegramChat, CommandContext } from "./types"
import { bot } from "./bot"
import { OWNER_ID } from "./constants"

// Supabase client for bot operations
const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
const supabaseKey = process.env.SUPABASE_SERVICE_ROLE_KEY

if (!supabaseUrl || !supabaseKey) {
  console.error("[v0] Missing Supabase credentials")
}

export const supabase = createClient(supabaseUrl || "", supabaseKey || "")

// Parse command and arguments from message
export function parseCommand(text: string): { command: string; args: string[] } | null {
  if (!text || !text.startsWith("/")) return null

  const parts = text.split(/\s+/)
  let command = parts[0].substring(1).toLowerCase()

  // Handle commands with @botusername
  if (command.includes("@")) {
    const [cmd, botUsername] = command.split("@")
    const ourBotUsername = process.env.BOT_USERNAME?.toLowerCase()
    if (ourBotUsername && botUsername !== ourBotUsername) {
      return null // Command is for another bot
    }
    command = cmd
  }

  const args = parts.slice(1)
  return { command, args }
}

// Get target user from message (reply or args)
export async function getTargetUser(
  message: TelegramMessage,
  args: string[],
): Promise<{ userId: number; user?: TelegramUser } | null> {
  console.log("[v0] getTargetUser called")
  console.log("[v0] message.reply_to_message:", JSON.stringify(message.reply_to_message, null, 2))
  console.log("[v0] args:", args)

  // If replying to a message, get that user
  if (message.reply_to_message) {
    // Check for 'from' field (normal messages)
    if (message.reply_to_message.from) {
      console.log("[v0] Found reply_to_message.from:", message.reply_to_message.from)
      return {
        userId: message.reply_to_message.from.id,
        user: message.reply_to_message.from,
      }
    }
    // Check for 'sender_chat' field (messages from channels/anonymous admins)
    if (message.reply_to_message.sender_chat) {
      console.log("[v0] Found reply_to_message.sender_chat:", message.reply_to_message.sender_chat)
      return {
        userId: message.reply_to_message.sender_chat.id,
      }
    }
    // Check for forwarded message
    if (message.reply_to_message.forward_from) {
      console.log("[v0] Found reply_to_message.forward_from:", message.reply_to_message.forward_from)
      return {
        userId: message.reply_to_message.forward_from.id,
        user: message.reply_to_message.forward_from,
      }
    }
  }

  // If username or user ID provided in args
  if (args.length > 0) {
    const target = args[0]
    console.log("[v0] Checking args target:", target)

    // If it's a user ID (numeric)
    if (/^\d+$/.test(target)) {
      console.log("[v0] Target is user ID:", target)
      return { userId: Number.parseInt(target) }
    }

    // If it's a username (with or without @)
    const username = target.startsWith("@") ? target.substring(1) : target
    if (username && /^[a-zA-Z][a-zA-Z0-9_]{4,31}$/.test(username)) {
      console.log("[v0] Looking up username:", username)
      const { data: user, error } = await supabase.from("users").select("user_id").ilike("username", username).single()

      if (error) {
        console.log("[v0] Error looking up username:", error)
      }

      if (user) {
        console.log("[v0] Found user by username:", user.user_id)
        return { userId: user.user_id }
      }

      // If not found in DB but looks like a valid username, return null with a note
      console.log("[v0] Username not found in database:", username)
    }
  }

  console.log("[v0] No target user found")
  return null
}

// Check if user is admin in chat
export async function isUserAdmin(chatId: number, userId: number): Promise<boolean> {
  try {
    const member = await bot.getChatMember(chatId, userId)
    return ["creator", "administrator"].includes(member.status)
  } catch (e) {
    console.log("[v0] Error checking admin status:", e)
    return false
  }
}

// Check if user is chat owner
export async function isUserOwner(chatId: number, userId: number): Promise<boolean> {
  try {
    const member = await bot.getChatMember(chatId, userId)
    return member.status === "creator"
  } catch (e) {
    console.log("[v0] Error checking owner status:", e)
    return false
  }
}

// Check if user is bot sudoer
export async function isSudoer(userId: number): Promise<boolean> {
  // Owner is always a sudoer
  if (userId === OWNER_ID) return true

  const { data } = await supabase.from("sudo_users").select("user_id").eq("user_id", userId).maybeSingle()
  return !!data
}

// Check if user is THE owner
export function isOwnerBot(userId: number): boolean {
  return userId === OWNER_ID
}

// Check if user is approved (bypass restrictions)
export async function isApproved(chatId: number, userId: number): Promise<boolean> {
  const { data } = await supabase
    .from("approved_users")
    .select("id")
    .eq("chat_id", chatId)
    .eq("user_id", userId)
    .maybeSingle()

  return !!data
}

// Ensure user exists in database
export async function ensureUser(user: TelegramUser): Promise<void> {
  try {
    await supabase.from("users").upsert(
      {
        user_id: user.id,
        username: user.username,
        first_name: user.first_name,
        last_name: user.last_name,
        language_code: user.language_code,
        is_bot: user.is_bot,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "user_id" },
    )
  } catch (e) {
    console.log("[v0] Error ensuring user:", e)
  }
}

// Ensure chat exists in database
export async function ensureChat(chat: TelegramChat): Promise<void> {
  try {
    await supabase.from("chats").upsert(
      {
        chat_id: chat.id,
        chat_type: chat.type,
        chat_name: chat.title || chat.first_name,
        chat_username: chat.username,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
  } catch (e) {
    console.log("[v0] Error ensuring chat:", e)
  }
}

// Build command context
export async function buildContext(message: TelegramMessage, args: string[]): Promise<CommandContext> {
  const chat = message.chat
  const user = message.from!

  const [isAdmin, isOwner, isSudoerUser, isApprovedUser] = await Promise.all([
    isUserAdmin(chat.id, user.id),
    isUserOwner(chat.id, user.id),
    isSudoer(user.id),
    isApproved(chat.id, user.id),
  ])

  return {
    message,
    chat,
    user,
    args,
    replyToMessage: message.reply_to_message,
    isAdmin,
    isOwner,
    isSudoer: isSudoerUser,
    isApproved: isApprovedUser,
  }
}

// Parse time string (1d, 2h, 30m, etc.)
export function parseTime(timeStr: string): number | null {
  const match = timeStr.match(/^(\d+)([smhdw])$/)
  if (!match) return null

  const value = Number.parseInt(match[1])
  const unit = match[2]

  const multipliers: Record<string, number> = {
    s: 1,
    m: 60,
    h: 3600,
    d: 86400,
    w: 604800,
  }

  return value * multipliers[unit]
}

// Format user mention
export function mentionUser(user: TelegramUser): string {
  return `<a href="tg://user?id=${user.id}">${escapeHtml(user.first_name)}</a>`
}

// Escape HTML for Telegram
export function escapeHtml(text: string): string {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
}

// Format variables in welcome/note text
export function formatText(text: string, user: TelegramUser, chat: TelegramChat): string {
  return text
    .replace(/{first}/gi, escapeHtml(user.first_name))
    .replace(/{last}/gi, escapeHtml(user.last_name || ""))
    .replace(/{fullname}/gi, escapeHtml(`${user.first_name}${user.last_name ? " " + user.last_name : ""}`))
    .replace(/{username}/gi, user.username ? `@${user.username}` : mentionUser(user))
    .replace(/{mention}/gi, mentionUser(user))
    .replace(/{id}/gi, user.id.toString())
    .replace(/{chatname}/gi, escapeHtml(chat.title || ""))
    .replace(/{chatid}/gi, chat.id.toString())
}

// Parse buttons from text [Button Text](buttonurl://example.com)
export function parseButtons(text: string): {
  text: string
  buttons: { text: string; url?: string; callback_data?: string }[][]
} {
  const buttons: { text: string; url?: string; callback_data?: string }[][] = []
  let currentRow: { text: string; url?: string; callback_data?: string }[] = []

  // Match button patterns
  const buttonRegex = /\[([^\]]+)\]$$buttonurl:\/\/([^)]+)$$/g
  let cleanText = text
  let match

  while ((match = buttonRegex.exec(text)) !== null) {
    const [fullMatch, buttonText, url] = match

    // Check for :same to keep in same row
    const isSameRow = url.endsWith(":same")
    const cleanUrl = isSameRow ? url.slice(0, -5) : url

    currentRow.push({ text: buttonText, url: cleanUrl })

    if (!isSameRow) {
      buttons.push(currentRow)
      currentRow = []
    }

    cleanText = cleanText.replace(fullMatch, "")
  }

  if (currentRow.length > 0) {
    buttons.push(currentRow)
  }

  return { text: cleanText.trim(), buttons }
}

// Check if command is disabled in chat
export async function isCommandDisabled(chatId: number, command: string): Promise<boolean> {
  const { data } = await supabase
    .from("disabled_commands")
    .select("id")
    .eq("chat_id", chatId)
    .eq("command", command)
    .single()

  return !!data
}

// Log action to log channel
export async function logAction(chatId: number, action: string, details: string): Promise<void> {
  const { data: logSettings } = await supabase
    .from("log_channels")
    .select("log_channel_id")
    .eq("chat_id", chatId)
    .single()

  if (logSettings?.log_channel_id) {
    try {
      await bot.sendMessage(logSettings.log_channel_id, `<b>${action}</b>\n${details}`, { parse_mode: "HTML" })
    } catch (error) {
      console.error("[v0] Failed to log action:", error)
    }
  }
}

// Get reason from args (skip first arg if it's a user)
export function getReason(args: string[], skipFirst = true): string | undefined {
  const reasonArgs = skipFirst ? args.slice(1) : args
  return reasonArgs.length > 0 ? reasonArgs.join(" ") : undefined
}

export async function getAdminPermissions(
  chatId: number,
  userId: number,
): Promise<{
  isAdmin: boolean
  isCreator: boolean
  canRestrictMembers: boolean
  canDeleteMessages: boolean
  canPinMessages: boolean
  canPromoteMembers: boolean
  canChangeInfo: boolean
  canInviteUsers: boolean
  canManageChat: boolean
}> {
  try {
    const member = await bot.getChatMember(chatId, userId)

    if (member.status === "creator") {
      return {
        isAdmin: true,
        isCreator: true,
        canRestrictMembers: true,
        canDeleteMessages: true,
        canPinMessages: true,
        canPromoteMembers: true,
        canChangeInfo: true,
        canInviteUsers: true,
        canManageChat: true,
      }
    }

    if (member.status === "administrator") {
      return {
        isAdmin: true,
        isCreator: false,
        canRestrictMembers: member.can_restrict_members || false,
        canDeleteMessages: member.can_delete_messages || false,
        canPinMessages: member.can_pin_messages || false,
        canPromoteMembers: member.can_promote_members || false,
        canChangeInfo: member.can_change_info || false,
        canInviteUsers: member.can_invite_users || false,
        canManageChat: member.can_manage_chat || false,
      }
    }

    return {
      isAdmin: false,
      isCreator: false,
      canRestrictMembers: false,
      canDeleteMessages: false,
      canPinMessages: false,
      canPromoteMembers: false,
      canChangeInfo: false,
      canInviteUsers: false,
      canManageChat: false,
    }
  } catch (e) {
    console.log("[v0] Error getting admin permissions:", e)
    return {
      isAdmin: false,
      isCreator: false,
      canRestrictMembers: false,
      canDeleteMessages: false,
      canPinMessages: false,
      canPromoteMembers: false,
      canChangeInfo: false,
      canInviteUsers: false,
      canManageChat: false,
    }
  }
}

export async function getChatAdmins(
  chatId: number,
): Promise<Array<{ userId: number; username?: string; firstName: string }>> {
  try {
    const admins = await bot.getChatAdministrators(chatId)
    return admins
      .filter((admin) => !admin.user.is_bot)
      .map((admin) => ({
        userId: admin.user.id,
        username: admin.user.username,
        firstName: admin.user.first_name,
      }))
  } catch (e) {
    console.log("[v0] Error getting chat admins:", e)
    return []
  }
}
