---
type: Feature
title: Temple
description: Read-only catalogue of Vietnamese temples (chùa, thiền viện, đạo tràng, tịnh xá, thiền xá) — a top list, a paged list and a detail page, all public, loaded into Postgres by a separate seed command.
resource: services/appService/
tags: [content, temple, catalogue, seed, public]
status: draft
generated: { by: claude-code, at: 2026-09-30T17:00:00+07:00 }
sources:
  - path: docs/specs/features/temple/requirements.md
    kind: spec
  - path: docs/specs/features/temple/design.md
    kind: spec
  - path: docs/specs/features/temple/api-contract.md
    kind: spec
  - path: docs/specs/features/temple/flow.md
    kind: spec
  - path: migrations/0005_temples.sql
    kind: migration
  - path: migrations/0011_temple_content.sql
    kind: migration
  - path: services/appService/delivery/http/temple_handler.go
    kind: code
  - path: services/appService/usecase/temple_usecase.go
    kind: code
  - path: services/appService/repository/postgres_temple_repository.go
    kind: code
  - path: services/appService/cmd/seedtemples/main.go
    kind: code
---

# Summary

Three public `GET` routes serve the temple browse screens in the `content` module.
`GET /v1/temples/top` returns the first 10 temples as summary cards, `GET /v1/temples?page=N`
returns 20 per page with `page`, `page_size`, `total` and `total_pages`, and
`GET /v1/temples/{id}` returns one temple with its facilities and editorial text. No route takes an
`Authorization` header. Failures use the shared `utils.ApiResponse` envelope: a non-integer or
`< 1` page is `400 INVALID_PAGE`, an unknown id is `404 TEMPLE_NOT_FOUND`. A page past the end is a
valid empty page (`items: []`, real `total`).

Ranking is `heritage_priority ASC, established_year ASC, id ASC`: the heritage rank decides first
(world heritage → special national monument → … → unranked, 8 levels), then older temples, then id
as a stable tiebreaker. `thumbnail_url` and `image_url` are JSON `null` when empty. In the current
data set there are 179 temples (60 north, 59 central, 60 south); all 179 have an introduction,
architecture/heritage text and visiting notes, and none has an image.

Unlike articles, the service does **not** seed this table: rows arrive only when someone runs
`go run ./services/appService/cmd/seedtemples`, which upserts `assets/data/temple.json` by id in one
transaction.

**Current limitation**: no temple has an image (`image_url`/`thumbnail_url` are `null` for all 179),
the seed command uses dev config and dev secrets, and no black-box E2E test covers the routes.

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. Reads are single statements except `List` (count, then page, no shared
snapshot). The only transaction is the seed upsert, owned by one repository method.

| Layer | Type | File |
|---|---|---|
| Domain / Entity | Domain Entity | `services/appService/domain/temple.go` — `Temple`, `TempleAmenities`, `TemplePage`; `HeritagePriority(rank)` is the single rank→weight map |
| Domain / Errors | sentinel errors | `domain/temple.go` — `ErrTempleNotFound`, `ErrInvalidPage` (shared with deity and article) |
| Domain / Port | interface | `domain/temple.go` — `TempleRepository` (`ListTop`, `List`, `FindByID`, `Upsert`); ordering is part of its contract |
| Usecase | UseCase | `services/appService/usecase/temple_usecase.go` — `Top`, `List` (offset, `total_pages`, rejects `page < 1` or `> MaxInt32`), `Get` |
| Repository | Repository | `services/appService/repository/postgres_temple_repository.go` — SQL on `temples`, shared `templeColumns`/`templeOrder`; `Upsert` = one transaction, `ON CONFLICT (id) DO UPDATE`; `pq.Array` for `aliases`/`visiting_notes` |
| Delivery / Handler | Handler | `services/appService/delivery/http/temple_handler.go` — `Top`, `List`, `Get`, summary/detail DTOs; `delivery/http/errors.go` maps `ErrTempleNotFound` → `404 TEMPLE_NOT_FOUND` and `ErrInvalidPage` → `400 INVALID_PAGE` |
| Seed command | cmd | `services/appService/cmd/seedtemples/main.go` + `parse.go` — `parseTemples` strips `/* */` comments, derives `region` from the id (`temple_<north\|central\|south>_<n>`), validates unique ids and a known heritage rank, sets `heritage_priority` |
| Config | policy | `services/appService/config/config.go` `TempleConfig` and the `temple:` block in `config.dev.yaml` (`top_limit: 10`, `page_size: 20`) |
| Wiring | composition root | `services/appService/cmd/main.go` — repo → usecase → handler, 3 public routes. Seeds nothing at startup and touches no `⚠️ DEV ONLY` block |
| Migration | schema | `migrations/0005_temples.sql` — table, CHECK constraints on `type`/`region`/`heritage_rank`, `idx_temples_ranking`; `migrations/0011_temple_content.sql` — `introduction`, `architecture_heritage`, `visiting_notes` |

`pkg/` dependencies: `pkg/database`, `pkg/utils` (`WriteSuccessResponse`, `WriteBusinessError`),
`github.com/lib/pq`. The seed command also uses `pkg/config` and the dev secret resolver
`appConfig.NewMockSecretGetterFromYAML`.

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/v1/temples/top` | public | `TempleHandler.Top` |
| `GET` | `/v1/temples` | public | `TempleHandler.List` |
| `GET` | `/v1/temples/{id}` | public | `TempleHandler.Get` |

`/v1/temples/top` is a literal segment, so `ServeMux` picks it over `{id}`. The three routes are in
`docs/specs/api/openapi.yaml`. No `api/` endpoint concepts exist yet.

Summary fields: `id`, `name`, `type`, `region`, `thumbnail_url` and `image_url` (`string | null`),
`sect`, `heritage_rank`, `established_year`, `location{lat,lng,address}`. Detail adds `aliases`,
`founder`, `current_abbot`, `operating_hours`, `amenities{parking_car,parking_motorcycle,
vegetarian_meals,summer_retreat}`, `introduction`, `architecture_heritage`, `visiting_notes`.
`aliases` and `visiting_notes` are `[]`, never `null`.

# Request flow

```
GET /v1/temples?page=N
  └─ public
       └─ TempleHandler.List                           (delivery/http)
            ├─ page empty → 1; strconv.Atoi failure → 400 INVALID_PAGE
            └─ TempleUsecase.List(ctx, page)            (usecase)
                 ├─ page < 1 or > MaxInt32 → ErrInvalidPage
                 └─ TempleRepository.List(offset, size)  (repository)
                      ├─ SELECT count(*) FROM temples
                      └─ SELECT … ORDER BY heritage_priority, established_year, id OFFSET $1 LIMIT $2
```

Seeding, run by hand:

```
go run ./services/appService/cmd/seedtemples [-file assets/data/temple.json]
  ├─ parseTemples(file)                → validation failure aborts before any write
  ├─ load config.dev.yaml + dev secrets  (⚠️ DEV ONLY)
  └─ TempleRepository.Upsert           → one transaction, all rows or none
```

# Validation rules

- `page` — parsed in the handler with `strconv.Atoi`; empty means 1. A non-integer is
  `ErrInvalidPage`. The usecase adds `page < 1 || page > math.MaxInt32`, also `ErrInvalidPage` →
  `400 INVALID_PAGE` with `details.field = "page"`.
- `{id}` — no format rule; an id that matches no row is `ErrTempleNotFound`.
- Seed — `parseTemples` rejects malformed JSON, an empty list, a duplicate id, an id not shaped
  `temple_<region>_<n>` with a known region, and an unknown `heritage_rank`. The database adds
  CHECKs on `type` (5 values), `region` (3) and `heritage_rank` (8).

# Pending work

- No E2E test in `services/appService/e2e/` for the three routes or for the seed command; the
  repository tests need Postgres and are skipped under `-short`.
- Images: `image_url` and `thumbnail_url` are empty for all 179 temples; no source, storage or
  licence is decided.
- `cmd/seedtemples` reads config from `config.dev.yaml` and dev plaintext secrets (`⚠️ DEV ONLY`),
  and reads the JSON from disk rather than embedding it in the binary; who seeds staging/production,
  and how, is not decided.
- The `temples` table stores no source for the editorial text (`introduction`,
  `architecture_heritage`, `visiting_notes`), which was compiled from public pages; the conditions
  for showing it are not confirmed.
- `established_year` comes from the seed file and can disagree with the year written in the
  introduction text; nobody is assigned to review it.
- No backoffice CRUD for temples; `docs/specs/features/admin/` mentions it but there is no temple
  spec for it.
- Promote to `stable` once the routes are exercised by an automated end-to-end test against the
  real binary and a seeded Postgres.

# Known gaps vs. design

None between the code and `docs/specs/features/temple/design.md` that this author found; the
spec was written from the code and keeps its own `⚠️ Not confirmed` list (user-story ids, the
"featured" criterion, image source, data accuracy).

- `total` and `items` of the list come from two statements without a shared snapshot.
- Re-running the seed overwrites every column of existing rows, so a hand edit in the database is lost.

# Related concepts

- [Deity](deity.md) and [Article](article.md) — sibling catalogues with the same top / paged list /
  detail shape, the same `ErrInvalidPage` and the same public-route wiring.
- [Login](login.md) and [Register](register.md) — identity features; temples share only
  `delivery/http/errors.go`'s `writeDomainError` and the middleware chain from
  [packages/middleware.md](../packages/middleware.md).
- No `Migration` concepts exist yet for `0005_temples.sql` or `0011_temple_content.sql`.
