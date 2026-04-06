# Configuration

## Source Of Truth

The canonical env template is [bot-core/.env.example](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/bot-core/.env.example).

## Required Variables

`DATABASE_URL`
- PostgreSQL connection string used by the runtime and migration command

`REDIS_ADDR` or `REDIS_URL`
- Valkey or Redis connection target

`PUBLIC_WEBHOOK_BASE_URL`
- public base URL used for webhook registration, for example `https://bot.example.com`

`PRIMARY_BOT_TOKEN`
- Telegram token for the primary bot

`PRIMARY_BOT_WEBHOOK_KEY`
- opaque path segment used in `/webhook/<key>`

`PRIMARY_BOT_WEBHOOK_SECRET`
- Telegram webhook secret token

`PRIMARY_BOT_USERNAME`
- bot username without URL

`BOT_OWNER_USER_IDS`
- comma-separated Telegram user IDs with owner access

## Runtime Variables

`APP_ENV`
- `development` or `production`

`APP_MODE`
- `all`
- `web`
- `worker`

`APP_ADDR`
- listen address for the HTTP ingress, defaults to `:8080`

`PORT`
- fallback source for `APP_ADDR` on platforms like Heroku

## Worker Tuning

`WORKER_CONCURRENCY`
- number of worker goroutines

`WORKER_POLL_INTERVAL`
- polling interval for update and job claims

## Telegram Client Tuning

`TELEGRAM_BASE_URL`
- defaults to `https://api.telegram.org`

`TELEGRAM_REQUEST_TIMEOUT`
- per-request timeout

`TELEGRAM_MAX_RETRIES`
- retry count for Telegram API failures

`TELEGRAM_INITIAL_BACKOFF`
- first retry backoff duration

## Redis Resolution Rules

If `REDIS_URL` is present, the runtime derives:

- host
- password
- database index

Otherwise it uses:

- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`

## Security Guidance

- keep webhook keys unguessable
- keep webhook secrets distinct per bot instance
- never commit `.env` files
- rotate bot tokens and webhook secrets together if compromise is suspected
- do not reuse production database or Redis credentials in integration tests
