# Design — password-reset

Version: 1.1
Last updated: 2026-09-25
Status: prospective — chưa có code; mọi file trong bảng Layers đều chưa tồn tại
Module: `identity`
Code: `services/appService/`
Feature folder: `services/appService/` + `docs/specs/features/password-reset/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này phủ **US-003 (quên mật khẩu)** gồm hai route: `request-otp` và `confirm`, qua kênh **SMS OTP**
(`purpose = 'reset_password'`). Các thứ tái dùng (đã có code): `domain.OTPService` (cơ chế OTP generic),
bảng `otp_codes` + `otp_send_limits`, `domain.User`, `validation.NormalizePhone`, `minPasswordLength`
(hằng package-level ở `usecase`), sentinel OTP (`ErrOTPInvalid`/`ErrOTPExpired`/`*OTPLockedError`/
`*OTPResendTooSoonError`/`*OTPSendLimitedError`/`ErrSMSSendFailed`), mapping lỗi ở
`delivery/http/errors.go`.

## Overview

Reset là luồng **hai bước** giống đăng ký, nhưng **không có payload tạm**: không có gì để lưu giữa hai
bước ngoài chính hàng `otp_codes` (new_password đến ở bước confirm). Hai bất biến trung tâm:

1. **Enumeration (AC2):** `request-otp` trả một response duy nhất cho số có/không có tài khoản; và —
   quan trọng hơn body — **send-limit được áp đồng bộ** cho cả hai, nên `429` cũng không tiết lộ trạng
   thái. Số chưa đăng ký **không** nhận SMS nhưng **có** tiêu một lượt gửi.
2. **Reset phải đá mọi thiết bị (AC7):** cập nhật `password_hash` + đánh `used_at` cho OTP + revoke
   **toàn bộ** session của user phải là **một transaction** — không có cửa sổ "mật khẩu đã đổi mà
   session cũ còn sống".

Khác với register, đây là feature **đầu tiên tiêu thụ `OTPService` cho một purpose khác `register`** —
bằng chứng thật rằng refactor `otp-mechanism` đạt mục tiêu tái dùng. Và nó cần **một primitive mới**
`ApplySendLimit` trên `OTPRepository` (xem "Phụ thuộc").

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/password_reset.go` *(mới)* | Input: `RequestPasswordResetOTPInput{PhoneNumber}`, `ConfirmPasswordResetInput{PhoneNumber, OTPCode, NewPassword}`. Không cần result struct (response là message cố định / `{success:true}` — handler viết trực tiếp, không có số liệu để compute). Thuần dữ liệu, không JSON tag. |
| Domain / Repository interface | `services/appService/domain/password_reset.go` *(mới)* | `PasswordResetRepository` — hai method: `FindUserByPhone(ctx, phoneNumber) (*User, error)` (trả `ErrUserNotFound` khi không có; **không** phân biệt SĐT chỉ có `pending_registration`) và `ResetPassword(ctx, phoneNumber, passwordHash, otpID) error` (một tx, xem Flow confirm). |
| Domain / Errors | *(không thêm sentinel mới)* | Tái dùng sentinel OTP + `ErrInvalidPhone`/`ErrInvalidPassword`/`ErrUserNotFound`. Thêm hằng `RevokeReasonPasswordReset = "password_reset"` vào nhóm revoke reason (`domain/login.go`). |
| Domain / Mock | `services/appService/domain/mock_password_reset_repository.go` *(mới)* | Mock tay cho `PasswordResetRepository` (test usecase), cùng kiểu `mock_login_repository.go`. |
| Usecase | `services/appService/usecase/password_reset_usecase.go` *(mới)* | `PasswordResetUsecase` hai method: `RequestPasswordResetOTP` (validate → tra user → `Issue` **hoặc** `ApplySendLimit`) và `ConfirmPasswordReset` (validate → `OTPService.Verify` → hash password → `ResetPassword`). Không sở hữu số liệu policy: nhận `*OTPService` (đã gói `OTPConfig`) + `logging.Logger` qua constructor. Tái dùng `minPasswordLength`. |
| Usecase / Dependency | *(dùng `OTPService`)* | `OTPService.ApplySendLimit(ctx, phone, purpose)` *(thêm — xem Phụ thuộc)* cho nhánh số chưa đăng ký. |
| Repository | `services/appService/repository/postgres_password_reset_repository.go` *(mới)* | Concrete implement `PasswordResetRepository`; `ResetPassword` = một tx. `FindUserByPhone` SQL trùng với login (cùng package, tái dùng `userColumns`/helper scan). |
| Delivery / Handler | `services/appService/delivery/http/password_reset_handler.go` *(mới)* | Hai handler (`RequestPasswordResetOTP`, `ConfirmPasswordReset`), DTO request/response, decode strict (`DecodeJSONBodyStrict`). Không cần `userResponse`. |
| Wiring | `services/appService/cmd/main.go` *(thêm)* | `NewPasswordResetRepository(db)` → `NewPasswordResetUsecase(repo, otpService, logger)` → `NewPasswordResetHandler(...)`; 2 `mux.HandleFunc` (cả hai public, không bọc middleware). |
| Migration | — *(không có)* | `chk_otp_codes_purpose` + `chk_otp_send_limits_purpose` đã gồm `reset_password`; `users.password_hash` + `user_sessions.revoked_reason` đã có. |
| API spec | `docs/specs/api/openapi.yaml` *(thêm 2 path)* | Viết **sau** khi handler chạy thật qua `curl` (tiền lệ US-001/US-002). |

Dependency rule: `delivery → usecase → domain`; `repository → domain`; `domain` không import layer ngoài.
Usecase import `pkg/validation`, `domain`, và `OTPService` (cùng package `usecase`) — như `RegisterUsecase`.

### Phụ thuộc: mở rộng `otp-mechanism` (`ApplySendLimit`)

Nhánh "số chưa đăng ký" (AC2) cần ghi nhận một lượt gửi vào `otp_send_limits` **mà không cấp code**
(không ghi `otp_codes`, không gửi SMS). `OTPRepository` hiện chỉ có `IssueOTP` (apply send-limit **và**
insert OTP trong một tx) — không có primitive "chỉ apply send-limit". Vì vậy thêm:

- `domain.OTPRepository.ApplySendLimit(ctx, phoneNumber, purpose, limits SendLimits) (*SendState, error)` —
  áp chính sách gửi (cooldown / cap / khoá-gửi), trả `*OTPResendTooSoonError` / `*OTPSendLimitedError`
  khi bị từ chối, **không** đụng `otp_codes`. Concrete tái dùng đúng SQL helper `applySendLimits` đã có
  (chia sẻ ở tầng primitive, cùng nguyên tắc `otp-mechanism`).
- `OTPService.ApplySendLimit(ctx, phoneNumber, purpose) error` — bọc repo + `wrapSendError`, để usecase
  chỉ phụ thuộc `OTPService` (không phụ thuộc `OTPRepository` trực tiếp).

Đây là thay đổi **additive** lên feature `otp-mechanism` (thêm method, không đổi hành vi method cũ).
`otp-mechanism` đang `Status: complete`; đổi này phải được ghi ở `otp-mechanism/changelog.md` như một
bổ sung nhỏ (không bump major — không đổi contract hiện có). `ResendOTP` của register không bị ảnh hưởng.

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Request OTP: `POST /v1/auth/password-reset/request-otp`

```
Step 1 — Validate và chuẩn hoá (feature này)
- Trigger: handler nhận POST với body decode được
- Pre-condition: DecodeJSONBodyStrict pass (field lạ bị từ chối, cap 1 MB)
- Call: NormalizePhone → rỗng thì ErrInvalidPhone
    - Success → Step 2
    - Failure → 400 INVALID_PHONE; không chạm DB, không tính lượt gửi (AC3)

Step 2 — Tra user theo SĐT (feature này)
- Trigger: Step 1 pass
- Call: PasswordResetRepository.FindUserByPhone(phone)
    - Success (có hàng) → Step 3 (nhánh "có tài khoản")
    - Success (ErrUserNotFound) → Step 4 (nhánh "không có") — vẫn đi tiếp, không trả lỗi (AC2)
    - Failure (lỗi DB) → wrapped "find user: %w" → 500 INTERNAL_ERROR

Step 3 — Cấp + gửi OTP (feature này, nhánh có tài khoản)
- Trigger: có hàng users
- Call:
    (a) OTPService.EnsureNotLocked(phone, reset_password) → nếu khoá nhập sai đang giữ → *OTPLockedError
        → 429 OTP_LOCKED, dừng (AC4)
    (b) OTPService.Issue(phone, reset_password) → Generate (111111 + bcrypt) → IssueOTP (apply
        send-limit + invalidate OTP cũ + insert mới, MỘT tx) → Send (SMS)
    - Success → 200 { message } (generic)
    - Failure (send-limit/cooldown) → *OTPResendTooSoonError / *OTPSendLimitedError → 429 (AC5)
    - Failure (SMS) → ErrSMSSendFailed → 500 SMS_SEND_FAILED (AC6; hàng OTP đã commit)
    - Failure (DB) → 500
  Confirmed: (b) dùng đúng `Issue` — mỗi lần gọi lại là một lần resend, invalidate OTP cũ.

Step 4 — Chỉ áp send-limit (feature này, nhánh KHÔNG có tài khoản)
- Trigger: Step 2 trả ErrUserNotFound
- Call: OTPService.ApplySendLimit(phone, reset_password) → UPSERT otp_send_limits (cooldown / cap /
        khoá-gửi), KHÔNG ghi otp_codes, KHÔNG gửi SMS
    - Success → 200 { message } (generic — Y HỆT nhánh Step 3)
    - Failure (send-limit/cooldown) → 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED (AC5 — đồng bộ với
        nhánh có tài khoản, nên 429 không tiết lộ trạng thái)
    - Failure (DB) → 500
  Confirmed: không gửi SMS (AC2); body 200 giống hệt Step 3; không cấp code cho số lạ.
```

## Step-by-step Flow — Confirm: `POST /v1/auth/password-reset/confirm`

```
Step 1 — Validate (feature này)
- Trigger: handler nhận POST với body decode được
- Pre-condition: DecodeJSONBodyStrict pass
- Call: NormalizePhone (rỗng → ErrInvalidPhone); len(new_password) < 8 → ErrInvalidPassword
    - Success → Step 2
    - Failure → 400 INVALID_PHONE / INVALID_PASSWORD; không chạm DB (AC10)

Step 2 — Verify OTP (cơ chế dùng chung OTPService)
- Trigger: Step 1 pass
- Call: OTPService.Verify(phone, reset_password, otp_code)
    - Success (đúng) → *OTPCode → Step 3
    - Failure (sai) → ErrOTPInvalid → 400 OTP_INVALID; đếm sai, lần 3 → LockOTP → lần sau 429 (AC8)
    - Failure (không có OTP active: hết hạn / đã dùng / chưa cấp) → ErrOTPExpired → 400 OTP_EXPIRED (AC9)
    - Failure (khoá nhập sai) → *OTPLockedError → 429 OTP_LOCKED (AC8)
  Confirmed: Verify không phân biệt "chưa từng cấp" với "hết hạn/đã dùng" — cùng một mã OTP_EXPIRED (AC9).
  Số chưa đăng ký luôn rơi OTP_EXPIRED (chưa từng có hàng OTP) — cùng đánh đổi đã chấp nhận ở register.

Step 3 — Hash mật khẩu mới (feature này)
- Trigger: Step 2 pass
- Call: bcrypt.GenerateFromPassword(new_password, bcrypt.DefaultCost)
    - Success → Step 4
    - Failure → wrapped "hash password: %w" → 500

Step 4 — Đổi mật khẩu + revoke mọi session (feature này, MỘT transaction)
- Trigger: Step 3 pass
- Call: PasswordResetRepository.ResetPassword(ctx, phone, passwordHash, otp.ID) → BEGIN
        (a) UPDATE users SET password_hash = $2, updated_at = now()
            WHERE phone_number = $1 RETURNING id   -- lấy userID; 0 hàng → ErrUserNotFound
        (b) UPDATE otp_codes SET used_at = now() WHERE id = $3
        (c) UPDATE user_sessions SET is_active = false, is_revoked = true, revoked_at = now(),
            revoked_reason = 'password_reset' WHERE user_id = $userID AND is_revoked = false
        COMMIT
    - Success → 200 {"success": true} (AC7)
    - Failure (ErrUserNotFound — user bị xoá giữa request và confirm) → usecase map sang ErrOTPInvalid
        (400 OTP_INVALID, không lộ); hiếm, chỉ là race xoá tài khoản
    - Failure (DB khác) → wrapped "reset password: %w" → 500; rollback cả (a)-(c)
  Confirmed: (a)-(c) chung MỘT tx — không có cửa sổ "mật khẩu đã đổi, session cũ còn sống" (AC7).
  Revoke mọi session (mọi device + session device-less của US-001) — refresh token chết ngay; access
  token còn ≤ 15 phút (stateless, Known gap như login).
```

## Quyết định

- **Không có bảng `password_reset_tokens`** (spec cũ đề xuất): hàng `otp_codes` đã là trạng thái đủ giữa
  hai bước. Reset bằng OTP dùng đúng cơ chế chung, không cần token dài + hash riêng.
- **Response `request-otp` cố định, không window numbers:** nhánh "không có tài khoản" không cấp code
  nên báo `expires_in_seconds` là nói dối; client lấy cooldown từ `429 retry_after_seconds`.
- **`PasswordResetUsecase` phụ thuộc `OTPService` chứ không phải `OTPRepository`:** method
  `OTPService.ApplySendLimit` mới giữ policy trong cơ chế, usecase chỉ gọi service.
- **Không thêm sentinel/không đổi `errors.go`:** mọi lỗi đã có mapping; `ErrUserNotFound` nội bộ.
- **`ResetPassword` gộp "đổi mật khẩu + mark OTP + revoke sessions" một tx**, không tách `RevokeAllSessionsForUser` (chỉ reset dùng).
- **`minPasswordLength` tái dùng** (hằng package-level `usecase`), không khai báo lại — một định nghĩa.

## Test plan

| AC | Test | Kiểu |
|---|---|---|
| AC1, AC2 | `usecase/password_reset_usecase_test.go`: có tài khoản → gọi `Issue`; không có → gọi `ApplySendLimit` (không `Issue`, không `Send`); hai nhánh trả cùng kết quả rỗng | unit (mock) |
| AC3, AC10 | usecase test: SĐT sai định dạng / password < 8 → không gọi repo | unit (mock) |
| AC4, AC5 | usecase test: `EnsureNotLocked`/`Issue`/`ApplySendLimit` pass-through typed error (OTP_LOCKED / TOO_SOON / SEND_LIMITED) | unit (mock) |
| AC7, AC8, AC9 | usecase test: `Verify` đúng → `ResetPassword` nhận đúng `passwordHash` + `otpID`; sai/expired/locked pass-through | unit (mock) |
| AC7 | `repository/postgres_password_reset_repository_test.go`: `ResetPassword` một tx — đổi hash + mark OTP used + revoke **mọi** session (kể cả device-less), rollback khi issuer/lỗi | unit (testcontainers Postgres) |
| AC2 | repository test: `ApplySendLimit` ghi `otp_send_limits` mà **không** ghi `otp_codes` | unit (testcontainers) |
| — | handler test: success, từng nhánh lỗi, body hỏng/field lạ | unit |
| — | `go test -short ./...` + `go vet ./...` + 2 script kiến trúc | CI |

## Deliberately stubbed for now

| Thứ | Trạng thái | Ghi chú |
|---|---|---|
| `pkg/mailer` | **Có sẵn nhưng KHÔNG wire** | Interface + `LogMailer` tạo từ trước cho US-003 email-based; feature này dùng SMS nên bỏ qua. Giữ cho feature email sau (verify email + reset qua email). |
| `RequireAttestation` | **Không wire** | Chống bot để giai đoạn sau; MVP dựa vào `otp_send_limits`. |
| `resend-otp` purpose=reset_password | **Không wire** | request-otp re-callable = resend. |
| ERD `docs/specs/db/` | Chưa có | Không có schema mới (không migration); không cần ERD mới. |

## Known gaps carried by this feature

- **Access token cũ sống tới hết hạn sau reset.** Revoke session chặn refresh, nhưng access token đã cấp
  (stateless) còn dùng được ≤ 15 phút — cùng gap đã chốt chấp nhận ở `login/` (logout). Không tra
  `user_sessions` mỗi request.
- **Timing nhẹ giữa hai nhánh request-otp.** Nhánh có tài khoản làm thêm bcrypt + insert + send; nhánh
  không có chỉ UPSERT send-limit. Số round-trip DB gần bằng nhau; chấp nhận ở MVP (login cần dummy bcrypt
  vì bcrypt là chi phí chi phối ở đó; ở đây chi phí chi phối là DB).
- **`429 OTP_LOCKED` (nhập sai) chỉ có với số đã đăng ký** — rò rỉ tối thiểu, cùng đánh đổi với register.
- **Reset DoS-được với một SĐT cụ thể.** Ai cũng có thể gọi request-otp 4 lần để khoá gửi 30 phút cho số
  của người khác (đánh đổi của khoá theo tài khoản, cùng login AC7). Giảm nhẹ bằng rate-limit theo IP —
  Deferred.
- **`otp_codes` / `otp_send_limits` không có cron dọn** — tích tụ hàng hết hạn (đã là gap chung từ
  register, chờ scheduler). MVP chấp nhận.
- **TTL OTP 90s hơi ngắn cho reset** — dùng chung `OTPConfig`; mở rộng per-purpose nếu UX đòi hỏi.
