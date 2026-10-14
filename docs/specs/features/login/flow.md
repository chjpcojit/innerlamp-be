# Flow — login

Version: 2.2
Last updated: 2026-10-14
Status: prospective (v2.0) — đăng nhập bằng email, kiểm phiên còn sống, quên mật khẩu; chưa có code. Sơ đồ refresh/logout của v1.13 không đổi (xem git history bản v1.13 nếu cần).
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của bốn section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự). Mỗi message trace
1:1 về một Step; không thêm/bỏ bước.

## Sequence — Login

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as authRateLimit("login")
    participant U as LoginUsecase
    participant R as LoginRepository
    participant DB as Postgres

    C->>L: POST /v1/auth/login {email, password, device}
    alt vượt ngân sách IP (Step L-1)
        L--xC: 429 RATE_LIMITED
    end
    L->>U: Login(input)
    U->>U: validate email / password không rỗng / device (Step L-2)
    alt sai định dạng
        U--xC: 400 INVALID_EMAIL / INVALID_PASSWORD / INVALID_DEVICE
    end
    U->>R: ReserveLoginAttempt(email) — một UPSERT (Step L-3)
    alt đang khoá từ trước
        R--xC: 429 LOGIN_LOCKED {locked_until}
    end
    U->>R: FindUserByEmail(email) (Step L-4)
    U->>U: so mật khẩu (hash giả nếu không có user hoặc chưa xác minh)
    alt sai
        U--xC: 401 INVALID_CREDENTIALS
    end
    U->>R: ResetLoginAttempts(email) (Step L-5)
    U->>R: CreateLoginSession(user, device, ttl, issuer) (Step L-6)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction như v1.13 (Step L-6)
        R->>DB: upsert device, revoke phiên cũ của device, INSERT session, mint token có sid
    end
    U-->>C: 200 {access_token, refresh_token, expires_in, user}
```

## Sequence — Kiểm phiên còn sống (mọi route Bearer)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant J as authMiddleware (pkg/jwt)
    participant S as activeSession
    participant DB as Postgres
    participant H as Handler bất kỳ

    C->>J: request + Bearer access token
    alt token sai / hết hạn / không phải access (Step A-1)
        J--xC: 401 UNAUTHORIZED
    end
    J->>S: claims {user_id, sid}
    S->>DB: IsSessionActive(user_id, sid) (Step A-2)
    alt phiên đã thu hồi / hết hạn
        S--xC: 401 UNAUTHORIZED
    else DB lỗi
        S--xC: 503
    end
    S->>H: đi tiếp
```

## Sequence — Xin mã đặt lại mật khẩu

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as authRateLimit("send")
    participant U as PasswordResetUsecase
    participant R as PasswordResetRepository
    participant O as OTPService
    participant M as AuthMailer

    C->>L: POST /v1/auth/password-reset/request-otp {email}
    alt vượt ngân sách IP (Step P-1)
        L--xC: 429 RATE_LIMITED
    end
    L->>U: RequestOTP(email) — ghi start (Step P-1)
    U->>U: NormalizeEmail (Step P-2)
    alt email sai
        U--xC: 400 INVALID_EMAIL
    end
    U->>R: FindVerifiedUserByEmail(email) (Step P-3)
    alt có tài khoản (Step P-4)
        U->>O: EnsureNotLocked + Issue(email, reset_password)
        alt khoá / cooldown / quá lượt
            O--xU: 429 OTP_LOCKED / OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
        end
        U->>M: SendPasswordResetCode(email, mã, ttl)
        Note over U,M: mailer lỗi → ReleaseSendLimit + log, vẫn trả như thành công
    else không có tài khoản (Step P-4)
        U->>O: ApplySendLimit(email, reset_password)
        alt cooldown / quá lượt
            O--xU: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
        end
    end
    U->>U: chờ tới start + min_response_time (Step P-5)
    U-->>C: 200 {message, expires_in_seconds, resend_after_seconds} hoặc 429 đã nhận ở P-4
```

## Sequence — Đặt mật khẩu mới

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as authRateLimit("verify")
    participant U as PasswordResetUsecase
    participant O as OTPService
    participant R as PasswordResetRepository
    participant DB as Postgres

    C->>L: POST /v1/auth/password-reset/confirm {email, otp_code, new_password}
    alt vượt ngân sách IP (Step C-1)
        L--xC: 429 RATE_LIMITED
    end
    L->>U: Confirm(input)
    U->>U: validate email, mã, mật khẩu mới (Step C-2)
    alt sai
        U--xC: 400 INVALID_EMAIL / INVALID_PASSWORD
    end
    U->>O: Verify(email, reset_password, mã) (Step C-3)
    alt sai / hết hạn / khoá
        O--xC: 400 OTP_INVALID / OTP_EXPIRED · 429 OTP_LOCKED
    end
    U->>U: hash mật khẩu mới (Step C-4)
    U->>R: ResetPassword(userID, hash, otpID) (Step C-4)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step C-4)
        R->>DB: UPDATE users.password_hash
        R->>DB: otp_codes.used_at = now()
        R->>DB: revoke mọi user_sessions của user (password_reset)
        R->>DB: DELETE login_attempts của email (gỡ khoá)
    end
    U-->>C: 200 {success: true}
```
