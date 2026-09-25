# Flow — onboarding

Version: 6.1
Last updated: 2026-09-25
Status: implemented (US-001) — bản vẽ Mermaid mirror 1:1 ba section "Step-by-step Flow" của `design.md`
Module: `identity`
Code: `services/appService/`
Feature folder: `services/appService/` + `docs/specs/features/onboarding/`

> See `changelog.md` in this folder for the version history of this spec.

<!--
File này là **bản vẽ** (visual) của các section "Step-by-step Flow" trong `design.md`. Nó không thêm
bước mới, không bỏ bước — mỗi mũi tên/message ở đây trace 1:1 về một Step trong `design.md`. Ngược
lại, mỗi Step trong `design.md` phải xuất hiện ở đây. Thấy chỗ lệch là bug của chính file này.

Quy ước:
- Một section `### Sequence — ...` cho MỖI section "Step-by-step Flow — ..." của `design.md` (cùng
  tên, cùng thứ tự). Feature này có **ba** flow: request-otp, verify-otp, resend-otp.
- Transaction của repository được bọc bằng `rect` + `Note over R,DB: MỘT transaction`.
- Bước gửi SMS ghi `Note` "thuộc pkg/sms, ngoài scope feature này" (impl log-only) — không vẽ nội bộ
  của `pkg/sms`.
- Nhãn participant/message viết tiếng Việt ngắn; keyword Mermaid giữ tiếng Anh.
-->

## Sequence — Đăng ký bước 1: yêu cầu OTP (`POST /v1/auth/register/request-otp`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as RegisterHandler<br/>(delivery/http)
    participant U as RegisterUsecase
    participant R as RegistrationRepository
    participant DB as Postgres
    participant S as pkg/sms

    C->>H: POST /v1/auth/register/request-otp {phone_number, email, password, dharma_name?, ordination_place?}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: RequestOTP(ctx, input)

    U->>U: NormalizeEmail + validate email + NormalizePhone + len(password) >= 8 (Step 1)
    alt sai email / phone / password
        U--xH: ErrInvalidEmail / ErrInvalidPhone / ErrInvalidPassword
        H--xC: 400 INVALID_EMAIL / INVALID_PHONE / INVALID_PASSWORD (không chạm DB)
    end

    U->>R: FindConflicts(phone, email) (Step 2)
    R->>DB: 1 query quét users.phone_number ∪ users.email ∪ pending_registration.email (SĐT khác, còn hạn)
    alt trùng SĐT (đã có tài khoản)
        R--xU: ErrPhoneTaken
        U--xH: ErrPhoneTaken
        H--xC: 409 PHONE_TAKEN
    else trùng email (tài khoản, hoặc pending của SĐT khác)
        R--xU: ErrEmailTaken
        U--xH: ErrEmailTaken
        H--xC: 409 EMAIL_TAKEN
    end

    U->>U: bcrypt.GenerateFromPassword (Step 3)
    Note over U: lỗi hash → wrapped "hash password: %w" → 500 (hiếm, không chạm DB)

    U->>R: SavePendingAndIssueOTP(ctx, pending, otpHash, expiresAt, limits) (Step 4)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>DB: (a) INSERT otp_send_limits ON CONFLICT DO NOTHING + SELECT ... FOR UPDATE
        R->>DB: (b) áp send-limit (cooldown 60s / trần 4 lượt / khoá-gửi 30 phút)
        R->>DB: (c) UPSERT pending_registration (theo phone_number, ghi đè payload)
        R->>DB: (d) UPDATE otp_codes SET used_at (vô hiệu OTP cũ)
        R->>DB: (e) INSERT otp_codes (mới, expires_at = now() + 90s)
        alt bị chặn (cooldown / trần / khoá-nhập-sai)
            Note over R,DB: rollback (a)-(e)
            R--xU: *OTPResendTooSoonError / *OTPSendLimitedError / *OTPLockedError
            U--xH: typed error
            H--xC: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED / OTP_LOCKED
        else lỗi DB khác
            Note over R,DB: rollback (a)-(e)
            R--xU: wrapped "save pending registration: %w"
            U--xH: lỗi
            H--xC: 500 INTERNAL_ERROR
        end
    end

    U->>S: SendOTP(phone, otpCode, OTPPurposeRegister) (Step 5)
    Note over U,S: thuộc pkg/sms (qua adapter sms_otp_sender), ngoài scope feature này — impl log-only
    alt gửi OK
        S-->>U: nil
        U-->>H: result
        H-->>C: 200 {message, expires_in_seconds: 90, resend_after_seconds: 60}
    else SMS thất bại
        S--xU: ErrSMSSendFailed
        U--xH: ErrSMSSendFailed
        H--xC: 500 SMS_SEND_FAILED (hàng OTP đã commit — KHÔNG rollback Step 4)
    end

    Note over R,DB: Step 6 — dọn dẹp định kỳ (ngoài request path, Deferred — chưa có scheduler)
```

## Sequence — Đăng ký bước 2: xác thực OTP và tạo tài khoản (`POST /v1/auth/register/verify-otp`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as RegisterHandler<br/>(delivery/http)
    participant U as RegisterUsecase
    participant R as RegistrationRepository
    participant DB as Postgres

    C->>H: POST /v1/auth/register/verify-otp {phone_number, otp_code}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: VerifyOTP(ctx, input)

    U->>U: NormalizePhone + validate otp_code (len=6, numeric) (Step 1)
    alt sai field
        U--xH: validation error
        H--xC: 400 VALIDATION_ERROR (không chạm DB)
    end

    U->>R: FindPendingRegistration(phone) (Step 2)
    alt còn hạn
        R-->>U: pending
    else không có hàng
        R--xU: ErrPendingRegistrationNotFound
        U--xH: ErrPendingRegistrationNotFound
        H--xC: 400 OTP_INVALID (không trả 404 — tránh dò SĐT đang có lượt dở)
    else hết hạn
        R--xU: ErrPendingRegistrationExpired
        U--xH: ErrPendingRegistrationExpired
        H--xC: 400 PENDING_REGISTRATION_EXPIRED
    end

    U->>R: FindActiveOTP(phone, 'register') (Step 3)
    alt hàng OTP hiệu lực, chưa khoá
        R-->>U: otp
    else khoá-nhập-sai còn giữ
        R--xU: ErrOTPLocked
        U--xH: ErrOTPLocked
        H--xC: 429 OTP_LOCKED {locked_until}
    else không có hàng hiệu lực
        R--xU: ErrOTPExpired
        U--xH: ErrOTPExpired
        H--xC: 400 OTP_EXPIRED
    end

    U->>U: bcrypt.CompareHashAndPassword(otp_hash, otp_code) (Step 4)
    alt khớp
        U->>R: CreateUserAndSession(ctx, pending, issuer) (Step 5-6)
        rect rgb(235, 240, 255)
            Note over R,DB: MỘT transaction
            R->>DB: (a) SELECT 1 FROM users WHERE email = $1 (re-check trong tx)
            R->>DB: (b) INSERT INTO users (...) RETURNING id (email_verified_at = NULL)
            R->>DB: (c) DELETE FROM pending_registration WHERE id = $1
            R->>DB: (d) UPDATE otp_codes SET used_at = now()
            R->>DB: (e) INSERT INTO user_sessions (...) + mint JWT (closure TokenIssuer: access mang sid, refresh hash SHA-256)
            alt unique violation email/phone (23505)
                Note over R,DB: rollback (a)-(e)
                R--xU: ErrEmailTaken / ErrPhoneTaken
                U--xH: ErrEmailTaken / ErrPhoneTaken
                H--xC: 409 EMAIL_TAKEN / PHONE_TAKEN
            else lỗi mint token / insert session / DB khác
                Note over R,DB: rollback (a)-(e)
                R--xU: wrapped "issue tokens / insert session / create user: %w"
                U--xH: lỗi
                H--xC: 500 INTERNAL_ERROR
            end
        end
        R-->>U: RegisterResult{access_token, refresh_token, user}
        U-->>H: RegisterResult
        H-->>C: 201 {access_token, refresh_token, user}
    else sai mã
        U->>R: IncrementOTPAttempts / LockOTP (lần 3 → khoá 30 phút) (Step 4)
        R->>DB: attempts+1, hoặc locked_until = now()+30m + used_at = now()
        U--xH: ErrOTPInvalid (không trả số lần còn lại)
        H--xC: 400 OTP_INVALID
    end
```

## Sequence — Gửi lại OTP: `POST /v1/auth/resend-otp`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as RegisterHandler<br/>(delivery/http)
    participant U as RegisterUsecase
    participant R as RegistrationRepository
    participant DB as Postgres
    participant S as pkg/sms

    C->>H: POST /v1/auth/resend-otp {phone_number, purpose}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: ResendOTP(ctx, input)

    U->>U: NormalizePhone + validate purpose (oneof) (Step 1)
    Note over U: purpose != 'register' → ErrUnsupportedOTPPurpose → 400 (ngoài scope US-001)
    alt sai field
        U--xH: ErrInvalidPhone / validation error
        H--xC: 400 INVALID_PHONE / VALIDATION_ERROR (không chạm DB)
    end

    U->>R: FindPendingRegistration(phone) (Step 2 — chỉ nhánh purpose='register')
    alt còn hạn
        R-->>U: pending
    else không có hàng
        R--xU: ErrPendingRegistrationNotFound
        U--xH: ErrPendingRegistrationNotFound
        H--xC: 400 OTP_INVALID
    else hết hạn (quá 24h)
        R--xU: ErrPendingRegistrationExpired
        U--xH: ErrPendingRegistrationExpired
        H--xC: 400 PENDING_REGISTRATION_EXPIRED
    end

    U->>R: FindActiveOTP(phone, purpose) (Step 3 — chỉ kiểm khoá-nhập-sai)
    alt khoá-nhập-sai còn giữ
        R--xU: ErrOTPLocked
        U--xH: ErrOTPLocked
        H--xC: 429 OTP_LOCKED {locked_until}
    end

    U->>R: ReissueOTP(ctx, phone, purpose, otpHash, expiresAt, limits) (Step 4)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>DB: (a) INSERT otp_send_limits ON CONFLICT DO NOTHING + SELECT ... FOR UPDATE
        R->>DB: (b) áp send-limit (cooldown 60s / trần 4 lượt / khoá-gửi 30 phút)
        R->>DB: (c) UPDATE otp_codes SET used_at (vô hiệu OTP cũ)
        R->>DB: (d) INSERT otp_codes (mới, expires_at = now() + 90s; KHÔNG đụng pending_registration)
        alt bị chặn (cooldown / trần / khoá-gửi)
            Note over R,DB: rollback (a)-(d)
            R--xU: *OTPResendTooSoonError / *OTPSendLimitedError
            U--xH: typed error
            H--xC: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
        else lỗi DB khác
            Note over R,DB: rollback (a)-(d)
            R--xU: wrapped "reissue otp: %w"
            U--xH: lỗi
            H--xC: 500 INTERNAL_ERROR
        end
    end

    U->>S: SendOTP(phone, otpCode, purpose) (Step 5)
    Note over U,S: thuộc pkg/sms (qua adapter sms_otp_sender), ngoài scope feature này — impl log-only
    alt gửi OK
        S-->>U: nil
        U-->>H: result
        H-->>C: 200 {message, expires_in_seconds: 90, resend_after_seconds: 60, resend_remaining}
    else SMS thất bại
        S--xU: ErrSMSSendFailed
        U--xH: ErrSMSSendFailed
        H--xC: 500 SMS_SEND_FAILED (send_count đã tăng + OTP cũ đã vô hiệu — KHÔNG rollback)
    end
```
