# Architecture

## Overview

Sukoon runs as a worker-style Telegram backend, not a web application pretending to be a bot runtime.

Core runtime pieces:

- webhook ingress HTTP service
- PostgreSQL-backed durable update queue
- worker service for updates and jobs
- Telegram HTTP client with timeout, retry, and backoff
- Valkey or Redis shared state for leases, flood tracking, and bot cache
- canonical SQL migrations in `bot-core/migrations`

## Update Lifecycle

1. Telegram sends a webhook request to `/webhook/<webhook_key>`.
2. The ingress validates the Telegram secret token.
3. The ingress resolves the active bot by webhook key.
4. The ingress stores the update in `telegram_updates` with bot-scoped idempotency.
5. The ingress returns quickly with `202 Accepted`.
6. A worker claims queued updates from PostgreSQL with lock-safe polling.
7. The worker loads the active bot transport, runtime bundle, permissions, and handlers.
8. The router dispatches moderation, admin, antispam, content, captcha, owner, federation, clone, utility, and other feature flows.
9. Background fanout work such as purge, broadcast, global bans, and fed bans runs through the `jobs` table, not inline in the webhook request path.

## Module Boundaries

`bot-core/internal/config`
- env loading and runtime mode selection

`bot-core/internal/webhook`
- authenticated ingress and update enqueue

`bot-core/internal/worker`
- worker loop, retries, dead-letter handling, captcha sweeps

`bot-core/internal/processor`
- update decoding, bot resolution, router invocation

`bot-core/internal/router`
- transport-independent update routing and service orchestration

`bot-core/internal/telegram`
- Telegram API transport

`bot-core/internal/persistence/postgres`
- canonical Postgres persistence layer

`bot-core/internal/state/redis`
- shared state, leases, flood tracking, bot cache

`bot-core/internal/service/*`
- domain features with isolated responsibilities

## Canonical Storage Contracts

Key tables:

- `bot_instances`
- `bot_roles`
- `chats`
- `chat_settings`
- `moderation_settings`
- `warnings`
- `approvals`
- `disabled_commands`
- `locks`
- `blocklist_rules`
- `antiflood_settings`
- `captcha_settings`
- `captcha_challenges`
- `notes`
- `filters`
- `afk_states`
- `antiabuse_settings`
- `antibio_settings`
- `antibio_exemptions`
- `federations`
- `federation_chats`
- `federation_admins`
- `federation_bans`
- `global_blacklist_users`
- `global_blacklist_chats`
- `jobs`
- `telegram_updates`

The repository no longer carries duplicate warn, report, or log-channel contracts.

## Bot And Clone Isolation

Every bot instance has its own:

- Telegram token
- webhook key
- webhook secret
- owner relationship
- bot-scoped data

There is no global default bot transport and no legacy clone leakage path.

## Performance And Reliability Decisions

- webhook ingress is fast-ack only
- heavy fanout is job-backed
- no critical moderation state depends on process memory
- retries and dead-letter paths are explicit
- per-update structured logs exist across webhook, update, and job paths
- permission loading and runtime bundle loading happen once per update

## Intentionally Removed

These are not part of the architecture anymore:

- Next.js as bot runtime
- Supabase as bot-core persistence
- public bot setup/dashboard endpoints
- in-memory-only dedupe
- in-memory-only captcha or flood correctness
