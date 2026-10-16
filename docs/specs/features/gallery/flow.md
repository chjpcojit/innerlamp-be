# Flow — gallery

Version: 2.0
Last updated: 2026-10-15
Status: in-progress — mirrors the three Step-by-step Flow sections of `design.md` 1:1; the `top` flow is **v2.0** (not implemented).
Module: `gallery`

> See `changelog.md` in this folder for the version history of this spec.

The detail route is a single SQL statement, so it has no diagram — see "Request flow" in `design.md`.

## Sequence — Featured photos (`GET /v1/gallery/top`) — v2.0

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as GalleryHandler<br/>(delivery/http)
    participant U as GalleryUsecase
    participant R as GalleryRepository
    participant DB as Postgres

    C->>H: GET /v1/gallery/top
    H->>U: Top(ctx)
    U->>R: Step 1 — List(ctx, "", 0, top_limit)
    R->>DB: Step 2 — SELECT count(*) FROM gallery_images
    DB-->>R: total
    Note over R,DB: Steps 2 and 3 are independent statements, NOT one snapshot
    R->>DB: Step 3 — SELECT … ORDER BY sort_order, id LIMIT top_limit
    DB-->>R: items (empty table → [])
    R-->>U: items, total
    U-->>H: items, total
    H-->>C: Step 4 — 200 {items (with credit), total}
    opt DB error in Step 2 or 3
        R--xU: wrapped error
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Paged list (`GET /v1/gallery?type=&page=`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as GalleryHandler<br/>(delivery/http)
    participant U as GalleryUsecase
    participant R as GalleryRepository
    participant DB as Postgres

    C->>H: GET /v1/gallery?type=&page=
    H->>H: Step 1 — empty page → 1, else strconv.Atoi
    alt page not an integer
        H--xC: 400 INVALID_PAGE {details.field: page}
    else page parses
        H->>U: List(ctx, type, page)
        U->>U: Step 2 — page outside [1, MaxInt32] or unknown type?
        alt page out of range
            U--xH: ErrInvalidPage
            H-->>C: 400 INVALID_PAGE
        else type not one of the 4 values
            U--xH: ErrInvalidGalleryType
            H-->>C: 400 INVALID_GALLERY_TYPE {details.field: type}
        else valid
            U->>R: List(ctx, type, offset, page_size)
            R->>DB: Step 3 — SELECT count(*) FROM gallery_images [WHERE type]
            DB-->>R: total
            Note over R,DB: Steps 3 and 4 are independent statements, NOT one snapshot
            R->>DB: Step 4 — SELECT … ORDER BY sort_order, id OFFSET LIMIT
            DB-->>R: items (empty past the end)
            R-->>U: items, total
            U->>U: Step 5 — nil items → [], total_pages = ceil(total / page_size)
            U-->>H: GalleryPage
            H-->>C: 200 {items (with credit), page, page_size, total, total_pages}
        end
    end
    opt DB error in Step 3 or 4
        R--xU: wrapped error
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Content load (`cmd/seedgallery`)

```mermaid
sequenceDiagram
    autonumber
    actor O as Operator
    participant M as seedgallery/main.go
    participant P as parseGallery<br/>(seedgallery/parse.go)
    participant R as GalleryRepository
    participant DB as Postgres

    O->>M: go run ./services/appService/cmd/seedgallery
    M->>P: Step 1 — parse + check id, type, title, picture paths, author, license (v2.0)
    alt invalid file
        P--xM: error
        M--xO: log.Fatalf (no DB access)
    else valid
        P->>P: Step 2 — rotate the 4 types, assign sort_order
        P-->>M: []GalleryImage
        M->>R: Upsert(ctx, images)
        rect rgb(235, 240, 255)
            Note over R,DB: Step 3 — ONE transaction for every row
            R->>DB: INSERT … ON CONFLICT (id) DO UPDATE (per image)
            alt success
                R-->>M: nil → COMMIT
                M-->>O: log "seeded N images"
            else a row fails
                R--xM: error → ROLLBACK everything
                M--xO: log.Fatalf
            end
        end
    end
```
