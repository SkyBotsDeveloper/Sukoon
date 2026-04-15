# Current Data Architecture Audit

## 1. Executive Summary

Sukoon currently runs on a two-tier storage model:

- PostgreSQL is the durable system of record.
- Valkey/Redis is used only for ephemeral hot-path state and short-lived coordination.

That split is already real in the code today. The Go runtime in `bot-core` always boots both a Postgres store and a Redis state store in [`bot-core/internal/app/app.go`](bot-core/internal/app/app.go). There is no alternate durable backend, no Supabase path, and no Redis-backed durable queue.

The practical storage shape right now is:

- Postgres stores bot definitions, chats, users, chat settings, moderation state, notes, filters, rules, approvals, locks, blocklists, antiflood/antiraid/captcha settings, federation data, clone ownership, global blacklists, AFK state, durable jobs, and the durable Telegram update queue.
- Valkey/Redis stores webhook bot resolution cache, antiflood counters, antiraid join-burst counters, and short leases used to suppress repeated work.
- Some very short-lived in-process memory caches exist in the router and permission layer, but those are local-only process caches, not shared storage.

The current architecture is already compatible with an eventual "external Postgres + local Redis" model:

- Most durable data fits external Postgres well.
- The update queue and jobs queue can live in external Postgres, but worker latency and DB round trips will matter more than they do with a local DB.
- Redis is currently used only for latency-sensitive temporary state, so it is the strongest candidate to keep local on the VPS.

The biggest migration risk is not correctness of the schema. The biggest risk is latency amplification on the message path, because the bot currently does multiple Postgres reads per update:

- webhook bot lookup fallback
- runtime bundle load
- approvals/global blacklist checks
- filters listing on text messages
- connection lookups for PM-connected commands
- federation lookups on joins/messages

So the storage audit conclusion is:

- durable persistent data is already clearly concentrated in Postgres
- hot ephemeral counters are already clearly concentrated in Redis
- the main caution is not data model mismatch, but hot-path query volume

## 2. Current Storage Stack

### Runtime stack

Actual runtime wiring:

- `bot-core/internal/app/app.go`
  - creates Postgres store with `postgres.New(...)`
  - runs migrations with `store.Migrate(...)`
  - creates Redis state with `redisstate.New(...)`
  - starts webhook server and/or worker depending on `APP_MODE`

### Deployment defaults

Default local deployment in [`docker-compose.yml`](docker-compose.yml):

- `postgres` container with persistent Docker volume `pgdata`
- `valkey` container with persistent Docker volume `valkeydata`
- `migrate` one-shot migration container
- `bot-core-web`
- `bot-core-worker`

Important point:

- Compose persists both Postgres and Valkey to local Docker volumes.
- Application semantics still treat Postgres as durable truth and Redis as disposable hot state.
- Valkey persistence is operational convenience, not a correctness dependency.

### Platform deployment assumptions

- Railway and Heroku deployment files expect platform-provided `DATABASE_URL` and Redis envs.
- The Go code does not assume local Postgres specifically; it assumes a reachable Postgres URL and a reachable Redis endpoint.

## 3. Environment And Runtime Storage Inputs

Primary config source: [`bot-core/internal/config/config.go`](bot-core/internal/config/config.go)

### Storage-related environment variables

- `DATABASE_URL`
  - required
  - defines the only durable relational store
  - used by both web and worker modes

- `REDIS_ADDR`
  - Redis/Valkey host:port when `REDIS_URL` is not used

- `REDIS_PASSWORD`
  - Redis password if needed

- `REDIS_DB`
  - Redis logical DB index

- `REDIS_URL`
  - optional full Redis URL
  - if present, parsed and preferred over separate Redis pieces

- `APP_MODE`
  - `all`, `web`, or `worker`
  - affects whether the process serves webhook ingress, worker loops, or both
  - storage dependencies remain the same: Postgres + Redis are still initialized

- `PUBLIC_WEBHOOK_BASE_URL`
  - not a storage backend variable, but operationally important for clone webhook registration
  - clone lifecycle depends on this being correct

- `WORKER_CONCURRENCY`
  - affects how many worker goroutines can claim/process Postgres-backed updates and jobs
  - higher values increase pressure on Postgres queue tables

- `WORKER_POLL_INTERVAL`
  - controls how frequently workers poll Postgres for new updates/jobs
  - directly affects DB polling frequency

### Runtime modes and storage behavior

- `web`
  - uses Redis on the webhook ingress path
  - writes raw updates durably into Postgres

- `worker`
  - claims updates and jobs from Postgres
  - uses Redis only for certain runtime checks through downstream services

- `all`
  - combines both

### Default backend assumptions

From [`bot-core/.env.example`](bot-core/.env.example):

- Postgres defaults to local compose service `postgres`
- Redis defaults to local compose service `valkey`

That means the repo currently assumes:

- local durable Postgres by default for self-hosted Docker Compose
- local Redis/Valkey by default for self-hosted Docker Compose
- platform deployments replace those with managed or externally reachable services via env vars

### Fixed Postgres client behavior in code

From [`bot-core/internal/persistence/postgres/store.go`](bot-core/internal/persistence/postgres/store.go):

- `MaxConns = 10`
- `MinConns = 2`
- `MaxConnLifetime = 1h`
- `MaxConnIdleTime = 15m`

These are not env-configurable today. They matter if Postgres is later moved off-VPS, because queue polling and hot-path reads will then share a small fixed connection pool against a higher-latency database.

## 4. PostgreSQL Data Map

Postgres is the canonical persistent data layer. It is used for both configuration/state and durable work queues.

### Practical table groups

#### Core bot and identity tables

- `bot_instances`
  - every bot runtime identity
  - includes primary bot and clones
  - stores Telegram token, webhook identifiers, owner linkage, status

- `bot_roles`
  - bot-scoped owner/sudo roles

- `users`
  - known Telegram users observed by the system

- `chats`
  - known chats observed by the system, scoped per bot

- `chat_roles`
  - bot+chat scoped roles such as mod/muter

#### Chat configuration tables

- `chat_settings`
  - largest general settings surface
  - language, reports, log channel, clean command flags, clean service flags, welcome/goodbye/rules text, blocklist defaults, disabled command behavior, admin behavior toggles

- `moderation_settings`
  - warn limit and warn mode

- `antiflood_settings`
  - antiflood thresholds and action settings

- `antiraid_settings`
  - antiraid activation window, durations, auto-threshold

- `captcha_settings`
  - captcha mode, timeout, rules requirement, button text, kick behavior

- `antiabuse_settings`
  - antiabuse enabled/action

- `antibio_settings`
  - antibio enabled/action

#### Moderation and safety state

- `warnings`
  - current warning count per user per chat

- `approvals`
  - approved user bypass state per chat

- `disabled_commands`
  - disabled command list per chat

- `locks`
  - active lock rules per chat

- `lock_allowlist`
  - allowlisted exceptions for lock enforcement

- `blocklist_rules`
  - blocklisted patterns and overrides per chat

- `antibio_exemptions`
  - antibio bypass users per chat

#### Content and chat UX

- `notes`
  - saved named notes with text, buttons, parse mode

- `filters`
  - trigger-response definitions, including media support and buttons

#### CAPTCHA and user-presence state

- `captcha_challenges`
  - active and historical captcha challenges

- `afk_states`
  - current AFK reason per user per bot

#### Connection and operator convenience

- `chat_connections`
  - current PM connection target per user per bot

- `chat_connection_history`
  - recent connected chats per user per bot

#### Federation / global moderation

- `federations`
  - federation metadata and owner

- `federation_chats`
  - chats joined to federations

- `federation_admins`
  - federation admin assignments

- `federation_bans`
  - federation bans

- `federation_subscriptions`
  - federation-to-federation subscription graph

- `global_blacklist_users`
  - owner/global blacklist users

- `global_blacklist_chats`
  - owner/global blacklist chats

#### Durable work and ingress

- `jobs`
  - durable job queue and progress tracking
  - broadcast, purge, federation/global fanout work

- `telegram_updates`
  - durable raw update queue used between webhook ingress and workers

#### Meta

- `schema_migrations`
  - tracks applied SQL migrations

### Table A — Postgres data map

| Table/Area | Purpose | Persistent? (yes/no) | Hot path? (low/medium/high) | Growth risk (low/medium/high) | Can move to external Postgres? (yes/no/with caution) | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `schema_migrations` | migration bookkeeping | yes | low | low | yes | operational metadata only |
| `bot_instances` | bot identities, tokens, webhook keys, clone ownership/status | yes | medium | low | yes | critical durable control-plane data |
| `bot_roles` | owner/sudo roles per bot | yes | medium | low | yes | permission checks use it, cached briefly in-process |
| `users` | observed Telegram user metadata | yes | medium | medium | yes | updated periodically from router |
| `chats` | observed chat metadata per bot | yes | medium | medium | yes | updated periodically from router |
| `chat_roles` | per-chat mod/muter style roles | yes | high | low | yes | read in permission checks |
| `chat_settings` | main chat config surface | yes | high | low | yes | part of runtime bundle loaded on most updates |
| `moderation_settings` | warn limit and warn mode | yes | high | low | yes | part of runtime bundle |
| `warnings` | per-user warning counters | yes | medium | medium | yes | written on warns/antiabuse/antispam actions |
| `approvals` | approved-user bypass state | yes | high | medium | yes | checked directly on moderation path |
| `disabled_commands` | disabled commands per chat | yes | high | low | yes | loaded into runtime bundle |
| `locks` | active lock rules | yes | high | medium | yes | loaded into runtime bundle |
| `lock_allowlist` | lock exceptions | yes | high | medium | yes | loaded into runtime bundle |
| `blocklist_rules` | blocklist patterns/actions | yes | high | high | yes | loaded into runtime bundle; can grow heavily in busy groups |
| `antiflood_settings` | antiflood config | yes | high | low | yes | config only; counters are in Redis |
| `antiraid_settings` | antiraid config | yes | medium | low | yes | joins consult it; auto-enable writes here |
| `antiabuse_settings` | antiabuse config | yes | high | low | yes | config only |
| `antibio_settings` | antibio config | yes | high | low | yes | config only |
| `antibio_exemptions` | antibio exempt users | yes | high | medium | yes | checked on antibio path |
| `notes` | saved named content | yes | medium | medium | yes | durable content data |
| `filters` | trigger-response content, buttons, media | yes | high | high | yes, with caution | listed from DB on text-message path today |
| `captcha_settings` | captcha config | yes | medium | low | yes | config only |
| `captcha_challenges` | pending/solved/expired captcha challenges | yes | medium | high | yes, with caution | join/callback/sweeper depend on it; likely prune candidate |
| `chat_connections` | current PM connection target | yes | medium | low | yes | PM management flow uses it directly |
| `chat_connection_history` | recent PM-connected chats | yes | low | medium | yes | convenience/history data |
| `afk_states` | AFK status per user | yes | medium | medium | yes | user convenience state; not catastrophic to lose |
| `federations` | federation metadata | yes | medium | medium | yes | critical if federations are used |
| `federation_chats` | chat membership in federations | yes | medium | medium | yes | join/message checks depend on it |
| `federation_admins` | federation admin assignments | yes | medium | low | yes | federation permission path |
| `federation_bans` | federation ban records | yes | medium | high | yes | important durable moderation data |
| `federation_subscriptions` | federation inheritance graph | yes | medium | medium | yes | affects effective bans |
| `global_blacklist_users` | owner/global banned users | yes | high | medium | yes | checked on update/join path |
| `global_blacklist_chats` | owner/global banned chats | yes | high | low | yes | checked on update/join path |
| `jobs` | durable fanout jobs and progress | yes | medium | high | yes, with caution | worker claims from here; remote latency acceptable but noticeable |
| `telegram_updates` | durable webhook-to-worker queue | yes | high | high | yes, with caution | absolutely core to ingress durability; remote DB latency matters |

### Practical persistence characteristics by area

#### Control-plane data that is serious to lose

- `bot_instances`
- `bot_roles`
- `federations`
- `federation_*`
- `global_blacklist_*`
- `chat_settings`
- `moderation_settings`
- `antiflood_settings`
- `antiraid_settings`
- `captcha_settings`
- `antiabuse_settings`
- `antibio_settings`

These define how the bot behaves. Losing them is operationally serious.

#### User/content data that is serious or very annoying to lose

- `notes`
- `filters`
- `rules`/welcome/goodbye content inside `chat_settings`
- `approvals`
- `locks`
- `lock_allowlist`
- `blocklist_rules`
- `warnings`
- `chat_roles`

Losing these would not destroy the runtime, but it would materially damage chat state and moderator trust.

#### Queue/history data that is durable by design but can be pruned

- `telegram_updates`
- `jobs`
- `captcha_challenges`
- `chat_connection_history`

These are legitimate durable tables, but they are also the clearest archive/prune candidates over time.

## 5. Redis/Valkey Data Map

Actual implementation: [`bot-core/internal/state/redis/store.go`](bot-core/internal/state/redis/store.go)

Redis is not used as a durable queue or primary database. Its responsibilities are narrow and operationally hot:

- fast webhook bot lookup cache
- antiflood counters
- antiraid join-burst counters
- short leases to suppress repeated expensive actions

### Key families in use

- `bot:webhook:<webhookKey>`
  - cached serialized `domain.BotInstance`
  - avoids repeated Postgres bot lookup on webhook ingress

- `flood:timed:<botID>:<chatID>:<userID>`
  - sorted set for timed flood counting

- `flood:streak:messages:<botID>:<chatID>`
  - list of consecutive message IDs for current streak

- `flood:streak:last:<botID>:<chatID>`
  - last sender ID for the streak logic

- `joins:<botID>:<chatID>`
  - sorted set tracking recent joins for antiraid auto-threshold

- `lease:<key>`
  - generic short TTL lease namespace
  - currently used for:
    - `lease:antibio:<botID>:<chatID>:<userID>`
    - `lease:antiraid:auto:<botID>:<chatID>`

### Table B — Redis/Valkey usage map

| Key/Usage area | Purpose | Ephemeral? (yes/no) | Hot path? (low/medium/high) | Safe to keep local only? (yes/no) | Safe to move remote? (yes/no/with caution) | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `bot:webhook:*` | cache bot instance lookup during webhook ingress | yes | high | yes | with caution | remote Redis would work, but adds latency directly on ingress |
| `flood:timed:*` | timed antiflood counters per user | yes | high | yes | with caution | remote latency would directly slow moderation checks |
| `flood:streak:messages:*` | consecutive-message streak tracking | yes | high | yes | with caution | hot path on spam-heavy chats |
| `flood:streak:last:*` | last sender marker for streak logic | yes | high | yes | with caution | paired with flood streak path |
| `joins:*` | join-burst counting for auto antiraid | yes | medium | yes | with caution | only relevant on member joins |
| `lease:antibio:*` | suppress repeated antibio scans for same user | yes | high | yes | with caution | on message path when antibio is enabled |
| `lease:antiraid:auto:*` | suppress repeated auto-antiraid announcements | yes | medium | yes | yes | correctness impact is low; mostly UX throttling |

### Redis durability characteristics

Losing Redis state is acceptable from a correctness perspective:

- webhook bot cache repopulates from Postgres
- flood counters reset
- join burst counters reset
- antibio leases reset and may cause some repeated checks
- antiraid announce lease resets and may send an extra notification

That means Redis loss is operationally tolerable, even if mildly noisy.

## 6. Hot-Path Storage Analysis

### Webhook ingress path

Actual flow in [`bot-core/internal/webhook/server.go`](bot-core/internal/webhook/server.go):

1. resolve bot by webhook key
   - Redis cache first
   - Postgres fallback
2. enqueue update into Postgres `telegram_updates`
3. return quickly

Critical path characteristics:

- Redis cache hit is ideal here
- Postgres insert into `telegram_updates` is mandatory durability work
- large fanout is intentionally not done inline

### Worker path

Actual flow in [`bot-core/internal/worker/service.go`](bot-core/internal/worker/service.go):

- claim pending updates from `telegram_updates`
- process them
- mark processed/retry/dead
- claim pending jobs from `jobs`
- update progress / complete / retry / dead
- sweep expired captchas

This means Postgres is the durable execution backbone.

### Normal message/update path

The highest storage pressure is not webhook ingress. It is worker-side per-update processing.

#### Repeated Postgres reads on normal updates

From [`bot-core/internal/router/router.go`](bot-core/internal/router/router.go):

- `EnsureChat(...)` periodically refreshes `chats` and default config rows
- `EnsureUser(...)` periodically refreshes `users`
- `LoadRuntimeBundle(...)` loads chat configuration and moderation/runtime data
- permission loading uses `GetBotRoles(...)` and `GetChatRoles(...)`
- PM-connected commands may do `GetChatConnection(...)` and another `LoadRuntimeBundle(...)`

#### Feature-specific hot-path lookups

- approvals
  - `IsApproved(...)` is used in `antispam`, `antiabuse`, and `antibio`
  - this is a direct Postgres check on the message path

- global blacklist
  - `GetGlobalBlacklistChat(...)` and `GetGlobalBlacklistUser(...)` are used in owner enforcement on update/join handling
  - direct Postgres checks

- filters
  - `ListFilters(...)` is called from the content service on text messages
  - this is one of the most performance-sensitive DB reads because it happens per text message in filter-enabled chats

- federation checks
  - federation service does `GetFederationByChat(...)`, `GetFederationBan(...)`, and related calls on message/join enforcement
  - these are less universal than runtime bundle reads, but still latency-sensitive where federations are active

- warnings
  - warning increments are Postgres writes when moderation/antiabuse/antispam escalates

#### Redis hot-path work

- antiflood counters via `TrackFlood(...)`
- antiraid join burst via `TrackJoinBurst(...)`
- antibio lease via `AcquireLease(...)`

These are exactly the kinds of operations that benefit from staying physically close to the bot runtime.

### Features that hit storage on almost every update

Highest frequency:

- runtime bundle load from Postgres
- chat/user refresh writes to Postgres on TTL expiry
- approvals/global blacklist checks from Postgres
- permission-role reads from Postgres
- filters listing from Postgres on text messages
- webhook bot cache read from Redis

### Features that are latency-sensitive

Very latency-sensitive:

- webhook bot resolution cache
- antiflood tracking
- antibio lease acquisition
- text-message filter matching path
- runtime bundle load

Moderately latency-sensitive:

- federation join/message checks
- global blacklist lookups
- PM connection lookups
- captcha callback resolution

Less latency-sensitive:

- durable jobs
- stats
- privacy export/delete
- clone listing
- historical connection listing

## 7. Durability And Backup Importance

### Data that absolutely should survive VPS loss/reinstall

- `bot_instances`
- `bot_roles`
- `chat_settings`
- `moderation_settings`
- `antiflood_settings`
- `antiraid_settings`
- `captcha_settings`
- `antiabuse_settings`
- `antibio_settings`
- `chat_roles`
- `approvals`
- `disabled_commands`
- `locks`
- `lock_allowlist`
- `blocklist_rules`
- `notes`
- `filters`
- `federations`
- `federation_chats`
- `federation_admins`
- `federation_bans`
- `federation_subscriptions`
- `global_blacklist_users`
- `global_blacklist_chats`

These define actual product behavior and moderator expectations.

### Data that should survive if possible because losing it is annoying or trust-damaging

- `warnings`
- `captcha_challenges` that are currently pending
- `chat_connections`
- `chat_connection_history`
- `afk_states`
- `users`
- `chats`

Losing these does not destroy the bot configuration, but operators and users will notice.

### Data that can be rebuilt or safely lost

- Redis webhook cache
- Redis flood counters
- Redis join-burst counters
- Redis leases
- in-process caches in router/permissions

These are intentionally ephemeral.

### Data designed to be durable today

Durable by explicit architecture:

- `telegram_updates`
- `jobs`
- all config/state/content tables in Postgres

Webhook ingress depends on durable update storage in Postgres; worker durability depends on Postgres queue semantics.

## 8. Data Growth Risk Areas

### Highest growth risk

- `telegram_updates`
  - every inbound update is inserted
  - even processed rows remain until explicit cleanup strategy exists

- `jobs`
  - broadcast/purge/federation/global fanout history accumulates

- `filters`
  - potentially large in groups that rely heavily on custom replies
  - especially because runtime currently lists them directly per text message

- `blocklist_rules`
  - can become large if admins add many patterns or imported rules

- `captcha_challenges`
  - accumulates solved/expired rows over time

- `federation_bans`
  - can grow heavily in active federation setups

### Medium growth risk

- `warnings`
- `users`
- `chats`
- `chat_connection_history`
- `antibio_exemptions`

### Low growth risk / mostly config tables

- `chat_settings`
- `moderation_settings`
- `antiflood_settings`
- `antiraid_settings`
- `captcha_settings`
- `antiabuse_settings`
- `antibio_settings`
- `bot_roles`
- `chat_roles`
- `locks`
- `lock_allowlist`
- `global_blacklist_*`

### Clear prune/archive candidates if needed later

- `telegram_updates`
- `jobs`
- `captcha_challenges`
- possibly stale `chat_connection_history`
- possibly stale `warnings` if product policy permits reset/archive

## 9. Suitability For External Postgres + Local Redis Model

Target model being assessed:

- primary persistent DB on external Postgres
- local VPS Redis/Valkey for speed-critical hot state
- optional standby/backup DB later

### Fit assessment by storage area

#### Best fit for external Postgres

- bot identities and clone ownership
- user/chat metadata
- chat settings and moderation config
- notes, filters, rules, welcome/goodbye content
- approvals, disabled commands, locks, blocklists
- federations and global blacklists
- privacy export/delete source data

Reason:

- this is durable business data
- correctness matters more than ultra-low local latency
- it already lives fully in Postgres

#### Safe to move to external Postgres, but with caution

- `telegram_updates`
- `jobs`
- `filters`
- `captcha_challenges`
- federation-heavy moderation paths

Reason:

- these are still semantically correct in external Postgres
- but they are more latency-sensitive or higher-volume than static settings
- the code currently does not minimize DB round trips aggressively on these paths

#### Best kept local on VPS

- Redis webhook bot cache
- Redis flood counters
- Redis join-burst counters
- Redis short leases
- in-process router/permission caches

Reason:

- they are ephemeral
- they are hot-path
- correctness loss is acceptable
- low latency matters more than durability

#### Should stay local unless scaling demands otherwise

- Redis as a whole, given current usage profile

Remote Redis is possible, but current usage gives little benefit and obvious latency downside.

### Table C — Migration priority

| Data area | Current backend | Importance | Migration difficulty | Recommended future location | Reason |
| --- | --- | --- | --- | --- | --- |
| durable config tables (`chat_settings`, moderation, antiflood, antiraid, captcha, antiabuse, antibio) | Postgres | critical | low | external Postgres | strong fit for managed durable DB |
| bot/clone metadata (`bot_instances`, `bot_roles`) | Postgres | critical | low | external Postgres | core control-plane data, small volume |
| notes/filters/rules content | Postgres | high | medium | external Postgres | durable user data; filters need latency watch |
| approvals/locks/blocklists/chat roles | Postgres | high | low-medium | external Postgres | durable moderation state |
| federations/global blacklists | Postgres | critical for those features | medium | external Postgres | cross-chat/global data benefits from durable shared DB |
| update queue (`telegram_updates`) | Postgres | critical | medium | external Postgres, with caution | already durable queue, but worker/webhook latency depends on DB RTT |
| jobs queue/history (`jobs`) | Postgres | high | medium | external Postgres, with caution | durable fanout work fits Postgres, but polling/progress writes add RTT |
| captcha challenges | Postgres | medium | low-medium | external Postgres | durable enough to externalize, but table should be managed for growth |
| users/chats metadata | Postgres | medium | low | external Postgres | durable metadata, easily externalized |
| Redis webhook cache | Redis/Valkey | medium | low | local Redis | ingress latency benefit; disposable |
| Redis flood/join counters | Redis/Valkey | high on hot path | low | local Redis | exact use case for local ephemeral store |
| Redis leases | Redis/Valkey | medium | low | local Redis | cheap local coordination; not worth external latency |

## 10. Recommended Next Migration Scope

This section is intentionally scoped as audit guidance, not an implementation plan.

### Safest first migration boundary

If the next phase adopts "external Postgres + local Redis", the cleanest first boundary is:

1. move only the primary Postgres durable store
2. keep Redis/Valkey local on the VPS
3. keep the Go runtime architecture unchanged
4. validate webhook ingress, worker claim loops, and message-path latency against the external DB

### Why that boundary is safe

- Postgres is already the canonical system of record.
- Redis is already narrowly scoped to temporary hot state.
- No logic depends on Redis durability.
- No table redesign is required just to externalize Postgres.

### What should not be bundled into the first storage move

- moving Redis remote at the same time
- redesigning the queue model
- changing webhook/worker split
- changing runtime modes

That would make it harder to isolate performance regressions.

### Suggested next migration focus areas

- validate external Postgres connectivity and worker/webhook behavior
- verify connection pooling assumptions under higher RTT
- identify whether `LoadRuntimeBundle(...)`, `ListFilters(...)`, and blacklist/approval checks need later optimization
- define retention strategy for `telegram_updates`, `jobs`, and `captcha_challenges`

## 11. Risks To Watch Before Changing Storage

### 1. Message-path query volume

The main risk is not schema compatibility. It is the number of DB round trips per update.

Particularly sensitive:

- `LoadRuntimeBundle(...)`
- `ListFilters(...)`
- `IsApproved(...)`
- global blacklist lookups
- federation membership/ban lookups

### 2. Durable queue latency

Both webhook ingress and worker execution depend on Postgres queue operations:

- insert into `telegram_updates`
- claim/update rows in `telegram_updates`
- claim/update rows in `jobs`

External Postgres is viable, but queue throughput and retry behavior should be measured after the move.

### 3. Filter scalability

Filters are durable Postgres content and are currently read with `ListFilters(...)` on text-message handling. If a chat has many filters, both DB load and application-side match cost will rise.

### 4. Queue/history bloat

Without pruning:

- `telegram_updates`
- `jobs`
- `captcha_challenges`

can grow indefinitely and change performance characteristics over time.

### 5. Redis locality matters for moderation smoothness

Remote Redis would not break correctness, but it would directly affect:

- webhook cache lookups
- flood tracking
- join burst counting
- antibio lease checks

That is a poor trade if the goal is a fast VPS runtime.

### 6. Platform failure-domain assumptions

If Postgres moves off-VPS while Redis stays local:

- VPS loss should no longer destroy durable chat/config/federation/content data
- VPS loss will still wipe ephemeral hot counters and leases
- that is acceptable, but operators should understand the distinction

### 7. Backup expectations

Right now, true business continuity depends overwhelmingly on backing up Postgres. Backing up Redis is optional from a product-correctness perspective.
