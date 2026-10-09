---
type: Feature
title: Article
description: Read-only Buddhist article catalogue — a top-5 list, a paged list and a detail page, all public, backed by a table that is seeded once from JSON embedded in the binary.
resource: services/appService/
tags: [content, article, catalogue, seed, public]
status: draft
generated: { by: claude-code, at: 2026-09-30T17:00:00+07:00 }
sources:
  - path: docs/specs/features/article/requirements.md
    kind: spec
  - path: docs/specs/features/article/design.md
    kind: spec
  - path: docs/specs/features/article/flow.md
    kind: spec
  - path: docs/specs/features/article/tasks.md
    kind: spec
  - path: migrations/0012_articles.sql
    kind: migration
  - path: services/appService/delivery/http/article_handler.go
    kind: code
  - path: services/appService/usecase/article_usecase.go
    kind: code
  - path: services/appService/repository/postgres_article_repository.go
    kind: code
  - path: assets/embed.go
    kind: code
  - path: services/appService/cmd/seedarticles/main.go
    kind: code
---

# Summary

Three public `GET` routes serve the "Bài viết Phật pháp hay" reading list in the `content` module.
`GET /v1/articles/top` returns the first 5 articles as summary cards, `GET /v1/articles?page=N`
returns 10 per page with `page`, `page_size`, `total` and `total_pages`, and
`GET /v1/articles/{id}` returns one article with its full text, quote and source attribution. No
route takes an `Authorization` header. Every failure uses the shared `utils.ApiResponse` envelope:
a bad `page` is `400 INVALID_PAGE`, an unknown id is `404 ARTICLE_NOT_FOUND`.

Articles are ordered by `sort_order ASC, id ASC`, which is the curated order of the seed file: the
source carries no reliable date or popularity, so "top" and paging both follow that order.
`published_date` (`"YYYY-MM-DD"`) and `image_url` are JSON `null` when unknown. In the current
data 29 of 60 articles have a date and all 60 have an image (hotlinked: 38 from the source page's `og:image`, 15 from an image inside the source page, 7 illustrative Wikimedia Commons photos — see A-001).

On startup the service counts the `articles` rows and, only when the table is empty, inserts the
60 articles from `assets/data/buddhist_articles.json`, which is embedded in the binary. A restart
with data present neither parses the seed nor writes anything.

Articles added to the JSON later do not reach an existing database through the service. The manual
command `go run ./services/appService/cmd/seedarticles` inserts the missing ones by id and, with
`-fill-images`, sets `image_url` on rows where it is still `NULL`; it never changes any other column
of an existing row.

**Current limitation**: no black-box E2E test exists for these routes, for the first-run seed or for
`seedarticles`. Editing the JSON file does not change an already-seeded row.

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. There are no multi-statement reads; the only transaction is the seed
insert, owned by one repository method (`InsertMissing`). The list read is two independent
statements (count, then page) with no shared snapshot.

| Layer | Type | File |
|---|---|---|
| Domain / Entity | Domain Entity | `services/appService/domain/article.go` — `Article` (`PublishedDate *time.Time`, `ImageURL` empty = none), `ArticlePage` |
| Domain / Errors | sentinel errors | `domain/article.go` — `ErrArticleNotFound`; `ErrInvalidPage` is shared with the temple and deity catalogues (`domain/errors.go`) |
| Domain / Port | interface | `domain/article.go` — `ArticleRepository` (`ListTop`, `List`, `FindByID`, `Count`, `InsertMissing`, `FillMissingImages`); ordering is part of its contract |
| Usecase | UseCase | `services/appService/usecase/article_usecase.go` — `Top`, `List` (offset and `total_pages`, rejects `page < 1` or `> MaxInt32`), `Get`, `SeedIfEmpty(load)` |
| Repository | Repository | `services/appService/repository/postgres_article_repository.go` — SQL on `articles`; `InsertMissing` = one transaction, `ON CONFLICT (id) DO NOTHING`; `FillMissingImages` = one transaction; `pq.Array` for `body_paragraphs`/`tags` |
| Repository / Seed parser | function | `services/appService/repository/article_seed.go` — `ParseArticleSeed`: validates non-empty unique ids, non-empty titles, `YYYY-MM-DD` dates; `SortOrder` = position in the file |
| Seed command | cmd | `services/appService/cmd/seedarticles/main.go` — `InsertMissing` (new ids only) and, with `-fill-images`, `FillMissingImages` (`UPDATE … WHERE image_url IS NULL`, one transaction); dev config and secrets (`⚠️ DEV ONLY`) |
| Embedded data | `go:embed` | `assets/embed.go` (`BuddhistArticlesJSON`) embedding `assets/data/buddhist_articles.json` |
| Delivery / Handler | Handler | `services/appService/delivery/http/article_handler.go` — `Top`, `List`, `Get`, summary/detail DTOs; `services/appService/delivery/http/errors.go` maps `ErrArticleNotFound` → `404 ARTICLE_NOT_FOUND` |
| Config | policy | `services/appService/config/config.go` `ArticleConfig` and the `article:` block in `config.dev.yaml` (`top_limit: 5`, `page_size: 10`) |
| Wiring | composition root | `services/appService/cmd/main.go` — builds repo → usecase → handler, runs `SeedIfEmpty` right after the DB connects (`log.Fatalf` on failure), registers the 3 routes. Touches no `⚠️ DEV ONLY` block |
| Migration | schema | `migrations/0012_articles.sql` — `articles` table and `idx_articles_order (sort_order, id)` |

`pkg/` dependencies: `pkg/database` (queries and the seed transaction), `pkg/utils`
(`WriteSuccessResponse`, `WriteBusinessError`), `github.com/lib/pq` for array columns.

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/v1/articles/top` | public | `ArticleHandler.Top` |
| `GET` | `/v1/articles` | public | `ArticleHandler.List` |
| `GET` | `/v1/articles/{id}` | public | `ArticleHandler.Get` |

`/v1/articles/top` is a literal segment, so `ServeMux` picks it over `{id}`; an article whose id
were `top` could not be read through the detail route. No `api/` endpoint concepts exist yet.

Summary fields: `id`, `title`, `category`, `author`, `author_initials`, `published_date`
(`string | null`), `read_minutes`, `summary`, `tags`, `image_url` (`string | null`). Detail adds
`body_paragraphs`, `quote`, `quote_source`, `language`, `source_name`, `source_url`, `license`.
`tags` and `body_paragraphs` are `[]`, never `null`.

# Request flow

```
GET /v1/articles?page=N
  └─ public
       └─ ArticleHandler.List                          (delivery/http)
            ├─ page empty → 1; strconv.Atoi failure → 400 INVALID_PAGE
            └─ ArticleUsecase.List(ctx, page)           (usecase)
                 ├─ page < 1 or > MaxInt32 → ErrInvalidPage
                 └─ ArticleRepository.List(offset, size) (repository)
                      ├─ SELECT count(*) FROM articles
                      └─ SELECT … ORDER BY sort_order, id OFFSET $1 LIMIT $2
```

First-run seed, at startup in `cmd/main.go`:

```
ArticleUsecase.SeedIfEmpty(ctx, load)
  ├─ ArticleRepository.Count                     → n > 0: stop, nothing parsed or written
  ├─ load() = ParseArticleSeed(BuddhistArticlesJSON)
  └─ ArticleRepository.InsertMissing             → one transaction, ON CONFLICT (id) DO NOTHING
```

# Validation rules

- `page` — parsed in the handler with `strconv.Atoi`; empty means 1. A non-integer (including an
  overflowing one) is `ErrInvalidPage`. The usecase adds the range rule `page < 1 || page > math.MaxInt32`,
  also `ErrInvalidPage` → `400 INVALID_PAGE` with `details.field = "page"`.
- `{id}` — no format rule; an id that matches no row is `ErrArticleNotFound`.
- Seed — `ParseArticleSeed` rejects malformed JSON, an empty article list, an empty or duplicate
  `id`, an empty `title` and a `publishedDate` that is not `YYYY-MM-DD`. Any rejection aborts startup.

# Pending work

- No E2E test in `services/appService/e2e/` for the three routes or for the first-run seed
  (start twice, second start seeds nothing); none for `seedarticles` either.
- `image_url` is set for all 60 seed articles (hotlinked; 7 are topic illustrations from Wikimedia Commons, not the article's own image). Rows seeded before 2026-10-06 still have `NULL` for 22 of them until `seedarticles -fill-images` is run.
- Content edits have no tooling: `seedarticles` only adds new ids and fills empty images, so a
  corrected title or text in the JSON never reaches an existing row, and there is no backoffice CRUD.
  `seedarticles` uses dev config and plaintext dev secrets (`⚠️ DEV ONLY`).
- Every article carries `license = all-rights-reserved-summarized` with `source_name`/`source_url`
  pointing at the original piece; the conditions for showing these summaries are not confirmed.
- Promote to `stable` once the routes and the seed are exercised by an automated end-to-end test
  against the real binary and Postgres.

# Known gaps vs. design

None between the code and `docs/specs/features/article/design.md`; the spec was written from the
code. Behaviours worth knowing that the spec records as gaps rather than fixing:

- Emptying the whole table and restarting reseeds all 60 articles; "never resurrect deleted rows"
  holds only while at least one row remains.
- `total` and `items` come from two statements without a shared snapshot.
- The repository tests `TRUNCATE` the dev database's tables, `articles` included, and are skipped
  under `-short`.

# Related concepts

- [Login](login.md) and [Register](register.md) — the identity features. Articles share only
  `delivery/http/errors.go`'s `writeDomainError` and the middleware chain from
  [packages/middleware.md](../packages/middleware.md); no auth is involved.
- [Temple](temple.md) and [Deity](deity.md) — the closest siblings (same top / paged list / detail
  shape and shared `ErrInvalidPage`). Unlike them, articles are also seeded automatically on first
  start from an embedded file.
- No `Migration` concept exists yet for `migrations/0012_articles.sql`.
