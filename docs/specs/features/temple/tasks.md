# Tasks — temple

Version: 1.6
Last updated: 2026-10-10
Status: implemented (2026-09-30) cho Phase 1–4; Phase 5 chưa làm E2E và OKF; Phase 6 là việc hoãn có chủ đích; Phase 7 (`options`: phân trang, `province`, `?q=` không dấu) và Phase 8 (cột `province`, migration `0024`) và Phase 9 (34 đơn vị mới, migration `0025`) đã làm và kiểm chứng trên Postgres thật; còn thiếu E2E riêng cho `options` và làm mới `erd.svg`/OKF.
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

## Phase 7 — `GET /v1/temples/options?page=N&q=` (v1.5, US-TMP-06, AC 19–25)

Trace: hàng "Domain / Option", "Repository", "Usecase", "Delivery / Handler", "Delivery / DTO", "Delivery / Error map", "Config", "Wiring", "API spec" trong `design.md`. Inside-out, test cùng lúc với code.

- [x] `domain/temple.go`: `TempleOption{ID, Name, Province}`, `TempleOptionPage`, `ErrInvalidTempleQuery`, và `ListOptions(ctx, query, offset, limit) ([]TempleOption, int, error)` trong `TempleRepository` (thứ tự ranking nằm trong hợp đồng)
- [x] `domain/mock_temple_repository.go`: `ListOptions(ctx, query, offset, limit)` (mock viết tay)
- [x] `config/config.go` + `config.dev.yaml`: `TempleConfig.OptionsPageSize` (`options_page_size: 100`), `TempleConfig.Validate` (cả ba khoá `>= 1`); test `TestTempleConfig_Validate`; `cmd/main.go` gọi `cfg.Temple.Validate()` lúc khởi động
- [x] `usecase/temple_usecase.go`: `Options(ctx, query *string, page)` — quy tắc `page`/`total_pages` như `List`, kích thước `cfg.OptionsPageSize`; `q` có mặt thì trim, rỗng hoặc > 100 ký tự → `ErrInvalidTempleQuery`; test `TestTempleUsecase_Options` và `_Options_Query`
- [x] `repository/postgres_temple_repository.go`: `ListOptions` — `count(*)` rồi `SELECT id, name, province … OFFSET/LIMIT`; khi có `q`: `WHERE unaccent(lower(name)) LIKE unaccent(lower($1)) ESCAPE '\'` với `%`, `_`, `\` được escape. Test `TestTempleRepository_ListOptions` (3 field, `province` là cột đã lưu chứ không suy từ `address`, thứ tự ranking, phân trang, không dấu, `đ`↔`d`, ký tự đại diện LIKE, không khớp, bảng rỗng) **đã chạy và pass** trên một DB Postgres tạm riêng (`TEST_DB_NAME`), không phải DB dev
- [x] `delivery/http/temple_handler.go`: `Options` (dùng chung bước parse `page` với `List`; `q` truyền theo "có mặt/không", `?q=` rỗng là có mặt), `templeOptionResponse{id, name, province}`, `templeOptionsResponse`; `delivery/http/errors.go`: `ErrInvalidTempleQuery` → `400 INVALID_FILTER` (`details.field = "q"`); test `TestTempleHandler_Options` và `_Options_Query`
- [x] `cmd/main.go`: route `GET /v1/temples/options` đã đăng ký (không bọc `webAPIKey`/JWT)
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: route có `page` và `q`, schema `TempleOption{id, name, province}`, `TempleOptionsEnvelope`, `400 INVALID_PAGE`/`INVALID_FILTER`; mô tả `ordination_temple_id` trỏ `options`
- [x] Chạy thật: build binary, nạp DB tạm bằng migration `0001`–`0024` + `cmd/seeddeities` + `cmd/seedtemples`, gọi route qua HTTP — `q=bich dong` và `q=BÍCH ĐỘNG` cùng một kết quả, `q=` → `400 INVALID_FILTER`, `page=0` → `400 INVALID_PAGE`, `page=2` → `items: []`
- [ ] E2E: thêm ca `options` vào bộ E2E ở Phase 5 (khi bộ đó được làm)

## Phase 8 — Cột `temples.province` (v1.5, US-TMP-07, AC 26–29)

Trace: hàng "Domain / Entity", "Seed CLI", "Data", "Repository", "Delivery / DTO", "Migration", "API spec", "ERD" trong `design.md`.

- [x] `migrations/0024_temple_province.sql`: `ADD COLUMN IF NOT EXISTS province VARCHAR(100) NOT NULL DEFAULT ''` + `UPDATE … WHERE province = ''` backfill từ `address`. Đã kiểm trên DB tạm: 93/93 dòng bằng đúng `location.province` trong JSON, mười ca biên (`TP.HCM`, `TP. Cần Thơ`, `Thành phố Huế`, `Tỉnh` đứng một mình, dấu phẩy cuối, chữ thường, `""`) cho kết quả đúng, áp lần hai không đổi gì
- [x] `domain/temple.go`: `Temple.Province`; hàm suy ra `province` từ `address` (có ở bản v1.4) **đã bỏ** — không còn đường chạy nào đoán
- [x] `repository/postgres_temple_repository.go`: `templeColumns`, `scanTemple`, `Upsert` (25 cột, ghi đè `province`); test `TestTempleRepository_UpsertAndFind` (vòng đi về nguyên vẹn) pass trên DB tạm
- [x] `assets/data/temple.json`: `location.province` cho cả 93 chùa (sinh một lần từ `address`, đối chiếu bằng cách chạy backfill SQL; từ giờ là dữ liệu tường minh)
- [x] `cmd/seedtemples/parse.go`: `location.province` bắt buộc, trim, ≤ 100 ký tự (`maxProvinceRunes`, cố định theo schema); test `TestParseTemples_Province` (trim, thiếu, rỗng, 101 ký tự bị từ chối, đúng 100 được nhận) và `_RealDataFile`
- [x] `delivery/http/temple_handler.go`: `locationResponse.Province` → `location.province` ở `top`, danh sách, chi tiết; test cập nhật
- [x] `repository/postgres_registration_repository_test.go`: thêm `0024` vào danh sách migration test
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: `Location.province` (bắt buộc)
- [x] `docs/specs/db/erd.mmd`: cột `province` của `temples`; `migrations/README.md`: hàng `0024`
- [x] Áp `0024` lên DB dev `innerlamp` và chạy lại `cmd/seedtemples`: 93 chùa, 0 dòng `province` rỗng, 37 giá trị khác nhau
- [ ] `docs/specs/db/erd.svg`: cần render lại từ `erd.mmd` (môi trường này không có `mermaid-cli`)
- [ ] Form `create-temple` của backoffice có `province` (chọn từ danh sách chuẩn) — thuộc spec `temple-suggestion`, chưa làm

## Phase 9 — 34 đơn vị cấp tỉnh (v1.6, US-TMP-07, AC 30–31)

Trace: hàng "Domain / Province", "Seed CLI", "Data", "Migration" trong `design.md`.

- [x] `domain/province.go`: danh sách 34 đơn vị (6 thành phố + 28 tỉnh, không tiền tố) và `IsProvince`; test `TestProvinces_AreThe34UnitsOf2025` (đúng 34, không trùng, không dư khoảng trắng) và `TestIsProvince` (tên mới đúng; tên cũ đã sáp nhập, có tiền tố, viết tắt, sai hoa/thường, rỗng đều sai)
- [x] `cmd/seedtemples/parse.go`: `location.province` phải thuộc danh sách; bỏ kiểm độ dài 100 (danh sách đóng đã bảo đảm); test `TestParseTemples_Province` (nhận bản trim; từ chối thiếu, rỗng, tên cũ `Thừa Thiên Huế`/`Hà Nam`, có tiền tố, `TP.HCM`, tên lạ, sai hoa/thường) và `_RealDataFile`
- [x] `assets/data/temple.json`: đổi `location.province` theo bảng 63 → 34 cho 34 dòng (từ 37 giá trị khác nhau còn 24, mọi giá trị thuộc danh sách)
- [x] `migrations/0025_temple_province_34_units.sql`: `UPDATE … FROM (VALUES …)` đổi 29 tên đã sáp nhập (kèm `Thừa Thiên Huế` → `Huế`) và `^Bà Rịa` → `Hồ Chí Minh`; không đụng `address`. Đã kiểm trên DB tạm với 66 dòng (đủ 63 tên cũ, hai biến thể `Bà Rịa`, một dòng đã ở dạng mới): 0 lệch so với bảng sáp nhập, ra đúng 34 giá trị, áp lần hai không đổi gì
- [x] `migrations/README.md`: hàng `0025`
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: mô tả `Location.province` nêu 34 đơn vị
- [x] Áp `0025` lên DB dev `innerlamp` rồi chạy lại `cmd/seedtemples`: 93 chùa, 0 `province` rỗng, 24 đơn vị khác nhau, cùng kết quả khi chỉ áp migration và khi nạp lại
- [ ] Cập nhật `address` theo đơn vị hành chính mới (huyện/xã) — việc dữ liệu riêng, chưa có nguồn ánh xạ
- [ ] Form `create-temple` của backoffice chọn `province` từ danh sách 34 đơn vị (cần bản sao hoặc chỗ dùng chung của `domain/province.go`) — thuộc spec `temple-suggestion`

## Deferred (separate feature, not blocking temple)

- Tạo/sửa/xoá chùa qua backoffice (`admin/requirements.md` trỏ về đây; chưa có spec CRUD — xem `⚠️ Not confirmed` trong `requirements.md`).
- Ảnh chùa: nguồn, nơi lưu (GCS), quyền sử dụng.
- Tìm kiếm, lọc theo `region`/`sect`/`heritage_rank`, tìm gần vị trí.
- Ghi nguồn cho nội dung biên tập (`introduction`, `architecture_heritage`, `visiting_notes`).
