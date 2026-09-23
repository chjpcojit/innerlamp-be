---
type: Feature
title: Login
description: US-002's phone + password login, refresh-token rotation with reuse detection, and access-token logout — one session per user device, brute-force lockout per phone number.
resource: services/appService/
tags: [auth, identity, login, session, jwt]
status: stable
generated: { by: claude-code, at: 2026-09-22T16:25:00+07:00 }
sources:
  - path: docs/specs/features/login/requirements.md
    kind: spec
  - path: docs/specs/features/login/design.md
    kind: spec
  - path: docs/specs/features/login/tasks.md
    kind: spec
  - path: docs/specs/features/login/test-coverage-report.md
    kind: spec
  - path: docs/specs/features/api-response-envelope/requirements.md
    kind: spec
  - path: docs/specs/api/openapi.yaml
    kind: openapi
  - path: migrations/0002_login_devices_and_rate_limit.sql
    kind: migration
  - path: services/appService/domain/README.md
    kind: implementation-notes
  - path: services/appService/usecase/README.md
    kind: implementation-notes
---

# Summary

Three routes implement US-002 (đăng nhập) in the `identity` module. `POST /v1/auth/login` takes
`phone_number`, `password` and a `device` description, and returns an access/refresh token pair
plus the user. The response has the same shape as US-001's `verify-otp`. `POST /v1/auth/refresh`
exchanges a refresh token for a new pair and rotates the stored hash. `POST /v1/auth/logout`
revokes the session named by the access token's `sid` claim and returns `200` `{"success": true}`.
Failures use the shared `utils.ApiResponse` envelope:

- A wrong password and an unknown phone number both return the same `401` `INVALID_CREDENTIALS`
  with the same body.
- The 5th consecutive failure sets the lock. Every login for that phone after that, even with the
  correct password, returns `429` `LOGIN_LOCKED` with `details.locked_until` until the lock expires.
- Every refresh failure is `401` `REFRESH_TOKEN_INVALID`.
- Every logout auth failure is `401` `UNAUTHORIZED`.

A session is bound to a device. Logging in again from the same device revokes that device's
previous session (`superseded_by_login`). Sessions on other devices, and US-001's device-less
sessions, are left alone.

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. Each multi-write step owns one SQL transaction inside a single repository
method:

- `CreateLoginSession` upserts the device, revokes that device's old session, inserts the new
  session and clears `login_attempts`, all in one transaction.
- `RotateSession` locks the session row with `SELECT … FOR UPDATE`, re-checks the hash, then
  rotates it.

The usecase never drives a transaction itself. Token minting is handed to the repository as a
`domain.TokenIssuer` closure, because the session ID must exist before the access token (claim
`sid`) is minted inside the same transaction.

| Layer | Type | File |
|---|---|---|
| Domain / I/O + ports | inputs/results, entities, port | `services/appService/domain/login.go`: `DeviceInfo`, `LoginAttempt`, `Session`, `LoginInput`/`LoginResult` (alias of `RegisterResult`), `RefreshInput`/`RefreshResult`, `LogoutInput{UserID, SessionID}`, `LoginRepository` interface, `RevokeReason*` constants |
| Domain / Security helper | function | `services/appService/domain/session_token.go`: `HashRefreshToken` (SHA-256 hex). Shared by usecase (lookup) and repository (write) so the rule has one definition |
| Domain / Errors | sentinel errors | `services/appService/domain/errors.go`: `ErrInvalidCredentials`, `ErrLoginLocked` (+ typed `LoginLockedError{LockedUntil}`), `ErrInvalidDevice`, `ErrRefreshTokenInvalid`, `ErrUserNotFound` (internal only) |
| Usecase | UseCase | `services/appService/usecase/login_usecase.go`: `LoginUsecase.Login`/`Refresh`/`Logout`. Constant-time path for unknown phones (bcrypt against `dummyPasswordHash`); lockout check before the password check; reuse detection; access token carries `extra{"sid": sessionID}` |
| Repository | Repository | `services/appService/repository/postgres_login_repository.go`: all SQL for `users` (read), `login_attempts`, `user_devices`, `user_sessions`. `deviceFingerprint` = SHA-256 of `platform\|os_version\|device_model\|user_agent`, computed server-side |
| Delivery / Handler | Handler | `services/appService/delivery/http/login_handler.go`: 3 handlers + DTOs; `clientIP` from `RemoteAddr` (ignores `X-Forwarded-For`); `services/appService/delivery/http/errors.go`: `writeDomainError` mappings above, and `WriteAuthError` (always `401` `UNAUTHORIZED`, message `unauthorized`, never the reason) |
| Wiring | composition root | `services/appService/cmd/main.go`: builds `LoginRepository` → `LoginUsecase` (session TTL = `cfg.JWT.RefreshTokenExpiration`, policy = `cfg.Login`) → `LoginHandler`; registers 3 routes; wraps only logout in `jwtmiddleware.NewHTTPAuthMiddlewareWithOptions` with `RequireTokenType: "access"` and `ErrorHandler: deliveryhttp.WriteAuthError` |
| Migration | schema | `migrations/0002_login_devices_and_rate_limit.sql`: `user_devices`, `user_sessions.device_id`/`previous_refresh_token_hash`, partial unique index `uq_user_sessions_active_device`, `login_attempts` |
| Config | policy | `services/appService/config/config.dev.yaml` `login:` block: `max_failed_attempts: 5`, `lock_duration: 15m`, `failure_reset_window: 15m`; `jwt.refresh_token_expiration: 168h` (sliding session TTL) |

`pkg/` dependencies: `pkg/jwt` (mint/validate; `jti` makes every token unique, see
`docs/specs/features/jwt-token-uniqueness/`), `pkg/jwt/middleware` + `pkg/jwt/context` (logout
auth and claims), `pkg/database` (transactions), `pkg/validation` (`NormalizePhone`),
`pkg/utils` (`DecodeJSONBodyStrict`, `ApiResponse` writers), `pkg/logging`.

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `POST` | `/v1/auth/login` | public | `LoginHandler.Login` |
| `POST` | `/v1/auth/refresh` | public (refresh token in body) | `LoginHandler.Refresh` |
| `POST` | `/v1/auth/logout` | `pkg/jwt/middleware` bearer, access token only | `LoginHandler.Logout` |

Wire contract: `docs/specs/api/openapi.yaml` (v3.7). No per-endpoint `api/` concepts exist yet.
Logout's auth wrapping was confirmed by reading `cmd/main.go`, and a request without a token
returns `401` `UNAUTHORIZED` on the running binary.

# Request flow

```
POST /v1/auth/login                                   (public)
  └─ LoginHandler.Login                               (delivery/http)
       ├─ DecodeJSONBodyStrict (unknown field → 400 VALIDATION_ERROR)
       └─ LoginUsecase.Login                          (usecase)
            ├─ NormalizePhone / password non-empty / device.platform+device_model
            ├─ FindLoginAttempt → locked_until > now  → LoginLockedError (429)
            ├─ FindUserByPhone (not found → dummy hash, same bcrypt cost)
            ├─ mismatch → RecordLoginFailure → 401 INVALID_CREDENTIALS
            │    (one upsert: counter restarts at 1 if the lock has expired or the last failure
            │     is older than failure_reset_window; locked_until = now + lock_duration at 5)
            └─ CreateLoginSession                     (repository, 1 tx)
                 ├─ upsert user_devices (row lock serializes same-device logins)
                 ├─ revoke this device's active session  (superseded_by_login)
                 ├─ sessionID = uuid → TokenIssuer(userID, sessionID)
                 ├─ insert user_sessions (refresh_token_hash, expires_at = now + TTL)
                 └─ delete login_attempts for the phone

POST /v1/auth/refresh                                 (public)
  └─ LoginUsecase.Refresh
       ├─ ValidateToken + TokenType == refresh
       ├─ FindSessionByRefreshHash(hash)  matches refresh_token_hash OR previous_refresh_token_hash
       │    ├─ no row / revoked / inactive / expired → 401
       │    └─ matched previous → RevokeSession(refresh_token_reuse) → 401
       ├─ FindUserByID  (error → 401)
       └─ RotateSession                               (repository, 1 tx, FOR UPDATE)
            ├─ re-check: hash still current, not revoked, active, unexpired, else 401
            └─ previous_hash = hash; hash = new; token_version++; expires_at = now + TTL (sliding)

POST /v1/auth/logout                                  (access-token middleware)
  └─ LoginHandler.Logout
       ├─ claims.UserID numeric, claims.Extra["sid"] non-empty string, else 401 UNAUTHORIZED
       └─ LoginUsecase.Logout → RevokeSession(user_id, session_id, 'logout')
            └─ 0 or 1 row changed → 200 {"success": true} (idempotent)
```

# Validation rules

- Body shape: `DecodeJSONBodyStrict` in the handler (`writeDecodeError` in `errors.go`) rejects:
  - unknown fields and malformed JSON: `400` `VALIDATION_ERROR`, with `details.field` when the
    field is known
  - bodies over the size cap: `413` `PAYLOAD_TOO_LARGE`

  The login/refresh DTOs have no `validate:` struct tags. Every rule below is in the usecase.
- `phone_number`: `validation.NormalizePhone`. Empty after normalizing gives `ErrInvalidPhone`
  (`400` `INVALID_PHONE`). A format error does **not** increment the brute-force counter.
- `password`: must be non-empty (`ErrInvalidPassword`, `400`). There is no length rule at login,
  so the only possible answer is match or no match.
- `device.platform`, `device.device_model`: required (`ErrInvalidDevice`, `400` `INVALID_DEVICE`).
  `os_version`, `app_version` and `device_name` are optional. `ip_address` and `user_agent` come
  from the request, never from the body.
- `refresh_token`: empty, invalid signature, expired, or not `type: refresh` all return
  `ErrRefreshTokenInvalid`.

# Verification

- 2026-09-21, binary built from `./services/appService/cmd` against Postgres 16 (Docker
  `innerlamp-pg`), exercised with curl. Observed:
  - login `200`
  - refresh `200`
  - reusing the old refresh token → `401`, and the rotated token also dies (session revoked
    `refresh_token_reuse`, `token_version = 2`)
  - re-login on the same device revokes the older session (`superseded_by_login`); the
    device-less session created by `verify-otp` is untouched
  - 5 wrong passwords → `401`×5, then the correct password → `429` `LOGIN_LOCKED` with
    `locked_until`
  - logout `200` `{"success":true}`, twice (idempotent)
  - refresh after logout → `401`
  - logout without a token → `401` `UNAUTHORIZED`
- `go test -count=1 ./services/appService/repository/` was run against the same Postgres (not
  skipped). All `TestLoginRepository_*` tests pass. Usecase and handler unit tests pass.
- Automated black-box E2E test: `services/appService/e2e/login_e2e_test.go` `TestLoginFlow_E2E`.
  It builds the real binary, runs it against a testcontainers Postgres, and covers AC1–AC15 in 10
  subtests. Skipped under `-short`; needs Docker. AC-by-AC table:
  `docs/specs/features/login/test-coverage-report.md`.
- Temporarily removing the auth middleware from logout in `cmd/main.go` makes AC12/AC13 fail. The
  rejection cases (AC15) alone would not notice, because the handler also answers 401 when there
  are no claims.

# Pending work

- There is no cleanup job for `login_attempts`, or for revoked/expired `user_sessions` rows.
  Accepted for MVP on 2026-09-21 (see Deferred in `login/tasks.md`). The rows are already inert
  through `expires_at`/`is_revoked`.
- `services/appService/delivery/http/`, `services/appService/repository/` and
  `services/appService/cmd/` have no `README.md`, although `CLAUDE.md`'s per-folder convention
  expects one.
- Cross-feature dev-only wiring (`appConfig.NewMockSecretGetterFromYAML`, `⚠️ DEV ONLY`) also covers
  the JWT signing secret this feature depends on.

# Known gaps vs. design

These are accepted trade-offs recorded in `docs/specs/features/login/design.md` "Known gaps", not
divergences:

- An access token stays valid until it expires (≤ 15 min) after logout or reuse-revoke. The
  middleware is stateless and does not consult `user_sessions`.
- Logout needs an unexpired access token. If it has expired, the client must refresh first.
- Two physical devices with identical `platform|os_version|device_model|user_agent` share one
  fingerprint, so a login on one supersedes the other's session.
- There is no per-IP rate limit on `login`/`refresh`. Lockout is per phone number only.

# Related concepts

- [Register](register.md): US-001. Its `verify-otp` creates the first (device-less) session and
  mints the access token with the same `sid` claim that logout relies on. It shares `users`,
  `user_sessions`, `domain.TokenIssuer` and `delivery/http/errors.go`.
- [packages/middleware.md](../packages/middleware.md) — the `Recovery`/`CORS`/`Logger` chain that
  wraps all three routes (via `newHandler` in `cmd/main.go`), outside the per-route
  `pkg/jwt/middleware` auth wrapper; it provides the `404`/`405`/panic envelope and the CORS headers
  every response now carries.
- No `Package` concepts exist yet for `pkg/jwt` (and its `middleware`/`context` subpackages),
  `pkg/database`, `pkg/validation` or `pkg/utils`.
- No `Migration` concept exists yet for `migrations/0002_login_devices_and_rate_limit.sql`.
- No `Domain Entity` concept exists yet for `Session` or `LoginAttempt`. They are documented only
  in this feature's Architecture table, because `LoginUsecase` is their only consumer.
