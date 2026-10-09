# Flow — gallery

Version: 1.1
Last updated: 2026-10-07
Status: implemented (2026-10-06) — mirror 1:1 của hai Step-by-step Flow trong `design.md`.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

`top` và `detail` chỉ một câu SQL nên không có diagram — xem "Request flow" trong `design.md`.

## Sequence — Danh sách phân trang (`GET /v1/gallery?type=&page=`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as GalleryHandler<br/>(delivery/http)
    participant U as GalleryUsecase
    participant R as GalleryRepository
    participant DB as Postgres

    C->>H: GET /v1/gallery?type=&page=
    H->>H: Step 1 — page rỗng → 1, ngược lại strconv.Atoi
    alt page không phải số nguyên
        H--xC: 400 INVALID_PAGE {details.field: page}
    else page hợp lệ về kiểu
        H->>U: List(ctx, type, page)
        U->>U: Step 2 — page ngoài [1, MaxInt32] hoặc type lạ?
        alt page ngoài khoảng
            U--xH: ErrInvalidPage
            H-->>C: 400 INVALID_PAGE
        else type không thuộc 4 giá trị
            U--xH: ErrInvalidGalleryType
            H-->>C: 400 INVALID_GALLERY_TYPE {details.field: type}
        else hợp lệ
            U->>R: List(ctx, type, offset, page_size)
            R->>DB: Step 3 — SELECT count(*) FROM gallery_images [WHERE type]
            DB-->>R: total
            Note over R,DB: Step 3 và Step 4 là hai câu độc lập, KHÔNG chung transaction/snapshot
            R->>DB: Step 4 — SELECT … ORDER BY sort_order, id OFFSET LIMIT
            DB-->>R: items (rỗng nếu vượt cuối)
            R-->>U: items, total
            U->>U: Step 5 — items nil → [], total_pages = ceil(total / page_size)
            U-->>H: GalleryPage
            H-->>C: 200 {items, page, page_size, total, total_pages}
        end
    end
    opt lỗi DB ở Step 3 hoặc Step 4
        R--xU: error bọc %w
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Nạp dữ liệu bằng `cmd/seedgallery`

```mermaid
sequenceDiagram
    autonumber
    actor O as Operator
    participant M as seedgallery/main.go
    participant P as parseGallery<br/>(seedgallery/parse.go)
    participant R as GalleryRepository
    participant DB as Postgres

    O->>M: go run ./services/appService/cmd/seedgallery
    M->>P: Step 1 — parse + kiểm id duy nhất, type, title, URL
    alt file lỗi
        P--xM: error
        M--xO: log.Fatalf (không chạm DB)
    else hợp lệ
        P->>P: Step 2 — xen kẽ 4 loại, gán sort_order
        P-->>M: []GalleryImage
        M->>R: Upsert(ctx, images)
        rect rgb(235, 240, 255)
            Note over R,DB: Step 3 — MỘT transaction cho mọi hàng
            R->>DB: INSERT … ON CONFLICT (id) DO UPDATE (lặp từng ảnh)
            alt thành công
                R-->>M: nil → COMMIT
                M-->>O: log "seeded N images"
            else lỗi một hàng
                R--xM: error → ROLLBACK toàn bộ
                M--xO: log.Fatalf
            end
        end
    end
```
