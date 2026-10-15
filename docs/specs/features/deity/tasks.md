# Tasks — deity

Version: 3.1
Last updated: 2026-10-15
Status: in-progress — Phase 1–4 and 7 done (except the items explicitly marked `[ ]`); Phase 5 still has E2E for `top`/`detail` and the OKF; **Phase 8–10 (v3.0) not started**; Phase 6 is intentionally deferred work.
Module: `deity`

> See `changelog.md` in this folder for the version history of this spec.

Every item below traces to a row of the "Layers & components" table or a Step of a "Step-by-step Flow" in `design.md`. Only what has been verified in code is `[x]`. The v3.0 pass (2026-10-15) re-read `domain/deity.go`, `usecase/deity_usecase.go`, `delivery/http/deity_handler.go`, `image_url.go`, `errors.go`, `cmd/seeddeities/parse.go`, `cmd/main.go`, migrations `0015`/`0021`, and ran `go test -short ./services/appService/cmd/seeddeities/ ./services/appService/usecase/ -run Deit` — the usecase passes, `TestParseDeities_RealDataFile` **fails** because the catalogue file is missing.

## Phase 1 — Domain

- [x] `Deity`, `MemorialDay`, `Iconography`, `Mantras`, `DeityPage`, constants `DeityBuddha`/`DeityBodhisattva`/`DeityDeva` in `services/appService/domain/deity.go` (no tags)
- [x] `ErrDeityNotFound` in `domain/deity.go`; reuses `ErrInvalidPage` (`domain/temple.go`)
- [x] `DeityRepository` interface (4 methods, the order `sort_order ASC, id ASC` is part of the contract)
- [x] `domain/mock_deity_repository.go` for usecase tests

## Phase 2 — Usecase

- [x] `usecase/deity_usecase.go`: `Top`, `ListDeities` (all types, offset, `total_pages`, rejects `page < 1` / `> MaxInt32`, nil `items` → `[]`), `Get`
- [x] Unit tests: `usecase/deity_usecase_test.go`, hand-written mocks, no real DB

## Phase 3 — Repository and data

- [x] `migrations/0007_deities.sql` — table `deities`, `CHECK` on `type`, JSONB + `TEXT[]`, two indexes; `0015_deities_deva_type.sql` widens the `CHECK` for `DEVA`
- [x] `repository/postgres_deity_repository.go`: `ListTop`, `List`, `FindByID`, `Upsert` (one transaction, `ON CONFLICT DO UPDATE`)
- [x] `cmd/seeddeities/main.go` + `parse.go`: reads the file, checks id/type (3 types)/`name_vi`, `SortOrder` = position, connects with dev config, calls `Upsert`
- [ ] `assets/data/phat_botat.json` (20 entries) in the repo — currently caught by `.gitignore` and missing from this checkout; see Phase 8
- [x] Tests: `cmd/seeddeities/parse_test.go` (`_Validation`, `_OrderFollowsFile`, `_AcceptsDeva` pass under `-short`; `_RealDataFile` **fails** when the file is missing — Phase 8); `repository/postgres_deity_repository_test.go` present, needs Postgres
- [x] Images servable: root-relative paths (`0021_relative_image_urls.sql`), `imageHost` prefixes `server.public_base_url`, 20 `full` + 20 `thumbnails` images in `assets/images/buddhas/` (the URL values in the catalogue file not re-checked because the file is missing)
- [ ] Move the seed off dev config so it can be used on staging/production — blocked by the `⚠️ Not confirmed (product)` (product Appendix A item 4) in `requirements.md`

## Phase 4 — Delivery and documentation

- [x] `delivery/http/deity_handler.go`: `Top`, `ListDeities`, `Get` + summary/detail DTOs (arrays never `null`; image URLs are strings via `imageHost`)
- [x] `delivery/http/errors.go`: `DEITY_NOT_FOUND`, `INVALID_PAGE`, `INVALID_DEITY_TYPE`
- [x] `config`: `DeityConfig`, `deity:` block in `config.dev.yaml`
- [x] `cmd/main.go`: wiring and registration of the 3 routes (no seed call)
- [x] Handler tests: `delivery/http/deity_handler_test.go` + `mock_deity_usecase_test.go`
- [x] `0007` recorded in `migrations/README.md`
- [x] `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml`: 3 routes + schemas (up to v2.3)
- [x] `deities` table in `docs/specs/db/erd.mmd` and `erd.svg`

## Phase 5 — Integration & documentation

- [ ] Black-box E2E in `services/appService/e2e/`: `top`, `detail`, `DEITY_NOT_FOUND`, an unseeded DB returns `items: []` (the `type` filter already exists in `deity_e2e_test.go`)
- [x] `go build ./...` and `go vet ./services/appService/...` clean (v2.3 pass)
- [ ] OKF concept `docs/knowledge/features/deity.md` — `il-okf-writer` creates it after v3.0 ships
- [ ] Review `api-contract.md`: delete the file once the website / iOS / Android deity specs are approved (precedent: `temple`, `calendar`) — Phase 10

## Phase 7 — Filter by `type` (v2.2, AC 20–22; completed 2026-10-12)

- [x] Domain: `ErrInvalidDeityType`, `IsValidDeityType`; `DeityRepository.List` takes `deityType`; mock updated
- [x] Usecase: `ListDeities(ctx, deityType, page)` — checks `page` first, then `type`
- [x] Repository: `WHERE type = $n` for both the count and the page fetch; per-type filter test
- [x] Handler: reads `type`; `ErrInvalidDeityType` → `400 INVALID_DEITY_TYPE`
- [x] `openapi.yaml` (both copies): `type` parameter, `400 INVALID_DEITY_TYPE` response
- [x] `api-contract.md` (iOS client) v1.3
- [x] E2E `TestDeity_TypeFilter_E2E`

## Phase 8 — Data, schema and seed (v3.0; AC 5, 15, 16, 18, 33)

Trace: the Migration, Data, Seed CLI, Repository (`Upsert`, `MissingCalendarEvents`), Config, Make, ERD rows of `design.md`; Steps 1–6 of the seed flow.

- [ ] `.gitignore`: add the exception `!assets/data/phat_botat.json` (next to `administrative-units.json`), commit the 20-entry file (AC 33); `TestParseDeities_RealDataFile` passes again
- [ ] Catalogue file: replace each entry's `lunar_memorial_days` with `via_event_codes` — **blocked by the `⚠️ Not confirmed (product)` on the ngày vía (vía day) ↔ calendar event mapping** in `requirements.md`; also needs `assets/data/calendar-events.json` (`calendar` spec E26) to know the codes
- [ ] `migrations/00NN_deity_via_events.sql` (next free number, `0037` today): table `deity_via_events` + two FKs + index `(event_code)`, `DROP COLUMN IF EXISTS lunar_memorial_days`; record it in `migrations/README.md`
- [ ] `docs/specs/db/erd.mmd` + `erd.svg`: add `deity_via_events`, drop `lunar_memorial_days`
- [ ] Domain: `Deity.ViaEventCodes`, drop `LunarMemorialDays` / `MemorialDay`; `DeityRepository.MissingCalendarEvents`; mock updated
- [ ] `cmd/seeddeities/parse.go`: `via_event_codes` (non-empty, no duplicates within one entry); table test for the two new errors
- [ ] `cmd/seeddeities/main.go`: call `MissingCalendarEvents` before `Upsert`; missing codes → `log.Fatalf` listing the codes (Step 4)
- [ ] Repository `Upsert`: in the same transaction, `DELETE` the links of the `id`s in the file then `INSERT` them again; `MissingCalendarEvents`; Postgres test (`TEST_DB_NAME=innerlamp_test`): links replaced on re-seed, FK error → the whole deity set rolled back, missing codes reported correctly
- [ ] `Makefile`: `ensure-calendar-events` runs before `ensure-deities` in `ensure-data`; `ensure-deities`/`seed-deities` depend on `ensure-calendar-events`
- [ ] `config.dev.yaml`: `deity.page_size: 12` (AC 5, BR-DTY-03)

## Phase 9 — Reads: next vía occurrence and contract (v3.0; AC 2, 8, 12, 22–32, 34)

Trace: the Domain / Vía, Usecase, Repository (reads), Delivery / Handler, DTO, Wiring, API spec, E2E harness rows; Steps 1–7 / 1–6 / 1–6 of the three read flows.

- [ ] `domain/deity_via.go`: `PickNextVia`, `ResolveDeityVias`, `NextVia`, `DeityVia`, `DeityWithVia`; a unit test runs **every case** of `docs/product/deity/rules/deity-next-via.json` read directly from `docs/product/` (asserts `event`, `solar_date`, `days`; `label` belongs to the client — BR-CAL-13) (AC 32, BR-DTY-06); separate tests: an event with no occurrence in `Y..Y+1` → `Next = nil`, sorted last; `Y = 2100` does not resolve `2101`
- [ ] Repository: `deityColumns` includes `ARRAY(SELECT event_code …) AS via_event_codes` for `ListTop`/`List`/`FindByID`; drop JSONB; a Postgres test runs **every case** of `docs/product/deity/rules/deity-order.json` (seed by `position`/`type`, assert `List` and `ListTop` against `order`/`featured`) (AC 32, BR-DTY-02)
- [ ] Usecase: constructor takes `CalendarEventRepository`, `config.CalendarConfig`, `now`; `Top(ctx, date)`, `ListDeities(ctx, type, page, date)`, `Get(ctx, id, date)`; check order `page` → `type` → `date`; skip `ListActiveEvents` when there are no codes; inactive/unlisted events are dropped (AC 24–27); mock tests: empty `date` uses `calendar.timezone`, invalid `date` → `ErrInvalidDate` without touching the repo, a `ListActiveEvents` error → wrapped error
- [ ] Handler + DTO: read `date` for the 3 routes; `next_via` (summary), `via_events`, derived `lunar_memorial_days` (detail); `lunar_date` in the same shape as `GET /v1/calendar/events`; arrays never `null`, `next_via`/`next_occurrence` are `null` when absent (AC 2, 8, 12, 28–30); handler test for 400 `INVALID_DATE` `details.field = "date"`
- [ ] `cmd/main.go`: pass `calendarEventRepo`, `cfg.Calendar`, `time.Now` into `NewDeityUsecase` (build `calendarEventRepo` before deity if needed)
- [ ] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: `page_size` 12, `date` parameter for the 3 routes, `400 INVALID_DATE`, `DeityVia` schema, `next_via`, `via_events`, `lunar_memorial_days` `deprecated: true`, image examples without the dev IP; description referencing `GET /v1/deities/{id}/sutras` for related sutras (AC 34)
- [ ] E2E (`e2e/harness.go` additionally seeds a few calendar events + links): `top`/list have the correct `next_via` for a fixed `date`, detail has `via_events` per AC 28, `date=2026-02-30` → `400 INVALID_DATE`, inactive events do not appear, `page_size = 12`
- [ ] `go build ./...`, `go vet ./...`, `go test -short ./...` clean

## Phase 10 — Cleanup and documentation after v3.0

- [ ] Remove `lunar_memorial_days` from `DeityDetail` (DTO, `openapi.yaml`) — blocked by the `⚠️ Not confirmed` on the removal timing in `requirements.md`
- [ ] Delete `api-contract.md` once the website / iOS / Android deity specs are approved
- [ ] Ask `ba-writer` to update the Backend column of `docs/product/deity/product.md` §4.2 (BR-DTY-03, BR-DTY-05, BR-DTY-06, FR-DTY-002, FR-DTY-006 → done) — this skill does not edit `docs/product/`
- [ ] Web docs `web/docs/src/content/docs/guides/catalogues/deities.md`: page size 12, `next_via`, `via_events`, `date`
- [ ] OKF `docs/knowledge/features/deity.md` via `il-okf-writer` (same item as Phase 5)

## Phase 6 / Deferred (separate feature, not blocking deity)

- Creating/editing/deleting entries via a backoffice (no CRUD spec yet).
- Review of the religious content and image rights (product Appendix A item 1).
- Filtering by several `type` values at once; search (product §1.2: not doing).
