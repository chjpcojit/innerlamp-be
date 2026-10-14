# Tasks — administrative-units

Version: 1.8
Last updated: 2026-10-11
Status: Phase 1–7 xong. Phase 7 (tìm xã theo tên `?q=`) thêm sau quyết định 2026-10-11 "app chỉ cho chọn hệ mới". 22 test mới ở các lớp + 2 bài E2E trên binary thật và Postgres testcontainers; AC 1–26 (trừ AC 13 đã hoãn) đều có test chứng minh — xem `test-coverage-report.md`. Còn lại duy nhất việc của skill khác: concept OKF `docs/knowledge/features/administrative-units.md` (`il-okf-writer`, chỉ viết khi được yêu cầu). Phần hoãn có chủ đích nằm ở mục Deferred.
Module: `administrative-units`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong `design.md` §Layers & components hoặc một bước trong §Step-by-step Flow.

## Phase 1 — Domain

- [x] Thêm entity `CurrentProvince`, `CurrentWard`, `LegacyProvince`, `LegacyDistrict`, `LegacyWard` (có `SortOrder`), `WardMapping`, `DistrictMapping`, `CurrentWardCandidate`, `LegacyWardCandidate`, `Conversion[T]` (có `Unique()`) vào `services/appService/domain/administrative_unit.go` (chỉ trường nghiệp vụ, không JSON tag) — tên `Current*`/`Legacy*` thay cho `New*`/`Old*` của bản v1.0 vì `NewX` đọc như hàm khởi tạo trong Go; route và JSON vẫn dùng `new`/`old`
- [x] Khai báo `ErrAdministrativeUnitNotFound` cùng file
- [x] Định nghĩa `AdministrativeUnitRepository` (9 hàm đọc + `ReplaceDataset`) ở `domain/administrative_unit.go`
- [x] Thêm `Dataset`, `Dataset.Counts()` và `(Dataset).Verify() []IntegrityIssue` với bảy quy tắc toàn vẹn (AC 21: trùng mã, cha không tồn tại, xã khác tỉnh với huyện, xã cũ thiếu mapping, xã mới không là đích, mapping trỏ vào đơn vị không có, tỉnh cũ tách sang nhiều tỉnh mới); test bảng cho từng quy tắc trong `domain/administrative_unit_test.go` (12 ca + thứ tự báo cáo + không đổi dữ liệu)
- [x] Viết `domain/mock_administrative_unit_repository.go` cho test usecase (có kiểm `var _ AdministrativeUnitRepository = (*Mock…)(nil)`)

## Phase 2 — Usecase

- [x] Tạo `usecase/administrative_unit_usecase.go`: chín method (một mỗi route), `NewAdministrativeUnitUsecase(repo)` không cần config; trim `code`, mã trống → `ErrAdministrativeUnitNotFound` không gọi repository; danh sách nil → `[]`; bọc kết quả chuyển đổi vào `Conversion[T]` và không tự chọn ứng viên (AC 11)
- [x] Bọc lỗi repository bằng `%w` để `errors.Is(err, domain.ErrAdministrativeUnitNotFound)` còn đúng và lỗi khác không bị nhận nhầm là not-found
- [x] Unit test `usecase/administrative_unit_usecase_test.go`, table-driven, `testify/assert` + `testify/require`, dùng mock tay — không DB; bảy method nhận `code` cùng chạy một bộ năm kiểm tra (trim, mã trống, not-found, lỗi khác, nil → rỗng) cộng test `Unique` và test giữ nguyên chi tiết ứng viên

## Phase 3 — Repository

- [x] `migrations/0026_administrative_units.sql` — bảy bảng `addr_*` theo mục Schema của `design.md`; thêm vào danh sách `applyMigration` của test; cập nhật `migrations/README.md` và `docs/specs/db/erd.mmd` (đã kiểm cú pháp)
- [x] Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` (mermaid-cli 11.4.2 chạy bằng Chrome cài sẵn, không tải trình duyệt); file mới chứa cả `addr_*` và các bảng lamp/genealogy của phiên khác vì chúng đã nằm trong `erd.mmd`
- [x] Tạo `repository/postgres_administrative_unit_repository.go` implement interface domain; mỗi read một câu SELECT (+ một câu kiểm tồn tại chỉ khi kết quả rỗng), `ORDER BY sort_order, code`; cột liệt kê tường minh, `context` xuyên suốt, bọc lỗi `%w`
- [x] `ReplaceDataset`: một transaction upsert theo thứ tự cha trước con, rồi `addr_ward_mappings`, rồi ghi `addr_dataset_versions`, rollback khi lỗi, **không DELETE** (Step 4–6 của flow nạp); INSERT nhiều dòng theo lô 1.000 (nạp file thật 1,3 s thay vì 22 s khi mỗi dòng một câu lệnh); từ chối `Version` rỗng
- [x] Repository test `repository/postgres_administrative_unit_repository_test.go` — 11 test trên Postgres thật: danh sách hai hệ theo `sort_order` (dữ liệu test cố ý đảo thứ tự so với mã), huyện không có xã là rỗng chứ không phải 404, chuyển đổi 1-1 / 1-nhiều / đặc khu rỗng / tỉnh suy ra từ xã, nạp lại không đổi gì (AC 20), cập nhật tại chỗ và không xoá đơn vị vắng, rollback nguyên giao dịch khi một dòng vi phạm FK, từ chối version rỗng, cùng một mã ở hai hệ không đụng nhau. *(thứ tự chữ cái "Phường 2" trước "Phường 10" nằm ở file dữ liệu — `sort_order` tính sẵn — nên được kiểm ở test parse của Phase 5, không ở đây)*

## Phase 4 — Delivery

- [x] Block config `administrative_units` (`rate_limit` + `cache_max_age`) + `Validate` trong `config/config.go` và `config.dev.yaml`, test `TestAdministrativeUnitsConfig_Validate` và `TestLoad_DevYAMLAdministrativeUnitsConfig`; `main.go` gọi `Validate` lúc khởi động
- [x] DTO + handler `delivery/http/administrative_unit_handler.go` cho 9 route (JSON tag chỉ ở đây); `Cache-Control` lấy từ config và chỉ đặt khi thành công (AC 24)
- [x] Thêm `ErrAdministrativeUnitNotFound` → `404 ADMINISTRATIVE_UNIT_NOT_FOUND` trong `delivery/http/errors.go`
- [x] Đăng ký route trong `services/appService/cmd/main.go` dạng `webAPIKey.WrapFunc(limit.WrapFunc(...))`; đã xác minh bằng cách chạy binary thật (key sai → 401, header vắng/đúng → 200, `Cache-Control` có ở 200 và không có ở 404) và bằng test lắp ráp cùng thứ tự ở `administrative_unit_handler_test.go`. `main()` không test được trực tiếp nên bước E2E ở Phase 6 vẫn cần để khoá hành vi này lại
- [x] Cập nhật `docs/specs/api/openapi.yaml` (+ bản đồng bộ `web/docs/public/openapi.yaml`): tag, chín route, 13 schema; `mapping_type` được mô tả là tập mở theo nhãn của nguồn. *(Quyết định có đặt lại tên các nhãn này cho client vẫn là marker mở ở `requirements.md` — đổi tên sẽ chỉ sửa DTO và schema.)*
- [x] Handler test `delivery/http/administrative_unit_handler_test.go` — danh sách đủ trường và không có `sort_order`, `Cache-Control` có ở 200 và không ở 404/500, `items = []` cho huyện không có xã, unique/nhiều ứng viên, đặc khu không ứng viên, 404 ở cả ba route danh sách, 500 không lộ nội dung lỗi, `Cache-Control` theo config, và chuỗi lắp ráp thật `RequireWebAPIKey` → `ClientIPRateLimit`: không token, key đúng, key sai 401, vượt ngân sách 429 không chạy handler, key sai không tiêu hao ngân sách

## Phase 5 — Seed CLI và dữ liệu

- [x] Tạo file dữ liệu duy nhất `assets/data/administrative-units.json` (3,2 MB; 34 / 3.321 / 63 / 696 / 10.033 / 10.565 mapping) — kèm `meta.source`, `meta.counts`, `meta.corrections`, `meta.unresolved`; đã tự kiểm toàn vẹn khi sinh (2026-10-10)
- [x] Xử lý hai lỗi nhãn: sửa trong file và ghi `meta.corrections` (quyết định Cách A, 2026-10-10)
- [x] Tạo `cmd/seedadminunits/parse.go` (đọc strict → `Dataset`, NFC có báo, không sửa im lặng; đọc nhưng không nạp `district_mappings`) và test `parse_test.go`: 13 ca file hỏng + đọc đủ các phần + NFC + counts, và `TestParseDataset_RealDataFile` (qua `Verify`, `meta.counts` khớp, thứ tự "Phường 9" trước "Phường 11"; **skip** khi file vắng vì `*.json` bị `.gitignore`)
- [x] Tạo `cmd/seedadminunits/main.go`: cờ `-file` và `-dry-run`, `Verify` trước khi chạm DB (in tối đa 50 vi phạm, exit 1), đọc config + secret dev, kết nối DB, `ReplaceDataset`, in báo cáo (Step 1–7). Đã chạy thật: dry-run không đổi DB; nạp lần đầu 34/3.321/63/696/10.033/10.565 trong 1,1 s; nạp lần hai không đổi số dòng và chỉ một dòng version; file bị làm hỏng bị từ chối (3 vi phạm) và DB không đổi; nạp lại sửa đúng dòng bị đánh dấu
- [x] Đưa script sinh file dữ liệu vào repo: `scripts/build_administrative_units.py` (docstring nêu cách lấy nguồn và chạy; ghi trong `scripts/README.md`). Đã kiểm: dựng lại từ nguồn cho ra file **giống từng byte** với `assets/data/administrative-units.json` đang có
- [x] `.gitignore`: thêm ngoại lệ `!assets/data/administrative-units.json` (repo chặn `*.json`); file nay hiện ra ở `git status` và commit được

## Phase 6 — Integration & documentation

- [x] E2E chạy binary thật + Postgres (testcontainers): `e2e/administrative_unit_e2e_test.go` — `TestAdministrativeUnits_E2E` (CLI dry-run / file hỏng / nạp hai lần, rồi chín route: không token, key đúng/vắng/sai, hai hệ, mã trùng khác nghĩa, chuyển đổi cũ→mới, mới→cũ, cấp tỉnh, đặc khu đảo, đọc không ghi, route/method lạ) và `TestAdministrativeUnits_RateLimit_E2E` (ngân sách chung, key sai không tiêu hao). Nạp dữ liệu bằng chính binary `cmd/seedadminunits`. Báo cáo AC ở `test-coverage-report.md`
- [x] `go test -short ./...` và `go vet ./...` sạch cho các package của feature; toàn repo xem changelog v1.7
- [ ] Ghi vào changelog concept OKF cần refresh sau khi hành vi đã ship: `docs/knowledge/features/administrative-units.md` (chưa có) — `il-okf-writer` tạo sau khi có E2E

## Phase 7 — Tìm xã theo tên (`?q=`, hệ mới)

Thêm sau quyết định 2026-10-11: app chỉ cho chọn hệ mới nên danh sách xã của một tỉnh (tới 168 mục) cần ô tìm.

- [x] Domain: `ErrInvalidAdministrativeUnitQuery` và `SearchCurrentWards(ctx, provinceCode, query)` trong `AdministrativeUnitRepository`; cập nhật `MockAdministrativeUnitRepository`
- [x] Usecase `SearchCurrentWards`: trim + chuẩn hoá NFC, rỗng hoặc quá 100 ký tự (đếm ký tự, không phải byte) → `ErrInvalidAdministrativeUnitQuery` mà không gọi repository; mã tỉnh trống → not found; test `TestAdministrativeUnitUsecase_SearchCurrentWards` (6 ca, gồm dạng tổ hợp `Hoa`+U+0300)
- [x] Repository `SearchCurrentWards`: `fold(full_name) LIKE fold($2) ESCAPE` với `fold = unaccent(lower(translate(x, 5 kiểu dấu nháy, ')))`, ràng buộc trong tỉnh, thứ tự `sort_order, code`, rỗng → kiểm tỉnh tồn tại; test trên Postgres thật `TestAdministrativeUnitRepository_SearchCurrentWards` (vị trí dấu thanh cả hai chiều, dấu nháy cong/thẳng cả hai chiều, `đ`/`d`, `a_b` nghĩa đen, tiền tố loại, ràng buộc tỉnh, không khớp, tỉnh lạ)
- [x] Handler: `CurrentWards` rẽ sang tìm khi `q` có mặt (rỗng cũng là có mặt); `ErrInvalidAdministrativeUnitQuery` → `400 INVALID_FILTER` (`details.field = "q"`) trong `errors.go`; test `TestAdministrativeUnitHandler_CurrentWardsSearch`
- [x] `docs/specs/api/openapi.yaml` (+ bản đồng bộ `web/docs/public/openapi.yaml`): tham số `q`, mô tả, response `400`; đã kiểm YAML và `$ref`
- [x] E2E (`TestAdministrativeUnits_E2E/ward_search…`) trên dữ liệu thật: `Hóa`/`Hoá`/`HÓA`/`hoá` ra cùng tập xã ở Thanh Hóa; `b'lao`, `b’lao`, `B’Lao` tìm ra "B’Lao" ở Lâm Đồng; `phuong tan dinh` ra "Tân Định" ở Hồ Chí Minh; `q` rỗng/trắng/101 ký tự → 400; tỉnh lạ → 404; không `q` thì vẫn đủ xã

## Deferred (separate feature, not blocking administrative-units)

- Chuyển đổi cấp huyện + bảng `addr_district_mappings` (người dùng hoãn, 2026-10-10); dữ liệu đã nằm sẵn trong file.
- Đối chiếu bộ dữ liệu với `conversion-table.xlsx` chính thức, kiểm giấy phép dữ liệu, và xử lý dòng "Kim Sơn → Kim Đông" (hoãn, 2026-10-10).
- Đánh dấu hết hiệu lực cho đơn vị vắng trong bản dữ liệu mới.
- Tìm theo tên không dấu (`?q=`) nếu cần.
- Chuyển `temples.province` (chuỗi tự do) sang mã `addr_new_provinces` — thuộc feature `temple` (tên đã trùng khít `domain/province.go`).
- Lưu địa chỉ người dùng / gia phả / sổ giỗ theo mã đơn vị — mỗi feature dùng địa chỉ có spec riêng.
- Hợp nhất `domain/province.go` với bảng `addr_new_provinces` nếu muốn một nguồn duy nhất.
