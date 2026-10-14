# Tasks — memorial

Version: 3.0
Last updated: 2026-10-14
Status: v1 (Phase 1–5, 5 route CRUD, không ảnh) đã implement và kiểm chứng 2026-10-07 — unit, repository (Postgres thật) và E2E (binary thật + Postgres) đều pass. Còn: chốt marker bucket (chỉ cần trước Phase 6). Phase 6 (ảnh/GCS) hoãn, ghi nợ ở `docs/tech-debt/memorial/TD-001-photo-upload-blocked-on-gcs.md`. Phase 7 (gắn product spec v1.0) chưa làm.
Module: `memorial`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`.

## Phase 1 — Domain

- [x] `domain/memorial.go`: entity `Memorial` (`DeathDate *time.Time`, `DeathDay/Month *int`, `DeathLeapMonth *bool`), `Reminder`, enum `Relationship` (10 giá trị), `DeathCalendar`
- [x] Sentinel errors `ErrMemorialNotFound`, `ErrMemorialLimitReached`, `ErrMemorialInvalid` + `MemorialValidationError{Field, Message}`
- [x] Interface `MemorialRepository` (`Create`, `ListByUser`, `FindByIDAndUser`, `Update`, `Delete`; mọi method một mục nhận `id` + `userID`). Đổi so với bản trước: `Delete` chỉ trả `error` và `SetPhotoKey` chưa khai báo — cả hai thay đổi chữ ký thuộc Phase 6 (v1 không có ảnh nên không có gì để trả/ghi)
- [x] `domain/mock_memorial_repository.go`

## Phase 2 — Usecase

- [x] `usecase/memorial_usecase.go`: `Create`, `List`, `Get`, `Update`, `Delete` (`photo_url` luôn `null` ở v1)
- [x] Validation AC3–AC10 (trim tên, enum, hai dạng ngày mất: đầy đủ 1900..hôm nay theo `Asia/Ho_Chi_Minh` hoặc ngày+tháng theo lịch đã nhập với quy tắc solar/lunar của AC6a/6b, `time.LoadLocation`, `HH:mm`, cờ OS cần reminder)
- [x] Unit tests table-driven (`testify`), mock repo: mỗi AC validation, 404 đồng nhất, trần 100

## Phase 3 — Repository

- [x] `migrations/0018_memorials.sql` (idempotent `IF NOT EXISTS`, gồm cột `photo_object_key` NULL; CHECK reminder all-or-none, CHECK OS cần reminder, CHECK hai dạng ngày mất loại trừ nhau + `death_leap_month` chỉ với `lunar`) + index `idx_memorials_user`; cập nhật `migrations/README.md`
- [x] Cập nhật `docs/specs/db/erd.mmd` (bảng `memorials` + quan hệ `users 1—N memorials`)
- [x] Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` bằng `mmdc` (mermaid-cli) — render thành công nên cú pháp `erd.mmd` hợp lệ; bản svg cũ còn thiếu cả `gallery_images`, bản mới có đủ
- [x] `repository/postgres_memorial_repository.go`: `Create` (tx: khoá `users`, đếm, chèn), `ListByUser`, `FindByIDAndUser`, `Update`, `Delete ... RETURNING`; `WHERE … AND user_id`
- [x] Repository tests (DB thật): trần 100 dưới tải song song, người lạ không đọc/sửa/xoá được, `ON DELETE CASCADE`, CHECK constraints bị DB từ chối

## Phase 4 — Delivery

- [x] DTO + handler `Create`, `List`, `Get`, `Update`, `Delete` trong `delivery/http/memorial_handler.go`
- [x] Map lỗi trong `delivery/http/errors.go` (404/409)
- [x] `config.go` + `config.dev.yaml`: `MemorialConfig{MaxPerUser}`
- [x] Đăng ký 5 route trong `cmd/main.go`, **mỗi route bọc `authMiddleware.AuthenticateFunc`**
- [x] Cập nhật `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml` cùng thay đổi handler (5 route, field `photo_url` luôn `null`)
- [x] Handler tests: thành công, từng nhánh lỗi, body sai/field lạ, hai dạng ngày mất

## Phase 5 — Integration & documentation

- [x] E2E (skill `golang-e2e-verification`): binary thật + Postgres — user B nhận 404 trên mục của A cho cả 5 route; thiếu token → 401 cho cả 5 route (chứng minh wiring middleware); tạo đồng thời > 100
- [x] `go test -short ./...` và `go vet ./...` sạch (toàn repo). Chạy đủ không `-short`: chỉ `TestSutraRepository_ShippedData` fail, không liên quan memorial (số `DeityLinks` của dữ liệu seed kinh: 191 thực tế so với 151 mong đợi)
- [ ] Marker `⚠️ Not confirmed` còn lại (bucket GCS) chỉ cần chốt trước Phase 6; ghi vào `changelog.md` khi chốt
- [x] Ghi vào changelog: concept OKF `docs/knowledge/features/memorial.md` cần tạo bởi `il-okf-writer` sau khi code được kiểm chứng

## Phase 6 — Ảnh (US-MEM-05) ⏸ HOÃN, chặn bởi GCS

Chỉ bắt đầu khi GCS dùng được và marker bucket/ký URL đã chốt.

- [ ] `domain/memorial.go`: port `MemorialPhotoStore` (`Put`/`Delete`/`SignedURL`), sentinel `ErrPhotoTooLarge`, `ErrPhotoUnsupportedType`; `domain/mock_memorial_photo_store.go`
- [ ] `repository/gcs_memorial_photo_store.go` bọc `pkg/storage` (constructor không mở kết nối; `Connect` ở `main`); `SetPhotoKey` trong repo
- [ ] Usecase `SetPhoto`/`RemovePhoto`, build `photo_url` ở `List`/`Get`, nhánh xoá object ở `Delete` (Step P-1..P-6, D-3); unit tests: thất bại GCS không đổi key trong DB, xoá object lỗi vẫn trả thành công
- [ ] Handler `UploadPhoto`/`DeletePhoto`: `MaxBytesReader` 5 MB, multipart, sniff content type, 404 trước khi đọc body (AC31); map lỗi 413/415
- [ ] `MemorialConfig`: `PhotoMaxBytes`, `PhotoURLTTL`, `PhotoBucket`, `PhotoPrefix`; storage client ở `cmd/main.go`; đăng ký 2 route ảnh bọc `authMiddleware.AuthenticateFunc`
- [ ] `openapi.yaml` (+ bản `web/`) thêm 2 route ảnh; E2E mở rộng cho 7 route
- [ ] Cơ chế dọn object mồ côi (đã chốt "tính sau")

## Phase 7 — Product spec alignment (`docs/product/memorial/product.md` @ v1.0)

Không đổi hành vi; chỉ test và mô tả contract.

- [ ] `domain/lunarcal/memorial_vectors_test.go`: chạy mọi case của `docs/product/memorial/rules/death-date-conversion.json` (BR-MEM-03, BR-MEM-04, BR-MEM-10) trên `SolarToLunar`/`LunarToSolar`, đọc file từ gốc repo, không chép — theo dòng "Product vectors" của `design.md`
- [ ] `openapi.yaml` (+ `web/docs/public/openapi.yaml`): mô tả `death_leap_month` (`null` khi có `death_date`; client khôi phục cờ nhuận bằng cách đổi `death_date` dương → âm), `reminder.timezone` (múi giờ của máy, mặc định `Asia/Ho_Chi_Minh`), `PUT` thay toàn bộ nên client phải gửi lại `reminder` đang có — product Appendix B mục 1, 3, 6. Không đổi schema

## Deferred (separate feature, not blocking memorial)

- Push notification từ server + lịch âm ở BE (cần thư viện âm lịch VN và scheduler)
- Chia sẻ sổ giỗ cho người thân khác
- Xoá tài khoản kéo theo dọn ảnh trên GCS
