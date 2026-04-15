# Sukoon

Sukoon is a production-oriented Telegram moderation and group-management bot built as a long-running Go service.

The repository now has one canonical runtime:

- Go bot core in `bot-core`
- PostgreSQL for durable data and update/job state
- Valkey or Redis for shared hot state
- Docker-first deployment for VPS, plus Railway and Heroku support

Recommended production storage model:

- external PostgreSQL as the durable system of record
- local VPS Valkey or Redis for hot-path state
- VPS-hosted `bot-core` web and worker processes

The old Next.js and Supabase bot runtime has been removed. This repository is no longer a migration workspace or dual-runtime project.

## Status

Sukoon V2 is ready for self-hosted production use with a durable webhook ingress, worker-based update processing, canonical SQL migrations, clone-safe bot scoping, structured logs, automated tests, and CI validation.

Current product direction focuses on fast moderation ergonomics, strong group protection, and a clean operator experience within the safer Go architecture.

## Highlights

- Fast webhook ack plus async worker execution
- Durable update idempotency by bot and `update_id`
- Reliable moderation primitives: bans, mutes, kicks, warns, approvals, locks with custom modes and allowlists, blocklists, antiflood, captcha
- rich help surfaces for admin, approval, bans, antiflood, blocklists, captcha, clean commands, disabling, locks, log channels, federations, filters, and formatting
- Owner/global tooling with job-backed broadcast and global blacklist controls
- Federation support with canonical V2 storage, owner/admin/user help pages, shared bans, subscriptions, quiet-fed controls, import/export, stats, and log/notification settings
- Safe clone lifecycle with explicit per-bot webhook routing
- Structured note and filter buttons without legacy regex parsing, plus random-content separators and contextual fillings
- Button-driven help and rules flows with PM-first guidance, in-place menu editing, and scoped help subpages for blocklists, locks, federations, filters, and formatting
- Privacy export and delete flows against the canonical schema
- VPS, Railway, and Heroku deployment artifacts

## Quick Start

If you are setting up Sukoon on a VPS for the first time, follow the beginner guide in [DEPLOYMENT.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/DEPLOYMENT.md) first. The recommended production model is:

- external Postgres
- local VPS Valkey/Redis
- VPS-hosted `bot-core-web` and `bot-core-worker`

1. Copy the env template:

```powershell
Copy-Item bot-core\.env.example bot-core\.env
```

2. Fill in the required Telegram, database, Redis, and webhook values.

Production recommendation:

- set `DATABASE_URL` to your external Postgres runtime URL
- set `MIGRATE_DATABASE_URL` to the provider's direct Postgres URL when available
- keep `REDIS_ADDR` pointed at local VPS Valkey/Redis

For a VPS + external Postgres starting point, use:

```powershell
Copy-Item bot-core\.env.vps-external-postgres.example bot-core\.env
```

3. For local development or a local fallback database, start local infrastructure:

```powershell
docker compose up -d postgres valkey
```

For the recommended VPS production model, start only local Valkey and use your external Postgres URL:

```powershell
docker compose up -d valkey
```

4. Run migrations:

```powershell
cd bot-core
$env:GOCACHE="$PWD\.gocache"
$env:GOMODCACHE="$PWD\.gomodcache"
go run ./cmd/migrate
```

5. Start the bot:

```powershell
go run ./cmd/bot-core
```

See [DEPLOYMENT.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/DEPLOYMENT.md) for the full external Postgres + local Redis deployment flow.

## Docs

- [ARCHITECTURE.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/ARCHITECTURE.md)
- [CONFIGURATION.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/CONFIGURATION.md)
- [DEPLOYMENT.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/DEPLOYMENT.md)
- [OPERATIONS.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/OPERATIONS.md)
- [TESTING.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/TESTING.md)
- [MIGRATION.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/MIGRATION.md)
- [FEATURE_STATUS.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/FEATURE_STATUS.md)
- [FINAL_HANDOFF_REPORT.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/FINAL_HANDOFF_REPORT.md)
- [PUSH_READY_CHECKLIST.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/PUSH_READY_CHECKLIST.md)

## Repo Layout

```text
.
|-- bot-core/               # Go runtime, migrations, tests, Dockerfile
|-- .github/workflows/      # CI validation
|-- docker-compose.yml      # Canonical local and VPS stack
|-- heroku.yml              # Heroku container deployment
|-- railway.json            # Railway web service config
`-- railway.worker.json     # Railway worker service config
```

## Validation

Validated locally in this repo:

- `gofmt -w ./cmd ./internal ./migrations`
- `go test ./...`
- `go test -tags=integration ./...`
- `go build ./cmd/bot-core`
- `go build ./cmd/migrate`

Live deployment still requires operator credentials and platform access. See [DEPLOYMENT.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/DEPLOYMENT.md) and [FINAL_HANDOFF_REPORT.md](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/FINAL_HANDOFF_REPORT.md).
