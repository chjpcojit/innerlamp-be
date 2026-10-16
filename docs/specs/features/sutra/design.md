# Design — sutra

Version: 4.0
Last updated: 2026-10-16
Status: implemented (2026-10-03; seed moved to the command 2026-10-11) — domain, usecase, repository, seed parser, handlers, 4 routes, `cmd/seedsutras`, migrations `0013`, `0019`, `0020`, `openapi.yaml`, ERD; run against a real binary + Postgres with its own E2E tests. **v4.0 not implemented**: the seeder requires `durationMinutes` ≥ 1 and the contract drops `null` from `duration_minutes` (AC 34, 38). Still open: `sutraSeedTimeout`-style constants (`seedTimeout` in `cmd/seedsutras`) might belong in `config/`.
Module: `sutra`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

The four routes are all `GET`, public and outside the auth middleware: `/v1/sutras`, `/v1/sutras/{id}`, `/v1/sutras/{id}/lines` and `/v1/deities/{id}/sutras`. The feature touches all four Clean Architecture layers of `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, wired in `cmd/main.go`), owns migrations `0013`, `0019` and `0020`, and extends the `assets/` package to `go:embed` the data directory. It has the shape of `article` (paged list, detail, seed) with three differences: each text has **several versions**, the content is **paged by number of lines**, and a **foreign key to `deities`** makes the seed depend on `cmd/seeddeities`.

The JSON files are a **one-time source**: after the first load Postgres is the only source and every route reads it. `ParseSutraSeed` takes an `fs.FS`, so one parser serves the data embedded in the binary and a directory on disk (`cmd/seedsutras -dir`). The service itself does not seed.

`GET /v1/deities/{id}/sutras` belongs to this feature but is registered under the `/v1/deities/` prefix; `deityHandler` is unchanged, and the usecase reuses `DeityRepository.FindByID` to check that the deity exists.

Search (`q`) and the card filters are one `WHERE` built by `sutraWhere`, shared by `Count` and `List`; the search term is a bound parameter and its LIKE wildcards are escaped.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/sutra.go` | `Sutra`, `SutraTranslation`, `SutraSection`, `SutraLine`, `SutraDeity`, `DeitySutra` (= `Sutra` + `Role`), `SutraPage`, `SutraLinesPage`, `SutraFilter` (incl. `Query *string`), `SutraFilterInput` (raw strings incl. `Q`), `SutraSeed`; enums `Tradition`, `Kind`, `Variant`, `DeityRole`, `Purpose`; licence codes (`LicenseCC0` = `CC0-1.0`, `LicenseCCBYNC`, `LicensePublicDomain`, `LicenseCopyAllowed`, `LicenseUnknown`) and `ValidLicense`. `DurationMinutes *int`, `AudioURL string` (`""` = none), `HanTranslationYear *int`. No JSON/db tags. |
| Domain / Errors | `domain/sutra.go` | `ErrSutraNotFound`, `ErrSutraTranslationNotFound`, `InvalidSutraFilterError` (carries the field), `ErrInvalidPageSize`, `ErrInvalidTranslation`. Reuses `ErrInvalidPage` (`domain/temple.go`) and `ErrDeityNotFound` (`domain/deity.go`). |
| Domain / Repository interface | `domain/sutra.go` | `SutraRepository`: `Count(ctx, filter)`, `List(ctx, filter, offset, limit)`, `FindByID`, `FindTranslation(ctx, sutraID, key)`, `ListLines(ctx, translationID, fromN, toN)`, `CountByDeity`, `ListByDeity(ctx, deityID, role, offset, limit)`, `MissingDeities(ctx, ids)`, `InsertAll(ctx, []SutraSeed)`. The order `sort_order ASC, id ASC` is part of the contract. |
| Domain / Mock | `domain/mock_sutra_repository.go` | Hand-written mock for usecase tests. |
| Usecase | `services/appService/usecase/sutra_usecase.go` | `List` (validates `page` and every filter in `parseSutraFilter` incl. `q`: trimmed, 1–100 runes, `maxSutraQueryRunes`), `Get`, `Lines` (validates `page`/`page_size`, clamps `page_size`, converts to an `n` range), `ListByDeity` (validates `role`/`page`, checks the deity exists), `SeedIfEmpty(load)`. Page sizes come from `config.SutraConfig`. |
| Repository | `services/appService/repository/postgres_sutra_repository.go` | SQL for the 4 tables; `sutraWhere` builds the `WHERE` (filters, `q` with `unaccent(lower(…)) LIKE … ESCAPE '\'` over `collection || ' ' || reference` and every value of `titles`); `List`/`Count`; versions and deity links loaded with `= ANY($1)`; `ListLines` queries the key range `(translation_id, n)`; `InsertAll` runs ONE transaction with `ON CONFLICT (id) DO NOTHING` for `sutras` and multi-row inserts for lines; `pq.Array` for `purposes`; `NULL` ↔ `nil` for `duration_minutes`/`section`/`han_translation_year`. |
| Repository / Seed parser | `services/appService/repository/sutra_seed.go` | `ParseSutraSeed(fs.FS)`: reads `index.json` + each `<id>.json`, checks every condition of AC 37 (and, from v4.0, AC 38), assigns `SortOrder` = position in `index.json`, computes `sections`. Takes `fs.FS` so tests use `fstest.MapFS`. |
| Embedded data | `assets/embed.go`, `assets/data/sutras/*.json` | `SutrasFS embed.FS` with `//go:embed data/sutras/*.json`; only `*.json`. The files are kept out of git (TD-001). |
| Seed command | `services/appService/cmd/seedsutras/main.go` | The only way the library is loaded: loads config, connects to the DB like `cmd/seeddeities`, builds repository + usecase, calls `SeedIfEmpty` with the embedded data (or `os.DirFS(-dir)`). A table with rows → logs "already seeded" and exits 0; invalid data or missing deities → exit non-zero. Source choice lives in `run`/`dataFS` so it is testable. |
| Delivery / Handler | `services/appService/delivery/http/sutra_handler.go` | `List`, `Get`, `Lines`, `ListByDeity`; reads the query (`page`, `page_size`, `tradition`, `kind`, `collection`, `purpose`, `daily`, `max_minutes`, `lang`, `q`, `translation`, `role`); maps entities to DTOs; `nonNil` for arrays. |
| Delivery / DTO | `delivery/http/sutra_handler.go` | `sutraHeadResponse` (fields shared by every shape, incl. `summary` and `audio_url`), `sutraSummaryResponse`, `sutraDetailResponse` (the detail-only fields of AC 12), `sutraTranslationResponse`, `sutraLineResponse`, `sutraLinesResponse`, `sutraDeityResponse`, `sutraListResponse`, `sutraByDeityItemResponse`, `sutraByDeityListResponse` — JSON tags only here. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrSutraNotFound` → `404 SUTRA_NOT_FOUND`; `ErrSutraTranslationNotFound` → `404 TRANSLATION_NOT_FOUND`; `InvalidSutraFilterError` → `400 INVALID_FILTER` (`details.field`); `ErrInvalidPageSize` → `400 INVALID_PAGE_SIZE`; `ErrInvalidTranslation` → `400 INVALID_TRANSLATION`; `ErrInvalidPage` → `400 INVALID_PAGE` and `ErrDeityNotFound` → `404 DEITY_NOT_FOUND` already exist. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `SutraConfig{PageSize, LinesPageSize, LinesMaxPageSize}` and the `sutra:` block (dev: 20, 200, 500). `SutraConfig.Validate()` rejects values < 1 and `lines_page_size` > `lines_max_page_size`; called by `cmd/main.go` right after `Load`. |
| Wiring | `services/appService/cmd/main.go` | Builds repository → usecase → handler; **does not seed on start**; registers the 4 routes with no `RequireAuth`. No `⚠️ DEV ONLY` block involved. |
| E2E | `services/appService/e2e/sutra_e2e_test.go`, `e2e/harness.go` | Real binary + testcontainers Postgres; the harness seeds `deities` from `assets/data/phat_botat.json` after the migrations. |
| Migration | `migrations/0013_sutras.sql` | 4 tables + indexes (see "Data model"). Idempotent. |
| Migration | `migrations/0019_sutra_audio_search.sql` | `audio_url TEXT NOT NULL DEFAULT ''`, `CREATE EXTENSION unaccent`, and the duration backfill (only rows still `NULL`: lines of the first version ÷ 4, rounded up, at least 1). Idempotent. |
| Migration | `migrations/0020_sutra_reference_license_summary.sql` | 6 columns on `sutras` and 2 on `sutra_translations`. Idempotent; does not change existing data. |
| Data | `assets/data/sutras/*.json`, `index.json` | One file per text plus the index (position = library order). See "Data file". |
| API spec | `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` | 4 routes (tag Sutras) and schemas `SutraSummary` / `SutraDetail` / `Line`. **Follow-up (not edited here):** state `q` (1–100 characters, empty → `400 INVALID_FILTER`), `audio_url` as "none = `\"\"`", and — from v4.0 — `duration_minutes` non-nullable. |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | The 4 tables and relations; follow-up check against `0019` (`audio_url`). |

Route note: `"GET /v1/sutras/{id}/lines"` and `"GET /v1/sutras/{id}"` are patterns of different length and do not collide; `"GET /v1/deities/{id}/sutras"` differs from the existing `"GET /v1/deities/{id}"`.

## Data model

```
sutras
  id               VARCHAR(128) PRIMARY KEY            -- stable slug
  sort_order       INTEGER NOT NULL                    -- position in index.json
  tradition        VARCHAR(16) NOT NULL CHECK (IN 'THERAVADA','MAHAYANA','VAJRAYANA')
  kind             VARCHAR(16) NOT NULL CHECK (IN 'SUTRA','VERSE','DHARANI')
  collection       VARCHAR(128) NOT NULL DEFAULT ''
  reference        TEXT NOT NULL DEFAULT ''            -- citation code only, e.g. 'Taishō T0450', 'MN 86'
  original_title   TEXT NOT NULL DEFAULT ''            -- full original title (or the sutra a mantra is taken from)
  author           TEXT NOT NULL DEFAULT ''            -- author (not the translator)
  han_translator   TEXT NOT NULL DEFAULT ''            -- translator into Chinese
  han_translation_year INTEGER CHECK (han_translation_year >= 1)  -- CE; NULL = unknown
  reference_note   TEXT NOT NULL DEFAULT ''            -- the rest of the source remark
  summary          TEXT NOT NULL DEFAULT ''            -- Vietnamese summary, '' when none
  titles           JSONB NOT NULL                      -- {"vi": "...", "en": "..."}; always has "vi"
  daily_chant      BOOLEAN NOT NULL DEFAULT false
  purposes         TEXT[] NOT NULL DEFAULT '{}'
  duration_minutes INTEGER CHECK (duration_minutes >= 1)   -- editorial estimate; v4.0: never NULL in the contract
  audio_url        TEXT NOT NULL DEFAULT ''            -- '' = no recording
  review_status    VARCHAR(16) NOT NULL DEFAULT 'UNREVIEWED'
  created_at, updated_at TIMESTAMPTZ

sutra_translations
  id           BIGSERIAL PRIMARY KEY
  sutra_id     VARCHAR(128) NOT NULL REFERENCES sutras(id) ON DELETE CASCADE
  key          VARCHAR(64) NOT NULL                    -- e.g. vi-chant, vi-meaning-2
  ordinal      INTEGER NOT NULL                        -- order in the file
  lang         VARCHAR(8) NOT NULL
  variant      VARCHAR(16) NOT NULL CHECK (IN 'translation','chant','meaning','poem')
  translator, source_name, source_url, license, note   TEXT NOT NULL DEFAULT ''   -- license: CC0-1.0 | CC-BY-NC-4.0 | PUBLIC_DOMAIN | COPY_ALLOWED | UNKNOWN
  license_url, license_note                            TEXT NOT NULL DEFAULT ''
  line_count   INTEGER NOT NULL
  sections     JSONB NOT NULL DEFAULT '[]'             -- [{name, start_n, line_count}]
  UNIQUE (sutra_id, key)

sutra_lines
  translation_id BIGINT NOT NULL REFERENCES sutra_translations(id) ON DELETE CASCADE
  n              INTEGER NOT NULL
  text           TEXT NOT NULL
  section        TEXT                                  -- NULL when the line belongs to no section
  PRIMARY KEY (translation_id, n)

sutra_deities
  sutra_id  VARCHAR(128) NOT NULL REFERENCES sutras(id) ON DELETE CASCADE
  deity_id  VARCHAR(64)  NOT NULL REFERENCES deities(id)
  role      VARCHAR(16)  NOT NULL CHECK (IN 'SPEAKER','SUBJECT','MENTIONED')
  PRIMARY KEY (sutra_id, deity_id)
```

Indexes: `idx_sutras_order (sort_order, id)`; `idx_sutras_tradition_kind (tradition, kind, sort_order)`; GIN `idx_sutras_purposes` on `purposes`; `idx_sutra_deities_deity (deity_id, role)` for the reverse lookup. The primary key `(translation_id, n)` of `sutra_lines` is the index of the range query `n > $from AND n <= $to`. Search uses no index: `unaccent(lower(…)) LIKE '%term%'` scans the 153 rows (see AC 44).

Why:
- **Split fields**: `reference` and `license` once packed several things into one string (Taishō code + Chinese translator + year; licence code + URL + explanation), which a client cannot show separately. Each is a column now; `license` is a code from a closed set so the client maps labels, and the source's own sentence stays in `license_note`, so no information is lost. There is no `CHECK` on the licence code in the DB — the parser checks the set (AC 37), and a code can be added without a migration. `original_title` also holds the sutra a mantra is taken from, because the source does not tell the two cases apart.
- **`reference` is `TEXT`**: `lang-nghiem` has a 135-character `reference`; the seed failed with `value too long` on `VARCHAR(128)`.
- **One row per line** (`sutra_lines`, ~64,000 rows) instead of a `TEXT[]` per version: a page is read by primary key, not by unpacking a 6,967-element array.
- **Pages of lines by `n` range, not `OFFSET`**: `n` runs 1..N within a version (AC 21), so page `p` of size `s` is `n ∈ ((p−1)·s, p·s]` and no row is skipped.
- **`sections` precomputed JSONB** at seed time: the table of contents `[{name, start_n, line_count}]` comes with the detail without scanning `sutra_lines`; the client turns `start_n` into a page with `floor((start_n−1)/page_size)+1`.
- **`line_count` stored**: `total` and `total_pages` of a line page need no `count(*)`.
- **`sutra_deities` keyed `(sutra_id, deity_id)`**: no text links the same deity in two roles today; the parser enforces it.
- **`audio_url` is `''`, never `NULL`**: the API field is always a string, so a client tests for "none" without a null check (AC 20).
- **Duration backfill**: 4 lines a minute is the median of the hand-made estimates; it fills only rows still `NULL`, so re-applying changes nothing.

## Data file

Each `assets/data/sutras/<id>.json` carries, besides titles, tradition, kind, collection and versions: `dailyChant`, `purposes`, `durationMinutes`, per-version `key`, and the split fields (`reference`, `originalTitle`, `author`, `hanTranslator`, `hanTranslationYear`, `referenceNote`, `summary`, per-version `license`, `licenseUrl`, `licenseNote`). `index.json` lists the ids in library order.

- `key` is `"<lang>-<variant>"`, with a suffix `-2`, `-3` for a repeated pair (only `tam-kinh` repeats).
- `dailyChant`, `purposes` and `durationMinutes` were assigned by the author (Claude) with no religious reviewer: `daily_chant` is true for 34 texts (daily liturgy, repentance, the Ten Mantras, Great Compassion, Heart, Amitabha, Universal Gate, Eight Realisations, the Five Precepts, the loving-kindness and happiness sutras), `purposes` follow the group of the text, and the duration is a rounded editorial estimate (suggested as ceil(syllables ÷ 80), at least 1). `review_status` stays `UNREVIEWED`. See `docs/assumptions/sutra/` A-004, A-005, A-011 and `docs/tech-debt/sutra/` TD-002, TD-007.
- Known deviations of the data from the group rule (kept): the opening verse, Lễ Phật, Tán Phật and Tán lễ have `purposes = []`; Bát Đại Nhân Giác is `HOC_GIAO_LY`; four of the Ten Mantras carry an extra purpose by name; Kim Cang Thừa has two vow texts as `PHAT_NGUYEN` and `tara-21-praise` as `[]`.

## Request flow

> See `flow.md` for the Mermaid sequence diagrams of the five flows below (1:1 mirror). All four read routes make two or more sequential DB calls, so there is no separate ASCII version.

## Step-by-step Flow — Load the library with `cmd/seedsutras` (the service does not seed)

```
Step 1 — Count the texts that exist (usecase → repository)
- Trigger: cmd/seedsutras (`make ensure-sutras` or by hand) calls SutraUsecase.SeedIfEmpty(ctx, load) after connecting to the DB. cmd/main.go does not call it.
- Pre-condition: DB reachable; migration 0013 applied
- Call: SutraRepository.Count(ctx, SutraFilter{}) → SELECT count(*) FROM sutras
    - Success (n = 0) → Step 2
    - Success (n > 0) → stop: 0 texts inserted, load is NOT called, nothing is written; cmd/seedsutras logs "sutras already seeded, skipping" and exits 0
    - Failure (e.g. the table does not exist) → error wrapped with %w → the command logs it fatally and exits non-zero

Step 2 — Load and validate the seed data (repository)
- Trigger: n = 0
- Pre-condition: —
- Call: load() = repository.ParseSutraSeed(fs.Sub(assets.SutrasFS, "data/sutras")) (or os.DirFS(-dir)) — reads index.json and each <id>.json; checks every condition of AC 37 (missing file; empty/duplicate id or different from the file name; enum outside the allowed set; empty titles.vi; dailyChant or durationMinutes of the wrong type; missing/duplicate key; n with gaps; repeated deity_id; licence outside the closed set; hanTranslationYear not null or an integer ≥ 1) and, from v4.0, AC 38 (durationMinutes missing or null); assigns SortOrder = position in index.json; computes sections
    - Success → list of domain.SutraSeed, go to Step 3
    - Failure (any check) → error naming the id and the reason → fatal log, nothing written

Step 3 — Check that the deities exist (repository)
- Trigger: Step 2 done
- Pre-condition: the set of referenced deity_id
- Call: SutraRepository.MissingDeities → SELECT id FROM deities WHERE id = ANY($1), then the referenced ids not in the result
    - Success (none missing) → Step 4
    - Failure (missing deity_id) → error listing the missing ids and telling to run cmd/seeddeities → fatal log, nothing written
    - Failure (DB error) → error wrapped with %w → fatal log
  Decided (2026-10-03): a missing deity aborts; skipping the link would lose it for good because the seed does not run again once the table has rows. Deployment order: migration → cmd/seeddeities → cmd/seedsutras.

Step 4 — Insert every text (repository)
- Trigger: Step 3 done
- Pre-condition: valid data, all deities present
- Call: SutraRepository.InsertAll — in ONE transaction, for each text in index.json order:
    (a) INSERT INTO sutras … ON CONFLICT (id) DO NOTHING RETURNING id; no row returned (id already exists) → skip (b)–(d) for that text
    (b) INSERT INTO sutra_translations … RETURNING id for each version
    (c) INSERT INTO sutra_lines as multi-row batches for each version
    (d) INSERT INTO sutra_deities for each link
    - Success → the counts actually inserted (153 on the first run), go to Step 5
    - Failure (DB error on any statement, including a foreign-key violation) → whole rollback → error → fatal log
    - Transaction boundary: every statement of Step 4 shares ONE transaction — all or nothing

Step 5 — Log (cmd/seedsutras)
- Trigger: SeedIfEmpty returned the number of texts inserted
- Pre-condition: —
- Call: if the number > 0 → log "seeded N sutras, …" with the counts of texts, versions, lines and deity links
    - Success → exit 0
    - Failure → no failure branch
```

## Step-by-step Flow — List, filter and search (`GET /v1/sutras`)

```
Step 1 — Read page and filters (handler)
- Trigger: a request arrives on the public route
- Pre-condition: —
- Call: r.URL.Query() reads page (empty → 1, otherwise strconv.Atoi), tradition, kind, collection, purpose, daily, max_minutes, lang, q; a parameter that is present but empty still counts as present
    - Success → Step 2
    - Failure (page is not an integer, including int overflow) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Validate the page range and the filter values (usecase)
- Trigger: the handler calls SutraUsecase.List(ctx, filter, page)
- Pre-condition: page is an integer
- Call: page < 1 or page > math.MaxInt32; tradition/kind/purpose in the allowed set; daily ∈ {true,false}; max_minutes an integer ≥ 1 (and ≤ MaxInt32); collection and lang not empty; q trimmed and then 1–100 runes
    - Success → Step 3 (offset = (page-1) × page_size, computed in int64)
    - Failure (page) → domain.ErrInvalidPage → 400 INVALID_PAGE
    - Failure (filter) → InvalidSutraFilterError{Field} → 400 INVALID_FILTER, details.field (including "q")

Step 3 — Count the matching texts (repository)
- Trigger: Step 2 done
- Pre-condition: valid filter
- Call: SutraRepository.Count → SELECT count(*) FROM sutras s WHERE <AND conditions>; max_minutes → duration_minutes <= $n (NULL drops out); purpose → purposes @> ARRAY[$p]; lang → EXISTS (SELECT 1 FROM sutra_translations …); q → (unaccent(lower(collection || ' ' || reference)) LIKE unaccent(lower($t)) ESCAPE '\' OR EXISTS (SELECT 1 FROM jsonb_each_text(titles) …)) with $t = %term% and \, % and _ escaped
    - Success → total, go to Step 4
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 4 — Fetch the page of texts (repository)
- Trigger: Step 3 done
- Pre-condition: total known
- Call: SutraRepository.List → SELECT <sutras columns> FROM sutras s WHERE <AND conditions> ORDER BY sort_order ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (empty if the offset is past the end), go to Step 5
    - Failure (DB / scan error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 5 — Load the versions (repository)
- Trigger: Step 4 returned at least one text (none → skip Steps 5–6, go to Step 7)
- Pre-condition: the ids of the page
- Call: SELECT key, lang, variant, translator, license, line_count FROM sutra_translations WHERE sutra_id = ANY($1) ORDER BY sutra_id, ordinal
    - Success → attach to each text, go to Step 6
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 6 — Load the deity links (repository)
- Trigger: Step 5 done
- Pre-condition: the ids of the page
- Call: SELECT sd.sutra_id, sd.deity_id, sd.role, d.name_vi, d.thumbnail_url FROM sutra_deities sd JOIN deities d ON d.id = sd.deity_id WHERE sd.sutra_id = ANY($1) ORDER BY sd.sutra_id, role_rank, sd.deity_id
    - Success → attach to each text, go to Step 7
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — Steps 3 to 6 are independent statements, not one snapshot

Step 7 — Build the page and respond (usecase + handler)
- Trigger: the repository returned
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; the handler maps to the card DTO (nil arrays → []; duration_minutes nil → null until v4.0; audio_url stays "")
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → no new failure branch
```

## Step-by-step Flow — Sutra detail (`GET /v1/sutras/{id}`)

```
Step 1 — Find the text (repository)
- Trigger: the handler reads {id} and calls SutraUsecase.Get(ctx, id)
- Pre-condition: —
- Call: SutraRepository.FindByID → SELECT <sutras columns, incl. summary and audio_url> + original_title, author, han_translator, han_translation_year, reference_note FROM sutras WHERE id = $1 (the last five columns are read only here; the list does not read them)
    - Success → Step 2
    - Failure (sql.ErrNoRows) → domain.ErrSutraNotFound → 404 SUTRA_NOT_FOUND, stop
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 2 — Load the versions with sources and sections (repository)
- Trigger: Step 1 done
- Pre-condition: the text exists
- Call: SELECT key, lang, variant, translator, source_name, source_url, license, license_url, license_note, note, line_count, sections FROM sutra_translations WHERE sutra_id = $1 ORDER BY ordinal
    - Success → attach to the text, go to Step 3
    - Failure (DB error / sections decode) → error wrapped with %w → 500 INTERNAL_ERROR

Step 3 — Load the deity links (repository)
- Trigger: Step 2 done
- Pre-condition: —
- Call: the same JOIN on `deities` as Step 6 of the list, for one sutra_id
    - Success → attach to the text, go to Step 4
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — three independent statements

Step 4 — Build the response (handler)
- Trigger: the usecase returned
- Pre-condition: the text with versions and links
- Call: map to sutraDetailResponse (the split fields; license_url/license_note per version; han_translation_year nil → null); sections nil → []
    - Success → 200 SutraDetail
    - Failure → no new failure branch
```

## Step-by-step Flow — Read a version page by page (`GET /v1/sutras/{id}/lines`)

```
Step 1 — Read and check the parameters (handler)
- Trigger: a request arrives on the public route
- Pre-condition: —
- Call: reads translation (required), page (empty → 1), page_size (empty/absent → the config default); strconv.Atoi for page and page_size
    - Success → Step 2
    - Failure (translation absent or empty) → ErrInvalidTranslation → 400 INVALID_TRANSLATION, details.field = "translation"
    - Failure (page is not an integer, or overflows) → ErrInvalidPage → 400 INVALID_PAGE
    - Failure (page_size is not an integer) → ErrInvalidPageSize → 400 INVALID_PAGE_SIZE

Step 2 — Validate the range and clamp page_size (usecase)
- Trigger: the handler calls SutraUsecase.Lines(ctx, id, key, page, pageSize)
- Pre-condition: page and page_size are integers
- Call: page < 1 or > math.MaxInt32 → error; page_size < 1 → error; page_size > sutra.lines_max_page_size → clamped to it; absent → sutra.lines_page_size
    - Success → Step 3 (fromN = (page-1) × page_size, toN = page × page_size, computed in int64)
    - Failure → ErrInvalidPage / ErrInvalidPageSize → 400, stop

Step 3 — Check that the text and the version exist (repository)
- Trigger: Step 2 done
- Pre-condition: valid parameters
- Call: SutraRepository.FindTranslation → (a) SELECT 1 FROM sutras WHERE id = $1; (b) SELECT id, line_count FROM sutra_translations WHERE sutra_id = $1 AND key = $2
    - Success → translation_id and line_count, go to Step 4
    - Failure ((a) no row) → ErrSutraNotFound → 404 SUTRA_NOT_FOUND, stop
    - Failure ((b) no row) → ErrSutraTranslationNotFound → 404 TRANSLATION_NOT_FOUND, stop
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 4 — Fetch the page of lines (repository)
- Trigger: Step 3 done
- Pre-condition: translation_id known
- Call: SutraRepository.ListLines → SELECT n, text, section FROM sutra_lines WHERE translation_id = $1 AND n > $2 AND n <= $3 ORDER BY n ASC
    - Success → the lines (empty if fromN ≥ line_count), go to Step 5
    - Failure (DB / scan error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — Steps 3 and 4 are independent statements, not one snapshot

Step 5 — Build the page and respond (usecase + handler)
- Trigger: the repository returned
- Pre-condition: items, line_count
- Call: items nil → []; total = line_count; total_pages = (total + page_size - 1) / page_size; empty section → null
    - Success → 200 {translation, items, page, page_size, total, total_pages}
    - Failure → no new failure branch
```

## Step-by-step Flow — Sutras of a Buddha or bodhisattva (`GET /v1/deities/{id}/sutras`)

```
Step 1 — Read page and role filter (handler)
- Trigger: a request arrives on the public route
- Pre-condition: —
- Call: reads {id}, page (empty → 1, otherwise strconv.Atoi) and role (optional; present but empty still counts as present)
    - Success → Step 2
    - Failure (page is not an integer, or overflows) → ErrInvalidPage → 400 INVALID_PAGE

Step 2 — Validate page and role (usecase)
- Trigger: the handler calls SutraUsecase.ListByDeity(ctx, deityID, role, page)
- Pre-condition: page is an integer
- Call: page ∈ [1, MaxInt32]; role absent or ∈ {SPEAKER, SUBJECT, MENTIONED}
    - Success → Step 3
    - Failure (page) → ErrInvalidPage → 400 INVALID_PAGE
    - Failure (role) → InvalidSutraFilterError{Field: "role"} → 400 INVALID_FILTER

Step 3 — Check that the deity exists (repository of the deity feature)
- Trigger: Step 2 done
- Pre-condition: —
- Call: DeityRepository.FindByID(ctx, deityID) — exists in the `deity` feature, not changed
    - Success → Step 4
    - Failure (none) → ErrDeityNotFound → 404 DEITY_NOT_FOUND, stop
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 4 — Count the linked texts (repository)
- Trigger: Step 3 done
- Pre-condition: the deity exists
- Call: SutraRepository.CountByDeity → SELECT count(*) FROM sutra_deities WHERE deity_id = $1 [AND role = $2]
    - Success → total, go to Step 5
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 5 — Fetch the page of linked texts (repository)
- Trigger: Step 4 done
- Pre-condition: total known
- Call: SutraRepository.ListByDeity → SELECT <sutras columns>, sd.role FROM sutras s JOIN sutra_deities sd ON sd.sutra_id = s.id WHERE sd.deity_id = $1 [AND sd.role = $2] ORDER BY role_rank, s.sort_order, s.id OFFSET $3 LIMIT $4
    - Success → items (may be empty), go to Step 6
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 6 — Load the versions (repository)
- Trigger: Step 5 returned at least one text (none → skip Steps 6–7, go to Step 8)
- Pre-condition: the ids of the page
- Call: the same statement as Step 5 of the list
    - Success → Step 7
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 7 — Load the deity links of those texts (repository)
- Trigger: Step 6 done
- Pre-condition: the ids of the page
- Call: the same JOIN as Step 6 of the list — returns every deity of each text, not only the one asked about
    - Success → Step 8
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — Steps 4 to 7 are independent, not one snapshot

Step 8 — Build the page and respond (usecase + handler)
- Trigger: the usecase returned
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; map to sutraByDeityItemResponse = card + role
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → no new failure branch
```

## Deliberately stubbed for now

Nothing is stubbed: every method of `SutraRepository` has real SQL and has run on Postgres. What is deliberately not done sits in "Out of scope" of `requirements.md` (reseed command, delta sync, audio files, full-text search, bookmarks).

## Known gaps carried by this feature

- **Labels and Great Compassion Mantra version are unreviewed**: `daily_chant`, `purposes` and the summaries were assigned by the author, the Great Compassion Mantra uses the 84-verse phatgiao.org.vn version, and the deity links were assigned by the person who collected the data; no knowledgeable reviewer exists. `review_status = UNREVIEWED` is kept so it can change when one does (product Appendix A item 1; TD-002, A-004, A-006, A-011).
- **Licences are unresolved**: most Vietnamese versions are `UNKNOWN`; Kim Cang Thừa versions are CC BY-NC 4.0. The API returns every text; the code is descriptive. Needs a decision before any paid or commercial use (product Appendix A item 2; TD-003).
- **Duration estimates are unreviewed and uneven**: Dược Sư comes out at 80 minutes because the translation version is longer than the chant; the long collections take 14–15 hours; the 10 Vajrayāna texts carry a formula value (lines ÷ 4). Accepted (TD-007, A-005, A-009).
- **Content cannot be corrected after the first load** except by hand-written SQL; a changed JSON does not reach a database that already has rows (product Appendix A item 15; TD-004).
- **Dropping every row reloads the library**: "never bring back a deleted text" only holds while the table keeps at least one row.
- **Two concurrent runs**: both may see an empty table and parse the seed; the second transaction waits, then every insert hits `ON CONFLICT DO NOTHING`. Safe, but wasteful.
- **Statements do not share a snapshot**: the list, detail, line page and by-deity routes use several independent statements; content changes only by hand SQL, so the risk is low.
- **The client chooses `page_size` of line pages**: changing it between two calls shifts the pages; a client keeps one `page_size` for a whole reading, or recomputes the page from `n`.
- **Search is an unindexed scan** (`unaccent` over titles, collection and reference); fine at 153 texts, not timed yet (AC 44) and not designed for a library several times larger (NFR-STR-03).
- **The seed files are not in git** (TD-001): a fresh checkout cannot run `cmd/seedsutras` against the real library or the tests that read it.
- **Binary size and first-load time**: the embedded data adds 10.25 MB (70.91 MB vs 60.66 MB); loading ~64,000 lines took 1.2–1.8 s on a local Postgres, not measured on a remote DB. `seedTimeout` is a constant in `cmd/seedsutras` (2 minutes), not in `config/` (TD-008, TD-010).
- **The deity foreign key couples the seeds**: `TRUNCATE … deities CASCADE` also empties `sutra_deities`, and because `sutras` keeps rows the seed does not restore them; both must be reseeded.
- **Tests that use `connectTestDB`** run `TRUNCATE` on the dev database, like the other features (TD-009).
