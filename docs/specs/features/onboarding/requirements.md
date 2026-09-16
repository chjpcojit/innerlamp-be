# Requirements — onboarding

Version: 1.2
Last updated: 2026-09-16
Status: implemented — all four user stories shipped and tested; two open caveats (auth middleware not wired, email delivery is a dev stub)
Service package: `services/appService/` (module `identity` / `UserService`)

> See `changelog.md` in this folder for the version history of this spec.

This spec is written **retroactively** from the already-implemented `onboarding` feature
(US-001..US-004), which lives in `services/appService/{domain,usecase,repository,delivery/http}/`
and `migrations/0001..0004`. It documents what the shipped code actually does, not an idealized
version — see `references/modes.md`.

> ℹ️ The acceptance criteria below are a normalized EARS rewrite of the original
> "Given/When/Then" user stories this file carried at v1.0. The behavior is unchanged; only the
> phrasing now follows `references/ears_notation.md`. The Vietnamese user-story prose, field
> tables and "Ghi chú kỹ thuật liên quan" notes from the original are preserved under
> [Original user stories](#original-user-stories) below.

## User Story

As a **new or returning user of the InnerLamp app**, I want **to register, log in, recover a
forgotten password, and maintain my practice profile (dharma name, ordination place, avatar) over
HTTP**, so that **my practice journey is stored server-side and available across devices**.

Four distinct stories are bundled in this feature because they share one domain entity (`User`),
one repository (`domain.UserRepository` → `repository.userRepository`), and one token-issuing
mechanism (`jwt.Service`):

| Story | Endpoint(s) | Priority |
|---|---|---|
| US-001 Register | `POST /v1/auth/register` | P0 |
| US-002 Login | `POST /v1/auth/login` | P0 |
| US-003 Forgot / reset password | `POST /v1/auth/forgot-password`, `POST /v1/auth/reset-password` | P1 |
| US-004 Profile & avatar | `GET /v1/users/me`, `PATCH /v1/users/me`, `POST /v1/users/me/avatar` | P1 |

## Acceptance Criteria (EARS notation)

### US-001 — Register

- WHEN a `POST /v1/auth/register` request carries a valid email, an E.164 phone number, and a
  password of at least 8 characters, THE SYSTEM SHALL create the user row, register the device,
  open a session, and respond `201` with `access_token`, `refresh_token`, and the user profile —
  with no separate email/SMS verification step at this stage.
- WHEN the submitted email already exists, THE SYSTEM SHALL respond `409` with code `EMAIL_TAKEN`
  and message `Email đã được sử dụng`, and SHALL NOT create a duplicate account.
- WHEN the submitted phone number already exists, THE SYSTEM SHALL respond `409` with code
  `PHONE_TAKEN` and message `Số điện thoại đã được sử dụng`, and SHALL NOT create a duplicate
  account.
- WHEN the submitted email fails format validation, THE SYSTEM SHALL respond `400` with code
  `INVALID_EMAIL` and details `{"field":"email"}`.
- WHEN the submitted phone number does not match E.164 (`^\+[1-9]\d{6,14}$`), THE SYSTEM SHALL
  respond `400` with code `INVALID_PHONE` and details `{"field":"phone"}`.
- WHEN the submitted password is shorter than 8 characters, THE SYSTEM SHALL respond `400` with
  code `INVALID_PASSWORD` and details `{"field":"password"}`.
- WHEN the user row is written, THE SYSTEM SHALL store a bcrypt hash of the password and SHALL
  NEVER store the plaintext.
- WHEN `dharma_name` and/or `ordination_place` are omitted, THE SYSTEM SHALL still create the
  account — both fields are optional.

### US-002 — Login

- WHEN a `POST /v1/auth/login` request carries credentials matching an existing account, THE
  SYSTEM SHALL respond `200` with `access_token`, `refresh_token`, and the user profile.
- WHEN the submitted password does not match, THE SYSTEM SHALL respond `401` with code
  `INVALID_CREDENTIALS` and the single generic message `email hoặc mật khẩu không đúng`.
- WHEN the submitted email does not exist, THE SYSTEM SHALL respond with the **same** `401`
  `INVALID_CREDENTIALS` response as a wrong password — the response SHALL NOT disclose which of
  the two was wrong.
- WHEN a login succeeds from a device that has logged in before (same `device_id` fingerprint for
  the same user), THE SYSTEM SHALL update that device row rather than inserting a duplicate, and
  SHALL open an additional session, leaving sessions on other devices untouched.
- WHEN a login succeeds from a device not seen before for that user, THE SYSTEM SHALL insert a new
  device row and a new session.

### US-003 — Forgot / reset password

- WHEN `POST /v1/auth/forgot-password` receives an email that exists, THE SYSTEM SHALL generate a
  32-byte random token, store only its SHA-256 hash with a 15-minute expiry, and send the raw token
  to that email address.
- WHEN `POST /v1/auth/forgot-password` receives an email that does NOT exist, THE SYSTEM SHALL
  respond with the same `200` and the same message as the success case —
  `Nếu email tồn tại trong hệ thống, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu` — and SHALL NOT
  create a reset token or disclose that the address is unknown.
- WHEN the mail provider fails while sending the reset email, THE SYSTEM SHALL log the failure and
  still respond `200` with the same message (the caller must not learn whether delivery succeeded).
- WHEN `POST /v1/auth/reset-password` receives a token that is unexpired and unused along with a
  new password of at least 8 characters, THE SYSTEM SHALL update the password hash, mark the token
  used, and revoke every active session of that user (`is_revoked = true`, `is_active = false`,
  `revoked_reason = 'password_reset'`).
- WHEN the reset token is expired, already used, or unknown, THE SYSTEM SHALL respond `400` with
  code `RESET_TOKEN_INVALID`, and SHALL NOT change the password.
- WHEN the new password is shorter than 8 characters, THE SYSTEM SHALL respond `400` with code
  `INVALID_PASSWORD` and details `{"field":"new_password"}`.

### US-004 — Profile & avatar

- WHEN an authenticated `GET /v1/users/me` arrives, THE SYSTEM SHALL respond `200` with `id`,
  `email`, `phone`, `dharma_name`, `ordination_place`, and `avatar_url`, rendering unset
  optional fields as empty strings rather than `null`.
- WHEN an authenticated `PATCH /v1/users/me` arrives with `dharma_name` and/or `ordination_place`,
  THE SYSTEM SHALL persist both fields and return the updated profile.
- WHEN `dharma_name` or `ordination_place` is sent as an empty string, THE SYSTEM SHALL clear that
  column (store SQL `NULL`), allowing the user to remove previously entered information.
- WHEN a `PATCH /v1/users/me` request attempts to change `email` or `phone`, THE SYSTEM
  SHALL ignore those fields — the update input DTO physically carries no such field, and identity
  changes require a separate verified flow not built in this stage.
- WHEN an authenticated `POST /v1/users/me/avatar` arrives with a body that sniffs as JPEG, PNG, or
  WebP and is at most 5 MB, THE SYSTEM SHALL upload it to the object store at the deterministic
  key `avatars/{userID}`, persist the resulting public URL on the user row, and return the updated
  profile.
- WHEN an avatar upload body exceeds 5 MB, THE SYSTEM SHALL respond `400` with code
  `FILE_TOO_LARGE` and SHALL NOT upload anything.
- WHEN an avatar upload body sniffs as a type outside {JPEG, PNG, WebP}, THE SYSTEM SHALL respond
  `400` with code `INVALID_IMAGE` and `details.detected_type`, and SHALL NOT upload anything.
- WHEN a request to `GET /v1/users/me`, `PATCH /v1/users/me`, or `POST /v1/users/me/avatar` carries
  no valid authenticated user in its context, THE SYSTEM SHALL respond `401` with code
  `UNAUTHORIZED` and message `authentication required`.

> ⚠️ Not confirmed: whether the `401 UNAUTHORIZED` behavior in the last criterion is reachable
> from an external client today — see [Pending](#pending-not-fully-implemented) item 1. The check
> exists inside the handlers, but no middleware populates the context.

- WHEN any of the six routes receives a method it does not handle, THE SYSTEM SHALL respond `405`
  with code `METHOD_NOT_ALLOWED`.
- WHEN any request body is malformed, empty, or larger than the 1 MB decode limit, THE SYSTEM SHALL
  respond `400` with code `INVALID_REQUEST_BODY` and, when a field can be attributed, `details.field`.

## Out of scope (this feature boundary, not backend work)

- Email or SMS/OTP verification of the registered address — no verification step at this stage.
- Social login (Google / Apple).
- Changing email or phone number — needs a re-verification flow (verify new email / OTP).
- Changing password while logged in (distinct from the US-003 forgot-password flow).
- Account deletion (GDPR-like) — needs its own story.
- "Logout everywhere" and 2FA.
- Password reset via SMS/OTP — plausible follow-up since `phone_number` is already collected.
- IP-based rate limiting policy itself — AC6 of US-002 and AC5 of US-003 are handled at the
  middleware/gateway layer, not in this feature's usecases (see `design.md`).

## Pending (not fully implemented)

1. **`RequireAuth` middleware is not wired.** `pkg/jwt/middleware.HTTPAuthMiddleware.Authenticate`
   exists, but `services/appService/cmd/main.go` registers `/v1/users/me` and
   `/v1/users/me/avatar` with plain `mux.Handle(...)` — no middleware wraps them. Today those
   routes reach the handler, which calls `authenticatedUserID(r)`; with no middleware populating
   `jwtcontext`, that call fails and every request returns `401 UNAUTHORIZED`. So the routes are
   *accidentally* closed rather than deliberately protected, and they cannot currently succeed at
   all. Tracked in `tasks.md` Phase 6 item 1.
2. **Email delivery is a dev stub.** `mailer.NewLogMailer(logger)` logs the reset token instead of
   sending mail (marked `⚠️ DEV ONLY` in `cmd/main.go`). A real provider (SendGrid / SES / GCP
   equivalent) is not integrated. Tracked in `tasks.md` Phase 6 item 3.
3. **`pkg/ratelimit` is not applied.** US-002 AC6 and US-003 AC5 (brute-force / email-spam limits)
   are documented as intended behavior but no limiter is wired to either route. Tracked in
   `tasks.md` Phase 6 items 5-6.
4. **Refresh-token rotation and reuse detection are not implemented.** `pkg/jwt.Service` exposes
   `RefreshAccessToken`, but no HTTP endpoint calls it, and `insertSession` always writes
   `token_version = 1` without ever bumping it. Rotating on use (as the original user story
   described) is not what the shipped code does.
5. **No `revoked_at` / `revoked_reason` clearing on login.** A device whose sessions were revoked
   by a password reset is reused via the `ON CONFLICT ... DO UPDATE` upsert, but the new session is
   a fresh row — this is correct, noted only because the device row itself is never marked.

## Open questions (resolved during implementation)

- ~~Is `phone_number` required or optional at registration?~~ → **Required.** The original spec
  flagged this as a guess; the shipped code enforces `UNIQUE NOT NULL` on the column
  (`migrations/0001_register_onboarding.sql`) and rejects a missing/invalid value with
  `INVALID_PHONE`. Resolved 2026-09-15.
- ~~Is the "email already registered" check a separate API call, or part of the register call?~~ →
  **Part of the register call**, enforced by the DB unique constraint `users_email_key` inside the
  same transaction that inserts the user — there is no pre-flight check call. Resolved 2026-09-15.
- ~~Does the reset token need a `refresh_tokens`-style revoke flag, or is a used timestamp
  enough?~~ → **`used_at` timestamp is enough.** `ResetPassword` checks `used_at IS NOT NULL` or
  `expires_at < now()` and returns `RESET_TOKEN_INVALID`; the row is never deleted or revoked.
  Resolved 2026-09-15.
- ~~Does the login response shape differ from register?~~ → **No.** `domain.LoginResult` is a type
  alias for `domain.RegisterResult`, and the login handler reuses `registerResponse` /
  `userResponse` verbatim. Resolved 2026-09-15.
- ~~Is the profile GET response allowed to omit empty fields?~~ → **No, it emits empty strings.**
  `profileResponse` in `profile_handler.go` carries no `omitempty` tags precisely so AC1's "shown
  as not-yet-updated, not `null`" holds. Resolved 2026-09-15.

## Original user stories

The Vietnamese narratives this EARS section was derived from — kept verbatim so the product
intent is not lost in normalization. Field tables and "Ghi chú kỹ thuật liên quan" notes are as
originally written.

### US-001: Đăng ký tài khoản

**Module:** `identity`
**Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

> Là một **người dùng mới**,
> Tôi muốn **đăng ký tài khoản bằng email, mật khẩu và số điện thoại**,
> Để **có thể lưu và đồng bộ hành trình tu tập của mình qua nhiều thiết bị**.

| Trường | Bắt buộc? | Ghi chú |
|---|---|---|
| Email | Bắt buộc | Dùng để đăng nhập, phải unique |
| Mật khẩu | Bắt buộc | ≥ 8 ký tự |
| Số điện thoại | Bắt buộc | Chuẩn hóa định dạng E.164 (`+84...`), không bắt buộc verify OTP ở MVP nhưng lưu sẵn cho tính năng sau (nhắc nhở qua SMS, khôi phục tài khoản...) |
| Pháp danh (dharma name) | Tùy chọn | Tên đạo, nếu đã quy y |
| Nơi quy y / xuất gia | Tùy chọn | Tên chùa/tổ đình nơi quy y hoặc xuất gia |

- Endpoint: `POST /v1/auth/register`
- Access token JWT sống 15 phút, refresh token sống 30 ngày, rotate mỗi lần dùng

> ⚠️ Not confirmed: the "access token 15 phút / refresh token 30 ngày, rotate mỗi lần dùng" note
> contradicts the shipped code — token lifetimes come from config
> (`jwt.access_token_expiration`, `jwt.refresh_token_expiration`, currently `15m` / `168h` in
> `config.dev.yaml`), and rotation is not implemented at all. See
> [Pending](#pending-not-fully-implemented) item 4. Which value is intended for production is
> unresolved.

### US-002: Đăng nhập

**Module:** `identity`
**Priority:** P0

> Là một **người dùng đã có tài khoản**,
> Tôi muốn **đăng nhập bằng email và mật khẩu**,
> Để **tiếp tục hành trình tu tập đã lưu trước đó trên thiết bị này hoặc thiết bị khác**.

- Endpoint: `POST /v1/auth/login`
- Dùng `bcrypt.CompareHashAndPassword` để so khớp mật khẩu, không so sánh plaintext
- Rate limiting (AC6) nên implement ở middleware layer hoặc API gateway, không phải trong usecase
  — tránh lẫn logic bảo mật hạ tầng vào logic nghiệp vụ
- Response giống hệt US-001 (`access_token`, `refresh_token`, thông tin user)

### US-003: Quên mật khẩu

**Module:** `identity`
**Priority:** P1

> Là một **người dùng đã có tài khoản nhưng quên mật khẩu**,
> Tôi muốn **đặt lại mật khẩu thông qua email đã đăng ký**,
> Để **có thể tiếp tục đăng nhập mà không cần tạo tài khoản mới**.

- Endpoints: `POST /v1/auth/forgot-password { email }`, `POST /v1/auth/reset-password { token, new_password }`
- Reset token lưu dạng hash, có `expires_at`, đánh dấu đã dùng bằng `used_at` thay vì revoke
  (bảng `password_reset_tokens`)
- Cần dịch vụ gửi email (SendGrid, AWS SES, hoặc GCP tương đương) — dependency ngoại vi mới, đã bổ
  sung vào `pkg/mailer` nhưng hiện chỉ có `LogMailer` cho dev
- Rate limiting (AC5) cùng cơ chế với US-002 AC6, áp dụng theo email thay vì theo IP

### US-004: Cập nhật hồ sơ cá nhân

**Module:** `identity`
**Priority:** P1

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **cập nhật pháp danh, nơi quy y/xuất gia và ảnh đại diện của mình**,
> Để **hồ sơ tu tập phản ánh đúng thông tin cá nhân, đặc biệt khi tôi mới quy y sau khi đã tạo tài khoản**.

- Endpoints: `GET /v1/users/me`, `PATCH /v1/users/me` (yêu cầu `RequireAuth` middleware),
  `POST /v1/users/me/avatar`
- Ảnh đại diện: upload lên Cloud Storage, lưu `avatar_url` trong bảng `users`; endpoint upload
  riêng vì `PATCH /v1/users/me` chỉ nhận JSON, không nhận file trực tiếp
- Thuộc `UserService` trong `usecase/` (tách khỏi `AuthService` vì đây là CRUD hồ sơ, không phải
  nghiệp vụ xác thực)
