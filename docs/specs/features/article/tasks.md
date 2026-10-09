# Tasks — article

Version: 1.6
Last updated: 2026-10-06
Status: implemented (2026-09-30) cho Phase 1–4; Phase 5 chưa làm E2E; Phase 6 là việc hoãn có chủ đích.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Chỉ `[x]` những gì đã kiểm chứng tồn tại trong code (đã chạy `go build ./...`, `go vet ./services/appService/... ./assets/...`, `go test -short` cho `usecase`, `delivery/http`, `repository` và gọi thật ba route trên server dev ngày 2026-09-30).

## Phase 1 — Domain

- [x] `Article`, `ArticlePage` trong `services/appService/domain/article.go` (không tag)
- [x] `ErrArticleNotFound` trong `domain/article.go`; tái dùng `ErrInvalidPage` (định nghĩa ở `domain/temple.go`)
- [x] Interface `ArticleRepository` (5 method, thứ tự `sort_order ASC, id ASC` nằm trong hợp đồng) trong `domain/article.go`
- [x] `domain/mock_article_repository.go` cho test usecase

## Phase 2 — Usecase

- [x] `usecase/article_usecase.go`: `Top`, `List` (offset, `total_pages`, chặn `page < 1` / `> MaxInt32`), `Get`, `SeedIfEmpty(load)` (chỉ gọi `load` khi bảng rỗng)
- [x] Unit tests: `usecase/article_usecase_test.go` — `Top`, `List`, `Get`, `SeedIfEmpty`, dùng mock tay, không DB thật

## Phase 3 — Repository và dữ liệu

- [x] `migrations/0012_articles.sql` — bảng `articles`, index `idx_articles_order`
- [x] `repository/postgres_article_repository.go`: `ListTop`, `List`, `FindByID`, `Count`, `InsertMissing` (một transaction, `ON CONFLICT DO NOTHING`), scan `NULL` ↔ `nil`/`""`
- [x] `FillMissingImages` (domain interface + mock + `postgres_article_repository.go`) và flag `-fill-images` của `cmd/seedarticles`; test `TestArticleRepository_FillMissingImages` (chạy trên DB scratch, không phải DB dev); đã chạy trên DB dev: 38 dòng được điền
- [x] Chạy `seedarticles -fill-images` trên DB dev để 22 ảnh mới vào dòng đang `NULL` (đã chạy 2026-10-06: DB 60/60)
- [x] `assets/data/buddhist_articles.json`: điền `imageURL` cho 60/60 bài: 38 từ `og:image` của `sourceURL`, 15 từ ảnh trong trang nguồn, 7 từ Wikimedia Commons (A-001); `TestParseArticleSeed_EmbeddedData` kiểm mọi bài có URL `https://`
- [x] `repository/article_seed.go`: `ParseArticleSeed` (kiểm id, title, ngày; `SortOrder` = vị trí)
- [x] `assets/embed.go` nhúng `assets/data/buddhist_articles.json`
- [x] `thumbnail_url`: `migrations/0016_article_thumbnail.sql`; `domain.Article.ThumbnailURL`; repository (cột, scan, `InsertMissing`, `FillMissingImages` điền độc lập từng cột); `ParseArticleSeed` đọc `thumbnailURL`; DTO summary/detail trả `thumbnail_url` (`null` khi thiếu); 60 ảnh trong `assets/images/articles/thumb/`; `thumbnailURL` điền cho 60/60 bài; `make lan-urls` cập nhật host cả trong DB
- [x] Tests cho `thumbnail_url`: `article_handler_test.go` (null và có giá trị), `article_seed_test.go` (mọi bài có URL đúng dạng và file ảnh tồn tại), `postgres_article_repository_test.go` (round-trip và `FillMissingImages_Thumbnail`, đã chạy trên DB scratch `innerlamp_test`; thêm `0016` vào danh sách migration của test)
- [x] Tests: `repository/postgres_article_repository_test.go` (insert/find, không ghi đè, atomic, thứ tự và phân trang; cần Postgres, bỏ qua khi `-short`, và `TRUNCATE` dữ liệu dev — xem Known gaps trong `design.md`; chưa chạy lại ở lượt viết spec này), `repository/article_seed_test.go` (dữ liệu nhúng và validation)

## Phase 4 — Delivery và tài liệu

- [x] `delivery/http/article_handler.go`: `Top`, `List`, `Get` + DTO summary/detail (`published_date`/`image_url` là `null` khi thiếu; mảng không `null`)
- [x] `delivery/http/errors.go`: `ErrArticleNotFound` → `404 ARTICLE_NOT_FOUND`
- [x] `config`: `ArticleConfig`, block `article:` trong `config.dev.yaml`
- [x] `cmd/main.go`: dựng dây, gọi `SeedIfEmpty` lúc khởi động, đăng ký 3 route
- [x] Handler tests: `delivery/http/article_handler_test.go` (+ `mock_article_usecase_test.go`)
- [x] Ghi `0012` vào `migrations/README.md`
- [x] `web/docs/public/openapi.yaml` (nguồn của site tài liệu): 3 route + schema, ví dụ lấy từ dữ liệu seed
- [x] `docs/specs/api/openapi.yaml`: 3 route + schema (đã kiểm có `/v1/articles/top`, `/v1/articles`, `/v1/articles/{id}` và `ARTICLE_NOT_FOUND`)
- [x] Bảng `articles` trong `docs/specs/db/erd.mmd` và `erd.svg` (đã kiểm: render bằng mermaid-cli không lỗi, SVG có bảng)

## Phase 5 — Integration & documentation

- [ ] E2E black-box trong `services/appService/e2e/`: ba route, `INVALID_PAGE`, `ARTICLE_NOT_FOUND`, và seed lần đầu (khởi động hai lần, lần hai không seed lại)
- [x] `go build ./...` và `go vet ./services/appService/... ./assets/...` sạch
- [x] Khái niệm OKF `docs/knowledge/features/article.md` (`status: draft`)

## Deferred (separate feature, not blocking article)

- Tạo/sửa/xoá bài viết qua backoffice hoặc lệnh reseed (`⚠️ Not confirmed` trong `requirements.md`).
- Tìm kiếm, lọc theo `category`/`tag`, bookmark.
- Sao chép ảnh về GCS (thay cho hotlink) — cần chốt bucket, quyền bản quyền ảnh và thông tin xác thực; xem `requirements.md`.
