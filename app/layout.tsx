import type React from "react"
import type { Metadata } from "next"
import { Inter } from "next/font/google"
import { Analytics } from "@vercel/analytics/next"
import "./globals.css"

const inter = Inter({ subsets: ["latin"] })

export const metadata: Metadata = {
  title: "Sukoon - Telegram Group Management Bot",
  description:
    "A powerful Telegram group management bot with moderation, anti-spam, notes, filters, welcomes, federations and more.",
  keywords: ["telegram", "bot", "group management", "moderation", "anti-spam"],
  authors: [{ name: "Sukoon Team" }],
  openGraph: {
    title: "Sukoon - Telegram Group Management Bot",
    description: "A powerful Telegram group management bot with all the features you need.",
    type: "website",
  },
    generator: 'v0.app'
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className={`${inter.className} antialiased`}>
        {children}
        <Analytics />
      </body>
    </html>
  )
}
