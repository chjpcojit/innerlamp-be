# Flow — otp-mechanism

Version: 1.2
Last updated: 2026-09-25
Status: complete — bản vẽ Mermaid mirror 1:1 hai section "Step-by-step Flow" của `design.md`
Module: `otp`
Feature folder: `services/appService/` + `docs/specs/features/otp-mechanism/`

> See `changelog.md` in this folder for the version history of this spec.

<!--
File này là **bản vẽ** (visual) của các section "Step-by-step Flow" trong `design.md`. Nó không thêm
bước mới, không bỏ bước — mỗi mũi tên/message ở đây trace 1:1 về một Step trong `design.md`. Ngược
lại, mỗi Step trong `design.md` phải xuất hiện ở đây. Thấy chỗ lệch là bug của chính file này.

Quy ước:
- Một section `### Sequence — ...` cho MỖI section "Step-by-step Flow — ..." của `design.md` (cùng
  tên, cùng thứ tự). Feature này có **hai** flow: Issue OTP (cơ chế dùng chung), Verify OTP + completion.
- Transaction của repository được bọc bằng `rect` + `Note over R,DB: MỘT transaction`.
- Bước gửi SMS ghi `Note` "thuộc pkg/sms, ngoài scope cơ chế này" (impl log-only) — không vẽ nội bộ
  của `pkg/sms`.
- `OTPRepository` và `RegistrationRepository` là hai interface, nhưng được implement bởi **cùng một**
  concrete `registrationRepository` (`repository/postgres_registration_repository.go`) — trong diagram
  chúng là hai participant riêng để giữ đúng ranh giới dependency, kèm ghi chú.
- Nhãn participant/message viết tiếng Việt ngắn; keyword Mermaid giữ tiếng Anh.
-->

## Sequence — Issue OTP (cơ chế dùng chung)

Áp dụng cho `resend-otp` (register) hôm nay, và cho luồng gửi đầu của mọi feature OTP tương lai.
Không đụng `pending_registration`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as RegisterHandler<br/>(delivery/http)
    participant U as RegisterUsecase
    participant O as OTPService<br/>(usecase)
    participant R as OTPRepository
    participant DB as Postgres
    participant S as pkg/sms

    C->>H: POST /v1/auth/resend-otp {phone_number, purpose}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: ResendOTP(ctx, input)
    Note over U,O: pre-steps register (validate purpose → FindPendingRegistration → EnsureNotLocked)<br/>chạy trước; ngoài cơ chế này, không vẽ ở đây

    U->>O: Issue(ctx, phone, register) (Step 1 trigger)
    O->>O: Generate() → generateOTP() (111111) → bcrypt hash (Step 1)
    alt lỗi hash
        O--xU: wrapped "hash otp: %w"
        U--xH: lỗi
        H--xC: 500 INTERNAL_ERROR (không ghi gì)
    end

    O->>R: IssueOTP(phone, purpose, otpHash, expiresAt, sendLimits) (Step 2)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>DB: (a) applySendLimits(phone, purpose) — cooldown 60s / trần 4 lượt / khoá-gửi 30 phút
        R->>DB: (b) invalidateActiveOTP(phone, purpose)
        R->>DB: (c) insertOTP(phone, purpose, otpHash, expiresAt)
        alt bị chặn send-limit
            Note over R,DB: rollback (a)-(c), không ghi gì
            R--xO: *OTPResendTooSoonError / *OTPSendLimitedError
            O--xU: typed error
            U--xH: typed error
            H--xC: 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
        else lỗi DB khác
            Note over R,DB: rollback (a)-(c)
            R--xO: wrapped "issue otp: %w"
            O--xU: lỗi
            U--xH: lỗi
            H--xC: 500 INTERNAL_ERROR
        end
    end

    O->>S: SendOTP(phone, code, purpose) (Step 3)
    Note over O,S: thuộc pkg/sms, ngoài scope cơ chế này — impl log-only
    alt gửi OK
        S-->>O: nil
        O-->>U: OTPIssueResult{expires_in_seconds, resend_after_seconds, resend_remaining}
        U-->>H: ResendOTPResult
        H-->>C: 200 {message, expires_in_seconds, resend_after_seconds, resend_remaining}
    else SMS thất bại
        S--xO: ErrSMSSendFailed
        O--xU: ErrSMSSendFailed
        U--xH: ErrSMSSendFailed
        H--xC: 500 SMS_SEND_FAILED (hàng OTP đã commit — KHÔNG rollback Step 2)
    end
```

## Sequence — Verify OTP (cơ chế dùng chung) + completion của register

Steps 1–3 chạy trong `OTPService.Verify` (cơ chế dùng chung); Step 4 là completion riêng của
register (`CreateUserAndSession`).

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as RegisterHandler<br/>(delivery/http)
    participant U as RegisterUsecase
    participant O as OTPService<br/>(usecase)
    participant R as OTPRepository
    participant RR as RegistrationRepository
    participant DB as Postgres

    C->>H: POST /v1/auth/register/verify-otp {phone_number, otp_code}
    H->>H: DecodeJSONBodyStrict + validate DTO
    H->>U: VerifyOTP(ctx, input)
    Note over U: FindPendingRegistration(phone) — pre-step register (pending còn hạn), ngoài cơ chế

    U->>O: Verify(ctx, phone, register, code)
    O->>R: FindActiveOTP(phone, purpose) (Step 1)
    alt có hàng active/locked
        R-->>O: otp
    else không có hàng hiệu lực
        R--xO: ErrOTPInvalid
        O--xU: ErrOTPExpired
        U--xH: ErrOTPExpired
        H--xC: 400 OTP_EXPIRED (dừng)
    end

    O->>O: đọc otp.LockedUntil (Step 2)
    alt locked_until > now
        O--xU: *OTPLockedError
        U--xH: *OTPLockedError
        H--xC: 429 OTP_LOCKED {locked_until} (không ghi gì)
    end

    O->>O: bcrypt.CompareHashAndPassword(otp.OTPHash, code) (Step 3)
    alt khớp
        O-->>U: *OTPCode (đã verify)
    else sai mã
        O->>R: IncrementOTPAttempts(otp.ID)
        R->>DB: attempts+1
        opt attempts >= MaxAttempts (3)
            O->>R: LockOTP(otp.ID, now()+LockDuration)
            R->>DB: locked_until = now()+30m
        end
        O--xU: ErrOTPInvalid (không trả số lần còn lại)
        U--xH: ErrOTPInvalid
        H--xC: 400 OTP_INVALID (khoá set ở đây; lần kế tiếp → 429 OTP_LOCKED ở Step 2)
    end

    U->>RR: CreateUserAndSession(pending, otp.ID, sessionTTL, issueTokens) (Step 4)
    rect rgb(235, 240, 255)
        Note over RR,DB: MỘT transaction
        RR->>DB: INSERT users + DELETE pending + UPDATE otp_codes (used_at)<br/>+ INSERT user_sessions + mint JWT (closure TokenIssuer)
        alt email/phone race (unique 23505)
            Note over RR,DB: rollback
            RR--xU: ErrEmailTaken / ErrPhoneTaken
            U--xH: ErrEmailTaken / ErrPhoneTaken
            H--xC: 409 EMAIL_TAKEN / PHONE_TAKEN
        else lỗi DB khác
            Note over RR,DB: rollback
            RR--xU: wrapped lỗi
            U--xH: lỗi
            H--xC: 500 INTERNAL_ERROR
        end
    end
    RR-->>U: RegisterResult{access_token, refresh_token, user}
    U-->>H: RegisterResult
    H-->>C: 201 {access_token, refresh_token, user}
```
