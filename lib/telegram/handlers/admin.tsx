import type { CommandContext } from "../types"
import { supabase, getTargetUser, mentionUser, getAdminPermissions, escapeHtml } from "../utils"

// ===========================================
// ADMIN TOOLS
// ===========================================

export async function handlePin(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canPinMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to pin messages. You need 'Pin Messages' permission.",
    )
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to pin it.")
    return
  }

  const silent = ctx.args.includes("silent") || ctx.args.includes("notify")

  try {
    await ctx.bot.pinChatMessage(ctx.chat.id, ctx.replyToMessage.message_id, {
      disable_notification: silent,
    })
  } catch {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to pin message.")
  }
}

export async function handleUnpin(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canPinMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to unpin messages. You need 'Pin Messages' permission.",
    )
    return
  }

  try {
    if (ctx.replyToMessage) {
      await ctx.bot.unpinChatMessage(ctx.chat.id, ctx.replyToMessage.message_id)
    } else {
      await ctx.bot.unpinChatMessage(ctx.chat.id)
    }
    await ctx.bot.sendMessage(ctx.chat.id, "Message unpinned!", {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to unpin message.")
  }
}

export async function handleUnpinAll(ctx: CommandContext): Promise<void> {
  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can unpin all messages.")
    return
  }

  try {
    await ctx.bot.unpinAllChatMessages(ctx.chat.id)
    await ctx.bot.sendMessage(ctx.chat.id, "All messages have been unpinned!", {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to unpin all messages.")
  }
}

export async function handlePurge(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canDeleteMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to delete messages. You need 'Delete Messages' permission.",
    )
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to start purging from there.")
    return
  }

  const startId = ctx.replyToMessage.message_id
  const endId = ctx.message.message_id

  let deleted = 0
  for (let i = startId; i <= endId; i++) {
    try {
      await ctx.bot.deleteMessage(ctx.chat.id, i)
      deleted++
    } catch {
      // Skip if can't delete
    }
  }

  const msg = await ctx.bot.sendMessage(ctx.chat.id, `Purged ${deleted} messages.`)

  setTimeout(async () => {
    try {
      await ctx.bot.deleteMessage(ctx.chat.id, msg.message_id)
    } catch {
      // Ignore
    }
  }, 3000)
}

export async function handleDel(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canDeleteMessages) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to delete messages. You need 'Delete Messages' permission.",
    )
    return
  }

  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to delete it.")
    return
  }

  try {
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.replyToMessage.message_id)
    await ctx.bot.deleteMessage(ctx.chat.id, ctx.message.message_id)
  } catch {
    // Ignore
  }
}

// ===========================================
// APPROVAL COMMANDS
// ===========================================

export async function handleApprove(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to approve users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to approve.")
    return
  }

  await supabase
    .from("approved_users")
    .upsert(
      { chat_id: ctx.chat.id, user_id: target.userId, approved_by: ctx.user.id },
      { onConflict: "chat_id,user_id" },
    )

  await ctx.bot.sendMessage(ctx.chat.id, `User ${target.userId} has been approved and will bypass restrictions.`)
}

export async function handleUnapprove(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to unapprove users.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to unapprove.")
    return
  }

  await supabase.from("approved_users").delete().eq("chat_id", ctx.chat.id).eq("user_id", target.userId)

  await ctx.bot.sendMessage(ctx.chat.id, `User ${target.userId} has been unapproved.`)
}

export async function handleApproved(ctx: CommandContext): Promise<void> {
  const { data: approved } = await supabase.from("approved_users").select("user_id").eq("chat_id", ctx.chat.id)

  if (!approved || approved.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No approved users in this chat.")
    return
  }

  let text = "<b>Approved Users:</b>\n\n"
  approved.forEach((user, i) => {
    text += `${i + 1}. <code>${user.user_id}</code>\n`
  })

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleApproval(ctx: CommandContext): Promise<void> {
  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID/username to check.")
    return
  }

  const { data: approved } = await supabase
    .from("approved_users")
    .select("*")
    .eq("chat_id", ctx.chat.id)
    .eq("user_id", target.userId)
    .maybeSingle()

  if (approved) {
    await ctx.bot.sendMessage(ctx.chat.id, `User <code>${target.userId}</code> is approved in this chat.`, {
      parse_mode: "HTML",
    })
  } else {
    await ctx.bot.sendMessage(ctx.chat.id, `User <code>${target.userId}</code> is not approved in this chat.`, {
      parse_mode: "HTML",
    })
  }
}

// ===========================================
// REPORT COMMANDS
// ===========================================

export async function handleReport(ctx: CommandContext): Promise<void> {
  if (!ctx.replyToMessage) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a message to report it to admins.")
    return
  }

  const { data: settings } = await supabase
    .from("chat_settings")
    .select("reports_enabled")
    .eq("chat_id", ctx.chat.id)
    .single()

  if (settings?.reports_enabled === false) {
    return
  }

  try {
    const admins = await ctx.bot.getChatAdministrators(ctx.chat.id)
    // Filter out bots and create invisible mentions (zero-width space)
    const humanAdmins = admins.filter((a) => !a.user.is_bot)
    // Use invisible character after mention to hide the link text
    const invisibleMentions = humanAdmins.map((a) => `<a href="tg://user?id=${a.user.id}">&#8203;</a>`).join("")

    const reporterName = ctx.user.first_name
    const reportedUser = ctx.replyToMessage.from
    const reportedName = reportedUser ? reportedUser.first_name : "Unknown"

    await ctx.bot.sendMessage(ctx.chat.id, `<b>Reported ${reportedName} to admins.</b>${invisibleMentions}`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.replyToMessage.message_id,
    })
  } catch {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to report.")
  }
}

export async function handleReports(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to toggle reports.")
    return
  }

  const setting = ctx.args[0]?.toLowerCase()

  if (setting === "on" || setting === "yes") {
    await supabase
      .from("chat_settings")
      .upsert({ chat_id: ctx.chat.id, reports_enabled: true }, { onConflict: "chat_id" })
    await ctx.bot.sendMessage(ctx.chat.id, "Reports have been enabled.")
  } else if (setting === "off" || setting === "no") {
    await supabase
      .from("chat_settings")
      .upsert({ chat_id: ctx.chat.id, reports_enabled: false }, { onConflict: "chat_id" })
    await ctx.bot.sendMessage(ctx.chat.id, "Reports have been disabled.")
  } else {
    const { data } = await supabase.from("chat_settings").select("reports_enabled").eq("chat_id", ctx.chat.id).single()
    const status = data?.reports_enabled !== false ? "enabled" : "disabled"
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Reports are currently ${status}. Use /reports on or /reports off to toggle.`,
    )
  }
}

// ===========================================
// CONNECTION COMMANDS
// ===========================================

export async function handleConnect(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type !== "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "Please use this command in private chat.")
    return
  }

  const chatId = ctx.args[0]
  if (!chatId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a chat ID to connect to.")
    return
  }

  await supabase.from("user_connections").upsert({ user_id: ctx.user.id, chat_id: chatId }, { onConflict: "user_id" })

  await ctx.bot.sendMessage(ctx.chat.id, `Connected to chat ${chatId}. You can now manage it from here.`)
}

export async function handleDisconnect(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type !== "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "Please use this command in private chat.")
    return
  }

  await supabase.from("user_connections").delete().eq("user_id", ctx.user.id)

  await ctx.bot.sendMessage(ctx.chat.id, "Disconnected from all chats.")
}

// ===========================================
// DISABLE COMMANDS
// ===========================================

export async function handleDisable(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to disable commands.")
    return
  }

  const command = ctx.args[0]?.toLowerCase()
  if (!command) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please specify a command to disable.")
    return
  }

  const protectedCommands = ["disable", "enable", "disabled"]
  if (protectedCommands.includes(command)) {
    await ctx.bot.sendMessage(ctx.chat.id, "This command cannot be disabled.")
    return
  }

  await supabase.from("disabled_commands").upsert({ chat_id: ctx.chat.id, command }, { onConflict: "chat_id,command" })

  await ctx.bot.sendMessage(ctx.chat.id, `Command /${command} has been disabled.`)
}

export async function handleEnable(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to enable commands.")
    return
  }

  const command = ctx.args[0]?.toLowerCase()
  if (!command) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please specify a command to enable.")
    return
  }

  await supabase.from("disabled_commands").delete().eq("chat_id", ctx.chat.id).eq("command", command)

  await ctx.bot.sendMessage(ctx.chat.id, `Command /${command} has been enabled.`)
}

export async function handleDisabled(ctx: CommandContext): Promise<void> {
  const { data: disabled } = await supabase.from("disabled_commands").select("command").eq("chat_id", ctx.chat.id)

  if (!disabled || disabled.length === 0) {
    await ctx.bot.sendMessage(ctx.chat.id, "No disabled commands in this chat.")
    return
  }

  const list = disabled.map((d) => `/${d.command}`).join(", ")
  await ctx.bot.sendMessage(ctx.chat.id, `<b>Disabled commands:</b>\n${list}`, { parse_mode: "HTML" })
}

// ===========================================
// ADMIN INFO COMMANDS
// ===========================================

export async function handleAdmins(ctx: CommandContext): Promise<void> {
  try {
    const admins = await ctx.bot.getChatAdministrators(ctx.chat.id)
    // Filter out ALL bots - only show human admins
    const humanAdmins = admins.filter((a) => !a.user.is_bot)

    if (humanAdmins.length === 0) {
      await ctx.bot.sendMessage(ctx.chat.id, "No human admins found in this chat.")
      return
    }

    let text = `<b>Admins in ${escapeHtml(ctx.chat.title || "this chat")}:</b>\n\n`

    const creator = humanAdmins.find((a) => a.status === "creator")
    if (creator) {
      text += `👑 ${mentionUser(creator.user)} <i>(Creator)</i>\n`
    }

    const adminList = humanAdmins.filter((a) => a.status === "administrator")
    adminList.forEach((admin) => {
      const title = (admin as { custom_title?: string }).custom_title
        ? ` - ${(admin as { custom_title?: string }).custom_title}`
        : ""
      text += `⭐ ${mentionUser(admin.user)}${title}\n`
    })

    text += `\n<i>Total: ${humanAdmins.length} admin${humanAdmins.length > 1 ? "s" : ""}</i>`

    await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
  } catch {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to get admin list.")
  }
}

export async function handleInfo(ctx: CommandContext): Promise<void> {
  const target = await getTargetUser(ctx.message, ctx.args)
  const userId = target?.userId || ctx.user.id

  let userInfo = target?.user

  // If we only have userId, try to get user info from database or API
  if (!userInfo && target?.userId) {
    const { data: dbUser } = await supabase.from("users").select("*").eq("user_id", target.userId).single()

    if (dbUser) {
      userInfo = {
        id: dbUser.user_id,
        first_name: dbUser.first_name || "Unknown",
        last_name: dbUser.last_name,
        username: dbUser.username,
        is_bot: false,
      }
    } else if (ctx.chat.type !== "private") {
      try {
        const member = await ctx.bot.getChatMember(ctx.chat.id, target.userId)
        userInfo = member.user

        await supabase.from("users").upsert(
          {
            user_id: member.user.id,
            first_name: member.user.first_name,
            last_name: member.user.last_name,
            username: member.user.username,
          },
          { onConflict: "user_id" },
        )
      } catch {
        // User not found in chat
      }
    }
  }

  if (!userInfo && !target?.userId) {
    userInfo = ctx.user
  }

  const firstName = userInfo?.first_name || "Unknown"
  const lastName = userInfo?.last_name || ""
  const username = userInfo?.username
  const displayId = target?.userId || userId

  // Check global ban
  const { data: gban } = await supabase.from("global_bans").select("reason").eq("user_id", displayId).single()

  // Check warnings in this chat
  const { data: warnings } = await supabase
    .from("warnings")
    .select("*")
    .eq("chat_id", ctx.chat.id)
    .eq("user_id", displayId)

  // Check if approved
  const { data: approved } = await supabase
    .from("approved_users")
    .select("*")
    .eq("chat_id", ctx.chat.id)
    .eq("user_id", displayId)
    .single()

  let text = `<b>User Info</b>\n\n`
  text += `<b>ID:</b> <code>${displayId}</code>\n`
  text += `<b>First Name:</b> ${firstName}\n`
  if (lastName) text += `<b>Last Name:</b> ${lastName}\n`
  if (username) text += `<b>Username:</b> @${username}\n`

  if (ctx.chat.type !== "private") {
    text += `\n<b>Warnings:</b> ${warnings?.length || 0}\n`
    if (approved) text += `<b>Approved:</b> Yes\n`
  }

  if (gban) {
    text += `\n⚠️ <b>Globally Banned:</b> ${gban.reason || "No reason"}\n`
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

// ===========================================
// LOG CHANNEL COMMANDS
// ===========================================

export async function handleSetLog(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set the log channel.")
    return
  }

  const channelId = ctx.args[0]
  if (!channelId) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Please provide a channel ID or username.\nUsage: /setlog @channel or /setlog -100123456789",
    )
    return
  }

  try {
    // Try to send a test message to verify the bot is admin in the channel
    const testMsg = await ctx.bot.sendMessage(
      channelId,
      "✅ Log channel set successfully! This channel will now receive logs.",
    )

    await supabase
      .from("chat_settings")
      .upsert({ chat_id: ctx.chat.id, log_channel: channelId }, { onConflict: "chat_id" })

    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Log channel has been set to ${channelId}. All admin actions will be logged there.`,
    )
  } catch (error) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Failed to set log channel. Make sure the bot is an admin in the target channel.",
    )
  }
}

export async function handleUnsetLog(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to unset the log channel.")
    return
  }

  await supabase.from("chat_settings").update({ log_channel: null }).eq("chat_id", ctx.chat.id)

  await ctx.bot.sendMessage(ctx.chat.id, "Log channel has been removed. Actions will no longer be logged.")
}

export async function handleLogChannel(ctx: CommandContext): Promise<void> {
  const { data: settings } = await supabase
    .from("chat_settings")
    .select("log_channel")
    .eq("chat_id", ctx.chat.id)
    .single()

  if (settings?.log_channel) {
    await ctx.bot.sendMessage(ctx.chat.id, `Current log channel: ${settings.log_channel}`)
  } else {
    await ctx.bot.sendMessage(ctx.chat.id, "No log channel is set for this chat.")
  }
}

// ===========================================
// ADMIN MANAGEMENT COMMANDS
// ===========================================

export async function handlePromote(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canPromoteMembers) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to promote users. You need 'Add New Admins' permission.",
    )
    return
  }

  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to promote users.")
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

    const title = ctx.args.filter((a) => !a.startsWith("@") && !/^\d+$/.test(a)).join(" ")
    if (title) {
      try {
        await ctx.bot.setChatAdministratorCustomTitle(ctx.chat.id, target.userId, title.slice(0, 16))
      } catch {
        // Title setting may fail
      }
    }

    await ctx.bot.sendMessage(ctx.chat.id, `User <code>${target.userId}</code> has been promoted to admin!`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to promote user: ${errorMsg}`)
  }
}

export async function handleDemote(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canPromoteMembers) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to demote users. You need 'Add New Admins' permission.",
    )
    return
  }

  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to demote users.")
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
      can_change_info: false,
      can_promote_members: false,
    })

    await ctx.bot.sendMessage(ctx.chat.id, `User <code>${target.userId}</code> has been demoted.`, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to demote user: ${errorMsg}`)
  }
}

export async function handleTitle(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to set titles.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to an admin or provide their user ID/username.")
    return
  }

  const title = ctx.args.filter((a) => !a.startsWith("@") && !/^\d+$/.test(a)).join(" ")
  if (!title) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a custom title (max 16 characters).")
    return
  }

  try {
    await ctx.bot.setChatAdministratorCustomTitle(ctx.chat.id, target.userId, title.slice(0, 16))
    await ctx.bot.sendMessage(ctx.chat.id, `Custom title set to "${title.slice(0, 16)}" for user.`, {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to set title: ${errorMsg}`)
  }
}

export async function handleSetGTitle(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canChangeInfo) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to change group info. You need 'Change Group Info' permission.",
    )
    return
  }

  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can change the group title.")
    return
  }

  const title = ctx.args.join(" ")
  if (!title) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a new group title.")
    return
  }

  try {
    await ctx.bot.request("setChatTitle", { chat_id: ctx.chat.id, title })
    await ctx.bot.sendMessage(ctx.chat.id, `Group title changed to "${title}".`, {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to change title: ${errorMsg}`)
  }
}

export async function handleSetGPic(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canChangeInfo) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to change group photo. You need 'Change Group Info' permission.",
    )
    return
  }

  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can change the group photo.")
    return
  }

  if (!ctx.replyToMessage?.photo) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a photo to set it as the group photo.")
    return
  }

  const photo = ctx.replyToMessage.photo[ctx.replyToMessage.photo.length - 1]

  try {
    await ctx.bot.request("setChatPhoto", { chat_id: ctx.chat.id, photo: photo.file_id })
    await ctx.bot.sendMessage(ctx.chat.id, "Group photo has been updated!", {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to change group photo: ${errorMsg}`)
  }
}

export async function handleDelGPic(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canChangeInfo) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to delete group photo. You need 'Change Group Info' permission.",
    )
    return
  }

  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can delete the group photo.")
    return
  }

  try {
    await ctx.bot.request("deleteChatPhoto", { chat_id: ctx.chat.id })
    await ctx.bot.sendMessage(ctx.chat.id, "Group photo has been deleted!", {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to delete group photo: ${errorMsg}`)
  }
}

export async function handleSetGDesc(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canChangeInfo) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to change group description. You need 'Change Group Info' permission.",
    )
    return
  }

  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can change the group description.")
    return
  }

  const description = ctx.args.join(" ")

  try {
    await ctx.bot.request("setChatDescription", { chat_id: ctx.chat.id, description })
    await ctx.bot.sendMessage(
      ctx.chat.id,
      description ? "Group description has been updated!" : "Group description has been cleared!",
      {
        reply_to_message_id: ctx.message.message_id,
      },
    )
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to change description: ${errorMsg}`)
  }
}

export async function handleSetSticker(ctx: CommandContext): Promise<void> {
  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can change the group sticker set.")
    return
  }

  const stickerSet = ctx.args[0]
  if (!stickerSet) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide the sticker set name.")
    return
  }

  try {
    await ctx.bot.request("setChatStickerSet", { chat_id: ctx.chat.id, sticker_set_name: stickerSet })
    await ctx.bot.sendMessage(ctx.chat.id, `Group sticker set changed to "${stickerSet}".`, {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to set sticker pack: ${errorMsg}`)
  }
}

export async function handleDelSticker(ctx: CommandContext): Promise<void> {
  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the chat owner can delete the group sticker set.")
    return
  }

  try {
    await ctx.bot.request("deleteChatStickerSet", { chat_id: ctx.chat.id })
    await ctx.bot.sendMessage(ctx.chat.id, "Group sticker set has been deleted!", {
      reply_to_message_id: ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to delete sticker pack: ${errorMsg}`)
  }
}

export async function handleInviteLink(ctx: CommandContext): Promise<void> {
  const permissions = await getAdminPermissions(ctx.chat.id, ctx.user.id)
  if (!permissions.canInviteUsers) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "You don't have permission to create invite links. You need 'Invite Users via Link' permission.",
    )
    return
  }

  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to get the invite link.")
    return
  }

  try {
    const result = await ctx.bot.request("exportChatInviteLink", { chat_id: ctx.chat.id })
    await ctx.bot.sendMessage(ctx.chat.id, `Invite link: ${result}`, {
      reply_to_message_id: ctx.replyToMessage?.message_id || ctx.message.message_id,
    })
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : "Unknown error"
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to get invite link: ${errorMsg}`)
  }
}

// ===========================================
// ADMIN CALL COMMAND
// ===========================================

export async function handleAdminCall(ctx: CommandContext): Promise<void> {
  try {
    const admins = await ctx.bot.getChatAdministrators(ctx.chat.id)

    if (admins.length === 0) {
      await ctx.bot.sendMessage(ctx.chat.id, "No admins found in this chat.")
      return
    }

    // Filter out bots and create invisible mentions
    const humanAdmins = admins.filter((admin) => !admin.user.is_bot)

    if (humanAdmins.length === 0) {
      await ctx.bot.sendMessage(ctx.chat.id, "No human admins found in this chat.")
      return
    }

    // Invisible mentions - admins get notified but no names shown
    const invisibleMentions = humanAdmins.map((a) => `<a href="tg://user?id=${a.user.id}">&#8203;</a>`).join("")

    const reason = ctx.args.join(" ")
    const callerName = `<a href="tg://user?id=${ctx.user.id}">${escapeHtml(ctx.user.first_name)}</a>`

    let message = `${callerName} is calling the admins!${invisibleMentions}`
    if (reason) {
      message = `${callerName} is calling the admins!\n<b>Reason:</b> ${escapeHtml(reason)}${invisibleMentions}`
    }

    await ctx.bot.sendMessage(ctx.chat.id, message, {
      parse_mode: "HTML",
      reply_to_message_id: ctx.replyToMessage?.message_id || ctx.message.message_id,
    })
  } catch (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to notify admins.")
  }
}
