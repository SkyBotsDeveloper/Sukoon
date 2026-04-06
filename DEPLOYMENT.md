# Deployment

## Supported Targets

- VPS with Docker Compose
- Railway
- Heroku

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

## VPS

Use [docker-compose.yml](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/docker-compose.yml).

1. Create `bot-core/.env` from the example.
2. Fill production secrets and URLs.
3. Start dependencies:

```powershell
docker compose up -d postgres valkey
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
- `REDIS_URL` or `REDIS_ADDR`
- `PUBLIC_WEBHOOK_BASE_URL`
- `PRIMARY_BOT_TOKEN`
- `PRIMARY_BOT_WEBHOOK_KEY`
- `PRIMARY_BOT_WEBHOOK_SECRET`
- `PRIMARY_BOT_USERNAME`
- `BOT_OWNER_USER_IDS`

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

## Telegram Webhook Setup

After the web process is reachable publicly, register the webhook:

```text
https://api.telegram.org/bot<PRIMARY_BOT_TOKEN>/setWebhook
```

with:

- `url=https://your-domain.example/webhook/<PRIMARY_BOT_WEBHOOK_KEY>`
- `secret_token=<PRIMARY_BOT_WEBHOOK_SECRET>`

The same base URL is used for clone webhook registration.
