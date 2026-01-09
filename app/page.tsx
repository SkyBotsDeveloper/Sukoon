import Link from "next/link"
import { Shield, MessageSquare, Users, Zap, Lock, Bell, Globe, Settings, ChevronRight, Bot } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

const SUPPORT_GROUP = "https://t.me/VivaanSupport"
const UPDATES_CHANNEL = "https://t.me/VivaanUpdates"

const features = [
  {
    icon: Shield,
    title: "Moderation",
    description: "Ban, mute, kick, and warn users with powerful moderation tools. Supports timed bans and mutes.",
  },
  {
    icon: Lock,
    title: "Anti-Spam",
    description: "Locks, blocklists, and antiflood protection to keep your group clean and spam-free.",
  },
  {
    icon: MessageSquare,
    title: "Notes & Filters",
    description: "Save notes and create auto-reply filters with support for media, buttons, and formatting.",
  },
  {
    icon: Bell,
    title: "Welcome Messages",
    description: "Greet new members with customizable welcome messages. Supports variables and buttons.",
  },
  {
    icon: Globe,
    title: "Federations",
    description: "Create federations to manage bans across multiple groups simultaneously.",
  },
  {
    icon: Settings,
    title: "Admin Tools",
    description: "Pin messages, purge chats, approve users, disable commands, and set up log channels.",
  },
]

const commands = [
  { category: "Moderation", cmds: ["/ban", "/mute", "/kick", "/warn", "/unban", "/unmute"] },
  { category: "Anti-Spam", cmds: ["/lock", "/unlock", "/blocklist", "/setflood"] },
  { category: "Content", cmds: ["/save", "/notes", "/filter", "/rules"] },
  { category: "Welcome", cmds: ["/setwelcome", "/welcome", "/setgoodbye"] },
  { category: "Federation", cmds: ["/newfed", "/joinfed", "/fban", "/fedinfo"] },
  { category: "Admin", cmds: ["/pin", "/purge", "/approve", "/setlog"] },
]

export default function Home() {
  const botUsername = process.env.BOT_USERNAME || "SukoonBot"

  return (
    <main className="min-h-screen bg-gradient-to-b from-background to-muted/20">
      {/* Header */}
      <header className="border-b bg-background/80 backdrop-blur-sm sticky top-0 z-50">
        <div className="container mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bot className="h-8 w-8 text-primary" />
            <span className="text-xl font-bold">Sukoon</span>
          </div>
          <nav className="flex items-center gap-4">
            <Link href="#features" className="text-sm text-muted-foreground hover:text-foreground transition-colors">
              Features
            </Link>
            <Link href="#commands" className="text-sm text-muted-foreground hover:text-foreground transition-colors">
              Commands
            </Link>
            <a
              href={SUPPORT_GROUP}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              Support
            </a>
            <a
              href={UPDATES_CHANNEL}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              Updates
            </a>
            <Button asChild size="sm">
              <a href={`https://t.me/${botUsername}?startgroup=true`} target="_blank" rel="noopener noreferrer">
                Add to Group
              </a>
            </Button>
          </nav>
        </div>
      </header>

      {/* Hero Section */}
      <section className="container mx-auto px-4 py-24 text-center">
        <div className="max-w-3xl mx-auto">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/10 text-primary text-sm mb-6">
            <Zap className="h-4 w-4" />
            Powerful Group Management
          </div>
          <h1 className="text-5xl md:text-6xl font-bold tracking-tight mb-6 bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text">
            Sukoon
          </h1>
          <p className="text-xl text-muted-foreground mb-8 leading-relaxed">
            A powerful Telegram group management bot inspired by Miss Rose. Moderation, anti-spam, notes, filters,
            welcomes, federations, and much more.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Button asChild size="lg" className="gap-2">
              <a href={`https://t.me/${botUsername}?startgroup=true`} target="_blank" rel="noopener noreferrer">
                <Users className="h-5 w-5" />
                Add to Group
              </a>
            </Button>
            <Button asChild variant="outline" size="lg" className="gap-2 bg-transparent">
              <a href={`https://t.me/${botUsername}`} target="_blank" rel="noopener noreferrer">
                <MessageSquare className="h-5 w-5" />
                Start in PM
              </a>
            </Button>
          </div>
          <div className="flex gap-4 justify-center mt-6">
            <Button asChild variant="ghost" size="sm">
              <a href={UPDATES_CHANNEL} target="_blank" rel="noopener noreferrer">
                <Bell className="h-4 w-4 mr-2" />
                Updates Channel
              </a>
            </Button>
            <Button asChild variant="ghost" size="sm">
              <a href={SUPPORT_GROUP} target="_blank" rel="noopener noreferrer">
                <Users className="h-4 w-4 mr-2" />
                Support Group
              </a>
            </Button>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="container mx-auto px-4 py-24">
        <div className="text-center mb-16">
          <h2 className="text-3xl font-bold mb-4">Everything You Need</h2>
          <p className="text-muted-foreground max-w-2xl mx-auto">
            Sukoon comes packed with all the features you need to manage your Telegram groups effectively.
          </p>
        </div>
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((feature) => (
            <Card
              key={feature.title}
              className="bg-card/50 backdrop-blur-sm border-border/50 hover:border-primary/50 transition-colors"
            >
              <CardHeader>
                <div className="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                  <feature.icon className="h-6 w-6 text-primary" />
                </div>
                <CardTitle>{feature.title}</CardTitle>
                <CardDescription>{feature.description}</CardDescription>
              </CardHeader>
            </Card>
          ))}
        </div>
      </section>

      {/* Commands Section */}
      <section id="commands" className="container mx-auto px-4 py-24">
        <div className="text-center mb-16">
          <h2 className="text-3xl font-bold mb-4">Quick Commands Reference</h2>
          <p className="text-muted-foreground max-w-2xl mx-auto">
            Here are some of the most commonly used commands. Use /help in the bot for full documentation.
          </p>
        </div>
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {commands.map((group) => (
            <Card key={group.category} className="bg-card/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-lg">{group.category}</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {group.cmds.map((cmd) => (
                    <code key={cmd} className="px-2 py-1 rounded bg-muted text-sm font-mono">
                      {cmd}
                    </code>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* CTA Section */}
      <section className="container mx-auto px-4 py-24">
        <Card className="bg-gradient-to-r from-primary/10 to-primary/5 border-primary/20">
          <CardContent className="py-12 text-center">
            <h2 className="text-3xl font-bold mb-4">Ready to Get Started?</h2>
            <p className="text-muted-foreground mb-8 max-w-xl mx-auto">
              Add Sukoon to your group and make it an admin to unlock all features. It only takes a few seconds!
            </p>
            <Button asChild size="lg" className="gap-2">
              <a href={`https://t.me/${botUsername}?startgroup=true`} target="_blank" rel="noopener noreferrer">
                Add Sukoon Now
                <ChevronRight className="h-5 w-5" />
              </a>
            </Button>
          </CardContent>
        </Card>
      </section>

      {/* Footer */}
      <footer className="border-t bg-muted/30">
        <div className="container mx-auto px-4 py-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <Bot className="h-6 w-6 text-primary" />
              <span className="font-semibold">Sukoon</span>
              <span className="text-muted-foreground text-sm">- Telegram Group Management Bot</span>
            </div>
            <div className="flex items-center gap-6">
              <a
                href={UPDATES_CHANNEL}
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1 text-sm"
              >
                <Bell className="h-4 w-4" />
                Updates
              </a>
              <a
                href={SUPPORT_GROUP}
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1 text-sm"
              >
                <Users className="h-4 w-4" />
                Support
              </a>
              <a
                href={`https://t.me/${botUsername}`}
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground transition-colors"
              >
                <MessageSquare className="h-5 w-5" />
              </a>
            </div>
          </div>
        </div>
      </footer>
    </main>
  )
}
