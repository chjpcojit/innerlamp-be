# Tasks — login

Version: 2.0
Last updated: 2026-10-14
Status: v2.0 chưa bắt đầu — mọi task `[ ]`. Task của v1.x (login/refresh/logout bằng SĐT) đã xong, xem `changelog.md` và git history.
Module: `identity` (spec folder `login`)

> See `changelog.md` in this folder for the version history of this spec.

Mỗi item trace về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Làm sau
`onboarding/tasks.md` Phase 1–3 (luật email/mật khẩu, migration `0037`, mailer) và
`otp-mechanism/tasks.md`.

## Phase 1 — Đăng nhập bằng email

- [ ] `domain/login.go`: `Email`, `FindUserByEmail`, `ReserveLoginAttempt`, `ResetLoginAttempts`.
- [ ] `postgres_login_repository.go`: SQL theo email; `ReserveLoginAttempt` một UPSERT (Step L-3); bỏ `FindLoginAttempt`/`RecordLoginFailure`.
- [ ] Test repository trên Postgres thật: 10 request đồng thời sai mật khẩu chỉ cho tối đa 5 lần so (AC5).
- [ ] `login_usecase.go`: Step L-1…L-6; `email_verified_at IS NULL` → `ErrInvalidCredentials`; `PasswordHasher`.
- [ ] `login_handler.go`: DTO `email`, thông điệp "Email hoặc mật khẩu không đúng".
- [ ] Chạy lại test refresh/logout hiện có (AC9–AC15) sau khi đổi khoá.

## Phase 2 — Kiểm phiên còn sống

- [ ] `LoginRepository.IsSessionActive` + test.
- [ ] `delivery/http/active_session.go` + test (`401` phiên thu hồi, `503` lỗi DB).
- [ ] Bọc `activeSession` sau mọi `authMiddleware.AuthenticateFunc` (`cmd/main.go`, `cmd/profile_routes.go`, `cmd/temple_announcement_routes.go`); test đảm bảo không route Bearer nào sót.

## Phase 3 — Quên mật khẩu

- [ ] `domain/password_reset.go`, `RevokeReasonPasswordReset`.
- [ ] `postgres_password_reset_repository.go`: `FindVerifiedUserByEmail`, `ResetPassword` một transaction (Step C-4) + test trên Postgres thật.
- [ ] `password_reset_usecase.go`: Step P-1…P-5, C-1…C-4; thời gian tối thiểu với đồng hồ giả trong test.
- [ ] `password_reset_handler.go` + route trong `cmd/main.go`; `no-store`.
- [ ] Test usecase/handler cho AC23–AC30.

## Phase 4 — (chờ trả lời) Gỡ khoá đăng nhập sau khi đặt lại

- [ ] Phụ thuộc `⚠️ Not confirmed (product)` sau AC30: nếu "gỡ", thêm `DELETE FROM login_attempts WHERE email = $1` vào transaction C-4 + test.

## Phase 5 — Giới hạn tần suất

- [ ] Migration `0037` bước 7: bảng `auth_rate_limits`.
- [ ] `postgres_rate_limit_store.go` (`Hit`) + test đồng thời trên Postgres thật.
- [ ] `delivery/http/auth_rate_limit.go` + test (IP theo `trust_forwarded_for`, `429` + `retry_after_seconds`).
- [ ] `repository/budgeted_mailer.go` + test (vượt `mailer.daily_budget` → `ErrRateLimited`).
- [ ] Config `auth_rate_limit.*`, `mailer.daily_budget`, `password_reset.min_response_time` + `Validate()`.
- [ ] Bọc các route `/v1/auth/*` theo nhóm `send` / `verify` / `login` / `refresh` (AC31).

## Phase 6 — E2E

- [ ] `e2e/login_e2e_test.go` viết lại theo email: sai thông tin đồng nhất, khoá theo email, email chưa xác minh không đăng nhập được.
- [ ] E2E quên mật khẩu: đặt lại → thiết bị khác nhận `401` ở request kế tiếp; email không có tài khoản trả y hệt và không có thư.
- [ ] E2E giới hạn tần suất: vượt ngân sách → `429 RATE_LIMITED`.
- [ ] `go build ./... && go vet ./... && go test -short ./...` xanh; `go test ./services/appService/e2e/` xanh.

## Phase 7 — Tài liệu theo sau

- [ ] `docs/specs/api/openapi.yaml`: năm operation + `401` mới trên route Bearer (sau khi handler chạy thật).
- [ ] `docs/knowledge/features/login.md`: làm mới bằng `il-okf-writer`.
- [ ] Product spec §4.2: cột Backend sang `done` (báo `ba-writer`).

## Deferred

- [ ] Job dọn hàng cũ của `auth_rate_limits` (chung với onboarding Deferred).
- [ ] Khoá tăng dần, trần tuyệt đối của phiên (giữ từ v1.x).
