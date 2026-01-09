import { supabase } from "../utils"
import type { CommandContext } from "../types"
import { getTargetUser, mentionUser } from "../utils"
import { OWNER_ID } from "../constants"

// Check if user is owner
export function isOwner(userId: number): boolean {
  return userId === OWNER_ID
}

// Check if user is sudoer (from database)
export async function isSudoUser(userId: number): Promise<boolean> {
  if (userId === OWNER_ID) return true

  const { data } = await supabase.from("sudo_users").select("user_id").eq("user_id", userId).maybeSingle()

  return !!data
}

// Check if user is globally banned
export async function isGloballyBanned(userId: number): Promise<boolean> {
  const { data } = await supabase.from("global_bans").select("user_id").eq("user_id", userId).maybeSingle()

  return !!data
}

// Get global ban info
export async function getGbanInfo(userId: number): Promise<{ reason: string; banned_by: number } | null> {
  const { data } = await supabase.from("global_bans").select("reason, banned_by").eq("user_id", userId).maybeSingle()

  return data
}

// Get all chats where bot is admin
async function getAllBotChats(): Promise<number[]> {
  const { data } = await supabase.from("chats").select("chat_id")

  return data?.map((c) => Number(c.chat_id)) || []
}

// Handle /gban command - Only owner and sudo users
export async function handleGban(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id

  // Check if user is owner or sudoer
  if (!isOwner(userId) && !(await isSudoUser(userId))) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used by the bot owner or sudo users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to globally ban.")
    return
  }

  // Can't gban owner or sudoers
  if (isOwner(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "You cannot globally ban the bot owner.")
    return
  }

  if (await isSudoUser(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "You cannot globally ban a sudo user. Remove their sudo first.")
    return
  }

  // Check if already gbanned
  if (await isGloballyBanned(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This user is already globally banned.")
    return
  }

  // Get reason
  const reasonArgs = target.user ? ctx.args : ctx.args.slice(1)
  const reason = reasonArgs.join(" ") || "No reason provided"

  // Add to global bans
  const { error } = await supabase.from("global_bans").insert({
    user_id: target.userId,
    reason,
    banned_by: userId,
  })

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to globally ban user: ${error.message}`)
    return
  }

  const targetName = target.user ? mentionUser(target.user) : `User ${target.userId}`

  // Send initial message
  const statusMsg = await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>New Global Ban</b>\n\n` +
      `<b>User:</b> ${targetName}\n` +
      `<b>ID:</b> <code>${target.userId}</code>\n` +
      `<b>Reason:</b> ${reason}\n\n` +
      `Banning across all chats...`,
    { parse_mode: "HTML" },
  )

  // Ban from all chats where bot is present
  const chats = await getAllBotChats()
  let successCount = 0
  let failCount = 0

  for (const chatId of chats) {
    try {
      await ctx.bot.banChatMember(chatId, target.userId)
      successCount++
    } catch {
      failCount++
    }
  }

  // Update status message
  await ctx.bot.editMessageText(
    ctx.chat.id,
    statusMsg.message_id,
    `<b>Global Ban Complete</b>\n\n` +
      `<b>User:</b> ${targetName}\n` +
      `<b>ID:</b> <code>${target.userId}</code>\n` +
      `<b>Reason:</b> ${reason}\n\n` +
      `<b>Results:</b>\n` +
      `• Banned in ${successCount} chats\n` +
      `• Failed in ${failCount} chats`,
    { parse_mode: "HTML" },
  )
}

// Handle /ungban command - Only owner and sudo users
export async function handleUngban(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id

  if (!isOwner(userId) && !(await isSudoUser(userId))) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used by the bot owner or sudo users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to remove global ban.")
    return
  }

  // Check if gbanned
  if (!(await isGloballyBanned(target.userId))) {
    await ctx.bot.sendMessage(ctx.chat.id, "This user is not globally banned.")
    return
  }

  // Remove from global bans
  const { error } = await supabase.from("global_bans").delete().eq("user_id", target.userId)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to remove global ban: ${error.message}`)
    return
  }

  const targetName = target.user ? mentionUser(target.user) : `User ${target.userId}`

  // Unban from all chats
  const chats = await getAllBotChats()
  let successCount = 0

  for (const chatId of chats) {
    try {
      await ctx.bot.unbanChatMember(chatId, target.userId, { only_if_banned: true })
      successCount++
    } catch {
      // Ignore errors
    }
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Global Ban Removed</b>\n\n` +
      `<b>User:</b> ${targetName}\n` +
      `<b>ID:</b> <code>${target.userId}</code>\n\n` +
      `Unbanned from ${successCount} chats.`,
    { parse_mode: "HTML" },
  )
}

// Handle /gbanlist command
export async function handleGbanList(ctx: CommandContext): Promise<void> {
  const userId = ctx.user.id

  if (!isOwner(userId) && !(await isSudoUser(userId))) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used by the bot owner or sudo users.")
    return
  }

  const { data: gbans } = await supabase
    .from("global_bans")
    .select("user_id, reason, created_at")
    .order("created_at", { ascending: false })
    .limit(50)

  if (!gbans || gbans.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No users are globally banned.")
    return
  }

  let text = `<b>Globally Banned Users</b> (${gbans.length})\n\n`

  for (const gban of gbans) {
    text += `• <code>${gban.user_id}</code> - ${gban.reason || "No reason"}\n`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// Handle /addsudo command - Only owner
export async function handleAddSudo(ctx: CommandContext): Promise<void> {
  if (!isOwner(ctx.user.id)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used by the bot owner.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to add as sudo.")
    return
  }

  if (isOwner(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "The owner is already a super user by default.")
    return
  }

  if (await isSudoUser(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This user is already a sudo user.")
    return
  }

  const { error } = await supabase.from("sudo_users").insert({
    user_id: target.userId,
    added_by: ctx.user.id,
  })

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to add sudo user: ${error.message}`)
    return
  }

  const targetName = target.user ? mentionUser(target.user) : `User ${target.userId}`

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>New Sudo User</b>\n\n` +
      `${targetName} has been added as a sudo user.\n` +
      `They can now use /gban and /ungban commands.`,
    { parse_mode: "HTML" },
  )
}

// Handle /rmsudo command - Only owner
export async function handleRmSudo(ctx: CommandContext): Promise<void> {
  if (!isOwner(ctx.user.id)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used by the bot owner.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target?.userId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to remove sudo.")
    return
  }

  if (!(await isSudoUser(target.userId)) || isOwner(target.userId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This user is not a sudo user.")
    return
  }

  const { error } = await supabase.from("sudo_users").delete().eq("user_id", target.userId)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to remove sudo user: ${error.message}`)
    return
  }

  const targetName = target.user ? mentionUser(target.user) : `User ${target.userId}`

  await ctx.bot.sendMessage(ctx.chat.id, `<b>Sudo User Removed</b>\n\n` + `${targetName} is no longer a sudo user.`, {
    parse_mode: "HTML",
  })
}

// Handle /sudolist command - Only owner
export async function handleSudoList(ctx: CommandContext): Promise<void> {
  if (!isOwner(ctx.user.id)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used by the bot owner.")
    return
  }

  const { data: sudos } = await supabase
    .from("sudo_users")
    .select("user_id, created_at")
    .order("created_at", { ascending: true })

  let text = `<b>Sudo Users</b>\n\n`
  text += `<b>Owner:</b> <code>${OWNER_ID}</code>\n\n`

  if (!sudos || sudos.length === 0) {
    text += "No additional sudo users."
  } else {
    text += `<b>Sudo Users:</b>\n`
    for (const sudo of sudos) {
      text += `• <code>${sudo.user_id}</code>\n`
    }
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// Handle /gbanstat command - Check if user is gbanned
export async function handleGbanStat(ctx: CommandContext): Promise<void> {
  const target = await getTargetUser(ctx.message, ctx.args)
  const targetId = target?.userId || ctx.user.id

  const gbanInfo = await getGbanInfo(targetId)

  if (!gbanInfo) {
    await ctx.bot.sendMessage(ctx.chat.id, `User <code>${targetId}</code> is not globally banned.`, {
      parse_mode: "HTML",
    })
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Global Ban Info</b>\n\n` +
      `<b>User ID:</b> <code>${targetId}</code>\n` +
      `<b>Reason:</b> ${gbanInfo.reason || "No reason provided"}\n` +
      `<b>Banned by:</b> <code>${gbanInfo.banned_by}</code>`,
    { parse_mode: "HTML" },
  )
}
