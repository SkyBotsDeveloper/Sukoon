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

## Help And Parity Research Policy

Rose-style help and command-parity batches must be researched before implementation.

Source priority:

1. Official Miss Rose docs are the primary source of truth for behavior, command names, help structure, and examples.
2. Trusted public command or setup guides may be used as secondary cross-check references for common aliases, expected UX, and operator-facing examples.
3. Public GitHub repositories may be used only as tertiary implementation-reference material, never as product truth.

When sources disagree:

- Miss Rose official docs win
- public guides do not override official docs
- unverified blog or repository examples must not be exposed in Sukoon help

Required workflow for each help or parity batch:

1. Read the matching Miss Rose docs pages for the batch.
2. Cross-check one or more trusted public guides for common usage expectations.
3. Compare those references against Sukoon's actual implemented command surface.
4. Use the user-provided screenshots, PDFs, and Rose docs to shape menu structure and explanatory copy.
5. Implement only the safe, verified subset that fits Sukoon's current Go architecture.
6. Keep unimplemented or unsafe commands out of live help and report them as deferred.

Current Rose references used for Sukoon help and parity work:

- Introduction: https://missrose.org/docs/
- Connecting To Chats: https://missrose.org/docs/basics/connections/
- Rules: https://missrose.org/docs/basics/rules/
- Admins in Rose: https://missrose.org/docs/moderation/admins/
- Command Disabling: https://missrose.org/docs/moderation/disabling/
- Filters: https://missrose.org/docs/filters/
- Notes: https://missrose.org/docs/notes/
- Message formatting: https://missrose.org/docs/formatting/
- About Federations: https://missrose.org/docs/federations/
- Managing Your Federation: https://missrose.org/docs/federations/managing/
- User Federation Commands: https://missrose.org/docs/federations/user-commands/

Current secondary cross-check reference:

- Quantaps Rose bot guide: https://quantaps.com/en/guides/telegram-rose-bot-settings-and-commands/

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
