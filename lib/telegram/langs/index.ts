// Language system index
import { en } from "./en"
import type { TranslationKey } from "./en"
import { hi } from "./hi"
import { es } from "./es"
import { supabase } from "../utils"

// All available languages
export const languages: Record<string, typeof en> = {
  en,
  hi,
  es,
}

// Language metadata
export const languageInfo: Record<string, { name: string; flag: string }> = {
  en: { name: "English", flag: "🇬🇧" },
  hi: { name: "हिन्दी", flag: "🇮🇳" },
  es: { name: "Español", flag: "🇪🇸" },
}

// Cache for chat languages to avoid repeated DB calls
const languageCache: Map<number, string> = new Map()

// Get language for a chat
export async function getChatLanguage(chatId: number): Promise<string> {
  // Check cache first
  const cached = languageCache.get(chatId)
  if (cached) return cached

  try {
    const { data } = await supabase.from("chat_settings").select("language").eq("chat_id", chatId).maybeSingle()

    const lang = data?.language || "en"
    languageCache.set(chatId, lang)
    return lang
  } catch {
    return "en"
  }
}

// Set language for a chat
export async function setChatLanguage(chatId: number, langCode: string): Promise<boolean> {
  if (!languages[langCode]) return false

  try {
    await supabase.from("chat_settings").upsert(
      {
        chat_id: chatId,
        language: langCode,
        updated_at: new Date().toISOString(),
      },
      { onConflict: "chat_id" },
    )

    languageCache.set(chatId, langCode)
    return true
  } catch {
    return false
  }
}

// Get translated string with variable replacement
export function t(lang: string, key: TranslationKey, vars?: Record<string, string | number>): string {
  const translations = languages[lang] || languages.en
  let text = (translations[key] as string) || (languages.en[key] as string) || key

  if (vars) {
    for (const [varKey, value] of Object.entries(vars)) {
      text = text.replace(new RegExp(`\\{${varKey}\\}`, "g"), String(value))
    }
  }

  return text
}

// Async version that fetches chat language
export async function tr(chatId: number, key: TranslationKey, vars?: Record<string, string | number>): Promise<string> {
  const lang = await getChatLanguage(chatId)
  return t(lang, key, vars)
}

// Export types
export type { TranslationKey }
