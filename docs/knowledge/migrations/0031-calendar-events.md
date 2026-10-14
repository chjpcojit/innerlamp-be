---
type: Migration
title: 0031 Calendar Events
description: Creates calendar_groups, calendar_events, calendar_event_translations and calendar_event_groups — the repeat-rule catalogue behind GET /v1/calendar/events.
resource: migrations/0031_calendar_events.sql
tags: [migration, schema, calendar, postgres]
status: stable
generated: { by: claude-code, at: 2026-10-11T04:50:00+07:00 }
sources:
  - path: docs/specs/features/calendar/design.md
    kind: spec
  - path: docs/specs/db/erd.mmd
    kind: erd
---

# Summary

Incremental migration on top of nothing: it depends on no other table. It creates the four tables of the events half
of [Calendar](/features/calendar.md). It is **forward-only**: there is no `DOWN` section and no rollback file; the
statements are idempotent (`CREATE ... IF NOT EXISTS`), so applying it twice is a no-op. It was numbered `0031` because
`0030` was already `temple_suggestions_address_codes`.

No table has a year column: a row holds a repeat **rule**, and the dates of a year are computed on read.

# Tables

| Table | Operation | Purpose |
|---|---|---|
| `calendar_groups` | `CREATE TABLE` | Labels (`buddhist`, `vietnam`, `folk`; more can be added). PK `code VARCHAR(32)`; `sort_order >= 1` (`chk_calendar_groups_sort_order`). |
| `calendar_events` | `CREATE TABLE` | One row per event. PK `code VARCHAR(64)`; `kind`; the rule parameters; `is_leap_month`; `is_public_holiday`; `image_url`; `sort_order`; `is_active` (default `true`). |
| `calendar_event_translations` | `CREATE TABLE` | Name, short/long description and guidance per language. PK `(event_code, lang)`; FK to `calendar_events` `ON DELETE CASCADE`. v1 has `lang = 'vi'` only; a new language is new rows. |
| `calendar_event_groups` | `CREATE TABLE` | Event ↔ group links. PK `(event_code, group_code)`; FK to events `ON DELETE CASCADE`, FK to groups `ON DELETE RESTRICT`. |

Constraints on `calendar_events` that matter (verified by inserting valid and invalid rows on Postgres 16):

- `chk_calendar_events_kind`: `kind IN ('solar','lunar','lunar_monthly','lunar_last_day','solar_term')`.
- `chk_calendar_events_params`: each kind must carry exactly its parameter columns and the rest must be `NULL` — `solar`
  needs `solar_month`+`solar_day`; `lunar` needs `lunar_month`+`lunar_day`; `lunar_monthly` needs `lunar_day`;
  `lunar_last_day` needs `lunar_month`; `solar_term` needs `solar_term_longitude`. `is_leap_month` may be true only for
  `lunar`.
- `chk_calendar_events_ranges`: months 1..12, `solar_day` 1..31, `lunar_day` 1..30, `solar_term_longitude` 0..345 and a
  multiple of 15 (0° is valid — Xuân phân).
- `chk_calendar_events_sort_order`: `sort_order >= 1`.

"Every event has at least one group" cannot be a table CHECK; the seed (`CalendarCatalogue.Verify`) enforces it.

# Indexes

| Index | Columns | Why it exists |
|---|---|---|
| `idx_calendar_event_groups_group` | `(group_code, event_code)` | Events of a group, and the `RESTRICT` check when a group is deleted. |

The primary keys cover the reads: `ListActiveEvents` joins translations by `(event_code, lang)` and links by
`event_code`. The catalogue is tens of rows, so `is_active` is not indexed.

# Seed data

None in the migration. The rows come from `assets/data/calendar-events.json` through
`go run ./services/appService/cmd/seedcalendarevents` (or `make seed-calendar-events`, `make ensure-calendar-events`
for "only when empty"). The seed upserts by `code`, never deletes, and never writes `is_active`.

# Applied by

`make migrate` / `make up` runs the `migrate` compose service over every file in `migrations/` (they are written to
be re-applied). The repository tests and the E2E harness apply the file themselves: `applyMigration` in
`services/appService/repository/postgres_registration_repository_test.go` lists `0031_calendar_events.sql`, and
`e2e/harness.go` globs `migrations/*.sql`. `cmd/main.go` does not apply migrations.

# Related concepts

- [Calendar](/features/calendar.md) — the feature that reads and seeds these tables.
- [GET /v1/calendar/events](/api/calendar-events.md) — the route that serves them.
- ERD: `docs/specs/db/erd.mmd` / `erd.svg` (regenerated 2026-10-11 with the four tables).
