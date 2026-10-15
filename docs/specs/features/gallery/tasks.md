# Tasks — gallery

Version: 2.0
Last updated: 2026-10-15
Status: in-progress — Phase 1–4 done (except the items marked `[ ]`); **Phase 6–8 (v2.0) not started**; Phase 5 is intentionally deferred work.
Module: `gallery`

> See `changelog.md` in this folder for the version history of this spec.

Every item traces to a row of "Layers & components" or a Step of a "Step-by-step Flow" in `design.md`. Only what is verified in code is `[x]`. The v2.0 pass (2026-10-15) re-read `domain/gallery.go`, `usecase/gallery_usecase.go`, `delivery/http/gallery_handler.go`, `cmd/seedgallery/parse.go`, `cmd/main.go`, migration `0017`, and ran `go test -short ./services/appService/cmd/seedgallery/ ./services/appService/usecase/ ./services/appService/delivery/http/ -run Gallery` — all pass (the seeder's real-file test now finds the committed file).

## Phase 1 — Domain

- [x] `GalleryImage`, `GalleryPage`, type constants, `GalleryTypes`, `IsValidGalleryType` in `services/appService/domain/gallery.go` (no tags)
- [x] `ErrGalleryImageNotFound`, `ErrInvalidGalleryType`; reuses `ErrInvalidPage`
- [x] `GalleryRepository` interface (order `sort_order ASC, id ASC` in the contract)
- [x] `domain/mock_gallery_repository.go`

## Phase 2 — Usecase

- [x] `usecase/gallery_usecase.go`: `Top`, `List` (page bounds, type check, offset, `total_pages`, nil → `[]`), `Get`
- [x] Unit tests `usecase/gallery_usecase_test.go` with hand-written mocks

## Phase 3 — Repository, config and data

- [x] `migrations/0017_gallery_images.sql` — table, `CHECK` on `type`, two indexes
- [x] `repository/postgres_gallery_repository.go`: `ListTop`, `List`, `FindByID`, `Upsert` (one transaction)
- [x] `config`: `GalleryConfig` + `gallery:` block (10 / 20) in `config.dev.yaml`
- [x] `cmd/seedgallery/main.go` + `parse.go`: id/type/title/URL checks, type rotation, `Upsert`; `make seed-gallery`
- [x] Tests: `cmd/seedgallery/parse_test.go`; `repository/postgres_gallery_repository_test.go` (needs Postgres, skipped under `-short`)
- [x] `assets/data/buddhist_gallery.json` committed with a `.gitignore` exception (be#35) — AC 20

## Phase 4 — Delivery and docs

- [x] `delivery/http/gallery_handler.go`: `Top`, `List`, `Get` + summary/detail DTOs
- [x] `delivery/http/errors.go`: `404 GALLERY_IMAGE_NOT_FOUND`, `400 INVALID_GALLERY_TYPE`
- [x] `cmd/main.go`: wiring + 3 routes
- [x] Handler tests `delivery/http/gallery_handler_test.go` + `mock_gallery_usecase_test.go`
- [x] `openapi.yaml` (+ `web/docs/public/`): 3 routes, tag Gallery, schemas (v1.x shape)
- [ ] Re-render `docs/specs/db/erd.svg` from `erd.mmd`

## Phase 5 — Deferred (not this feature's v2.0)

- Production photo storage and serving — waits on the cross-feature ADR (`requirements.md` ⚠️)
- Content load outside dev — waits on the same product question

## Phase 6 — v2.0 usecase and delivery (credit in lists, `total` on `top`)

- [ ] `GalleryUsecase.Top` returns `(items, total)` via `GalleryRepository.List(ctx, "", 0, cfg.TopLimit)` — design Step-by-step "Featured photos" Steps 1–3 (AC 1–3)
- [ ] Remove `GalleryRepository.ListTop` (repository, interface, mock, tests) once `Top` no longer calls it
- [ ] `gallerySummaryResponse` gains `author`, `license`, `source_url`; `galleryDetailResponse` = same fields; `galleryTopResponse` gains `total` (AC 3, 11, 13)
- [ ] Update `usecase/gallery_usecase_test.go` (`Top` returns total, empty table → `[]` and `0`) and `delivery/http/gallery_handler_test.go` (credit fields on top and list items, `total` on top)

## Phase 7 — v2.0 seeder and contract

- [ ] `parseGallery` refuses an empty `author` or `license` (AC 16); add cases to `parse_test.go`
- [ ] `cmd/seedgallery/parse_vectors_test.go`: run every case of `docs/product/gallery/rules/gallery-order.json` (type mapping TEMPLE→CHUA, STATUE→TUONG_PHAT, MEDITATION→THIEN, FESTIVAL→LE_HOI; featured = first `top_limit` of the order; filter = order restricted to the type) (AC 19)
- [ ] `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml`: `GallerySummary` + `author`, `license`, `source_url` (required strings); `GalleryTopEnvelope` + `total`; gallery descriptions in English (copy `diff` must be empty)

## Phase 8 — v2.0 verification

- [ ] `e2e/gallery_e2e_test.go`: `top` (10 items, `total`, credit fields), list page 1 and past the end, `type` filter counts, `INVALID_PAGE`, `INVALID_GALLERY_TYPE` (`details.field`), `GALLERY_IMAGE_NOT_FOUND`, empty table → `items: []`
- [ ] `go build ./...`, `go vet ./services/appService/...`, `go test -short ./services/appService/...` clean
- [ ] OKF concept `docs/knowledge/features/gallery.md` — by `il-okf-writer` after E2E
