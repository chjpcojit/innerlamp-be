# Tasks — profile

Version: 1.5
Last updated: 2026-10-11
Status: implemented, Phase 1–5 xong và đã kiểm chứng (unit + repository trên Postgres thật + adapter và E2E luồng avatar đầy đủ trên emulator `fake-gcs-server` thật, URL không ký). Còn mở: hai mục kiểm tra trước deploy/go-live (`TD-001`, `TD-002`), mục `US-004` của người duyệt spec và concept OKF. Không còn marker `⚠️ Not confirmed`; hai nợ kỹ thuật `TD-001`, `TD-002` ở `docs/tech-debt/profile/`.
Module: `profile`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Làm theo TDD: test của từng phase viết cùng lúc với code của phase đó. Dùng skill `golang-clean-arch-tdd` để implement, `golang-e2e-verification` để kiểm chứng AC → test.

## Phase 1 — Domain & Config

- [x] `domain/profile.go`: entity `Profile`, `TempleRef`, `ProfilePatch` (hai trường kiểu `OptionalString{Set, Value}` phân biệt vắng mặt / xoá / giá trị), `AvatarUploadRequest`, `AvatarUploadTicket`, `ObjectMeta`, và `ProfileView` (Profile kèm địa chỉ ảnh đã ký, kết quả của usecase)
- [x] Sentinel errors `ErrAvatarUploadNotFound`, `ErrInvalidAvatar`, `ErrObjectNotFound`, `ErrProfileInvalid` + `ProfileValidationError{Field, Message}` (dùng lại `ErrInvalidTemple`, `ErrUserNotFound`)
- [x] Interface `ProfileRepository` (`Get`, `Update`, `SwapAvatar`, `ClearAvatar`, `TempleExists`) và `AvatarStorage` (`SignUpload`, `SignRead`, `Stat`, `Copy`, `Delete`)
- [x] `domain/mock_profile_repository.go`, `domain/mock_avatar_storage.go`
- [x] `config/config.go`: `ProfileConfig{DharmaNameMaxChars}` + `ProfileDharmaNameColumnChars = 255` + `Validate` (từ chối `< 1` và `> 255`) + test `config_test.go`; `config.dev.yaml`: `profile.dharma_name_max_chars: 255` — làm trước ở v1.1 vì đăng ký dùng chung
- [x] `config/config.go`: **phần còn lại** của `ProfileConfig{DharmaNameMaxChars, AvatarMaxBytes, AvatarContentTypes, UploadURLTTL, AvatarReadURLTTL, Bucket, AvatarTmpPrefix, AvatarPrefix, RateLimit{Max, Window}}` + `Validate` (từ chối số `< 1`, `DharmaNameMaxChars > 255`, danh sách rỗng/trùng, TTL `<= 0`, `Bucket` rỗng, hai prefix rỗng hoặc trùng nhau) + test trong `config_test.go`
- [x] `config/config.dev.yaml`: block `profile:` (`dharma_name_max_chars: 255`, `avatar_max_bytes: 5242880`, `avatar_content_types`, `upload_url_ttl: "5m"`, `avatar_read_url_ttl: "15m"`, `bucket`, `avatar_tmp_prefix: "avatars/tmp/"`, `avatar_prefix: "avatars/"`, `rate_limit: {max: 10, window: "1m"}`); đánh dấu `⚠️ DEV CONFIG` cho `bucket`

- [x] `config/config.go`: `StorageConfig.Timeout` + `StorageConfig.Validate` (từ chối backend khác `gcs`, `project_id` rỗng, timeout `<= 0`) + test; `config.dev.yaml`: `storage.timeout: "30s"` — thêm vì `pkg/storage` bắt buộc timeout > 0 và đây là giá trị đổi theo môi trường nên không để hằng trong `cmd/`

## Phase 2 — Usecase

- [x] Hàm chuẩn hoá + validate pháp danh dùng chung (trim, NFC, đếm ký tự, rỗng → `nil`) đặt ở nơi cả `RegisterUsecase` và `ProfileUsecase` gọi được
- [x] `usecase/register_usecase.go`: dùng hàm chung (`normalizeDharmaName`, v1.1; `ProfileUsecase` sẽ gọi lại cùng hàm); pháp danh quá độ dài trả `400 INVALID_DHARMA_NAME` thay vì `500` (khoảng trống ở `docs/HANDOFF.md`); test hồi quy cho 256 ký tự
- [x] `usecase/profile_usecase.go`: `Get`, `Update` (Step P-3…P-5: AC8–AC15), `RequestAvatarUpload` (Step U-3, U-4: AC18–AC20), `ConfirmAvatar` (Step C-2…C-8: AC22–AC25), `RemoveAvatar` (Step D-2, D-3: AC26); `now` và sinh UUID inject được
- [x] Unit tests table-driven (`testify`), mock repo + mock storage: biên 255/256 ký tự với chữ có dấu tổ hợp và dựng sẵn; toàn dấu cách = xoá; vắng mặt vs `null` vs rỗng; chùa lạ không ghi gì kể cả khi pháp danh hợp lệ; `size_bytes` đúng 5 MB hợp lệ, 5 MB + 1 không; content type ngoài danh sách; `Stat` không có object; copy lỗi → DB không đổi; xoá ảnh cũ lỗi vẫn thành công; thứ tự gọi `Copy` → `SwapAvatar` → `Delete`

## Phase 3 — Repository & Migration

- [x] `migrations/0029_users_avatar.sql`: `ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_object VARCHAR(255)` (idempotent); cập nhật `migrations/README.md`
- [x] Cập nhật `docs/specs/db/erd.mmd` (cột `avatar_object` của `users`)
- [x] Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` (đã sinh ở v1.1 bằng `npx @mermaid-js/mermaid-cli`; `erd.svg` có cột `avatar_object`)
- [x] `repository/postgres_profile_repository.go`: `Get` (LEFT JOIN `temples`), `Update` (chỉ SET cột có trong patch; xoá = `NULL`; FK 23503 → `ErrInvalidTemple`), `SwapAvatar`/`ClearAvatar` (một tx, `SELECT … FOR UPDATE` rồi `UPDATE`), `TempleExists`; explicit column list, `context` mọi query
- [x] Repository tests (Postgres thật): patch từng phần không đụng cột khác; xoá thành `NULL` không phải `''`; hai `SwapAvatar` đồng thời mỗi lần nhận đúng khoá cũ của mình; `ClearAvatar` khi chưa có ảnh trả rỗng; chùa bị xoá giữa chừng → `ErrInvalidTemple`

## Phase 4 — Storage adapter

- [x] `services/appService/storage/avatar_storage.go`: cài `AvatarStorage` trên `pkg/storage.Client` — `SignUpload` (PUT, `ContentType`, header `x-goog-content-length-range`), `SignRead` (GET), `Stat` (`GetObjectInfo`, map not-found → `ErrObjectNotFound`), `Copy`, `Delete`
- [x] Test adapter với `storage.Client` giả (đúng method/headers/TTL được truyền xuống)
- [x] Test tích hợp adapter với **emulator** `fake-gcs-server` (testcontainers; bỏ qua khi `-short` hoặc không có Docker): ticket → upload → `Stat` → `Copy` → URL đọc → `Delete` (`storage/avatar_storage_emulator_test.go`)
- [x] Chế độ emulator của adapter (`WithEmulator`): URL **không ký** (upload `POST /upload/storage/v1/b/{bucket}/o?uploadType=media&name=…` vì emulator không có PUT upload; đọc `/download/storage/v1/b/{bucket}/o/{key}?alt=media`); cổng `AvatarStorage.SignUpload` trả `SignedUpload{URL, Method, Headers}` nên usecase không còn cố định `PUT`
- [x] `config`: `storage.emulator_host` (`host:port`, để trống ngoài dev/test) + validate + test; `config.dev.yaml: "localhost:4443"` đánh dấu `⚠️ DEV ONLY`
- [x] `docker-compose.yml`: service `gcs` (`fsouza/fake-gcs-server:1.56.1`, cổng 4443, volume `gcs-data`) chạy cùng `make up` / `make run`; `cmd/main.go` đặt `STORAGE_EMULATOR_HOST` từ config và tạo bucket nếu thiếu (chỉ cảnh báo khi emulator chưa lên)

- [x] Ghi chú trong code `// TD-001` tại `SignUpload`/`SignRead` (signed URL chỉ được kiểm chứng trên emulator) và `// TD-002` tại nhánh xoá file cũ lỗi

## Phase 5 — Delivery & Wiring

- [x] `delivery/http/user_rate_limit.go`: `UserRateLimit` (khoá `user_id` từ `pkg/jwt/context`) + test: request thứ `Max+1` → `429`, tài khoản khác không bị ảnh hưởng, request không có userID không lọt vào bộ đếm
- [x] `delivery/http/profile_handler.go`: 5 handler + DTO (`omitempty` đúng, kiểu `optionalString` tri-state), `Cache-Control: no-store`
- [x] `delivery/http/errors.go`: map `*ProfileValidationError` (`ErrInvalidDharmaName` đã map ở v1.1), `ErrInvalidTemple` (`details.field = "ordination_temple_id"`), `ErrInvalidAvatar` → `400 INVALID_AVATAR`, `ErrAvatarUploadNotFound` → `404 AVATAR_UPLOAD_NOT_FOUND`
- [x] Handler tests (`httptest`): AC2 (key vắng mặt), AC15 (`email`/`phone_number`/`{}`/body rỗng), 401 khi thiếu token
- [x] `cmd/main.go`: dựng `storage.Client` + `Connect(ctx)` lúc khởi động, adapter, repo, usecase, handler; đăng ký 5 route bọc `authMiddleware.AuthenticateFunc` → `userRateLimit.WrapFunc` → `noStore` qua `registerProfileRoutes` (`cmd/profile_routes.go`); `cmd/profile_routes_test.go` chứng minh cả 5 route bị chặn 401 khi thiếu auth, có `Cache-Control: no-store`, và dùng chung một ngân sách giới hạn
- [x] `pkg/middleware`: CORS cho phép thêm method `PATCH` (trước đó chỉ `GET, POST, PUT, DELETE, OPTIONS` nên preflight của `PATCH /v1/users/me` từ web bị chặn); cập nhật test và README của package
- [x] Chạy được không cần credential GCP khi dev: nhờ `storage.emulator_host` (GCS client không dựng được khi thiếu ADC nếu không trỏ emulator); e2e harness dùng **một** container emulator dùng chung và ghi đè `storage.emulator_host`, mỗi test dùng bucket riêng, không đụng emulator dev

## Phase 6 — Docs, kiểm chứng, việc hoãn

- [x] `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`): tag `Profile`, 5 route, 5 schema — **đã viết cùng bộ spec này (v1.0)**, tick sau khi đối chiếu với handler thật
- [x] `web/docs/src/content/docs/guides/catalogues/profile.md` + sidebar `astro.config.mjs` + `changelog.md` — **đã viết cùng bộ spec này (v1.0)**, tick sau khi đối chiếu với handler thật
- [x] `services/appService/e2e/profile_e2e_test.go` (binary thật + Postgres + emulator): 401 cả 5 route (thiếu/rác/refresh token), PATCH tri-state, từ chối `email`/`phone_number`, chùa lạ không lưu gì, `429` ở request thứ 11 và không ảnh hưởng tài khoản khác, upload-url từ chối sai loại/quá dung lượng
- [x] E2E luồng avatar đầy đủ: upload-url → upload → confirm → đọc lại ảnh → thay (ảnh cũ bị xoá khỏi storage) → gỡ; id chưa upload hoặc của người khác → `404 AVATAR_UPLOAD_NOT_FOUND`; object sai loại → `400 INVALID_AVATAR` và avatar cũ còn nguyên. **Lưu ý:** emulator không ký/không ép giới hạn, nên các test này không chứng minh gì về chữ ký, 5 MB ở phía storage hay hạn URL (`TD-001`)
- [x] `go vet ./...`, `go test -short ./...`, `go test ./services/appService/e2e/` pass
- [ ] Trỏ mục `US-004` của `docs/specs/features/onboarding/requirements.md` sang spec này (việc của người duyệt spec)
- [ ] **Trước deploy đầu tiên có route avatar (`TD-001`):** chạy ticket → PUT → confirm trên bucket GCS thật bằng service account của deploy; kiểm tra PUT bị từ chối khi sai `Content-Type`, quá hạn, quá 5 MB; kiểm tra quyền `signBlob`, CORS và hạn 15 phút của URL đọc
- [ ] **Trước go-live (`TD-002`):** dọn thư mục tạm (đề xuất: lifecycle rule của bucket xoá `avatar_tmp_prefix` sau 1 ngày) và cách xử lý file cũ xoá lỗi — đang hoãn có chủ đích
- [ ] Concept OKF `docs/knowledge/features/profile.md` + `index.md` + `log.md` (`il-okf-writer`, sau khi flow được dùng thật end-to-end)

## Phase 7 — Họ tên `full_name` (v1.5)

- [x] `domain/profile.go`: `Profile.FullName`, `ProfilePatch.FullName` (+ `IsEmpty`); `domain/errors.go`: `ErrInvalidFullName` (dùng chung với đăng ký)
- [x] `usecase/profile_usecase.go`: `normalizePatch` — `null`/rỗng/quá `FullNameMaxChars` → `ErrInvalidFullName`, không xoá được; dùng lại `normalizeFullName` của đăng ký
- [x] `repository/postgres_profile_repository.go`: `profileSelect`, `scanProfile`, `RETURNING` và `add("full_name", ...)`
- [x] `delivery/http/profile_handler.go`: `profilePatchRequest.FullName`, `profileResponse.full_name` (`omitempty`); `delivery/http/errors.go` map `ErrInvalidFullName` → `400 INVALID_FULL_NAME`
- [x] Test: usecase (NFC, null/blank/quá dài bị từ chối và không ghi, đúng giới hạn 300 được nhận), handler (tri-state, mã + `details.field`), repository (round-trip, giữ nguyên `dharma_name`), E2E (đọc sau đăng ký → sửa → null/blank/301 ký tự bị từ chối, giá trị không đổi)
- [ ] Concept OKF `docs/knowledge/features/profile.md` nên nhắc `full_name` khi được làm mới (`il-okf-writer`)
