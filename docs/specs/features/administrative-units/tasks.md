# Tasks — administrative-units

Version: 1.4
Last updated: 2026-10-10
Status: Phase 1 (domain), 2 (usecase) và 3 (repository) xong, đã test — repository chạy trên Postgres thật (11 test xanh, nạp file dữ liệu thật 1,3 s); còn lại `erd.svg` chưa sinh lại, Phase 4, 6 và CLI của Phase 5. Không còn mục nào bị chặn bởi câu hỏi mở.
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
- [ ] Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` — chưa làm: máy này không có `mmdc` (mermaid-cli); `erd.svg` hiện vẫn là bản trước `0026`
- [x] Tạo `repository/postgres_administrative_unit_repository.go` implement interface domain; mỗi read một câu SELECT (+ một câu kiểm tồn tại chỉ khi kết quả rỗng), `ORDER BY sort_order, code`; cột liệt kê tường minh, `context` xuyên suốt, bọc lỗi `%w`
- [x] `ReplaceDataset`: một transaction upsert theo thứ tự cha trước con, rồi `addr_ward_mappings`, rồi ghi `addr_dataset_versions`, rollback khi lỗi, **không DELETE** (Step 4–6 của flow nạp); INSERT nhiều dòng theo lô 1.000 (nạp file thật 1,3 s thay vì 22 s khi mỗi dòng một câu lệnh); từ chối `Version` rỗng
- [x] Repository test `repository/postgres_administrative_unit_repository_test.go` — 11 test trên Postgres thật: danh sách hai hệ theo `sort_order` (dữ liệu test cố ý đảo thứ tự so với mã), huyện không có xã là rỗng chứ không phải 404, chuyển đổi 1-1 / 1-nhiều / đặc khu rỗng / tỉnh suy ra từ xã, nạp lại không đổi gì (AC 20), cập nhật tại chỗ và không xoá đơn vị vắng, rollback nguyên giao dịch khi một dòng vi phạm FK, từ chối version rỗng, cùng một mã ở hai hệ không đụng nhau. *(thứ tự chữ cái "Phường 2" trước "Phường 10" nằm ở file dữ liệu — `sort_order` tính sẵn — nên được kiểm ở test parse của Phase 5, không ở đây)*

## Phase 4 — Delivery

- [ ] Block config `administrative_units.rate_limit` + `Validate` trong `config/config.go` và `config.dev.yaml` (+ test trong `config_test.go`)
- [ ] DTO + handler `delivery/http/administrative_unit_handler.go` cho 9 route (JSON tag chỉ ở đây); đặt `Cache-Control: public, max-age=86400` khi thành công (AC 24)
- [ ] Thêm `ErrAdministrativeUnitNotFound` → `404 ADMINISTRATIVE_UNIT_NOT_FOUND` trong `delivery/http/errors.go`
- [ ] Đăng ký route trong `services/appService/cmd/main.go` dạng `webAPIKey.WrapFunc(limit.WrapFunc(...))`; xác minh thật sự đã bọc (test wiring giống `cmd/main_test.go`), không chỉ ghi trong tài liệu
- [ ] Chốt tập giá trị `mapping_type` trả ra client (marker còn mở) rồi cập nhật `docs/specs/api/openapi.yaml` (+ bản đồng bộ `web/docs/public/openapi.yaml`) cùng thay đổi
- [ ] Handler test `delivery/http/administrative_unit_handler_test.go` — thành công (có `Cache-Control`), 404, header key đúng / vắng / sai (`401 INVALID_API_KEY`), `items = []` cho huyện không có xã, 429 khi vượt ngân sách

## Phase 5 — Seed CLI và dữ liệu

- [x] Tạo file dữ liệu duy nhất `assets/data/administrative-units.json` (3,2 MB; 34 / 3.321 / 63 / 696 / 10.033 / 10.565 mapping) — kèm `meta.source`, `meta.counts`, `meta.corrections`, `meta.unresolved`; đã tự kiểm toàn vẹn khi sinh (2026-10-10)
- [x] Xử lý hai lỗi nhãn: sửa trong file và ghi `meta.corrections` (quyết định Cách A, 2026-10-10)
- [ ] Tạo `cmd/seedadminunits/parse.go` (đọc → `Dataset`, NFC, không sửa im lặng; đọc nhưng không nạp `district_mappings`) và test `parse_test.go`, gồm một test nạp chính file thật và so với `meta.counts`
- [ ] Tạo `cmd/seedadminunits/main.go`: cờ `-file`, đọc config + secret dev, kết nối DB, `Verify` rồi `ReplaceDataset`, in báo cáo (Step 1–7)
- [ ] Quyết định có đưa script sinh file dữ liệu vào repo (thư mục `scripts/`) hay không — hiện script chỉ nằm ngoài repo; cần nếu muốn cập nhật từ nguồn khi có nghị quyết mới

## Phase 6 — Integration & documentation

- [ ] E2E chạy binary thật + Postgres (testcontainers) cho các route: danh sách hai hệ, chuyển đổi, gọi không token, key sai, vượt giới hạn (skill `golang-e2e-verification`)
- [ ] `go test -short ./...` và `go vet ./...` sạch
- [ ] Ghi vào changelog concept OKF cần refresh sau khi hành vi đã ship: `docs/knowledge/features/administrative-units.md` (chưa có) — `il-okf-writer` tạo sau khi có E2E

## Deferred (separate feature, not blocking administrative-units)

- Chuyển đổi cấp huyện + bảng `addr_district_mappings` (người dùng hoãn, 2026-10-10); dữ liệu đã nằm sẵn trong file.
- Đối chiếu bộ dữ liệu với `conversion-table.xlsx` chính thức, kiểm giấy phép dữ liệu, và xử lý dòng "Kim Sơn → Kim Đông" (hoãn, 2026-10-10).
- Đánh dấu hết hiệu lực cho đơn vị vắng trong bản dữ liệu mới.
- Tìm theo tên không dấu (`?q=`) nếu cần.
- Chuyển `temples.province` (chuỗi tự do) sang mã `addr_new_provinces` — thuộc feature `temple` (tên đã trùng khít `domain/province.go`).
- Lưu địa chỉ người dùng / gia phả / sổ giỗ theo mã đơn vị — mỗi feature dùng địa chỉ có spec riêng.
- Hợp nhất `domain/province.go` với bảng `addr_new_provinces` nếu muốn một nguồn duy nhất.
