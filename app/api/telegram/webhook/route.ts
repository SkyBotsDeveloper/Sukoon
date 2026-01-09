import { type NextRequest, NextResponse } from "next/server"
import { handleUpdate } from "@/lib/telegram/router"
import type { TelegramUpdate } from "@/lib/telegram/types"
import { bot, TelegramBot } from "@/lib/telegram/bot"
import { createClient } from "@supabase/supabase-js"

// Create Supabase client for webhook route
const supabase = createClient(process.env.NEXT_PUBLIC_SUPABASE_URL!, process.env.SUPABASE_SERVICE_ROLE_KEY!)

const cloneTokenCache = new Map<string, { valid: boolean; botId: number; timestamp: number }>()
const CACHE_TTL = 60000 // 1 minute

const processedUpdates = new Map<number, number>() // update_id -> timestamp
const UPDATE_TTL = 30000 // 30 seconds

function isUpdateProcessed(updateId: number): boolean {
  // Clean old entries
  const now = Date.now()
  for (const [id, timestamp] of processedUpdates.entries()) {
    if (now - timestamp > UPDATE_TTL) {
      processedUpdates.delete(id)
    }
  }

  if (processedUpdates.has(updateId)) {
    return true
  }

  processedUpdates.set(updateId, now)
  return false
}

async function validateCloneToken(token: string): Promise<{ valid: boolean; botId?: number }> {
  const cached = cloneTokenCache.get(token)
  if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
    return { valid: cached.valid, botId: cached.botId }
  }

  const { data } = await supabase.from("bot_clones").select("bot_id").eq("bot_token", token).maybeSingle()

  const valid = !!data
  const botId = data?.bot_id
  cloneTokenCache.set(token, { valid, botId, timestamp: Date.now() })
  return { valid, botId }
}

export async function POST(request: NextRequest) {
  try {
    const url = new URL(request.url)
    const customToken = url.searchParams.get("token")

    let currentBot: TelegramBot

    if (customToken) {
      const { valid } = await validateCloneToken(customToken)
      if (!valid) {
        console.error("[v0] Invalid clone token received")
        return NextResponse.json({ error: "Invalid token" }, { status: 401 })
      }
      currentBot = new TelegramBot(customToken)
    } else {
      currentBot = bot
    }

    const update: TelegramUpdate = await request.json()

    if (isUpdateProcessed(update.update_id)) {
      console.log("[v0] Duplicate update ignored:", update.update_id)
      return NextResponse.json({ ok: true })
    }

    // Process the update with the appropriate bot instance
    await handleUpdate(update, currentBot)

    return NextResponse.json({ ok: true })
  } catch (error) {
    console.error("[v0] Webhook error:", error)
    return NextResponse.json({ ok: true })
  }
}

export async function GET() {
  return NextResponse.json({
    status: "Sukoon Bot Webhook Active",
    timestamp: new Date().toISOString(),
  })
}
