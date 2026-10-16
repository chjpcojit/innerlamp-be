# Tasks — login

Version: 2.6
Last updated: 2026-10-16
Status: Phase 1–6 xong (2026-10-16); Phase 7: `openapi.yaml` xong (v2.6), còn OKF và product spec §4.2. Task của v1.x (login/refresh/logout bằng SĐT) đã xong, xem `changelog.md` và git history.
Module: `identity` (spec folder `login`)

> See `changelog.md` in this folder for the version history of this spec.

Mỗi item trace về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Làm sau
`onboarding/tasks.md` Phase 1–3 (luật email/mật khẩu, migration `0037`, mailer) và
`otp-mechanism/tasks.md`.

## Phase 1 — Đăng nhập bằng email

- [x] Migration: bước 5 của onboarding "Migration 0037" (`login_attempts` khoá theo `email`) — tách khỏi `0037` ở onboarding v8.1, viết thành `migrations/0038_login_attempts_email.sql` (TRUNCATE có guard, PK `email`).
- [x] `domain/login.go`: `Email`, `FindUserByEmail`, `ReserveLoginAttempt` (`LoginAttempt.AlreadyLocked`). Không có `ResetLoginAttempts` riêng: Step L-5 là `DELETE FROM login_attempts WHERE email` trong transaction của `CreateLoginSession` (như v1.13), để login rollback không xoá bộ đếm.
- [x] `postgres_login_repository.go`: SQL theo email; `ReserveLoginAttempt` một UPSERT (Step L-3); bỏ `FindLoginAttempt`/`RecordLoginFailure`.
- [x] Test repository trên Postgres thật: 10 request đồng thời sai mật khẩu chỉ cho tối đa 5 lần so (AC5).
- [x] `login_usecase.go`: Step L-2…L-6 (L-1 là middleware, Phase 5); `email_verified_at IS NULL` → `ErrInvalidCredentials`; `PasswordHasher`.
- [x] `login_handler.go`: DTO `email`, thông điệp "Email hoặc mật khẩu không đúng".
- [x] Chạy lại test refresh/logout hiện có (AC9–AC15) sau khi đổi khoá.

## Phase 2 — Kiểm phiên còn sống

- [x] `LoginRepository.IsSessionActive` + test.
- [x] `delivery/http/active_session.go` + test (`401` phiên thu hồi, `503` lỗi DB).
- [x] Bọc `activeSession` sau mọi `authMiddleware.AuthenticateFunc`: `cmd/main.go` dựng `authed` và truyền cho `registerProfileRoutes`/`registerTempleAnnouncementRoutes` (hai file đó không tự dựng middleware). Ngoại lệ có chủ ý: `POST /v1/auth/logout` (idempotent, AC14). Test `cmd/auth_wiring_test.go` bắt route sót.

## Phase 3 — Quên mật khẩu

- [x] `domain/password_reset.go`, `RevokeReasonPasswordReset`.
- [x] `postgres_password_reset_repository.go`: `FindVerifiedUserByEmail`, `ResetPassword` một transaction (Step C-4) + test trên Postgres thật.
- [x] `password_reset_usecase.go`: Step P-1…P-5, C-1…C-4; thời gian tối thiểu với đồng hồ giả trong test.
- [x] `password_reset_handler.go` + route trong `cmd/main.go`; `no-store`.
- [x] Test usecase/handler cho AC23–AC30.

## Phase 4 — Gỡ khoá đăng nhập sau khi đặt lại

- [x] `ResetPassword` thêm `DELETE FROM login_attempts WHERE email = $2` trong transaction C-4 + test: email đang khoá, đặt lại xong đăng nhập bằng mật khẩu mới được ngay (AC28).

## Phase 5 — Giới hạn tần suất

- [x] ~~Migration `0037` bước 7: bảng `auth_rate_limits`.~~ **Bỏ** (quyết định 2026-10-16: đếm ở Redis, store của commit fc31897). Không có bảng.
- [x] ~~`postgres_rate_limit_store.go`~~ **Bỏ**: dùng `ratelimit.Store` dùng chung (`cmd/ratelimit_store.go`, Redis, fallback memory).
- [x] Giới hạn theo IP: `ClientIPRateLimit` có sẵn (`delivery/http/client_ip_rate_limit.go`) thay cho `auth_rate_limit.go`; test `429` + `retry_after_seconds` + IP theo AC34 ở `client_ip_rate_limit_test.go`.
- [x] `RequireWebAPIKey` gắn cờ "key hợp lệ" vào context; `clientIPFor` + test: key hợp lệ + `X-InnerLamp-Client-IP` hợp lệ → dùng header; không có key → bỏ qua header; giá trị sai → rơi về XFF/kết nối. `clientIP` của `login_handler.go` (metadata thiết bị) dùng cùng hàm (AC34).
- [x] `repository/budgeted_mailer.go` (decorator quanh `domain.AuthMailer`, đếm theo ngày UTC trong store dùng chung) + test (vượt `mailer.daily_budget` → `ErrRateLimited`).
- [x] Config `auth_rate_limit.*`, `mailer.daily_budget` + `Validate()` (`password_reset.min_response_time` đã thêm ở Phase 3: `config.PasswordResetConfig`).
- [x] Bọc các route `/v1/auth/*` theo nhóm `send` / `verify` / `login` / `refresh` (AC31).

## Phase 6 — E2E

- [x] `e2e/login_e2e_test.go` viết lại theo email: sai thông tin đồng nhất, khoá theo email, email chưa xác minh không đăng nhập được.
- [x] E2E quên mật khẩu: đặt lại → thiết bị khác nhận `401` ở request kế tiếp; email không có tài khoản trả y hệt và không có thư.
- [x] E2E giới hạn tần suất: vượt ngân sách → `429 RATE_LIMITED`.
- [x] `go build ./... && go vet ./... && go test -short ./...` xanh; `go test ./services/appService/e2e/` xanh.

## Phase 7 — Tài liệu theo sau

- [x] `docs/specs/api/openapi.yaml`: năm operation + `401` mới trên route Bearer (sau khi handler chạy thật). Làm ở v2.5: năm operation đầy đủ; `401`/`503` của kiểm phiên ghi ở mô tả `bearerAuth` và tag `Auth`, không lặp ở từng route Bearer vì các route đó đã liệt kê `401 UNAUTHORIZED`.
- [ ] `docs/knowledge/features/login.md`: làm mới bằng `il-okf-writer`.
- [ ] Product spec §4.2: cột Backend sang `done` (báo `ba-writer`).

## Deferred

- [x] ~~Job dọn hàng cũ của `auth_rate_limits`~~ — không còn bảng (đếm ở Redis; key hết hạn theo TTL, memory store: `docs/tech-debt/ratelimit/TD-002`).
- [ ] Khoá tăng dần, trần tuyệt đối của phiên (giữ từ v1.x).
