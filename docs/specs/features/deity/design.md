# Design — deity

Version: 3.1
Last updated: 2026-10-15
Status: in-progress — 3 read routes + seed have run for real (dev); the `type` filter has E2E. **v3.0 not implemented**: the rows / Steps marked **(v3.0)** are design, with no code yet (page size 12, vía links to the festival calendar, `next_via` / `via_events`, the `date` parameter, catalogue file in git).
Module: `deity`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

The three routes (`GET /v1/deities/top`, `GET /v1/deities`, `GET /v1/deities/{id}`) are all public and do not go through the auth middleware. The feature touches all four Clean Architecture layers of `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, wired in `cmd/main.go`) and owns migrations `0007_deities.sql`, `0015_deities_deva_type.sql` and (v3.0) the vía-link migration. Data enters the table through the `cmd/seeddeities` command (upsert by `id`); there is **no** seeding at startup, and the JSON file is **not** embedded in the binary. Same shape as `temple` and `article`; the difference is that ordering relies entirely on `sort_order` (position in the file).

**(v3.0)** Ngày vía (a deity's commemoration days, "vía days") are no longer deity data: each deity links to festival-calendar events (`calendar_events.code`), and every deity read computes the next vía occurrence with the `calendar` feature's own date calculator (`domain.ResolveOccurrences`) — the server computes, the client only counts down. The deity usecase therefore also depends on `domain.CalendarEventRepository` (read-only `ListActiveEvents`) and a clock; it does not call `CalendarEventUsecase` and changes nothing in the `calendar` feature.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/deity.go` | `Deity`, `Iconography`, `Mantras`, `DeityPage`, constants `DeityBuddha`/`DeityBodhisattva`/`DeityDeva`, `IsValidDeityType`. No JSON/db tags. **(v3.0)** `Deity` drops `LunarMemorialDays` and the `MemorialDay` type; adds `ViaEventCodes []string` (linked calendar event codes, catalogue content). |
| Domain / Vía (v3.0, not yet built) | `services/appService/domain/deity_via.go` | Pure functions, no clock, no I/O. `DeityVia{Event CalendarEvent; Next *CalendarOccurrence}`. `PickNextVia(occurrences []CalendarOccurrence, sortOrder map[string]int, today time.Time) *CalendarOccurrence` — the earliest occurrence with `SolarDate >= today`, ties broken by `sortOrder` then `EventCode` (BR-DTY-06, BR-CAL-12); this is the function `rules/deity-next-via.json` runs against. `ResolveDeityVias(events []CalendarEvent, today time.Time) []DeityVia` — calls `ResolveOccurrences(events, Y)` and `(…, Y+1)` (skipping `Y+1` when `Y = 2100`), takes the first occurrence `>= today` for each event, sorts per AC 28 (events with no occurrence last). `NextVia([]DeityVia) *DeityVia` = the first element with `Next != nil`. |
| Domain / Errors | `domain/deity.go` | `ErrDeityNotFound`, `ErrInvalidDeityType`; reuses `ErrInvalidPage` (`domain/temple.go`). **(v3.0)** reuses `ErrInvalidDate` (`domain/calendar.go`). |
| Domain / Repository interface | `domain/deity.go` | `DeityRepository`: `ListTop`, `List(ctx, deityType, offset, limit)`, `FindByID`, `Upsert`. The order `sort_order ASC, id ASC` is part of the contract. **(v3.0)** every read returns `ViaEventCodes` (sorted by code); `Upsert` also writes the vía links; adds `MissingCalendarEvents(ctx, codes []string) ([]string, error)` for the seed. |
| Domain / Mock | `domain/mock_deity_repository.go` | Hand-written mock for usecase tests. **(v3.0)** adds `MissingCalendarEvents`. The usecase reuses the existing `domain/mock_calendar_event_repository.go`. |
| Usecase | `services/appService/usecase/deity_usecase.go` | `Top`, `ListDeities` (offset, `total_pages`, rejects `page < 1` / `> MaxInt32`, checks `type`, nil `items` → `[]`), `Get`. **(v3.0)** the constructor also takes `domain.CalendarEventRepository`, `config.CalendarConfig` (`Timezone`, `DefaultLanguage`) and `now func() time.Time`; the three methods also take `date string`: empty → today in `calendar.timezone`, otherwise parse `YYYY-MM-DD` with year 1900..2100 like `CalendarUsecase` (error → `ErrInvalidDate`), checked after `page` and `type`. After reading the deities: one `ListActiveEvents(lang)` call, filtered by each deity's `ViaEventCodes` (codes not in the active list are ignored — BR-CAL-16), `ResolveDeityVias`, `NextVia`. Returns `domain.DeityWithVia{Deity; Vias []DeityVia; Next *DeityVia}` (or a page of it). |
| Repository | `services/appService/repository/postgres_deity_repository.go` | SQL for `deities` with shared `deityColumns`/`deityOrder`; `List` adds `WHERE type = $n` when filtering, for both the count query and the page query; `Upsert` is one transaction `ON CONFLICT (id) DO UPDATE` overwriting every column; `pq.Array` for `sacred_artifacts`/`great_vows`; `sql.ErrNoRows` → `ErrDeityNotFound`. **(v3.0)** drops the `lunar_memorial_days` column and `memorialDayJSON`; `deityColumns` adds `ARRAY(SELECT event_code FROM deity_via_events v WHERE v.deity_id = d.id ORDER BY event_code)` (no extra query); `Upsert` in the same transaction: `DELETE FROM deity_via_events WHERE deity_id = ANY($1)` then `INSERT` the file's links again; `MissingCalendarEvents` = the codes in `codes` that are not in `calendar_events` (inactive included — an inactive code is still valid to link). |
| Seed CLI | `services/appService/cmd/seeddeities/main.go`, `parse.go` | `main`: reads the file (`-file`), loads dev config + secrets (`⚠️ DEV ONLY`), connects to the DB, calls `Upsert`. `parseDeities`: plain JSON, checks empty/duplicate id, `type` ∈ the 3 types, `name_vi`; `SortOrder` = position in the file. **(v3.0)** `sourceDeity` drops `lunar_memorial_days`, adds `via_event_codes: [string]` (each code non-empty, no duplicates within one deity); `main` calls `MissingCalendarEvents` with every code before `Upsert`; any missing code → `log.Fatalf` listing the codes, nothing written. |
| Data | `assets/data/phat_botat.json` | 20 entries (7 Phật (Buddhas), 12 Bồ tát (Bodhisattvas), 1 Chư Thiên (deva)), plain JSON, read from disk at seed time. **Currently caught by `.gitignore` (`*.json`) and missing from this checkout.** **(v3.0)** put in git via the exception `!assets/data/phat_botat.json`; each entry has `via_event_codes` instead of `lunar_memorial_days` (content blocked by the `⚠️ Not confirmed (product)` on the vía mapping in `requirements.md`). |
| Delivery / Handler | `services/appService/delivery/http/deity_handler.go` | `Top`, `ListDeities`, `Get`; parses `page`; reads `type` raw. The `deityUsecase` interface is declared on the consumer side. **(v3.0)** reads `date` raw (`r.URL.Query().Get("date")`) for all three routes and passes it to the usecase without checking it itself. |
| Delivery / DTO | `delivery/http/deity_handler.go` | `deitySummaryResponse`, `deityDetailResponse`, `iconographyResponse`, `mantrasResponse`, `deityTopResponse`, `deityListResponse` — JSON tags only here. Image URLs go through `imageHost.url` (root-relative → absolute; `""` stays `""`). **(v3.0)** summary adds `next_via *deityViaResponse` (`{event_code, name, solar_date, lunar_date}`, `lunar_date` in the same shape as the `GET /v1/calendar/events` occurrence); detail adds `via_events []deityViaEventResponse` (`{event_code, name, next_occurrence *{solar_date, lunar_date}}`) and `lunar_memorial_days` derived from `via_events` (AC 30, `memorialDayResponse` kept as a pure DTO with no domain counterpart any more). Arrays are never `null`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrDeityNotFound` → `404 DEITY_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE`; `ErrInvalidDeityType` → `400 INVALID_DEITY_TYPE`; everything else → `500`. **(v3.0)** `ErrInvalidDate` is already mapped to `400 INVALID_DATE`, `details.field` defaulting to `"date"` (`errors.go` was read) — no change needed in this file. |
| Delivery / Images | `delivery/http/image_url.go` | Shared `imageHost`: a value starting with `/` is prefixed with `server.public_base_url` (empty = the request's origin); any other value is returned unchanged. |
| Delivery / Fallback | `delivery/http/fallback.go` | `WithAPIFallback` turns `ServeMux` 404/405 into the envelope. Shared. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `DeityConfig{TopLimit, PageSize}` (dev: 10, 20). **(v3.0)** `page_size: 12`. The deity usecase also reads `cfg.Calendar.Timezone` and `cfg.Calendar.DefaultLanguage` (no new keys). |
| Wiring | `services/appService/cmd/main.go` | Builds repo → usecase → handler (`imageBase` = `WithImageBase(cfg.Server.PublicBaseURL)`); registers the 3 routes. **(v3.0)** passes `calendarEventRepo` (already built for calendar), `cfg.Calendar`, `time.Now` into `NewDeityUsecase`. Does not touch the `⚠️ DEV ONLY` block of `main.go`. `GET /v1/deities/{id}/sutras` belongs to `sutra` and is unchanged. |
| Migration | `migrations/0007_deities.sql`, `0015_deities_deva_type.sql`, `0021_relative_image_urls.sql` | Table `deities`, `CHECK` on the three types, indexes `idx_deities_order`, `idx_deities_type_order`; relative images. **(v3.0, not yet built)** `migrations/00NN_deity_via_events.sql` (the next free number at implementation time, `0037` today): `CREATE TABLE IF NOT EXISTS deity_via_events (deity_id VARCHAR(64) REFERENCES deities(id) ON DELETE CASCADE, event_code VARCHAR(64) REFERENCES calendar_events(code), PRIMARY KEY (deity_id, event_code))`, index `(event_code)`; `ALTER TABLE deities DROP COLUMN IF EXISTS lunar_memorial_days`. Depends on `0031_calendar_events.sql`. |
| Make / data | `Makefile` | `ensure-deities` / `seed-deities`. **(v3.0)** `ensure-data` runs `ensure-calendar-events` before `ensure-deities` (FK + code check), `ensure-deities` depends on `ensure-calendar-events`. |
| E2E harness | `services/appService/e2e/harness.go` | Seeds `deities` from `assets/data/phat_botat.json` (minimal columns). **(v3.0)** reads the committed file; vía tests also need a minimal seed of calendar events and links. |
| API spec | `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` | Has the 3 routes + schemas (tag Deities). **(v3.0) not yet updated**: `page_size` 12, `date`, `INVALID_DATE`, `next_via`, `via_events`, `lunar_memorial_days` deprecated, image examples (currently `http://192.168.1.186:8080/...`). Change in the same pass as the handler. |
| ERD | `docs/specs/db/erd.mmd`, `erd.svg` | Has the `deities` table. **(v3.0)** add `deity_via_events` + its two FKs, drop `lunar_memorial_days`. |

Note on routes: `GET /v1/deities/top` is a literal segment, so `ServeMux` prefers it over `GET /v1/deities/{id}`; `GET /v1/deities/{id}/sutras` (the `sutra` spec) is longer, so it does not clash with detail. The new parameter (`date`) is a query parameter and does not change the route table.

## Request flow

From v3.0 all three read routes have several steps (read deities → read the calendar catalogue → compute vía), so each route has its own Step-by-step Flow below.

> See `flow.md` for the Mermaid sequence diagrams (a 1:1 mirror of the Step-by-step Flows below).

## Step-by-step Flow — Paged list of Buddhas and Bodhisattvas (`GET /v1/deities?type=&page=N&date=`)

```
Step 1 — Read `page`, `type`, `date` (handler)
- Trigger: a request reaches the public route
- Pre-condition: —
- Call: r.URL.Query().Get("page"); empty → page = 1, otherwise strconv.Atoi. deityType = Get("type") (raw string). (v3.0) date = Get("date") (raw string)
    - Success → Step 2
    - Failure (page is not an integer, including int overflow) → ErrInvalidPage → 400 INVALID_PAGE, details.field = "page"

Step 2 — Check `page`, then `type`, then `date` (usecase)
- Trigger: the handler calls DeityUsecase.ListDeities(ctx, deityType, page, date)
- Pre-condition: page is an integer
- Call: page < 1 or > math.MaxInt32; then deityType != "" and !IsValidDeityType; then (v3.0) date empty → today = now() in calendar.timezone, otherwise parse YYYY-MM-DD, year 1900..2100
    - Success → Step 3 (offset = (page-1) * page_size on the filtered list; page_size = 12 from v3.0)
    - Failure (page) → ErrInvalidPage → 400 INVALID_PAGE
    - Failure (unknown type) → ErrInvalidDeityType → 400 INVALID_DEITY_TYPE, details.field = "type"
    - Failure (v3.0, invalid date) → ErrInvalidDate → 400 INVALID_DATE, details.field = "date"; DB not touched yet

Step 3 — Count the deities (repository)
- Trigger: the usecase calls DeityRepository.List(ctx, deityType, offset, limit)
- Pre-condition: parameters are valid
- Call: SELECT count(*) FROM deities [WHERE type = $1]
    - Success → total (matching the filter), go to Step 4
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 4 — Fetch the page (repository)
- Trigger: Step 3 done
- Pre-condition: total is known
- Call: SELECT <columns> (v3.0: plus ARRAY(SELECT event_code FROM deity_via_events …) AS via_event_codes) FROM deities d [WHERE type = $1] ORDER BY sort_order, id OFFSET $n LIMIT $m
    - Success → items (empty if offset is past the end) and total, go to Step 5
    - Failure (DB / scan error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — Step 3 and Step 4 are two independent statements, not sharing a snapshot

Step 5 — Read the festival-calendar catalogue (v3.0, calendar repository)
- Trigger: Step 4 returned; skipped when items is empty or no deity has via_event_codes
- Pre-condition: lang = calendar.default_language
- Call: CalendarEventRepository.ListActiveEvents(ctx, lang) — one query, only is_active events (BR-CAL-16), with rule, name, sort_order
    - Success → Step 6
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR (no page returned without vía)

Step 6 — Compute the next vía occurrence (v3.0, pure domain)
- Trigger: Step 5 done
- Pre-condition: today (Step 2), the list of active events
- Call: for each deity: filter events by ViaEventCodes (non-active code → dropped) → ResolveDeityVias(events, today) → NextVia
    - Success → Step 7
    - Failure → no error branch (an invalid rule is skipped by ResolveOccurrences, as in calendar)

Step 7 — Build the page and return the response (usecase + handler)
- Trigger: Step 6 done (or Step 4/5 when skipped)
- Pre-condition: items, total
- Call: nil items → []; total_pages = (total + page_size - 1) / page_size; the handler maps to the summary DTO (+ next_via, null when none)
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → no new error branch
```

## Step-by-step Flow — Featured Buddhas and Bodhisattvas (`GET /v1/deities/top?date=`) (v3.0)

Before v3.0 this was one SQL query; v3.0 adds reading the calendar catalogue and computing vía, so it now has several steps.

```
Step 1 — Read `date` (handler)
- Trigger: a request reaches the public route
- Pre-condition: —
- Call: date = r.URL.Query().Get("date") (raw string) → DeityUsecase.Top(ctx, date)
    - Success → Step 2
    - Failure → none (the handler does not check it itself)

Step 2 — Determine "today" (usecase)
- Trigger: Step 1
- Pre-condition: —
- Call: date empty → today in calendar.timezone; otherwise parse as in list Step 2
    - Success → Step 3
    - Failure (invalid date) → ErrInvalidDate → 400 INVALID_DATE, details.field = "date"; DB not touched yet

Step 3 — Fetch the featured deities (repository)
- Trigger: Step 2 done
- Pre-condition: limit = deity.top_limit (10)
- Call: SELECT <columns> + via_event_codes FROM deities ORDER BY sort_order, id LIMIT $1
    - Success → items (may be empty), go to Step 4
    - Failure (DB / scan error) → 500 INTERNAL_ERROR

Step 4 — Read the festival-calendar catalogue (calendar repository)
- Trigger: Step 3 done; skipped when no deity has via_event_codes
- Call: CalendarEventRepository.ListActiveEvents(ctx, lang)
    - Success → Step 5
    - Failure (DB error) → 500 INTERNAL_ERROR

Step 5 — Compute the next vía occurrence (pure domain)
- Trigger: Step 4 done
- Call: as in list Step 6
    - Success → Step 6
    - Failure → no error branch

Step 6 — Return the response (handler)
- Trigger: Step 5 done
- Call: map to the summary DTO (+ next_via); nil items → []
    - Success → 200 {items}
    - Failure → no new error branch
```

## Step-by-step Flow — Detail of one deity (`GET /v1/deities/{id}?date=`) (v3.0)

```
Step 1 — Read `id`, `date` (handler)
- Trigger: a request reaches the public route
- Pre-condition: —
- Call: r.PathValue("id"), r.URL.Query().Get("date") → DeityUsecase.Get(ctx, id, date)
    - Success → Step 2
    - Failure → none

Step 2 — Determine "today" (usecase)
- Trigger: Step 1
- Call: as in top Step 2
    - Success → Step 3
    - Failure (invalid date) → 400 INVALID_DATE; DB not touched yet (date is checked before looking up id)

Step 3 — Find the deity (repository)
- Trigger: Step 2 done
- Call: SELECT <columns> + via_event_codes FROM deities WHERE id = $1
    - Success → deity, go to Step 4
    - Failure (sql.ErrNoRows) → ErrDeityNotFound → 404 DEITY_NOT_FOUND
    - Failure (other DB error) → 500 INTERNAL_ERROR

Step 4 — Read the festival-calendar catalogue (calendar repository)
- Trigger: Step 3 done; skipped when via_event_codes is empty
- Call: CalendarEventRepository.ListActiveEvents(ctx, lang)
    - Success → Step 5
    - Failure (DB error) → 500 INTERNAL_ERROR

Step 5 — Compute every vía day and the next occurrence (pure domain)
- Trigger: Step 4 done
- Call: ResolveDeityVias(the deity's events, today) → vias (each active event once, Next or nil, sorted per AC 28); NextVia(vias)
    - Success → Step 6
    - Failure → no error branch

Step 6 — Return the response (handler)
- Trigger: Step 5 done
- Call: map to the detail DTO: next_via, via_events, lunar_memorial_days (derived from the via_events that have a next_occurrence: that occurrence's lunar day/month + the event name), nil arrays → []
    - Success → 200 DeityDetail
    - Failure → no new error branch
```

## Step-by-step Flow — Loading data with `cmd/seeddeities`

A manually run command, **not** part of appService startup. (v3.0) Needs the calendar catalogue loaded first (`make ensure-calendar-events`).

```
Step 1 — Read the source file (seeddeities/main.go)
- Trigger: the operator runs `go run ./services/appService/cmd/seeddeities [-file <path>]`
- Pre-condition: the current directory is the repo root when using the default path `assets/data/phat_botat.json` (v3.0: the file is in git)
- Call: os.ReadFile(*file)
    - Success → Step 2
    - Failure (no file / unreadable) → log.Fatalf "read <file>", exit; DB not touched yet

Step 2 — Validate the data (seeddeities/parse.go)
- Trigger: Step 1 done
- Pre-condition: the file content is in memory
- Call: parseDeities → json.Unmarshal → each entry: id non-empty/not duplicated, type ∈ {BUDDHA, BODHISATTVA, DEVA}, name_vi non-empty; (v3.0) via_event_codes: each code non-empty, no duplicates within the same entry; SortOrder = index in the file
    - Success → []domain.Deity, go to Step 3
    - Failure (broken JSON, no entries, empty/duplicate id, unknown type, empty name_vi, v3.0 empty/duplicate vía code) → log.Fatalf "parse <file>"; DB not touched yet

Step 3 — Load config and connect to the DB (seeddeities/main.go)
- Trigger: Step 2 done
- Pre-condition: `CONFIG_PATH` (default `services/appService/config/config.dev.yaml`)
- Call: pkgconfig.LoadSource/LoadYAML → appConfig.NewMockSecretGetterFromYAML (⚠️ DEV ONLY) → appConfig.Load → database.New + Connect (30-second timeout)
    - Success → Step 4
    - Failure (missing config/secret, cannot connect to the DB) → log.Fatalf; nothing written yet

Step 4 — Check that the calendar event codes exist (v3.0, repository)
- Trigger: Step 3 done
- Pre-condition: DB connected; migration 0031 + the vía-link migration applied; calendar catalogue loaded
- Call: DeityRepository.MissingCalendarEvents(ctx, every via_event_codes code) — SELECT the codes not present in calendar_events
    - Success (none missing) → Step 5
    - Failure (some codes missing) → log.Fatalf listing the missing codes; nothing written
    - Failure (DB error) → log.Fatalf; nothing written

Step 5 — Upsert every entry and the vía links (repository)
- Trigger: Step 4 done
- Pre-condition: data valid, every code exists
- Call: DeityRepository.Upsert → INSERT INTO deities (<columns>) … ON CONFLICT (id) DO UPDATE SET <every column>, updated_at = now(); (v3.0) DELETE FROM deity_via_events WHERE deity_id = ANY($ids); INSERT INTO deity_via_events (deity_id, event_code) for each link
    - Success → commit, go to Step 6
    - Failure (DB error: CHECK type, FK event_code because a code was deleted between Step 4 and 5, missing table, lost connection) → full rollback → log.Fatalf "seed deities"
    - Transaction boundary: every INSERT/UPDATE/DELETE of Step 5 shares ONE transaction — all or nothing

Step 6 — Log (seeddeities/main.go)
- Trigger: Upsert succeeded
- Call: log.Printf "seeded %d deities from %s"
    - Success → exit 0
    - Failure → no error branch
```

Note: a deity removed from the file stays in the table (there is no cleanup step); its vía links stay unchanged because the `DELETE` only touches the `id`s present in the file.

## Deliberately stubbed for now (if applicable)

No stubbed code. Everything **(v3.0)** is not yet done (not a stub) — `tasks.md` Phase 8–10.

## Known gaps carried by this feature

- **Catalogue file missing and ignored**: `assets/data/phat_botat.json` matches `*.json` in `.gitignore`; this checkout has no such file, so `TestParseDeities_RealDataFile` fails (`open ../../../../assets/data/phat_botat.json: no such file or directory`, run 2026-10-15) and the E2E harness cannot seed. Fixed in Phase 8 (AC 33).
- **The calendar catalogue is not in git either** (`assets/data/calendar-events.json`, `calendar` spec E26 not done) — deity v3.0 depends on it for both the seed (FK) and E2E.
- **E2E covers only the `type` filter** (`e2e/deity_e2e_test.go`); `top` and `detail` have none yet.
- **The repository test uses a real DB**: `postgres_deity_repository_test.go` needs Postgres (`TEST_DB_NAME=innerlamp_test` to avoid `TRUNCATE`-ing dev data), skipped under `-short`.
- **The seed uses dev config** (`⚠️ DEV ONLY`), not yet usable for staging/production (product Appendix A item 4).
- **Upsert overwrites every column and never deletes** entries removed from the file.
- **Seeder validation is partial**: mantras, đại nguyện (great vows), etc. are not checked.
- **Count and page fetch do not share a snapshot**: data only changes on seed, low risk.
- **(v3.0) Two-year window**: an event that only exists in years with a leap month may have no occurrence in `Y`..`Y+1` → `next_occurrence: null` even though the deity has a link.
- **(v3.0) Every request reads the whole calendar catalogue** (~51 events) to filter by code: cheap at the current size; if the catalogue grows, replace it with a query by code list.
- **(v3.0) The response depends on "today"**: no public `Cache-Control` added; without `date` the result changes with the date in `calendar.timezone`.
- **No OKF for deity yet** (`docs/knowledge/features/deity.md`).
