# Tasks — dharmatalk

Version: 1.1
Last updated: 2026-10-03
Status: implemented (2026-10-03) cho Phase 1–4 và E2E ở Phase 5; còn `[ ]` cho `erd.svg` (chờ render) và khái niệm OKF. Mọi `⚠️ Not confirmed` trong `requirements.md` vẫn mở.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Làm theo TDD (skill `golang-clean-arch-tdd`): viết test trước từng layer. Đọc `article` làm mẫu trước khi viết mỗi file.

## Phase 1 — Domain

- [x] `DharmaTalk`, `DharmaTalkChapter`, `FollowUpMeditation`, `DharmaTalkPage`, `MediaType` trong `services/appService/domain/dharma_talk.go` (không tag)
- [x] `ErrDharmaTalkNotFound`; tái dùng `ErrInvalidPage`
- [x] Interface `DharmaTalkRepository` (`List`, `FindByID`, `Count`, `InsertMissing`) — hợp đồng ghi rõ `List`/`FindByID` lọc `needs_review = false`, `Count` (dùng cho seed) thì không lọc
- [x] `domain/mock_dharma_talk_repository.go`

> Đã chốt khi code: `List` tự đếm số bài hiển thị được (trả `total` cùng trang, như `article`), còn `Count` đếm mọi dòng cho seed — không cần hai method đếm; `design.md` đã khớp.

## Phase 2 — Usecase

- [x] `usecase/dharma_talk_usecase.go`: `List`, `Get`, `SeedIfEmpty(load)`
- [x] Unit tests: `usecase/dharma_talk_usecase_test.go` — `List` (offset, `total_pages`, `page` ngoài khoảng, trang vượt cuối), `Get` (không tồn tại), `SeedIfEmpty` (bảng có dòng toàn `needs_review` thì không seed)

## Phase 3 — Repository và dữ liệu

- [x] `migrations/0014_dharma_talks.sql` — bảng, các `CHECK` (media nhất quán, `duration_minutes > 0`), index `idx_dharma_talks_order`
- [x] `repository/postgres_dharma_talk_repository.go`: `List`, `FindByID`, `CountVisible`/`CountAll`, `InsertMissing`; scan `NULL` ↔ `nil`/`0`/`""`; `JSONB` cho `chapters`/`follow_up_meditation`
- [x] `repository/dharma_talk_seed.go`: `ParseDharmaTalkSeed` — AC 19, ánh xạ `video` → `youtube`, rút `youtube_id`
- [x] `assets/embed.go`: `//go:embed data/dharma_talks.json` → `DharmaTalksJSON`
- [x] Chuẩn hoá `assets/data/dharma_talks.json` theo "Seed JSON" trong `design.md`: không cần sửa file — `sortOrder` thừa bị parser bỏ qua, và `TestParseDharmaTalkSeed_EmbeddedDataIsValid` chứng minh mọi `mediaUrl` YouTube ra được id 11 ký tự và mọi `audio` là URL `http(s)`
- [x] Tests: `repository/dharma_talk_seed_test.go` (dữ liệu nhúng hợp lệ + từng vi phạm AC 19), `repository/postgres_dharma_talk_repository_test.go` (cần Postgres, bỏ qua khi `-short`; **`TRUNCATE` dữ liệu dev** như test repository của `article` — chạy trên DB scratch)

## Phase 4 — Delivery và tài liệu

- [x] `delivery/http/dharma_talk_handler.go`: `List`, `Get` + DTO summary/detail (`null` đúng AC 11–14)
- [x] `delivery/http/errors.go`: `ErrDharmaTalkNotFound` → `404 DHARMA_TALK_NOT_FOUND`
- [x] `config`: `DharmaTalkConfig`, block `dharma_talk:` trong `config.dev.yaml`; test `config_test.go`
- [x] `cmd/main.go`: dựng dây, `SeedIfEmpty`, đăng ký 2 route
- [x] Handler tests: `delivery/http/dharma_talk_handler_test.go` (+ mock usecase)
- [x] Ghi `0014` vào `migrations/README.md`
- [x] `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: 2 route + schema
- [x] Bảng `dharma_talks` trong `docs/specs/db/erd.mmd` và `erd.svg` (đã kiểm: render bằng mermaid-cli không lỗi, SVG có bảng `dharma_talks` và cột `needs_review`)

## Phase 5 — Integration & documentation

- [x] E2E black-box trong `services/appService/e2e/` (skill `golang-e2e-verification`): hai route, `INVALID_PAGE`, `DHARMA_TALK_NOT_FOUND`, bài `needs_review` không lộ ra, seed lần đầu (khởi động hai lần, lần hai không seed lại)
- [x] `go build ./...` và `go vet ./...` sạch; `go test -short ./...` xanh
- [ ] Khái niệm OKF `docs/knowledge/features/dharmatalk.md` (`status: draft`, bởi `il-okf-writer` sau khi code xong)

## Deferred (separate feature, not blocking dharmatalk)

- Lượt nghe (cột + route ghi) — chờ người dùng hoàn tất ý còn cụt.
- `follow_up_meditation` có dữ liệu thật — chờ app có entity bài thiền.
- Điền `chapters` (từ chapter YouTube hoặc ghi tay) và `season`.
- Tìm thêm nguồn audio hợp lệ (Pháp Âm, archive.org, xin phép chùa) để tăng số bài `audio`; bổ sung `duration_minutes`.
- Route `top`, tìm kiếm, lọc, bookmark, backoffice CRUD / lệnh reseed.
- Quyết định giữ hay loại 19 bài `needs_review` (gồm 3 bài Thông Triệt), và bản quyền.
- Liên kết `related_sutra_id` giữa pháp thoại và kinh (chỉ khi người dùng muốn).
