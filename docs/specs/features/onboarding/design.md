# Design — onboarding

Version: 8.4
Last updated: 2026-10-16
Status: implemented (v8.4) — đăng ký xác minh bằng email chạy thật (mailer `log` ở dev); giới hạn theo IP (AC22) đã gắn ở login v2.4; `openapi.yaml` đã cập nhật (v8.3). Còn mở: `erd.svg`, OKF — xem `tasks.md`.
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route public `/v1/auth/register/*` (request-otp, resend-otp, verify-otp), cùng shape Clean
Architecture như v7.5: `RegisterHandler` → `RegisterUsecase` → `OTPService` (dùng chung, xem
[`../otp-mechanism/design.md`](../otp-mechanism/design.md)) + `RegistrationRepository` → Postgres.
v8.0 đổi kênh gửi từ SMS sang email, đổi khoá từ `phone_number` sang `email`, thêm nhánh "email đã có
tài khoản" không lộ thông tin, và thêm giới hạn theo IP (middleware chung của `/v1/auth/*`, xem
[`../login/design.md`](../login/design.md) "Giới hạn tần suất"). Migration: `0037_auth_email_identity.sql`
(dùng chung với `login/` và `otp-mechanism/`).

## Layers & components

| Layer | File | Responsibility | Trạng thái |
|---|---|---|---|
| Validation | `pkg/validation/phone_countries.go` | Bảng 12 nước (ISO, mã quốc gia, số chữ số) và `NormalizePhoneForCountry(country, typed) (e164 string, ok bool)` đúng BR-ONB-04: bỏ khoảng trắng, chấm, gạch; bỏ **một** số 0 đầu; đếm chữ số. | mới |
| Validation | `pkg/validation/email.go` | `NormalizeEmail(raw) (string, bool)`: trim, lower-case, regex của BR-ONB-05, ≤ 254. | mới (thay `validate:"email"` của go-playground ở handler) |
| Domain / Password | `services/appService/domain/password.go` | `ValidatePasswordLength(pw) error` (8–64 rune sau NFC) và interface `PasswordHasher { Hash(pw) (string, error); Compare(hash, pw) error }`. | mới |
| Usecase / Password | `services/appService/usecase/password_hasher.go` | `bcrypt(base64(SHA-256(NFC(pw))))`, `bcrypt.DefaultCost`. Dùng chung với login và đặt lại mật khẩu. | mới |
| Domain / Entity | `services/appService/domain/pending_registration.go` | `PendingRegistration`: thêm `AccountExists bool`; `Email` là khoá. | sửa |
| Domain / Input | `services/appService/domain/register.go` | `RequestRegisterOTPInput` thêm `PhoneCountry`; `ResendOTPInput{Email}`; `VerifyRegisterOTPInput{Email, OTPCode}`. Bỏ interface `OTPSender`. | sửa |
| Domain / Mailer | `services/appService/domain/auth_mailer.go` | `AuthMailer { SendRegisterCode(ctx, to, code, ttl); SendAccountExistsNotice(ctx, to); SendPasswordResetCode(ctx, to, code, ttl) }`. | mới |
| Domain / Errors | `services/appService/domain/errors.go` | Thêm `ErrEmailSendFailed`; bỏ `ErrSMSSendFailed`, `ErrEmailTaken` (không còn ai trả). | sửa |
| Domain / Repository | `services/appService/domain/register.go` | `RegistrationRepository`: `PhoneTaken(ctx, phone) bool`, `EmailHasAccount(ctx, email) bool`, `SavePendingAndIssueOTP(ctx, pending, otpHash, expiresAt, limits)` (khoá email), `FindPendingRegistration(ctx, email)`, `CreateUserAndSession(ctx, pending, otpID, issuer)`. Bỏ `FindConflicts`. | sửa |
| Usecase | `services/appService/usecase/register_usecase.go` | Step R/S/V bên dưới. | sửa |
| Repository / Mailer | `services/appService/repository/email_auth_mailer.go` | Dựng nội dung thư tiếng Việt (tiêu đề, mã, thời hạn) và gọi `pkg/mailer.Mailer.Send`. | mới |
| Repository | `services/appService/repository/postgres_registration_repository.go` | SQL theo email; `CreateUserAndSession` đặt `email_verified_at = now()`; map lỗi UNIQUE `users.phone_number` → `ErrPhoneTaken`. | sửa |
| Repository | `services/appService/repository/sms_otp_sender.go` (+ test) | **Xoá.** | xoá |
| Package | `pkg/mailer` | Thêm `Send(ctx, Message{To, Subject, Text, HTML})` vào interface; `LogMailer` log thư (kể cả mã, chỉ dev). Giữ hai method cũ cho back office. | sửa |
| Package | `pkg/sms`, `pkg/sms/logsms` | **Xoá** (không còn ai import sau khi bỏ `sms_otp_sender.go`). | xoá |
| Delivery / Handler | `services/appService/delivery/http/register_handler.go` | DTO mới (bảng API ở `requirements.md`); route `resend-otp` chuyển path. | sửa |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | `ErrEmailSendFailed` → `500 EMAIL_SEND_FAILED`; bỏ mapping SMS và `EMAIL_TAKEN`. | sửa |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | Thêm `app.environment`; `otp.ttl: 600s`; `otp.dev_fixed_code`; `mailer.provider`, `mailer.from`; bỏ `sms`. Kiểm AC21 trong `Validate()`. | sửa |
| Wiring | `services/appService/cmd/main.go` | Bỏ SMS; dựng `mailer` + `EmailAuthMailer`; route mới; bọc giới hạn tần suất (login design). | sửa |
| Migration | `migrations/0037_auth_email_identity.sql` | Xem "Migration 0037". | mới |
| ERD | `docs/specs/db/erd.mmd` (+ `erd.svg`) | Cột đổi tên, bảng `auth_rate_limits`. | sửa |
| API spec | `docs/specs/api/openapi.yaml` | Ba operation đăng ký theo bảng API. | sửa |

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Bước 1: request-otp

**Step R-1 — Giới hạn theo IP**
- Trigger: `POST /v1/auth/register/request-otp`.
- API: middleware `authRateLimit("send")` (login design).
- Success: đi tiếp. Failure: `429 RATE_LIMITED` + `retry_after_seconds`, handler không chạy.

**Step R-2 — Decode + validate (không chạm DB)**
- Handler `DecodeJSONBodyStrict`; usecase chạy theo thứ tự: email (AC3) → SĐT theo nước (AC4) →
  mật khẩu (AC5) → họ tên, pháp danh (AC6).
- Failure: `400` với mã của trường đầu tiên sai; không ghi gì.

**Step R-3 — SĐT đã thuộc tài khoản?**
- API: `RegistrationRepository.PhoneTaken(phone)` — `SELECT 1 FROM users WHERE phone_number = $1`.
- Failure: `409 PHONE_TAKEN` (AC7). Chạy trước kiểm email nên không lộ gì về email.

**Step R-4 — Chùa**
- API: `TempleRepository.Exists(id)` khi `ordination_temple_id` khác rỗng.
- Failure: `400 INVALID_TEMPLE` (AC8).

**Step R-5 — Email đã có tài khoản?**
- API: `EmailHasAccount(email)` — `SELECT 1 FROM users WHERE email = $1`.
- Kết quả chỉ chọn nhánh thư ở Step R-8; mọi bước giữa hai nhánh giống nhau.

**Step R-6 — Hash**
- `PasswordHasher.Hash(password)`; `OTPService.Generate()` → `(code, codeHash)`.
- Cả hai nhánh làm cùng công việc (AC2).

**Step R-7 — Ghi pending + mã (một transaction)**
- API: `SavePendingAndIssueOTP(pending{…, AccountExists}, codeHash, expiresAt, limits, canReplace)`;
  `canReplace(existing) bool` do usecase dựng (so SĐT, họ tên, pháp danh, chùa và
  `PasswordHasher.Compare(existing.PasswordHash, password)`); repository không băm/so mật khẩu.
- Trong transaction, theo thứ tự: khoá email (`pg_advisory_xact_lock`) + `SELECT … FOR UPDATE` hàng
  pending → nếu có mã sống (`otp_codes` `(email,'register')`, `used_at IS NULL AND expires_at > now()`) mà
  `canReplace` = false → `409 REGISTRATION_IN_PROGRESS` (AC24), chưa ghi/đếm gì → áp send policy trên
  `otp_send_limits (email, 'register')` → UPSERT `pending_registration` theo email
  (`expires_at = now() + pending_registration_ttl`) → vô hiệu mã đang sống → INSERT `otp_codes`.
  Kiểm tra chạy TRƯỚC send policy nên request bị từ chối không để lại dấu vết và không tốn lượt gửi.
- Failure: `409 REGISTRATION_IN_PROGRESS` (+`retry_after_seconds` = giây tới khi mã sống hết hạn, làm tròn lên,
  tối thiểu 1); `429 OTP_RESEND_TOO_SOON` / `OTP_SEND_LIMITED` (AC9); đều không ghi gì. Hai nhánh
  `account_exists` đều ghi hàng pending nên đều có thể gặp 409 y hệt (AC2).

**Step R-8 — Gửi thư (sau commit)**
- API: `AuthMailer.SendRegisterCode(email, code, ttl)` hoặc, khi `AccountExists`,
  `SendAccountExistsNotice(email)`.
- Failure: `OTPService.InvalidateActive(email, register)` (mã đã commit nhưng chưa tới người dùng; nếu còn sống thì form sửa lại sẽ bị AC24 từ chối), `OTPService.ReleaseSendLimit(email, register)`, rồi `500 EMAIL_SEND_FAILED` (AC10). Lỗi của hai lệnh dọn chỉ được log; người dùng vẫn nhận `EMAIL_SEND_FAILED`.
- Success: `200 {message, expires_in_seconds, resend_after_seconds}`.

## Step-by-step Flow — resend-otp

**Step S-1** — Giới hạn theo IP như R-1.
**Step S-2** — Validate email (AC3).
**Step S-3** — `FindPendingRegistration(email)`; không có hoặc hết hạn → `400 PENDING_REGISTRATION_EXPIRED` (AC12).
**Step S-4** — `OTPService.Issue(email, register)`: áp send policy, vô hiệu mã cũ, INSERT mã mới (một transaction). Failure: `429` như AC9.
**Step S-5** — Gửi thư theo `AccountExists` như R-8. Success: `200` kèm `resend_remaining`.

## Step-by-step Flow — Bước 2: verify-otp

**Step V-1** — Giới hạn theo IP `authRateLimit("verify")`.
**Step V-2** — Validate email, `otp_code` 6 chữ số.
**Step V-3** — `FindPendingRegistration(email)`; không có hoặc hết hạn → `400 PENDING_REGISTRATION_EXPIRED` (AC17).
**Step V-4 — Xác minh mã**
- API: `OTPService.Verify(email, register, code)`: tìm mã đang sống → `ConsumeOTPAttempt` (tăng
  `attempts` nguyên tử, khoá khi chạm `max_attempts`) → so bcrypt.
- Failure: `400 OTP_INVALID` / `400 OTP_EXPIRED` / `429 OTP_LOCKED` (AC15).
- Khi `pending.AccountExists`: usecase coi kết quả là `OTP_INVALID` dù mã đúng (AC16) — thực tế không
  ai có mã này.

**Step V-5 — Tạo tài khoản + phiên (một transaction)**
- API: `CreateUserAndSession(pending, otpID, tokenIssuer)`.
- Trong transaction: INSERT `users` (`email_verified_at = now()`) → INSERT `user_sessions` (token mang
  `sid`) → `UPDATE otp_codes SET used_at = now()` → DELETE `pending_registration`.
- Failure: UNIQUE `users.phone_number` → `409 PHONE_TAKEN`; FK chùa → `400 INVALID_TEMPLE` (AC18);
  khác → `500`. Rollback toàn bộ.
- Success: `201 {access_token, refresh_token, expires_in, user}`.

## Migration 0037

`migrations/0037_auth_email_identity.sql` (dùng chung cho onboarding, login, otp-mechanism):

1. `TRUNCATE otp_codes, otp_send_limits, pending_registration, login_attempts` — dữ liệu tạm, không có
   người dùng thật (ADR-0003).
2. `otp_codes`: bỏ `phone_number`, thêm `email VARCHAR(254) NOT NULL`; index `(email, purpose, created_at DESC)`.
3. `otp_send_limits`: bỏ `phone_number`, thêm `email`; `UNIQUE (email, purpose)`.
4. `pending_registration`: bỏ UNIQUE và index trên `phone_number` (cột vẫn `NOT NULL`); `UNIQUE (email)`;
   thêm `account_exists BOOLEAN NOT NULL DEFAULT false`.
5. `login_attempts`: khoá chính đổi sang `email VARCHAR(254)` (login).
6. `users`: `UPDATE users SET email = lower(btrim(email))`; `CHECK (email = lower(email))`.
7. Bảng `auth_rate_limits` (login design).

Idempotent theo quy ước repo (`IF EXISTS` / `IF NOT EXISTS`).

**Đã làm (v8.1):** file `0037` chỉ chứa bước 1–4 và 6. Bước 5 làm ở `migrations/0038_login_attempts_email.sql` (login Phase 1, 2026-10-16); bước 7 (`auth_rate_limits`) bỏ — login v2.4 đếm giới hạn ở Redis. Bước 1 chỉ TRUNCATE khi `otp_codes.phone_number` còn tồn tại, để `make migrate` (chạy lại mọi file) không xoá mã đang sống. `0001` có guard tương ứng cho index `idx_otp_codes_phone_purpose`.

## Mailer

- `domain.AuthMailer` là cổng duy nhất usecase thấy; `repository/email_auth_mailer.go` dựng thư và
  gọi `pkg/mailer`. Nội dung thư không chứa mật khẩu; thư mã chỉ chứa mã và thời hạn (NFR-ONB-03).
- `mailer.provider`: `log` (dev, mặc định) — chặn khởi động ngoài dev (AC21).
- Ngân sách thư mỗi ngày: decorator ở login design ("Giới hạn tần suất"), bọc quanh `pkg/mailer`.

> ⚠️ Not confirmed: nhà cung cấp email thật (khuyến nghị Resend: miễn phí 3.000 thư/tháng, 100/ngày; hoặc Amazon SES) — chỉ chặn production, cần mua domain gửi thư trước (ADR-0003 rule 7, "Tạm mock").

## Known gaps

- Bảo mật (v8.2, AC24): trước đây `request-otp` ghi đè hàng pending của email bất kỳ, nên kẻ lạ thay được
  mật khẩu/SĐT rồi nạn nhân nhập mã mail tới mình (pre-hijack). Nay chỉ ghi đè khi không có mã sống hoặc cùng
  payload. Còn lại: kẻ lạ vẫn có thể bắt nạn nhân chờ tới `otp.ttl` (tối đa 10 phút) bằng cách gửi trước
  payload của mình; thư mã không mời vẫn có thể gửi tới email của người khác (bị chặn bởi send limit, AC9);
  thư "email đã có tài khoản" không bị ảnh hưởng.

- Thời gian gọi mailer khác nhau giữa thư mã và thư báo là nhỏ (cùng một API) — chấp nhận.
- Hàng hết hạn trong `pending_registration`, `otp_codes`, `auth_rate_limits` chưa có job dọn
  (đã Deferred từ v4.4) — `tasks.md` Deferred.
- Attestation cho app (ADR-0003 rule 8, "where available") chưa gắn vì app chưa gửi attestation —
  `tasks.md` Deferred.
