# Flow — password-reset

Version: 1.1
Last updated: 2026-09-25
Status: prospective — chưa có code; bản vẽ Mermaid mirror 1:1 các section "Step-by-step Flow" trong `design.md`
Module: `identity`
Code: `services/appService/`
Feature folder: `services/appService/` + `docs/specs/features/password-reset/`

> See `changelog.md` in this folder for the version history of this spec.

<!--
File này là **bản vẽ** (visual) của các section "Step-by-step Flow" trong `design.md`. Nó không thêm
bước mới, không bỏ bước — mỗi mũi tên/message ở đây trace 1:1 về một Step trong `design.md`. Ngược
lại, mỗi Step trong `design.md` phải xuất hiện ở đây. Thấy chỗ lệch là bug của chính file này.

Quy ước:
- Một section `### Sequence — ...` cho MỖI section "Step-by-step Flow — ..." của `design.md` (cùng
  tên, cùng thứ tự).
- `OTPService` là cơ chế dùng chung thuộc feature `otp-mechanism` — bước gọi sang nó được ghi rõ
  bằng `Note` là "ngoài scope feature này", không vẽ nội bộ SQL của nó (đó là việc của spec
  `otp-mechanism`). Chỉ transaction của chính feature này (`ResetPassword`) được vẽ bằng `rect`.
- Nhãn participant/message viết tiếng Việt ngắn; keyword Mermaid giữ tiếng Anh.
-->

## Sequence — Request OTP: `POST /v1/auth/password-reset/request-otp`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as PasswordResetHandler<br/>(delivery/http)
    participant U as PasswordResetUsecase
    participant R as PasswordResetRepository
    participant O as OTPService<br/>(otp-mechanism)
    participant S as pkg/sms
    participant DB as Postgres

    C->>H: POST /v1/auth/password-reset/request-otp {phone_number}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: RequestPasswordResetOTP(ctx, input)

    U->>U: NormalizePhone (Step 1)
    alt SĐT sai định dạng
        U--xH: ErrInvalidPhone
        H--xC: 400 INVALID_PHONE (không chạm DB, không tính lượt gửi)
    end

    U->>R: FindUserByPhone(phone) (Step 2)
    R->>DB: SELECT ... FROM users WHERE phone_number = $1
    alt có hàng users
        R-->>U: *User

        U->>O: EnsureNotLocked(phone, reset_password) (Step 3a)
        Note over U,O: OTPService thuộc feature otp-mechanism, ngoài scope feature này
        alt khoá nhập sai còn giữ
            O--xU: *OTPLockedError
            U--xH: ErrOTPLocked
            H--xC: 429 OTP_LOCKED {locked_until}
        end

        U->>O: Issue(phone, reset_password) (Step 3b)
        Note over O: Generate (111111 + bcrypt) → IssueOTP (MỘT tx: apply send-limit<br/>+ invalidate OTP cũ + INSERT otp_codes) → Send SMS
        O->>S: Send SMS (OTP)
        alt gửi OK
            O-->>U: ok
            U-->>H: nil
            H-->>C: 200 {message} (generic)
        else send-limit / cooldown
            O--xU: *OTPResendTooSoonError / *OTPSendLimitedError
            U--xH: lỗi typed
            H--xC: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
        else SMS thất bại
            O--xU: ErrSMSSendFailed
            U--xH: lỗi
            H--xC: 500 SMS_SEND_FAILED (hàng OTP đã commit)
        end
    else ErrUserNotFound (số chưa đăng ký)
        R-->>U: ErrUserNotFound (không trả lỗi, chuyển Step 4)

        U->>O: ApplySendLimit(phone, reset_password) (Step 4)
        Note over O: UPSERT otp_send_limits — KHÔNG ghi otp_codes, KHÔNG gửi SMS
        alt trong giới hạn
            O-->>U: ok
            U-->>H: nil
            H-->>C: 200 {message} (generic, giống nhánh có tài khoản)
        else send-limit / cooldown
            O--xU: *OTPResendTooSoonError / *OTPSendLimitedError
            U--xH: lỗi typed
            H--xC: 429 (đồng bộ với nhánh có tài khoản — không lộ trạng thái)
        end
    else lỗi DB
        R--xU: wrapped "find user: %w"
        U--xH: lỗi
        H--xC: 500 INTERNAL_ERROR
    end
```

## Sequence — Confirm: `POST /v1/auth/password-reset/confirm`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as PasswordResetHandler<br/>(delivery/http)
    participant U as PasswordResetUsecase
    participant O as OTPService<br/>(otp-mechanism)
    participant R as PasswordResetRepository
    participant DB as Postgres

    C->>H: POST /v1/auth/password-reset/confirm {phone_number, otp_code, new_password}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: ConfirmPasswordReset(ctx, input)

    U->>U: NormalizePhone + len(new_password) >= 8 (Step 1)
    alt SĐT sai / password < 8
        U--xH: ErrInvalidPhone / ErrInvalidPassword
        H--xC: 400 INVALID_PHONE / INVALID_PASSWORD (không chạm DB)
    end

    U->>O: Verify(phone, reset_password, otp_code) (Step 2)
    Note over U,O: OTPService thuộc feature otp-mechanism, ngoài scope feature này
    alt OTP đúng, còn hạn, chưa dùng
        O-->>U: *OTPCode
    else sai mã (đếm sai, lần 3 khoá)
        Note over O: attempts+1, hoặc locked_until = now()+30m
        O--xU: ErrOTPInvalid / *OTPLockedError
        U--xH: lỗi
        H--xC: 400 OTP_INVALID / 429 OTP_LOCKED
    else hết hạn / đã dùng / chưa từng cấp
        O--xU: ErrOTPExpired
        U--xH: lỗi
        H--xC: 400 OTP_EXPIRED (một mã cho cả ba trạng thái)
    end

    U->>U: bcrypt.GenerateFromPassword(new_password, DefaultCost) (Step 3)
    Note over U: lỗi hash → wrapped "hash password: %w" → 500 (hiếm, không chạm DB)

    U->>R: ResetPassword(ctx, phone, passwordHash, otp.ID) (Step 4)
    alt thành công
        rect rgb(235, 240, 255)
            Note over R,DB: MỘT transaction
            R->>DB: (a) UPDATE users SET password_hash ... RETURNING id
            R->>DB: (b) UPDATE otp_codes SET used_at = now()
            R->>DB: (c) UPDATE user_sessions ... revoked_reason='password_reset' (revoke MỌI session)
        end
        R-->>U: ok
        U-->>H: nil
        H-->>C: 200 {"success": true}
    else 0 hàng users — user bị xoá giữa request và confirm
        Note over R,DB: rollback (a)-(c)
        R--xU: ErrUserNotFound
        U->>U: map ErrUserNotFound → ErrOTPInvalid
        U--xH: ErrOTPInvalid
        H--xC: 400 OTP_INVALID (không lộ)
    else lỗi DB khác
        Note over R,DB: rollback (a)-(c)
        R--xU: wrapped "reset password: %w"
        U--xH: lỗi
        H--xC: 500 INTERNAL_ERROR
    end
```
