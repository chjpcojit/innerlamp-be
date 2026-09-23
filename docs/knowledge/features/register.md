---
type: Feature
title: Register
description: US-001's two-step, OTP-over-SMS registration flow — request-otp, resend-otp, verify-otp — that never writes a users row until the phone number is verified.
resource: services/appService/
tags: [auth, onboarding, otp, identity]
status: draft
generated: { by: claude-code, at: 2026-09-22T16:20:00+07:00 }
sources:
  - path: docs/specs/features/onboarding/requirements.md
    kind: spec
  - path: docs/specs/features/onboarding/design.md
    kind: spec
  - path: docs/specs/features/onboarding/tasks.md
    kind: spec
  - path: services/appService/domain/README.md
    kind: implementation-notes
  - path: services/appService/usecase/README.md
    kind: implementation-notes
  - path: docs/specs/features/api-response-envelope/requirements.md
    kind: spec
  - path: docs/specs/features/json-body-hardening/requirements.md
    kind: spec
  - path: docs/specs/api/openapi.yaml
    kind: openapi
  - path: migrations/0001_register_onboarding.sql
    kind: migration
  - path: migrations/0003_otp_send_limits.sql
    kind: migration
---

# Summary

Three public routes implement US-001 (đăng ký tài khoản): `POST /v1/auth/register/request-otp`
takes email/password/phone/profile, checks the unique keys, stores the payload in
`pending_registration`, and SMS's a 6-digit code (`200`); `POST /v1/auth/resend-otp` reissues a code
for an in-flight attempt (shared endpoint across OTP purposes, only `register` implemented);
`POST /v1/auth/register/verify-otp` checks the code and, only on match, inserts the `users` row and
a `user_sessions` row in the same transaction — the account is created already logged in, so the
response (`201`) carries access/refresh tokens and the user. No `users` row exists in an unverified
state at any point in the flow.

**Sending is rate-limited per phone number** (onboarding v5.0/v5.1). One attempt gets four sends:
the first is free, the following three are counted, and the fourth send — the third counted — is
still delivered and locks sending for 30 minutes (`otp_send_limits.locked_until`). `request-otp`
and `resend-otp` spend from the **same** counter; hitting the cap answers `429` `OTP_SEND_LIMITED`
with `locked_until`, and once the lock lifts the cycle resets. That send lock is deliberately separate from the wrong-guess lock
(`otp_codes.locked_until`, also 30 minutes): the send lock must **not** block `verify-otp`, or the
last code a user was sent would be unusable during its own 90-second window.

**Restarting an attempt** (onboarding v5.1). `pending_registration` now outlives the OTP by a lot —
its TTL is 24h against the code's 90s — so a caller can resume after the 30-minute send lock. A
pending row for the **same** phone is not a conflict: `request-otp` overwrites it in place
(`expires_at = now + 24h`), which is how a reinstalled app starts over. `resend-otp` instead leaves
the payload untouched and simply issues a fresh code — including when the previous code has already
expired, which is why "press resend after the lock lifts" works. The email is the exception: a live
pending row claiming a given email for a **different** phone still answers `409` `EMAIL_TAKEN`,
because email is the account's globally-unique key.

**Current limitation**: the OTP value itself is a hardcoded placeholder
(`usecase/register_usecase.go`'s `generateOTP()` always returns `"111111"`), and SMS delivery goes
through `pkg/sms/logsms` (logs the message instead of sending it) wired in `cmd/main.go` — both
marked `⚠️ DEV ONLY`/`⚠️ DEV CONFIG` in the source. The flow has been run end-to-end against a real
Postgres (see Verification), but only through these placeholders, not their production equivalents.

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. `repository` imports only `pkg/database` and `domain`, and is handed token
minting as a `domain.TokenIssuer` closure. `domain` imports nothing from the other layers. The
usecase layer owns every transaction boundary that must not be observed half-done —
`SavePendingAndIssueOTP`, `ReissueOTP`, and `CreateUserAndSession` are each a single repository
method wrapping one SQL transaction (including the `otp_send_limits` accounting), not a sequence the
usecase drives itself.

| Layer | Type | File |
|---|---|---|
| Domain / Entities | Domain Entity | `services/appService/domain/user.go` — `User`, maps `users`; `services/appService/domain/pending_registration.go` — `PendingRegistration`, maps `pending_registration`; `services/appService/domain/otp.go` — `OTPCode`/`OTPPurpose`, maps `otp_codes`, shared by every OTP flow; `services/appService/domain/otp_send_limit.go` — `OTPSendLimit` (maps `otp_send_limits`) + the `SendLimits`/`SendState` value types the usecase threads down |
| Domain / Ports + I/O | inputs/results, ports | `services/appService/domain/register.go` — `RequestRegisterOTPInput`/`ResendOTPInput`/`VerifyRegisterOTPInput`, `RegisterResult` (also reused as login's `LoginResult`), `RegistrationConflicts`, `RegistrationRepository` and `OTPSender` interfaces, `TokenIssuer func(userID, sessionID string)` callback type |
| Domain / Errors | sentinel errors | `services/appService/domain/errors.go` — `ErrInvalidPhone`/`ErrInvalidEmail`/`ErrInvalidPassword`, `ErrUnsupportedOTPPurpose`, `ErrPhoneTaken`, `ErrEmailTaken`, `ErrOTPInvalid`/`ErrOTPExpired`/`ErrOTPLocked` (+ typed `OTPLockedError`), `ErrOTPResendTooSoon` (+ typed `OTPResendTooSoonError`), `ErrOTPSendLimited` (+ typed `OTPSendLimitedError`), `ErrPendingRegistrationNotFound`/`Expired`, `ErrSMSSendFailed`. v5.0 removed `ErrOTPResendExhausted` (the permanent block it represented is gone). |
| Usecase | UseCase | `services/appService/usecase/register_usecase.go` — `RegisterUsecase.RequestOTP`/`ResendOTP`/`VerifyOTP`; guarantees no `users` row before OTP verification, applies the AC5 send policy through the repository, and enforces AC12 lockout; `tokenIssuer` mints the access token with `extra{"sid": sessionID}` so [login](login.md)'s logout can revoke this session |
| Repository | Repository | `services/appService/repository/postgres_registration_repository.go` — every SQL statement for the five tables, `applySendLimits` (the `otp_send_limits` state machine, run inside `SavePendingAndIssueOTP`/`ReissueOTP`'s transaction), `ReleaseSendLimit`, and unique-violation translation by constraint name; `services/appService/repository/sms_otp_sender.go` — `domain.OTPSender` over `pkg/sms.Sender` |
| Delivery / Handler | Handler | `services/appService/delivery/http/register_handler.go` — 3 routes + DTOs; `verify-otp` answers `201` (`WriteSuccessResponseWithStatus`), the other two `200`; `services/appService/delivery/http/errors.go` — `writeDomainError` sentinel/typed→status mapping and `writeDecodeError` (shared with login) |
| Wiring | composition root | `services/appService/cmd/main.go` — constructs `RegistrationRepository`/`OTPSender`/`RegisterUsecase`/`RegisterHandler`, registers the 3 routes on `http.ServeMux` (public, no per-route auth middleware), and since the `http-middleware-wiring` change serves the mux through `newHandler(mux)` = `httpmiddleware.Recovery(httpmiddleware.CORS(httpmiddleware.Logger(deliveryhttp.WithAPIFallback(mux))))`, so 404/405/panics use the envelope and every response also carries CORS headers; touches the `⚠️ DEV ONLY` (`appConfig.NewMockSecretGetterFromYAML`) and `⚠️ DEV CONFIG` (`pkg/sms/logsms`) blocks. See [packages/middleware.md](../packages/middleware.md) for the chain and why its order is load-bearing |

`pkg/` dependencies: `pkg/database` (Postgres connection + `Begin`/transactions), `pkg/jwt`
(access/refresh token minting via the `TokenIssuer` closure), `pkg/logging`, `pkg/validation`
(`phone`/`email` struct tags), `pkg/sms` (provider-agnostic send, `logsms` driver active in dev),
`pkg/utils` (`DecodeJSONBodyStrict`, `WriteSuccessResponse`/`WriteBusinessError`/`WriteSystemError`
envelope `{success, data}` / `{success, error{code,message,type}, details}`). None of these `pkg/`
packages have an OKF `Package` concept in this bundle yet — see Related concepts.

Policy comes from `config.dev.yaml`'s `otp:` block (`config.OTPConfig`, passed to
`NewRegisterUsecase`):

| Setting | Value | Meaning |
|---|---|---|
| `ttl` | 90s | how long an issued code can be verified |
| `send_cooldown` | 60s | floor between two sends, for both `request-otp` and `resend-otp` |
| `max_send_attempts` | 3 | counted sends per cycle (the first send is free, so 4 sends total) |
| `send_lock_duration` | 30m | how long sending is locked at the cap — and how long a quiet cycle may sit before it resets |
| `max_attempts` | 3 | wrong codes before the wrong-guess lock |
| `lock_duration` | 30m | how long that wrong-guess lock holds |
| `pending_registration_ttl` | 24h | how long the captured payload survives an abandoned attempt |

The session TTL is `jwt.refresh_token_expiration` (168h).

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `POST` | `/v1/auth/register/request-otp` | public | `RegisterHandler.RequestOTP` |
| `POST` | `/v1/auth/resend-otp` | public | `RegisterHandler.ResendOTP` |
| `POST` | `/v1/auth/register/verify-otp` | public | `RegisterHandler.VerifyOTP` |

All three are correctly public in `cmd/main.go` — the caller has no account yet, so there is nothing
to authenticate; this is by design, not a missing auth-middleware gap. (The generic
`Recovery`/`CORS`/`Logger` chain from [packages/middleware.md](../packages/middleware.md) does wrap
them, but it does not authenticate.) No `api/` concepts exist yet for these three routes in this
bundle — see Related concepts.

# Request flow

```
POST /v1/auth/register/request-otp
  └─ public
       └─ RegisterHandler.RequestOTP                 (delivery/http)
            ├─ decode + validate (email/phone/password required, `phone` tag)
            └─ RegisterUsecase.RequestOTP(ctx, input) (usecase)
                 ├─ NormalizeEmail/IsValidEmail, NormalizePhone, len(password) >= 8 (AC7/AC8)
                 ├─ RegistrationRepository.FindConflicts             (repository, AC6)
                 │    phone in users → 409 PHONE_TAKEN; email in users, or held by another
                 │    phone's unexpired pending_registration → 409 EMAIL_TAKEN
                 ├─ RegistrationRepository.FindActiveOTP
                 │    none → proceed; locked (wrong guesses) → 429 OTP_LOCKED + locked_until
                 ├─ bcrypt-hash password and OTP
                 ├─ RegistrationRepository.SavePendingAndIssueOTP    (repository, 1 tx)
                 │    (a) INSERT otp_send_limits … ON CONFLICT DO NOTHING + SELECT … FOR UPDATE
                 │    (b) apply the send policy → typed error + ROLLBACK (nothing written):
                 │        cycle reset/free first send, 429 OTP_SEND_LIMITED, or
                 │        429 OTP_RESEND_TOO_SOON + retry_after_seconds
                 │    (c) UPSERT pending_registration (overwrites; expires_at = now + 24h)
                 │    (d) mark any active otp_codes row used
                 │    (e) INSERT otp_codes (expires_at = now + 90s)
                 └─ OTPSender.SendOTP  (failure → 500 SMS_SEND_FAILED; pending + OTP stay)

POST /v1/auth/resend-otp
  └─ public
       └─ RegisterHandler.ResendOTP                   (delivery/http)
            ├─ decode + validate (phone required, purpose oneof 4 values)
            └─ RegisterUsecase.ResendOTP(ctx, input)   (usecase)
                 ├─ reject if purpose != register (400 UNSUPPORTED_OTP_PURPOSE)
                 ├─ NormalizePhone
                 ├─ RegistrationRepository.FindPendingRegistration
                 │    none → 400 OTP_INVALID; older than 24h → 400 PENDING_REGISTRATION_EXPIRED
                 ├─ RegistrationRepository.FindActiveOTP
                 │    none/expired/used → NOT an error (v5.1): proceed and issue a new code;
                 │    present and locked → 429 OTP_LOCKED + locked_until
                 ├─ RegistrationRepository.ReissueOTP                (repository, 1 tx)
                 │    (a) INSERT … ON CONFLICT DO NOTHING + SELECT … FOR UPDATE
                 │    (b) apply the send policy → 429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED
                 │    (c) mark active otp_codes row used
                 │    (d) INSERT otp_codes (expires_at = now + 90s)
                 │    pending_registration is NOT touched (v5.1)
                 └─ OTPSender.SendOTP  → 200 {…, resend_remaining}

POST /v1/auth/register/verify-otp
  └─ public
       └─ RegisterHandler.VerifyOTP                   (delivery/http)
            ├─ decode + validate (phone required, otp_code len=6 numeric)
            └─ RegisterUsecase.VerifyOTP(ctx, input)   (usecase)
                 ├─ NormalizePhone
                 ├─ RegistrationRepository.FindPendingRegistration
                 │    none → 400 OTP_INVALID; expired → 400 PENDING_REGISTRATION_EXPIRED
                 ├─ RegistrationRepository.FindActiveOTP   (none → 400 OTP_EXPIRED, AC4)
                 ├─ locked → 429 OTP_LOCKED + locked_until
                 ├─ bcrypt compare; mismatch → IncrementOTPAttempts → 400 OTP_INVALID;
                 │    attempts >= 3 → LockOTP (locked_until = now + 30m, used_at = now) (AC12)
                 └─ RegistrationRepository.CreateUserAndSession       (repository, 1 tx:
                       re-check users.email, insert users (email_verified_at NULL),
                       delete pending_registration, consume otp_codes,
                       delete otp_send_limits for the phone (ReleaseSendLimit),
                       sessionID = uuid → issueTokens(userID, sessionID) → pkg/jwt,
                       insert user_sessions (no device_id))
```

The `verify-otp` send-limit interaction is the point of splitting the two locks: while a number is
`OTP_SEND_LIMITED`, this route still runs normally and the last code can be verified.

# Validation rules

- Body shape (all three routes) — `utils.DecodeJSONBodyStrict` via `writeDecodeError`: unknown
  field or malformed JSON → `400` `VALIDATION_ERROR`; oversized body → `413` `PAYLOAD_TOO_LARGE`.
  Struct-tag failures below → `400` `VALIDATION_ERROR` with one `field: tag` pair per field.
- `email` — delivery struct tag `required,email` on `requestOTPRequest.Email`
  (`register_handler.go`); business-normalized and re-checked by
  `validation.IsValidEmail(NormalizeEmail(...))` in `RegisterUsecase.RequestOTP`, producing
  `domain.ErrInvalidEmail`.
- `phone_number` — delivery tag `required,phone` on all three request structs (the `phone`
  validator tag registered once in `cmd/main.go`); business-normalized by `validation.NormalizePhone`,
  empty-after-normalize produces `domain.ErrInvalidPhone`.
- `password` — delivery tag `required` only, no `min` on purpose (see the comment on
  `requestOTPRequest`); the usecase enforces the AC7 floor, `len(input.Password) <
  minPasswordLength (8 bytes)` → `domain.ErrInvalidPassword`.
- `otp_code` — delivery tag `required,len=6,numeric` on `verifyOTPRequest.OTPCode`. A wrong shape
  is rejected by the validator (`400` `VALIDATION_ERROR`, `details.otp_code`) before the usecase
  runs, and does **not** count as a failed attempt. A well-formed wrong code is compared against the
  bcrypt hash and surfaces as `domain.ErrOTPInvalid` (`400` `OTP_INVALID`), incrementing
  `otp_codes.attempts`.
- `purpose` (resend only) — delivery tag `required,oneof=register login reset_password
  verify_email`; the usecase further rejects everything but `register` with
  `domain.ErrUnsupportedOTPPurpose` (`400` `UNSUPPORTED_OTP_PURPOSE`) since only US-001 is
  implemented.
- Uniqueness (AC6, v5.1) — not a struct tag. `RegistrationRepository.FindConflicts` runs three
  `EXISTS` checks in one query: `users.phone_number` → `ErrPhoneTaken`; `users.email` →
  `ErrEmailTaken`; and `pending_registration.email` **for a different phone** with `expires_at >
  now()` → `ErrEmailTaken`. A pending row for the same phone is not a conflict — it is the caller's
  own attempt and `request-otp` overwrites it. The phone is checked first, so when both collide the
  caller gets `PHONE_TAKEN`. `CreateUserAndSession` re-checks `users.email` inside its own
  transaction and translates a unique violation on `users_email_key` / `users_phone_number_key` into
  `ErrEmailTaken` / `ErrPhoneTaken` as the race-loser path.
- Send policy (AC5) — enforced inside the repository transaction, not by a struct tag or a usecase
  check: the first send of a cycle is free, the next three are counted, and the fourth send (the
  third counted) is still delivered and sets `locked_until`. A refusal leaves no
  `pending_registration`, no new `otp_codes` row, and does not spend a slot.
- Messages: `ErrPhoneTaken` = "phone number already in use". Codes and statuses for every sentinel
  and typed error are in `delivery/http/errors.go` `writeDomainError` and appear in the Request flow
  above.

# Verification

- 2026-09-22: `go test -count=1 ./services/appService/repository/` against Postgres 16 (Docker
  `innerlamp-pg`, not skipped). Includes the v5.0 send-limit tests
  (`TestRegistrationRepository_SendLimits`: free first send, cooldown refusal does not spend a slot,
  cap sets a 30m lock, aged-out cycle resets, `ReleaseSendLimit` / completion clears the counter) and
  the v5.1 conflict tests (`TestRegistrationRepository_FindConflicts`: own pending is not a conflict,
  another phone's live pending is, expired rows are not, expired pending does not hide an account).
- 2026-09-22: `go test -count=1 ./services/appService/e2e/` (real `cmd/main.go` binary + a
  testcontainers Postgres) passes:
  - `TestRegisterFlow_SendLimitAndVerifyDuringLock_E2E` — one request + three resends, the 5th send
    answers `429` `OTP_SEND_LIMITED` with `locked_until`, and `verify-otp` still answers `201` while
    send-locked (the split-lock invariant).
  - `TestRegisterFlow_RestartOverwritesPending_E2E` — the same phone retries and gets
    `429 OTP_RESEND_TOO_SOON` (not `PHONE_TAKEN`), then after the cooldown restarts and the new
    payload replaces the old one; another phone's live pending still answers `409 EMAIL_TAKEN`.
  - `TestRegisterFlow_ResendAfterCodeExpired_E2E` — after the code's TTL has passed but with the
    payload alive, `resend-otp` issues a new code (`200`) instead of `OTP_EXPIRED`, and `verify-otp`
    then answers `201`.
- 2026-09-21: the real binary was driven with curl (request-otp `200` → verify-otp with `111111` →
  tokens) as setup for login; `services/appService/e2e/login_e2e_test.go` also calls `request-otp`
  and `verify-otp` (asserting `201`) for every login scenario.
- This is **not** a full AC audit of US-001. The register-specific ACs are covered by unit,
  repository and the three E2E tests above, but the `golang-e2e-verification` audit that maps each
  AC to a test has not been run for this feature.

# Pending work

- `usecase/register_usecase.go`'s `generateOTP()` returns the constant `"111111"` — not a random
  6-digit code. Isolated in one function per its own doc comment, but must be replaced before any
  real user sees this flow. The E2E harness relies on it (`verifyOTP` in `e2e/login_e2e_test.go`),
  so replacing it also needs a way for tests to read the code.
- `cmd/main.go` wires `pkg/sms/logsms` (`⚠️ DEV CONFIG`) — OTPs are logged, not actually sent over
  SMS. Swap `cfg.SMS.Provider` and the blank import once a real provider subpackage exists.
- `cmd/main.go` wires `appConfig.NewMockSecretGetterFromYAML` (`⚠️ DEV ONLY`) reading plaintext
  `dev_secrets:` from YAML instead of `pkg/secrets/gcp`/`aws` — affects every feature in this
  service, not just register.
- No cleanup job for `pending_registration`/`otp_codes`/`otp_send_limits` rows — accepted for MVP
  (Deferred in `docs/specs/features/onboarding/tasks.md`). Since v5.1 a stale payload holds its
  bcrypt `password_hash` for up to 24h, so the job now matters more than it did at 90s. Until it
  runs: `request-otp` overwrites the same phone's row, `FindConflicts` ignores expired rows, and an
  email claimed by another phone's live pending is blocked for up to 24h. `DeletePendingRegistration`
  exists but has no production caller.
- No AC audit via `golang-e2e-verification` for US-001 (see Verification). The harness in
  `services/appService/e2e/` already exists and can be reused.
- `services/appService/repository/`, `delivery/http/`, and `cmd/` have no `README.md`, though
  `CLAUDE.md`'s per-folder convention calls for one on any folder handling non-trivial logic.
- No `api/` concepts exist yet for the three routes (see API surface) — write
  `api/auth-register-request-otp.md`, `api/auth-resend-otp.md`, `api/auth-register-verify-otp.md` if
  per-endpoint documentation is wanted. No `migrations/0003-otp-send-limits.md` concept either.
- Promote to `stable` once (a) the OTP/SMS placeholders above are replaced with production
  equivalents, and (b) the US-001 ACs are audited end-to-end (`golang-e2e-verification`).

# Known gaps vs. design

- None found while reading the code against `docs/specs/features/onboarding/design.md` (v5.2). The
  two divergences this bundle previously tracked were fixed upstream: the expired-pending block
  (onboarding v4.5, Phase 8) and the permanent `resend_count` cap (onboarding v5.0/v5.1,
  Phases 9–10).

# Related concepts

- [Login](login.md) — US-002. Shares `users`, `user_sessions`, `RegisterResult`,
  `domain.TokenIssuer`, and `delivery/http/errors.go`. Logout depends on the `sid` claim this
  feature's `verify-otp` token carries.
- [packages/middleware.md](../packages/middleware.md) — the `Recovery`/`CORS`/`Logger` chain that
  wraps all three routes (via `newHandler` in `cmd/main.go`); it also writes the `404`/`405`/panic
  envelope this feature relies on.
- No `Package` concepts exist yet for `pkg/database`, `pkg/jwt`, `pkg/sms`, `pkg/validation`,
  `pkg/logging`, or `pkg/utils`, all of which this feature depends on (see Architecture).
- No `Migration` concept exists yet for `migrations/0001_register_onboarding.sql` (the five tables'
  baseline) or `migrations/0003_otp_send_limits.sql` (the send-limit table).
- No `Domain Entity` or `UseCase` concepts exist yet for `User`, `PendingRegistration`, `OTPCode`,
  `OTPSendLimit`, or `RegisterUsecase` as standalone files — each is documented only inside this
  feature's Architecture table, per `concept_types.md`'s "Feature vs UseCase" rule.
