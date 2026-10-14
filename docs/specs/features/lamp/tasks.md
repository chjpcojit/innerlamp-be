# Tasks — lamp

Version: 1.3
Last updated: 2026-10-11
Status: v1 (Phase 1–4: domain, config, usecase, repository, migration `0027`, handler, wiring, openapi) đã implement và kiểm chứng 2026-10-10 — unit, repository (Postgres thật) và E2E (binary thật + Postgres) đều pass. Còn 2 marker `⚠️ Not confirmed` (hiệu năng/khối lượng; retention + xoá tài khoản — migration đang dùng `ON DELETE CASCADE` tạm thời). Chưa sinh lại `erd.svg` (thiếu `mmdc`).
Module: `lamp`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Làm theo TDD: test của từng phase viết cùng lúc với code của phase đó.

## Phase 1 — Domain & Config

- [x] `domain/lamp.go`: entity `Lamp`, `LampInput`, `GardenLamp`, `GardenFilter`, `GardenQuery`, `GardenPage`, `Cursor`, `LightLampParams`; enum `LampType` (3 giá trị), `Intention` (4 giá trị); hằng `AnonymousLighterName = "Vô danh"`
- [x] Sentinel errors `ErrLampLimitReached`, `ErrInvalidLampFilter`, `ErrInvalidLimit`, `ErrInvalidCursor`, `ErrLampInvalid` + `LampValidationError{Field, Message}`
- [x] Interface `LampRepository` (`Light`, `Garden`)
- [x] `domain/mock_lamp_repository.go`
- [x] `config/config.go`: `LampConfig{MaxActivePerUser, PageSize, DedupeWindow, DurationsDays}` + `Validate` (từ chối `< 1`, cửa sổ `<= 0`, danh sách rỗng/trùng/`< 1`) + test trong `config_test.go`
- [x] `config/config.dev.yaml`: block `lamp:` (`max_active_per_user: 20`, `page_size: 12`, `dedupe_window: "5s"`, `durations_days: [1, 7, 21, 49]`)
- [x] v1.3 — `config/config.go`: thêm `RecipientMaxChars`, `PrayerMaxChars` vào `LampConfig`, hằng trần `LampRecipientColumnChars = 100` / `LampPrayerColumnChars = 300`, `Validate` từ chối `< 1` hoặc vượt trần + test trong `config_test.go`
- [x] v1.3 — `config/config.dev.yaml`: thêm `recipient_max_chars: 100` và `prayer_max_chars: 300` vào block `lamp:`
- [x] v1.3 — `usecase/lamp_usecase.go`: bỏ hằng `lampRecipientMax`/`lampPrayerMax`, kiểm độ dài theo `cfg.RecipientMaxChars`/`cfg.PrayerMaxChars`; test với giới hạn khác 100/300 chứng minh giá trị lấy từ config
- [x] v1.3 — `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) và `web/docs/.../lamps.md`: ghi 100/300 là giá trị mặc định của config

## Phase 2 — Usecase

- [x] `usecase/lamp_usecase.go`: `Light` (trim + NFC, validate AC3–AC8, tính `litAt`/`endAt`), `Garden` (validate AC17/AC19/AC21, kẹp `limit`, giải mã cursor, dựng `next_cursor`/`has_more`, `NULL` → `Vô danh`); `now` inject được
- [x] Mã hoá/giải mã cursor `<lit_at unix micro>.<id>` base64url (nằm trong `usecase/lamp_usecase.go`)
- [x] Unit tests table-driven (`testify`), mock repo: mỗi AC validation, biên độ dài 1/100/101 và 300/301 với chữ có dấu tổ hợp và dựng sẵn, chuỗi chỉ toàn dấu cách, `end_at` đúng N×24h với cả 4 thời lượng, kẹp `limit`, cursor hỏng, `lighter_name` ẩn danh/không pháp danh

## Phase 3 — Repository

- [x] Chốt mệnh đề `ON DELETE` của `lamps.user_id`: **tạm thời `ON DELETE CASCADE`** (giống `memorials`; marker retention vẫn mở, đổi bằng migration mới khi chốt)
- [x] `migrations/0027_lamps.sql` (idempotent `IF NOT EXISTS`; CHECK enum, `end_at > lit_at`, ẩn danh ⇒ `lighter_name IS NULL`; ba index `idx_lamps_lit`, `idx_lamps_type_lit`, `idx_lamps_user_end`); cập nhật `migrations/README.md` — số `0027` vì `0026` đã bị `0026_administrative_units.sql` chiếm
- [x] Cập nhật `docs/specs/db/erd.mmd` (bảng `lamps` + quan hệ `users 1—N lamps`)
- [ ] Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` bằng `mmdc` — máy này chưa có `mmdc`
- [x] `repository/postgres_lamp_repository.go`: `Light` (một tx: khoá `users` + đọc `dharma_name`, chống trùng, đếm trần, chèn), `Garden` (tx `READ ONLY` `REPEATABLE READ`: `GROUP BY` đếm + trang keyset `LIMIT limit+1`); explicit column list, `context`, `%w`
- [x] Repository tests (`postgres_lamp_repository_test.go`, Postgres thật): trần 20 kể cả khi hai `Light` chạy song song, đèn hết hạn không tính, chống trùng trong/ngoài cửa sổ, ẩn danh không lưu tên, đổi `dharma_name` không đổi đèn cũ, keyset không lặp/sót khi một đèn hết hạn giữa hai trang, `counts` không đổi theo `lamp_type`

## Phase 4 — Delivery

- [x] `delivery/http/lamp_handler.go`: DTO (`lampRequest`, `lampResponse`, `gardenLampResponse`, `gardenResponse`), `Light` (`201`/`200`), `Garden`; `items` luôn `[]`, `counts` luôn đủ 3 khoá, `lit_at`/`end_at` RFC 3339 UTC
- [x] `delivery/http/errors.go`: map `ErrLampLimitReached`, `ErrInvalidLampFilter`, `ErrInvalidLimit`, `ErrInvalidCursor`, `*LampValidationError`
- [x] Đăng ký `POST /v1/lamps` và `GET /v1/lamps` trong `cmd/main.go`, **cả hai bọc `authMiddleware.AuthenticateFunc`** — kiểm tra thật sự đã bọc
- [x] Cập nhật `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) cùng thay đổi handler
- [x] Handler tests (`lamp_handler_test.go`): thành công, `201` vs `200`, từng nhánh lỗi, body thừa trường/rỗng/không phải JSON, `limit`/`cursor`/`lamp_type` sai

## Phase 5 — Integration & documentation

- [x] `e2e/lamp_e2e_test.go` (binary thật + Postgres): 401 cho cả hai route khi thiếu token, token rác, refresh token; thắp → xuất hiện trong vườn đèn; đèn hết hạn biến mất và không tính trần; `total`/`counts` không đổi theo bộ lọc; ẩn danh không lộ pháp danh; tâm nguyện/lời nguyện không có trong response vườn đèn
- [x] `go vet ./services/appService/...` sạch; `go test -short ./services/appService/...` và `go test ./services/appService/e2e/ -run TestLampFlow_E2E` đều pass (gói `usecase` chạy qua `-overlay` vì `genealogy_member_usecase_test.go` của công việc khác đang không biên dịch)
- [ ] Ghi vào `changelog.md` concept OKF cần làm mới bằng `il-okf-writer` sau khi code đã kiểm chứng (`docs/knowledge/features/lamp.md`, chưa tồn tại)

## Deferred (separate feature, not blocking lamp)

- Thắp đèn tại chùa có phí (v2) — chưa có spec.
- Màn "đèn của tôi" và lịch sử đèn hết hạn — chưa có spec.
- Kiểm duyệt lời nguyện và gỡ đèn qua backoffice — chưa có spec; nếu làm, đi qua `services/backofficeService`.
- Job dọn đèn hết hạn / chính sách retention — chờ chốt marker retention.
