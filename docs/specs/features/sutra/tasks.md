# Tasks — sutra

Version: 4.0
Last updated: 2026-10-16
Status: Phases 0–6 are done and were verified (2026-10-03 to 2026-10-11, evidence below; `go build`, `go vet`, `go test -short` on the whole repo, repository tests on a real Postgres 16, the full E2E suite against the real binary). **Phase 7 (v4.0) is open**: nothing in it is implemented. The search route (Phase 3–4) was shipped after v3.0 was written and is tested at the usecase, repository and E2E level.
Module: `sutra`

> See `changelog.md` in this folder for the version history of this spec.

Every item traces to a row of the "Layers & components" table or a Step of a "Step-by-step Flow" in `design.md`. An item is `[x]` only when verified: the code is written, `go build`/`go vet` are clean and its test passes (repository tests ran on a real Postgres, not a mock). Evidence (numbers, test names) is in the item; the old audit of the 35 acceptance criteria of v3.0 is `test-coverage-report.md` (old AC numbering — see the crosswalk in `changelog.md`; to be regenerated).

## Phase 0 — Data (`assets/data/sutras/`, "Data file")

- [x] Add `translations[].key` to every version of the 153 texts (`<lang>-<variant>`, suffix `-2`, `-3` on repeats; only `tam-kinh` repeats) — 2026-10-03, 153 files checked, `key` unique within each text
- [x] Add `dailyChant` (true for 34 texts), `purposes` (6 texts have `[]`) and `durationMinutes` to the 153 texts, assigned by the author with no reviewer — 2026-10-03; durations suggested as `ceil(syllables ÷ 80)` and rounded; the estimates were kept as generated (TD-007, A-005); the 10 Vajrayāna texts had none until migration `0019` and the matching value in the seed JSON
- [x] Update `assets/data/sutras/index.json` and `assets/data/sutras/README.md` (the four fields) — `index.json` matches every file (153/153)
- [x] Split `reference` / `license` and add `originalTitle`, `author`, `hanTranslator`, `hanTranslationYear`, `referenceNote`, `summary`, `licenseUrl`, `licenseNote` for the 153 files and `index.json` — 2026-10-07; `TestParseSutraSeed_ShippedDataIsSplit` (every text has a summary; every licence is in the closed set)

## Phase 1 — Domain

- [x] `Sutra`, `SutraTranslation`, `SutraSection`, `SutraLine`, `SutraDeity`, `DeitySutra`, `SutraPage`, `SutraLinesPage`, `SutraFilter`, `SutraFilterInput`, `SutraSeed`, `SutraSeedResult` and the enum constants in `services/appService/domain/sutra.go` (no tags; `DurationMinutes *int`, `AudioURL string`)
- [x] Licence codes and `ValidLicense`; the split fields on `Sutra` and `SutraTranslation`
- [x] Errors `ErrSutraNotFound`, `ErrSutraTranslationNotFound`, `ErrInvalidSutraFilter` / `InvalidSutraFilterError` (carries the field), `ErrInvalidPageSize`, `ErrInvalidTranslation`; reuse `ErrInvalidPage` and `ErrDeityNotFound`
- [x] Interface `SutraRepository` (`Count`, `List`, `FindByID`, `FindTranslation`, `ListLines`, `CountByDeity`, `ListByDeity`, `MissingDeities`, `InsertAll`; the order `sort_order ASC, id ASC` is in the contract)
- [x] `domain/mock_sutra_repository.go` for the usecase tests

## Phase 2 — Usecase

- [x] `usecase/sutra_usecase.go`: constructor takes `SutraRepository`, `DeityRepository` and `config.SutraConfig`
- [x] `List` — validates `page` ∈ [1, MaxInt32] and every filter in `parseSutraFilter` (enums, `daily`, `max_minutes` ≥ 1, no empty value, `q` trimmed and 1–100 runes) → `ErrInvalidPage` / `InvalidSutraFilterError{Field}`; int64 offset; `total_pages` (Steps 2, 7 of the list flow; AC 1, 3–7). Tests in `usecase/sutra_usecase_test.go`: `empty q`, `blank q`, `q over 100 characters`, a trimmed `"  bát nhã "`
- [x] `Get` — calls `FindByID`, returns `ErrSutraNotFound` (Steps 1–3 of the detail flow)
- [x] `Lines` — validates `page`/`page_size`, clamps to `lines_max_page_size`, defaults to `lines_page_size`, converts to `fromN`/`toN`, `total_pages` (Steps 2, 5 of the lines flow)
- [x] `ListByDeity` — validates `page` and `role`, calls `DeityRepository.FindByID`, then `CountByDeity` and `ListByDeity`, `total_pages` (Steps 2–5, 8 of the by-deity flow)
- [x] `SeedIfEmpty(ctx, load)` — `Count` → `load()` → `MissingDeities` → `InsertAll`; does not call `load` when the table has rows (Steps 1–4 of the load flow)
- [x] Map repository errors to the sentinels (`errors.Is`-compatible)
- [x] Table-driven unit tests with `testify`: every error branch of the five flows, the clamp, rounding up of `total_pages`, `page > total_pages`, `SeedIfEmpty` skipping and aborting on missing deities

## Phase 3 — Repository, seed and embedded data

- [x] `migrations/0013_sutras.sql` — the 4 tables, the `CHECK`s, foreign keys (cascade except `deities`) and indexes; idempotent; `reference` is `TEXT` (`lang-nghiem` has 135 characters)
- [x] `migrations/0019_sutra_audio_search.sql` — `audio_url TEXT NOT NULL DEFAULT ''`, `CREATE EXTENSION IF NOT EXISTS unaccent`, duration backfill of the rows still `NULL`; idempotent
- [x] `migrations/0020_sutra_reference_license_summary.sql` — 6 columns on `sutras`, 2 on `sutra_translations`; idempotent
- [x] `repository/postgres_sutra_repository.go` — `sutraWhere` (filters, `q` with `unaccent` and escaped wildcards, bound parameters only), `Count`/`List`, `FindByID`, `FindTranslation`, `ListLines` (range on `n`), `CountByDeity`, `ListByDeity`, versions and links with `= ANY($1)`; explicit columns, `context` throughout, errors wrapped with `%w`; `NULL` ↔ `nil`; `pq.Array` for `purposes`
- [x] `MissingDeities` and `InsertAll` — ONE transaction, `ON CONFLICT (id) DO NOTHING RETURNING id`, multi-row line batches (Steps 3–4 of the load flow)
- [x] `repository/sutra_seed.go` — `ParseSutraSeed(fsys fs.FS)` checks everything in AC 37, assigns `SortOrder`, computes `sections`; tests with `fstest.MapFS` for each condition (missing file, duplicate / mismatched id, enum, key, `n` gap, repeated deity, licence, year, `durationMinutes` = 0) and for valid data
- [x] `assets/embed.go` — `SutrasFS embed.FS` with `//go:embed data/sutras/*.json`
- [x] `cmd/seedsutras/main.go` — loads config, connects, builds repository + usecase, calls `SeedIfEmpty` with the embedded data or `os.DirFS(-dir)`; rows present → "already seeded", exit 0; error → non-zero (AC 37–43); tests of the source choice and "reads nothing when seeded" in `cmd/seedsutras/main_test.go`; run for real: seeded DB → exit 0, empty DB → 153 / 391 / 64,009 / 151, missing deity → exit 1 with the ids
- [x] Repository tests on a real Postgres (`repository/postgres_sutra_repository_test.go`, skipped with `-short`): `InsertAll` in one transaction, rollback on error, `ON CONFLICT` skip, the `n` range of `ListLines`, filters and `total`, `TestSutraRepository_Search`, `TestSutraRepository_AudioURLDefaultsToEmptyString`

## Phase 4 — Delivery

- [x] DTOs in `delivery/http/sutra_handler.go` (`sutraHeadResponse` shared fields incl. `summary` and `audio_url`, summary, detail with the six detail-only fields, translation, line, lines, deity, list, by-deity) — JSON tags only here
- [x] Handlers `List` (incl. `q`), `Get`, `Lines`, `ListByDeity`: read the query → call the usecase → map to DTOs (`nonNil` for arrays; `null` for `duration_minutes`, `section`, `han_translation_year`)
- [x] Error map in `delivery/http/errors.go` (`SUTRA_NOT_FOUND`, `TRANSLATION_NOT_FOUND`, `INVALID_FILTER`, `INVALID_PAGE_SIZE`, `INVALID_TRANSLATION`)
- [x] `SutraConfig{PageSize, LinesPageSize, LinesMaxPageSize}` and the `sutra:` block in `config/config.go` and `config.dev.yaml`; `Validate()` rejects values < 1 and `lines_page_size` > `lines_max_page_size`; called by `cmd/main.go` right after `Load`; tests `TestSutraConfig_Validate`, `TestLoad_DevYAMLSutraConfig`
- [x] `cmd/main.go` — wires the routes; **does not seed on start** (since 2026-10-11); registers `GET /v1/sutras`, `/v1/sutras/{id}`, `/v1/sutras/{id}/lines`, `/v1/deities/{id}/sutras`; no route wrapped in `RequireAuth` (AC 36)
- [x] `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml` — the four routes, schemas `SutraSummary` / `SutraDetail` / `Line`; `q` and `audio_url` are in the contract (the follow-up in Phase 7 only tightens the wording and `duration_minutes`)
- [x] Handler tests (`delivery/http/sutra_handler_test.go`, `mock_sutra_usecase_test.go`): success, every error branch (400/404/405/500), `[]` not `null`, `null` for `duration_minutes` / `section`, `page > total_pages` → `items: []`, `audio_url` is `""` (never null, never omitted) and carries a real URL when set
- [x] `e2e/harness.go` — seeds `deities` after the migrations; prints the binary's stderr when it does not start

## Phase 5 — Integration and documentation

- [x] Run on a real Postgres: migrations, `cmd/seeddeities`, start the service, `cmd/seedsutras` on an empty and on a seeded database — 2026-10-03, throwaway Postgres 16: empty DB → 153 / 391 / 64,009 / 151; seeded DB → exit 0, nothing written; missing deity → exit 1
- [x] Time the first load and size the binary — 2026-10-03, 5 runs each: first run median 1.82 s (1.73–2.93 s) to the first `200`, later runs 0.13 s; binary 70.91 MB with the library vs 60.66 MB (+10.25 MB, +17%)
- [x] Time the responses — 2026-10-03, loopback, 40 requests after 5 warm-ups: list page 1 (20 cards) 2.49 ms (p95 2.74), 28.4 KB (gzip 2.0 KB); detail `tam-kinh` 1.73 ms, `phap-hoa` 2.08 ms (27 sections, 7.5 KB); 200 lines of `phap-hoa/vi` 1.81 ms, 21.2 KB; `page_size=500` 1.94 ms, 49.6 KB; by-deity `buddha_shakyamuni` (81 texts) 3.47 ms. Not measured under concurrent load or over a real network; **search not measured** (Phase 7)
- [x] E2E in `services/appService/e2e/sutra_e2e_test.go` against the real binary and an ephemeral Postgres: the four routes, filters, search (`KINH HANH PHUC`, accented, English title, no hit, `q` ANDed with `tradition`, empty / blank `q` → `INVALID_FILTER`), the load command and its refusals
- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` clean on the whole repo; the full E2E suite passes
- [x] ERD `docs/specs/db/erd.mmd` / `erd.svg`, `migrations/README.md`, `Makefile` (`ensure-sutras`, `seed-sutras`, `seed`), `docs/development/local-stack.md`
- [x] OKF concept `docs/knowledge/features/sutra.md`, assumptions `docs/assumptions/sutra/A-001…011`, tech debt `docs/tech-debt/sutra/TD-001…010` — written; the OKF concept predates search, `audio_url` and the split fields and is stale (follow-up for `il-okf-writer`, see below)

## Phase 6 — Split fields and summary (v2.12–v2.13, now AC 10–13, 16, 19, 37)

- [x] Seed parser reads the new fields (still `DisallowUnknownFields`), checks `license` against the closed set and `hanTranslationYear`; tests for each new error branch and for the `""` defaults
- [x] `InsertAll` writes the new columns; `FindByID` reads the five detail-only columns; the detail reads `license_url` / `license_note`; repository tests round-trip every field (`han_translation_year` NULL ↔ `nil`)
- [x] Detail DTO with the five detail-only fields and per-version `license_url` / `license_note`; `summary` on every card (`sutraColumns` reads `s.summary`); handler and E2E tests (`a-di-da` returns `han_translator`, `han_translation_year` = 402, a non-empty `summary`; the list carries `summary`)
- [x] Applied to the dev database — checked 2026-10-08: 153/153 texts match the JSON, no old-style `reference`, all 411 version licences in the closed set. An environment with real data would need its own backfill script; none exists yet

## Phase 7 — v4.0 (open)

- [ ] `repository/sutra_seed.go` — `ParseSutraSeed` refuses a text whose `durationMinutes` is missing or `null` (AC 38); update the seed-parser tests so a `null` / missing `durationMinutes` is an invalid case and check the real data file where it lives: all 153 texts carry a value (the JSON is not in the git checkout)
- [ ] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` — `duration_minutes` non-nullable (AC 34); `q`: "1–100 characters after trimming; empty → `400 INVALID_FILTER`"; `audio_url`: "`\"\"` = no audio" (AC 5, 20); keep the two copies identical. Follow-up of this spec — done in the code change, not in this docs PR
- [ ] After the contract change: `SutraSummary.DurationMinutes` becomes `int` in the DTO if the domain no longer holds `null`, otherwise keep `*int` for rows read from a database loaded before v4.0; decide in the implementing change and record it in `design.md`
- [ ] `repository/sutra_vectors_test.go` — run every case of `docs/product/sutra/rules/sutra-browse.json` and `sutra-search-match.json` against a real Postgres: insert each case's illustrative texts, apply its topic / `max_minutes` / term, compare the identifiers and their order (AC 4, 8). The files `sutra-default-version.json` and `sutra-daily-quote.json` belong to the clients (BR-STR-05, BR-STR-10), not to this backend
- [ ] Time `GET /v1/sutras?q=` on the full library as in Phase 5 (a one-term search over 153 texts, with and without accents) and record the figures (AC 44)
- [ ] Regenerate the coverage audit against the 44 criteria of this spec (`golang-e2e-verification`) and close the follow-ups listed in the old `test-coverage-report.md`

## Follow-ups outside this spec (flagged, not edited here)

- `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml` — Phase 7 wording above
- `docs/knowledge/features/sutra.md` — stale (search, `audio_url`, the split fields, the by-deity order); refresh with `il-okf-writer`
- `docs/specs/db/erd.mmd` — check `audio_url` (migration `0019`)
- `docs/specs/features/sutra/test-coverage-report.md` — uses the old AC numbering of v3.0; regenerate or delete in the cleanup step
- `docs/assumptions/sutra/` and `docs/tech-debt/sutra/` — entries that this spec and the product spec now supersede are cleaned up in the final cleanup PR, after the user approves the list

## Deferred (separate feature, not blocking sutra)

- Updating content after the first load with a reseed command or a back office — deferred; today only by hand-written SQL (product Appendix A item 15)
- Delta sync (`updated_since`), `version`, offline-first, downloads — separate specs, each needs an ADR (product §1.2)
- Audio files and playback — a later spec; `audio_url` already carries the "none" value
- Bookmarks, favourites, chant history and counters — module `journal`
- Where the 300 quotes live (bundled or served) — product Appendix A item 3; the backend serves nothing for it today
