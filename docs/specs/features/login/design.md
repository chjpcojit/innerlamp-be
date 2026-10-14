# Design — login

Version: 2.1
Last updated: 2026-10-14
Status: prospective (v2.0) — đăng nhập bằng email, quên mật khẩu, giới hạn tần suất, kiểm phiên còn sống; chưa có code. Phiên/refresh/logout của v1.13 giữ nguyên.
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Năm route `/v1/auth/*` của feature này: `login`, `refresh`, `logout` (đã có, đổi khoá sang email) và
`password-reset/request-otp`, `password-reset/confirm` (mới). Shape giữ như v1.13: `LoginHandler` →
`LoginUsecase` → `LoginRepository`; quên mật khẩu là `PasswordResetUsecase` mới dùng `OTPService`
([`../otp-mechanism/`](../otp-mechanism/design.md)) và `AuthMailer` ([`../onboarding/design.md`](../onboarding/design.md)).
Hai phần chung cho mọi `/v1/auth/*` cũng nằm ở đây: middleware giới hạn tần suất (Postgres) và
middleware kiểm phiên còn sống cho mọi route Bearer. Migration dùng chung: `0037_auth_email_identity.sql`.

## Layers & components

| Layer | File | Responsibility | Trạng thái |
|---|---|---|---|
| Domain | `services/appService/domain/login.go` | `LoginInput.Email` thay `PhoneNumber`; `LoginRepository`: `FindUserByEmail` thay `FindUserByPhone`; `ReserveLoginAttempt(email, max, lock, window) (*LoginAttempt, error)` thay `FindLoginAttempt` + `RecordLoginFailure`; `ResetLoginAttempts(email)`; `IsSessionActive(userID, sessionID) (bool, error)`. Hằng `RevokeReasonPasswordReset`. | sửa |
| Domain | `services/appService/domain/password_reset.go` | `PasswordResetRepository { FindVerifiedUserByEmail; ResetPassword(ctx, userID, newHash, otpID) }`; input/result types. | mới |
| Domain / Errors | `services/appService/domain/errors.go` | Thêm `ErrRateLimited` (kèm `RetryAfter`), `ErrSessionRevoked`. | sửa |
| Usecase | `services/appService/usecase/login_usecase.go` | Step L-1…L-6; hash qua `PasswordHasher` (onboarding). Refresh/Logout không đổi. | sửa |
| Usecase | `services/appService/usecase/password_reset_usecase.go` | Step P-1…P-5, C-1…C-4. | mới |
| Repository | `services/appService/repository/postgres_login_repository.go` | SQL theo email; `ReserveLoginAttempt` một UPSERT; `IsSessionActive`. | sửa |
| Repository | `services/appService/repository/postgres_password_reset_repository.go` | `ResetPassword` trong một transaction: UPDATE `users.password_hash` → `otp_codes.used_at` → revoke mọi session. | mới |
| Repository | `services/appService/repository/postgres_rate_limit_store.go` | `Hit(ctx, bucket, window) (count int, windowEnd time.Time, error)` trên `auth_rate_limits`. | mới |
| Repository / Mailer | `services/appService/repository/budgeted_mailer.go` | Decorator quanh `pkg/mailer.Mailer`: `Hit("mail:daily", 24h)` trước mỗi thư; vượt `mailer.daily_budget` → `ErrRateLimited`. | mới |
| Delivery | `services/appService/delivery/http/auth_rate_limit.go` | Middleware `authRateLimit(group)` dùng `postgres_rate_limit_store`; IP theo AC34. | mới |
| Delivery | `services/appService/delivery/http/active_session.go` | Middleware sau `authMiddleware`: đọc `user_id` + `sid` từ claims, `IsSessionActive`; không còn → `401 UNAUTHORIZED`; lỗi DB → `503`. | mới |
| Delivery | `services/appService/delivery/http/login_handler.go` | DTO `email`; thông điệp "Email hoặc mật khẩu không đúng". | sửa |
| Delivery | `services/appService/delivery/http/password_reset_handler.go` | Hai route US-003; `no-store`. | mới |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | `ErrRateLimited` → `429 RATE_LIMITED` + `retry_after_seconds`. | sửa |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `auth_rate_limit.{send,verify,login,refresh}.{max,window}`, `auth_rate_limit.trust_forwarded_for`; `mailer.daily_budget`; `password_reset.min_response_time`. | sửa |
| Wiring | `services/appService/cmd/main.go`, `cmd/profile_routes.go`, các file route khác | Bọc `authRateLimit` cho các route `/v1/auth/*`; bọc `activeSession` sau mọi `authMiddleware.AuthenticateFunc`. | sửa |
| Migration | `migrations/0037_auth_email_identity.sql` | `login_attempts` theo email (onboarding design bước 5); bảng `auth_rate_limits` (bước 7, bên dưới). | mới |
| API spec | `docs/specs/api/openapi.yaml` | Năm operation theo bảng API; `401` mới trên route Bearer. | sửa |

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Login

**Step L-1 — Giới hạn theo IP:** `authRateLimit("login")`; vượt → `429 RATE_LIMITED`.
**Step L-2 — Validate:** email (`NormalizeEmail`), password không rỗng, device (AC3, AC4). Lỗi → `400`, không giữ lượt.
**Step L-3 — Giữ lượt (nguyên tử)**
- API: `ReserveLoginAttempt(email, …)` — một UPSERT trên `login_attempts`:
  nếu `locked_until > now()` thì không đổi gì và trả hàng đang khoá; nếu lần trước quá `failure_reset_window`
  thì `failed_count = 1`, ngược lại `failed_count + 1`; khi chạm `max_failed_attempts` đặt
  `locked_until = now() + lock_duration`. `RETURNING`.
- Hàng đang khoá từ trước → `429 LOGIN_LOCKED` + `locked_until` (AC5), không so mật khẩu.
- Lượt vừa chạm ngưỡng vẫn đi tiếp và, nếu sai, trả `401` (như v1.13: chỉ request sau thấy `429`).
**Step L-4 — Tìm user + so mật khẩu**
- `FindUserByEmail(email)`; không có hoặc `email_verified_at IS NULL` → so với hash giả cố định.
- `PasswordHasher.Compare`. Sai → `401 INVALID_CREDENTIALS` (AC2); lượt đã giữ ở L-3 được tính là thất bại.
**Step L-5 — Thành công:** `ResetLoginAttempts(email)`.
**Step L-6 — Tạo phiên:** `CreateLoginSession(user, device, ttl, issuer)` — một transaction như v1.13
(upsert device, revoke phiên cũ của device, INSERT session, mint token có `sid`). `200`.

## Step-by-step Flow — Kiểm phiên còn sống (mọi route Bearer)

**Step A-1** — `authMiddleware` (pkg/jwt) kiểm chữ ký, hạn, loại `access` → lỗi `401`.
**Step A-2** — `activeSession`: `IsSessionActive(user_id, sid)` — `SELECT 1 FROM user_sessions WHERE session_id = $1 AND user_id = $2 AND is_revoked = false AND expires_at > now()`.
Không có → `401 UNAUTHORIZED` (AC21); lỗi DB → `503` (AC22). Có → handler chạy.

## Step-by-step Flow — Xin mã đặt lại mật khẩu

**Step P-1** — `authRateLimit("send")`; ghi `start = now()`.
**Step P-2** — `NormalizeEmail`; sai → `400 INVALID_EMAIL` (trả ngay, không chờ AC25: chưa chạm DB).
**Step P-3** — `FindVerifiedUserByEmail(email)`.
**Step P-4 — Nhánh**
- Có tài khoản: `OTPService.EnsureNotLocked` (→ `429 OTP_LOCKED`) rồi `OTPService.Issue(email, reset_password)`
  (send policy + vô hiệu mã cũ + INSERT, một transaction; → `429` khi bị từ chối) rồi
  `AuthMailer.SendPasswordResetCode`. Lỗi mailer: `ReleaseSendLimit`, log, vẫn trả như thành công.
- Không có: `OTPService.ApplySendLimit(email, reset_password)` (chỉ ghi send policy; → `429` y hệt).
**Step P-5** — Chờ tới `start + password_reset.min_response_time` (cả nhánh lỗi `429`), rồi
`200 {message, expires_in_seconds, resend_after_seconds}`.

## Step-by-step Flow — Đặt mật khẩu mới

**Step C-1** — `authRateLimit("verify")`.
**Step C-2** — Validate email, `otp_code` 6 chữ số, `new_password` theo BR-ONB-06 → `400` (AC30).
**Step C-3** — `OTPService.Verify(email, reset_password, code)` (giữ lượt nguyên tử rồi so) → `400 OTP_INVALID` / `400 OTP_EXPIRED` / `429 OTP_LOCKED`. Email không có tài khoản không có mã → `OTP_EXPIRED`.
**Step C-4 — Đổi mật khẩu (một transaction)**
- `PasswordHasher.Hash(new_password)` ở usecase, rồi `ResetPassword(userID, newHash, otpID)`:
  UPDATE `users.password_hash` → `otp_codes.used_at = now()` →
  `UPDATE user_sessions SET is_revoked = true, revoked_at = now(), revoked_reason = 'password_reset' WHERE user_id = $1 AND is_revoked = false` →
  `DELETE FROM login_attempts WHERE email = $2` (gỡ khoá đăng nhập, BR-LGN-05 v1.2).
- `200 {"success": true}`. Thiết bị khác nhận `401` ở request kế tiếp (Step A-2).

## Giới hạn tần suất

- Bảng (migration `0037` bước 7):
  ```sql
  CREATE TABLE IF NOT EXISTS auth_rate_limits (
      bucket       VARCHAR(128) NOT NULL,   -- "ip:login:203.0.113.7", "mail:daily"
      window_start TIMESTAMPTZ  NOT NULL,   -- date_bin(window, now(), epoch)
      hits         INT          NOT NULL DEFAULT 0,
      PRIMARY KEY (bucket, window_start)
  );
  ```
- `Hit`: `INSERT … VALUES ($1, date_bin($2, now(), 'epoch'), 1) ON CONFLICT (bucket, window_start) DO UPDATE SET hits = auth_rate_limits.hits + 1 RETURNING hits, window_start`. Cửa sổ cố định; `retry_after = window_start + window - now()`.
- Đếm cả request bị từ chối (không có cách "lách" bằng cách gửi dồn).
- `mail:daily` dùng cửa sổ 24h căn theo epoch (tức theo ngày UTC).
- Ngân sách mặc định và nhóm route: `requirements.md` AC31, AC33.

## Known gaps

- Hash giả ở Step L-4 dùng cùng hàm `PasswordHasher` để thời gian như nhau.
- Hàng cũ trong `auth_rate_limits` chưa có job dọn — Deferred chung với onboarding.
- Lỗi mailer khi đặt lại mật khẩu bị nuốt (chỉ log) để không lộ nhánh; người dùng bấm "Gửi lại mã".
