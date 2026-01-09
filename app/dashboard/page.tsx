"use client"

import { useState, useEffect } from "react"
import { Bot, RefreshCw, CheckCircle, XCircle, Globe, Server, Users, MessageSquare } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

interface WebhookInfo {
  url?: string
  has_custom_certificate?: boolean
  pending_update_count?: number
  last_error_date?: number
  last_error_message?: string
}

interface BotInfo {
  id?: number
  first_name?: string
  username?: string
}

export default function DashboardPage() {
  const [loading, setLoading] = useState(false)
  const [webhookInfo, setWebhookInfo] = useState<WebhookInfo | null>(null)
  const [botInfo, setBotInfo] = useState<BotInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [customUrl, setCustomUrl] = useState("")

  const fetchStatus = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch("/api/telegram/set-webhook")
      const data = await res.json()
      if (data.error) {
        setError(data.error)
      } else {
        setWebhookInfo(data.webhook)
        setBotInfo(data.bot)
      }
    } catch {
      setError("Failed to fetch status")
    }
    setLoading(false)
  }

  const setWebhook = async () => {
    setLoading(true)
    setError(null)
    setSuccess(null)
    try {
      const res = await fetch("/api/telegram/set-webhook", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: customUrl || undefined }),
      })
      const data = await res.json()

      if (data.error) {
        const errorMsg = data.details ? `${data.error}: ${data.details}` : data.error
        setError(errorMsg)
        if (data.suggestions) {
          console.error("Suggestions:", data.suggestions)
        }
      } else {
        setSuccess("Webhook set successfully!")
        setWebhookInfo(data.webhook)
        setCustomUrl("")
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to set webhook"
      setError(msg)
      console.error("Error:", err)
    }
    setLoading(false)
  }

  const deleteWebhook = async () => {
    setLoading(true)
    setError(null)
    setSuccess(null)
    try {
      const res = await fetch("/api/telegram/set-webhook", { method: "DELETE" })
      const data = await res.json()
      if (data.error) {
        setError(data.error)
      } else {
        setSuccess("Webhook deleted successfully!")
        setWebhookInfo(null)
      }
    } catch {
      setError("Failed to delete webhook")
    }
    setLoading(false)
  }

  useEffect(() => {
    fetchStatus()
  }, [])

  return (
    <main className="min-h-screen bg-gradient-to-b from-background to-muted/20">
      {/* Header */}
      <header className="border-b bg-background/80 backdrop-blur-sm sticky top-0 z-50">
        <div className="container mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bot className="h-8 w-8 text-primary" />
            <span className="text-xl font-bold">Sukoon Dashboard</span>
          </div>
          <Button variant="outline" size="sm" onClick={fetchStatus} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </header>

      <div className="container mx-auto px-4 py-8">
        {/* Status Messages */}
        {error && (
          <div className="mb-6 p-4 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive flex items-center gap-2">
            <XCircle className="h-5 w-5" />
            {error}
          </div>
        )}
        {success && (
          <div className="mb-6 p-4 rounded-lg bg-green-500/10 border border-green-500/20 text-green-600 flex items-center gap-2">
            <CheckCircle className="h-5 w-5" />
            {success}
          </div>
        )}

        {/* Bot Info */}
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Bot Status</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold flex items-center gap-2">
                {botInfo ? (
                  <>
                    <span className="h-3 w-3 rounded-full bg-green-500" />
                    Online
                  </>
                ) : (
                  <>
                    <span className="h-3 w-3 rounded-full bg-yellow-500" />
                    Unknown
                  </>
                )}
              </div>
              <p className="text-xs text-muted-foreground mt-1">@{botInfo?.username || "loading..."}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Bot ID</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold font-mono">{botInfo?.id || "---"}</div>
              <p className="text-xs text-muted-foreground mt-1">{botInfo?.first_name || "Loading..."}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Webhook Status</CardTitle>
              <Globe className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold flex items-center gap-2">
                {webhookInfo?.url ? (
                  <>
                    <span className="h-3 w-3 rounded-full bg-green-500" />
                    Active
                  </>
                ) : (
                  <>
                    <span className="h-3 w-3 rounded-full bg-red-500" />
                    Inactive
                  </>
                )}
              </div>
              <p className="text-xs text-muted-foreground mt-1 truncate max-w-[200px]">
                {webhookInfo?.url || "No webhook set"}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Pending Updates</CardTitle>
              <MessageSquare className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{webhookInfo?.pending_update_count || 0}</div>
              <p className="text-xs text-muted-foreground mt-1">Updates in queue</p>
            </CardContent>
          </Card>
        </div>

        {/* Webhook Management */}
        <Card className="mb-8">
          <CardHeader>
            <CardTitle>Webhook Management</CardTitle>
            <CardDescription>Configure the webhook URL for your Telegram bot to receive updates.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="webhook-url">Custom Webhook URL (optional)</Label>
              <Input
                id="webhook-url"
                placeholder="https://your-domain.com/api/telegram/webhook"
                value={customUrl}
                onChange={(e) => setCustomUrl(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Leave empty to use the default URL based on your deployment.
              </p>
            </div>

            {webhookInfo?.last_error_message && (
              <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20">
                <p className="text-sm text-destructive">
                  <strong>Last Error:</strong> {webhookInfo.last_error_message}
                </p>
                {webhookInfo.last_error_date && (
                  <p className="text-xs text-muted-foreground mt-1">
                    {new Date(webhookInfo.last_error_date * 1000).toLocaleString()}
                  </p>
                )}
              </div>
            )}

            <div className="flex gap-3">
              <Button onClick={setWebhook} disabled={loading}>
                {loading ? <RefreshCw className="h-4 w-4 mr-2 animate-spin" /> : <Globe className="h-4 w-4 mr-2" />}
                Set Webhook
              </Button>
              <Button variant="destructive" onClick={deleteWebhook} disabled={loading}>
                Delete Webhook
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Quick Links */}
        <Card>
          <CardHeader>
            <CardTitle>Quick Links</CardTitle>
            <CardDescription>Useful links for managing your bot.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid md:grid-cols-3 gap-4">
              <a
                href={`https://t.me/${botInfo?.username || ""}`}
                target="_blank"
                rel="noopener noreferrer"
                className="p-4 rounded-lg border hover:bg-muted/50 transition-colors flex items-center gap-3"
              >
                <MessageSquare className="h-5 w-5 text-primary" />
                <div>
                  <p className="font-medium">Open Bot</p>
                  <p className="text-xs text-muted-foreground">Chat with your bot</p>
                </div>
              </a>
              <a
                href={`https://t.me/${botInfo?.username || ""}?startgroup=true`}
                target="_blank"
                rel="noopener noreferrer"
                className="p-4 rounded-lg border hover:bg-muted/50 transition-colors flex items-center gap-3"
              >
                <Users className="h-5 w-5 text-primary" />
                <div>
                  <p className="font-medium">Add to Group</p>
                  <p className="text-xs text-muted-foreground">Add bot to a group</p>
                </div>
              </a>
              <a
                href="https://core.telegram.org/bots/api"
                target="_blank"
                rel="noopener noreferrer"
                className="p-4 rounded-lg border hover:bg-muted/50 transition-colors flex items-center gap-3"
              >
                <Server className="h-5 w-5 text-primary" />
                <div>
                  <p className="font-medium">Bot API Docs</p>
                  <p className="text-xs text-muted-foreground">Telegram Bot API</p>
                </div>
              </a>
            </div>
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
