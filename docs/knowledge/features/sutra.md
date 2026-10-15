---
type: Feature
title: Sutra
description: Read-only Buddhist sutra / verse / mantra library — a filtered paged list, a detail page, a line-paged reader and a by-deity lookup, all public, backed by four tables seeded once from JSON embedded in the binary.
resource: services/appService/
tags: [content, sutra, catalogue, seed, public, deity]
status: draft
generated: { by: claude-code, at: 2026-10-03T17:26:46+07:00 }
sources:
  - path: docs/specs/features/sutra/requirements.md
    kind: spec
  - path: docs/specs/features/sutra/design.md
    kind: spec
  - path: docs/specs/features/sutra/flow.md
    kind: spec
  - path: docs/specs/features/sutra/tasks.md
    kind: spec
  - path: migrations/0013_sutras.sql
    kind: migration
  - path: services/appService/delivery/http/sutra_handler.go
    kind: code
  - path: services/appService/usecase/sutra_usecase.go
    kind: code
  - path: services/appService/repository/postgres_sutra_repository.go
    kind: code
  - path: services/appService/repository/sutra_seed.go
    kind: code
  - path: assets/embed.go
    kind: code
  - path: services/appService/cmd/seedsutras/main.go
    kind: code
  - path: services/appService/e2e/sutra_e2e_test.go
    kind: test
---

# Summary

Four public `GET` routes serve the sutra library (kinh / kệ / chú) of the `content` module. No route
takes an `Authorization` header. Every failure uses the shared `utils.ApiResponse` envelope.

- `GET /v1/sutras?page=N` — 20 per page, ordered `sort_order, id`, with optional filters `tradition`,
  `kind`, `collection`, `purpose`, `daily` (`true|false`), `max_minutes`, `lang`, combined with AND.
- `GET /v1/sutras/{id}` — the summary plus `review_status`, and per translation `source_name`,
  `source_url`, `note` and `sections` (`[{name, start_n, line_count}]`).
- `GET /v1/sutras/{id}/lines?translation=<key>&page=N&page_size=M` — the text of one translation, one
  display line per item (`n`, `text`, `section`), `page_size` default 200, capped at 500. Lines are
  numbered 1..`line_count` without gaps, so page `p` is exactly `(p-1)*size < n <= p*size`.
- `GET /v1/deities/{id}/sutras?page=N&role=` — the sutras linked to a Buddha or Bồ Tát, with the
  deity's `role` (`SPEAKER`, `SUBJECT`, `MENTIONED`) in each item.

The shipped library is 153 texts, 391 translations, 64,009 lines and 151 deity links, from
`assets/data/sutras/*.json` plus `index.json`. Each sutra has a `tradition` (Nam Tông, Bắc Tông, Kim
Cang Thừa), a `kind`, a `collection`, `titles` (always including `vi`), `daily_chant`, `purposes`, an
editorial estimate `duration_minutes` (`null`, never `0`, when unknown; `null` for the 10 Kim Cang Thừa texts that have no Vietnamese version) and linked deities.

On startup the service counts the `sutras` rows and, only when the table is empty, validates the whole
embedded seed, checks that every deity it links to exists, and inserts everything in one transaction
(`ON CONFLICT (id) DO NOTHING`). A start with data present neither parses nor writes. If a deity is
missing the service refuses to start with `run cmd/seeddeities first`; the same happens if the seed is
invalid or the insert fails (all rolled back). `go run ./services/appService/cmd/seedsutras [-dir]` does
the same by hand and is also a no-op when the table has rows. `make run` seeds deities when empty, and
`make seed` runs deities then sutras.

Measured on 2026-10-03 on a local Postgres: first start 1.82 s, later starts 0.13 s, binary +10.25 MB;
a request takes 1.8–2.5 ms on loopback.

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. The only transaction is the seed insert (`InsertAll`); reads are independent
statements (count, then page) with no shared snapshot.

| Layer | Type | File |
|---|---|---|
| Domain / Entity | Domain Entity | `services/appService/domain/sutra.go` — `Sutra`, `SutraTranslation`, `SutraLine`, `SutraSection`, `DeitySutra`, page types, `SutraSeed`/`SutraSeedResult`, enums `Tradition`, `Kind`, `Purpose`, `DeityRole` |
| Domain / Errors | sentinel errors | `domain/sutra.go` — `ErrSutraNotFound`, `ErrSutraTranslationNotFound`, `ErrInvalidTranslation`, `ErrInvalidPageSize`, `ErrSutraSeedDeitiesMissing`, `*InvalidSutraFilterError`; `ErrInvalidPage` and `ErrDeityNotFound` are shared |
| Domain / Port | interface | `domain/sutra.go` — `SutraRepository` (`Count`, `List`, `FindByID`, `FindTranslation`, `ListLines`, `CountByDeity`, `ListByDeity`, `MissingDeities`, `InsertAll`); ordering is part of its contract. Mock: `domain/mock_sutra_repository.go` |
| Usecase | UseCase | `services/appService/usecase/sutra_usecase.go` — `List`, `Get`, `Lines`, `ListByDeity`, `SeedIfEmpty(load)` and filter validation |
| Repository | Repository | `services/appService/repository/postgres_sutra_repository.go` — SQL on the four tables; `InsertAll` = one transaction |
| Repository / Seed parser | function | `services/appService/repository/sutra_seed.go` — `ParseSutraSeed(fs.FS)`: validates everything before any write, derives `sections` (`sectionsOf`), `SortOrder` = position in `index.json` |
| Seed command | cmd | `services/appService/cmd/seedsutras/main.go` — same rules, `-dir`, dev config and secrets (`⚠️ DEV ONLY`) |
| Embedded data | `go:embed` | `assets/embed.go` (`SutrasFS`) embedding `assets/data/sutras/*.json` |
| Delivery / Handler | Handler | `services/appService/delivery/http/sutra_handler.go` — `List`, `Get`, `Lines`, `ListByDeity` and their DTOs; `delivery/http/errors.go` maps the sentinel errors |
| Config | policy | `services/appService/config/config.go` `SutraConfig` + `Validate()`, and the `sutra:` block in `config.dev.yaml` (`page_size: 20`, `lines_page_size: 200`, `lines_max_page_size: 500`) |
| Wiring | composition root | `services/appService/cmd/main.go` — `SutraConfig.Validate()` (fatal on error), repo → usecase → handler, `SeedIfEmpty` right after the DB connects (`sutraSeedTimeout`, fatal on failure), 4 routes. Touches no `⚠️ DEV ONLY` block |
| Migration | schema | `migrations/0013_sutras.sql` — `sutras`, `sutra_translations`, `sutra_lines` (PK `(translation_id, n)`), `sutra_deities` (FK to `deities`, no cascade) |

`pkg/` dependencies: `pkg/database`, `pkg/utils` (`WriteSuccessResponse`, `WriteBusinessError`),
`github.com/lib/pq`.

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/v1/sutras` | public | `SutraHandler.List` |
| `GET` | `/v1/sutras/{id}` | public | `SutraHandler.Get` |
| `GET` | `/v1/sutras/{id}/lines` | public | `SutraHandler.Lines` |
| `GET` | `/v1/deities/{id}/sutras` | public | `SutraHandler.ListByDeity` |

Errors: `400 INVALID_FILTER` (with `details.field`), `400 INVALID_PAGE`, `400 INVALID_PAGE_SIZE`,
`400 INVALID_TRANSLATION`, `404 SUTRA_NOT_FOUND`, `404 TRANSLATION_NOT_FOUND`, `404 DEITY_NOT_FOUND`.
A non-`GET` method is `405` with `Allow: GET, HEAD`. A page past the end is a valid, empty page. The sutra
is checked before the translation key. Arrays are `[]`, never `null`; `section` on a line is `null` when
it belongs to none. Adding the by-deity route does not change the other deity routes (checked on the real
binary). No `api/` endpoint concepts exist yet.

# Request flow

```
GET /v1/sutras/{id}/lines?translation=K&page=P&page_size=S
  └─ public
       └─ SutraHandler.Lines                              (delivery/http)
            ├─ translation empty → 400 INVALID_TRANSLATION
            ├─ page non-integer → 400 INVALID_PAGE; page_size non-integer → 400 INVALID_PAGE_SIZE
            └─ SutraUsecase.Lines                          (usecase)
                 ├─ page < 1 or > MaxInt32 → ErrInvalidPage; page_size < 1 → ErrInvalidPageSize
                 ├─ size = min(page_size, lines_max_page_size)
                 ├─ SutraRepository.FindTranslation        → 404 SUTRA_NOT_FOUND / TRANSLATION_NOT_FOUND
                 └─ SutraRepository.ListLines(translationID, (P-1)*S, P*S)   n > from AND n <= to
```

First-run seed, at startup in `cmd/main.go`:

```
SutraUsecase.SeedIfEmpty(ctx, load)
  ├─ SutraRepository.Count                       → n > 0: stop, nothing parsed or written
  ├─ load() = ParseSutraSeed(fs.Sub(SutrasFS, "data/sutras"))     validates everything first
  ├─ SutraRepository.MissingDeities              → any missing: ErrSutraSeedDeitiesMissing, abort
  └─ SutraRepository.InsertAll                   → one transaction, ON CONFLICT (id) DO NOTHING
```

# Validation rules

- Filters — `tradition`, `kind`, `purpose` must be a known enum value; `daily` is `true` or `false`;
  `max_minutes` an integer from 1 to MaxInt32; `collection` and `lang` non-empty. A parameter that is
  present but empty is invalid (`INVALID_FILTER`); an absent one is no filter.
- `page` — empty means 1; a non-integer, `< 1` or `> MaxInt32` is `INVALID_PAGE`. `role` on the by-deity
  route must be a valid role (`INVALID_FILTER`).
- Seed — `ParseSutraSeed` rejects (27 invalid-sutra and 7 invalid-index cases are tested) malformed or
  unknown JSON fields, an empty or duplicate id, a missing `titles.vi`, a duplicate translation key, lines
  with a gap or repeat, an unknown enum, and an `index.json` that disagrees with the files. Any rejection
  aborts startup.
- Config — `SutraConfig.Validate()` rejects a zero size and a default above the cap; the binary exits
  before touching the database.

# Pending work

- **Content is unreviewed.** All 153 texts carry `review_status: UNREVIEWED`; `daily_chant`, `purposes`,
  deity links and Hán/Sanskrit titles were assigned by the collector, not by a reviewer, and source typos
  are kept verbatim. Tracked in `docs/tech-debt/sutra/TD-002-unreviewed-religious-content.md`.
- **The data is not in git.** `assets/` (including `embed.go`) is untracked and both `openapi.yaml` copies
  are hidden by `.gitignore`; a fresh clone does not build. See TD-001.
- Licences are unresolved (`unknown`, non-commercial for the Kim Cang Thừa texts) — TD-003.
- No way to change content after the first seed except SQL by hand; no per-sutra version or `updated_at`;
  no admin CRUD — TD-004. No audio, ETag or delta sync (dropped with the old `content` spec) — TD-005.
- `duration_minutes` is kept as generated (syllables ÷ 80, rounded) — the owner decided on 2026-10-03 not to hand-edit it, so the Phase 0 review task is closed. Known skew stays: Dược Sư reads 80 minutes against a realistic 45–60, and Pháp Hoa / Lăng Nghiêm / Kim Quang Minh are 14–15 hours — TD-007, A-005.
- `seedsutras` uses dev config and plaintext dev secrets (`⚠️ DEV ONLY`).
- `delivery/http/`, `repository/` and `cmd/` have no `README.md` (repo-wide gap).
- Promote to `stable` once the content has been reviewed and `assets/` and the OpenAPI file are under
  version control; the E2E test already exercises the routes and the seed on the real binary.

# Known gaps vs. design

None between the code and `docs/specs/features/sutra/design.md` (v2.10). Limits of what the tests reach,
recorded in `docs/specs/features/sutra/test-coverage-report.md`:

- `500` without leaking the cause is proven at handler level only.
- The main binary aborting on invalid embedded data is covered through the parser and `seedsutras -dir`.
- Two instances starting at the same moment are not tested; emptying the `sutras` table and restarting
  reseeds the whole library.
- The repository tests `TRUNCATE` the dev database's tables and, like the E2E test, need Postgres/Docker
  (skipped under `-short`).

# Related concepts

- [Deity](deity.md) — every sutra links to deities; seeding stops if they are missing, and
  `GET /v1/deities/{id}/sutras` lives beside the deity routes.
- [Article](article.md) — the same first-run seed from embedded JSON, and the same `ErrInvalidPage`.
- [Temple](temple.md) — the other public catalogue.
- Login (`docs/specs/features/login/`) and Register (`docs/specs/features/onboarding/`) — the identity features; sutra involves no auth.
- No `Migration` concept exists yet for `migrations/0013_sutras.sql`.
