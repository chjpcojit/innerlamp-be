# Tasks — temple

Version: 1.2
Last updated: 2026-10-03
Status: implemented (2026-09-30) cho Phase 1–4; Phase 5 chưa làm E2E và OKF; Phase 6 là việc hoãn có chủ đích.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Chỉ `[x]` những gì đã kiểm chứng tồn tại trong code (đã đọc file, chạy `go build ./...`, `go vet ./services/appService/... ./assets/...` và `go test -short ./services/appService/...` ngày 2026-09-30). Test repository cần Postgres và xoá dữ liệu dev nên **chưa chạy** ở lượt này; các route chưa được gọi lại trên server ở lượt này.

## Phase 1 — Domain

- [x] `Temple`, `TempleAmenities`, `TemplePage` trong `services/appService/domain/temple.go` (không tag)
- [x] Bảng `heritagePriority` + `HeritagePriority()` trong `domain/temple.go`
- [x] `ErrTempleNotFound`, `ErrInvalidPage` trong `domain/temple.go` (`ErrInvalidPage` dùng chung với `deity`/`article`)
- [x] Interface `TempleRepository` (4 method, thứ tự `heritage_priority, established_year, id` nằm trong hợp đồng) trong `domain/temple.go`
- [x] `domain/mock_temple_repository.go` cho test usecase

## Phase 2 — Usecase

- [x] `usecase/temple_usecase.go`: `Top`, `List` (offset, `total_pages`, chặn `page < 1` / `> MaxInt32`, `items` nil → `[]`), `Get`
- [x] Unit tests: `usecase/temple_usecase_test.go` — `TestTempleUsecase_Top`, `_List`, `_Get`, dùng mock tay, không DB thật

## Phase 3 — Repository và dữ liệu

- [x] `migrations/0005_temples.sql` — bảng `temples`, ba `CHECK`, index `idx_temples_ranking`
- [x] `migrations/0011_temple_content.sql` — `introduction`, `architecture_heritage`, `visiting_notes`
- [x] `repository/postgres_temple_repository.go`: `ListTop`, `List`, `FindByID`, `Upsert` (một transaction, `ON CONFLICT DO UPDATE`), `scanTemple(s)` với `pq.Array`
- [x] `cmd/seedtemples/main.go` + `parse.go`: đọc file, bỏ chú thích `/* */`, kiểm id/vùng/hạng, kết nối bằng config dev, gọi `Upsert`
- [x] `assets/data/temple.json`: 179 chùa, đủ ba trường nội dung biên tập
- [x] Tests: `cmd/seedtemples/parse_test.go` (`TestParseTemples`, `_Rejects`, `_RealDataFile`) chạy và pass ở `-short`; `repository/postgres_temple_repository_test.go` (upsert/find, not found, atomic, ordering, paging) có mặt nhưng cần Postgres, bỏ qua khi `-short`, và `TRUNCATE` dữ liệu dev — chưa chạy lại ở lượt viết spec này
- [ ] Đưa seed ra khỏi config dev để dùng được ở staging/production, hoặc chốt cách nạp khác — chặn bởi `⚠️ Not confirmed` về quy trình nạp trong `requirements.md`

## Phase 4 — Delivery và tài liệu

- [x] `delivery/http/temple_handler.go`: `Top`, `List`, `Get` + DTO summary/detail (`image_url`/`thumbnail_url` là `null` khi thiếu; `aliases`/`visiting_notes` không `null`)
- [x] `delivery/http/errors.go`: `ErrTempleNotFound` → `404 TEMPLE_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE`
- [x] `config`: `TempleConfig`, block `temple:` trong `config.dev.yaml`
- [x] `cmd/main.go`: dựng dây và đăng ký 3 route (không gọi seed)
- [x] Handler tests: `delivery/http/temple_handler_test.go` (`TestTempleHandler_Top`, `_List`, `_Get`) + `mock_temple_usecase_test.go`, pass ở `-short`
- [x] Ghi `0005` và `0011` vào `migrations/README.md`
- [x] `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: 3 route + schema (đã kiểm có `/v1/temples/top`, `/v1/temples`, `/v1/temples/{id}` và `TEMPLE_NOT_FOUND`)
- [x] `api-contract.md` cho client iOS (tài liệu có sẵn, giữ nguyên)
- [x] Bảng `temples` trong `docs/specs/db/erd.mmd` và `erd.svg` (đã kiểm: render bằng mermaid-cli không lỗi, SVG có bảng)

## Phase 5 — Integration & documentation

- [ ] E2E black-box trong `services/appService/e2e/`: ba route, `INVALID_PAGE`, `TEMPLE_NOT_FOUND`, DB chưa seed trả `items: []`
- [x] `go build ./...` và `go vet ./services/appService/... ./assets/...` sạch
- [ ] Khái niệm OKF `docs/knowledge/features/temple.md` — do `bdd-okf-docs` tạo sau khi có E2E; chưa có file
- [ ] Rà `api-contract.md` cho khớp spec (hai nơi cùng mô tả hợp đồng; chọn giữ cả hai hay gộp về `openapi.yaml`)

## Deferred (separate feature, not blocking temple)

- Tạo/sửa/xoá chùa qua backoffice (`admin/requirements.md` trỏ về đây; chưa có spec CRUD — xem `⚠️ Not confirmed` trong `requirements.md`).
- Ảnh chùa: nguồn, nơi lưu (GCS), quyền sử dụng.
- Tìm kiếm, lọc theo `region`/`sect`/`heritage_rank`, tìm gần vị trí.
- Ghi nguồn cho nội dung biên tập (`introduction`, `architecture_heritage`, `visiting_notes`).
