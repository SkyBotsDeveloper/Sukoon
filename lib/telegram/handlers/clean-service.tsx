import type { CommandContext } from "../types"
import { bot } from "../bot"
import { supabase } from "../utils"

export async function handleCleanService(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const type = ctx.args[0]?.toLowerCase()
  if (!type) {
    const { data } = await supabase.from("clean_service_settings").select("*").eq("chat_id", ctx.chat.id).single()

    let status = "Clean Service Settings:\n\n"
    if (!data || !data.clean_service_enabled) {
      status += "Service message cleaning is disabled."
    } else {
      if (data.clean_all) status += "• Cleaning ALL service messages\n"
      else {
        if (data.clean_join) status += "• Cleaning join messages\n"
        if (data.clean_leave) status += "• Cleaning leave messages\n"
        if (data.clean_pin) status += "• Cleaning pin messages\n"
        if (data.clean_title) status += "• Cleaning title change messages\n"
        if (data.clean_photo) status += "• Cleaning photo change messages\n"
        if (data.clean_video_chat) status += "• Cleaning video chat messages\n"
        if (data.clean_other) status += "• Cleaning other service messages\n"
      }
    }
    status +=
      "\n\nUse /cleanservice <type> to enable cleaning.\nTypes: all, join, leave, pin, title, photo, videochat, other"

    await bot.sendMessage(ctx.chat.id, status, {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  if (type === "off" || type === "no") {
    await supabase
      .from("clean_service_settings")
      .upsert({ chat_id: ctx.chat.id, clean_service_enabled: false }, { onConflict: "chat_id" })
    await bot.sendMessage(ctx.chat.id, "✅ Service message cleaning disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const validTypes = ["all", "join", "leave", "pin", "title", "photo", "videochat", "other"]
  if (!validTypes.includes(type)) {
    await bot.sendMessage(
      ctx.chat.id,
      "❌ Invalid type. Use: all, join, leave, pin, title, photo, videochat, or other.",
      { reply_to_message_id: ctx.message.message_id },
    )
    return
  }

  await supabase.from("clean_service_settings").upsert(
    {
      chat_id: ctx.chat.id,
      clean_service_enabled: true,
      clean_all: type === "all",
      clean_join: type === "join" || type === "all",
      clean_leave: type === "leave" || type === "all",
      clean_pin: type === "pin" || type === "all",
      clean_title: type === "title" || type === "all",
      clean_photo: type === "photo" || type === "all",
      clean_video_chat: type === "videochat" || type === "all",
      clean_other: type === "other" || type === "all",
    },
    { onConflict: "chat_id" },
  )

  await bot.sendMessage(ctx.chat.id, `✅ Now cleaning ${type} service messages.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleKeepService(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const type = ctx.args[0]?.toLowerCase()
  if (!type || type === "all") {
    await supabase.from("clean_service_settings").upsert(
      {
        chat_id: ctx.chat.id,
        clean_service_enabled: false,
        clean_all: false,
        clean_join: false,
        clean_leave: false,
        clean_pin: false,
        clean_title: false,
        clean_photo: false,
        clean_video_chat: false,
        clean_other: false,
      },
      { onConflict: "chat_id" },
    )
    await bot.sendMessage(ctx.chat.id, "✅ All service message cleaning has been disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const validTypes = ["join", "leave", "pin", "title", "photo", "videochat", "other"]
  if (!validTypes.includes(type)) {
    await bot.sendMessage(ctx.chat.id, "❌ Invalid type.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const columnName = type === "videochat" ? "clean_video_chat" : `clean_${type}`
  await supabase
    .from("clean_service_settings")
    .upsert({ chat_id: ctx.chat.id, [columnName]: false }, { onConflict: "chat_id" })
  await bot.sendMessage(ctx.chat.id, `✅ Stopped cleaning ${type} service messages.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

export async function handleCleanServiceTypes(ctx: CommandContext): Promise<void> {
  const message =
    `<b>Available service message types to clean:</b>\n\n` +
    `• <code>all</code>: All service messages\n` +
    `• <code>join</code>: "X joined the group" messages\n` +
    `• <code>leave</code>: "X left the group" messages\n` +
    `• <code>pin</code>: "X pinned a message" messages\n` +
    `• <code>title</code>: "X changed the group title" messages\n` +
    `• <code>photo</code>: "X changed the group photo" messages\n` +
    `• <code>videochat</code>: Video chat started/ended messages\n` +
    `• <code>other</code>: Other miscellaneous service messages\n\n` +
    `<b>Admin commands:</b>\n` +
    `• /cleanservice &lt;type&gt;: Select which service messages to delete.\n` +
    `• /cleanservice off: Disable service message cleaning.\n` +
    `• /nocleanservice &lt;type&gt;: Select which service messages to keep.\n` +
    `• /cleanservicetypes: List the different service types which can be cleaned.`

  await bot.sendMessage(ctx.chat.id, message, {
    parse_mode: "HTML",
    reply_to_message_id: ctx.message.message_id,
  })
}

// Check if a service message should be cleaned
export async function shouldCleanService(
  chatId: number,
  serviceType: "join" | "leave" | "pin" | "title" | "photo" | "video_chat" | "other",
): Promise<boolean> {
  const { data } = await supabase.from("clean_service_settings").select("*").eq("chat_id", chatId).single()

  if (!data || !data.clean_service_enabled) return false
  if (data.clean_all) return true
  return data[`clean_${serviceType}`] || false
}
