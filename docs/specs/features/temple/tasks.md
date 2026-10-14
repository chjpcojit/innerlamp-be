# Tasks — temple

Version: 1.14
Last updated: 2026-10-11
Status: **Phase 10 (v1.7, địa chỉ có cấu trúc) đã làm và kiểm chứng ngày 2026-10-11**: migration `0032`, domain, repository, DTO, dữ liệu 93 chùa, seed, E2E, openapi, web docs, api-contract. Đã áp lên DB dev. Còn `[ ]`: `erd.svg`, OKF. Trạng thái trước v1.7: implemented (2026-09-30) cho Phase 1–4; Phase 5 chưa làm E2E và OKF; Phase 6 là việc hoãn có chủ đích; Phase 7 (`options`: phân trang, `province`, `?q=` không dấu) và Phase 8 (cột `province`, migration `0024`) và Phase 9 (34 đơn vị mới, migration `0025`) đã làm và kiểm chứng trên Postgres thật; còn thiếu E2E riêng cho `options` và làm mới `erd.svg`/OKF.
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
- [x] `assets/data/temple.json`: 93 chùa, đủ ba trường nội dung biên tập
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
- [ ] Form `create-temple` của backoffice có `province` (chọn từ danh sách chuẩn) — thuộc spec `temple-suggestion`, chưa làm *(v1.7: thay bằng `province_code`/`ward_code` — `temple-suggestion` v3.2 Phase 8)*

## Phase 9 — 34 đơn vị cấp tỉnh (v1.6, US-TMP-07, AC 30–31)

Trace: hàng "Domain / Province", "Seed CLI", "Data", "Migration" trong `design.md`.

- [x] `domain/province.go`: danh sách 34 đơn vị (6 thành phố + 28 tỉnh, không tiền tố) và `IsProvince`; test `TestProvinces_AreThe34UnitsOf2025` (đúng 34, không trùng, không dư khoảng trắng) và `TestIsProvince` (tên mới đúng; tên cũ đã sáp nhập, có tiền tố, viết tắt, sai hoa/thường, rỗng đều sai)
- [x] `cmd/seedtemples/parse.go`: `location.province` phải thuộc danh sách; bỏ kiểm độ dài 100 (danh sách đóng đã bảo đảm); test `TestParseTemples_Province` (nhận bản trim; từ chối thiếu, rỗng, tên cũ `Thừa Thiên Huế`/`Hà Nam`, có tiền tố, `TP.HCM`, tên lạ, sai hoa/thường) và `_RealDataFile`
- [x] `assets/data/temple.json`: đổi `location.province` theo bảng 63 → 34 cho 34 dòng (từ 37 giá trị khác nhau còn 24, mọi giá trị thuộc danh sách)
- [x] `migrations/0025_temple_province_34_units.sql`: `UPDATE … FROM (VALUES …)` đổi 29 tên đã sáp nhập (kèm `Thừa Thiên Huế` → `Huế`) và `^Bà Rịa` → `Hồ Chí Minh`; không đụng `address`. Đã kiểm trên DB tạm với 66 dòng (đủ 63 tên cũ, hai biến thể `Bà Rịa`, một dòng đã ở dạng mới): 0 lệch so với bảng sáp nhập, ra đúng 34 giá trị, áp lần hai không đổi gì
- [x] `migrations/README.md`: hàng `0025`
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: mô tả `Location.province` nêu 34 đơn vị
- [x] Áp `0025` lên DB dev `innerlamp` rồi chạy lại `cmd/seedtemples`: 93 chùa, 0 `province` rỗng, 24 đơn vị khác nhau, cùng kết quả khi chỉ áp migration và khi nạp lại
- [ ] Cập nhật `address` theo đơn vị hành chính mới (huyện/xã) — việc dữ liệu riêng, chưa có nguồn ánh xạ *(v1.7: nguồn ánh xạ đã có — `addr_old_*` + ánh xạ xã; làm ở Phase 10)*
- [ ] Form `create-temple` của backoffice chọn `province` từ danh sách 34 đơn vị (cần bản sao hoặc chỗ dùng chung của `domain/province.go`) — thuộc spec `temple-suggestion` *(v1.7: thay bằng mã — `temple-suggestion` v3.2 Phase 8; không cần bản sao `domain/province.go`)*

## Phase 10 — Địa chỉ có cấu trúc (v1.7, US-TMP-08, AC 32–38)

Trace: hàng "Domain / Address", "Domain / Option", "Domain / Province", "Repository", "Seed CLI", "Data", "Delivery / DTO", "Migration", "API spec", "ERD" trong `design.md`; Step 2 và Step 4–6 của flow seed. Inside-out, test cùng lúc với code (`golang-clean-arch-tdd`). Test repository luôn chạy với `TEST_DB_NAME=innerlamp_test`.

Thứ tự khuyến nghị: migration → domain → repository → script dữ liệu (người duyệt) → seed → DTO/API → E2E → tài liệu. Mục đã `[x]` được kiểm chứng; mục `[ ]` chưa làm.

- [x] `migrations/0032_temple_address_codes.sql`: ba cột nullable + CHECK `chk_temples_address_codes`, idempotent, không đụng cột cũ; `migrations/README.md` (hàng `0032` + ghi chú), `docs/specs/db/erd.mmd` (ba cột và dòng nguồn), `0032` thêm vào danh sách migration của `postgres_registration_repository_test.go`. Đã kiểm trên Postgres 16 tạm (không phải DB dev): áp trên bảng đã có một chùa (qua `0005`, `0011`, `0024`) → dòng cũ giữ nguyên `address`/`province`, ba cột mới `NULL`; áp lần hai không lỗi; `UPDATE` chỉ một cột mã bị CHECK từ chối; đủ bộ ba và trả về cùng `NULL` đều được; đúng một constraint. `go vet ./services/appService/repository/` sạch. Chưa áp lên DB dev `innerlamp`
- [x] `domain/temple.go`: `Temple.ProvinceCode/WardCode/Ward`, `FullAddress()` (AC 33), `TempleOption.ProvinceCode`; test bảng `TestTemple_FullAddress` (đủ ba phần, `address` rỗng hoặc chỉ khoảng trắng, dòng cũ giữ chuỗi đã lưu, có xã thiếu tỉnh) trong `domain/temple_address_test.go`. `mock_temple_repository.go` không cần đổi (chữ ký interface không đổi)
- [x] `repository/postgres_temple_repository.go`: `templeColumns` đọc ba cột bằng `COALESCE(…, '')`, `scanTemple`, `Upsert` (28 cột, `NULLIF($n,'')` nên `""` ↔ `NULL`, upsert ghi đè cả ba cột), `ListOptions` đọc `province_code`. Test `TestTempleRepository_AddressCodes` trên Postgres thật (`TEST_DB_NAME=innerlamp_test`): vòng đi về đủ bộ ba, dòng cũ đọc ra `""`, `FullAddress` đúng ở cả hai, `ListOptions` trả `province_code`, nạp lại về không mã xoá cả ba, CHECK từ chối bộ ba đi lẻ và không ghi gì; các test temple cũ vẫn pass
- [x] **Dữ liệu** — script `scripts/build_temple_addresses.py` (chỉ thư viện chuẩn, `python3 -I`): tách `address` hiện tại của 93 chùa thành phần dưới cấp xã và phần xã/huyện/tỉnh cũ, dò xã cũ qua `old_*` rồi sang `new_wards` bằng `ward_mappings`, xuất `assets/data/temple-address-proposal.json` và bảng duyệt `docs/specs/features/temple/address-review.md`; không ghi vào `assets/data/temple.json`. Đã chạy trên dataset `2026-09-25`: **68 `auto`** (đúng một xã mới, đúng tỉnh), **25 `review`** (11 xã cũ tách thành nhiều xã mới, 8 huyện không còn trong bảng cũ — chủ yếu TP. Huế, 6 xã không có trong bảng cũ; 7 dòng có gợi ý một ứng viên), **20 chùa có `address` dưới cấp xã trống**
- [x] **Duyệt 25 dòng `review`** — anh Đức uỷ quyền ("em tự xử hết"); em tra nguồn qua hai agent (Hà Nội/Hải Phòng; miền Trung + TP.HCM + Bắc Giang), kết quả `address-decisions.json` / `address-decisions.md`: 19 `high`, 5 `medium`, 1 `unverified` (`temple_south_021`). Giả định ghi ở `docs/assumptions/temple/A-005` (`unvalidated`)
- [x] **Áp dụng vào `assets/data/temple.json`** bằng `scripts/apply_temple_addresses.py --write` (đọc proposal + quyết định, kiểm mọi mã trong `new_wards` và đúng tỉnh, sao lưu `temple.json.pre-v1.7.bak`, vá dạng văn bản nên giữ nguyên chú thích `/* */`): 93/93 chùa có `province_code` + `ward_code`, 20 chùa có `address` rỗng (A-004), không còn khoá `province` trong file
- [x] `cmd/seedtemples/parse.go`: `province_code`/`ward_code` bắt buộc không rỗng, `address` bắt buộc có mặt nhưng được rỗng (AC 34, A-004), `location.province` còn là tuỳ chọn để so khớp; bỏ kiểm `IsProvince`. Test `TestParseTemples_AddressFields` (trim, address rỗng, thiếu/rỗng từng field) và `_RealDataFile` (93 chùa đều có mã) pass
- [x] `cmd/seedtemples/resolve.go` + `main.go`: `resolveAddresses` (Step 4) từ chối cả file khi mã tỉnh lạ, xã lệch tỉnh, `location.province` lệch tên, dataset rỗng (thông báo nêu `seedadminunits`), lỗi DB; điền `Province`/`Ward` từ dataset (AC 35–36). 7 test với resolver giả; chạy thật trên DB test `innerlamp_test`: dataset rỗng → thất bại, 0 dòng ghi; sau `seedadminunits` → 93 chùa (93 có `province_code`/`ward_code`/`ward`, 20 `address` rỗng, 0 xã lệch tỉnh); chạy lại → vẫn 93 (idempotent)
- [x] `delivery/http/temple_handler.go`: `locationResponse` thêm `province_code`, `ward_code`, `ward` (null khi chưa nạp lại, dùng lại `nullIfEmpty`) và `full_address` (`Temple.FullAddress()`); `templeOptionResponse` thêm `province_code`; test `TestTempleHandler_LocationAddress` (chùa cũ, chùa có cấu trúc ở `top`, ở chi tiết) và `TestTempleHandler_Options` (có và không có mã) pass
- [x] `go build ./...`, `go vet ./services/appService/...`, `go test -short ./services/appService/...` sạch
- [x] Áp `0032` lên DB dev `innerlamp` rồi `seedtemples` (dataset hành chính đã có sẵn, 34 tỉnh): 93/93 chùa có `province_code`/`ward_code`/`ward`, 20 `address` rỗng, 0 xã lệch tỉnh; `docker exec … psql < migrations/0032_temple_address_codes.sql` (idempotent) rồi `go run ./services/appService/cmd/seedtemples`
- [x] E2E: `services/appService/e2e/temple_e2e_test.go` `TestTemple_E2E` (binary thật + Postgres container, migration `0032` qua glob): `options` (4 field, thứ tự ranking, `province_code` null cho dòng cũ, `q`), chi tiết (đủ trường, address rỗng không bị nối, dòng chưa nạp lại giữ chuỗi đã lưu), `top`, `list`, 404 — pass
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` + `web/docs/src/content/docs/guides/catalogues/temples.md` + `web/docs/src/content/docs/changelog.md`: `Location` và `TempleOption` có trường mới; mô tả `address` đổi nghĩa; ví dụ dùng dữ liệu thật (`temple_north_011` → `Phường Nam Hoa Lư`, `14359`)
- [x] `docs/specs/features/temple/api-contract.md` (tài liệu client iOS) v1.3: thêm `options` và `q`, `Location` mới, đếm lại 93, ảnh là URL
- [x] Gỡ `domain/province.go` và `province_test.go` — `grep` xác nhận chỉ còn test của chính nó dùng `IsProvince`
- [ ] Sinh lại `docs/specs/db/erd.svg` (cần `mmdc`) và OKF `docs/knowledge/features/temple.md` — `il-okf-writer` sau khi code đã kiểm chứng

## Deferred (separate feature, not blocking temple)

- Tạo/sửa/xoá chùa qua backoffice (`admin/requirements.md` trỏ về đây; chưa có spec CRUD — xem `⚠️ Not confirmed` trong `requirements.md`).
- Ảnh chùa: nguồn, nơi lưu (GCS), quyền sử dụng.
- Tìm kiếm, lọc theo `region`/`sect`/`heritage_rank`, tìm gần vị trí.
- Ghi nguồn cho nội dung biên tập (`introduction`, `architecture_heritage`, `visiting_notes`).
