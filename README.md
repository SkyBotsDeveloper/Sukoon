# Sukoon - Telegram Group Management Bot

A powerful, industry-grade Telegram group management bot inspired by Miss Rose, built with Next.js 16 and Supabase. Features comprehensive moderation tools, anti-spam protection, federations, and much more.

## Features

### Moderation
| Command | Description |
|---------|-------------|
| `/ban`, `/tban`, `/dban`, `/sban` | Ban users (timed, delete msg, silent) |
| `/unban` | Unban a user |
| `/mute`, `/tmute`, `/dmute`, `/smute` | Mute users (timed, delete msg, silent) |
| `/unmute` | Unmute a user |
| `/kick`, `/dkick`, `/skick` | Kick users |
| `/warn`, `/dwarn` | Warn users |
| `/warns`, `/resetwarns` | View/reset warnings |
| `/setwarnlimit`, `/setwarnmode` | Configure warnings |

### Anti-Spam
| Command | Description |
|---------|-------------|
| `/lock`, `/unlock`, `/locks` | Lock media, stickers, URLs, forwards, etc. |
| `/blocklist`, `/addblock`, `/unblock` | Manage blocklisted words |
| `/blocklistmode` | Set blocklist action (delete/ban/mute/kick) |
| `/setflood`, `/flood`, `/setfloodmode` | Configure antiflood |
| `/clearflood` | Toggle flood message deletion |
| `/antiraid`, `/raidtime`, `/raidmode` | AntiRaid protection |

### Content Management
| Command | Description |
|---------|-------------|
| `/save`, `/get`, `/notes`, `/clear` | Save and retrieve notes |
| `#notename` | Quick note access |
| `/filter`, `/filters`, `/stop` | Auto-reply filters (supports all media types) |
| `/setwelcome`, `/welcome` | Welcome messages |
| `/setgoodbye`, `/goodbye` | Goodbye messages |
| `/setrules`, `/rules`, `/clearrules` | Group rules |

### Federations
| Command | Description |
|---------|-------------|
| `/newfed`, `/delfed` | Create/delete federation |
| `/joinfed`, `/leavefed` | Join/leave federation |
| `/fban`, `/unfban` | Federation ban/unban |
| `/fedinfo`, `/fedadmins`, `/myfeds` | Federation info |
| `/fedpromote`, `/feddemote` | Manage fed admins |

### Admin Tools
| Command | Description |
|---------|-------------|
| `/pin`, `/unpin`, `/unpinall` | Pin messages |
| `/purge`, `/del` | Delete messages |
| `/approve`, `/unapprove`, `/approved` | Approve users (bypass restrictions) |
| `/disable`, `/enable`, `/disabled` | Disable commands |
| `/setlog`, `/unsetlog` | Log channel |
| `/promote`, `/demote`, `/title` | Manage admins |
| `/adminlist`, `/admin` | List/call admins |
| `/report` | Report to admins |

### Silent Power (Hidden Admins)
| Command | Description |
|---------|-------------|
| `/mod`, `/unmod` | Give/remove full silent powers |
| `/muter`, `/unmuter` | Give/remove mute power only |
| `/mods` | List silent mods |

### BioCheck
| Command | Description |
|---------|-------------|
| `/antibio on/off` | Toggle bio link detection |
| `/free`, `/unfree` | Exempt users from bio check |
| `/freelist` | List exempt users |

### Antiabuse
| Command | Description |
|---------|-------------|
| `/antiabuse on/off` | Toggle abuse detection (500+ words, Hinglish focused) |

### Bot Cloning
| Command | Description |
|---------|-------------|
| `/clone [token]` | Clone Sukoon with your bot token |
| `/clones` | List your clones |
| `/rmclone @username` | Remove a clone |

### Misc
| Command | Description |
|---------|-------------|
| `/start`, `/help` | Bot info and help |
| `/id`, `/info` | User/chat info |
| `/afk`, `/brb` | Set AFK status |
| `/setlang`, `/language` | Change language |

### Owner Commands (Hidden)
| Command | Description |
|---------|-------------|
| `/gban`, `/ungban` | Global ban across all chats |
| `/broadcast` | Send message to all chats/users |
| `/blchat`, `/unblchat` | Blacklist chats |
| `/bluser`, `/unbluser` | Blacklist users |
| `/stats` | Bot statistics |
| `/addsudo`, `/rmsudo` | Manage sudo users |

---

## Deployment

### Prerequisites
1. Create a Telegram bot via [@BotFather](https://t.me/BotFather)
2. Set up a Supabase project at [supabase.com](https://supabase.com)
3. Node.js 18+ installed

### Environment Variables

```env
# Required
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_WEBHOOK_SECRET=random_secret_string_32_chars
BOT_USERNAME=YourBotUsername

# Supabase
NEXT_PUBLIC_SUPABASE_URL=https://xxxxx.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=your_anon_key
SUPABASE_SERVICE_ROLE_KEY=your_service_role_key

# App URL (for webhook)
NEXT_PUBLIC_APP_URL=https://your-domain.com
```

---

## Deploy to Vercel (Recommended)

1. Click "Publish" button in v0 or fork this repository
2. Import to [Vercel](https://vercel.com)
3. Add environment variables in Project Settings
4. Deploy
5. Visit `/api/telegram/setup` to set webhook

---

## Deploy to Ubuntu VPS (Step-by-Step)

### 1. Server Setup

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Node.js 20
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install PM2 globally
sudo npm install -g pm2

# Install Nginx (optional, for domain)
sudo apt install -y nginx
```

### 2. Clone and Setup

```bash
# Create app directory
mkdir -p /var/www/sukoon
cd /var/www/sukoon

# Clone repository (or upload files)
git clone https://github.com/yourusername/sukoon-bot.git .

# Install dependencies
npm install

# Create environment file
nano .env
```

Add your environment variables to `.env`:

```env
TELEGRAM_BOT_TOKEN=your_token
TELEGRAM_WEBHOOK_SECRET=your_secret
BOT_USERNAME=YourBot
NEXT_PUBLIC_SUPABASE_URL=https://xxx.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=your_anon_key
SUPABASE_SERVICE_ROLE_KEY=your_service_key
NEXT_PUBLIC_APP_URL=https://your-domain.com
```

### 3. Build and Start

```bash
# Build the application
npm run build

# Start with PM2
pm2 start npm --name "sukoon" -- start

# Save PM2 config
pm2 save

# Setup PM2 to start on boot
pm2 startup
```

### 4. Nginx Configuration (with SSL)

```bash
# Create Nginx config
sudo nano /etc/nginx/sites-available/sukoon
```

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 86400;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/sukoon /etc/nginx/sites-enabled/

# Test and reload Nginx
sudo nginx -t
sudo systemctl reload nginx

# Install SSL with Certbot
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

### 5. Set Webhook

```bash
# Set webhook via curl
curl -X POST https://your-domain.com/api/telegram/setup
```

### 6. PM2 Monitoring Commands

```bash
pm2 status          # Check status
pm2 logs sukoon     # View logs
pm2 restart sukoon  # Restart bot
pm2 stop sukoon     # Stop bot
pm2 monit           # Real-time monitoring
```

---

## Deploy with Docker

### Using Docker Compose

```bash
# Clone repository
git clone https://github.com/yourusername/sukoon-bot.git
cd sukoon-bot

# Create .env file with your variables
nano .env

# Start with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f
```

### Manual Docker Build

```bash
# Build image
docker build -t sukoon-bot .

# Run container
docker run -d \
  --name sukoon \
  --restart unless-stopped \
  -p 3000:3000 \
  --env-file .env \
  sukoon-bot
```

---

## Database Setup

Run all SQL scripts in order in your Supabase SQL Editor:

1. `scripts/001_create_database_schema.sql` - Core tables
2. `scripts/010_create_silent_mods_table.sql` - Silent power feature
3. `scripts/011_create_blacklist_tables.sql` - Blacklist feature
4. `scripts/012_create_antibio_tables.sql` - Bio check feature
5. `scripts/013_add_antiabuse_column.sql` - Antiabuse feature

---

## Message Formatting

### Text Formatting
- `*bold*` → **bold**
- `_italic_` → _italic_
- `` `code` `` → `code`
- `[text](URL)` → hyperlink

### Button Syntax
```
[Button Text](buttonurl://example.com)
[Same Row](buttonurl://example.com:same)
```

### Variables
| Variable | Description |
|----------|-------------|
| `{first}` | First name |
| `{last}` | Last name |
| `{fullname}` | Full name |
| `{username}` | @username |
| `{mention}` | Mention user |
| `{id}` | User ID |
| `{chatname}` | Chat name |
| `{chatid}` | Chat ID |

---

## Performance Optimizations

- **In-memory caching** for admin status, flood tracking, and settings
- **Parallel database queries** using Promise.all()
- **Non-blocking operations** for logging and analytics
- **Connection pooling** via Supabase
- **Optimized regex patterns** for abuse detection

---

## Security Features

- Webhook secret verification
- Clone token validation with caching
- Rate limiting via antiflood
- Global ban system
- Chat/user blacklisting
- Federation ban propagation

---

## Support

- Documentation: [misssukoon.vercel.app](https://misssukoon.vercel.app)
- Support Group: [@VivaanSupport](https://t.me/VivaanSupport)
- Updates Channel: [@VivaanUpdates](https://t.me/VivaanUpdates)

---

## License

MIT License - Feel free to use, modify, and distribute.
