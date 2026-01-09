import type { CommandContext, TelegramUser, TelegramChat } from "../types"
import { bot } from "../bot"
import { supabase, parseTime } from "../utils"

// CAPTCHA types
type CaptchaMode = "button" | "math" | "text"

interface CaptchaChallenge {
  chatId: number
  userId: number
  answer: string
  expiresAt: number
  messageId?: number
}

// In-memory store for pending captchas
const pendingCaptchas = new Map<string, CaptchaChallenge>()

// Generate a random math problem
function generateMathCaptcha(): { question: string; answer: string } {
  const a = Math.floor(Math.random() * 10) + 1
  const b = Math.floor(Math.random() * 10) + 1
  const ops = ["+", "-", "*"]
  const op = ops[Math.floor(Math.random() * ops.length)]

  let answer: number
  switch (op) {
    case "+":
      answer = a + b
      break
    case "-":
      answer = a - b
      break
    case "*":
      answer = a * b
      break
    default:
      answer = a + b
  }

  return {
    question: `What is ${a} ${op} ${b}?`,
    answer: answer.toString(),
  }
}

// Generate random text captcha
function generateTextCaptcha(): { text: string; answer: string } {
  const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
  let code = ""
  for (let i = 0; i < 5; i++) {
    code += chars[Math.floor(Math.random() * chars.length)]
  }
  return { text: code, answer: code }
}

// Handle CAPTCHA command
export async function handleCaptcha(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to configure CAPTCHA.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const setting = ctx.args[0]?.toLowerCase()

  if (!setting) {
    // Show current status
    const { data } = await supabase.from("captcha_settings").select("*").eq("chat_id", ctx.chat.id).single()

    const status = data?.captcha_enabled ? "enabled" : "disabled"
    const mode = data?.captcha_mode || "button"
    const muteTime = data?.mute_time || 300
    const kickOnFail = data?.kick_on_fail ? "yes" : "no"

    await bot.sendMessage(
      ctx.chat.id,
      `<b>CAPTCHA Settings</b>\n\n` +
        `Status: <code>${status}</code>\n` +
        `Mode: <code>${mode}</code>\n` +
        `Mute time: <code>${muteTime}s</code>\n` +
        `Kick on fail: <code>${kickOnFail}</code>\n\n` +
        `Use /captcha on or /captcha off to toggle.`,
      { parse_mode: "HTML", reply_to_message_id: ctx.message.message_id },
    )
    return
  }

  if (setting === "on" || setting === "yes" || setting === "true") {
    await supabase.from("captcha_settings").upsert(
      {
        chat_id: ctx.chat.id,
        captcha_enabled: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await bot.sendMessage(ctx.chat.id, "CAPTCHA has been enabled! New members will need to verify.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else if (setting === "off" || setting === "no" || setting === "false") {
    await supabase.from("captcha_settings").upsert(
      {
        chat_id: ctx.chat.id,
        captcha_enabled: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await bot.sendMessage(ctx.chat.id, "CAPTCHA has been disabled.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else {
    await bot.sendMessage(ctx.chat.id, "Usage: /captcha <on/off>", {
      reply_to_message_id: ctx.message.message_id,
    })
  }
}

// Handle CAPTCHA mode command
export async function handleCaptchaMode(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to configure CAPTCHA.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const mode = ctx.args[0]?.toLowerCase() as CaptchaMode

  if (!mode || !["button", "math", "text"].includes(mode)) {
    await bot.sendMessage(
      ctx.chat.id,
      "Please specify a valid mode: button, math, or text\n\n" +
        "• button: User clicks a button to verify\n" +
        "• math: User solves a math problem\n" +
        "• text: User types a code",
      { reply_to_message_id: ctx.message.message_id },
    )
    return
  }

  await supabase.from("captcha_settings").upsert(
    {
      chat_id: ctx.chat.id,
      captcha_mode: mode,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await bot.sendMessage(ctx.chat.id, `CAPTCHA mode set to: ${mode}`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// Handle CAPTCHA mute time
export async function handleCaptchaMuteTime(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to configure CAPTCHA.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const timeArg = ctx.args[0]
  if (!timeArg) {
    await bot.sendMessage(ctx.chat.id, "Please provide a time (e.g., 5m, 1h).", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const seconds = parseTime(timeArg)
  if (!seconds || seconds < 60 || seconds > 86400) {
    await bot.sendMessage(ctx.chat.id, "Please provide a time between 1 minute and 24 hours.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  await supabase.from("captcha_settings").upsert(
    {
      chat_id: ctx.chat.id,
      mute_time: seconds,
      updated_at: new Date().toISOString(),
    },
    { onConflict: "chat_id" },
  )

  await bot.sendMessage(ctx.chat.id, `CAPTCHA mute time set to ${timeArg}.`, {
    reply_to_message_id: ctx.message.message_id,
  })
}

// Handle CAPTCHA kick setting
export async function handleCaptchaKick(ctx: CommandContext): Promise<void> {
  if (!ctx.isAdmin) {
    await bot.sendMessage(ctx.chat.id, "You need to be an admin to configure CAPTCHA.", {
      reply_to_message_id: ctx.message.message_id,
    })
    return
  }

  const setting = ctx.args[0]?.toLowerCase()

  if (setting === "on" || setting === "yes") {
    await supabase.from("captcha_settings").upsert(
      {
        chat_id: ctx.chat.id,
        kick_on_fail: true,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await bot.sendMessage(ctx.chat.id, "Users who fail CAPTCHA will now be kicked.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else if (setting === "off" || setting === "no") {
    await supabase.from("captcha_settings").upsert(
      {
        chat_id: ctx.chat.id,
        kick_on_fail: false,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )
    await bot.sendMessage(ctx.chat.id, "Users who fail CAPTCHA will remain muted but not kicked.", {
      reply_to_message_id: ctx.message.message_id,
    })
  } else {
    await bot.sendMessage(ctx.chat.id, "Usage: /captchakick <on/off>", {
      reply_to_message_id: ctx.message.message_id,
    })
  }
}

// Check if CAPTCHA is enabled and send challenge
export async function checkAndSendCaptcha(chatId: number, user: TelegramUser, chat: TelegramChat): Promise<boolean> {
  const { data: settings } = await supabase.from("captcha_settings").select("*").eq("chat_id", chatId).single()

  if (!settings?.captcha_enabled) {
    return false // CAPTCHA not enabled
  }

  const mode = settings.captcha_mode || "button"
  const muteTime = settings.mute_time || 300

  // Mute the user first
  try {
    await bot.restrictChatMember(
      chatId,
      user.id,
      { can_send_messages: false },
      { until_date: Math.floor(Date.now() / 1000) + muteTime },
    )
  } catch (e) {
    console.log("[v0] Failed to mute user for CAPTCHA:", e)
  }

  let keyboard
  let text = `Welcome ${user.first_name}! Please verify you're human.\n\n`

  if (mode === "button") {
    text += "Click the button below to verify:"
    keyboard = {
      inline_keyboard: [[{ text: "✅ I'm not a robot", callback_data: `captcha_verify_${user.id}` }]],
    }

    // Store simple verification
    pendingCaptchas.set(`${chatId}_${user.id}`, {
      chatId,
      userId: user.id,
      answer: "button",
      expiresAt: Date.now() + muteTime * 1000,
    })
  } else if (mode === "math") {
    const { question, answer } = generateMathCaptcha()
    text += `Solve this: <b>${question}</b>\n\nReply with the answer.`

    pendingCaptchas.set(`${chatId}_${user.id}`, {
      chatId,
      userId: user.id,
      answer,
      expiresAt: Date.now() + muteTime * 1000,
    })
  } else if (mode === "text") {
    const { text: code, answer } = generateTextCaptcha()
    text += `Type this code: <b>${code}</b>`

    pendingCaptchas.set(`${chatId}_${user.id}`, {
      chatId,
      userId: user.id,
      answer,
      expiresAt: Date.now() + muteTime * 1000,
    })
  }

  const msg = await bot.sendMessage(chatId, text, {
    parse_mode: "HTML",
    reply_markup: keyboard,
  })

  // Store message ID for cleanup
  const challenge = pendingCaptchas.get(`${chatId}_${user.id}`)
  if (challenge) {
    challenge.messageId = msg.message_id
  }

  // Schedule cleanup
  setTimeout(async () => {
    const pending = pendingCaptchas.get(`${chatId}_${user.id}`)
    if (pending) {
      pendingCaptchas.delete(`${chatId}_${user.id}`)

      // Delete the captcha message
      if (pending.messageId) {
        try {
          await bot.deleteMessage(chatId, pending.messageId)
        } catch {
          // Ignore
        }
      }

      // Check if should kick
      if (settings.kick_on_fail) {
        try {
          await bot.banChatMember(chatId, user.id)
          await bot.unbanChatMember(chatId, user.id)
          await bot.sendMessage(chatId, `${user.first_name} was removed for not completing CAPTCHA.`)
        } catch {
          // Ignore
        }
      }
    }
  }, muteTime * 1000)

  return true
}

// Handle CAPTCHA button callback
export async function handleCaptchaCallback(chatId: number, userId: number, callbackQueryId: string): Promise<boolean> {
  const key = `${chatId}_${userId}`
  const challenge = pendingCaptchas.get(key)

  if (!challenge || challenge.expiresAt < Date.now()) {
    await bot.answerCallbackQuery(callbackQueryId, {
      text: "This verification has expired.",
      show_alert: true,
    })
    return false
  }

  // Verify passed!
  pendingCaptchas.delete(key)

  // Unmute user
  try {
    await bot.restrictChatMember(chatId, userId, {
      can_send_messages: true,
      can_send_media_messages: true,
      can_send_other_messages: true,
      can_add_web_page_previews: true,
    })
  } catch (e) {
    console.log("[v0] Failed to unmute user after CAPTCHA:", e)
  }

  // Delete captcha message
  if (challenge.messageId) {
    try {
      await bot.deleteMessage(chatId, challenge.messageId)
    } catch {
      // Ignore
    }
  }

  await bot.answerCallbackQuery(callbackQueryId, {
    text: "Verification successful! Welcome to the group.",
    show_alert: false,
  })

  return true
}

// Check text CAPTCHA answer
export async function checkCaptchaAnswer(chatId: number, userId: number, text: string): Promise<boolean> {
  const key = `${chatId}_${userId}`
  const challenge = pendingCaptchas.get(key)

  if (!challenge || challenge.expiresAt < Date.now()) {
    return false
  }

  if (text.trim().toLowerCase() === challenge.answer.toLowerCase()) {
    pendingCaptchas.delete(key)

    // Unmute user
    try {
      await bot.restrictChatMember(chatId, userId, {
        can_send_messages: true,
        can_send_media_messages: true,
        can_send_other_messages: true,
        can_add_web_page_previews: true,
      })
    } catch (e) {
      console.log("[v0] Failed to unmute user after CAPTCHA:", e)
    }

    // Delete captcha message
    if (challenge.messageId) {
      try {
        await bot.deleteMessage(chatId, challenge.messageId)
      } catch {
        // Ignore
      }
    }

    await bot.sendMessage(chatId, "Verification successful! Welcome to the group.", {
      reply_to_message_id: undefined,
    })

    return true
  }

  return false
}

// Check if user has pending captcha
export function hasPendingCaptcha(chatId: number, userId: number): boolean {
  const key = `${chatId}_${userId}`
  const challenge = pendingCaptchas.get(key)
  return !!(challenge && challenge.expiresAt > Date.now())
}
