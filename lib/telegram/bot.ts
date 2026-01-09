import type { InlineKeyboardMarkup } from "./types"

const BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN
const API_URL = BOT_TOKEN ? `https://api.telegram.org/bot${BOT_TOKEN}` : ""

// Telegram API wrapper
export class TelegramBot {
  private token: string
  private apiUrl: string

  constructor(token?: string) {
    this.token = token || BOT_TOKEN || ""
    this.apiUrl = `https://api.telegram.org/bot${this.token}`
  }

  private async request(method: string, params: Record<string, unknown> = {}) {
    if (!this.token) {
      console.error("TELEGRAM_BOT_TOKEN is not set!")
      throw new Error("TELEGRAM_BOT_TOKEN is not configured")
    }

    const response = await fetch(`${this.apiUrl}/${method}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(params),
    })

    const data = await response.json()

    if (!data.ok) {
      console.error(`Telegram API error (${method}):`, data.description)
      throw new Error(data.description)
    }

    return data.result
  }

  // Message methods
  async sendMessage(
    chatId: number | string,
    text: string,
    options: {
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
      disable_web_page_preview?: boolean
    } = {},
  ) {
    return this.request("sendMessage", {
      chat_id: chatId,
      text,
      ...options,
    })
  }

  async deleteMessage(chatId: number | string, messageId: number) {
    try {
      return await this.request("deleteMessage", {
        chat_id: chatId,
        message_id: messageId,
      })
    } catch {
      return false
    }
  }

  async editMessageText(
    chatId: number | string,
    messageId: number,
    text: string,
    options: {
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("editMessageText", {
      chat_id: chatId,
      message_id: messageId,
      text,
      ...options,
    })
  }

  async copyMessage(
    chatId: number | string,
    fromChatId: number | string,
    messageId: number,
    options: {
      caption?: string
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("copyMessage", {
      chat_id: chatId,
      from_chat_id: fromChatId,
      message_id: messageId,
      ...options,
    })
  }

  async forwardMessage(chatId: number | string, fromChatId: number | string, messageId: number) {
    return this.request("forwardMessage", {
      chat_id: chatId,
      from_chat_id: fromChatId,
      message_id: messageId,
    })
  }

  // Media methods
  async sendPhoto(
    chatId: number | string,
    photo: string,
    options: {
      caption?: string
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("sendPhoto", {
      chat_id: chatId,
      photo,
      ...options,
    })
  }

  async sendDocument(
    chatId: number | string,
    document: string,
    options: {
      caption?: string
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("sendDocument", {
      chat_id: chatId,
      document,
      ...options,
    })
  }

  async sendVideo(
    chatId: number | string,
    video: string,
    options: {
      caption?: string
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("sendVideo", {
      chat_id: chatId,
      video,
      ...options,
    })
  }

  async sendSticker(chatId: number | string, sticker: string) {
    return this.request("sendSticker", {
      chat_id: chatId,
      sticker,
    })
  }

  async sendAudio(
    chatId: number | string,
    audio: string,
    options: {
      caption?: string
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("sendAudio", {
      chat_id: chatId,
      audio,
      ...options,
    })
  }

  async sendVoice(
    chatId: number | string,
    voice: string,
    options: {
      caption?: string
      reply_to_message_id?: number
    } = {},
  ) {
    return this.request("sendVoice", {
      chat_id: chatId,
      voice,
      ...options,
    })
  }

  async sendAnimation(
    chatId: number | string,
    animation: string,
    options: {
      caption?: string
      parse_mode?: "HTML" | "Markdown" | "MarkdownV2"
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("sendAnimation", {
      chat_id: chatId,
      animation,
      ...options,
    })
  }

  async sendVideoNote(
    chatId: number | string,
    videoNote: string,
    options: {
      reply_to_message_id?: number
      reply_markup?: InlineKeyboardMarkup
    } = {},
  ) {
    return this.request("sendVideoNote", {
      chat_id: chatId,
      video_note: videoNote,
      ...options,
    })
  }

  // Chat member methods
  async banChatMember(
    chatId: number | string,
    userId: number,
    options: { until_date?: number; revoke_messages?: boolean } = {},
  ) {
    return this.request("banChatMember", {
      chat_id: chatId,
      user_id: userId,
      ...options,
    })
  }

  async unbanChatMember(chatId: number | string, userId: number, options: { only_if_banned?: boolean } = {}) {
    return this.request("unbanChatMember", {
      chat_id: chatId,
      user_id: userId,
      ...options,
    })
  }

  async kickChatMember(chatId: number | string, userId: number) {
    await this.banChatMember(chatId, userId)
    await this.unbanChatMember(chatId, userId)
  }

  async restrictChatMember(
    chatId: number | string,
    userId: number,
    permissions: {
      can_send_messages?: boolean
      can_send_audios?: boolean
      can_send_documents?: boolean
      can_send_photos?: boolean
      can_send_videos?: boolean
      can_send_video_notes?: boolean
      can_send_voice_notes?: boolean
      can_send_polls?: boolean
      can_send_other_messages?: boolean
      can_add_web_page_previews?: boolean
      can_change_info?: boolean
      can_invite_users?: boolean
      can_pin_messages?: boolean
      can_manage_topics?: boolean
    },
    options: { until_date?: number; use_independent_chat_permissions?: boolean } = {},
  ) {
    return this.request("restrictChatMember", {
      chat_id: chatId,
      user_id: userId,
      permissions: permissions,
      ...options,
    })
  }

  async promoteChatMember(
    chatId: number | string,
    userId: number,
    permissions: {
      is_anonymous?: boolean
      can_manage_chat?: boolean
      can_delete_messages?: boolean
      can_manage_video_chats?: boolean
      can_restrict_members?: boolean
      can_promote_members?: boolean
      can_change_info?: boolean
      can_invite_users?: boolean
      can_post_stories?: boolean
      can_edit_stories?: boolean
      can_delete_stories?: boolean
      can_post_messages?: boolean
      can_edit_messages?: boolean
      can_pin_messages?: boolean
      can_manage_topics?: boolean
    } = {},
  ) {
    return this.request("promoteChatMember", {
      chat_id: chatId,
      user_id: userId,
      ...permissions,
    })
  }

  async getChatMember(chatId: number | string, userId: number) {
    return this.request("getChatMember", {
      chat_id: chatId,
      user_id: userId,
    })
  }

  async getChatAdministrators(chatId: number | string) {
    return this.request("getChatAdministrators", {
      chat_id: chatId,
    })
  }

  async getChat(chatId: number | string) {
    return this.request("getChat", {
      chat_id: chatId,
    })
  }

  async getChatMemberCount(chatId: number | string) {
    return this.request("getChatMemberCount", {
      chat_id: chatId,
    })
  }

  async leaveChat(chatId: number | string) {
    return this.request("leaveChat", {
      chat_id: chatId,
    })
  }

  // Callback query
  async answerCallbackQuery(callbackQueryId: string, options: { text?: string; show_alert?: boolean } = {}) {
    return this.request("answerCallbackQuery", {
      callback_query_id: callbackQueryId,
      ...options,
    })
  }

  // Pin message
  async pinChatMessage(chatId: number | string, messageId: number, options: { disable_notification?: boolean } = {}) {
    return this.request("pinChatMessage", {
      chat_id: chatId,
      message_id: messageId,
      ...options,
    })
  }

  async unpinChatMessage(chatId: number | string, messageId?: number) {
    const params: Record<string, unknown> = { chat_id: chatId }
    if (messageId) params.message_id = messageId
    return this.request("unpinChatMessage", params)
  }

  async unpinAllChatMessages(chatId: number | string) {
    return this.request("unpinAllChatMessages", {
      chat_id: chatId,
    })
  }

  async setChatAdministratorCustomTitle(chatId: number | string, userId: number, customTitle: string) {
    return this.request("setChatAdministratorCustomTitle", {
      chat_id: chatId,
      user_id: userId,
      custom_title: customTitle,
    })
  }

  async setChatTitle(chatId: number | string, title: string) {
    return this.request("setChatTitle", {
      chat_id: chatId,
      title,
    })
  }

  async setChatDescription(chatId: number | string, description: string) {
    return this.request("setChatDescription", {
      chat_id: chatId,
      description,
    })
  }

  async setChatPhoto(chatId: number | string, photo: string) {
    return this.request("setChatPhoto", {
      chat_id: chatId,
      photo,
    })
  }

  async deleteChatPhoto(chatId: number | string) {
    return this.request("deleteChatPhoto", {
      chat_id: chatId,
    })
  }

  async setChatStickerSet(chatId: number | string, stickerSetName: string) {
    return this.request("setChatStickerSet", {
      chat_id: chatId,
      sticker_set_name: stickerSetName,
    })
  }

  async deleteChatStickerSet(chatId: number | string) {
    return this.request("deleteChatStickerSet", {
      chat_id: chatId,
    })
  }

  async exportChatInviteLink(chatId: number | string) {
    return this.request("exportChatInviteLink", {
      chat_id: chatId,
    })
  }

  async setWebhook(url: string, options: { allowed_updates?: string[]; secret_token?: string } = {}) {
    return this.request("setWebhook", {
      url,
      drop_pending_updates: true,
      ...options,
    })
  }

  async deleteWebhook() {
    return this.request("deleteWebhook", {
      drop_pending_updates: true,
    })
  }

  async getWebhookInfo() {
    return this.request("getWebhookInfo")
  }

  async getMe() {
    return this.request("getMe")
  }

  async customRequest(method: string, params: Record<string, unknown> = {}) {
    return this.request(method, params)
  }
}

export const bot = new TelegramBot()
