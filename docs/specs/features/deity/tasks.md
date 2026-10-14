# Tasks — deity

Version: 2.3
Last updated: 2026-10-12
Status: implemented (2026-09-30) cho Phase 1–2 và 4; Phase 3 còn thiếu ảnh dùng được và seed ngoài dev; Phase 5 chưa làm E2E; v2.0 đổi route phân trang sang `/v1/deities` (code + unit test đã xong, repository test và server chưa chạy lại); Phase 6 là việc hoãn có chủ đích.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Chỉ `[x]` những gì đã kiểm chứng tồn tại trong code (đã đọc file, chạy `go build ./...`, `go vet ./services/appService/... ./assets/...` và `go test -short ./services/appService/...` ngày 2026-09-30). Test repository cần Postgres và xoá dữ liệu dev nên **chưa chạy** ở lượt này; các route chưa được gọi lại trên server ở lượt này.

## Phase 1 — Domain

- [x] `Deity`, `MemorialDay`, `Iconography`, `Mantras`, `DeityPage` và hằng `DeityBuddha`/`DeityBodhisattva` trong `services/appService/domain/deity.go` (không tag)
- [x] `ErrDeityNotFound` trong `domain/deity.go`; tái dùng `ErrInvalidPage` (định nghĩa ở `domain/temple.go`)
- [x] Interface `DeityRepository` (4 method, thứ tự `sort_order ASC, id ASC` nằm trong hợp đồng) trong `domain/deity.go`
- [x] `domain/mock_deity_repository.go` cho test usecase

## Phase 2 — Usecase

- [x] `usecase/deity_usecase.go`: `Top`, `ListDeities` (mọi loại, offset, `total_pages`, chặn `page < 1` / `> MaxInt32`, `items` nil → `[]`), `Get`
- [x] Unit tests: `usecase/deity_usecase_test.go` — `TestDeityUsecase_Top`, `_ListDeities`, `_Get`, dùng mock tay, không DB thật

## Phase 3 — Repository và dữ liệu

- [x] `migrations/0007_deities.sql` — bảng `deities`, `CHECK` trên `type`, JSONB + `TEXT[]`, hai index
- [x] `repository/postgres_deity_repository.go`: `ListTop`, `List`, `FindByID`, `Upsert` (một transaction, `ON CONFLICT DO UPDATE`), `scanDeity(ies)` với JSONB và `pq.Array`
- [x] `cmd/seeddeities/main.go` + `parse.go`: đọc file, kiểm id/type/`name_vi`, `SortOrder` = vị trí, kết nối bằng config dev, gọi `Upsert`
- [x] `assets/data/phat_botat.json`: 18 mục (6 Phật, 12 Bồ tát)
- [x] Tests: `cmd/seeddeities/parse_test.go` (`TestParseDeities_RealDataFile`, `_Validation`, `_OrderFollowsFile`) chạy và pass ở `-short`; `repository/postgres_deity_repository_test.go` (upsert/find, not found, atomic, ordering, `List` paging (gộp Phật + Bồ tát)) có mặt nhưng cần Postgres, bỏ qua khi `-short`, và `TRUNCATE` dữ liệu dev — chưa chạy lại ở lượt viết spec này
- [ ] Thay `thumbnail_url`/`image_url` giữ chỗ bằng URL dùng được rồi seed lại — chặn bởi `⚠️ Not confirmed` về nguồn ảnh trong `requirements.md`
- [ ] Đưa seed ra khỏi config dev để dùng được ở staging/production, hoặc chốt cách nạp khác — chặn bởi `⚠️ Not confirmed` về quy trình nạp trong `requirements.md`

## Phase 4 — Delivery và tài liệu

- [x] `delivery/http/deity_handler.go`: `Top`, `ListDeities`, `Get` + DTO summary/detail (mảng không `null`; URL ảnh là chuỗi)
- [x] `delivery/http/errors.go`: `ErrDeityNotFound` → `404 DEITY_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE`
- [x] `config`: `DeityConfig`, block `deity:` trong `config.dev.yaml`
- [x] `cmd/main.go`: dựng dây và đăng ký 3 route (không gọi seed)
- [x] Handler tests: `delivery/http/deity_handler_test.go` (`TestDeityHandler_Top`, `_Top_ServerError`, `_ListDeities`, `_Get`) + `mock_deity_usecase_test.go`, pass ở `-short`
- [x] Ghi `0007` vào `migrations/README.md`
- [x] `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: 3 route + schema (v2.0: route phân trang đổi thành `/v1/deities`; cần kiểm lại `/v1/deities/top`, `/v1/deities`, `/v1/deities/{id}` và `DEITY_NOT_FOUND`)
- [x] `api-contract.md` cho client iOS (tài liệu có sẵn, giữ nguyên)
- [x] Bảng `deities` trong `docs/specs/db/erd.mmd` và `erd.svg` (đã kiểm: render bằng mermaid-cli không lỗi, SVG có bảng)

## Phase 5 — Integration & documentation

- [ ] E2E black-box trong `services/appService/e2e/`: ba route, danh sách phân trang chứa cả Phật lẫn Bồ tát, `INVALID_PAGE`, `DEITY_NOT_FOUND`, DB chưa seed trả `items: []`
- [x] `go build ./...` và `go vet ./services/appService/... ./assets/...` sạch
- [ ] Khái niệm OKF `docs/knowledge/features/deity.md` — do `bdd-okf-docs` tạo sau khi có E2E; chưa có file
- [ ] Rà `api-contract.md` cho khớp spec (hai nơi cùng mô tả hợp đồng; chọn giữ cả hai hay gộp về `openapi.yaml`)

## Phase 7 — Lọc theo `type` (v2.2, AC 20–22; hoàn tất 2026-10-12)

Mỗi mục trace về hàng "(v2.2)" trong "Layers & components" hoặc Step 1–4 của flow phân trang trong `design.md`. Đã kiểm chứng 2026-10-12: `go build ./...`, `go vet ./services/appService/...`, `go test -short ./services/appService/...` sạch; `TEST_DB_NAME=innerlamp_test go test -run TestDeityRepository ./services/appService/repository/` pass; `go test -run TestDeity_TypeFilter_E2E ./services/appService/e2e/` pass (testcontainers).

- [x] Domain: `ErrInvalidDeityType` và `IsValidDeityType` trong `domain/deity.go`; đổi `DeityRepository.List` thêm `deityType` và cập nhật `domain/mock_deity_repository.go`
- [x] Usecase: `ListDeities(ctx, deityType, page)` — kiểm `page` trước rồi `type`; test: type hợp lệ ×3, rỗng = mọi loại, `buddha` (thường) → lỗi, `X` → lỗi, `page=0` + type lạ → `ErrInvalidPage`
- [x] Repository: `List` thêm `WHERE type = $n` cho cả đếm và lấy trang; test (cần Postgres, `TEST_DB_NAME=innerlamp_test`): lọc từng loại, `total` chỉ đếm loại đó, `page` vượt cuối của loại đó → `items: []`
- [x] Handler: đọc `type`, chuyển cho usecase; `delivery/http/errors.go` map `ErrInvalidDeityType` → `400 INVALID_DEITY_TYPE` (`details.field = "type"`); test handler cho 200 có/không `type` và 400
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: thêm parameter `type` (enum 3 giá trị) và response `400 INVALID_DEITY_TYPE` cho `/v1/deities`; sửa mô tả "không lọc"
- [x] `api-contract.md` (client iOS): cập nhật hàng `GET /v1/deities` và mục lỗi
- [x] E2E (gộp vào mục E2E ở Phase 5): `?type=BUDDHA`, `?type=BODHISATTVA`, `?type=DEVA`, `?type=buddha` → 400, `?type=BUDDHA&page=2` → `items: []`

## Deferred (separate feature, not blocking deity)

- Tạo/sửa/xoá mục qua backoffice (`admin/requirements.md` trỏ về đây; chưa có spec CRUD).
- Rà soát nội dung tôn giáo (ngày vía, mantra, đại nguyện) theo một quy trình duyệt.
- Tìm kiếm theo từ khoá, yêu thích; lọc nhiều `type` cùng lúc.
