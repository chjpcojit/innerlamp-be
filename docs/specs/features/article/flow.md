# Flow — article

Version: 1.6
Last updated: 2026-10-06
Status: implemented (2026-09-30) — mirror 1:1 của hai Step-by-step Flow trong `design.md`, mỗi Step ở đây trace về đúng Step cùng số.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

`top` và `detail` chỉ một câu SQL nên không có diagram — xem "Request flow" trong `design.md`.

## Sequence — Danh sách phân trang (`GET /v1/articles?page=N`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as ArticleHandler<br/>(delivery/http)
    participant U as ArticleUsecase
    participant R as ArticleRepository
    participant DB as Postgres

    C->>H: GET /v1/articles?page=N
    H->>H: Step 1 — page rỗng → 1, ngược lại strconv.Atoi
    alt page không phải số nguyên
        H--xC: 400 INVALID_PAGE {details.field: page}
    else page hợp lệ về kiểu
        H->>U: List(ctx, page)
        U->>U: Step 2 — page < 1 hoặc > MaxInt32?
        alt ngoài khoảng
            U--xH: ErrInvalidPage
            H-->>C: 400 INVALID_PAGE
        else trong khoảng
            U->>R: List(ctx, offset, page_size)
            R->>DB: Step 3 — SELECT count(*) FROM articles
            DB-->>R: total
            Note over R,DB: Step 3 và Step 4 là hai câu lệnh độc lập, KHÔNG chung transaction/snapshot
            R->>DB: Step 4 — SELECT … ORDER BY sort_order, id OFFSET $1 LIMIT $2
            DB-->>R: items (rỗng nếu vượt cuối)
            R-->>U: items, total
            U->>U: Step 5 — items nil → [], total_pages = ceil(total / page_size)
            U-->>H: ArticlePage
            H-->>C: 200 {items, page, page_size, total, total_pages}
        end
    end
    opt lỗi DB ở Step 3 hoặc Step 4
        R--xU: error bọc %w
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Seed lần đầu khi khởi động

```mermaid
sequenceDiagram
    autonumber
    participant M as cmd/main.go
    participant U as ArticleUsecase
    participant P as ParseArticleSeed<br/>(repository/article_seed.go)
    participant R as ArticleRepository
    participant DB as Postgres

    M->>U: SeedIfEmpty(ctx, load)
    U->>R: Step 1 — Count(ctx)
    R->>DB: SELECT count(*) FROM articles
    alt lỗi (vd. chưa apply migration 0012)
        R--xU: error bọc %w
        U--xM: error
        M->>M: log.Fatalf "failed to seed articles"
    else n > 0
        DB-->>R: n
        R-->>U: n
        U-->>M: 0 (không gọi load, không ghi gì)
    else n = 0
        DB-->>R: 0
        R-->>U: 0
        U->>P: Step 2 — load() với assets.BuddhistArticlesJSON
        alt seed không hợp lệ
            P--xU: error (JSON hỏng / không có bài / id rỗng hoặc trùng / title rỗng / ngày sai)
            U--xM: error
            M->>M: log.Fatalf
        else seed hợp lệ
            P-->>U: []Article (SortOrder = vị trí trong file)
            U->>R: Step 3 — InsertMissing(ctx, articles)
            rect rgb(235, 240, 255)
                Note over R,DB: MỘT transaction cho mọi INSERT
                loop từng bài
                    R->>DB: INSERT … ON CONFLICT (id) DO NOTHING
                end
                alt lỗi DB
                    R--xU: error → rollback, không dòng nào được ghi
                    U--xM: error
                    M->>M: log.Fatalf
                else commit
                    R-->>U: số dòng đã chèn
                end
            end
            U-->>M: số dòng đã chèn
            M->>M: Step 4 — nếu > 0: Info "seeded articles on first run" count
        end
    end
```
