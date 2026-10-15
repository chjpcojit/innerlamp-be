# Tasks — article

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — Phase 1–4 implemented (2026-09-30); Phase 5 E2E not done; Phase 6 is deliberately deferred; Phase 7 (v3.0 related articles) not started; Phase 8 (content file in git) done.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Every item traces to a row of "Layers & components" or a Step of a "Step-by-step Flow" in `design.md`.
Only items verified in the code are `[x]` (`go build ./...`, `go vet`, `go test -short` for `usecase`,
`delivery/http`, `repository`, and real calls to the three routes on the dev server, 2026-09-30).

## Phase 1 — Domain

- [x] `Article`, `ArticlePage` in `services/appService/domain/article.go` (no tags)
- [x] `ErrArticleNotFound` in `domain/article.go`; reuse `ErrInvalidPage` (defined in `domain/temple.go`)
- [x] `ArticleRepository` interface (order `sort_order ASC, id ASC` is part of the contract) in `domain/article.go`
- [x] `domain/mock_article_repository.go` for usecase tests

## Phase 2 — Usecase

- [x] `usecase/article_usecase.go`: `Top`, `List` (offset, `total_pages`, rejects `page < 1` / `> MaxInt32`), `Get`, `SeedIfEmpty(load)`
- [x] Unit tests: `usecase/article_usecase_test.go` — `Top`, `List`, `Get`, `SeedIfEmpty`, hand-written mock, no real DB

## Phase 3 — Repository and data

- [x] `migrations/0012_articles.sql` — table `articles`, index `idx_articles_order`
- [x] `repository/postgres_article_repository.go`: `ListTop`, `List`, `FindByID`, `Count`, `InsertMissing` (one transaction, `ON CONFLICT DO NOTHING`), `NULL` ↔ `nil`/`""`
- [x] `FillMissingImages` (interface + mock + repository) and the `-fill-images` flag of `cmd/seedarticles`; test `TestArticleRepository_FillMissingImages`
- [x] `repository/article_seed.go`: `ParseArticleSeed` (checks id, title, date; `SortOrder` = position)
- [x] `assets/embed.go` embeds `assets/data/buddhist_articles.json`
- [x] `thumbnail_url`: `migrations/0016_article_thumbnail.sql`, entity, repository, seed parser, DTOs (`null` when missing), 60 images in `assets/images/articles/thumb/`
- [x] Tests: `repository/postgres_article_repository_test.go` (insert/find, no overwrite, atomic, order and paging; needs Postgres, skipped under `-short`), `repository/article_seed_test.go` (embedded data and validation)

## Phase 4 — Delivery and docs

- [x] `delivery/http/article_handler.go`: `Top`, `List`, `Get` + summary/detail DTOs
- [x] `delivery/http/errors.go`: `ErrArticleNotFound` → `404 ARTICLE_NOT_FOUND`
- [x] `config`: `ArticleConfig`, block `article:` in `config.dev.yaml`
- [x] `cmd/main.go`: wiring, 3 routes registered; no seeding on start
- [x] Handler tests: `delivery/http/article_handler_test.go` (+ `mock_article_usecase_test.go`)
- [x] `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml`: 3 routes + schemas
- [x] Table `articles` in `docs/specs/db/erd.mmd` / `erd.svg`

## Phase 5 — Integration

- [ ] E2E black-box in `services/appService/e2e/article_e2e_test.go`: top, paged list (`INVALID_PAGE`, past the end), detail (`ARTICLE_NOT_FOUND`), related (Phase 7), after running `cmd/seedarticles` on the ephemeral database
- [x] `go build ./...` and `go vet ./services/appService/... ./assets/...` clean (2026-09-30)
- [x] OKF concept `docs/knowledge/features/article.md` (`status: draft`)

## Phase 6 — Deferred (separate features, not blocking article)

- Creating / editing / removing articles in the back office or by a full reseed (product A4).
- Search, filtering by `category`/`tag`, bookmarks (product out of scope / BR-ART-09).
- Copying cover images to GCS (instead of hotlinks) — needs a bucket and the image-rights answer (product A1).

## Phase 7 — Related articles (v3.0, design "Step-by-step Flow — Related articles")

- [ ] Config: `ArticleConfig.RelatedLimit` + `article.related_limit: 2` in `config.dev.yaml` (and every env file); validated `> 0` with the other two; config test
- [ ] Domain: `ArticleRepository.ListRelated(ctx, id, category, limit)` + mock method
- [ ] Repository: `ListRelated` SQL (`WHERE id <> $1 ORDER BY (lower(btrim(category)) = lower(btrim($2))) DESC, sort_order, id LIMIT $3`); `nil` → empty slice
- [ ] Repository test: run every case of `docs/product/article/rules/article-related.json` against Postgres (insert the case's articles, call `ListRelated`, compare ids) and `article-order.json` against `List`
- [ ] Usecase: `ArticleUsecase.Related(ctx, id)` — `FindByID` (→ `ErrArticleNotFound`) then `ListRelated(cfg.RelatedLimit)`; unit tests with the mock (found, not found, DB error on each step)
- [ ] Handler: `ArticleHandler.Related` → `200 {items}` (summary DTOs, `[]` not `null`); handler tests (200, 404, 500, 405 with `Allow: GET, HEAD`)
- [ ] Wiring: `mux.HandleFunc("GET /v1/articles/{id}/related", articleHandler.Related)` in `cmd/main.go`
- [ ] Contract: add `/v1/articles/{id}/related` (operation `articlesRelated`, tag Articles, response `{items: [ArticleSummary]}`, `404 ARTICLE_NOT_FOUND`) to `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml`
- [ ] E2E: related route in `article_e2e_test.go` (Phase 5)

## Phase 8 — Content file in git (v3.0)

- [x] `.gitignore` exception `!assets/data/buddhist_articles.json` and the file committed (60 articles, root-relative thumbnail paths)
