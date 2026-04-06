# Migration

## Legacy Status

The old Next.js and Supabase bot runtime has been removed from the repository.

Migration target:

- the Go runtime in `bot-core`

Non-targets:

- legacy webhook routes
- Supabase-era handler code
- duplicated warn/report/log contracts
- in-memory moderation correctness paths

## Data Mapping Principles

- migrate product intent, not broken legacy implementation detail
- keep one canonical storage contract per feature
- reject ambiguous legacy data rather than reintroducing drift
- keep all migrated data bot-scoped where applicable

## Important Mappings

| Legacy concept | Final destination |
| --- | --- |
| legacy webhook execution | `telegram_updates` + worker processing |
| `bot_clones` | `bot_instances` |
| warn setting drift | `moderation_settings` + `warnings` |
| report drift | `chat_settings.reports_enabled` |
| log channel drift | `chat_settings.log_channel_id` |
| in-memory captcha | `captcha_challenges` |
| in-memory flood state | Valkey plus `antiflood_settings` |
| legacy antiabuse lists | curated matcher plus `antiabuse_settings` |
| legacy antibio state | `antibio_settings` + `antibio_exemptions` |

## What Can Still Be Migrated

- bot and clone identities
- chats
- notes
- filters
- rules
- welcome and goodbye text
- approvals
- disabled commands
- warn counters if the source is trustworthy
- federation membership and bans after source validation

## What Should Be Discarded

- legacy in-memory dedupe
- legacy in-memory captcha state
- legacy in-memory flood state
- duplicate warn metadata
- duplicate report metadata
- duplicate log-channel metadata
- broken GDPR references
- obsolete Supabase-era schema repair scripts

## Suggested Migration Order

1. Extract only stable legacy data.
2. Load `bot_instances`.
3. Load chats and chat settings.
4. Load moderation settings and warnings.
5. Load approvals, disabled commands, locks, and blocklists.
6. Load notes, filters, welcome/goodbye, and rules.
7. Load owner/global data and federation data last.
