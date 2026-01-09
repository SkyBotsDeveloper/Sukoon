import type { CommandContext } from "../types"
import { supabase, getTargetUser, mentionUser, isUserOwner } from "../utils"

// Cache for silent mod permissions (group-specific)
const modCache = new Map<string, { permissions: Record<string, boolean>; timestamp: number }>()
const MOD_CACHE_TTL = 30000 // 30 seconds

// Check if user can perform a specific action in a specific chat
export async function canPerformAction(
  chatId: number,
  userId: number,
  action: "ban" | "mute" | "kick" | "warn",
): Promise<boolean> {
  const cacheKey = `${chatId}:${userId}`
  const cached = modCache.get(cacheKey)

  if (cached && Date.now() - cached.timestamp < MOD_CACHE_TTL) {
    return cached.permissions[action] || false
  }

  const { data } = await supabase
    .from("silent_mods")
    .select("can_ban, can_mute, can_kick, can_warn")
    .eq("chat_id", String(chatId))
    .eq("user_id", String(userId))
    .maybeSingle()

  if (!data) {
    modCache.set(cacheKey, { permissions: {}, timestamp: Date.now() })
    return false
  }

  const permissions = {
    ban: data.can_ban || false,
    mute: data.can_mute || false,
    kick: data.can_kick || false,
    warn: data.can_warn || false,
  }

  modCache.set(cacheKey, { permissions, timestamp: Date.now() })
  return permissions[action] || false
}

// Invalidate cache for a user in a chat
function invalidateModCache(chatId: number, userId: number): void {
  modCache.delete(`${chatId}:${userId}`)
}

// /mod - Give full silent powers (ban, mute, kick, warn)
export async function handleMod(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  // Only owner can give mod powers
  const isOwner = await isUserOwner(ctx.chat.id, ctx.user.id)
  if (!isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the group owner can give mod powers.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide a user ID/username to make them a mod.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't mod yourself!")
    return
  }

  try {
    await supabase.from("silent_mods").upsert(
      {
        chat_id: String(ctx.chat.id),
        user_id: String(target.userId),
        can_ban: true,
        can_mute: true,
        can_kick: true,
        can_warn: true,
        added_by: ctx.user.id,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id,user_id" },
    )

    invalidateModCache(ctx.chat.id, target.userId)

    const userName = target.user ? mentionUser(target.user) : `User ${target.userId}`
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} now has silent mod powers in this group!\n\nThey can now: ban, mute, kick, and warn users without being a visible admin.`,
      { parse_mode: "HTML" },
    )
  } catch (error) {
    console.error("[v0] Mod error:", error)
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to add mod. Please try again.")
  }
}

// /unmod - Remove all silent powers
export async function handleUnmod(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  const isOwner = await isUserOwner(ctx.chat.id, ctx.user.id)
  if (!isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the group owner can remove mod powers.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide a user ID/username to remove their mod powers.")
    return
  }

  try {
    const { data: existing } = await supabase
      .from("silent_mods")
      .select("id")
      .eq("chat_id", String(ctx.chat.id))
      .eq("user_id", String(target.userId))
      .maybeSingle()

    if (!existing) {
      await ctx.bot.sendMessage(ctx.chat.id, "This user doesn't have any silent powers in this group.")
      return
    }

    await supabase.from("silent_mods").delete().eq("chat_id", String(ctx.chat.id)).eq("user_id", String(target.userId))

    invalidateModCache(ctx.chat.id, target.userId)

    const userName = target.user ? mentionUser(target.user) : `User ${target.userId}`
    await ctx.bot.sendMessage(ctx.chat.id, `${userName}'s silent mod powers have been removed.`, {
      parse_mode: "HTML",
    })
  } catch (error) {
    console.error("[v0] Unmod error:", error)
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to remove mod. Please try again.")
  }
}

// /muter - Give only mute power
export async function handleMuter(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  const isOwner = await isUserOwner(ctx.chat.id, ctx.user.id)
  if (!isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the group owner can give muter powers.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide a user ID/username to make them a muter.")
    return
  }

  if (target.userId === ctx.user.id) {
    await ctx.bot.sendMessage(ctx.chat.id, "You can't give powers to yourself!")
    return
  }

  try {
    // Check if they already have permissions
    const { data: existing } = await supabase
      .from("silent_mods")
      .select("can_ban, can_kick, can_warn")
      .eq("chat_id", String(ctx.chat.id))
      .eq("user_id", String(target.userId))
      .maybeSingle()

    await supabase.from("silent_mods").upsert(
      {
        chat_id: String(ctx.chat.id),
        user_id: String(target.userId),
        can_ban: existing?.can_ban || false,
        can_mute: true,
        can_kick: existing?.can_kick || false,
        can_warn: existing?.can_warn || false,
        added_by: ctx.user.id,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id,user_id" },
    )

    invalidateModCache(ctx.chat.id, target.userId)

    const userName = target.user ? mentionUser(target.user) : `User ${target.userId}`
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `${userName} now has silent mute power in this group!\n\nThey can now mute users without being a visible admin.`,
      { parse_mode: "HTML" },
    )
  } catch (error) {
    console.error("[v0] Muter error:", error)
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to add muter. Please try again.")
  }
}

// /unmuter - Remove only mute power
export async function handleUnmuter(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  const isOwner = await isUserOwner(ctx.chat.id, ctx.user.id)
  if (!isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "Only the group owner can remove muter powers.")
    return
  }

  const target = await getTargetUser(ctx.message, ctx.args)
  if (!target) {
    await ctx.bot.sendMessage(ctx.chat.id, "Reply to a user or provide a user ID/username to remove their mute power.")
    return
  }

  try {
    const { data: existing } = await supabase
      .from("silent_mods")
      .select("can_ban, can_kick, can_warn")
      .eq("chat_id", String(ctx.chat.id))
      .eq("user_id", String(target.userId))
      .maybeSingle()

    if (!existing) {
      await ctx.bot.sendMessage(ctx.chat.id, "This user doesn't have any silent powers in this group.")
      return
    }

    // If they have other powers, just remove mute
    if (existing.can_ban || existing.can_kick || existing.can_warn) {
      await supabase
        .from("silent_mods")
        .update({ can_mute: false, updated_at: new Date().toISOString() })
        .eq("chat_id", String(ctx.chat.id))
        .eq("user_id", String(target.userId))
    } else {
      // No other powers, delete the row
      await supabase
        .from("silent_mods")
        .delete()
        .eq("chat_id", String(ctx.chat.id))
        .eq("user_id", String(target.userId))
    }

    invalidateModCache(ctx.chat.id, target.userId)

    const userName = target.user ? mentionUser(target.user) : `User ${target.userId}`
    await ctx.bot.sendMessage(ctx.chat.id, `${userName}'s silent mute power has been removed.`, {
      parse_mode: "HTML",
    })
  } catch (error) {
    console.error("[v0] Unmuter error:", error)
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to remove muter. Please try again.")
  }
}

// /mods - List all silent mods in this group
export async function handleMods(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command can only be used in groups.")
    return
  }

  try {
    const { data: mods } = await supabase
      .from("silent_mods")
      .select("user_id, can_ban, can_mute, can_kick, can_warn")
      .eq("chat_id", String(ctx.chat.id))

    if (!mods || mods.length === 0) {
      await ctx.bot.sendMessage(ctx.chat.id, "No silent mods in this group.")
      return
    }

    let message = `<b>Silent Mods in this group:</b>\n\n`

    for (const mod of mods) {
      const powers: string[] = []
      if (mod.can_ban) powers.push("ban")
      if (mod.can_mute) powers.push("mute")
      if (mod.can_kick) powers.push("kick")
      if (mod.can_warn) powers.push("warn")

      try {
        const member = await ctx.bot.getChatMember(ctx.chat.id, Number(mod.user_id))
        const name = member.user.first_name + (member.user.last_name ? ` ${member.user.last_name}` : "")
        message += `- <a href="tg://user?id=${mod.user_id}">${name}</a>: ${powers.join(", ")}\n`
      } catch {
        message += `- User ${mod.user_id}: ${powers.join(", ")}\n`
      }
    }

    message += `\n<i>Total: ${mods.length} silent mod(s)</i>`

    await ctx.bot.sendMessage(ctx.chat.id, message, { parse_mode: "HTML" })
  } catch (error) {
    console.error("[v0] Mods list error:", error)
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to get mods list. Please try again.")
  }
}
