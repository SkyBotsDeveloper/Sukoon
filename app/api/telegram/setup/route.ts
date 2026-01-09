import { type NextRequest, NextResponse } from "next/server"
import { bot } from "@/lib/telegram/bot"

export async function POST(request: NextRequest) {
  try {
    const { action } = await request.json()

    if (action === "setWebhook") {
      const webhookUrl = `${process.env.NEXT_PUBLIC_APP_URL}/api/telegram/webhook`

      await bot.setWebhook(webhookUrl, {
        allowed_updates: ["message", "edited_message", "callback_query", "chat_member"],
      })

      return NextResponse.json({
        success: true,
        message: `Webhook set to ${webhookUrl}`,
      })
    }

    if (action === "deleteWebhook") {
      await bot.deleteWebhook()
      return NextResponse.json({ success: true, message: "Webhook deleted" })
    }

    if (action === "getWebhookInfo") {
      const info = await bot.getWebhookInfo()
      return NextResponse.json({ success: true, info })
    }

    if (action === "getMe") {
      const me = await bot.getMe()
      return NextResponse.json({ success: true, bot: me })
    }

    return NextResponse.json({ error: "Invalid action" }, { status: 400 })
  } catch (error) {
    console.error("Setup error:", error)
    return NextResponse.json({ error: error instanceof Error ? error.message : "Setup failed" }, { status: 500 })
  }
}

export async function GET() {
  try {
    const webhookUrl = `${process.env.NEXT_PUBLIC_APP_URL}/api/telegram/webhook`

    // Set the webhook
    await bot.setWebhook(webhookUrl, {
      allowed_updates: ["message", "edited_message", "callback_query", "chat_member"],
    })

    // Get bot info and webhook info
    const [me, webhookInfo] = await Promise.all([bot.getMe(), bot.getWebhookInfo()])

    return NextResponse.json({
      success: true,
      message: `Webhook set to ${webhookUrl}`,
      bot: me,
      webhook: webhookInfo,
    })
  } catch (error) {
    console.error("Setup error:", error)
    return NextResponse.json({ error: error instanceof Error ? error.message : "Failed to setup" }, { status: 500 })
  }
}
