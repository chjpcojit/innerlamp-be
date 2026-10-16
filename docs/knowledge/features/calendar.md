---
type: Feature
title: Calendar (lunar-calendar image and events)
description: The lunar-calendar screen's data — a Buddha photo chosen by date (public) and the Buddhist, Vietnamese and folk events of a solar year with both solar and lunar dates (login required), computed from stored repeat rules rather than stored per year.
resource: services/appService/domain/lunarcal/
tags: [calendar, lunar-calendar, events, catalogue, seed, buddhist]
status: draft
generated: { by: claude-code, at: 2026-10-11T04:40:00+07:00 }
sources:
  - path: docs/product/calendar/product.md
    kind: spec
  - path: docs/specs/features/calendar/requirements.md
    kind: spec
  - path: docs/specs/features/calendar/design.md
    kind: spec
  - path: docs/specs/features/calendar/tasks.md
    kind: spec
  - path: docs/specs/api/openapi.yaml
    kind: openapi
  - path: migrations/0031_calendar_events.sql
    kind: implementation-notes
  - path: docs/specs/db/erd.mmd
    kind: erd
---

# Summary

The feature has two independent halves behind one module (`calendar`):

- **Image of the day** (US-CAL-001, `GET /v1/calendar/image`, public): one Buddha/Bodhisattva photo per date, chosen
  as a pure function of the date from an embedded manifest, with the licence attribution the photo requires.
- **Events** (US-CAL-002, `GET /v1/calendar/events?year=YYYY`, access token required): the Buddhist, Vietnamese
  public-holiday and folk events that fall in one solar year, each occurrence carrying **both** its solar and its lunar
  date. "Load more" is another call with `year + 1`.

The events half stores **repeat rules, not dates**: a fixed solar date (30/4), a fixed lunar date (Phật đản 15/4 ÂL,
optionally in the leap month), every lunar month (mùng 1, Rằm), the last day of a lunar month (Giao thừa; vía Địa Tạng
and vía Dược Sư use it too, so they fall on day 29 in a short month), or a solar term (Thanh minh, 15°). Each request
reads the catalogue and computes the year's dates in memory with `domain/lunarcal`; nothing is stored per year, so every
year in 1900..2100 has data.

**Current limitation**: the catalogue content (`assets/data/calendar-events.json`, 51 events) is a **draft**. The
Buddha/Bodhisattva vía dates were checked against one aggregator page, not against an official GHPGVN calendar; the
descriptions were written by the assistant, not reviewed by the product owner; every `image_url` is `null`.

# Architecture

`delivery → usecase → domain`, `repository → domain`. The two halves share only the config block and the handler
package. The events half is the only part with a database; the transaction boundary is `ReplaceCatalogue` in the
repository (seed), reads are two plain SELECTs. `domain/lunarcal` is a pure sub-package of `domain` (standard library
only, no I/O, no clock).

| Layer | Type | File |
|---|---|---|
| Domain / Pure logic | Package (inside `domain`) | `services/appService/domain/lunarcal/lunarcal.go` — `SolarToLunar`, `LunarToSolar`, `LunarMonths`, `LeapMonth`, `DaysInLunarMonth`; Hồ Ngọc Đức's algorithm at UTC+7, verified range 1900..2100 |
| Domain / Pure logic | Package (inside `domain`) | `services/appService/domain/lunarcal/solarterm.go` — `SolarTermDate(year, longitude)`: the UTC+7 day containing the Sun's crossing of a 15° multiple |
| Domain / Entity | Domain Entity | `services/appService/domain/calendar_event.go` — `CalendarEvent`, `RecurrenceRule` + `Validate()`, `CalendarGroup`, `CalendarOccurrence`, `CalendarLunarDate`, `CalendarEventsResult`, `ErrInvalidYear` |
| Domain / Pure logic | resolver | `services/appService/domain/calendar_event_resolver.go` — `ResolveOccurrences(events, year)`: expands each rule over lunar years `year-1` and `year`, keeps those whose solar year is `year`, sorts by solar date, `sort_order`, code |
| Domain / Catalogue | seed unit | `services/appService/domain/calendar_catalogue.go` — `CalendarCatalogue` and `Verify()` (codes, texts, rules, groups, column limits) |
| Domain / Repository interface | Repository | `services/appService/domain/calendar_event_repository.go` (+ `mock_calendar_event_repository.go`) — `ListGroups`, `ListActiveEvents(lang)`, `ReplaceCatalogue` |
| Repository | Repository | `services/appService/repository/postgres_calendar_event_repository.go` — one SELECT for events with their group codes (`string_agg`), one transaction of upserts for the seed |
| Usecase | UseCase | `services/appService/usecase/calendar_event_usecase.go` — `EventsForYear`: year parsing, reads, validates rules, resolves, drops unknown group codes, keeps only events that occur |
| Usecase (image half) | UseCase | `services/appService/usecase/calendar_usecase.go` — `ImageForDate`: `dayNumber mod N` over the embedded manifest |
| Delivery / Handler | Handler | `services/appService/delivery/http/calendar_event_handler.go` — DTOs, `Cache-Control`, `INVALID_YEAR` through `writeDomainError` in `errors.go` |
| Seed CLI | command | `services/appService/cmd/seedcalendarevents/` (`main.go`, `parse.go`) — strict decode, `Verify`, `-dry-run`, one-transaction upsert |
| Data | JSON | `assets/data/calendar-events.json` — `meta` + `groups` + `events`, each event with a `source` (review aid, not loaded) |
| Wiring | composition root | `services/appService/cmd/main.go` — `cfg.Calendar.Validate()`, repository/usecase/handler, `GET /v1/calendar/events` behind `authMiddleware.AuthenticateFunc` |

Config (`services/appService/config/config.go`, `CalendarConfig`): `timezone` (what "this year" means when `year` is
omitted), `default_language`, `events_cache_max_age`, `image_base_url`; the first three are required.

No `pkg/` package is imported by the pure logic. The usecase uses `pkg/logging`; the CLI uses `pkg/config` and
`pkg/database`. Schema: [0031 Calendar Events](/migrations/0031-calendar-events.md).

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/v1/calendar/image` | public | `CalendarHandler.Image` |
| `GET` | `/v1/calendar/events` | `pkg/jwt/middleware` bearer, `access` token | `CalendarEventHandler.Events` |

Endpoint concept: [GET /v1/calendar/events](/api/calendar-events.md). The image route has no endpoint concept yet.

# Request flow

```
GET /v1/calendar/events?year=2026
  └─ authMiddleware.AuthenticateFunc            (pkg/jwt/middleware, access token only)
       └─ CalendarEventHandler.Events           (delivery/http)
            └─ CalendarEventUsecase.EventsForYear(ctx, "2026")
                 ├─ parse year: empty → current year in calendar.timezone; digits only, 1900..2100, else ErrInvalidYear
                 ├─ CalendarEventRepository.ListGroups
                 ├─ CalendarEventRepository.ListActiveEvents(lang)   (is_active, has text in lang)
                 ├─ RecurrenceRule.Validate per event → invalid: log warning, skip that event
                 ├─ domain.ResolveOccurrences(events, 2026)          (lunarcal: no I/O)
                 └─ keep events with ≥1 occurrence; drop group codes not in the catalogue
```

Seed: `seedcalendarevents` → strict decode + `CalendarCatalogue.Verify` (refuses the whole file, writes nothing) →
`ReplaceCatalogue` (groups, events, texts, then each event's links replaced wholesale; never deletes; never writes
`is_active`).

# Validation rules

- `year` (usecase): empty, or plain ASCII digits within 1900..2100; anything else (`2026.5`, `+2026`, ` 2026`,
  `1899`) is `domain.ErrInvalidYear` → `400 INVALID_YEAR`, `details.field = "year"`, before any database read.
- Rule parameters (`RecurrenceRule.Validate`, domain): per kind — `solar` month 1..12 and a day valid for that month
  (29/2 allowed); `lunar` month 1..12, day 1..30; `lunar_monthly` day 1..30; `lunar_last_day` month 1..12;
  `solar_term` longitude a multiple of 15 in 0..345.
- The seed file (`parse.go` + `Verify`): unknown JSON key is an error; each kind requires exactly its parameters and
  rejects stray ones; `source` required; codes unique; names and descriptions non-empty; every event in ≥ 1 existing
  group; column length limits; `is_leap_month` only on `lunar`.
- A lunar date that does not exist in a given year (leap month the year lacks, day 30 in a 29-day month) simply gives
  no occurrence that year — it is not an error.

# Pending work

- **Content is unreviewed.** `assets/data/calendar-events.json` descriptions and guidance are first drafts; the
  `meta.status` says so. Vía dates matched `hoatieu.vn/cac-ngay-le-phat-trong-nam-207649` only; no GHPGVN page was found.
  Phật đản is stored at 15/4 (GHPGVN practice) although that page lists 8/4 as the traditional đản sanh day.
- **E20 has no independent oracle.** Lunar conversion is checked against known Tết dates (1900, 2000–2026), known leap
  months, a per-day round trip over 1900..2100, and a day-by-day scan oracle — but not against a full second published
  table. `usecase/genealogy_lunar.go` is the same algorithm, not an independent source.
- **Duplicate converter.** `usecase/genealogy_lunar.go` duplicates `domain/lunarcal` and returns day `0` on 2054-05-07
  and 2062-04-09; `usecase/lunar_crosscheck_test.go` logs those dates and keeps the two from drifting elsewhere.
  Moving genealogy onto `lunarcal` is not done.
- **No event images.** `image_url` is `null` for every event; source, licence and hosting are undecided.
- **Dev-only seeding.** The CLI loads config and secrets through `appConfig.NewMockSecretGetterFromYAML` (`⚠️ DEV ONLY`);
  the seed overwrites rows by code, so it is for first setup and dev, not for a database back-office has taken over.
- **No back-office.** Events, groups and the holiday flag are database rows so a back-office can edit them later, but
  none exists; `is_active = false` is the only switch and is set by hand.
- **No substitute days.** Nghỉ bù and the extra day next to 2/9 are out of scope, so `is_public_holiday` is true on the
  event's own day only.
- Image half: no thumbnail; `image_url` is built from the request host in dev unless `calendar.image_base_url` is set.
- Promote to `stable` once the content has been reviewed against an official source and the image question is decided.

# Known gaps vs. design

- Migration is `0031`, not the `0030` the first design draft named (`0030` was taken by `temple_suggestions`).
- The entity is `CalendarLunarDate`, not `LunarDate` (`domain/genealogy.go` already has a year-less `LunarDate`).
- `ReplaceCatalogue` does not write `is_active`, so re-running the seed cannot switch back on an event that
  back-office turned off (E18 wording "overwrite with the file's content" is read as content, not flags).

# Related concepts

- [GET /v1/calendar/events](/api/calendar-events.md) — the wire contract.
- [0031 Calendar Events](/migrations/0031-calendar-events.md) — the four tables.
- [Genealogy](/features/genealogy.md) — also derives lunar dates from solar ones, with its own copy of the algorithm.
- Login (`docs/specs/features/login/`) — issues the access token the events route requires.
