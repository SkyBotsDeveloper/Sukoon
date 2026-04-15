# Operations

## Runtime Modes

`APP_MODE=web`
- webhook ingress only

`APP_MODE=worker`
- background workers only

`APP_MODE=all`
- both ingress and worker in one process

Production should prefer split web and worker processes.

## Health And Logs

Health endpoint:

- `GET /healthz`

Structured logs cover:

- webhook acceptance and rejection
- update processing
- job starts, retries, completion, and dead-letter moves
- Telegram API requests

## Jobs

Job-backed flows:

- purge
- broadcast
- global ban and unban fanout
- federation ban and unban fanout

Operator-visible status commands:

- `/purge status`
- `/broadcast status`

## Webhook Operations

For the primary bot:

- use the `PRIMARY_BOT_WEBHOOK_KEY`
- use the `PRIMARY_BOT_WEBHOOK_SECRET`

For clone bots:

- create or sync clones through the clone commands
- each clone gets its own webhook key and secret

## Help And Parity Policy

Help and command-parity batches must be researched before implementation.

Source priority:

1. Sukoon's own implemented command surface and tests are the primary source of truth.
2. Trusted screenshots, operator notes, and provided reference material may be used as secondary cross-check inputs.
3. Public guides or repositories may be used only as implementation inspiration, never as product truth.

When sources disagree:

- Sukoon's implemented runtime and tests win
- secondary references do not override implemented behavior
- unverified blog or repository examples must not be exposed in Sukoon help

Required workflow for each help or parity batch:

1. Review the requested section batch and Sukoon's current command surface.
2. Cross-check any trusted screenshots, PDFs, or operator notes provided for that batch.
3. Compare those references against Sukoon's actual implemented behavior.
4. Shape menu structure and explanatory copy around the safe verified subset.
5. Keep unimplemented or unsafe commands out of live help and report them as deferred.

## Scaling

Safe scale-out assumptions:

- multiple workers are supported
- update claims use lock-safe database polling
- job claims use lock-safe database polling
- flood tracking and leases are shared in Valkey or Redis

## Storage Model

Recommended production model:

- external Postgres for durable state
- local VPS Valkey/Redis for hot-path state

Durable Postgres state includes:

- bot instances and clone ownership
- chat and moderation settings
- notes, filters, rules, approvals, locks, blocklists
- federation data and global blacklists
- durable jobs and queued updates

Ephemeral Redis state includes:

- webhook bot cache
- antiflood counters
- antiraid join burst counters
- short leases

## Backups

Back up:

- external PostgreSQL

Provider recommendations:

- enable provider backups or point-in-time restore where available
- keep at least one tested export/restore path for the Postgres database
- for Neon-compatible providers, keep the project in the nearest practical region to the VPS

Persist:

- Valkey appendonly data for better restart behavior

Do not depend on Redis persistence alone for canonical bot data.

## Failure Domains

What survives VPS loss:

- external Postgres durable state

What does not need to survive VPS loss:

- local Redis hot-state
- in-process runtime caches

Practical impact of losing local Redis:

- webhook bot cache repopulates
- flood counters reset
- antiraid join burst counters reset
- short leases reset

This is acceptable under the current architecture.

## VPS Replacement Checklist

1. Provision the replacement VPS.
2. Install Docker and Docker Compose.
3. Restore `bot-core/.env` with the same `DATABASE_URL`, `MIGRATE_DATABASE_URL`, Redis, Telegram, and webhook values.
4. Start local Valkey.
5. Run `docker compose run --rm migrate`.
6. Start `bot-core-web` and `bot-core-worker`.
7. Verify `/healthz`.
8. Verify Telegram webhook status.

## Future Standby Database Note

The current repo supports one primary Postgres URL cleanly. A standby or backup Postgres can be added later at the operator level, but automatic failover is not part of the current runtime.

## Operational Limits And Current Caveats

- language support is shared and deterministic, but not every bot response has localized variants yet
- rich note/filter formatting supports the implemented structured syntax, not every historical legacy variant
- metrics hooks exist, but a full external metrics backend is still optional future work

## Suggested Routine Checks

- `go test ./...`
- `go test -tags=integration ./...`
- check worker logs for dead-letter events
- check `/healthz`
- verify Telegram webhook status after deployment or secret rotation
