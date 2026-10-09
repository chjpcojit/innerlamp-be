# Design — article

Version: 1.6
Last updated: 2026-10-06
Status: implemented (2026-09-30) — 3 route đọc + seed lần đầu chạy thật qua binary + Postgres dev; có test usecase/handler/seed parser/repository. Thiếu: E2E (`openapi.yaml` và ERD đã có).
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route `GET /v1/articles/*` đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`), sở hữu migration `0012_articles.sql` và `0016_article_thumbnail.sql` (cột `thumbnail_url`), và thêm một package nhỏ `assets/` chỉ để `go:embed` file JSON seed. Cùng hình dạng với danh mục `temple` và `deity` (top + list phân trang + detail, cấu hình `top_limit`/`page_size`), khác ở chỗ seed chạy **tự động một lần khi khởi động** thay vì lệnh `cmd/seedtemples`/`cmd/seeddeities`.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/article.go` | `Article` (mọi cột; `PublishedDate *time.Time`, `ImageURL string` rỗng = chưa có) và `ArticlePage`. Không JSON/db tag. |
| Domain / Errors | `domain/article.go` | `ErrArticleNotFound` (định nghĩa cạnh entity); `ErrInvalidPage` dùng chung với `temple`/`deity` nhưng **định nghĩa ở `domain/temple.go`**, không phải `domain/errors.go`. |
| Domain / Repository interface | `domain/article.go` | `ArticleRepository`: `ListTop`, `List`, `FindByID`, `Count`, `InsertMissing`. Thứ tự `sort_order ASC, id ASC` là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_article_repository.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/article_usecase.go` | `Top`, `List` (tính offset/`total_pages`, chặn `page < 1` hoặc `> MaxInt32`), `Get`, `SeedIfEmpty(load)`. Kích thước trang lấy từ `config.ArticleConfig`. |
| Repository | `services/appService/repository/postgres_article_repository.go` | SQL cho bảng `articles`; `InsertMissing` chạy một transaction với `ON CONFLICT (id) DO NOTHING`; `FillMissingImages` chạy `UPDATE … SET image_url = COALESCE(image_url,$2), thumbnail_url = COALESCE(thumbnail_url,$3) …` trong một transaction (mỗi cột điền độc lập, không ghi đè giá trị đã có); `NULL` ↔ `nil`/`""` cho `published_date`/`image_url`/`thumbnail_url`; `pq.Array` cho `body_paragraphs`/`tags`. |
| Repository / Seed parser | `services/appService/repository/article_seed.go` | `ParseArticleSeed`: giải mã `{"articles":[…]}`, kiểm id/title/ngày, gán `SortOrder` = vị trí trong file. |
| Embedded data | `assets/embed.go`, `assets/data/buddhist_articles.json` | `go:embed` file JSON vào biến `BuddhistArticlesJSON` để binary tự mang dữ liệu seed, không phụ thuộc thư mục làm việc. |
| Delivery / Handler | `services/appService/delivery/http/article_handler.go` | `Top`, `List`, `Get`; parse `page`; DTO summary/detail; `published_date` định dạng `2006-01-02`; `nonNil` cho mảng. |
| Delivery / DTO | `delivery/http/article_handler.go` | `articleSummaryResponse`, `articleDetailResponse` (nhúng summary), `articleTopResponse`, `articleListResponse` — JSON tag chỉ nằm ở đây. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrArticleNotFound` → `404 ARTICLE_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`). |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `ArticleConfig{TopLimit, PageSize}`, block `article:` (dev: 5 và 10). |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; gọi `SeedIfEmpty` ngay sau khi DB kết nối (lỗi thì `log.Fatalf`); đăng ký 3 route. Không dính khối `⚠️ DEV ONLY` nào. |
| Migration | `migrations/0012_articles.sql`, `migrations/0016_article_thumbnail.sql` | Bảng `articles` + index `idx_articles_order (sort_order, id)`; `0016` thêm cột `thumbnail_url TEXT` (NULL). |
| API spec | `docs/specs/api/openapi.yaml` | **Đã có** 3 route (tag Articles) và schema `ArticleSummary`/`ArticleDetail`; `web/docs/public/openapi.yaml` khớp. |

Ghi chú về route: `mux.HandleFunc("GET /v1/articles/top", …)` là segment literal nên `ServeMux` ưu tiên nó hơn `GET /v1/articles/{id}`.

## Request flow

`top` và `detail` chỉ một câu SQL nên giữ dạng ASCII; `list` (đếm rồi lấy trang) và seed có nhiều bước nên nằm ở Step-by-step Flow và `flow.md`.

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

> Xem `flow.md` cho sơ đồ sequence Mermaid của hai flow có nhiều bước bên dưới (mirror 1:1).

## Step-by-step Flow — Danh sách phân trang (`GET /v1/articles?page=N`)

Hai lần truy vấn nối tiếp (đếm, rồi lấy trang) nên flow này có Step-by-step riêng.

```
Step 1 — Đọc và kiểm tra `page` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query().Get("page"); rỗng → page = 1, ngược lại strconv.Atoi
    - Success → Step 2
    - Failure (không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra khoảng của `page` (usecase)
- Trigger: handler gọi ArticleUsecase.List(ctx, page)
- Pre-condition: page là số nguyên
- Call: ArticleUsecase.List — page < 1 hoặc page > math.MaxInt32
    - Success → Step 3 (offset = (page-1) * page_size)
    - Failure → domain.ErrInvalidPage → 400 INVALID_PAGE

Step 3 — Đếm tổng số bài (repository)
- Trigger: usecase gọi ArticleRepository.List(offset, limit)
- Pre-condition: offset/limit hợp lệ
- Call: ArticleRepository.Count → SELECT count(*) FROM articles
    - Success → total, sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT <cột> FROM articles ORDER BY sort_order ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (rỗng nếu offset vượt cuối) và total, sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; handler map sang DTO summary
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Seed lần đầu khi khởi động

Bước đếm, bước đọc file và bước ghi là các tác động nối tiếp, trong đó bước ghi là side effect bền vững.

```
Step 1 — Đếm số bài đang có (usecase → repository)
- Trigger: cmd/main.go gọi ArticleUsecase.SeedIfEmpty(ctx, load) sau khi kết nối DB
- Pre-condition: DB kết nối được; migration 0012 đã apply
- Call: ArticleRepository.Count → SELECT count(*) FROM articles
    - Success (n = 0) → Step 2
    - Success (n > 0) → dừng: trả 0 bài đã chèn, KHÔNG gọi load, KHÔNG ghi gì (restart không đụng dữ liệu)
    - Failure (vd. bảng chưa tồn tại) → error bọc %w → main log.Fatalf "failed to seed articles"

Step 2 — Nạp và kiểm tra dữ liệu seed (repository)
- Trigger: n = 0
- Pre-condition: —
- Call: load() = repository.ParseArticleSeed(assets.BuddhistArticlesJSON)
    - Success → danh sách domain.Article, SortOrder = vị trí trong file, sang Step 3
    - Failure (JSON hỏng, không có bài, id rỗng/trùng, title rỗng, publishedDate sai định dạng) → error → log.Fatalf, chưa ghi gì

Step 3 — Chèn các bài còn thiếu (repository)
- Trigger: Step 2 xong
- Pre-condition: seed hợp lệ
- Call: ArticleRepository.InsertMissing → INSERT … ON CONFLICT (id) DO NOTHING cho từng bài
    - Success → số dòng thực sự được chèn (60 ở lần đầu), sang Step 4
    - Failure (lỗi DB) → rollback toàn bộ → error → log.Fatalf
    - Transaction boundary: mọi INSERT của Step 3 chung MỘT transaction — hoặc đủ hết hoặc không dòng nào

Step 4 — Ghi log (cmd/main.go)
- Trigger: SeedIfEmpty trả về số bài đã chèn
- Pre-condition: —
- Call: nếu số > 0 → logger.Info("seeded articles on first run", "count", n)
    - Success → tiếp tục khởi động, đăng ký route
    - Failure → không có nhánh lỗi
```

## Deliberately stubbed for now (if applicable)

Không có gì bị stub: mọi method của `ArticleRepository` đều có SQL thật và đã chạy trên Postgres dev.

## Known gaps carried by this feature

- **Không có E2E**: chưa có test chạy binary thật với Postgres cho ba route và cho seed lần đầu (`services/appService/e2e/` không có gì về article). Việc "đã chạy thật" là chạy tay trên server dev.
- **Test repository cần Postgres và xoá dữ liệu dev**: `postgres_article_repository_test.go` gọi `connectTestDB`, bỏ qua khi `-short`; helper đó chạy `TRUNCATE` trên nhiều bảng, trong đó có `articles`, `temples`, `deities` và `users` của DB `innerlamp` (mặc định `localhost:5432`). Chạy xong bảng `articles` của DB dev trống cho tới lần khởi động service kế tiếp (lúc đó seed lại), còn `temples`/`deities` phải seed lại bằng `cmd/seedtemples`/`cmd/seeddeities`. `go test -short` chỉ chạy usecase, handler và seed parser.
- **Tài liệu schema/API đã đủ**: ERD (`erd.mmd`, `erd.svg`) có bảng `articles`; `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml` đã có ba route.
- **Sửa file JSON không cập nhật DB có sẵn**: seed chỉ chạy khi bảng rỗng. `cmd/seedarticles` chèn bài mới (`InsertMissing`), và với `-fill-images` gọi `FillMissingImages` để đặt `image_url` cho dòng đang `NULL` (không ghi đè ảnh đã có, không đụng cột khác). Các cột nội dung khác vẫn chỉ đổi được bằng SQL tay; chưa có reseed đầy đủ hay backoffice CRUD.
- **Bảng bị xoá sạch thì được nạp lại**: điều kiện "không hồi sinh bài đã xoá" chỉ đúng khi còn ít nhất một dòng; xoá toàn bộ bảng rồi khởi động lại sẽ nạp lại đủ 60 bài.
- **Đếm và lấy trang không cùng snapshot**: nếu có dòng đổi giữa Step 3 và Step 4, `total` có thể lệch `items`. Dữ liệu hiện chỉ đổi bằng SQL tay nên rủi ro thấp.
- **Hai instance khởi động cùng lúc**: cả hai có thể cùng thấy bảng rỗng; `ON CONFLICT DO NOTHING` làm cho instance đến sau chèn 0 dòng thay vì lỗi, nên an toàn, nhưng cả hai đều parse seed.
- **`id = "top"` bị route literal che** (xem AC10).
- **`thumbnail_url` là ảnh vuông 360x360 WebP (~15 KB) do app phục vụ** tại `/static/images/articles/thumb/<id>.webp` (route dev, xem `delivery/http/static_images.go`), cắt giữa từ ảnh gốc và lệch lên trên 10%; URL trong dữ liệu seed mang IP LAN của máy dev nên đổi bằng `make lan-urls`. Đây là bản sao phái sinh của ảnh có bản quyền của bên khác, chỉ dùng cho dev (A-002). DB đã seed trước migration `0016` nhận thumbnail qua `seedarticles -fill-images`.
- **`image_url` là hotlink**: 60/60 bài trong dữ liệu seed có ảnh: 53 bài trỏ tới server nguồn (`og:image` hoặc ảnh trong bài), 7 bài trỏ tới `thumb.wikimedia.org` (ảnh minh hoạ chủ đề, xem A-001). Ảnh có thể hỏng nếu nguồn đổi URL hoặc chặn hotlink; bản sao trên GCS chưa có. DB đã seed trước đó chỉ nhận 22 ảnh mới qua `seedarticles -fill-images` (đã chạy trên DB dev ngày 2026-10-06: DB đủ 60/60).
