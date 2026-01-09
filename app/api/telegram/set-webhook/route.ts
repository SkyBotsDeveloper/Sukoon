import { type NextRequest, NextResponse } from "next/server"
import { bot } from "@/lib/telegram/bot"

function normalizeUrl(url: string): string {
  return url.replace(/\/+$/, "") // Remove trailing slashes
}

export async function POST(request: NextRequest) {
  try {
    // Get the webhook URL from the request body or use the default
    const body = await request.json().catch(() => ({}))
    let baseUrl = process.env.NEXT_PUBLIC_APP_URL || process.env.VERCEL_URL || ""

    baseUrl = normalizeUrl(baseUrl.startsWith("http") ? baseUrl : `https://${baseUrl}`)

    const webhookUrl = body.url || `${baseUrl}/api/telegram/webhook`

    console.log("[v0] Setting webhook to:", webhookUrl)
    console.log("[v0] Bot token present:", !!process.env.TELEGRAM_BOT_TOKEN)

    const webhookOptions: {
      allowed_updates: string[]
      secret_token?: string
      drop_pending_updates?: boolean
    } = {
      allowed_updates: ["message", "edited_message", "callback_query", "chat_member"],
      drop_pending_updates: true,
    }

    // Add secret token if configured
    if (process.env.TELEGRAM_WEBHOOK_SECRET) {
      webhookOptions.secret_token = process.env.TELEGRAM_WEBHOOK_SECRET
      console.log("[v0] Using webhook secret token")
    }

    // Set the webhook
    const setResult = await bot.setWebhook(webhookUrl, webhookOptions)
    console.log("[v0] Webhook set result:", setResult)

    // Get webhook info to verify
    const webhookInfo = await bot.getWebhookInfo()
    console.log("[v0] Webhook info after setting:", webhookInfo)

    if (!webhookInfo.url) {
      throw new Error("Webhook URL was not set properly. Check Telegram API response.")
    }

    return NextResponse.json({
      success: true,
      message: "Webhook set successfully",
      webhookUrl,
      webhook: webhookInfo,
    })
  } catch (error) {
    console.error("[v0] Error setting webhook:", error)
    const errorMessage = error instanceof Error ? error.message : "Unknown error"
    console.error("[v0] Full error details:", errorMessage)
    return NextResponse.json(
      {
        error: "Failed to set webhook",
        details: errorMessage,
        suggestions: [
          "Check that TELEGRAM_BOT_TOKEN is set correctly",
          "Verify the webhook URL is correct and accessible",
          "Check your internet connection",
          "The bot token may have been revoked - get a new one from @BotFather",
        ],
      },
      { status: 500 },
    )
  }
}

export async function DELETE() {
  try {
    await bot.deleteWebhook()

    return NextResponse.json({
      success: true,
      message: "Webhook deleted successfully",
    })
  } catch (error) {
    console.error("[v0] Error deleting webhook:", error)
    return NextResponse.json({ error: "Failed to delete webhook" }, { status: 500 })
  }
}

export async function GET() {
  try {
    const webhookInfo = await bot.getWebhookInfo()
    const botInfo = await bot.getMe()

    return NextResponse.json({
      bot: botInfo,
      webhook: webhookInfo,
      envCheck: {
        hasBotToken: !!process.env.TELEGRAM_BOT_TOKEN,
        hasBotUsername: !!process.env.BOT_USERNAME,
        hasWebhookSecret: !!process.env.TELEGRAM_WEBHOOK_SECRET,
        hasAppUrl: !!process.env.NEXT_PUBLIC_APP_URL,
      },
    })
  } catch (error) {
    console.error("[v0] Error getting webhook info:", error)
    return NextResponse.json(
      { error: "Failed to get webhook info", details: error instanceof Error ? error.message : "Unknown error" },
      { status: 500 },
    )
  }
}
