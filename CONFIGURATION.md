# Configuration

## Source Of Truth

The canonical env template is [bot-core/.env.example](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/bot-core/.env.example).

Recommended production template for Option 1:

- [bot-core/.env.vps-external-postgres.example](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/bot-core/.env.vps-external-postgres.example)

Recommended production storage layout:

- `DATABASE_URL` -> external Postgres
- `MIGRATE_DATABASE_URL` -> direct Postgres URL for migrations when available
- `REDIS_ADDR` or `REDIS_URL` -> local VPS Valkey/Redis

## Required Variables

`DATABASE_URL`
- PostgreSQL connection string used by the runtime
- this is the primary durable system of record
- for Neon-compatible production deployments, this should point at the external Postgres runtime URL
- when using Neon connection pooling, this can be the pooled runtime URL

`MIGRATE_DATABASE_URL`
- optional override used by `./cmd/migrate`
- if unset, the migration command falls back to `DATABASE_URL`
- recommended when your provider exposes separate pooled and direct Postgres URLs
- for Neon-compatible deployments, prefer the direct non-pooled Postgres URL here

`REDIS_ADDR` or `REDIS_URL`
- Valkey or Redis connection target
- keep this local to the VPS by default for hot-path speed

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
- default is `100ms` so webhook-queued updates feel responsive without waiting a full second between idle polls

## Database Pool Tuning

`DATABASE_MAX_CONNS`
- runtime Postgres pool upper bound
- default `10`

`DATABASE_MIN_CONNS`
- runtime Postgres pool lower bound
- default `2`

`DATABASE_MAX_CONN_LIFETIME`
- maximum connection lifetime
- default `1h`

`DATABASE_MAX_CONN_IDLE_TIME`
- maximum idle lifetime
- default `15m`

These settings are optional. The defaults match current runtime behavior.

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

Startup validation now fails clearly when:

- `DATABASE_URL` is missing or malformed
- `MIGRATE_DATABASE_URL` is malformed
- `REDIS_URL` is malformed
- `REDIS_ADDR` is not in `host:port` form
- worker or database pool tuning values are invalid

## External Postgres Guidance

For Neon-compatible external Postgres:

- use a provider URL with TLS enabled
- keep the Postgres region as close as possible to the VPS region
- use `MIGRATE_DATABASE_URL` for a direct connection when your provider documents a separate direct endpoint

The runtime does not assume Postgres is local. It only requires a valid reachable `DATABASE_URL`.

## Redis Guidance

Recommended default:

- keep Valkey/Redis on the same VPS as the Go runtime
- use it only for webhook cache, flood counters, join-burst counters, and short leases

Do not treat Redis as the durable source of truth for Sukoon state.

## Security Guidance

- keep webhook keys unguessable
- keep webhook secrets distinct per bot instance
- never commit `.env` files
- rotate bot tokens and webhook secrets together if compromise is suspected
- do not reuse production database or Redis credentials in integration tests
