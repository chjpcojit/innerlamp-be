# Design — otp-mechanism

Version: 2.1
Last updated: 2026-10-16
Status: implemented (v2.1) — cơ chế OTP khoá theo email, gửi qua `AuthMailer`, mã `crypto/rand` (mã cố định chỉ ở dev), đếm lượt nguyên tử bằng `ConsumeOTPAttempt`.
Module: `otp`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Giữ nguyên tách tầng của v1.2: `domain.OTPRepository` (persistence) + `usecase.OTPService` (sinh, hash,
áp send policy, verify, khoá). v2.0 đổi khoá từ `phone_number` sang `email`, bỏ `OTPSender`/`Send`
(caller gửi thư), sinh mã bằng `crypto/rand`, giữ lượt nguyên tử khi verify, thêm `ApplySendLimit`.
Register vẫn giữ `SavePendingAndIssueOTP` riêng (một transaction cho pending + mã) và dùng chung SQL
helper với `IssueOTP`.

## Layers & components

| Layer | File | Responsibility | Trạng thái |
|---|---|---|---|
| Domain | `services/appService/domain/otp.go` | `OTPCode.Email` thay `PhoneNumber`; `OTPPurpose` giữ bốn giá trị. | sửa |
| Domain | `services/appService/domain/otp_repository.go` | `OTPRepository`: `IssueOTP(email, purpose, hash, expiresAt, limits)`, `ApplySendLimit(email, purpose, limits)` (mới), `FindActiveOTP(email, purpose)`, `ConsumeOTPAttempt(id, max, lockUntil) (attempts int, ok bool, err)` (thay `IncrementOTPAttempts` + `LockOTP`), `ReleaseSendLimit(email, purpose)`. | sửa |
| Domain / Mock | `services/appService/domain/mock_otp_repository.go`, bỏ `mock_otp_sender.go` | Theo interface mới. | sửa / xoá |
| Domain | `services/appService/domain/otp_send_limit.go` | Khoá theo email. | sửa |
| Usecase | `services/appService/usecase/otp_service.go` | `Generate()` (crypto/rand hoặc mã dev), `Issue`, `ApplySendLimit`, `EnsureNotLocked`, `Verify` (Step V), `ReleaseSendLimit`, `ExpiresInSeconds`, `ResendAfterSeconds`. Bỏ `Send`, `wrapSendError`, phụ thuộc `OTPSender`. | sửa |
| Repository | `services/appService/repository/postgres_registration_repository.go` (concrete `OTPRepository`) | SQL theo email; `ConsumeOTPAttempt` một UPDATE; `ApplySendLimit` dùng lại `applySendLimits`. | sửa |
| Config | `services/appService/config/config.go` | `OTPConfig.DevFixedCode`; `ttl: 600s`; kiểm AC3 trong `Validate()` cùng `app.environment`. | sửa |
| Migration | `migrations/0037_auth_email_identity.sql` | Bước 2–3 (onboarding design). | mới |

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Issue

**Step I-1 — Sinh mã:** `Generate()` → `(code, bcrypt(code))`. Lỗi hash → `500`, không ghi gì.
**Step I-2 — Ghi (một transaction):** `IssueOTP`: `applySendLimits` trên `otp_send_limits (email, purpose)`
(INSERT … ON CONFLICT DO NOTHING rồi `SELECT … FOR UPDATE`, quyết định cooldown/cap/khoá) →
`invalidateActiveOTP` → `insertOTP`. Bị từ chối → `OTPResendTooSoonError` / `OTPSendLimitedError`, rollback.
**Step I-3 — Trả về:** `OTPIssueResult{Code, ExpiresAt, ResendRemaining}`; caller gửi thư và gọi
`ReleaseSendLimit` nếu gửi lỗi.

`ApplySendLimit` là Step I-2 không có `invalidateActiveOTP`/`insertOTP`.

## Step-by-step Flow — Verify

**Step V-1 — Tìm mã:** `FindActiveOTP(email, purpose)` — mã mới nhất chưa dùng. Không có → `ErrOTPExpired`.
**Step V-2 — Đã khoá?** `locked_until > now()` → `OTPLockedError`. `expires_at <= now()` → `ErrOTPExpired`.
**Step V-3 — Giữ lượt (nguyên tử):** `ConsumeOTPAttempt(id, max, now+lock_duration)`:
```sql
UPDATE otp_codes
SET attempts = attempts + 1,
    locked_until = CASE WHEN attempts + 1 >= $2 THEN $3 ELSE locked_until END
WHERE id = $1 AND used_at IS NULL AND attempts < $2
RETURNING attempts, locked_until;
```
Không có hàng → đã hết lượt → `OTPLockedError`.
**Step V-4 — So hash:** sai → `ErrOTPInvalid` (lượt đã tính ở V-3; nếu V-3 vừa đặt khoá thì request sau
thấy `429`). Đúng → trả `*OTPCode`; caller đánh `used_at` trong transaction của mình.

## Known gaps

- Mã đúng ở đúng lượt thứ 3 vẫn được chấp nhận (khoá chỉ chặn lượt sau) — giống v1.2, khớp "3 wrong entries lock".
