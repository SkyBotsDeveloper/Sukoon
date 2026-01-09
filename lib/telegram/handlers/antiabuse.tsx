import type { CommandContext, TelegramMessage } from "../types"
import type { TelegramBot } from "../bot"
import { supabase } from "../utils"

// Comprehensive abuse word list - Hinglish, Hindi transliteration, English, Urdu, and common variations
// This uses a Set for O(1) lookup performance
const ABUSE_WORDS = new Set([
  // Hinglish/Hindi transliteration (most common in Indian groups)
  "madarchod",
  "madarc",
  "mc",
  "maderchod",
  "madarchd",
  "motherchod",
  "m@darchod",
  "maadarchod",
  "madarjaat",
  "maadar",
  "mamaderchod",
  "madarchoot",
  "bhosdike",
  "bhosdi",
  "bhosdiwale",
  "bhosdk",
  "bsdk",
  "bhosdik",
  "b.s.d.k",
  "bh0sdk",
  "bhosdiwala",
  "bhosdina",
  "bhosadike",
  "bhosadiwale",
  "bhosda",
  "bhosdaa",
  "behenchod",
  "behen",
  "bc",
  "bhnchod",
  "benchod",
  "bhenchod",
  "bhnchd",
  "b.c",
  "beh3nchod",
  "behanchod",
  "behnchod",
  "behanchoot",
  "behen ke",
  "behenkelode",
  "behenkalauda",
  "chutiya",
  "chut",
  "chutiye",
  "chutiyapa",
  "chutia",
  "ch00tiya",
  "choot",
  "c*utiya",
  "chutiyagiri",
  "chutiyapanti",
  "chutad",
  "chutiyahai",
  "chootiyo",
  "chutmarani",
  "gandu",
  "gaandu",
  "gand",
  "gaand",
  "g@ndu",
  "g4ndu",
  "ganduu",
  "gandmasti",
  "gandmara",
  "gandphaad",
  "gandchaatu",
  "gandfat",
  "gandmein",
  "gaandmarau",
  "lodu",
  "lauda",
  "lund",
  "l*nd",
  "laude",
  "lundure",
  "l0du",
  "l@uda",
  "laudu",
  "loda",
  "lavda",
  "lavde",
  "lundka",
  "lundtopi",
  "laudalassan",
  "randi",
  "rand",
  "randikhana",
  "r@ndi",
  "r4ndi",
  "randee",
  "randiya",
  "randirona",
  "randikabachha",
  "randibaaz",
  "randikhaana",
  "randiyon",
  "harami",
  "haramkhor",
  "haram",
  "h@rami",
  "haraami",
  "haramipana",
  "haramzade",
  "saala",
  "sala",
  "saale",
  "saaley",
  "s@ala",
  "saalee",
  "salaa",
  "saleya",
  "kamina",
  "kameena",
  "kameene",
  "kam1na",
  "kaminey",
  "kaminepan",
  "tatti",
  "potty",
  "tatt1",
  "tattee",
  "tattikhana",
  "jhatu",
  "jhaant",
  "jhant",
  "jh@tu",
  "jhaantu",
  "jhaantke",
  "bakland",
  "bakl@nd",
  "baklund",
  "baklol",
  "chodu",
  "chod",
  "chodumal",
  "ch0d",
  "chodna",
  "chodenge",
  "choddunga",
  "teri maa",
  "tera baap",
  "tere baap",
  "maa ki",
  "baap ki",
  "behen ki",
  "teri behen",
  "teri ma",
  "tere maa",
  "tumhari maa",
  "tumhara baap",
  "chinal",
  "chinaal",
  "ch1nal",
  "chhinaal",
  "raand",
  "r@@nd",
  "raandh",
  "hijra",
  "hijre",
  "h1jra",
  "hijda",
  "hijde",
  "chakka",
  "ch@kka",
  "chakke",
  "chhakka",
  "bhadwa",
  "bhadve",
  "bh@dwa",
  "bhadwaa",
  "bhadwi",
  "dalla",
  "d@lla",
  "dalal",
  "dalali",
  "namard",
  "n@mard",
  "namardi",
  "napunsak",
  "lesbo",
  "l3sbo",
  "lesbi",
  "fattu",
  "phattu",
  "fuddu",
  "fuddi",
  "phuddi",
  "besharam",
  "besharmi",
  "besharmse",
  "nalayak",
  "nalayaq",
  "nalayakhai",
  "nikamma",
  "nikamme",
  "nikammi",
  "tharki",
  "tharkii",
  "tharak",
  "tharakpan",
  "charsi",
  "charasi",
  "ganjedi",

  // English abuse words
  "fuck",
  "fucking",
  "fucker",
  "fucked",
  "fck",
  "f*ck",
  "fuk",
  "fuc",
  "phuck",
  "f**k",
  "fuckyou",
  "fuckoff",
  "fuckface",
  "fucku",
  "fcker",
  "effing",
  "fugly",
  "shit",
  "shitty",
  "sh1t",
  "shiit",
  "sh!t",
  "s**t",
  "shithead",
  "shitface",
  "bullshit",
  "bitch",
  "b1tch",
  "biatch",
  "b!tch",
  "bi+ch",
  "bitchy",
  "bitches",
  "sonofabitch",
  "bastard",
  "b@stard",
  "bstrd",
  "bastards",
  "asshole",
  "assh0le",
  "a$$hole",
  "arsehole",
  "asshat",
  "asswipe",
  "dick",
  "d1ck",
  "dikk",
  "d!ck",
  "dickhead",
  "dicks",
  "dckhead",
  "pussy",
  "pu$$y",
  "pus5y",
  "p*ssy",
  "pussies",
  "cunt",
  "c*nt",
  "cvnt",
  "cunts",
  "nigga",
  "nigger",
  "n1gga",
  "n!gger",
  "nigg@",
  "negro",
  "n1gger",
  "whore",
  "wh0re",
  "h0e",
  "hoe",
  "whores",
  "slut",
  "sl*t",
  "s1ut",
  "slutty",
  "sluts",
  "retard",
  "ret@rd",
  "r3tard",
  "retarded",
  "fag",
  "faggot",
  "f@g",
  "f@ggot",
  "fags",
  "dumbass",
  "dumbfuck",
  "dumb@ss",
  "dumbshit",
  "motherfucker",
  "mofo",
  "mf",
  "m0therfucker",
  "mfkr",
  "mthrfckr",
  "cocksucker",
  "c0cksucker",
  "cock",
  "cocks",
  "piss",
  "p1ss",
  "pissy",
  "pissoff",
  "pissed",
  "damn",
  "dammit",
  "d@mn",
  "goddamn",
  "crap",
  "cr@p",
  "crappy",
  "sob",
  "s.o.b",
  "wtf",
  "stfu",
  "gtfo",
  "kys",
  "twat",
  "tw@t",
  "tw4t",
  "wanker",
  "w@nker",
  "wank",
  "prick",
  "pr1ck",
  "pricks",
  "douche",
  "douchebag",
  "d0uche",
  "scum",
  "scumbag",
  "scummy",
  "jackass",
  "jack@ss",
  "jackasses",
  "sucker",
  "suck",
  "sucks",
  "sucking",
  "pervert",
  "perverted",
  "perv",
  "sperm",
  "sp3rm",
  "cumshot",
  "cum",

  // Urdu/Punjabi abuse
  "kanjri",
  "kanjari",
  "k@njri",
  "kanjar",
  "kanjr",
  "k@njar",
  "kanjaron",
  "haramzada",
  "haramzadi",
  "h@ramzada",
  "haramzaday",
  "chirkut",
  "ch1rkut",
  "chirkutt",
  "bewakoof",
  "bewkoof",
  "bewak00f",
  "bewaqoof",
  "budtameez",
  "badtameez",
  "b@dtameez",
  "badtmeez",
  "sharabi",
  "sh@rabi",
  "sharaabi",
  "sharabion",
  "chamar",
  "ch@mar",
  "chamaar",
  "bhangi",
  "bh@ngi",
  "bhangee",
  "churra",
  "chura",
  "chuhra",
  "ghashti",
  "gh@shti",
  "gashti",

  // Tamil/Telugu abuse (transliterated)
  "punda",
  "pundai",
  "p*nda",
  "pundek",
  "pundamavan",
  "otha",
  "0tha",
  "oththa",
  "oththaa",
  "sunni",
  "sunn1",
  "sunniya",
  "poolu",
  "p00lu",
  "pooluthunai",
  "dengu",
  "d3ngu",
  "dengey",
  "denginaa",
  "lanja",
  "l@nja",
  "lanjaa",
  "lanjakoduku",
  "munda",
  "mundaa",
  "mundacode",
  "yerri",
  "y3rri",
  "yerruku",
  "pichhi",
  "p1chhi",
  "pichhipuka",
  "gudda",
  "guddaa",
  "guddalo",
  "modda",
  "m0dda",
  "moddala",
  "nakka",
  "n@kka",
  "nakkuu",
  "naakoduku",
  "naakodaka",

  // Bengali abuse (transliterated)
  "bokachoda",
  "bok@choda",
  "bokaa",
  "bokal",
  "chagol",
  "ch@gol",
  "chagole",
  "magir",
  "magi",
  "m@gi",
  "maagir",
  "maggir",
  "baler",
  "bal3r",
  "baal",
  "baaler",
  "shala",
  "sh@la",
  "shalaa",
  "shalay",
  "guimail",
  "gu1mail",
  "guimaile",
  "khanki",
  "kh@nki",
  "khankir",
  "nangta",
  "n@ngta",
  "nengta",
  "haramjada",
  "h@ramjada",
  "haramjaadi",
  "chudir",
  "chud1r",
  "chudirbhai",

  // Marathi abuse
  "zavnya",
  "z@vnya",
  "zavnyaa",
  "ghalat",
  "gh@lat",
  "ghalatpana",
  "bhadvya",
  "bh@dvya",
  "bhadavya",
  "aai",
  "aaichi",
  "aaicha",
  "aaila",
  "yedya",
  "y3dya",
  "yedyaa",
  "satakli",
  "s@takli",
  "satak",
  "shembdya",
  "sh3mbdya",
  "shembda",

  // Common symbol variations
  "@ss",
  "a$$",
  "@$$",
  "f@ck",
  "sh!t",
  "b!tch",
  "d!ck",
  "pr!ck",
  "c0ck",

  // Leetspeak variations
  "5h1t",
  "b17ch",
  "d1ck",
  "4ss",
  "f4ck",
  "pu55y",
  "a55",
  "c0ck",
  "fvck",
])

// Additional pattern-based detection for creative spellings
const ABUSE_PATTERNS = [
  /m+[a@4]+d+[ae@4]*r+\s*c+h+[ou0]*d/i, // madarchod variations
  /b+h+[ou0]+s+[dkt]+[ie1]*/i, // bhosdike variations
  /b+[eh3]+n+\s*c+h+[ou0]*d/i, // behenchod variations
  /c+h+[uo0]+t+[iy1]+[ae@4]*/i, // chutiya variations
  /g+[ae@4]+n+d+[uo0]*/i, // gandu variations
  /l+[aou0@4]+n+d+/i, // lund/lauda variations
  /r+[ae@4]+n+d+[iy1]*/i, // randi variations
  /f+[uo0]+c+k+/i, // fuck variations
  /s+h+[i1!]+t+/i, // shit variations
  /b+[i1!]+t+c+h+/i, // bitch variations
  /a+[s$5]+[s$5]+h+[o0]+l+e*/i, // asshole variations
  /m+[o0]+t+h+e+r+\s*f+/i, // motherfucker variations
  /n+[i1!]+g+[g@]+[ae@4]+r*/i, // n-word variations
  /c+[o0]+c+k+/i, // cock variations
  /d+[i1!]+c+k+/i, // dick variations
  /p+[u]+s+[s$5]+y+/i, // pussy variations
  /w+h+[o0]+r+e*/i, // whore variations
  /s+l+[u]+t+/i, // slut variations
  /h+[a@4]+r+[a@4]+m+[iz1]+/i, // harami variations
  /k+[a@4]+m+[i1!]+n+[a@4e3]*/i, // kamina variations
  /t+[e3]+r+[i1!]+\s*m+[a@4]+/i, // teri maa variations
  /t+[e3]+r+[a@4]+\s*b+[a@4]+[a@4]+p+/i, // tera baap variations
  /l+[a@4]+[u0]+d+[a@4e3]*/i, // lauda/laude variations
  /b+h+[a@4]+d+[vw]+[a@4e3]*/i, // bhadwa variations
  /c+h+[a@4]+k+k+[a@4]*/i, // chakka variations
  /h+[i1!]+j+[r]+[a@4e3]*/i, // hijra variations
]

// Normalize text for comparison - handles leetspeak and obfuscation
function normalizeText(text: string): string {
  return text
    .toLowerCase()
    .replace(/0/g, "o")
    .replace(/1/g, "i")
    .replace(/3/g, "e")
    .replace(/4/g, "a")
    .replace(/5/g, "s")
    .replace(/7/g, "t")
    .replace(/8/g, "b")
    .replace(/@/g, "a")
    .replace(/\$/g, "s")
    .replace(/\*/g, "")
    .replace(/\+/g, "t")
    .replace(/!/g, "i")
    .replace(/\|/g, "l")
    .replace(/[.,!?;:'"()[\]{}]/g, "")
    .replace(/\s+/g, " ")
    .trim()
}

// Check if text contains abuse
function containsAbuse(text: string): boolean {
  if (!text) return false

  // Normalize text
  const normalized = normalizeText(text)
  const original = text.toLowerCase().trim()

  // Split into words and check each
  const words = normalized.split(/\s+/)
  for (const word of words) {
    if (ABUSE_WORDS.has(word)) {
      return true
    }
    // Also check 2-word combinations for phrases like "teri maa"
    const idx = words.indexOf(word)
    if (idx < words.length - 1) {
      const twoWord = `${word} ${words[idx + 1]}`
      if (ABUSE_WORDS.has(twoWord)) {
        return true
      }
    }
  }

  // Also check original words (without normalization)
  const originalWords = original.split(/\s+/)
  for (const word of originalWords) {
    const cleaned = word.replace(/[.,!?;:'"()[\]{}]/g, "")
    if (ABUSE_WORDS.has(cleaned)) {
      return true
    }
  }

  // Check pattern-based detection on both normalized and original
  for (const pattern of ABUSE_PATTERNS) {
    if (pattern.test(normalized) || pattern.test(original)) {
      return true
    }
  }

  // Check for space-separated letters trying to bypass (like "f u c k")
  const spacedLetters = normalized.replace(/\s/g, "")
  if (spacedLetters.length <= 20) {
    // Only for short text to avoid false positives
    for (const word of ABUSE_WORDS) {
      if (word.length >= 3 && spacedLetters.includes(word)) {
        return true
      }
    }
  }

  return false
}

// Check antiabuse settings for a chat
async function isAntiAbuseEnabled(chatId: number): Promise<boolean> {
  const { data } = await supabase.from("chat_settings").select("antiabuse_enabled").eq("chat_id", chatId).maybeSingle()

  return data?.antiabuse_enabled === true
}

// Main function to check abuse in messages - NO ADMIN BYPASS
export async function checkAbuse(message: TelegramMessage, bot: TelegramBot): Promise<{ violated: boolean }> {
  // Only check in groups
  if (message.chat.type === "private") {
    return { violated: false }
  }

  const chatId = message.chat.id
  const userId = message.from?.id

  if (!userId) {
    return { violated: false }
  }

  // Check if antiabuse is enabled for this chat
  const enabled = await isAntiAbuseEnabled(chatId)
  if (!enabled) {
    return { violated: false }
  }

  // Get text from message (text, caption, or forwarded text)
  const text = message.text || message.caption || ""

  if (!text) {
    return { violated: false }
  }

  // Check for abuse
  if (containsAbuse(text)) {
    // Delete message and warn user
    try {
      await Promise.all([
        bot.deleteMessage(chatId, message.message_id),
        bot.sendMessage(
          chatId,
          `⚠️ <b>Warning:</b> <a href="tg://user?id=${userId}">${message.from?.first_name || "User"}</a>, abusive language is strictly prohibited!\n\n<i>This rule applies to everyone - including admins.</i>`,
          { parse_mode: "HTML" },
        ),
      ])
    } catch {
      // Ignore errors (message might already be deleted)
    }

    return { violated: true }
  }

  return { violated: false }
}

// Handle /antiabuse command
export async function handleAntiAbuse(ctx: CommandContext): Promise<void> {
  if (ctx.chat.type === "private") {
    await ctx.bot.sendMessage(ctx.chat.id, "This command only works in groups.")
    return
  }

  // Only admins and owners can change this setting
  if (!ctx.isAdmin && !ctx.isOwner) {
    await ctx.bot.sendMessage(ctx.chat.id, "You need to be an admin to use this command.")
    return
  }

  const arg = ctx.args[0]?.toLowerCase()

  // Check current status if no args
  if (!arg) {
    const enabled = await isAntiAbuseEnabled(ctx.chat.id)
    await ctx.bot.sendMessage(
      ctx.chat.id,
      `<b>🛡️ Antiabuse Status</b>\n\nCurrently: ${enabled ? "✅ Enabled" : "❌ Disabled"}\n\n<i>When enabled, abusive messages from <b>everyone</b> (including admins) will be deleted.</i>\n\nUse <code>/antiabuse on</code> or <code>/antiabuse off</code> to change.`,
      { parse_mode: "HTML" },
    )
    return
  }

  if (arg !== "on" && arg !== "off") {
    await ctx.bot.sendMessage(
      ctx.chat.id,
      "Usage: /antiabuse [on/off]\n\nExample:\n• /antiabuse on - Enable abuse detection\n• /antiabuse off - Disable abuse detection",
    )
    return
  }

  const enabled = arg === "on"

  // Update database
  const { error } = await supabase
    .from("chat_settings")
    .upsert({ chat_id: ctx.chat.id, antiabuse_enabled: enabled }, { onConflict: "chat_id" })

  if (error) {
    await ctx.bot.sendMessage(ctx.chat.id, "Failed to update antiabuse setting. Please try again.")
    return
  }

  await ctx.bot.sendMessage(
    ctx.chat.id,
    enabled
      ? "✅ <b>Antiabuse enabled!</b>\n\nMessages containing abusive language will be automatically deleted.\n\n<i>⚠️ This applies to <b>everyone</b> - including admins and owners!</i>"
      : "❌ <b>Antiabuse disabled!</b>\n\nAbuse detection is now turned off.",
    { parse_mode: "HTML" },
  )
}
