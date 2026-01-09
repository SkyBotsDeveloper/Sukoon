// Telegram Bot Types for Sukoon

export interface TelegramUser {
  id: number
  is_bot: boolean
  first_name: string
  last_name?: string
  username?: string
  language_code?: string
}

export interface TelegramChat {
  id: number
  type: "private" | "group" | "supergroup" | "channel"
  title?: string
  username?: string
  first_name?: string
  last_name?: string
}

export interface TelegramMessage {
  message_id: number
  from?: TelegramUser
  sender_chat?: TelegramChat
  date: number
  chat: TelegramChat
  forward_from?: TelegramUser
  forward_from_chat?: TelegramChat
  reply_to_message?: TelegramMessage
  text?: string
  caption?: string
  entities?: TelegramMessageEntity[]
  photo?: TelegramPhotoSize[]
  document?: TelegramDocument
  video?: TelegramVideo
  audio?: TelegramAudio
  voice?: TelegramVoice
  video_note?: TelegramVideoNote // Added video_note
  sticker?: TelegramSticker
  animation?: TelegramAnimation
  contact?: TelegramContact
  location?: TelegramLocation
  poll?: TelegramPoll
  new_chat_members?: TelegramUser[]
  left_chat_member?: TelegramUser
  new_chat_title?: string
  pinned_message?: TelegramMessage
  via_bot?: TelegramUser
}

export interface TelegramMessageEntity {
  type: string
  offset: number
  length: number
  url?: string
  user?: TelegramUser
}

export interface TelegramPhotoSize {
  file_id: string
  file_unique_id: string
  width: number
  height: number
  file_size?: number
}

export interface TelegramDocument {
  file_id: string
  file_unique_id: string
  file_name?: string
  mime_type?: string
  file_size?: number
}

export interface TelegramVideo {
  file_id: string
  file_unique_id: string
  width: number
  height: number
  duration: number
  file_name?: string
  mime_type?: string
  file_size?: number
}

export interface TelegramAudio {
  file_id: string
  file_unique_id: string
  duration: number
  performer?: string
  title?: string
  file_name?: string
  mime_type?: string
  file_size?: number
}

export interface TelegramVoice {
  file_id: string
  file_unique_id: string
  duration: number
  mime_type?: string
  file_size?: number
}

export interface TelegramVideoNote {
  file_id: string
  file_unique_id: string
  length: number
  duration: number
  file_size?: number
}

export interface TelegramSticker {
  file_id: string
  file_unique_id: string
  width: number
  height: number
  is_animated: boolean
  is_video: boolean
}

export interface TelegramAnimation {
  file_id: string
  file_unique_id: string
  width: number
  height: number
  duration: number
}

export interface TelegramContact {
  phone_number: string
  first_name: string
  last_name?: string
  user_id?: number
}

export interface TelegramLocation {
  longitude: number
  latitude: number
}

export interface TelegramPoll {
  id: string
  question: string
  options: { text: string; voter_count: number }[]
}

export interface TelegramUpdate {
  update_id: number
  message?: TelegramMessage
  edited_message?: TelegramMessage
  callback_query?: TelegramCallbackQuery
  chat_member?: TelegramChatMemberUpdated
}

export interface TelegramCallbackQuery {
  id: string
  from: TelegramUser
  message?: TelegramMessage
  chat_instance: string
  data?: string
}

export interface TelegramChatMemberUpdated {
  chat: TelegramChat
  from: TelegramUser
  date: number
  old_chat_member: TelegramChatMember
  new_chat_member: TelegramChatMember
}

export interface TelegramChatMember {
  user: TelegramUser
  status: "creator" | "administrator" | "member" | "restricted" | "left" | "kicked"
}

export interface InlineKeyboardButton {
  text: string
  url?: string
  callback_data?: string
}

export interface InlineKeyboardMarkup {
  inline_keyboard: InlineKeyboardButton[][]
}

import type { TelegramBot } from "./bot"

export interface CommandContext {
  message: TelegramMessage
  chat: TelegramChat
  user: TelegramUser
  args: string[]
  replyToMessage?: TelegramMessage
  isAdmin: boolean
  isOwner: boolean
  isSudoer: boolean
  bot: TelegramBot
}
