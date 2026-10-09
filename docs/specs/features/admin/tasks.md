# Tasks — admin

Version: 1.0
Last updated: 2026-09-29
Status: implemented (2026-09-29) cho Phase 1–4; Phase 3 thiếu test repository, Phase 4 thiếu `openapi.yaml`, Phase 5 chưa làm E2E, Phase 6 là việc hoãn có chủ đích.
Module: `backoffice`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Chỉ `[x]` những gì đã kiểm chứng tồn tại trong code (đã chạy `go build ./...`, `go vet`, `go test -short ./services/backofficeService/... ./pkg/mailer/...` và chạy thật với Postgres ngày 2026-09-29).

## Phase 1 — Domain

- [x] `Admin`, input/result types, hằng role + `ValidRole`, `MinPasswordLen`, `ExtraMustChangePassword` trong `services/backofficeService/domain/admin.go`
- [x] Sentinel errors (`ErrInvalidCredentials`, `ErrSessionRevoked`, `ErrPasswordExpired`, `ErrCannotResetSelf`, `ErrEmailTaken`, `ErrInviteEmailFailed`, `ErrWrongCurrentPassword`, `ErrWeakPassword`, `ErrInvalidAccountInput`, `ErrAdminNotFound`, `ErrPasswordChangeRequired`)
- [x] Interface `AdminRepository` (9 method) trong `domain/admin.go`
- [x] `domain/mock_admin_repository.go` cho test usecase
- [x] `pkg/mailer`: `SendAccountInviteEmail` trong interface, `LogMailer`, `MockMailer`

## Phase 2 — Usecase

- [x] `usecase/admin_auth_usecase.go`: `Login` (một bcrypt mỗi nhánh, kiểm hạn mật khẩu tạm sau khi khớp), `ChangePassword`, `EnsureBootstrapAdmin`, `ValidateSession`, `issueToken`
- [x] `usecase/admin_user_usecase.go`: `CreateAccount` (bù trừ khi email lỗi), `ResetPassword` (chặn tự reset), mật khẩu tạm random/fixed, `tempExpiry` theo TTL
- [x] Unit tests: `usecase/admin_auth_usecase_test.go`, `usecase/admin_user_usecase_test.go` — table-driven, `testify`, mock tay, không DB thật

## Phase 3 — Repository

- [x] `migrations/0006_admins.sql`, `0008_admin_must_change_password.sql`, `0009_admin_tokens_valid_after.sql`, `0010_admin_temp_password_expiry.sql` (đã áp dụng vào Postgres dev qua `make migrate`)
- [x] `repository/postgres_admin_repository.go`: `findOne`/`FindByEmail`/`FindByID`, `TouchLastLogin`, `UpdatePassword`, `CreateBootstrapIfEmpty`, `CreateAccount`, `ResetTempPassword`, `DeleteByID`, `Create`
- [ ] Repository tests trên Postgres thật: `repository/postgres_admin_repository_test.go` (hiện **chưa có file test nào** trong `services/backofficeService/repository/`; SQL mới chỉ được kiểm bằng curl thủ công)
- [ ] Thêm bảng `admins` vào ERD `docs/specs/db/erd.mmd` (+ `erd.svg`)

## Phase 4 — Delivery

- [x] `delivery/http/admin_auth_handler.go`: DTO + `Login`, `ChangePassword`, `Me`
- [x] `delivery/http/admin_user_handler.go`: DTO + `Create`, `ResetPassword` (không trả mật khẩu)
- [x] `delivery/http/session.go` (`RequireActiveSession`) và `delivery/http/errors.go` (`RequireRole`, `RequireStaff`, `RequireStaffAllowPasswordChange`, map sentinel → envelope)
- [x] `config/config.go` + `config.dev.yaml`: `bootstrap_admin`, `account.fixed_temp_password`, `account.temp_password_ttl`
- [x] `cmd/main.go`: bootstrap khi khởi động, 5 route, middleware `adminOnly`/`staff`/`adminPasswordChange` = JWT → session → role (đã kiểm bằng curl: token cũ 401, viewer tạo user 403, token must-change bị chặn ở route thường)
- [x] Handler tests: `delivery/http/admin_auth_handler_test.go`, `admin_user_handler_test.go` — nối dây middleware như `cmd/main.go`, phủ 401/403/409/502/400
- [ ] Cập nhật `docs/specs/api/openapi.yaml` cho 5 route backoffice (hiện **không có** `/v1/admin/*`)
- [ ] Cập nhật `services/backofficeService/README.md` (đã có mục Bootstrap; thiếu route tạo/reset tài khoản, TTL, `PASSWORD_EXPIRED`)

## Phase 5 — Integration & documentation

- [ ] E2E black-box cho backoffice (binary thật + testcontainers Postgres) phủ: bootstrap, login lần đầu → đổi mật khẩu, token cũ bị thu hồi, viewer tạo user 403, mật khẩu tạm hết hạn → reset → đăng nhập lại (dùng skill `golang-e2e-verification`)
- [ ] `go test -short ./...` và `go vet ./...` sạch trên toàn repo (mới kiểm `./services/backofficeService/...` và `./pkg/mailer/...`)
- [ ] Ghi vào `changelog.md` concept OKF cần `il-okf-writer` làm mới: hiện **chưa có** concept nào cho backoffice admin trong `docs/knowledge/features/` — cần tạo mới sau khi E2E xong

## Phase 6 — Hoãn có chủ đích (chưa làm, không chặn feature hiện tại)

- [ ] Rate limit / khoá brute-force cho login và cho thử `current_password`
- [ ] Refresh token / session / logout từng thiết bị (bảng `admin_sessions`, `jti`)
- [ ] Route liệt kê / sửa role / khoá-mở khoá tài khoản (khi đổi role phải set `tokens_valid_after`)
- [ ] RBAC (quyền chi tiết theo role)
- [ ] Provider email thật thay `LogMailer`; chốt hình thức gửi mật khẩu tạm (⚠️ Not confirmed trong `requirements.md`)
- [ ] Production: thay `MockSecretGetter` bằng `pkg/secrets/gcp|aws`, đặt `bootstrap_admin.enabled: false`, để trống `account.fixed_temp_password`
- [ ] Audit log ghi người tạo/reset tài khoản

## Deferred (separate feature, not blocking admin)

- Quản lý temple/deity qua backoffice — `docs/specs/features/temple/`, `deity/`.
- Đăng nhập người dùng app — `docs/specs/features/login/`.
