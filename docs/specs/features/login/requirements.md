# Requirements — login

Version: 2.5
Last updated: 2026-10-16
Status: implemented (v2.5, 2026-10-16) — đăng nhập bằng email, khoá theo email, quên mật khẩu bằng mã qua email, giới hạn tần suất mọi `/v1/auth/*`, chặn access token của phiên đã thu hồi. Phần phiên/refresh/logout giữ nguyên từ v1.13.
Module: `identity`
Code: `services/appService/`
Product spec: `docs/product/login/product.md` @ v1.2 (và `docs/product/onboarding/product.md` @ v1.2 cho BR-ONB-05/06/09)
Contract: `docs/specs/api/openapi.yaml` (this repo) — đã khớp bảng API bên dưới (v2.5)
Implements: BR-LGN-01, BR-LGN-02, BR-LGN-03, BR-LGN-05, BR-LGN-07, FR-LGN-001, FR-LGN-002, FR-LGN-003, FR-LGN-004, FR-LGN-006, FR-LGN-008, FR-LGN-009, NFR-LGN-01, NFR-LGN-02
Architecture: ADR-0003 (Accepted 2026-10-14)

> See `changelog.md` in this folder for the version history of this spec.

v2.0 viết lại spec và **gộp quên mật khẩu** (trước ở `password-reset/`, chưa từng có code) vào đây
thành US-003. Ngoài phạm vi backend của product spec: BR-LGN-04, BR-LGN-06, FR-LGN-005, FR-LGN-007,
FR-LGN-010, NFR-LGN-03 (chỉ client).

---

## US-002: Đăng nhập, giữ phiên, đăng xuất

**Priority:** P0

> Là một **người dùng đã có tài khoản**,
> Tôi muốn **đăng nhập bằng email và mật khẩu và giữ phiên trên từng thiết bị**,
> Để **tiếp tục hành trình tu tập trên bất kỳ thiết bị nào**.

### Acceptance Criteria (EARS notation)

`login`, `refresh` public; `logout` cần access token.

**Login — `POST /v1/auth/login`**

1. WHEN a request arrives with an email (normalised per `rules/email.json`), a password and a valid `device` that match a user whose `email_verified_at` is set, THE SYSTEM SHALL return `200` with `access_token`, `refresh_token`, `expires_in` and `user`, and create a session bound to that device. (FR-LGN-001, BR-LGN-01)
2. WHEN the password is wrong, the email has no account, the email exists only in a pending registration, or the account's `email_verified_at` is `NULL`, THE SYSTEM SHALL return the same `401 INVALID_CREDENTIALS` with message "Email hoặc mật khẩu không đúng", and SHALL run a password comparison against a fixed dummy hash when no account was found so the response time does not differ. (FR-LGN-002, BR-LGN-01, NFR-LGN-01)
3. WHEN `email` is malformed, THE SYSTEM SHALL reply `400 INVALID_EMAIL`; WHEN `password` is empty, `400 INVALID_PASSWORD`; no other password rule applies at sign-in. Neither counts as a failed attempt. (BR-LGN-01)
4. WHEN `device` is missing or lacks `platform`/`device_model`, THE SYSTEM SHALL reply `400 INVALID_DEVICE` and not count an attempt. (platform — unchanged from v1.13)
5. WHEN an attempt arrives for an email, THE SYSTEM SHALL reserve it atomically in `login_attempts` (keyed by normalised email, for emails with or without an account) before comparing the password; the fifth consecutive failure, each within `login.failure_reset_window` (15m) of the previous one, SHALL lock that email for `login.lock_duration` (15m). WHILE locked, THE SYSTEM SHALL reply `429 LOGIN_LOCKED` with `locked_until` without checking the password. A successful login SHALL reset the counter. (FR-LGN-003, BR-LGN-02)
6. WHEN a login comes from a device the user has not used, THE SYSTEM SHALL add a session without touching other devices' sessions; WHEN from a device already used, THE SYSTEM SHALL revoke that device's previous session and create a new one. (FR-LGN-004, BR-LGN-03 — unchanged)
7. WHEN a login succeeds, THE SYSTEM SHALL issue an access token (`jwt.access_token_expiration`, 15m) carrying `sid`, and a refresh token (168h) stored only as its SHA-256 hash. (BR-LGN-03 — unchanged)
8. WHEN a login succeeds, THE SYSTEM SHALL compare the password with the same hashing as registration (`bcrypt(base64(SHA-256(NFC(password))))`). (BR-ONB-06)

**Refresh — `POST /v1/auth/refresh`** (không đổi so với v1.13)

9. WHEN a refresh token of an active, unexpired, unrevoked session arrives, THE SYSTEM SHALL return `200` with a new access token (same `sid`), a new refresh token, and `expires_in`, extend the session to now + 7 days (sliding), and make the old refresh token unusable. (FR-LGN-004, BR-LGN-03)
10. WHEN the refresh token is malformed, badly signed, expired, of a revoked session or unknown, THE SYSTEM SHALL reply `401 REFRESH_TOKEN_INVALID`. (BR-LGN-03)
11. WHEN the token immediately preceding the session's current one is presented, THE SYSTEM SHALL reply `401 REFRESH_TOKEN_INVALID` and revoke that device's session (`refresh_token_reuse`). (BR-LGN-03)
12. WHEN the session has no device (created at registration), THE SYSTEM SHALL handle refresh and logout the same way. (platform)

**Logout — `POST /v1/auth/logout`** (không đổi)

13. WHEN a valid access token whose `sid` is an active session of its user arrives, THE SYSTEM SHALL revoke that session (`logout`) and return `200 {"success": true}`. (FR-LGN-006, BR-LGN-07)
14. WHEN the `sid` matches no active session of that user, THE SYSTEM SHALL still return `200` and change nothing. (FR-LGN-006 — idempotent, covers an offline sign-out that is retried)
15. WHEN the access token is missing, invalid, of type refresh or lacks `sid`, THE SYSTEM SHALL reply `401 UNAUTHORIZED`. (platform)

**Web client và response cấp token** (không đổi so với v1.13 AC16–AC22)

16. WHEN a `/v1/auth/*` request carries `X-API-Key`, THE SYSTEM SHALL accept it only if it equals `server.web_api_key_secret` (constant-time) and otherwise reply `401 INVALID_API_KEY`; a request without the header is a mobile caller. (platform)
17. WHEN `device.platform` is `web` without a valid `X-API-Key`, THE SYSTEM SHALL reply `401 INVALID_API_KEY`; an empty platform is stored as `mobile`. (platform)
18. WHEN `verify-otp`, `login`, `refresh`, `password-reset/request-otp` or `password-reset/confirm` responds, THE SYSTEM SHALL send `Cache-Control: no-store` and `Pragma: no-cache`. (platform)
19. WHEN `verify-otp`, `login` or `refresh` succeeds, `data.expires_in` SHALL equal the access token lifetime in seconds. (platform)
20. WHEN `login` or `verify-otp` succeeds, `data.user` SHALL have the shape `{ id, email, phone_number, full_name, dharma_name?, ordination_temple_id?, created_at }`. (platform)

**Phiên đã thu hồi (mới v2.0)**

21. WHEN a request to any route behind the access-token middleware carries an access token whose `sid` session is revoked or expired, THE SYSTEM SHALL reply `401 UNAUTHORIZED`, so a password reset or logout takes effect on the device's next request rather than when the access token expires. (FR-LGN-009 "next request", BR-LGN-03)
22. WHEN that check cannot reach the database, THE SYSTEM SHALL reply `503` rather than let the request through. (platform)

---

## US-003: Quên mật khẩu (mã qua email)

**Priority:** P1

> Là một **người dùng quên mật khẩu**,
> Tôi muốn **nhận mã qua email đã đăng ký và đặt mật khẩu mới**,
> Để **lấy lại tài khoản mà không cần SMS hay tạo tài khoản mới**.

### Acceptance Criteria (EARS notation)

**Xin mã — `POST /v1/auth/password-reset/request-otp`** (gọi lại = gửi lại)

23. WHEN a request arrives with an email that has a verified account, THE SYSTEM SHALL issue a code for purpose `reset_password` under the send policy of BR-ONB-09, email it, and return `200` with message "Nếu email này có tài khoản, mã đã được gửi", `expires_in_seconds` and `resend_after_seconds`. (FR-LGN-008, BR-LGN-05)
24. WHEN the email has no verified account, THE SYSTEM SHALL return the same `200` body, apply the same send policy to `(email, reset_password)` so a `429` looks the same, and send no email. (FR-LGN-008, NFR-LGN-01)
25. WHEN either branch answers, THE SYSTEM SHALL not answer sooner than `password_reset.min_response_time` (default 1.5s) after the request started, so the mail call does not reveal the branch. (NFR-LGN-01)
26. WHEN the send policy refuses, THE SYSTEM SHALL reply `429 OTP_RESEND_TOO_SOON` / `OTP_SEND_LIMITED`; WHEN the current code is wrong-guess-locked, `429 OTP_LOCKED`. (BR-ONB-09)
27. WHEN `email` is malformed, THE SYSTEM SHALL reply `400 INVALID_EMAIL` before any database access. (BR-ONB-05)

**Đặt mật khẩu mới — `POST /v1/auth/password-reset/confirm`**

28. WHEN the right, unexpired, unused code and a `new_password` valid per `rules/password.json` arrive, THE SYSTEM SHALL in one transaction update the password hash, mark the code used, revoke every session of the user (`password_reset`) and delete the email's `login_attempts` row so a sign-in lock is lifted, and return `200 {"success": true}` without issuing tokens. (FR-LGN-009, BR-LGN-05)
29. WHEN the code is wrong, THE SYSTEM SHALL reply `400 OTP_INVALID`, counting the attempt atomically; the third miss locks the code (`429 OTP_LOCKED`). WHEN the code is expired, used or was never issued for this email (including an email with no account), `400 OTP_EXPIRED`. (FR-LGN-009, BR-ONB-09, NFR-LGN-01)
30. WHEN `new_password` breaks BR-ONB-06, THE SYSTEM SHALL reply `400 INVALID_PASSWORD`; a malformed email, `400 INVALID_EMAIL` — both before any database access. (FR-LGN-009)

---

## Giới hạn tần suất cho mọi `/v1/auth/*` (NFR-ONB-02, NFR-LGN-02)

31. WHEN a request to a `/v1/auth/*` route exceeds its per-IP budget for the current window, THE SYSTEM SHALL reply `429 RATE_LIMITED` with `retry_after_seconds` and run no handler. Budgets are config (`auth_rate_limit.*`), defaults: `send` (register request-otp, register resend-otp, password-reset request-otp) 10/hour; `verify` (register verify-otp, password-reset confirm) 30/hour; `login` 30/15 min; `refresh` 120/15 min; `logout` not limited. (NFR-LGN-02, NFR-ONB-02)
32. WHEN counting, THE SYSTEM SHALL keep the per-IP counters and the daily mail counter in the service's shared rate-limit store (`rate_limit_store`: Redis, so the limit holds across Cloud Run instances; a Redis error at request time falls back to a per-instance count, never to "no limit"). (platform; supersedes the Postgres `auth_rate_limits` table of v2.0–v2.3, see Open questions)
33. WHEN the number of emails sent today (UTC) reaches `mailer.daily_budget` (default 100), THE SYSTEM SHALL send no more and reply `429 RATE_LIMITED` with `retry_after_seconds` until the next UTC day, for every route that would send. (NFR-ONB-02)
34. WHEN resolving the client IP for `/v1/auth/*` (rate limits AC31 and the session's device metadata), THE SYSTEM SHALL use, in order: (a) the `X-InnerLamp-Client-IP` header when the request carries a valid `X-API-Key` (AC16) and the header parses as one IPv4/IPv6 address — the website's server forwards its visitor's IP there; (b) otherwise, when `auth_rate_limit.trust_forwarded_for` is true, the right-most `X-Forwarded-For` entry added by the trusted proxy; (c) otherwise the connection's address. A request without a valid key never has `X-InnerLamp-Client-IP` read; an unparsable value falls through to (b)/(c). (NFR-LGN-02, platform)
35. WHEN per-email limits are needed, THE SYSTEM SHALL rely on the existing per-email counters: send policy (`otp_send_limits`), code attempts (`otp_codes`) and the sign-in lock (`login_attempts`). (NFR-LGN-02)

## API contract (confirmed surface for this feature)

| Method | Path | Auth | Request | Success | Failure codes |
|---|---|---|---|---|---|
| POST | `/v1/auth/login` | public | `{ email, password, device: { platform, device_model, os_version?, app_version?, device_name? } }` | `200` `{ access_token, refresh_token, expires_in, user }` | `400 INVALID_EMAIL / INVALID_PASSWORD / INVALID_DEVICE`, `401 INVALID_CREDENTIALS / INVALID_API_KEY`, `429 LOGIN_LOCKED (+locked_until) / RATE_LIMITED` |
| POST | `/v1/auth/refresh` | public | `{ refresh_token }` | `200` `{ access_token, refresh_token, expires_in }` | `401 REFRESH_TOKEN_INVALID / INVALID_API_KEY`, `429 RATE_LIMITED` |
| POST | `/v1/auth/logout` | Bearer access | — | `200` `{ success: true }` | `401 UNAUTHORIZED / INVALID_API_KEY` |
| POST | `/v1/auth/password-reset/request-otp` | public | `{ email }` | `200` `{ message, expires_in_seconds, resend_after_seconds }` | `400 INVALID_EMAIL`, `429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED / OTP_LOCKED / RATE_LIMITED` |
| POST | `/v1/auth/password-reset/confirm` | public | `{ email, otp_code, new_password }` | `200` `{ success: true }` | `400 INVALID_EMAIL / INVALID_PASSWORD / OTP_INVALID / OTP_EXPIRED`, `429 OTP_LOCKED / RATE_LIMITED` |

Mọi route Bearer khác (memorials, genealogies, lamps, users/me…) thêm `401 UNAUTHORIZED` khi phiên đã
bị thu hồi (AC21). `password-reset/request-otp` không trả `500 EMAIL_SEND_FAILED`: lỗi mailer được log,
response vẫn như AC23 để không lộ nhánh.

### Thay đổi so với code hiện tại (v1.13)

| Hiện tại | v2.0 |
|---|---|
| `login {phone_number, …}`, tra `FindUserByPhone` | `login {email, …}`, tra theo email, yêu cầu `email_verified_at` |
| "Số điện thoại hoặc mật khẩu không đúng" | "Email hoặc mật khẩu không đúng" |
| `login_attempts` khoá theo `phone_number`, đọc rồi so | Khoá theo `email`, giữ lượt nguyên tử trước khi so (AC5) |
| bcrypt thẳng trên mật khẩu | bcrypt trên SHA-256 của NFC (AC8) |
| Access token vẫn dùng được ≤ 15 phút sau khi thu hồi phiên | Middleware kiểm phiên còn sống (AC21) |
| Không có quên mật khẩu | `password-reset/request-otp`, `password-reset/confirm` (US-003) |
| Không giới hạn theo IP ở `/v1/auth/*` | `429 RATE_LIMITED` (AC31–AC34) |

## Out of scope (this feature boundary, not tracked here)

- Đăng nhập bằng mã một lần, Google/Apple, 2FA, "đăng xuất mọi thiết bị" thủ công, đổi mật khẩu khi đang đăng nhập: để sau (product §1.2).
- Đăng nhập back office: khác ranh giới tin cậy (ADR-0003 rule 9).
- Khôi phục khi mất email: admin xử lý tay qua back office (ADR-0003 rule 5).
- Nơi client lưu token: platform spec của từng client (NFR-LGN-03).

## Pending (not yet implemented)

Không còn AC nào chưa có code. `openapi.yaml` đã cập nhật (v2.5). Còn lại là tài liệu theo sau (`tasks.md` Phase 7):
`docs/knowledge/features/login.md`, product spec §4.2.

## Ghi chú kỹ thuật liên quan

- **AC21 đảo quyết định 2026-09-21** ("access token stateless, không tra `user_sessions`"): product
  v1.1 FR-LGN-009 yêu cầu thiết bị khác bị đăng xuất ở request kế tiếp sau khi đặt lại mật khẩu. Chi phí:
  một truy vấn theo khoá `session_id` (UNIQUE) cho mỗi request có Bearer.
- **Khoá đăng nhập theo email cho cả email không tồn tại** để khoá không lộ email nào có tài khoản.
- **Không có `resend-otp` riêng cho đặt lại mật khẩu:** gọi lại `request-otp` là gửi lại, chịu cùng send
  policy (FR-LGN-008 "Gửi lại mã").
- **Thời gian tối thiểu (AC25)** thay cho việc gửi thư nền: Cloud Run chỉ cấp CPU trong lúc xử lý
  request nên gửi sau khi trả lời không đáng tin.
- **Không backfill:** tài khoản dev tạo trước migration `0037` không đăng nhập được (onboarding v8.0).

## Open questions (resolved during implementation)

- ~~Website gọi từ server: giới hạn theo IP sẽ thấy IP của server website?~~ → Website gửi IP khách ở `X-InnerLamp-Client-IP`; backend chỉ đọc khi `X-API-Key` hợp lệ (AC34). Header riêng thay vì `X-Forwarded-For` vì Cloud Run nối thêm IP của server website vào XFF. Anh Đức đồng ý 2026-10-14. Resolved 2026-10-14.
- ~~Đặt lại mật khẩu có gỡ khoá đăng nhập không?~~ → Có (anh Đức: "Gỡ như em đề xuất"; product login v1.2 BR-LGN-05). AC28 xoá `login_attempts` trong cùng transaction. Resolved 2026-10-14.
- ~~Đăng nhập bằng gì?~~ → Email + mật khẩu (ADR-0003 rule 1). Resolved 2026-10-14.
- ~~Quên mật khẩu đi kênh nào?~~ → Mã qua email đã xác minh; đăng xuất mọi thiết bị; không tự đăng nhập (ADR-0003 rule 5, product BR-LGN-05). Resolved 2026-10-14.
- ~~Đếm giới hạn ở đâu?~~ → ~~Postgres, không Redis~~ (2026-10-14) **đảo 2026-10-16: Redis**, store dùng chung của commit `fc31897` (`rate_limit_store`). Lý do: store đó đã có, đã bọc fallback memory và ping lúc khởi động, và mọi limiter khác của appService đã dùng nó; một bảng Postgres thứ hai chỉ để đếm là thêm một truy vấn ghi cho mỗi request `/v1/auth/*`. Anh Đức chọn Redis 2026-10-16. Resolved 2026-10-16.
- ~~Folder cho quên mật khẩu?~~ → Gộp vào `login/` như product spec; `password-reset/` bị xoá ở bước dọn tài liệu cũ. Resolved 2026-10-14.
- Các quyết định của v1.13 (device, rotation, reuse detection, sliding 7 ngày, `sid`, `X-API-Key`) giữ nguyên — xem `changelog.md` v1.x.
