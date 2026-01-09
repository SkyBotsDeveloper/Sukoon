import type { CommandContext, TelegramUser } from "../types"
import {
  supabase,
  getTargetUser,
  parseTime,
  mentionUser,
  logAction,
  getReason,
  escapeHtml,
  getAdminPermissions,
} from "../utils"
import { canPerformAction } from "./silentpower"

function getUserDisplayName(target: { userId: number; user?: TelegramUser }): string {
  if (target.user) {
    return mentionUser(target.user)
  }
  return `<a href="tg://user?id=${target.userId}">User ${target.userId}</a>`
}

// ==================== BAN COMMANDS ====================

export async function handleBan(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  const canBan = permissions.canRestrictMembers || (await canPerformAction(ctx.chat.id, ctx.user.id, "ban"))

  if (!canBan) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to ban users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to ban.")
    return
  }

  // Check if user is trying to ban themselves
  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't ban yourself!")
    return
  }

  // Check if target is admin
  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't ban an admin!")
      return
    }
  } catch (e) {
    console.log("[v0] Error checking member status:", e)
  }

  const reason = getReason(ctx.args)

  try {
    await ctx.bot.banChatMember(ctx.chat.id, target.userId)

    await supabase.from("bans").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        banned_by: ctx.user.id,
        reason,
      },
      { onConflict: "chat_id,user_id" },
    )

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been banned!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.message.message_id,
      },
    )

    await logAction(
      ctx.chat.id,
      "BAN",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    console.error("[v0] Ban error:", errorMsg)
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to ban user: ${errorMsg}`)
  }
}

export async function handleTBan(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to ban users. You need 'Ban Users' permission.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Please reply to a user or provide a user ID and time (e.g., /tban @user 1d).",
    )
    return
  }

  // Find time argument
  let duration: number | null = null
  let timeArg = ""
  for (const arg of ctx.args) {
    const parsed = parseTime(arg)
    if (parsed) {
      duration = parsed
      timeArg = arg
      break
    }
  }

  if (!duration) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a valid time (e.g., 1h, 1d, 1w).")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't ban an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args.filter((a) => a !== timeArg))
  const untilDate = Math.floor(Date.now() / 1000) + duration

  try {
    await ctx.bot.banChatMember(ctx.chat.id, target.userId, { until_date: untilDate })

    await supabase.from("bans").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        banned_by: ctx.user.id,
        reason,
        until_date: new Date(untilDate * 1000).toISOString(),
      },
      { onConflict: "chat_id,user_id" },
    )

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been temporarily banned for ${timeArg}!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.message.message_id,
      },
    )

    await logAction(
      ctx.chat.id,
      "TBAN",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nDuration: ${timeArg}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to ban user: ${errorMsg}`)
  }
}

export async function handleDBan(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to ban users. You need 'Ban Users' permission.")
    return
  }
  if (!permissions.canDeleteMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to delete messages. You need 'Delete Messages' permission.",
    )
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to delete and ban the user.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Could not find target user.")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't ban an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = ctx.args.join(" ") || undefined

  try {
    // Delete the message first
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)

    await ctx.bot.banChatMember(ctx.chat.id, target.userId)

    await supabase.from("bans").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        banned_by: ctx.user.id,
        reason,
      },
      { onConflict: "chat_id,user_id" },
    )

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been banned and their message deleted!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      { parse_mode: "HTML" },
    )

    await logAction(
      ctx.chat.id,
      "DBAN",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to ban user: ${errorMsg}`)
  }
}

export async function handleSBan(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) return

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) return
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args)

  try {
    // Delete the command message
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.message.message_id)
  } catch {
    // Ignore
  }

  if (ctx.replyToMessage) {
    try {
      await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)
    } catch {
      // Ignore
    }
  }

  try {
    await ctx.bot.banChatMember(ctx.chat.id, target.userId)

    await supabase.from("bans").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        banned_by: ctx.user.id,
        reason,
      },
      { onConflict: "chat_id,user_id" },
    )

    await logAction(
      ctx.chat.id,
      "SBAN",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error) {
    console.error("[v0] SBan error:", error)
  }
}

export async function handleUnban(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to unban users. You need 'Ban Users' permission.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to unban.")
    return
  }

  try {
    await ctx.bot.unbanChatMember(ctx.chat.id, target.userId, { only_if_banned: true })

    await supabase.from("bans").delete().eq("chat_id", ctx.chat.id).eq("user_id", target.userId)

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(ctx.chat.id, `${userName} has been unbanned!`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.message.message_id,
    })

    await logAction(ctx.chat.id, "UNBAN", `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}`)
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to unban user: ${errorMsg}`)
  }
}

// ==================== MUTE COMMANDS ====================

export async function handleMute(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  const canMute = permissions.canRestrictMembers || (await canPerformAction(ctx.chat.id, ctx.user.id, "mute"))

  if (!canMute) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to mute users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to mute.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't mute yourself!")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't mute an admin!")
      return
    }
  } catch (e) {
    console.log("[v0] Error checking member:", e)
  }

  const reason = getReason(ctx.args)

  try {
    await ctx.bot.restrictChatMember(ctx.chat.id, target.userId, {
      can_send_messages: false,
      can_send_audios: false,
      can_send_documents: false,
      can_send_photos: false,
      can_send_videos: false,
      can_send_video_notes: false,
      can_send_voice_notes: false,
      can_send_polls: false,
      can_send_other_messages: false,
      can_add_web_page_previews: false,
      can_change_info: false,
      can_invite_users: false,
      can_pin_messages: false,
    })

    await supabase.from("mutes").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        muted_by: ctx.user.id,
        reason,
      },
      { onConflict: "chat_id,user_id" },
    )

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been muted!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.message.message_id,
      },
    )

    await logAction(
      ctx.chat.id,
      "MUTE",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    console.error("[v0] Mute error:", errorMsg)
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to mute user: ${errorMsg}`)
  }
}

export async function handleTMute(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to mute users. You need 'Restrict Members' permission.",
    )
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Please reply to a user or provide a user ID and time (e.g., /tmute @user 1h).",
    )
    return
  }

  let duration: number | null = null
  let timeArg = ""
  for (const arg of ctx.args) {
    const parsed = parseTime(arg)
    if (parsed) {
      duration = parsed
      timeArg = arg
      break
    }
  }

  if (!duration) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a valid time (e.g., 1h, 1d, 1w).")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't mute an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args.filter((a) => a !== timeArg))
  const untilDate = Math.floor(Date.now() / 1000) + duration

  try {
    await ctx.bot.restrictChatMember(
      ctx.chat.id,
      target.userId,
      {
        can_send_messages: false,
        can_send_audios: false,
        can_send_documents: false,
        can_send_photos: false,
        can_send_videos: false,
        can_send_video_notes: false,
        can_send_voice_notes: false,
        can_send_polls: false,
        can_send_other_messages: false,
        can_add_web_page_previews: false,
        can_change_info: false,
        can_invite_users: false,
        can_pin_messages: false,
      },
      untilDate,
    )

    await supabase.from("mutes").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        muted_by: ctx.user.id,
        reason,
        until_date: new Date(untilDate * 1000).toISOString(),
      },
      { onConflict: "chat_id,user_id" },
    )

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been muted for ${timeArg}!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.message.message_id,
      },
    )

    await logAction(
      ctx.chat.id,
      "TMUTE",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nDuration: ${timeArg}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to mute user: ${errorMsg}`)
  }
}

export async function handleDMute(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to mute users. You need 'Restrict Members' permission.",
    )
    return
  }
  if (!permissions.canDeleteMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to delete messages. You need 'Delete Messages' permission.",
    )
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to delete and mute the user.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Could not find target user.")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't mute an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = ctx.args.join(" ") || undefined

  try {
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)

    await ctx.bot.restrictChatMember(ctx.chat.id, target.userId, {
      can_send_messages: false,
      can_send_audios: false,
      can_send_documents: false,
      can_send_photos: false,
      can_send_videos: false,
      can_send_video_notes: false,
      can_send_voice_notes: false,
      can_send_polls: false,
      can_send_other_messages: false,
      can_add_web_page_previews: false,
      can_change_info: false,
      can_invite_users: false,
      can_pin_messages: false,
    })

    await supabase.from("mutes").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        muted_by: ctx.user.id,
        reason,
      },
      { onConflict: "chat_id,user_id" },
    )

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been muted and their message deleted!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      { parse_mode: "HTML" },
    )

    await logAction(
      ctx.chat.id,
      "DMUTE",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to mute user: ${errorMsg}`)
  }
}

export async function handleSMute(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) return

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) return
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args)

  try {
    // Delete the command message
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.message.message_id)
  } catch {
    // Ignore
  }

  if (ctx.replyToMessage) {
    try {
      await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)
    } catch {
      // Ignore
    }
  }

  try {
    await ctx.bot.restrictChatMember(ctx.chat.id, target.userId, {
      can_send_messages: false,
      can_send_audios: false,
      can_send_documents: false,
      can_send_photos: false,
      can_send_videos: false,
      can_send_video_notes: false,
      can_send_voice_notes: false,
      can_send_polls: false,
      can_send_other_messages: false,
      can_add_web_page_previews: false,
      can_change_info: false,
      can_invite_users: false,
      can_pin_messages: false,
    })

    await supabase.from("mutes").upsert(
      {
        chat_id: ctx.chat.id,
        user_id: target.userId,
        muted_by: ctx.user.id,
        reason,
      },
      { onConflict: "chat_id,user_id" },
    )

    await logAction(
      ctx.chat.id,
      "SMUTE",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error) {
    console.error("[v0] SMute error:", error)
  }
}

export async function handleUnmute(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to unmute users. You need 'Restrict Members' permission.",
    )
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to unmute.")
    return
  }

  try {
    await ctx.bot.restrictChatMember(ctx.chat.id, target.userId, {
      can_send_messages: true,
      can_send_audios: true,
      can_send_documents: true,
      can_send_photos: true,
      can_send_videos: true,
      can_send_video_notes: true,
      can_send_voice_notes: true,
      can_send_polls: true,
      can_send_other_messages: true,
      can_add_web_page_previews: true,
      can_change_info: true,
      can_invite_users: true,
      can_pin_messages: true,
    })

    await supabase.from("mutes").delete().eq("chat_id", ctx.chat.id).eq("user_id", target.userId)

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(ctx.chat.id, `${userName} has been unmuted!`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.message.message_id,
    })

    await logAction(ctx.chat.id, "UNMUTE", `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}`)
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to unmute user: ${errorMsg}`)
  }
}

// ==================== KICK COMMANDS ====================

export async function handleKick(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  const canKick = permissions.canRestrictMembers || (await canPerformAction(ctx.chat.id, ctx.user.id, "kick"))

  if (!canKick) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to kick users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to kick.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't kick yourself! Use /kickme instead.")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't kick an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args)

  try {
    await ctx.bot.banChatMember(ctx.chat.id, target.userId)
    await ctx.bot.unbanChatMember(ctx.chat.id, target.userId)

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been kicked!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.message.message_id,
      },
    )

    await logAction(
      ctx.chat.id,
      "KICK",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to kick user: ${errorMsg}`)
  }
}

export async function handleDKick(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to kick users. You need 'Ban Users' permission.")
    return
  }
  if (!permissions.canDeleteMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to delete messages. You need 'Delete Messages' permission.",
    )
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to delete and kick the user.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Could not find target user.")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't kick an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = ctx.args.join(" ") || undefined

  try {
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)
    await ctx.bot.banChatMember(ctx.chat.id, target.userId)
    await ctx.bot.unbanChatMember(ctx.chat.id, target.userId)

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been kicked and their message deleted!${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      { parse_mode: "HTML" },
    )

    await logAction(
      ctx.chat.id,
      "DKICK",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to kick user: ${errorMsg}`)
  }
}

export async function handleSKick(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canRestrictMembers) return

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) return

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) return
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args)

  try {
    // Delete the command message
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.message.message_id)
  } catch {
    // Ignore
  }

  if (ctx.replyToMessage) {
    try {
      await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)
    } catch {
      // Ignore
    }
  }

  try {
    await ctx.bot.banChatMember(ctx.chat.id, target.userId)
    await ctx.bot.unbanChatMember(ctx.chat.id, target.userId)

    await logAction(
      ctx.chat.id,
      "SKICK",
      `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nReason: ${reason || "None"}`,
    )
  } catch (error) {
    console.error("[v0] SKick error:", error)
  }
}

export async function handleKickMe(ctx: CommandContext): Promise<void> {
  try {
    await ctx.bot.kickChatMember(ctx.chat.id, ctx.user.id)
    await ctx.bot.sendMessage(ctx.chat.id, `${mentionUser(ctx.user)} has left the chat.`, { parse_mode: "HTML" })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to kick you: ${errorMsg}`)
  }
}

// ==================== WARNING COMMANDS ====================

export async function handleWarn(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  const canWarn = permissions.canRestrictMembers || (await canPerformAction(ctx.chat.id, ctx.user.id, "warn"))

  if (!canWarn) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't have permission to warn users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to warn.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't warn yourself!")
    return
  }

  try {
    const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
    if (["creator", "administrator"].includes(member.status)) {
      await ctx.bot.sendMessage(ctx.chat.id, "I can't warn an admin!")
      return
    }
  } catch {
    // Continue
  }

  const reason = getReason(ctx.args)

  // Get warning settings
  const { data: settings } = await supabase
    .from("warning_settings")
    .select("warn_limit, warn_mode, warn_time")
    .eq("chat_id", ctx.chat.id)
    .maybeSingle()

  const warnLimit = settings?.warn_limit || 3
  const warnMode = settings?.warn_mode || "mute"
  const warnTime = settings?.warn_time || 0

  // Add warning
  const { error: insertError } = await supabase.from("warnings").insert({
    chat_id: ctx.chat.id,
    user_id: target.userId,
    warned_by: ctx.user.id,
    reason,
  })

  if (insertError) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to add warning.")
    return
  }

  // Count warnings
  const { count } = await supabase
    .from("warnings")
    .select("*", { count: "exact", head: true })
    .eq("chat_id", ctx.chat.id)
    .eq("user_id", target.userId)

  const warnCount = count || 1
  const userName = getUserDisplayName(target)

  if (warnCount >= warnLimit) {
    // Execute action at limit
    try {
      if (warnMode === "ban") {
        await ctx.bot.banChatMember(ctx.chat.id, target.userId)
      } else if (warnMode === "kick") {
        await ctx.bot.banChatMember(ctx.chat.id, target.userId)
        await ctx.bot.unbanChatMember(ctx.chat.id, target.userId)
      } else if (warnMode === "mute") {
        const untilDate = warnTime > 0 ? Math.floor(Date.now() / 1000) + warnTime : undefined
        await ctx.bot.restrictChatMember(
          ctx.chat.id,
          target.userId,
          {
            can_send_messages: false,
            can_send_audios: false,
            can_send_documents: false,
            can_send_photos: false,
            can_send_videos: false,
            can_send_video_notes: false,
            can_send_voice_notes: false,
            can_send_polls: false,
            can_send_other_messages: false,
            can_add_web_page_previews: false,
          },
          untilDate,
        )
      }

      // Clear warnings
      await supabase.from("warnings").delete().eq("chat_id", ctx.chat.id).eq("user_id", target.userId)

      await ctx.bot.sendMessage(
        ctx.chat.id,
        `${userName} has reached ${warnLimit} warnings and has been ${warnMode === "ban" ? "banned" : warnMode === "kick" ? "kicked" : "muted"}!`,
        { parse_mode: "HTML", reply_to_message_id: ctx.message.message_id },
      )
    } catch (error: unknown) {
      const errorMsg = error instanceof Error ? error.message : "Unknown error"
      await ctx.bot.sendMessage(ctx.chat.id, `Failed to execute warn action: ${errorMsg}`)
    }
  } else {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} has been warned! (${warnCount}/${warnLimit})${reason ? `\nReason: ${escapeHtml(reason)}` : ""}`,
      {
        parse_mode: "HTML",
        reply_to_message_id: ctx.message.message_id,
      },
    )
  }

  await logAction(
    ctx.chat.id,
    "WARN",
    `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}\nWarnings: ${warnCount}/${warnLimit}\nReason: ${reason || "None"}`,
  )
}

export async function handleDWarn(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to warn users.")
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to delete and warn the user.")
    return
  }

  // Delete message first
  try {
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)
  } catch {
    // Ignore delete errors
  }

  // Then warn (reuse handleWarn logic)
  const originalMessage = ctx.message
  ctx.message = { ...ctx.message, reply_to_message: ctx.replyToMessage }
  await handleWarn(ctx)
  ctx.message = originalMessage
}

export async function handleWarns(ctx: CommandContext): Promise<void> {
  const target = await getTargetUser(ctx.message, ctx.args)
  const userId = target?.userId || ctx.user.id

  const { data: warnings } = await supabase
    .from("warnings")
    .select("*")
    .eq("chat_id", ctx.chat.id)
    .eq("user_id", userId)
    .order("created_at", { ascending: false })

  if (!warnings || warnings.length === 0) {
    if (target) {
      await ctx.bot.sendMessage(ctx.chat.id, "This user has no warnings.")
    } else {
      await ctx.bot.sendMessage(ctx.chat.id, "You have no warnings in this chat.")
    }
    return
  }

  const { data: settings } = await supabase
    .from("chat_settings")
    .select("warn_limit")
    .eq("chat_id", ctx.chat.id)
    .single()

  const warnLimit = settings?.warn_limit || 3

  let text = target
    ? `<b>Warnings for user ${userId}:</b> ${warnings.length}/${warnLimit}\n\n`
    : `<b>Your warnings:</b> ${warnings.length}/${warnLimit}\n\n`

  warnings.forEach((warn, i) => {
    text += `${i + 1}. ${warn.reason || "No reason"}\n`
  })

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleResetWarns(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to reset warnings.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to reset warnings.")
    return
  }

  const { error } = await supabase.from("warnings").delete().eq("chat_id", ctx.chat.id).eq("user_id", target.userId)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to reset warnings.")
    return
  }

  const userName = getUserDisplayName(target)
  await ctx.bot.sendMessage(ctx.chat.id, `Warnings for ${userName} have been reset!`, { parse_mode: "HTML" })

  await logAction(ctx.chat.id, "RESETWARNS", `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}`)
}

export async function handleSetWarnLimit(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set warn limit.")
    return
  }

  if (ctx.args.length === 0) {
    const { data: settings } = await supabase
      .from("chat_settings")
      .select("warn_limit")
      .eq("chat_id", ctx.chat.id)
      .single()

    await ctx.bot.sendMessage(ctx.chat.id, `Current warn limit: ${settings?.warn_limit || 3}`)
    return
  }

  const limit = Number.parseInt(ctx.args[0])
  if (Number.isNaN(limit) || limit < 1 || limit > 100) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a valid warn limit (1-100).")
    return
  }

  await supabase.from("chat_settings").upsert(
    {
      chat_id: ctx.chat.id,
      warn_limit: limit,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Warn limit has been set to ${limit}.`)
}

export async function handleSetWarnAction(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set warn action.")
    return
  }

  const validActions = ["ban", "kick", "mute"]

  if (ctx.args.length === 0) {
    const { data: settings } = await supabase
      .from("chat_settings")
      .select("warn_action")
      .eq("chat_id", ctx.chat.id)
      .single()

    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Current warn action: ${settings?.warn_action || "ban"}\nAvailable: ${validActions.join(", ")}`,
    )
    return
  }

  const action = ctx.args[0].toLowerCase()
  if (!validActions.includes(action)) {
    await ctx.bot.sendMessage(ctx.chat.id, `Invalid action. Available: ${validActions.join(", ")}`)
    return
  }

  await supabase.from("chat_settings").upsert(
    {
      chat_id: ctx.chat.id,
      warn_action: action,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Warn action has been set to ${action}.`)
}

// ==================== PROMOTE/DEMOTE ====================

export async function handlePromote(ctx: CommandContext): Promise<void> {
  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can promote users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to promote.")
    return
  }

  try {
    await ctx.bot.promoteChatMember(ctx.chat.id, target.userId, {
      can_manage_chat: true,
      can_delete_messages: true,
      can_restrict_members: true,
      can_invite_users: true,
      can_pin_messages: true,
      can_manage_video_chats: true,
    })

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(ctx.chat.id, `${userName} has been promoted to admin!`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.message.message_id,
    })

    await logAction(ctx.chat.id, "PROMOTE", `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}`)
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to promote user: ${errorMsg}`)
  }
}

export async function handleDemote(ctx: CommandContext): Promise<void> {
  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can demote users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to demote.")
    return
  }

  try {
    await ctx.bot.promoteChatMember(ctx.chat.id, target.userId, {
      can_manage_chat: false,
      can_delete_messages: false,
      can_restrict_members: false,
      can_invite_users: false,
      can_pin_messages: false,
      can_manage_video_chats: false,
      can_promote_members: false,
      can_change_info: false,
    })

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(ctx.chat.id, `${userName} has been demoted!`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.message.message_id,
    })

    await logAction(ctx.chat.id, "DEMOTE", `User: ${target.userId}\nBy: ${mentionUser(ctx.user)}`)
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to demote user: ${errorMsg}`)
  }
}

export async function handleAdminList(ctx: CommandContext): Promise<void> {
  try {
    const admins = await ctx.bot.getChatAdministrators(ctx.chat.id)
    // Filter out all bots
    const humanAdmins = admins.filter((a) => !a.user.is_bot)

    if (humanAdmins.length === 0) {
      await ctx.bot.sendMessage(ctx.chat.id, "No human admins found in this chat.")
      return
    }

    let text = `<b>Admins in ${escapeHtml(ctx.chat.title || "this chat")}:</b>\n\n`

    const creator = humanAdmins.find((a: { status: string }) => a.status === "creator")
    if (creator) {
      text += `👑 ${mentionUser(creator.user)} <i>(Creator)</i>\n`
    }

    const adminList = humanAdmins.filter((a: { status: string }) => a.status === "administrator")
    adminList.forEach((admin: { user: TelegramUser; custom_title?: string }) => {
      const title = admin.custom_title ? ` - ${admin.custom_title}` : ""
      text += `⭐ ${mentionUser(admin.user)}${title}\n`
    })

    text += `\n<i>Total: ${humanAdmins.length} admin${humanAdmins.length > 1 ? "s" : ""}</i>`

    await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to get admin list: ${errorMsg}`)
  }
}

export async function handleSetTitle(ctx: CommandContext): Promise<void> {
  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can set admin titles.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to an admin or provide a user ID and title.")
    return
  }

  const titleArgs = ctx.replyToMessage ? ctx.args : ctx.args.slice(1)
  const title = titleArgs.join(" ")

  if (!title) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a title.")
    return
  }

  if (title.length > 16) {
    await ctx.bot.sendMessage(ctx.chat.id, "Title must be 16 characters or less.")
    return
  }

  try {
    await ctx.bot.setChatAdministratorCustomTitle(ctx.chat.id, target.userId, title)

    const userName = getUserDisplayName(target)
    await ctx.bot.sendMessage(ctx.chat.id, `${userName}'s title has been set to: ${escapeHtml(title)}`, {
      parse_mode: "HTML",
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to set title: ${errorMsg}`)
  }
}
