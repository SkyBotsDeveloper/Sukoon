import type { CommandContext } from "../types"
import { supabase, getTargetUser, mentionUser } from "../utils"

// ===========================================
// FEDERATION COMMANDS
// ===========================================

export async function handleNewFed(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type !== "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "Please create federations in private chat with me.")
    return
  }

  const fedName = ctx.args.join(" ")
  if (!fedName) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a name for your federation.\nUsage: /newfed <name>")
    return
  }

  // Check if user already owns a federation
  const { data: existing } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (existing) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `You already own a federation: <b>${existing.fed_name}</b>\n\nDelete it first with /delfed before creating a new one.`,
      { parse_mode: "HTML" },
    )
    return
  }

  // Create new federation
  const { data: newFed, error } = await supabase
    .from("federations")
    .insert({
      fed_name: fedName,
      owner_id: ctx.user.id,
    })
    .select()
    .single()

  if (error) {
    console.error("[v0] Error creating federation:", error)
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to create federation: ${error.message}`)
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Federation Created!</b>\n\n` +
      `Name: <b>${fedName}</b>\n` +
      `ID: <code>${newFed.fed_id}</code>\n\n` +
      `Use this ID to have groups join your federation with:\n` +
      `<code>/joinfed ${newFed.fed_id}</code>`,
    { parse_mode: "HTML" },
  )
}

export async function handleDelFed(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type !== "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "Please delete federations in private chat with me.")
    return
  }

  // Get user's federation
  const { data: fed } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't own any federation.")
    return
  }

  // Delete all related data
  await supabase.from("fed_bans").delete().eq("fed_id", fed.fed_id)
  await supabase.from("fed_admins").delete().eq("fed_id", fed.fed_id)
  await supabase.from("fed_chats").delete().eq("fed_id", fed.fed_id)
  await supabase.from("federations").delete().eq("fed_id", fed.fed_id)

  await ctx.bot.sendMessage(ctx.chat.id, `Federation <b>${fed.fed_name}</b> has been deleted.`, { parse_mode: "HTML" })
}

export async function handleJoinFed(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the group owner can join a federation.")
    return
  }

  const fedId = ctx.args[0]
  if (!fedId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a federation ID.\nUsage: /joinfed <fed_id>")
    return
  }

  // Validate UUID format
  const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
  if (!uuidPattern.test(fedId)) {
    await ctx.bot.sendMessage(ctx.chat.id, "Invalid federation ID format.")
    return
  }

  // Check if federation exists
  const { data: fed } = await supabase.from("federations").select("*").eq("fed_id", fedId).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "Federation not found.")
    return
  }

  // Check if already in a federation
  const { data: existingFed } = await supabase
    .from("fed_chats")
    .select("*, federations(*)")
    .eq("chat_id", ctx.chat.id)
    .single()

  if (existingFed) {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `This chat is already in federation: <b>${(existingFed.federations as any)?.fed_name || "Unknown"}</b>\n\nUse /leavefed first to leave.`,
      { parse_mode: "HTML" },
    )
    return
  }

  // Join federation
  const { error } = await supabase.from("fed_chats").insert({
    fed_id: fedId,
    chat_id: ctx.chat.id,
  })

  if (error) {
    console.error("[v0] Error joining federation:", error)
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to join federation: ${error.message}`)
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `This chat has joined federation: <b>${fed.fed_name}</b>\n\nFederation bans will now apply to this chat.`,
    { parse_mode: "HTML" },
  )
}

export async function handleLeaveFed(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  if (!ctx.isOwner && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the group owner can leave a federation.")
    return
  }

  // Check if in a federation - use explicit type casting for chat_id
  const chatId = ctx.chat.id
  console.log("[v0] Checking federation for chat_id:", chatId)

  const { data: fedChat, error: fetchError } = await supabase
    .from("fed_chats")
    .select("*, federations(*)")
    .eq("chat_id", chatId)
    .maybeSingle()

  console.log("[v0] Fed chat lookup result:", fedChat, "Error:", fetchError)

  if (!fedChat) {
    await ctx.bot.sendMessage(ctx.chat.id, "This chat is not in any federation.")
    return
  }

  // Delete the federation membership
  const { error: deleteError } = await supabase.from("fed_chats").delete().eq("chat_id", chatId)

  if (deleteError) {
    console.error("[v0] Error leaving federation:", deleteError)
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to leave federation: ${deleteError.message}`)
    return
  }

  const fedName = (fedChat.federations as any)?.fed_name || "Unknown"
  await ctx.bot.sendMessage(ctx.chat.id, `This chat has left federation: <b>${fedName}</b>`, { parse_mode: "HTML" })
}

export async function handleChatFed(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  // Get federation for this chat
  const chatId = ctx.chat.id
  const { data: fedChat } = await supabase
    .from("fed_chats")
    .select("*, federations(*)")
    .eq("chat_id", chatId)
    .maybeSingle()

  if (!fedChat) {
    await ctx.bot.sendMessage(ctx.chat.id, "This chat is not in any federation.")
    return
  }

  const fed = fedChat.federations as any

  // Get owner info
  const { data: ownerData } = await supabase
    .from("users")
    .select("first_name, username")
    .eq("user_id", fed.owner_id)
    .single()

  const ownerName = ownerData?.first_name || "Unknown"
  const ownerUsername = ownerData?.username ? `@${ownerData.username}` : ""

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Federation Info for this Chat</b>\n\n` +
      `Name: <b>${fed.fed_name}</b>\n` +
      `ID: <code>${fed.fed_id}</code>\n` +
      `Owner: ${ownerName} ${ownerUsername} (<code>${fed.owner_id}</code>)`,
    { parse_mode: "HTML" },
  )
}

export async function handleFedInfo(ctx: CommandContext): Promise<void> {
  let fedId = ctx.args[0]

  // If no fed ID provided, check current chat's federation
  if (!fedId && ctx.chat.type !== "private") {
    const { data: fedChat } = await supabase.from("fed_chats").select("fed_id").eq("chat_id", ctx.chat.id).maybeSingle()

    if (fedChat) {
      fedId = fedChat.fed_id
    }
  }

  if (!fedId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a federation ID or use this in a federated group.")
    return
  }

  const { data: fed } = await supabase.from("federations").select("*").eq("fed_id", fedId).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "Federation not found.")
    return
  }

  // Get counts
  const { count: chatsCount } = await supabase
    .from("fed_chats")
    .select("*", { count: "exact", head: true })
    .eq("fed_id", fedId)

  const { count: bansCount } = await supabase
    .from("fed_bans")
    .select("*", { count: "exact", head: true })
    .eq("fed_id", fedId)

  const { count: adminsCount } = await supabase
    .from("fed_admins")
    .select("*", { count: "exact", head: true })
    .eq("fed_id", fedId)

  // Get owner info
  const { data: ownerData } = await supabase
    .from("users")
    .select("first_name, username")
    .eq("user_id", fed.owner_id)
    .single()

  const ownerName = ownerData?.first_name || "Unknown"

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Federation Info</b>\n\n` +
      `Name: <b>${fed.fed_name}</b>\n` +
      `ID: <code>${fed.fed_id}</code>\n` +
      `Owner: ${ownerName} (<code>${fed.owner_id}</code>)\n\n` +
      `Chats: ${chatsCount || 0}\n` +
      `Bans: ${bansCount || 0}\n` +
      `Admins: ${(adminsCount || 0) + 1}`,
    { parse_mode: "HTML" },
  )
}

export async function handleFBan(ctx: CommandContext): Promise<void> {
  // Check if user is fed owner or admin
  const { data: fedChat } = await supabase.from("fed_chats").select("fed_id").eq("chat_id", ctx.chat.id).maybeSingle()

  let fedId: string | null = null

  if (fedChat) {
    fedId = fedChat.fed_id
  } else if (ctx.chat.type === "private") {
    // Check if user owns a fed
    const { data: ownedFed } = await supabase.from("federations").select("fed_id").eq("owner_id", ctx.user.id).single()
    if (ownedFed) {
      fedId = ownedFed.fed_id
    }
  }

  if (!fedId) {
    await ctx.bot.sendMessage(ctx.chat.id, "This chat is not part of any federation.")
    return
  }

  // Check permissions
  const { data: fed } = await supabase.from("federations").select("*").eq("fed_id", fedId).single()
  const { data: fedAdmin } = await supabase
    .from("fed_admins")
    .select("*")
    .eq("fed_id", fedId)
    .eq("user_id", ctx.user.id)
    .maybeSingle()

  if (fed?.owner_id !== ctx.user.id && !fedAdmin && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "You must be a federation owner or admin to use this command.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to ban.")
    return
  }

  const reason = ctx.args.slice(1).join(" ") || "No reason"

  // Add to fed bans
  const { error } = await supabase.from("fed_bans").upsert(
    {
      fed_id: fedId,
      user_id: target.userId,
      reason,
      banned_by: ctx.user.id,
    },
    { onConflict: "fed_id,user_id" },
  )

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to federation ban: ${error.message}`)
    return
  }

  // Ban in all fed chats
  const { data: fedChats } = await supabase.from("fed_chats").select("chat_id").eq("fed_id", fedId)

  let bannedCount = 0
  if (fedChats) {
    for (const chat of fedChats) {
      try {
        await ctx.bot.banChatMember(chat.chat_id, target.userId)
        bannedCount++
      } catch {
        // Chat might be gone or bot lacks permissions
      }
    }
  }

  const userName = target.user?.first_name || `User ${target.userId}`
  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>New Federation Ban</b>\n\n` +
      `User: ${mentionUser(target.userId, userName)}\n` +
      `Federation: <b>${fed?.fed_name}</b>\n` +
      `Reason: ${reason}\n` +
      `Banned in ${bannedCount} chat(s)`,
    { parse_mode: "HTML" },
  )
}

export async function handleUnFBan(ctx: CommandContext): Promise<void> {
  // Get federation
  const { data: fedChat } = await supabase.from("fed_chats").select("fed_id").eq("chat_id", ctx.chat.id).maybeSingle()

  let fedId: string | null = null

  if (fedChat) {
    fedId = fedChat.fed_id
  } else if (ctx.chat.type === "private") {
    const { data: ownedFed } = await supabase.from("federations").select("fed_id").eq("owner_id", ctx.user.id).single()
    if (ownedFed) {
      fedId = ownedFed.fed_id
    }
  }

  if (!fedId) {
    await ctx.bot.sendMessage(ctx.chat.id, "This chat is not part of any federation.")
    return
  }

  // Check permissions
  const { data: fed } = await supabase.from("federations").select("*").eq("fed_id", fedId).single()
  const { data: fedAdmin } = await supabase
    .from("fed_admins")
    .select("*")
    .eq("fed_id", fedId)
    .eq("user_id", ctx.user.id)
    .maybeSingle()

  if (fed?.owner_id !== ctx.user.id && !fedAdmin && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "You must be a federation owner or admin to use this command.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to unban.")
    return
  }

  // Remove from fed bans
  const { error } = await supabase.from("fed_bans").delete().eq("fed_id", fedId).eq("user_id", target.userId)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to remove federation ban: ${error.message}`)
    return
  }

  // Unban in all fed chats
  const { data: fedChats } = await supabase.from("fed_chats").select("chat_id").eq("fed_id", fedId)

  if (fedChats) {
    for (const chat of fedChats) {
      try {
        await ctx.bot.unbanChatMember(chat.chat_id, target.userId)
      } catch {
        // Ignore errors
      }
    }
  }

  const userName = target.user?.first_name || `User ${target.userId}`
  await ctx.bot.sendMessage(
    ctx.chat.id,
    `User ${mentionUser(target.userId, userName)} has been removed from the federation ban list.`,
    { parse_mode: "HTML" },
  )
}

export async function handleFedAdmins(ctx: CommandContext): Promise<void> {
  let fedId = ctx.args[0]

  if (!fedId && ctx.chat.type !== "private") {
    const { data: fedChat } = await supabase.from("fed_chats").select("fed_id").eq("chat_id", ctx.chat.id).maybeSingle()

    if (fedChat) {
      fedId = fedChat.fed_id
    }
  }

  if (!fedId) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a federation ID or use this in a federated group.")
    return
  }

  const { data: fed } = await supabase.from("federations").select("*").eq("fed_id", fedId).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "Federation not found.")
    return
  }

  const { data: admins } = await supabase.from("fed_admins").select("user_id").eq("fed_id", fedId)

  // Get owner info
  const { data: ownerData } = await supabase
    .from("users")
    .select("first_name, username")
    .eq("user_id", fed.owner_id)
    .single()

  let text = `<b>Admins in ${fed.fed_name}</b>\n\n`
  text += `👑 Owner: ${ownerData?.first_name || "Unknown"} (<code>${fed.owner_id}</code>)\n\n`

  if (admins && admins.length > 0) {
    text += "<b>Admins:</b>\n"
    for (const admin of admins) {
      const { data: userData } = await supabase.from("users").select("first_name").eq("user_id", admin.user_id).single()
      text += `• ${userData?.first_name || "Unknown"} (<code>${admin.user_id}</code>)\n`
    }
  } else {
    text += "No admins."
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleFedPromote(ctx: CommandContext): Promise<void> {
  // Get user's federation
  const { data: fed } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (!fed && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't own any federation.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to promote.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't promote yourself.")
    return
  }

  // Check if already admin
  const { data: existing } = await supabase
    .from("fed_admins")
    .select("*")
    .eq("fed_id", fed!.fed_id)
    .eq("user_id", target.userId)
    .maybeSingle()

  if (existing) {
    await ctx.bot.sendMessage(ctx.chat.id, "This user is already a federation admin.")
    return
  }

  // Add as admin
  const { error } = await supabase.from("fed_admins").insert({
    fed_id: fed!.fed_id,
    user_id: target.userId,
    added_by: ctx.user.id,
  })

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to promote: ${error.message}`)
    return
  }

  const userName = target.user?.first_name || `User ${target.userId}`
  await ctx.bot.sendMessage(
    ctx.chat.id,
    `${mentionUser(target.userId, userName)} has been promoted to federation admin in <b>${fed!.fed_name}</b>.`,
    { parse_mode: "HTML" },
  )
}

export async function handleFedDemote(ctx: CommandContext): Promise<void> {
  const { data: fed } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (!fed && !ctx.isSudoer) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't own any federation.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to demote.")
    return
  }

  const { error } = await supabase.from("fed_admins").delete().eq("fed_id", fed!.fed_id).eq("user_id", target.userId)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to demote: ${error.message}`)
    return
  }

  const userName = target.user?.first_name || `User ${target.userId}`
  await ctx.bot.sendMessage(
    ctx.chat.id,
    `${mentionUser(target.userId, userName)} has been demoted from federation admin.`,
    { parse_mode: "HTML" },
  )
}

export async function handleRenameFed(ctx: CommandContext): Promise<void> {
  const newName = ctx.args.join(" ")
  if (!newName) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a new name.\nUsage: /renamefed <new name>")
    return
  }

  const { data: fed } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't own any federation.")
    return
  }

  const oldName = fed.fed_name

  const { error } = await supabase.from("federations").update({ fed_name: newName }).eq("fed_id", fed.fed_id)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to rename: ${error.message}`)
    return
  }

  await ctx.bot.sendMessage(ctx.chat.id, `Federation renamed from <b>${oldName}</b> to <b>${newName}</b>.`, {
    parse_mode: "HTML",
  })
}

export async function handleFedTransfer(ctx: CommandContext): Promise<void> {
  const { data: fed } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't own any federation.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please reply to a user or provide a user ID to transfer ownership.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You already own this federation.")
    return
  }

  // Check if target already owns a federation
  const { data: targetFed } = await supabase.from("federations").select("*").eq("owner_id", target.userId).single()

  if (targetFed) {
    await ctx.bot.sendMessage(ctx.chat.id, "This user already owns a federation.")
    return
  }

  const { error } = await supabase.from("federations").update({ owner_id: target.userId }).eq("fed_id", fed.fed_id)

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, `Failed to transfer: ${error.message}`)
    return
  }

  const userName = target.user?.first_name || `User ${target.userId}`
  await ctx.bot.sendMessage(
    ctx.chat.id,
    `Federation <b>${fed.fed_name}</b> has been transferred to ${mentionUser(target.userId, userName)}.`,
    { parse_mode: "HTML" },
  )
}

export async function handleMyFeds(ctx: CommandContext): Promise<void> {
  // Get owned federations
  const { data: ownedFeds } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id)

  // Get federations where user is admin
  const { data: adminFeds } = await supabase
    .from("fed_admins")
    .select("fed_id, federations(*)")
    .eq("user_id", ctx.user.id)

  let text = "<b>Your Federations</b>\n\n"

  if (ownedFeds && ownedFeds.length > 0) {
    text += "<b>Owned:</b>\n"
    for (const fed of ownedFeds) {
      text += `• ${fed.fed_name} (<code>${fed.fed_id}</code>)\n`
    }
    text += "\n"
  }

  if (adminFeds && adminFeds.length > 0) {
    text += "<b>Admin in:</b>\n"
    for (const entry of adminFeds) {
      const fed = entry.federations as any
      if (fed) {
        text += `• ${fed.fed_name} (<code>${fed.fed_id}</code>)\n`
      }
    }
  }

  if ((!ownedFeds || ownedFeds.length === 0) && (!adminFeds || adminFeds.length === 0)) {
    text = "You're not part of any federations."
  }

  await ctx.bot.sendMessage(ctx.chat.id, text, { parse_mode: "HTML" })
}

export async function handleFedStat(ctx: CommandContext): Promise<void> {
  const target = await getTargetUser(ctx.message, ctx.args)
  const userId = target?.userId || ctx.user.id

  // Count fed bans
  const { count: banCount } = await supabase
    .from("fed_bans")
    .select("*", { count: "exact", head: true })
    .eq("user_id", userId)

  const { data: userData } = await supabase.from("users").select("first_name").eq("user_id", userId).single()

  const userName = userData?.first_name || `User ${userId}`

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `<b>Federation Stats for ${userName}</b>\n\n` + `Banned in: ${banCount || 0} federation(s)`,
    { parse_mode: "HTML" },
  )
}

export async function handleQuietFed(ctx: CommandContext): Promise<void> {
  const { data: fed } = await supabase.from("federations").select("*").eq("owner_id", ctx.user.id).single()

  if (!fed) {
    await ctx.bot.sendMessage(ctx.chat.id, "You don't own any federation.")
    return
  }

  const setting = ctx.args[0]?.toLowerCase()

  if (!setting || !["on", "off"].includes(setting)) {
    await ctx.bot.sendMessage(ctx.chat.id, "Usage: /quietfed <on/off>")
    return
  }

  const quiet = setting === "on"

  await supabase.from("fed_settings").upsert(
    {
      fed_id: fed.fed_id,
      quiet_fed: quiet,
    },
    { onConflict: "fed_id" },
  )

  await ctx.bot.sendMessage(ctx.chat.id, `Quiet mode ${quiet ? "enabled" : "disabled"} for <b>${fed.fed_name}</b>.`, {
    parse_mode: "HTML",
  })
}

// Check if user is fed banned when joining
export async function checkFedBan(chatId: number, userId: number): Promise<boolean> {
  const { data: fedChat } = await supabase.from("fed_chats").select("fed_id").eq("chat_id", chatId).maybeSingle()

  if (!fedChat) return false

  const { data: ban } = await supabase
    .from("fed_bans")
    .select("*")
    .eq("fed_id", fedChat.fed_id)
    .eq("user_id", userId)
    .maybeSingle()

  return !!ban
}
