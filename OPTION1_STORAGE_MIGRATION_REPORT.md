# Option 1 Storage Migration Report

## 1. What Changed

- Added explicit support for a split database model:
  - `DATABASE_URL` for runtime Postgres connections
  - `MIGRATE_DATABASE_URL` for migration/admin connections when a direct URL is available
- Tightened config validation so malformed database and Redis settings fail fast at startup.
- Added optional Postgres pool tuning env vars:
  - `DATABASE_MAX_CONNS`
  - `DATABASE_MIN_CONNS`
  - `DATABASE_MAX_CONN_LIFETIME`
  - `DATABASE_MAX_CONN_IDLE_TIME`
- Updated `bot-core` startup so migrations can use `MIGRATE_DATABASE_URL` while the runtime still uses `DATABASE_URL`.
- Updated Docker Compose so it no longer hardcodes a local Postgres URL into the runtime containers.
- Added a VPS-focused env template for external Postgres + local Valkey:
  - `bot-core/.env.vps-external-postgres.example`
- Updated canonical docs to make external Postgres + local VPS Redis the recommended production model.

## 2. What Did NOT Change

- The Go runtime architecture did not change.
- Webhook ingress and worker execution remain separate modes under `APP_MODE`.
- PostgreSQL remains the durable system of record.
- Redis/Valkey remains the hot-state layer.
- No Supabase or legacy storage path was reintroduced.
- No queue model redesign was done.
- No remote Redis default was introduced.

## 3. How External Postgres Is Now Supported

- The runtime accepts any valid external Postgres URL via `DATABASE_URL`.
- Startup validation now rejects malformed Postgres URLs immediately.
- The migration command supports a separate `MIGRATE_DATABASE_URL`, which is useful for providers that expose a direct URL for schema/admin work.
- For Neon-compatible providers, this supports a pooled runtime URL plus a direct migration URL without changing the bot architecture.
- `bot-core` startup respects that same split, so startup migrations can run against the migration URL while runtime queries use the runtime URL.
- Docker Compose now defers to `bot-core/.env` instead of forcing a local container-only Postgres URL.

## 4. How Local Redis Remains In Use

- Redis is still configured through `REDIS_ADDR` or `REDIS_URL`.
- The recommended production model keeps Redis local to the VPS.
- Redis is still used only for:
  - webhook bot cache
  - antiflood counters
  - antiraid join-burst counters
  - short leases

## 5. Which Files Changed

- `bot-core/internal/config/config.go`
- `bot-core/internal/config/config_test.go`
- `bot-core/internal/app/app.go`
- `bot-core/cmd/migrate/main.go`
- `bot-core/internal/persistence/postgres/store.go`
- `bot-core/.env.example`
- `bot-core/.env.vps-external-postgres.example`
- `docker-compose.yml`
- `.gitignore`
- `README.md`
- `CONFIGURATION.md`
- `DEPLOYMENT.md`
- `OPERATIONS.md`

## 6. Risks Still Remaining

- External Postgres latency still affects:
  - `telegram_updates`
  - `jobs`
  - `LoadRuntimeBundle`
  - filter reads on text-message paths
  - approvals/global blacklist checks
- No major hot-path query refactor was done in this phase.
- Redis remains local-first by recommendation; moving it remote would still increase latency on moderation paths.

## 7. Exact Operator Steps For Neon/External Postgres + Local VPS Redis

1. Create an external Postgres project in the nearest possible region to the VPS.
2. Copy `bot-core/.env.vps-external-postgres.example` to `bot-core/.env`.
3. Set:
   - `DATABASE_URL` to the runtime Postgres URL
   - `MIGRATE_DATABASE_URL` to the direct Postgres URL if the provider gives one
   - `REDIS_ADDR` to `valkey:6379` for Docker Compose, or `127.0.0.1:6379` if running the Go binary directly on the host
   - Telegram and webhook secrets
4. Start local Valkey on the VPS.
5. Run migrations.
6. Start `bot-core-web` and `bot-core-worker`.
7. Verify `/healthz`.
8. Verify Telegram webhook registration.

## 8. Test/Build Results

Validation completed:

- `go test ./...` passed
- `go test -tags=integration ./...` passed
- `go build ./cmd/bot-core` passed
- `go build ./cmd/migrate` passed

Environment note:

- plain `go build` initially hit a Windows `go-build` cache permission error in this environment
- rerunning with a workspace-local `GOCACHE` succeeded
- this was not a Sukoon code issue
