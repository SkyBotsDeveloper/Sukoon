import { supabase } from "../utils"
import type { CommandContext } from "../types"
import { getTargetUser, mentionUser } from "../utils"
import { OWNER_ID, BOT_NAME } from "../constants"

// Check if user is the bot owner
export function isOwner(userId: number): boolean {
  return userId === OWNER_ID
}

// Check if user is sudoer (owner or in sudo_users table)
export async function isSudoUser(userId: number): Promise<boolean> {
  if (userId === OWNER_ID) return true
  const { data } = await supabase.from("sudo_users").select("user_id").eq("user_id", userId).maybeSingle()
  return !!data
}

// Check if chat is blacklisted
export async function isBlacklistedChat(chatId: number): Promise<boolean> {
  const { data } = await supabase.from("blacklisted_chats").select("chat_id").eq("chat_id", chatId).maybeSingle()
  return !!data
}

// Check if user is blacklisted
export async function isBlacklistedUser(userId: number): Promise<boolean> {
  const { data } = await supabase.from("blacklisted_users").select("user_id").eq("user_id", userId).maybeSingle()
  return !!data
}

// Get all chats where bot exists
async function getAllChats(): Promise<{ chat_id: number; chat_name: string }[]> {
  const { data } = await supabase.from("chats").select("chat_id, chat_name")
  return data?.map((c) => ({ chat_id: Number(c.chat_id), chat_name: c.chat_name })) || []
}

// Get all users who started the bot
async function getAllUsers(): Promise<number[]> {
  const { data } = await supabase.from("users").select("user_id")
  return data?.map((u) => Number(u.user_id)) || []
}

// Get all bot clones
async function getAllClones(): Promise<{ bot_token: string; bot_username: string }[]> {
  const { data } = await supabase.from("bot_clones").select("bot_token, bot_username")
  return data || []
}

// ============================================
// /gban - Global ban user from all chats
// ============================================
export async function handleGban(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide user ID to globally ban.")
    return
  }

  if (target.userId === OWNER_ID) {
    await ctx.bot.sendMessage(ctx.chat.id, "Cannot gban the owner.")
    return
  }

  if (await isSudoUser(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "Cannot gban a sudo user. Remove sudo first with /rmsudo.")
    return
  }

  const reason = (target.user ? ctx.args.slice(0) : ctx.args.slice(1)).join(" ") || "No reason"

  // Add to global bans
  await supabase
    .from("global_bans")
    .upsert({ user_id: target.userId, reason, banned_by: ctx.user.id }, { onConflict: "user_id" })

  const targetName = target.user ? mentionUser(target.user) : `User ${target.userId}`
  const statusMsg = await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Global Ban</b>\n\nUser: ${targetName}\nID: <code>${target.userId}</code>\nReason: ${reason}\n\nBanning across all chats...`,
    { parse_mode: "HTML" },
  )

  // Ban from all Sukoon chats
  const chats = await getAllChats()
  let sukoonBanned = 0
  for (const chat of chats) {
    try {
      await ctx.bot.banChatMember(chat.chat_id, target.userId)
      sukoonBanned++
    } catch {}
  }

  // Ban from all clone chats
  const clones = await getAllClones()
  let cloneBanned = 0
  for (const clone of clones) {
    try {
      const cloneChats = await getAllChats()
      for (const chat of cloneChats) {
        const res = await fetch(`https://api.telegram.org/bot${clone.bot_token}/banChatMember`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ chat_id: chat.chat_id, user_id: target.userId }),
        })
        if (res.ok) cloneBanned++
      }
    } catch {}
  }

  await ctx.bot.editMessageText(
    ctx.chat.id,
    statusMsg.message_id,
    `<b>Global Ban Complete</b>\n\nUser: ${targetName}\nID: <code>${target.userId}</code>\nReason: ${reason}\n\nBanned in ${sukoonBanned} Sukoon chats\nBanned via ${clones.length} clones`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /ungban - Remove global ban
// ============================================
export async function handleUngban(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide user ID to ungban.")
    return
  }

  await supabase.from("global_bans").delete().eq("user_id", target.userId)

  const chats = await getAllChats()
  let unbanned = 0
  for (const chat of chats) {
    try {
      await ctx.bot.unbanChatMember(chat.chat_id, target.userId, { only_if_banned: true })
      unbanned++
    } catch {}
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Global Unban</b>\n\nUser: <code>${target.userId}</code>\nUnbanned from ${unbanned} chats.`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /broadcast - Send message to all chats/users
// ============================================
export async function handleBroadcast(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const message = ctx.args.join(" ")
  if (!message && !ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: /broadcast <message> or reply to a message")
    return
  }

  const statusMsg = await ctx.bot.sendMessage(ctx.chat.id, "Broadcasting message...")

  const chats = await getAllChats()
  const users = await getAllUsers()

  const sentTo = new Set<number>()

  let chatSuccess = 0,
    chatFail = 0,
    userSuccess = 0,
    userFail = 0

  // Broadcast to all chats (groups/channels)
  for (const chat of chats) {
    try {
      // Skip if already sent (shouldn't happen for groups but just in case)
      if (sentTo.has(chat.chat_id)) continue
      sentTo.add(chat.chat_id)

      if (ctx.replyToMessage) {
        await ctx.bot.copyMessage(chat.chat_id, ctx.chat.id, ctx.replyToMessage.message_id)
      } else {
        await ctx.bot.sendMessage(chat.chat_id, message, { parse_mode: "HTML" })
      }
      chatSuccess++

      await new Promise((resolve) => setTimeout(resolve, 50))
    } catch {
      chatFail++
    }
  }

  // Broadcast to all users in DM (skip users who are already in groups)
  for (const userId of users) {
    try {
      if (sentTo.has(userId)) continue
      sentTo.add(userId)

      if (ctx.replyToMessage) {
        await ctx.bot.copyMessage(userId, ctx.chat.id, ctx.replyToMessage.message_id)
      } else {
        await ctx.bot.sendMessage(userId, message, { parse_mode: "HTML" })
      }
      userSuccess++

      await new Promise((resolve) => setTimeout(resolve, 50))
    } catch {
      userFail++
    }
  }

  await ctx.bot.editMessageText(
    ctx.chat.id,
    statusMsg.message_id,
    `<b>Broadcast Complete</b>\n\nChats: ${chatSuccess} sent, ${chatFail} failed\nUsers: ${userSuccess} sent, ${userFail} failed`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /blchat - Blacklist a chat
// ============================================
export async function handleBlChat(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const chatId = ctx.args[0]
  if (!chatId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: /blchat <chat_id> [reason]")
    return
  }

  const reason = ctx.args.slice(1).join(" ") || "No reason"
  const chatIdNum = Number(chatId)

  // Add to blacklist
  await supabase
    .from("blacklisted_chats")
    .upsert({ chat_id: chatIdNum, reason, blacklisted_by: ctx.user.id }, { onConflict: "chat_id" })

  // Leave the chat with Sukoon
  try {
    await ctx.bot.leaveChat(chatIdNum)
  } catch {}

  // Leave with all clones
  const clones = await getAllClones()
  for (const clone of clones) {
    try {
      await fetch(`https://api.telegram.org/bot${clone.bot_token}/leaveChat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ chat_id: chatIdNum }),
      })
    } catch {}
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Chat Blacklisted</b>\n\nChat ID: <code>${chatIdNum}</code>\nReason: ${reason}\n\n${BOT_NAME} and all clones have left this chat and cannot be added back.`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /unblchat - Remove chat from blacklist
// ============================================
export async function handleUnblChat(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const chatId = ctx.args[0]
  if (!chatId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: /unblchat <chat_id>")
    return
  }

  await supabase.from("blacklisted_chats").delete().eq("chat_id", Number(chatId))

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Chat Unblacklisted</b>\n\nChat ID: <code>${chatId}</code>\n\nThis chat can now add ${BOT_NAME} again.`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /bluser - Blacklist a user
// ============================================
export async function handleBlUser(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide user ID to blacklist.")
    return
  }

  if (target.userId === OWNER_ID || (await isSudoUser(target.userId))) {
    await ctx.bot.sendMessage(ctx.chat.id, "Cannot blacklist owner or sudo users.")
    return
  }

  const reason = (target.user ? ctx.args.slice(0) : ctx.args.slice(1)).join(" ") || "No reason"

  await supabase
    .from("blacklisted_users")
    .upsert({ user_id: target.userId, reason, blacklisted_by: ctx.user.id }, { onConflict: "user_id" })

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>User Blacklisted</b>\n\nUser: <code>${target.userId}</code>\nReason: ${reason}\n\nThis user can no longer use ${BOT_NAME} or any of its clones.`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /unbluser - Remove user from blacklist
// ============================================
export async function handleUnblUser(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide user ID to unblacklist.")
    return
  }

  await supabase.from("blacklisted_users").delete().eq("user_id", target.userId)

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>User Unblacklisted</b>\n\nUser: <code>${target.userId}</code>\n\nThis user can now use ${BOT_NAME} again.`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /stats - Show bot statistics
// ============================================
export async function handleStats(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const [
    { count: usersCount },
    { count: chatsCount },
    { count: clonesCount },
    { count: gbansCount },
    { count: fedsCount },
    { count: blChatsCount },
    { count: blUsersCount },
    { count: sudoCount },
  ] = await Promise.all([
    supabase.from("users").select("*", { count: "exact", head: true }),
    supabase.from("chats").select("*", { count: "exact", head: true }),
    supabase.from("bot_clones").select("*", { count: "exact", head: true }),
    supabase.from("global_bans").select("*", { count: "exact", head: true }),
    supabase.from("federations").select("*", { count: "exact", head: true }),
    supabase.from("blacklisted_chats").select("*", { count: "exact", head: true }),
    supabase.from("blacklisted_users").select("*", { count: "exact", head: true }),
    supabase.from("sudo_users").select("*", { count: "exact", head: true }),
  ])

  const text = `<b>${BOT_NAME} Statistics</b>

<b>Users:</b> ${usersCount || 0}
<b>Groups:</b> ${chatsCount || 0}
<b>Clones:</b> ${clonesCount || 0}
<b>Federations:</b> ${fedsCount || 0}

<b>Global Bans:</b> ${gbansCount || 0}
<b>Blacklisted Chats:</b> ${blChatsCount || 0}
<b>Blacklisted Users:</b> ${blUsersCount || 0}
<b>Sudo Users:</b> ${(sudoCount || 0) + 1} (including owner)`

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// ============================================
// /addsudo - Add sudo user (owner only)
// ============================================
export async function handleAddSudo(ctx: CommandContext): Promise<void> {
  if (!isOwner(ctx.user.id)) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide user ID to add as sudo.")
    return
  }

  if (target.userId === OWNER_ID) {
    await ctx.bot.sendMessage(ctx.chat.id, "Owner is already sudo by default.")
    return
  }

  await supabase.from("sudo_users").upsert({ user_id: target.userId, added_by: ctx.user.id }, { onConflict: "user_id" })

  const targetName = target.user ? mentionUser(target.user) : `User ${target.userId}`

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Sudo Added</b>\n\n${targetName} now has sudo powers.\nThey can use /gban, /broadcast, /blchat, /bluser, /stats`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /rmsudo - Remove sudo user (owner only)
// ============================================
export async function handleRmSudo(ctx: CommandContext): Promise<void> {
  if (!isOwner(ctx.user.id)) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide user ID to remove sudo.")
    return
  }

  await supabase.from("sudo_users").delete().eq("user_id", target.userId)

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Sudo Removed</b>\n\nUser <code>${target.userId}</code> is no longer a sudo user.`,
    { parse_mode: "HTML" },
  )
}

// ============================================
// /sudolist - List all sudo users
// ============================================
export async function handleSudoList(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const { data: sudos } = await supabase.from("sudo_users").select("user_id, created_at")

  let text = `<b>Sudo Users</b>\n\n<b>Owner:</b> <code>${OWNER_ID}</code>\n\n`

  if (sudos && sudos.length > 0) {
    text += `<b>Sudo Users:</b>\n`
    for (const sudo of sudos) {
      text += `• <code>${sudo.user_id}</code>\n`
    }
  } else {
    text += `No additional sudo users.`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// ============================================
// /gbanlist - List all global bans
// ============================================
export async function handleGbanList(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const { data: gbans } = await supabase
    .from("global_bans")
    .select("user_id, reason")
    .order("created_at", { ascending: false })
    .limit(50)

  if (!gbans || gbans.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No globally banned users.")
    return
  }

  let text = `<b>Global Bans</b> (${gbans.length})\n\n`
  for (const gban of gbans) {
    text += `• <code>${gban.user_id}</code> - ${gban.reason || "No reason"}\n`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// ============================================
// /bllist - List all blacklisted chats/users
// ============================================
export async function handleBlList(ctx: CommandContext): Promise<void> {
  if (!(await isSudoUser(ctx.user.id))) return

  const [{ data: blChats }, { data: blUsers }] = await Promise.all([
    supabase.from("blacklisted_chats").select("chat_id, reason").limit(25),
    supabase.from("blacklisted_users").select("user_id, reason").limit(25),
  ])

  let text = `<b>Blacklist</b>\n\n<b>Chats:</b>\n`

  if (blChats && blChats.length > 0) {
    for (const chat of blChats) {
      text += `• <code>${chat.chat_id}</code> - ${chat.reason || "No reason"}\n`
    }
  } else {
    text += `No blacklisted chats.\n`
  }

  text += `\n<b>Users:</b>\n`

  if (blUsers && blUsers.length > 0) {
    for (const user of blUsers) {
      text += `• <code>${user.user_id}</code> - ${user.reason || "No reason"}\n`
    }
  } else {
    text += `No blacklisted users.`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}
