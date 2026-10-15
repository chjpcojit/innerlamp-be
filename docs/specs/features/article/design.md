# Design — article

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — top / paged list / detail and the seed command are implemented; related articles (v3.0) are designed, not implemented.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

All `GET /v1/articles*` routes are public and pass no authentication middleware. The feature touches
the four Clean Architecture layers of `services/appService` (`domain` → `usecase` → `repository` /
`delivery/http`, wired in `cmd/main.go`), owns migrations `0012_articles.sql` and
`0016_article_thumbnail.sql`, and uses the small `assets/` package that `go:embed`s the content file.
Same shape as `temple` and `deity` (top + paged list + detail, `top_limit`/`page_size` config); data is
loaded only by `cmd/seedarticles`.

v3.0 adds one read: the related articles of one article (BR-ART-04), computed by one SQL query after an
existence check. No schema change.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/article.go` | `Article` (every column; `PublishedDate *time.Time`, `ImageURL`/`ThumbnailURL` `""` = none) and `ArticlePage`. No JSON/db tags. |
| Domain / Errors | `domain/article.go` | `ErrArticleNotFound`; `ErrInvalidPage` is shared with `temple`/`deity` and defined in `domain/temple.go`. |
| Domain / Repository interface | `domain/article.go` | `ArticleRepository`: `ListTop`, `List`, `FindByID`, `Count`, `InsertMissing`, `FillMissingImages`, **v3.0** `ListRelated(ctx, id, category, limit)`. The order `sort_order ASC, id ASC` is part of the contract. |
| Domain / Mock | `domain/mock_article_repository.go` | Hand-written mock for usecase tests; **v3.0** adds `ListRelated`. |
| Usecase | `services/appService/usecase/article_usecase.go` | `Top`, `List` (offset, `total_pages`, rejects `page < 1` or `> MaxInt32`), `Get`, `SeedIfEmpty(load)` (kept, no production caller), **v3.0** `Related(ctx, id)`: `FindByID` then `ListRelated` with `cfg.RelatedLimit`. |
| Repository | `services/appService/repository/postgres_article_repository.go` | SQL for `articles`; `InsertMissing` in one transaction with `ON CONFLICT (id) DO NOTHING`; `FillMissingImages` fills `image_url`/`thumbnail_url` only where `NULL`; `NULL` ↔ `nil`/`""`; `pq.Array` for `body_paragraphs`/`tags`. **v3.0** `ListRelated` (query below). |
| Repository / Seed parser | `services/appService/repository/article_seed.go` | `ParseArticleSeed`: decodes `{"articles":[…]}`, checks id/title/date, sets `SortOrder` = position in the file. |
| Embedded data | `assets/embed.go`, `assets/data/buddhist_articles.json` | `go:embed` of the content file into `BuddhistArticlesJSON`. **v3.0**: the JSON file is committed (`.gitignore` exception). |
| Delivery / Handler | `services/appService/delivery/http/article_handler.go` | `Top`, `List`, `Get`, **v3.0** `Related`; parses `page`; summary/detail DTOs; `published_date` as `2006-01-02`; image paths resolved by `imageHost` (`WithImageBase(server.public_base_url)`); `nonNil` for arrays. |
| Delivery / DTO | `delivery/http/article_handler.go` | `articleSummaryResponse`, `articleDetailResponse` (embeds summary), `articleTopResponse` (also used by `Related`), `articleListResponse`. JSON tags live only here. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrArticleNotFound` → `404 ARTICLE_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`). Unchanged in v3.0. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `ArticleConfig{TopLimit, PageSize}` (5, 10); **v3.0** `RelatedLimit` (2), validated `> 0` like the other two. |
| Wiring | `services/appService/cmd/main.go` | Builds repo → usecase → handler; registers the routes; **v3.0** adds `GET /v1/articles/{id}/related`. No seeding on start. No `⚠️ DEV ONLY` block involved. |
| Migration | `migrations/0012_articles.sql`, `migrations/0016_article_thumbnail.sql` | Table `articles` + index `idx_articles_order (sort_order, id)`; `thumbnail_url TEXT NULL`. No v3.0 migration. |
| API spec | `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` | Three shipped routes with `ArticleSummary`/`ArticleDetail`; **v3.0** adds `/v1/articles/{id}/related` (operation `articlesRelated`) to both copies. |

Routing note: `GET /v1/articles/top` is a literal segment, so `ServeMux` prefers it over
`GET /v1/articles/{id}`; `GET /v1/articles/{id}/related` has one more segment and does not collide.

## Request flow

`top` and `detail` are one SQL statement each, so they stay as ASCII; the paged list, related articles
and the seed have several steps and are in the Step-by-step Flows and `flow.md`.

```
GET /v1/articles/top
  └─ public
       └─ ArticleHandler.Top                          (delivery/http)
            └─ ArticleUsecase.Top(ctx)                (usecase)
                 └─ ArticleRepository.ListTop(limit)  (repository)
                      └─ SELECT … FROM articles ORDER BY sort_order, id LIMIT $1

GET /v1/articles/{id}
  └─ public
       └─ ArticleHandler.Get                          (delivery/http)
            └─ ArticleUsecase.Get(ctx, id)            (usecase)
                 └─ ArticleRepository.FindByID(id)    (repository)
                      └─ SELECT … FROM articles WHERE id = $1
                           └─ sql.ErrNoRows → ErrArticleNotFound → 404 ARTICLE_NOT_FOUND
```

> See `flow.md` for the Mermaid sequence diagrams of the three multi-step flows below (1:1 mirror).

## Step-by-step Flow — Paged list (`GET /v1/articles?page=N`)

Two sequential queries (count, then page).

```
Step 1 — Read and check `page` (handler)
- Trigger: request reaches the public route
- Pre-condition: —
- Call: r.URL.Query().Get("page"); empty → page = 1, else strconv.Atoi
    - Success → Step 2
    - Failure (not an integer, including int overflow) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Check the range of `page` (usecase)
- Trigger: handler calls ArticleUsecase.List(ctx, page)
- Pre-condition: page is an integer
- Call: page < 1 or page > math.MaxInt32
    - Success → Step 3 (offset = (page-1) * page_size)
    - Failure → domain.ErrInvalidPage → 400 INVALID_PAGE

Step 3 — Count articles (repository)
- Trigger: usecase calls ArticleRepository.List(offset, limit)
- Pre-condition: valid offset/limit
- Call: SELECT count(*) FROM articles
    - Success → total, go to Step 4
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 4 — Read the page (repository)
- Trigger: Step 3 done
- Pre-condition: total known
- Call: SELECT <columns> FROM articles ORDER BY sort_order ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (empty past the end) and total, go to Step 5
    - Failure (DB / scan error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — Step 3 and Step 4 are two independent statements, no shared snapshot

Step 5 — Build the page and respond (usecase + handler)
- Trigger: repository returns
- Pre-condition: items, total
- Call: nil items → []; total_pages = (total + page_size - 1) / page_size; handler maps to summary DTOs
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → no new failure branch
```

## Step-by-step Flow — Related articles (`GET /v1/articles/{id}/related`, v3.0)

An existence check, then one selection query. Implements BR-ART-04.

```
Step 1 — Read the path (handler)
- Trigger: request reaches the public route
- Pre-condition: —
- Call: r.PathValue("id")
    - Success → Step 2 (no format validation, same as Get: an unknown id is a 404)
    - Failure → none

Step 2 — Load the current article (usecase → repository)
- Trigger: handler calls ArticleUsecase.Related(ctx, id)
- Pre-condition: —
- Call: ArticleRepository.FindByID(ctx, id) → SELECT … FROM articles WHERE id = $1
    - Success → the article's category, go to Step 3
    - Failure (no row) → ErrArticleNotFound → 404 ARTICLE_NOT_FOUND
    - Failure (DB error) → error wrapped with %w → 500 INTERNAL_ERROR

Step 3 — Select the related articles (repository)
- Trigger: Step 2 found the article
- Pre-condition: limit = article.related_limit (> 0, checked at config load)
- Call: ArticleRepository.ListRelated(ctx, id, category, limit) →
    SELECT <summary columns> FROM articles
    WHERE id <> $1
    ORDER BY (lower(btrim(category)) = lower(btrim($2))) DESC, sort_order ASC, id ASC
    LIMIT $3
    - Success → 0..limit articles: same-category ones first in editorial order, then the rest in editorial order (BR-ART-04 fill-up); go to Step 4
    - Failure (DB / scan error) → error wrapped with %w → 500 INTERNAL_ERROR
    - Transaction boundary: none — Step 2 and Step 3 are independent reads; an article deleted in between only changes which rows come back

Step 4 — Respond (handler)
- Trigger: usecase returns
- Pre-condition: items (nil → [])
- Call: map to summary DTOs (articleTopResponse)
    - Success → 200 {items}
    - Failure → no new failure branch
```

`lower()` on Vietnamese text relies on a UTF-8 `LC_CTYPE` in Postgres, which the temple and sutra
searches already assume (`unaccent(lower(…))`). The rule's vectors are run against this query
(repository test) and against a pure ordering helper (usecase test), so both agree.

## Step-by-step Flow — Seed with the `cmd/seedarticles` command (the service never seeds)

Reading the file and the insert transaction are sequential effects; the insert is a persistent side
effect.

```
Step 1 — Parse and check the content (cmd/seedarticles → repository)
- Trigger: `make ensure-articles` (only when the table is empty) or a manual run of cmd/seedarticles
- Pre-condition: DB reachable; migration 0012 (and 0016) applied
- Call: repository.ParseArticleSeed(assets.BuddhistArticlesJSON, or the -file content)
    - Success → []domain.Article, SortOrder = position in the file, go to Step 2
    - Failure (bad JSON, no article, empty/duplicate id, empty title, bad publishedDate) → log.Fatalf, nothing written

Step 2 — Insert the missing articles (repository)
- Trigger: Step 1 done
- Pre-condition: valid content
- Call: ArticleRepository.InsertMissing → INSERT … ON CONFLICT (id) DO NOTHING per article
    - Success → number of rows actually inserted (60 on first run), go to Step 3
    - Failure (DB error, e.g. table missing) → full rollback → log.Fatalf
    - Transaction boundary: every INSERT of Step 2 in ONE transaction — all or nothing

Step 3 — Fill missing images (repository, only with -fill-images)
- Trigger: Step 2 done and -fill-images set
- Pre-condition: —
- Call: ArticleRepository.FillMissingImages → UPDATE … SET image_url = COALESCE(image_url,$2), thumbnail_url = COALESCE(thumbnail_url,$3) WHERE id = $1
    - Success → number of rows changed, go to Step 4
    - Failure → rollback → log.Fatalf
    - Transaction boundary: one transaction for all UPDATEs

Step 4 — Log (cmd/seedarticles)
- Trigger: previous step returns
- Pre-condition: —
- Call: logger.Info with the inserted (and filled) counts
    - Success → exit 0
    - Failure → no failure branch
```

## Deliberately stubbed for now (if applicable)

Nothing is stubbed: every shipped `ArticleRepository` method runs real SQL. `ListRelated` does not
exist yet (v3.0, `tasks.md` Phase 7).

## Known gaps carried by this feature

- **No E2E**: no test runs the real binary against Postgres for the article routes
  (`services/appService/e2e/` has none).
- **Repository tests need Postgres** and are skipped under `-short`; they run on the scratch database
  of the test helper.
- **Editing the content file does not update an existing database**: the seed inserts missing ids
  only; `-fill-images` fills `NULL` image columns only. Other columns change only by hand-written SQL
  (product Appendix A item 4).
- **An emptied table is reloaded**: "never bring back a deleted article" holds only while at least one
  row remains; `make ensure-articles` on an empty table reloads all 60.
- **Count and page are not one snapshot** (paged list Step 3/4); data changes only by hand, so the risk
  is low.
- **`id = "top"` is shadowed** by the literal route (AC 10).
- **Thumbnails** are 360x360 WebP copies served by `GET /static/images/` (`server.static_images_dir`),
  derived from third-party images (A-002); `image_url` is a hotlink (53 source sites, 7 Wikimedia
  Commons — A-001). No GCS copy yet; rights are open (product A1).
- **Other embedded data is still untracked**: `assets/embed.go` also embeds
  `assets/data/dharma_talks.json`, which is not in git, so a fresh clone still fails to build until
  that file is committed (feature `dharmatalk`, not this one).
