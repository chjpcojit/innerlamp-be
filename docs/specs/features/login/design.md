# Design — login

Version: 2.6
Last updated: 2026-10-16
Status: implemented (v2.6, 2026-10-16) — đăng nhập bằng email, quên mật khẩu, giới hạn tần suất, kiểm phiên còn sống. Phiên/refresh/logout của v1.13 giữ nguyên.
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Năm route `/v1/auth/*` của feature này: `login`, `refresh`, `logout` (đã có, đổi khoá sang email) và
`password-reset/request-otp`, `password-reset/confirm` (mới). Shape giữ như v1.13: `LoginHandler` →
`LoginUsecase` → `LoginRepository`; quên mật khẩu là `PasswordResetUsecase` mới dùng `OTPService`
([`../otp-mechanism/`](../otp-mechanism/design.md)) và `AuthMailer` ([`../onboarding/design.md`](../onboarding/design.md)).
Hai phần chung cho mọi `/v1/auth/*` cũng nằm ở đây: middleware giới hạn tần suất (store dùng chung, Redis) và
middleware kiểm phiên còn sống cho mọi route Bearer. Migration: `0038_login_attempts_email.sql` (bước 5 của onboarding "Migration 0037"); không có bảng đếm giới hạn.

## Layers & components

| Layer | File | Responsibility | Trạng thái |
|---|---|---|---|
| Domain | `services/appService/domain/login.go` | `LoginInput.Email` thay `PhoneNumber`; `LoginRepository`: `FindUserByEmail` thay `FindUserByPhone`; `ReserveLoginAttempt(email, max, lock, window) (*LoginAttempt, error)` thay `FindLoginAttempt` + `RecordLoginFailure` (kết quả có `AlreadyLocked`); `IsSessionActive(userID, sessionID) (bool, error)`. Xoá bộ đếm (Step L-5) là `DELETE` trong transaction của `CreateLoginSession`, không có method riêng. Hằng `RevokeReasonPasswordReset`. | sửa |
| Domain | `services/appService/domain/password_reset.go` | `PasswordResetRepository { FindVerifiedUserByEmail; ResetPassword(ctx, userID, newHash, otpID) }`; input/result types. | mới |
| Domain / Errors | `services/appService/domain/errors.go` | Thêm `ErrRateLimited` (kiểu `RateLimitedError{RetryAfter}`, chỉ cho ngân sách thư), `ErrSessionRevoked`. | sửa |
| Usecase | `services/appService/usecase/login_usecase.go` | Step L-1…L-6; hash qua `PasswordHasher` (onboarding). Refresh/Logout không đổi. | sửa |
| Usecase | `services/appService/usecase/password_reset_usecase.go` | Step P-1…P-5, C-1…C-4. | mới |
| Repository | `services/appService/repository/postgres_login_repository.go` | SQL theo email; `ReserveLoginAttempt` một UPSERT; `IsSessionActive`. | sửa |
| Repository | `services/appService/repository/postgres_password_reset_repository.go` | `ResetPassword` trong một transaction: UPDATE `users.password_hash` → `otp_codes.used_at` → revoke mọi session. | mới |
| Repository / Mailer | `services/appService/repository/budgeted_mailer.go` | Decorator quanh `domain.AuthMailer`: mỗi thư tiêu một đơn vị của `mailer.daily_budget` (key theo ngày UTC trong `ratelimit.Store` dùng chung) trước khi gửi; hết → `*RateLimitedError` (đến 0h UTC). Lỗi store → lỗi gửi thư thường (fail-closed). | mới |
| Delivery | `services/appService/delivery/http/client_ip_rate_limit.go` | `ClientIPRateLimit` có sẵn, mỗi nhóm một instance (scope `auth:<nhóm>:ip` trong store dùng chung); IP từ `rateLimitIP(r, trustXFF)` theo AC34: đọc cờ "key hợp lệ" mà `RequireWebAPIKey` gắn vào context, nên phải chạy sau nó. `LoginHandler` dùng cùng hàm cho metadata thiết bị. | sửa |
| Delivery | `services/appService/delivery/http/active_session.go` | Middleware sau `authMiddleware`: đọc `user_id` + `sid` từ claims, `IsSessionActive`; không còn → `401 UNAUTHORIZED`; lỗi DB → `503`. | mới |
| Delivery | `services/appService/delivery/http/login_handler.go` | DTO `email`; thông điệp "Email hoặc mật khẩu không đúng". | sửa |
| Delivery | `services/appService/delivery/http/password_reset_handler.go` | Hai route US-003; `no-store`. | mới |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | `ErrRateLimited` → `429 RATE_LIMITED` + `retry_after_seconds`. | sửa |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `auth_rate_limit.{send,verify,login,refresh}.{max,window}`, `auth_rate_limit.trust_forwarded_for`; `mailer.daily_budget`; `password_reset.min_response_time` (tất cả có `Validate()` chặn khởi động). | sửa |
| Wiring | `services/appService/cmd/main.go` | Bọc từng nhóm giới hạn + `NoStore` cho các route `/v1/auth/*`; dựng `authed` = `authMiddleware.AuthenticateFunc` + `activeSession` và truyền cho các file route (`profile_routes.go`, `temple_announcement_routes.go`); `logout` chỉ dùng `authMiddleware` (idempotent, AC14). `cmd/auth_wiring_test.go` bắt route sót. | sửa |
| Migration | `migrations/0038_login_attempts_email.sql` | `login_attempts` khoá chính theo `email` (onboarding design bước 5); TRUNCATE có guard. Bước 7 (`auth_rate_limits`) bỏ. | mới |
| API spec | `docs/specs/api/openapi.yaml` | Năm operation theo bảng API; `401` mới trên route Bearer. | sửa |

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Login

**Step L-1 — Giới hạn theo IP:** nhóm `login` của `ClientIPRateLimit`; vượt → `429 RATE_LIMITED`.
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

**Step P-1** — nhóm `send`; ghi `start = now()`.
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

**Step C-1** — nhóm `verify`.
**Step C-2** — Validate email, `otp_code` 6 chữ số, `new_password` theo BR-ONB-06 → `400` (AC30).
**Step C-3** — `OTPService.Verify(email, reset_password, code)` (giữ lượt nguyên tử rồi so) → `400 OTP_INVALID` / `400 OTP_EXPIRED` / `429 OTP_LOCKED`. Email không có tài khoản không có mã → `OTP_EXPIRED`.
**Step C-4 — Đổi mật khẩu (một transaction)**
- `PasswordHasher.Hash(new_password)` ở usecase, rồi `ResetPassword(userID, newHash, otpID)`:
  UPDATE `users.password_hash` → `otp_codes.used_at = now()` →
  `UPDATE user_sessions SET is_revoked = true, revoked_at = now(), revoked_reason = 'password_reset' WHERE user_id = $1 AND is_revoked = false` →
  `DELETE FROM login_attempts WHERE email = $2` (gỡ khoá đăng nhập, BR-LGN-05 v1.2).
- `200 {"success": true}`. Thiết bị khác nhận `401` ở request kế tiếp (Step A-2).

## Giới hạn tần suất

- Store: `ratelimit.Store` dùng chung của appService (`rate_limit_store`, Redis; `cmd/ratelimit_store.go`). Mỗi nhóm
  `send` / `verify` / `login` / `refresh` là một `ClientIPRateLimit` (cửa sổ trượt) với scope riêng
  `auth:<nhóm>:ip`; một nhóm = một ngân sách chung cho các route của nó. `logout` không giới hạn.
- Redis lỗi lúc xử lý request → đếm theo instance (A-001), không bao giờ "không giới hạn".
- `429 RATE_LIMITED` do `pkg/ratelimit` ghi: `retry_after_seconds` làm tròn lên, không bao giờ 0, kèm `Retry-After`.
- Ngân sách thư: `BudgetedAuthMailer` dùng key `mail:daily:<YYYY-MM-DD UTC>` (cửa sổ cố định 24h); hết → `RateLimitedError`
  với `RetryAfter` đến 0h UTC, các usecase trả nguyên (không bọc thành `EMAIL_SEND_FAILED`) và nhả lượt gửi của email.
- Ngân sách mặc định và nhóm route: `requirements.md` AC31, AC33.

## Known gaps

- Hash giả ở Step L-4 dùng cùng hàm `PasswordHasher` để thời gian như nhau.
- Hết ngân sách thư: nhánh có tài khoản của `password-reset/request-otp` trả `429` còn nhánh không có tài khoản trả `200` (nhánh sau không gọi mailer) — lộ nhánh chỉ khi ngân sách cả ngày đã cạn. Chấp nhận để AC33 đúng chữ; cách tránh là kiểm ngân sách (không tiêu) ở nhánh không có tài khoản.
- Redis chưa có TLS và store memory không dọn key: `docs/tech-debt/ratelimit/TD-001`, `TD-002`.
- Lỗi mailer khi đặt lại mật khẩu bị nuốt (chỉ log) để không lộ nhánh; người dùng bấm "Gửi lại mã".
