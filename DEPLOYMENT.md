# Deployment

## Supported Targets

- VPS with Docker Compose
- Railway
- Heroku

## Primary Recommendation

Recommended production model for Sukoon:

- external Postgres as the primary durable database
- local VPS Valkey/Redis for hot-path state
- VPS-hosted `bot-core-web` and `bot-core-worker`

This keeps durable product state outside the VPS while preserving local Redis latency for:

- webhook bot cache
- antiflood counters
- antiraid join-burst counters
- short leases used by moderation flows

## What Was Validated Directly

Validated in this repository:

- Go build for `bot-core`
- Go build for `migrate`
- runtime mode handling in config and startup code
- Railway config files present and consistent with the Go runtime
- Heroku container config present and consistent with the Go runtime
- canonical Docker Compose stack present and consistent with the Go runtime
- health endpoint path `/healthz`

Not validated live here:

- real VPS deployment with Docker
- real Railway deployment
- real Heroku deployment
- live webhook registration against Telegram

Those remaining checks require Docker and platform credentials not available in this environment.

## Beginner VPS Guide: Neon + Local Valkey

This is the recommended production setup if you want:

- durable data to survive VPS loss
- fast local Redis for hot-path moderation state
- the bot runtime to stay on your own VPS

### What You Need Before You Start

Prepare these values first. Do not paste real secrets into chats, screenshots, GitHub, or public notes.

| Value | Where it comes from | Example |
| --- | --- | --- |
| `DATABASE_URL` | Neon runtime connection string | `postgresql://user:pass@ep-...-pooler.../sukoon?sslmode=require` |
| `MIGRATE_DATABASE_URL` | Neon direct connection string | `postgresql://user:pass@ep-.../.../sukoon?sslmode=require` |
| `PUBLIC_WEBHOOK_BASE_URL` | your VPS domain or public HTTPS URL | `https://bot.example.com` |
| `PRIMARY_BOT_TOKEN` | BotFather | `123456:ABC...` |
| `PRIMARY_BOT_USERNAME` | BotFather | `MySukoonBot` |
| `PRIMARY_BOT_WEBHOOK_KEY` | you create this yourself | long random string |
| `PRIMARY_BOT_WEBHOOK_SECRET` | you create this yourself | long random string |
| `BOT_OWNER_USER_IDS` | your Telegram numeric user ID(s) | `123456789` |

### How To Choose The Correct Neon URLs

- `DATABASE_URL`
  - use the runtime URL the bot should use every day
  - with Neon, a pooled URL is a good default for runtime traffic

- `MIGRATE_DATABASE_URL`
  - use the direct non-pooled URL when Neon provides one
  - use this for migrations and admin-style DB work

- SSL/TLS
  - Neon requires TLS
  - keep `sslmode=require` at minimum unless you intentionally use a stricter mode

- Region
  - choose the Neon region nearest to your VPS region
  - this matters because Sukoon does real Postgres work on update and job paths

### VPS Prerequisites

On a fresh Ubuntu/Debian-style VPS you need:

- Docker Engine
- Docker Compose plugin
- Git
- a public domain or public HTTPS endpoint for Telegram webhook delivery
- ports `80` and `443` open to the VPS if you are using a reverse proxy

If Docker and Git are not installed yet, install them first using your distro's official instructions.

### Step 1: Create The Neon Project

1. Create a Neon project.
2. Create or choose the target database name.
3. Copy the runtime connection string.
4. Copy the direct connection string if Neon shows a separate direct endpoint.
5. Keep both strings private.

### Step 2: Connect To The VPS

```bash
ssh <your-user>@<your-vps-ip>
```

### Step 3: Clone The Repo

```bash
git clone https://github.com/SkyBotsDeveloper/Sukoon.git
cd Sukoon
```

### Step 4: Create `bot-core/.env`

Start from the production-oriented template:

```bash
cp bot-core/.env.vps-external-postgres.example bot-core/.env
```

Then edit it:

```bash
nano bot-core/.env
```

Replace the placeholder values.

Minimum fields you must fill:

- `DATABASE_URL`
- `MIGRATE_DATABASE_URL` if you have a direct Neon URL
- `PUBLIC_WEBHOOK_BASE_URL`
- `PRIMARY_BOT_TOKEN`
- `PRIMARY_BOT_WEBHOOK_KEY`
- `PRIMARY_BOT_WEBHOOK_SECRET`
- `PRIMARY_BOT_USERNAME`
- `BOT_OWNER_USER_IDS`

Important:

- if you are using Docker Compose, keep `REDIS_ADDR=valkey:6379`
- if you are not using Docker Compose and are running the Go binary directly on the host, use `REDIS_ADDR=127.0.0.1:6379`

### Step 5: Start Local Valkey On The VPS

```bash
docker compose up -d valkey
docker compose ps
```

You should see the `valkey` service running.

### Step 6: Run Migrations

```bash
docker compose run --rm migrate
```

This should exit successfully. If it fails, do not continue to the bot runtime until the migration error is fixed.

### Step 7: Start The Web And Worker Processes

```bash
docker compose up -d bot-core-web bot-core-worker
docker compose ps
```

You should see:

- `bot-core-web`
- `bot-core-worker`
- `valkey`

### Step 8: Check Health And Logs

Health check from the VPS:

```bash
curl -fsS http://127.0.0.1:8080/healthz
```

Follow logs:

```bash
docker compose logs -f bot-core-web
```

```bash
docker compose logs -f bot-core-worker
```

### Step 9: Put HTTPS In Front Of Port 8080

Telegram webhooks need a public HTTPS endpoint.

Your reverse proxy must forward traffic to:

- `http://127.0.0.1:8080`

Minimum requirements:

- HTTPS enabled
- webhook path forwarded without auth prompts
- POST requests and request body preserved
- no extra middleware that blocks Telegram

If you use Caddy, the important idea is:

- public domain -> reverse proxy -> `127.0.0.1:8080`

If you use Nginx, the important idea is the same:

- public domain -> `proxy_pass http://127.0.0.1:8080;`

Before setting the webhook, make sure this works in a browser or with curl:

```bash
curl -I https://<your-domain>/healthz
```

### Step 10: Set The Telegram Webhook

Replace the placeholders and run:

```bash
curl -sS "https://api.telegram.org/bot<PRIMARY_BOT_TOKEN>/setWebhook" \
  -d "url=https://<your-domain>/webhook/<PRIMARY_BOT_WEBHOOK_KEY>" \
  -d "secret_token=<PRIMARY_BOT_WEBHOOK_SECRET>"
```

Then verify:

```bash
curl -sS "https://api.telegram.org/bot<PRIMARY_BOT_TOKEN>/getWebhookInfo"
```

Healthy signs:

- `url` is correct
- `pending_update_count` is not growing without being processed
- `last_error_message` is empty

### Step 11: Test The Bot

1. Open Telegram.
2. Send `/start` to the bot in PM.
3. Add the bot to a test group.
4. Send a simple command like `/help`.
5. Watch the logs while you test.

What success looks like:

- webhook is accepted
- worker logs show updates being processed
- the bot replies in Telegram

### Step 12: Safe Restart Later

If you only changed env vars or pulled a new image/build:

```bash
docker compose up -d --build bot-core-web bot-core-worker
```

If you pulled new code:

```bash
git pull origin main
docker compose build --no-cache
docker compose run --rm migrate
docker compose up -d bot-core-web bot-core-worker
docker compose ps
```

If something looks wrong after an update, inspect logs before repeating commands blindly.

## VPS

Use [docker-compose.yml](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/docker-compose.yml).

### Option 1: External Postgres + Local Valkey

1. Create `bot-core/.env` from [bot-core/.env.vps-external-postgres.example](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/bot-core/.env.vps-external-postgres.example).
2. Set:
   - `DATABASE_URL` to the external Postgres runtime URL
   - `MIGRATE_DATABASE_URL` to the provider's direct Postgres URL if available
   - `REDIS_ADDR=valkey:6379` when using the local compose Valkey container
   - Telegram and webhook secrets
3. Start local Valkey:

```powershell
docker compose up -d valkey
```

4. Run migrations:

```powershell
docker compose run --rm migrate
```

5. Start the runtime:

```powershell
docker compose up -d bot-core-web bot-core-worker
```

Health check:

```text
GET /healthz
```

### Optional Local Postgres For Dev Or Fallback

If you want a fully local stack for development or temporary fallback testing:

1. Create `bot-core/.env` from [bot-core/.env.example](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/bot-core/.env.example).
2. Start local Postgres and Valkey:

```powershell
docker compose up -d postgres valkey
```

3. Run migrations:

```powershell
docker compose run --rm migrate
```

4. Start the runtime:

```powershell
docker compose up -d bot-core-web bot-core-worker
```

Health check:

```text
GET /healthz
```

### Neon-Compatible Setup Notes

Neon-compatible operator guidance:

- Neon requires TLS for connections. Use a connection string with `sslmode=require` or a stricter mode such as `verify-full`.
- Use the pooled runtime URL for `DATABASE_URL` when you want Neon connection pooling.
- Use the provider's direct endpoint for `MIGRATE_DATABASE_URL` when available.
- Keep the Neon region as close as possible to the VPS region to reduce queue and message-path latency.
- Sukoon keeps durable state in Postgres, so choose backup/PITR settings on the provider accordingly.

### What Persists Outside The VPS

When using Option 1:

- Postgres durable state persists outside the VPS
  - bot instances and clone ownership
  - chats/users metadata
  - chat settings and moderation state
  - notes, filters, rules, approvals, locks, blocklists
  - federations and global blacklists
  - durable jobs and queued Telegram updates

### What Stays Local On The VPS

- local Valkey/Redis hot state
  - webhook bot cache
  - flood counters
  - join-burst counters
  - short leases

If the VPS is replaced, this Redis state can be lost without losing canonical durable Sukoon data.

## Railway

Files:

- [railway.json](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/railway.json)
- [railway.worker.json](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/railway.worker.json)

Recommended model:

- one web service using `railway.json`
- one worker service using `railway.worker.json`

Web service behavior:

- build from `bot-core/Dockerfile`
- run `migrate` as `preDeployCommand`
- start with `APP_MODE=web`
- health check on `/healthz`

Worker service behavior:

- build from `bot-core/Dockerfile`
- start with `APP_MODE=worker`

Required env vars:

- `DATABASE_URL`
- `MIGRATE_DATABASE_URL` if using a separate direct Postgres URL
- `REDIS_URL` or `REDIS_ADDR`
- `PUBLIC_WEBHOOK_BASE_URL`
- `PRIMARY_BOT_TOKEN`
- `PRIMARY_BOT_WEBHOOK_KEY`
- `PRIMARY_BOT_WEBHOOK_SECRET`
- `PRIMARY_BOT_USERNAME`
- `BOT_OWNER_USER_IDS`

Latency guidance:

- keep Railway's region and the external Postgres region as close as possible
- Redis can be local to Railway only if you provision it there; for VPS-style local Redis, the VPS deployment remains the better fit

## Heroku

File:

- [heroku.yml](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/heroku.yml)

Heroku uses the container stack in this repository.

Process model:

- `release`: `migrate`
- `web`: `APP_MODE=web bot-core`
- `worker`: `APP_MODE=worker bot-core`

Required env vars:

- `DATABASE_URL`
- `MIGRATE_DATABASE_URL` if using a separate direct Postgres URL
- `REDIS_URL`
- `PUBLIC_WEBHOOK_BASE_URL`
- `PRIMARY_BOT_TOKEN`
- `PRIMARY_BOT_WEBHOOK_KEY`
- `PRIMARY_BOT_WEBHOOK_SECRET`
- `PRIMARY_BOT_USERNAME`
- `BOT_OWNER_USER_IDS`

Operator steps:

1. Create an app on the container stack.
2. Set the required config vars.
3. Deploy the repo using `heroku.yml`.
4. Scale:
   `web=1`
   `worker=1`

Operational note:

- Heroku does not naturally match the "external Postgres + local VPS Redis" model because dynos do not give you the same local Redis placement control as a VPS. VPS remains the best-practice deployment if you want Redis close to the runtime.

## Telegram Webhook Setup

After the web process is reachable publicly, register the webhook:

```text
https://api.telegram.org/bot<PRIMARY_BOT_TOKEN>/setWebhook
```

with:

- `url=https://your-domain.example/webhook/<PRIMARY_BOT_WEBHOOK_KEY>`
- `secret_token=<PRIMARY_BOT_WEBHOOK_SECRET>`

The same base URL is used for clone webhook registration.

## Startup Sequence Summary

For the recommended VPS Option 1 deployment:

1. Provision external Postgres.
2. Put runtime URL in `DATABASE_URL`.
3. Put direct admin/migration URL in `MIGRATE_DATABASE_URL` if available.
4. Start local Valkey on the VPS.
5. Run `docker compose run --rm migrate`.
6. Start `bot-core-web` and `bot-core-worker`.
7. Register or confirm the Telegram webhook.

## Troubleshooting

### `DATABASE_URL` is wrong

Symptoms:

- `config error`
- `database error`
- startup exits immediately

Checks:

- make sure the string is a real Postgres URL
- make sure you did not paste spaces or quote marks
- make sure the hostname, database, user, and password are correct

### `MIGRATE_DATABASE_URL` is wrong

Symptoms:

- `docker compose run --rm migrate` fails
- app startup migration fails before the bot starts

Checks:

- if Neon gives you a direct URL, use that here
- if you are unsure, temporarily leave `MIGRATE_DATABASE_URL` empty and rely on `DATABASE_URL`

### SSL / TLS issue

Symptoms:

- Postgres connection fails with SSL-related errors

Checks:

- keep `sslmode=require` for Neon-compatible URLs unless your provider told you to use a stricter mode
- do not remove SSL parameters from the Neon URL

### Redis connection issue

Symptoms:

- startup fails talking to Redis
- webhook cache / moderation hot-state errors in logs

Checks:

- if using Docker Compose, `REDIS_ADDR` should be `valkey:6379`
- make sure `docker compose ps` shows `valkey` is running
- if running directly on the host, use `127.0.0.1:6379`

### `/healthz` is failing

Symptoms:

- `curl http://127.0.0.1:8080/healthz` fails

Checks:

- `docker compose ps`
- `docker compose logs -f bot-core-web`
- make sure port `8080` is reachable inside the VPS

### Webhook is set but the bot is not replying

Checks:

- run:

```bash
curl -sS "https://api.telegram.org/bot<PRIMARY_BOT_TOKEN>/getWebhookInfo"
```

- look for `last_error_message`
- confirm the webhook URL matches your real public domain and webhook key
- confirm your reverse proxy forwards requests to `127.0.0.1:8080`
- confirm the worker is running, not just the web process

### Worker is not processing updates

Checks:

- `docker compose ps`
- `docker compose logs -f bot-core-worker`
- confirm `bot-core-worker` is running
- confirm `DATABASE_URL` points to the same database as the web process

### Caddy/Nginx reverse proxy issue

Symptoms:

- `/healthz` works locally but Telegram cannot deliver updates

Checks:

- confirm HTTPS is active on the public domain
- confirm proxy target is `127.0.0.1:8080`
- confirm webhook POST requests are not blocked by auth, redirect loops, or body-size limits

### Telegram `last_error_message` is not empty

This usually means Telegram can reach your webhook setup enough to report an error.

Common causes:

- wrong domain
- broken TLS certificate
- reverse proxy misroute
- app not listening behind the proxy

### Stale container or image issue

Symptoms:

- you changed code or env values but behavior did not change

Fix:

```bash
docker compose build --no-cache
docker compose up -d bot-core-web bot-core-worker
```

### Migration command is failing

Checks:

- confirm the database credentials are correct
- confirm the DB allows your VPS IP if network allowlists are enabled
- confirm `MIGRATE_DATABASE_URL` is the right URL
- inspect the exact migration error before retrying
