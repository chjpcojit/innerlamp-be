# Design — onboarding

Version: 1.2
Last updated: 2026-09-16
Status: implemented — all four user stories shipped and tested; two open caveats (auth middleware not wired, email delivery is a dev stub)
Module: `identity` (spec folder `onboarding`)

> See `changelog.md` in this folder for the version history of this spec.

This spec is written **retroactively** from the already-implemented feature. Every file named in
the layers table below exists and was read directly — nothing here is a planned-but-absent file.

## Overview

`identity` is the foundation module: it owns the `users` row, the device/session pair that backs
every JWT, and the password-recovery flow. It is the **user-scoped write** shape of this repo — the
opposite of `content`, which is public read. Four user stories (US-001..004) share one domain
entity (`domain.User`), one repository interface (`domain.UserRepository`, 9 methods) and one
token-minting mechanism (`pkg/jwt.Service` injected as a `domain.TokenIssuer` closure).

Routes split by auth model:

| Route | Auth model |
|---|---|
| `POST /v1/auth/register` | public — creates the identity |
| `POST /v1/auth/login` | public — proves the identity |
| `POST /v1/auth/forgot-password` | public — must stay public (user is locked out) |
| `POST /v1/auth/reset-password` | public — authorization is the reset token itself |
| `GET`/`PATCH /v1/users/me` | user-scoped — handler reads the user ID from request context |
| `POST /v1/users/me/avatar` | user-scoped — same |

No HTTP DTO leaks inward: all request/response shapes live in `delivery/http/*_handler.go`, and
`domain/` carries no JSON tags. Validation is split by concern — format/shape checks in the
usecase (`pkg/validation` + manual length/regex checks), transport limits (body size, image MIME
sniffing) in the handler.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/user.go` | `User`, `Device`, `Session`. Plain data, no methods, no tags. |
| Domain / Input-Output | `services/appService/domain/register.go` | `DeviceInfo`, `RegisterInput`, `RegisterResult`, `TokenIssuer` func type, and the `UserRepository` interface itself. |
| Domain / Input-Output | `services/appService/domain/login.go` | `LoginInput`; `LoginResult` is a type alias of `RegisterResult` — the response shapes are intentionally identical. |
| Domain / Input-Output | `services/appService/domain/password_reset.go` | `PasswordResetToken`, `ForgotPasswordInput`, `ResetPasswordInput`. |
| Domain / Input-Output | `services/appService/domain/profile.go` | `ProfileUpdateInput` — deliberately has only `DharmaName` and `OrdinationPlace`, so "email/phone are not editable here" is enforced by the type, not by a runtime check. |
| Domain / Errors | `services/appService/domain/errors.go` | Sentinels: `ErrEmailTaken`, `ErrPhoneTaken`, `ErrInvalidEmail`, `ErrInvalidPhone`, `ErrInvalidPassword`, `ErrUserNotFound`, `ErrInvalidCredentials`, `ErrResetTokenInvalid`. Handler maps each to a status code. |
| Domain / Mock | `services/appService/domain/mock_user_repository.go` | Mock of `UserRepository` for usecase tests — no real DB. |
| Usecase | `services/appService/usecase/register_usecase.go` | Normalize email → validate email → E.164 regex → password length → bcrypt hash → hand off to repository with a `TokenIssuer` closure. Owns `deviceFingerprint` (SHA-256 of platform/OS/model/UA). |
| Usecase | `services/appService/usecase/login_usecase.go` | Normalize email → `FindByEmail` → bcrypt compare → `CreateSession`. Collapses both "no such user" and "wrong password" into `ErrInvalidCredentials`. |
| Usecase | `services/appService/usecase/password_reset_usecase.go` | `ForgotPassword` (no-enumeration, token issue, mail) and `Reset` (validate, hash, hand off). Owns `resetTokenTTL` (15m), `generateResetToken` (32 bytes `crypto/rand`, hex) and `hashResetToken` (SHA-256). |
| Usecase | `services/appService/usecase/profile_usecase.go` | `GetProfile` / `UpdateProfile` — thin pass-through to the repository; the absent fields are the enforcement. |
| Usecase | `services/appService/usecase/avatar_usecase.go` | Defines the consumer-side port `AvatarStorage` (not in `domain/`), builds the deterministic object name `avatars/{userID}`, uploads, then persists the URL. |
| Repository | `services/appService/repository/postgres_user_repository.go` | All SQL and all transaction boundaries for this feature. Shared `insertSession` helper backs both `Register` and `CreateSession`. Translates `pq` unique violations (`23505` + constraint name) into domain sentinels. |
| Repository | `services/appService/repository/gcs_avatar_storage.go` | Implements `usecase.AvatarStorage` over `pkg/storage`; uploads with `ACLPublicRead` and returns a public `storage.googleapis.com` URL. |
| Delivery / Handler | `services/appService/delivery/http/register_handler.go` | DTOs, 201 response, error mapper; fills `IPAddress`/`UserAgent` server-side. Also defines the shared `clientIP` helper. |
| Delivery / Handler | `services/appService/delivery/http/login_handler.go` | DTOs (reuses `registerResponse`), 200 response, error mapper. |
| Delivery / Handler | `services/appService/delivery/http/forgot_password_handler.go` | Single-field DTO, fixed non-enumerating message, no error mapper (any usecase error → 500). |
| Delivery / Handler | `services/appService/delivery/http/reset_password_handler.go` | DTO, error mapper for `ErrResetTokenInvalid` / `ErrInvalidPassword`. |
| Delivery / Handler | `services/appService/delivery/http/profile_handler.go` | Method switch (GET/PATCH), auth-context read, `writeUnauthorized` + `newProfileResponse` shared helpers. |
| Delivery / Handler | `services/appService/delivery/http/avatar_handler.go` | 5 MB `MaxBytesReader`, `http.DetectContentType` sniffing against a JPEG/PNG/WebP allowlist, raw-body (no multipart) request. |
| Delivery / Shared | `services/appService/delivery/http/auth_context.go` | `authenticatedUserID(r)` — reads `jwtcontext.GetUserID(r.Context())` and parses it to `int64`. This is the single point where every user-scoped handler obtains its caller identity. |
| Wiring | `services/appService/cmd/main.go` | Constructor injection in dependency order + `mux.Handle` per route. Touches three `⚠️ DEV ONLY` blocks (see below). |
| Migration | `migrations/0001_register_onboarding.sql` | Adds `users.phone_number`, `users.ordination_place`; creates `user_devices` and `user_sessions`. |
| Migration | `migrations/0002_login_device_unique.sql` | Adds `UNIQUE (user_id, device_id)` on `user_devices` — the conflict target that makes login's device upsert possible. |
| Migration | `migrations/0003_password_reset.sql` | Creates `password_reset_tokens` + its unique index on `token_hash`. |
| Migration | `migrations/0004_profile_avatar.sql` | Adds `users.avatar_url`. |
| API spec | `docs/specs/api/openapi.yaml` | Contract for all six routes; must be updated in the same change as any handler edit. |

## Request flow

```
POST /v1/auth/register
  └─ public (no middleware)
       └─ RegisterHandler.ServeHTTP                  (delivery/http)
            ├─ DecodeJSONBodyStrict → registerRequest
            ├─ clientIP(r) / r.UserAgent() filled server-side
            └─ RegisterUsecase.Register(ctx, RegisterInput)   (usecase)
                 ├─ normalize email → ValidateEmail → E.164 regex → len(password) >= 8
                 ├─ bcrypt.GenerateFromPassword
                 └─ UserRepository.Register(ctx, user, device, issuer)   (repository)
                      └─ ONE TX: INSERT users → INSERT user_devices → INSERT user_sessions
                                 (JWT minted inside the tx via the TokenIssuer closure)

POST /v1/auth/login
  └─ public
       └─ LoginHandler.ServeHTTP
            └─ LoginUsecase.Login(ctx, LoginInput)
                 ├─ normalize email
                 ├─ UserRepository.FindByEmail       → ErrUserNotFound ⇒ ErrInvalidCredentials
                 ├─ bcrypt.CompareHashAndPassword    → mismatch       ⇒ ErrInvalidCredentials
                 └─ UserRepository.CreateSession
                      └─ ONE TX: UPSERT user_devices ON CONFLICT (user_id, device_id)
                                 → INSERT user_sessions

POST /v1/auth/forgot-password
  └─ public
       └─ PasswordResetUsecase.ForgotPassword
            ├─ UserRepository.FindByEmail  → ErrUserNotFound ⇒ return nil (200, no enumeration)
            ├─ generateResetToken (32B random hex) + SHA-256 hash
            ├─ UserRepository.CreatePasswordResetToken (expires_at = now() + 15m)
            └─ Mailer.SendPasswordResetEmail (raw token)   — failure logged, still returns nil

POST /v1/auth/reset-password
  └─ public
       └─ PasswordResetUsecase.Reset
            ├─ len(new_password) >= 8 → bcrypt hash
            └─ UserRepository.ResetPassword(ctx, tokenHash, newHash)
                 └─ ONE TX: SELECT ... FOR UPDATE → UPDATE users.password_hash
                            → UPDATE password_reset_tokens.used_at
                            → UPDATE user_sessions SET is_revoked, is_active=false,
                                                        revoked_reason='password_reset'

GET|PATCH /v1/users/me
  └─ authenticatedUserID(r)   — reads jwtcontext; 401 UNAUTHORIZED if absent
       └─ ProfileUsecase.GetProfile | UpdateProfile
            └─ UserRepository.FindByID | UpdateProfile

POST /v1/users/me/avatar
  └─ authenticatedUserID(r)   — 401 UNAUTHORIZED if absent
       └─ AvatarHandler: MaxBytesReader(5MB) → DetectContentType → allowlist check
            └─ AvatarUsecase.UpdateAvatar(ctx, userID, reader, contentType)
                 ├─ AvatarStorage.Upload → object avatars/{userID}, public-read
                 └─ UserRepository.UpdateAvatarURL
```

## Step-by-step Flow — Registration and session issue

```
Step 1 — Validate and hash (this module)
- Trigger: RegisterHandler receives POST /v1/auth/register with a decodable body
- Pre-condition: DecodeJSONBodyStrict passed (unknown fields rejected, 1 MB cap)
- Call: RegisterUsecase.Register — validation.NormalizeEmail → ValidateEmail → phoneE164Regex
  (^\+[1-9]\d{6,14}$) → len(password) >= 8 → bcrypt.GenerateFromPassword(DefaultCost)
    - Success → Step 2
    - Failure (ErrInvalidEmail) → 400 INVALID_EMAIL, details {field: email}; no DB touched
    - Failure (ErrInvalidPhone) → 400 INVALID_PHONE, details {field: phone}
    - Failure (ErrInvalidPassword) → 400 INVALID_PASSWORD, details {field: password}
    - Failure (bcrypt error) → wrapped "hash password: %w" → 500 INTERNAL_ERROR
  Note: the phone regex is evaluated on the raw trimmed string, NOT normalized to E.164 first —
  a local-format number ("0901234567") is rejected rather than coerced.

Step 2 — Duplicate check and insert, in one transaction (this module)
- Trigger: Step 1 passed
- Pre-condition: password_hash is populated on the in-memory User
- Call: UserRepository.Register → BEGIN
        INSERT INTO users (email, password_hash, phone_number, dharma_name, ordination_place)
          RETURNING id, created_at, updated_at
  Confirmed: there is NO separate pre-flight "does this email exist" query. Uniqueness is enforced
  solely by the DB constraints, and the violation is translated after the fact. This is why a race
  between two concurrent registrations cannot produce a duplicate — the loser gets the same 409.
    - Success → Step 3
    - Failure (users_email_key, 23505) → domain.ErrEmailTaken → 409 EMAIL_TAKEN, tx rolled back
    - Failure (users_phone_number_key, 23505) → domain.ErrPhoneTaken → 409 PHONE_TAKEN, rolled back
    - Failure (any other DB error) → wrapped "insert user: %w" → 500, rolled back

Step 3 — Register device (this module, same tx as Step 2)
- Trigger: Step 2 succeeded
- Pre-condition: user.ID known; device.DeviceID already computed by deviceFingerprint()
  (SHA-256 over Platform|OSVersion|DeviceModel|UserAgent — a stable fingerprint, not a client-
  supplied ID)
- Call: INSERT INTO user_devices (...) RETURNING id, created_at
    - Success → Step 4
    - Failure → wrapped "insert device: %w" → 500; the whole tx rolls back, so the user row from
      Step 2 does NOT survive. Registration is all-or-nothing.

Step 4 — Mint tokens and open the session (this module, same tx as Steps 2-3)
- Trigger: device row exists
- Pre-condition: user.ID known
- Call: the TokenIssuer closure supplied by RegisterUsecase →
        jwtService.GenerateAccessToken(userID, email, "user", nil)   [15m]
        jwtService.GenerateRefreshToken(userID, nil)                 [configured expiration]
      then INSERT INTO user_sessions (..., session_id = uuid, token_version = 1, is_active = true,
      is_revoked = false, expires_at = now() + sessionTTL)
  Confirmed: sessionTTL is 7 days (`const sessionTTL = 7 * 24 * time.Hour` in
  postgres_user_repository.go) — deliberately INDEPENDENT of the refresh-token expiration from
  config. The DB row can expire before the token it stores, or vice versa.
    - Success → COMMIT → 201 {access_token, refresh_token, user}
    - Failure (token generation) → wrapped "issue tokens: %w" → 500; tx rolls back
    - Failure (session insert) → wrapped "insert session: %w" → 500; tx rolls back
  Transaction boundary: Steps 2, 3 and 4 share ONE transaction. Any failure in Steps 2-4 rolls
  back all three writes — there is no partially-registered user. This is the opposite of the naive
  "INSERT user, then issue token, accept orphan on failure" design; the cost is that the JWT is
  minted while holding a DB transaction open.
```

## Step-by-step Flow — Login

```
Step 1 — Identify the user (this module)
- Trigger: LoginHandler receives POST /v1/auth/login
- Pre-condition: body decoded
- Call: normalize email → UserRepository.FindByEmail
    - Success → Step 2
    - Failure (ErrUserNotFound) → collapsed to domain.ErrInvalidCredentials → 401. The caller
      cannot distinguish this branch from Step 2's failure.
    - Failure (other DB error) → wrapped "find user: %w" → 500
  Note: login performs NO format validation on the email — unlike register. A malformed address
  simply misses FindByEmail and lands on ErrInvalidCredentials.

Step 2 — Verify the password (this module)
- Trigger: Step 1 found a row
- Pre-condition: user.PasswordHash is a bcrypt hash
- Call: bcrypt.CompareHashAndPassword(user.PasswordHash, input.Password)
    - Success → Step 3
    - Failure (mismatch) → domain.ErrInvalidCredentials → 401 INVALID_CREDENTIALS, same code and
      message as Step 1's failure (US-002 AC2/AC3 — no enumeration)
  Note: no timing-equalisation is attempted between the "no such user" and "wrong password" paths.

Step 3 — Upsert device and open a new session (this module)
- Trigger: Step 2 succeeded
- Pre-condition: same deviceFingerprint() algorithm as register
- Call: UserRepository.CreateSession → BEGIN
        INSERT INTO user_devices (...) ON CONFLICT (user_id, device_id) DO UPDATE SET <metadata>
          RETURNING id, created_at
        then INSERT INTO user_sessions (...)
  Confirmed: logging in again from a device whose fingerprint is unchanged REUSES the existing
  user_devices row and adds a SECOND user_sessions row. Old sessions are not revoked — US-002 AC4
  (per-device sessions, no implicit logout) is implemented by this insert-only behaviour.
    - Success → COMMIT → 200 {access_token, refresh_token, user}
    - Failure (any) → 500; tx rolls back, no session row is left behind
  Transaction boundary: the device upsert and the session insert share one transaction.
```

## Step-by-step Flow — Password reset

```
Step 1 — Request a reset (this module)
- Trigger: POST /v1/auth/forgot-password
- Pre-condition: body decoded
- Call: normalize email → UserRepository.FindByEmail
    - Success (user found) → Step 2
    - Failure (ErrUserNotFound) → return nil — the handler still responds 200 with the same
      message. An unknown address and a known one are indistinguishable to the caller.
    - Failure (other DB error) → wrapped "find user: %w" → 500

Step 2 — Issue the reset token (this module)
- Trigger: Step 1 found a user
- Pre-condition: none
- Call: generateResetToken() → 32 bytes from crypto/rand, hex-encoded; store only
      hashResetToken(raw) = SHA-256 hex, with expires_at = now() + 15m
      → UserRepository.CreatePasswordResetToken
  Confirmed: the raw token is never persisted — only its SHA-256. A DB read cannot recover a
  usable token. Multiple outstanding tokens for the same user are all valid simultaneously; there
  is no "invalidate previous tokens" step.
    - Success → Step 3
    - Failure (token generation) → wrapped "generate reset token: %w" → 500
    - Failure (insert) → wrapped "create reset token: %w" → 500

Step 3 — Send the email (this module, external dependency)
- Trigger: Step 2 committed
- Pre-condition: the token row exists — mail is sent AFTER the write, never before
- Call: Mailer.SendPasswordResetEmail(ctx, user.Email, rawToken)
  ⚠️ DEV ONLY: `cmd/main.go` wires `mailer.NewLogMailer(logger)`, which logs the reset link instead
  of sending it. The production provider is not chosen yet.
    - Success → 200 {message: "Nếu email tồn tại trong hệ thống, ..."}
    - Failure (any mail error) → logged via `WithError` at Error level, and the usecase STILL
      returns nil → the caller sees 200. Confirmed: this is intentional (no enumeration, and the
      token row is already written so a retry is possible), but it means a mail outage is invisible
      to the API consumer and to any client-side monitoring.
  Transaction boundary: Steps 2 and 3 are NOT in the same transaction — Step 3 runs after Step 2
  has committed. A mail failure cannot roll back the token row, which is what makes the silent-200
  behaviour coherent.

Step 4 — Consume the token and set the new password
- Trigger: POST /v1/auth/reset-password {token, new_password}
- Pre-condition: len(new_password) >= 8
- Call: hashResetToken(token) → UserRepository.ResetPassword → BEGIN
        SELECT id, user_id, expires_at, used_at FROM password_reset_tokens
          WHERE token_hash = $1 FOR UPDATE
        → UPDATE users SET password_hash = $1, updated_at = now()
        → UPDATE password_reset_tokens SET used_at = now()
        → UPDATE user_sessions SET is_revoked = true, is_active = false,
               revoked_at = now(), revoked_reason = 'password_reset'
             WHERE user_id = $1 AND is_revoked = false
    - Success → COMMIT → 200 {message: "Mật khẩu đã được cập nhật, vui lòng đăng nhập lại"}
    - Failure (no row for token_hash) → domain.ErrResetTokenInvalid → 400 RESET_TOKEN_INVALID
    - Failure (used_at already set, or now() past expires_at) → domain.ErrResetTokenInvalid → 400
    - Failure (password shorter than 8) → domain.ErrInvalidPassword → 400, before any DB write
    - Failure (any other error) → 500; tx rolls back
  Transaction boundary: Steps 4's four statements share one transaction, so "password changed"
  and "sessions revoked" can never diverge. The FOR UPDATE locks the token row, making a
  double-submit of the same token serialize rather than both succeed — the second one sees
  used_at set and gets 400.
```

## Step-by-step Flow — Avatar upload

```
Step 1 — Transport validation (this module)
- Trigger: POST /v1/users/me/avatar with raw image bytes as the body
- Pre-condition: authenticatedUserID(r) returned a user ID (else 401 UNAUTHORIZED, Step 0)
- Call: http.MaxBytesReader(w, r.Body, 5<<20) then io.ReadAll, then
      http.DetectContentType(data) checked against {image/jpeg, image/png, image/webp}
  Confirmed: the client's Content-Type header is never read. The type is sniffed from the bytes,
  so a mislabelled file cannot get through. There is no explicit dimension or pixel-count check —
  only the byte cap.
    - Success → Step 2
    - Failure (over 5 MB) → 400 FILE_TOO_LARGE
    - Failure (read error / empty body) → 400 INVALID_IMAGE, "unable to read uploaded image"
    - Failure (sniffed type not allowlisted) → 400 INVALID_IMAGE with details {detected_type}

Step 2 — Upload to object storage (this module → pkg/storage)
- Trigger: Step 1 passed
- Pre-condition: object name is built as fmt.Sprintf("avatars/%d", userID)
- Call: AvatarStorage.Upload → GCS PutObject with ACLPublicRead
  Confirmed: the object name is deterministic per user, so a re-upload OVERWRITES the previous
  avatar rather than accumulating versions. There is no CDN in front — the returned URL is a
  direct https://storage.googleapis.com/<bucket>/<object> public link, and no cache-busting
  parameter is added, so a client may serve a stale cached image after a replace.
    - Success → Step 3
    - Failure (upload error) → wrapped "upload avatar: %w" → 500; users.avatar_url is untouched

Step 3 — Persist the URL (this module)
- Trigger: Step 2 returned a URL
- Pre-condition: the object now exists in GCS
- Call: UserRepository.UpdateAvatarURL → UPDATE users SET avatar_url = $1, updated_at = now()
    - Success → 200 with the updated profile
    - Failure (ErrUserNotFound) → handler maps ANY error to 500 INTERNAL_ERROR
    - Failure (other DB error) → 500
  ⚠️ Not confirmed: if Step 3 fails after the GCS object was already written, no compensating
  delete is attempted — the object is left orphaned and users.avatar_url keeps pointing at the
  previous URL. Whether that is acceptable (cheap GCS storage, a later cleanup job) or needs a
  compensating action is a product decision — blocks tasks.md Phase 6 from being closeable.
  Transaction boundary: Steps 2 and 3 cannot share a transaction — one is object storage, the
  other is Postgres. This is the only write path in this feature with no atomicity guarantee.
```

## Deliberately stubbed for now

Three dev-only substitutions are wired in `services/appService/cmd/main.go` and are marked with
`⚠️ DEV ONLY` / `⚠️ DEV CONFIG` at the call site. None of them is hidden, and each is the exact
line to change before deploying:

| Stub | Current wiring | Production replacement |
|---|---|---|
| Secret resolution | `appConfig.MockSecretGetter` — reads plaintext from `dev_secrets` in `config.dev.yaml` | `pkg/secrets/gcp` (or `aws`) |
| Email delivery | `mailer.NewLogMailer(logger)` — logs the reset link | A real provider (SendGrid / SES / GCP equivalent). **Not yet chosen** — see the ⚠️ in Step 3 of the password-reset flow. |
| Storage credentials | `storage.GCSConfig{UseDefaultCredentials: true}` | A service account / workload identity |

These are covered by the tasks in Phase 6 below.

## Known gaps carried by this feature

- **The user-scoped routes are unreachable in practice, not unauthenticated.**
  `cmd/main.go` registers `/v1/users/me` and `/v1/users/me/avatar` with a bare `mux.Handle`, and
  never wraps them in `pkg/jwt/middleware.NewHTTPAuthMiddleware(...).Authenticate(...)`. The
  handlers *do* enforce auth — `authenticatedUserID(r)` reads `jwtcontext.GetUserID(r.Context())`
  and `writeUnauthorized` returns 401 when it is absent — but nothing ever writes that context
  value, because the middleware is the only thing that does. The practical result is that every
  request to these two routes returns 401 `UNAUTHORIZED`, rather than the handlers accepting
  unauthenticated traffic. Either way the feature is not usable end-to-end today. This is the
  single item that blocks US-004 from being called done; see `docs/ARCHITECTURE.md` §4.
- **`FindByEmail` does not select `avatar_url`.** The `users` SELECT in `FindByEmail` lists 8
  columns and omits `avatar_url`; `FindByID` selects it. Any future code that renders a profile
  from a login result will silently see an empty avatar. Not a bug today (login's response DTO
  does not include an avatar), but a trap.
- **No refresh endpoint exists, and `token_version` is never bumped.** `pkg/jwt.Service` exposes
  `RefreshAccessToken`, and every session row is written with `token_version = 1`
  (`insertSession`, shared by `Register` and `CreateSession`), but no route in `cmd/main.go`
  calls it. The original US-001 note ("rotate mỗi lần dùng") is therefore not what the shipped
  code does: a refresh token stays valid for its full configured lifetime and cannot be
  individually invalidated — only the bulk revoke in `ResetPassword` retires sessions. Treating
  rotation as shipped would be wrong.
- **`ErrUserNotFound` is not mapped to 404 on the profile routes.** `profile_handler.go` sends
  every usecase error to `WriteSystemError` → 500 `INTERNAL_ERROR`. A deleted user hitting
  `GET /v1/users/me` gets a 500, not a 404.
- **US-002 AC6 (login rate limiting) is not implemented at any layer.** The usecase carries an
  explicit comment that it belongs to middleware or the API gateway instead, and no such
  middleware exists in this repo yet. There is currently no brute-force protection on
  `POST /v1/auth/login`.
- **US-003 AC5 (forgot-password throttling) is similarly absent** — a caller can trigger unlimited
  reset emails (and, once the log mailer is replaced, unlimited mail to any address).
- **`users` has no `CREATE TABLE` in any migration.** `0001_register_onboarding.sql` opens by
  stating it *assumes* the table already exists with `id`, `email UNIQUE`, `password_hash`,
  `dharma_name`, `created_at`, `updated_at`. A database built from `migrations/` alone cannot be
  created from scratch — the baseline table is out-of-band.
