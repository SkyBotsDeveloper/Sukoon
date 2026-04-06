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

## Scaling

Safe scale-out assumptions:

- multiple workers are supported
- update claims use lock-safe database polling
- job claims use lock-safe database polling
- flood tracking and leases are shared in Valkey or Redis

## Backups

Back up:

- PostgreSQL

Persist:

- Valkey appendonly data for better restart behavior

Do not depend on Redis persistence alone for canonical bot data.

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
