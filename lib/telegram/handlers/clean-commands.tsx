import type { CommandContext } from "../types"
import { bot } from "../bot"
import { supabase } from "../utils"

export async function handleCleanCommand(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const type = ctx.args[0]?.toLowerCase()
  if (!type) {
    const { data } = await supabase.from("clean_commands_settings").select("*").eq("chat_id", ctx.chat.id).single()

    let status = "Clean Commands Settings:\n\n"
    if (!data || (!data.clean_all && !data.clean_admin && !data.clean_user && !data.clean_other)) {
      status += "No command cleaning is enabled."
    } else {
      if (data.clean_all) status += "• Cleaning ALL commands\n"
      if (data.clean_admin && !data.clean_all) status += "• Cleaning admin commands\n"
      if (data.clean_user && !data.clean_all) status += "• Cleaning user commands\n"
      if (data.clean_other && !data.clean_all) status += "• Cleaning other bot commands\n"
    }
    status += "\n\nUse /cleancommand <type> to enable cleaning.\nTypes: all, admin, user, other"

    await bot.sendMessage(ctx.chat.id, status, {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const validTypes = ["all", "admin", "user", "other"]
  if (!validTypes.includes(type)) {
    await bot.sendMessage(ctx.chat.id, "❌ Invalid type. Use: all, admin, user, or other.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await supabase.from("clean_commands_settings").upsert(
    {
      chat_id: ctx.chat.id,
      clean_all: type === "all",
      clean_admin: type === "admin" || type === "all",
      clean_user: type === "user" || type === "all",
      clean_other: type === "other" || type === "all",
    },
    { onConflict: "chat_id" },
  )

  await bot.sendMessage(ctx.chat.id, `✅ Now cleaning ${type} commands.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleKeepCommand(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const type = ctx.args[0]?.toLowerCase()
  if (!type || type === "all") {
    await supabase.from("clean_commands_settings").upsert(
      {
        chat_id: ctx.chat.id,
        clean_all: false,
        clean_admin: false,
        clean_user: false,
        clean_other: false,
      },
      { onConflict: "chat_id" },
    )
    await bot.sendMessage(ctx.chat.id, "✅ All command cleaning has been disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const validTypes = ["admin", "user", "other"]
  if (!validTypes.includes(type)) {
    await bot.sendMessage(ctx.chat.id, "❌ Invalid type. Use: all, admin, user, or other.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const columnName = `clean_${type}`
  await supabase
    .from("clean_commands_settings")
    .upsert({ chat_id: ctx.chat.id, [columnName]: false }, { onConflict: "chat_id" })
  await bot.sendMessage(ctx.chat.id, `✅ Stopped cleaning ${type} commands.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleCleanCommandTypes(ctx: CommandContext): Promise<void> {
  const message =
    `<b>Available command types to clean:</b>\n\n` +
    `• <code>all</code>: Delete ALL commands sent to the group.\n` +
    `• <code>admin</code>: Delete admin-only commands (e.g., /ban, /mute, settings changes).\n` +
    `• <code>user</code>: Delete user commands (e.g., /id, /info, /get). These will also be cleaned when admins use them.\n` +
    `• <code>other</code>: Delete commands which aren't recognised as being valid Sukoon commands.\n\n` +
    `<b>Admin commands:</b>\n` +
    `• /cleancommand &lt;type&gt;: Select which command types to delete.\n` +
    `• /keepcommand &lt;type&gt;: Select which command types to stop deleting.\n` +
    `• /cleancommandtypes: List the different command types which can be cleaned.\n\n` +
    `<b>Examples:</b>\n` +
    `• Delete all commands, but still respond to them:\n-> /cleancommand all\n\n` +
    `• Delete all users commands (but still respond), as well as commands for other bots:\n-> /cleancommand user other\n\n` +
    `• Stop deleting all commands:\n-> /keepcommand all`

  await bot.sendMessage(ctx.chat.id, message, {
    parse_mode: "HTML",
    reply_to_message_id: ctx.message.message_id,
  })
}

// Check if a command should be cleaned
export async function shouldCleanCommand(chatId: number, commandType: "admin" | "user" | "other"): Promise<boolean> {
  const { data } = await supabase.from("clean_commands_settings").select("*").eq("chat_id", chatId).single()

  if (!data) return false
  if (data.clean_all) return true
  return data[`clean_${commandType}`] || false
}
