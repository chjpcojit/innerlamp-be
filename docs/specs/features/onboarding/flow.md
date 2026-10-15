# Flow — onboarding

Version: 8.2
Last updated: 2026-10-16
Status: implemented (v8.2) — đăng ký xác minh bằng email chạy thật (mailer `log` ở dev). Còn mở: giới hạn theo IP (AC22, login Phase 5), `erd.svg`, OpenAPI, OKF — xem `tasks.md`.
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của ba section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự). Mỗi message trace
1:1 về một Step; không thêm/bỏ bước.

## Sequence — Bước 1: request-otp

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as authRateLimit("send")
    participant H as RegisterHandler
    participant U as RegisterUsecase
    participant O as OTPService
    participant R as RegistrationRepository
    participant DB as Postgres
    participant M as AuthMailer

    C->>L: POST /v1/auth/register/request-otp
    alt vượt ngân sách IP (Step R-1)
        L--xC: 429 RATE_LIMITED
    end
    L->>H: request
    H->>U: RequestOTP(input) (Step R-2)
    U->>U: validate email → SĐT theo nước → mật khẩu → tên (Step R-2)
    alt trường sai
        U--xC: 400 INVALID_*
    end
    U->>R: PhoneTaken(phone) (Step R-3)
    alt SĐT thuộc tài khoản
        U--xC: 409 PHONE_TAKEN
    end
    U->>R: temple tồn tại? (Step R-4)
    alt chùa không tồn tại
        U--xC: 400 INVALID_TEMPLE
    end
    U->>R: EmailHasAccount(email) (Step R-5)
    U->>U: hash mật khẩu + O.Generate() → mã, hash mã (Step R-6)
    U->>R: SavePendingAndIssueOTP(pending{AccountExists}, hash mã, canReplace) (Step R-7)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step R-7)
        R->>DB: khoá email + FOR UPDATE hàng pending (Step R-7)
        alt có mã sống và payload khác (canReplace = false)
            R--xC: 409 REGISTRATION_IN_PROGRESS (AC24)
        end
        R->>DB: áp send policy (email, register)
        alt cooldown / quá số lượt
            R--xC: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
        end
        R->>DB: UPSERT pending_registration theo email
        R->>DB: vô hiệu mã cũ + INSERT otp_codes
    end
    alt AccountExists = false
        U->>M: SendRegisterCode(email, mã, ttl) (Step R-8)
    else AccountExists = true
        U->>M: SendAccountExistsNotice(email) (Step R-8)
    end
    alt mailer lỗi
        U->>O: ReleaseSendLimit(email, register)
        U--xC: 500 EMAIL_SEND_FAILED
    end
    U-->>C: 200 {message, expires_in_seconds, resend_after_seconds}
```

## Sequence — resend-otp

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as authRateLimit("send")
    participant U as RegisterUsecase
    participant O as OTPService
    participant R as RegistrationRepository
    participant M as AuthMailer

    C->>L: POST /v1/auth/register/resend-otp {email}
    alt vượt ngân sách IP (Step S-1)
        L--xC: 429 RATE_LIMITED
    end
    L->>U: ResendOTP(email) — validate email (Step S-2)
    U->>R: FindPendingRegistration(email) (Step S-3)
    alt không có / hết hạn
        U--xC: 400 PENDING_REGISTRATION_EXPIRED
    end
    U->>O: Issue(email, register) — một transaction (Step S-4)
    alt cooldown / quá số lượt
        O--xC: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
    end
    U->>M: thư mã hoặc thư báo theo AccountExists (Step S-5)
    U-->>C: 200 {…, resend_remaining}
```

## Sequence — Bước 2: verify-otp

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as authRateLimit("verify")
    participant U as RegisterUsecase
    participant O as OTPService
    participant R as RegistrationRepository
    participant DB as Postgres

    C->>L: POST /v1/auth/register/verify-otp {email, otp_code}
    alt vượt ngân sách IP (Step V-1)
        L--xC: 429 RATE_LIMITED
    end
    L->>U: VerifyOTP — validate (Step V-2)
    U->>R: FindPendingRegistration(email) (Step V-3)
    alt không có / hết hạn
        U--xC: 400 PENDING_REGISTRATION_EXPIRED
    end
    U->>O: Verify(email, register, mã) (Step V-4)
    O->>DB: ConsumeOTPAttempt — tăng attempts nguyên tử, khoá ở lần thứ 3
    alt sai / hết hạn / bị khoá
        O--xC: 400 OTP_INVALID / OTP_EXPIRED · 429 OTP_LOCKED
    end
    alt pending.AccountExists
        U--xC: 400 OTP_INVALID
    end
    U->>R: CreateUserAndSession(pending, otpID, issuer) (Step V-5)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step V-5)
        R->>DB: INSERT users (email_verified_at = now())
        alt SĐT vừa bị lấy / chùa đã xoá
            R--xC: 409 PHONE_TAKEN / 400 INVALID_TEMPLE (rollback)
        end
        R->>DB: INSERT user_sessions (token mang sid)
        R->>DB: otp_codes.used_at = now()
        R->>DB: DELETE pending_registration
    end
    U-->>C: 201 {access_token, refresh_token, expires_in, user}
```
