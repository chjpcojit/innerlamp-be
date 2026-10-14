# Tasks — onboarding

Version: 8.0
Last updated: 2026-10-14
Status: v8.0 chưa bắt đầu — mọi task `[ ]`. Task của v1–v7.5 đã xong và nằm trong git history / `changelog.md`.
Module: `identity` (spec folder `onboarding`)

> See `changelog.md` in this folder for the version history of this spec.

Mỗi item trace về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Migration
`0037`, `OTPService` và giới hạn tần suất là việc chung với `login/` và `otp-mechanism/`; làm một lần,
tick ở cả ba spec.

## Phase 1 — Luật dùng chung (`pkg/validation`, password)

- [ ] `pkg/validation/phone_countries.go`: bảng 12 nước + `NormalizePhoneForCountry`; test chạy toàn bộ `docs/product/onboarding/rules/phone-number.json`.
- [ ] `pkg/validation/email.go`: `NormalizeEmail`; test chạy `rules/email.json`.
- [ ] `domain/password.go` + `usecase/password_hasher.go`: độ dài 8–64 rune sau NFC, hash SHA-256 → base64 → bcrypt; test chạy `rules/password.json` và một ca 64 ký tự có dấu (> 72 byte) để chứng minh ký tự cuối có tính.
- [ ] Họ tên / pháp danh: test usecase chạy `rules/names.json` (logic đã có từ v7.5).

## Phase 2 — Migration 0037

- [ ] `migrations/0037_auth_email_identity.sql` theo `design.md` "Migration 0037" (bước 1–6; bước 7 thuộc login Phase 5).
- [ ] Chạy hai lần liên tiếp trên Postgres thật để chứng minh idempotent.
- [ ] `docs/specs/db/erd.mmd` + render lại `erd.svg`.

## Phase 3 — Mailer

- [ ] `pkg/mailer`: thêm `Send(ctx, Message)`; `LogMailer` + `MockMailer` cài đặt; README cập nhật.
- [ ] `domain/auth_mailer.go` + `repository/email_auth_mailer.go` (nội dung thư tiếng Việt, không chứa mật khẩu); test nội dung.
- [ ] Config `mailer.provider`, `mailer.from`; `app.environment`; `Validate()` chặn `log` ngoài dev (AC21).

## Phase 4 — Bỏ SMS

- [ ] Xoá `repository/sms_otp_sender.go` (+ test), `domain.OTPSender` + mock, `pkg/sms`, `pkg/sms/logsms`, `config.SMSConfig`, khối SMS ở `cmd/main.go`.
- [ ] `ErrSMSSendFailed` → `ErrEmailSendFailed`; `errors.go` map `500 EMAIL_SEND_FAILED`; bỏ `EMAIL_TAKEN`.
- [ ] `grep -rn "sms" --include=*.go .` chỉ còn comment lịch sử (hoặc không còn).

## Phase 5 — Repository

- [ ] `postgres_registration_repository.go`: `PhoneTaken`, `EmailHasAccount`, `SavePendingAndIssueOTP` theo email + `account_exists`, `FindPendingRegistration(email)`, `CreateUserAndSession` đặt `email_verified_at = now()`, UNIQUE SĐT → `ErrPhoneTaken`.
- [ ] Test repository trên Postgres thật (testcontainers): ghi đè pending cùng email; hai pending cùng SĐT; xung đột SĐT lúc tạo tài khoản rollback sạch.

## Phase 6 — Usecase

- [ ] `RegisterUsecase.RequestOTP` theo Step R-2…R-8, cả hai nhánh làm cùng công việc (AC2).
- [ ] `ResendOTP` theo Step S-2…S-5.
- [ ] `VerifyOTP` theo Step V-2…V-5; `AccountExists` → `ErrOTPInvalid` (AC16).
- [ ] Test usecase cho AC1–AC20 (mock repository, mock mailer, đồng hồ giả).

## Phase 7 — Delivery + wiring

- [ ] `register_handler.go`: DTO mới; route `POST /v1/auth/register/resend-otp`, bỏ `POST /v1/auth/resend-otp`.
- [ ] `cmd/main.go`: dựng mailer, bọc `authRateLimit` (login Phase 5) cho ba route.
- [ ] Test handler: mã lỗi + shape response theo bảng API.

## Phase 8 — E2E

- [ ] `e2e/register_e2e_test.go` viết lại: đăng ký bằng email (mã đọc từ `MockMailer`/log), email đã có tài khoản trả y hệt và không tạo được tài khoản, SĐT trùng `409`, pending hết hạn, khoá sau 3 lần sai.
- [ ] `go build ./... && go vet ./... && go test -short ./...` xanh; `go test ./services/appService/e2e/` xanh.

## Phase 9 — Tài liệu theo sau

- [ ] `docs/specs/api/openapi.yaml`: ba operation đăng ký theo bảng API (viết sau khi handler chạy thật, theo convention repo).
- [ ] `docs/knowledge/features/register.md`: làm mới bằng `il-okf-writer` sau khi code xong.
- [ ] Product spec §4.2: đổi cột Backend của các ID trong `Implements` sang `done` (báo `ba-writer`).

## Deferred

- [ ] Job dọn hàng hết hạn (`pending_registration`, `otp_codes`, `auth_rate_limits`).
- [ ] Gắn `RequireAttestation` cho caller mobile ở `request-otp` khi iOS/Android gửi attestation (ADR-0003 rule 8).
- [ ] Driver email thật khi có domain (`design.md` "Mailer", ⚠️ Not confirmed).
