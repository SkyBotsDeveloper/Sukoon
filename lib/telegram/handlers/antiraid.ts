import type { CommandContext } from "../types"
import { supabase } from "../utils"

function parseDuration(duration: string): number {
  const match = duration.match(/^(\d+)([smhdw])$/)
  if (!match) return 21600 // default 6 hours

  const value = Number.parseInt(match[1])
  const unit = match[2]
  const multipliers: Record<string, number> = { s: 1, m: 60, h: 3600, d: 86400, w: 604800 }
  return value * (multipliers[unit] || 1)
}

export async function handleAntiRaid(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const duration = ctx.args[0]

  if (!duration || duration.toLowerCase() === "off" || duration.toLowerCase() === "no") {
    await supabase
      .from("antiraid_settings")
      .upsert({ chat_id: ctx.chat.id, antiraid_enabled: false }, { onConflict: "chat_id" })
    await ctx.bot.sendMessage(ctx.chat.id, "AntiRaid has been disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const seconds = parseDuration(duration)
  await supabase
    .from("antiraid_settings")
    .upsert({ chat_id: ctx.chat.id, antiraid_enabled: true, raid_duration: seconds }, { onConflict: "chat_id" })

  await ctx.bot.sendMessage(
    ctx.chat.id,
    `AntiRaid enabled for ${duration}. All new joins will be temporarily banned.`,
    {
      reply_to_message_id: ctx.message.message_id,
    },
  )
}

export async function handleRaidTime(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (!ctx.args[0]) {
    const { data } = await supabase
      .from("antiraid_settings")
      .select("raid_duration")
      .eq("chat_id", ctx.chat.id)
      .maybeSingle()
    const hours = Math.floor((data?.raid_duration || 21600) / 3600)
    await ctx.bot.sendMessage(ctx.chat.id, `Current raid duration: ${hours} hours`, {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const seconds = parseDuration(ctx.args[0])
  await supabase
    .from("antiraid_settings")
    .upsert({ chat_id: ctx.chat.id, raid_duration: seconds }, { onConflict: "chat_id" })
  await ctx.bot.sendMessage(ctx.chat.id, `Raid duration set to ${ctx.args[0]}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleRaidActionTime(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (!ctx.args[0]) {
    const { data } = await supabase
      .from("antiraid_settings")
      .select("raid_action_time")
      .eq("chat_id", ctx.chat.id)
      .maybeSingle()
    const hours = Math.floor((data?.raid_action_time || 3600) / 3600)
    await ctx.bot.sendMessage(ctx.chat.id, `Current raid action time: ${hours} hour(s)`, {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const seconds = parseDuration(ctx.args[0])
  await supabase
    .from("antiraid_settings")
    .upsert({ chat_id: ctx.chat.id, raid_action_time: seconds }, { onConflict: "chat_id" })
  await ctx.bot.sendMessage(ctx.chat.id, `Raid action time set to ${ctx.args[0]}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleRaidMode(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const mode = ctx.args[0]?.toLowerCase()
  const validModes = ["ban", "kick", "mute", "tban", "tmute"]

  if (!mode) {
    const { data } = await supabase
      .from("antiraid_settings")
      .select("raid_mode")
      .eq("chat_id", ctx.chat.id)
      .maybeSingle()
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `Current raid mode: ${data?.raid_mode || "ban"}\nAvailable modes: ${validModes.join(", ")}`,
      {
        reply_to_message_id: ctx.message.message_id,
      },
    )
    return
  }

  if (!validModes.includes(mode)) {
    await ctx.bot.sendMessage(ctx.chat.id, `Invalid mode. Available modes: ${validModes.join(", ")}`, {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await supabase.from("antiraid_settings").upsert({ chat_id: ctx.chat.id, raid_mode: mode }, { onConflict: "chat_id" })
  await ctx.bot.sendMessage(ctx.chat.id, `Raid mode set to ${mode}.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleAutoAntiRaid(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (
    !ctx.args[0] ||
    ctx.args[0].toLowerCase() === "off" ||
    ctx.args[0].toLowerCase() === "no" ||
    ctx.args[0] === "0"
  ) {
    await supabase
      .from("antiraid_settings")
      .upsert({ chat_id: ctx.chat.id, auto_raid_threshold: 0 }, { onConflict: "chat_id" })
    await ctx.bot.sendMessage(ctx.chat.id, "Automatic AntiRaid has been disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const threshold = Number.parseInt(ctx.args[0])
  if (isNaN(threshold) || threshold < 1) {
    await ctx.bot.sendMessage(ctx.chat.id, "Please provide a valid number (minimum 1).", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await supabase
    .from("antiraid_settings")
    .upsert({ chat_id: ctx.chat.id, auto_raid_threshold: threshold }, { onConflict: "chat_id" })
  await ctx.bot.sendMessage(ctx.chat.id, `Automatic AntiRaid set to activate at ${threshold} joins per minute.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// Check if antiraid is active for a chat
export async function checkAntiRaid(chatId: number): Promise<boolean> {
  const { data } = await supabase
    .from("antiraid_settings")
    .select("antiraid_enabled")
    .eq("chat_id", chatId)
    .maybeSingle()
  return data?.antiraid_enabled || false
}

// Get raid action time for temporary bans during raid
export async function getRaidActionTime(chatId: number): Promise<number> {
  const { data } = await supabase
    .from("antiraid_settings")
    .select("raid_action_time")
    .eq("chat_id", chatId)
    .maybeSingle()
  return data?.raid_action_time || 3600
}
