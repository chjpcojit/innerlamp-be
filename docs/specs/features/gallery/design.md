# Design — gallery

Version: 2.0.2
Last updated: 2026-10-15
Status: in-progress — v1.x layers implemented; the **v2.0** rows below are not.
Module: `gallery`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Three public `GET` routes (no `RequireAuth`), read-only, through the four Clean Architecture layers, plus the manually run seeder. v2.0 changes no table and no route: it widens the list item, adds `total` to `top` and tightens the seeder's validation.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/gallery.go` | `GalleryImage` (id, type, sortOrder, title, location, thumbnailURL, imageURL, author, license, sourceURL), `GalleryPage`, constants `GalleryChua`/`GalleryTuongPhat`/`GalleryThien`/`GalleryLeHoi`, `GalleryTypes` (also the rotation order), `IsValidGalleryType`. No tags. |
| Domain / Errors | `domain/gallery.go` | `ErrGalleryImageNotFound`, `ErrInvalidGalleryType`; reuses `ErrInvalidPage`. |
| Domain / Repository interface | `domain/gallery.go` | `GalleryRepository`: `ListTop`, `List(ctx, imageType, offset, limit) (items, total, err)`, `FindByID`, `Upsert`. Order `sort_order ASC, id ASC` is part of the contract. |
| Domain / Mock | `domain/mock_gallery_repository.go` | Hand-written mock for usecase tests. |
| Usecase | `usecase/gallery_usecase.go` | `List`, `Get` unchanged. **v2.0** `Top` returns `(items, total)`: it calls `List(ctx, "", 0, cfg.TopLimit)` instead of `ListTop`, so one call gives both (AC 1–3). `ListTop` stays in the interface but loses its only caller; remove it in the same change. |
| Repository | `repository/postgres_gallery_repository.go` | SQL, `Upsert` in one transaction, scan. Unchanged in v2.0 (already selects the credit columns). |
| Delivery / Handler + DTO | `delivery/http/gallery_handler.go` | **v2.0** `gallerySummaryResponse` gains `author`, `license`, `source_url`; `galleryDetailResponse` becomes the same shape; `galleryTopResponse` gains `total`. |
| Delivery / Image URLs | `delivery/http/image_url.go` | Absolute URLs from root-relative paths (A-001). Unchanged. |
| Delivery / Errors | `delivery/http/errors.go` | `GALLERY_IMAGE_NOT_FOUND` 404, `INVALID_GALLERY_TYPE` 400. Unchanged. |
| Config | `config/config.go`, `config.dev.yaml` | `GalleryConfig{TopLimit: 10, PageSize: 20}`. Unchanged (BR-GAL-03: apps 20, website cuts 12 from them). |
| Seeder | `cmd/seedgallery/{main,parse}.go` | **v2.0** `parseGallery` also refuses an empty `author` or `license` (AC 16). Rotation unchanged (AC 17). |
| Vector test | `cmd/seedgallery/parse_vectors_test.go` (new) | **v2.0** Runs `docs/product/gallery/rules/gallery-order.json` through `parseGallery`'s ordering + the `top` limit (AC 19). |
| Content | `assets/data/buddhist_gallery.json` | 86 photos with credits; committed (be#35). Photo files under `assets/images/gallery/` stay out of git (`⚠️ DEV ONLY`). |
| Wiring | `cmd/main.go` | Builds repo → usecase → handler, registers 3 routes. Unchanged. |
| E2E | `e2e/gallery_e2e_test.go` (new) | **v2.0** Black-box: three routes, filter, error codes, credit fields, `total` on `top`. |
| API spec | `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` | **v2.0** `GallerySummary` + 3 fields; `GalleryTopEnvelope` + `total`; descriptions in English. |

## Request flow

```
GET /v1/gallery/top
  └─ public
       └─ GalleryHandler.Top
            └─ GalleryUsecase.Top(ctx)                       (v2.0: returns items + total)
                 └─ GalleryRepository.List(ctx, "", 0, cfg.TopLimit)
                      ├─ SELECT count(*) FROM gallery_images
                      └─ SELECT … ORDER BY sort_order, id LIMIT $1

GET /v1/gallery/{id}
  └─ public
       └─ GalleryHandler.Get
            └─ GalleryUsecase.Get(ctx, id)
                 └─ GalleryRepository.FindByID → SELECT … WHERE id = $1 (no rows → ErrGalleryImageNotFound)
```

## Step-by-step Flow — Featured photos (`GET /v1/gallery/top`) — v2.0

```
Step 1 — Read featured page (usecase → repository)
- Trigger: client calls GET /v1/gallery/top
- Pre-condition: none (public)
- Call: GalleryUsecase.Top → GalleryRepository.List(ctx, "", 0, cfg.TopLimit)
    - Success → Step 2
    - Failure (DB error) → 500 INTERNAL_ERROR

Step 2 — Count (repository)
- Trigger: Step 1
- Call: SELECT count(*) FROM gallery_images
    - Success → total
    - Failure → 500 INTERNAL_ERROR
    - Transaction boundary: Steps 2 and 3 are two independent statements, NOT one snapshot (as in the list)

Step 3 — Read the first rows (repository)
- Trigger: Step 2 succeeded
- Call: SELECT <columns> FROM gallery_images ORDER BY sort_order ASC, id ASC OFFSET 0 LIMIT top_limit
    - Success → items (empty table → [])
    - Failure → 500 INTERNAL_ERROR

Step 4 — Respond (delivery)
- Trigger: Step 3 succeeded
- Call: map items to the summary DTO (absolute image URLs, credit fields)
    - Success → 200 {items, total}
```

## Step-by-step Flow — Paged list (`GET /v1/gallery?type=&page=`)

```
Step 1 — Parse query (delivery)
- Trigger: client calls GET /v1/gallery
- Pre-condition: none
- Call: empty page → 1, else strconv.Atoi; type = query "type" (empty = every type)
    - Success → GalleryUsecase.List(ctx, type, page)
    - Failure (page not an integer) → 400 INVALID_PAGE

Step 2 — Validate (usecase)
- Trigger: Step 1 succeeded
- Call: page < 1 or > MaxInt32 → ErrInvalidPage; type != "" and not one of the 4 values → ErrInvalidGalleryType
    - Success → GalleryRepository.List(ctx, type, offset, page_size)
    - Failure → 400 INVALID_PAGE / 400 INVALID_GALLERY_TYPE

Step 3 — Count (repository)
- Trigger: Step 2 succeeded
- Call: SELECT count(*) FROM gallery_images [WHERE type = $1]
    - Success → total
    - Failure → 500 INTERNAL_ERROR
    - Transaction boundary: Steps 3 and 4 are two independent statements, NOT one snapshot

Step 4 — Read the page (repository)
- Trigger: Step 3 succeeded
- Call: SELECT <columns> FROM gallery_images [WHERE type = $1] ORDER BY sort_order ASC, id ASC OFFSET $ LIMIT $
    - Success → items (empty past the end)
    - Failure → 500 INTERNAL_ERROR

Step 5 — Build the page (usecase → delivery)
- Trigger: Step 4 succeeded
- Call: nil items → []; total_pages = ceil(total / page_size); map to the summary DTO (v2.0: with credit)
    - Success → 200 {items, page, page_size, total, total_pages}
```

## Step-by-step Flow — Content load (`cmd/seedgallery`)

```
Step 1 — Read and validate the file (seedgallery/parse.go)
- Trigger: operator runs `go run ./services/appService/cmd/seedgallery [-file ...]`
- Pre-condition: the file exists (default assets/data/buddhist_gallery.json)
- Call: json.Unmarshal → non-empty list; unique non-empty id; known type; non-empty title, image_url, thumbnail_url; v2.0 non-empty author and license
    - Success → list of GalleryImage
    - Failure → log.Fatalf, no DB access

Step 2 — Compute sort_order (parse.go)
- Trigger: Step 1 succeeded
- Call: rotate CHUA → TUONG_PHAT → THIEN → LE_HOI, keep file order within a type, skip exhausted types; sort_order = position
    - Success → every image has SortOrder

Step 3 — Upsert (repository)
- Trigger: Step 2 succeeded
- Pre-condition: DB reachable with dev config, migration 0017 applied
- Call: GalleryRepository.Upsert → INSERT … ON CONFLICT (id) DO UPDATE
    - Success → log "seeded N images"
    - Failure → rollback everything, log.Fatalf
    - Transaction boundary: every row in ONE transaction — all or none
```

## Known gaps carried by this feature

- Photos are served from the dev machine's file system (`⚠️ DEV ONLY`); production storage waits on the cross-feature ADR (`requirements.md`).
- The seeder uses dev config, like the other seeders.
- `docs/specs/db/erd.svg` not re-rendered from `erd.mmd` since v1.0.
