---
type: Feature
title: Deity
description: Read-only catalogue of Phật (Buddhas) and Bồ tát (bodhisattvas) — a mixed top list, a paged list of both types and a detail page, all public, loaded into Postgres by a separate seed command.
resource: services/appService/
tags: [content, deity, buddha, bodhisattva, catalogue, seed, public]
status: draft
generated: { by: claude-code, at: 2026-09-30T18:30:00+07:00 }
sources:
  - path: docs/specs/features/deity/requirements.md
    kind: spec
  - path: docs/specs/features/deity/design.md
    kind: spec
  - path: docs/specs/features/deity/api-contract.md
    kind: spec
  - path: docs/specs/features/deity/flow.md
    kind: spec
  - path: migrations/0007_deities.sql
    kind: migration
  - path: services/appService/delivery/http/deity_handler.go
    kind: code
  - path: services/appService/usecase/deity_usecase.go
    kind: code
  - path: services/appService/repository/postgres_deity_repository.go
    kind: code
  - path: services/appService/cmd/seeddeities/main.go
    kind: code
---

# Summary

Three public `GET` routes serve the Phật / Bồ tát screens in the `content` module.
`GET /v1/deities/top` returns the first 10 entries of either type as summary cards,
`GET /v1/deities?page=N` returns 20 deities per page — Buddhas and bodhisattvas in one list, each
item carrying its `type` — with `page`, `page_size`, `total` and `total_pages`, and `GET /v1/deities/{id}` returns one Buddha or bodhisattva with its lunar
memorial days, iconography, vows and mantras. No route takes an `Authorization` header. A bad
`page` is `400 INVALID_PAGE`, an unknown id is `404 DEITY_NOT_FOUND`.

Order is `sort_order ASC, id ASC`, where `sort_order` is the position in
`assets/data/phat_botat.json`. The source has no popularity metric, so the curated file order is
the ranking: the 6 Buddhas first, then the bodhisattvas by editorial prominence. With
`deity.top_limit: 10` the top list is 6 Buddhas plus the first 4 bodhisattvas. The data set has 18
entries (6 `BUDDHA`, 12 `BODHISATTVA`), and the dev database matches.

The paged list has no `type` filter: it is the whole `deities` table in `sort_order`, so with the
current data page 1 holds all 18 (`total: 18`, `total_pages: 1`). Until 2026-09-30 the paged route
was `GET /v1/bodhisattvas` (bodhisattvas only); it was replaced, without an alias, so clients must
call `/v1/deities` and filter by `type` themselves if they want one kind. Like temples, the service does not seed this table; rows arrive only through
`go run ./services/appService/cmd/seeddeities`.

**Current limitation**: the image URLs in the data are placeholders. Every `image_url` points at
`https://cdn.yourapp.com/images/buddhas/full/<id>.webp` and every `thumbnail_url` is a relative
path `assets/images/buddhas/thumbnails/<id>.webp`; neither resolves to anything this repo serves.
The seed command uses dev config and secrets, and no black-box E2E test covers the routes.

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. `List` reads count then page in two statements (no shared
snapshot). The only transaction is the seed upsert.

| Layer | Type | File |
|---|---|---|
| Domain / Entity | Domain Entity | `services/appService/domain/deity.go` — `Deity`, `MemorialDay`, `Iconography`, `Mantras`, `DeityPage`; constants `DeityBuddha`, `DeityBodhisattva` |
| Domain / Errors | sentinel errors | `domain/deity.go` — `ErrDeityNotFound`; `ErrInvalidPage` is shared (`domain/temple.go`) |
| Domain / Port | interface | `domain/deity.go` — `DeityRepository` (`ListTop`, `List`, `FindByID`, `Upsert`); ordering is part of its contract |
| Usecase | UseCase | `services/appService/usecase/deity_usecase.go` — `Top`, `ListDeities` (no type filter; rejects `page < 1` or `> MaxInt32`), `Get` |
| Repository | Repository | `services/appService/repository/postgres_deity_repository.go` — SQL on `deities`; `lunar_memorial_days` is stored as JSONB and always read whole; `Upsert` = one transaction, `ON CONFLICT (id) DO UPDATE` |
| Delivery / Handler | Handler | `services/appService/delivery/http/deity_handler.go` — `Top`, `ListDeities`, `Get`, DTOs; `delivery/http/errors.go` maps `ErrDeityNotFound` → `404 DEITY_NOT_FOUND` |
| Seed command | cmd | `services/appService/cmd/seeddeities/main.go` + `parse.go` — `parseDeities` rejects a type other than `BUDDHA`/`BODHISATTVA`, sets `SortOrder` = file position |
| Config | policy | `services/appService/config/config.go` `DeityConfig` and the `deity:` block in `config.dev.yaml` (`top_limit: 10`, `page_size: 20`) |
| Wiring | composition root | `services/appService/cmd/main.go` — repo → usecase → handler, 3 public routes; seeds nothing at startup, touches no `⚠️ DEV ONLY` block |
| Migration | schema | `migrations/0007_deities.sql` — `deities`, CHECK on `type`, `idx_deities_type_order` (now unused by queries), `idx_deities_order` |

`pkg/` dependencies: `pkg/database`, `pkg/utils`, `github.com/lib/pq`; the seed command also uses
`pkg/config` and the dev secret resolver.

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/v1/deities/top` | public | `DeityHandler.Top` |
| `GET` | `/v1/deities` | public | `DeityHandler.ListDeities` |
| `GET` | `/v1/deities/{id}` | public | `DeityHandler.Get` |

`/v1/deities/top` is a literal segment, so `ServeMux` picks it over `{id}`; `/v1/deities` (no trailing segment) matches neither. All three routes are in
`docs/specs/api/openapi.yaml`. No `api/` endpoint concepts exist yet.

Summary fields: `id`, `type`, `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`,
`thumbnail_url`, `image_url`, `core_virtue`, `pure_land` (plain strings, `""` when unknown — unlike
temples there is no `null`). Detail adds `canonical_source`, `lunar_memorial_days[{lunar_day,
lunar_month,event_name}]`, `iconography{posture,mount_or_throne,sacred_artifacts,symbolic_meaning}`,
`biography_summary`, `great_vows`, `mantras{short_mantra_sanskrit,short_mantra_sino_vietnamese,
long_dharani_name,dharani_phonetic_snippet}`. The list-valued fields are `[]`, never `null`.

# Request flow

```
GET /v1/deities?page=N
  └─ public
       └─ DeityHandler.ListDeities                       (delivery/http)
            ├─ page empty → 1; strconv.Atoi failure → 400 INVALID_PAGE
            └─ DeityUsecase.ListDeities(ctx, page)        (usecase)
                 ├─ page < 1 or > MaxInt32 → ErrInvalidPage
                 └─ DeityRepository.List(offset, size)     (repository)
                      ├─ SELECT count(*) FROM deities
                      └─ SELECT … ORDER BY sort_order, id OFFSET $1 LIMIT $2
```

Seeding, run by hand:

```
go run ./services/appService/cmd/seeddeities [-file assets/data/phat_botat.json]
  ├─ parseDeities(file)                 → unknown type aborts before any write
  ├─ load config.dev.yaml + dev secrets   (⚠️ DEV ONLY)
  └─ DeityRepository.Upsert             → one transaction, all rows or none
```

# Validation rules

- `page` — same two-step rule as temples: handler `strconv.Atoi` (empty = 1), usecase
  `page < 1 || page > math.MaxInt32`; both are `ErrInvalidPage` → `400 INVALID_PAGE`.
- `{id}` — no format rule; an unmatched id is `ErrDeityNotFound`. A bodhisattva id and a Buddha id
  both resolve through the same detail route.
- Seed — `parseDeities` validates the `type` value and ids; the database CHECK repeats the type rule.

# Pending work

- Image URLs are placeholders: `cdn.yourapp.com` for `image_url`, a non-existent relative path for
  `thumbnail_url`. `assets/images/home-calendar/` holds 100 `buddha_NNN.jpg` files plus a `manifest.json` and a
  `README.md`, but no Go file or data file references them; I did not check whether they belong to
  this catalogue.
- The route change from `/v1/bodhisattvas` to `/v1/deities` has unit tests (usecase, handler,
  repository) but was not re-run against the dev server, and the repository tests need Postgres so
  they are skipped under `-short`; `idx_deities_type_order` is no longer used by any query.
- No E2E test in `services/appService/e2e/`; repository tests need Postgres and are skipped
  under `-short`.
- `cmd/seeddeities` uses dev config and plaintext dev secrets (`⚠️ DEV ONLY`) and reads the JSON
  from disk rather than embedding it; the production seeding process is not decided.
- The religious content (`canonical_source`, `great_vows`, `mantras`, `lunar_memorial_days`) has no
  formal review step; a wrong vía date or mantra line is a content error nobody owns.
- Promote to `stable` once the routes are exercised by an automated end-to-end test against the
  real binary and a seeded Postgres.

# Known gaps vs. design

None that this author found against `docs/specs/features/deity/design.md`, which was written from
the code. Behaviours worth knowing:

- `total` and `items` of the paged list come from two statements without a shared snapshot.
- Re-running the seed overwrites every column of existing rows.
- `top` mixes both types by `sort_order`, so its composition depends on file order and `top_limit`.

# Related concepts

- [Temple](temple.md) and [Article](article.md) — sibling catalogues with the same public-route
  shape and shared `ErrInvalidPage`.
- [Login](login.md) and [Register](register.md) — identity features; deities share only
  `delivery/http/errors.go` and the middleware chain from [packages/middleware.md](../packages/middleware.md).
- No `Migration` concept exists yet for `migrations/0007_deities.sql`.
