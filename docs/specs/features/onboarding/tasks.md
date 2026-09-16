# Tasks — onboarding

Version: 1.2
Last updated: 2026-09-16
Status: implemented — all four user stories shipped and tested; Phases 1-5 complete, Phase 6 open
Module: `identity` (spec folder `onboarding`)

> See `changelog.md` in this folder for the version history of this spec.

<!--
Every item below traces back to a row in design.md's "Layers & components" table or a step in its
"Step-by-step Flow" sections. Checked items were verified by reading the actual file, not inferred
from a README or from this spec's own earlier wording.
-->

## Phase 1 — Domain

- [x] `User`, `Device`, `Session` entities in `services/appService/domain/user.go` (business fields only, no JSON tags)
- [x] Input/output structs in `domain/register.go`, `domain/login.go`, `domain/password_reset.go`, `domain/profile.go`
- [x] `TokenIssuer` func type — how the usecase hands JWT minting to the repository without importing `pkg/jwt` from `domain/`
- [x] `UserRepository` interface (9 methods) declared in `domain/register.go`, implemented in `repository/`
- [x] Sentinel errors in `domain/errors.go`: `ErrEmailTaken`, `ErrPhoneTaken`, `ErrInvalidEmail`, `ErrInvalidPhone`, `ErrInvalidPassword`, `ErrUserNotFound`, `ErrInvalidCredentials`, `ErrResetTokenInvalid`
- [x] `domain/mock_user_repository.go` for usecase tests

## Phase 2 — Usecase

- [x] `usecase/register_usecase.go` — `Register`: normalize → validate email → E.164 regex → password length → bcrypt → delegate to repository
- [x] `deviceFingerprint()` — SHA-256 over `Platform|OSVersion|DeviceModel|UserAgent`
- [x] `usecase/login_usecase.go` — `Login`: normalize → `FindByEmail` → bcrypt compare → `CreateSession`; both failure paths collapse to `ErrInvalidCredentials`
- [x] `usecase/password_reset_usecase.go` — `ForgotPassword` (no-enumeration, 15m TTL, SHA-256 token hash) and `Reset` (validate → hash → delegate)
- [x] `usecase/profile_usecase.go` — `GetProfile` / `UpdateProfile` pass-through
- [x] `usecase/avatar_usecase.go` — `AvatarStorage` port declared consumer-side; deterministic object name `avatars/{userID}`
- [x] Unit tests, table-driven with `testify`: `register_usecase_test.go`, `login_usecase_test.go`, `password_reset_usecase_test.go`, `profile_usecase_test.go`, `avatar_usecase_test.go` — mock repository, no real DB
- [ ] Map `ErrUserNotFound` to a distinguishable error at the `GetProfile`/`UpdateProfile` boundary
      <!-- design.md "Known gaps": the profile handler currently turns it into a 500. Not a
      regression — the route is unreachable until Phase 6 item 1 lands. -->

## Phase 3 — Repository

- [x] `migrations/0001_register_onboarding.sql` — `users.phone_number` (UNIQUE NOT NULL), `users.ordination_place`, `user_devices`, `user_sessions`
- [x] `migrations/0002_login_device_unique.sql` — `UNIQUE (user_id, device_id)` on `user_devices`, the upsert conflict target
- [x] `migrations/0003_password_reset.sql` — `password_reset_tokens` + unique index on `token_hash`
- [x] `migrations/0004_profile_avatar.sql` — `users.avatar_url`
- [x] `repository/postgres_user_repository.go` implementing all 9 `UserRepository` methods
- [x] `Register` and `CreateSession` each wrap their writes in one transaction; shared `insertSession` helper mints the JWT via the `TokenIssuer` closure inside the tx
- [x] `isUniqueViolation()` translates `pq` code `23505` + constraint name into `ErrEmailTaken` / `ErrPhoneTaken`
- [x] `ResetPassword` uses `SELECT ... FOR UPDATE` and revokes all active sessions (`revoked_reason = 'password_reset'`) in the same tx
- [x] `nullableString()` — empty string persisted as SQL NULL for `dharma_name` / `ordination_place`
- [x] `repository/postgres_user_repository_test.go`
- [x] `repository/gcs_avatar_storage.go` implementing `usecase.AvatarStorage` (public-read ACL)
- [x] `repository/gcs_avatar_storage_test.go`
- [ ] Add `CREATE TABLE users` (or a documented baseline migration) so a database can be built from `migrations/` alone
      <!-- design.md "Known gaps": 0001 assumes the table pre-exists out-of-band. -->

## Phase 4 — Delivery

- [x] Request/response DTOs in each `delivery/http/*_handler.go` — JSON tags live here only, never in `domain/`
- [x] `register_handler.go` — 201, fills `IPAddress`/`UserAgent` server-side, maps 3 validation sentinels + 2 conflict sentinels to `400`/`409`
- [x] `login_handler.go` — 200, reuses `registerResponse` so the two response shapes cannot drift
- [x] `forgot_password_handler.go` — fixed non-enumerating message; any usecase error → 500
- [x] `reset_password_handler.go` — maps `ErrResetTokenInvalid` (400) and `ErrInvalidPassword` (400)
- [x] `profile_handler.go` — GET/PATCH method switch, `authenticatedUserID` guard → 401, `profileResponse` without `omitempty`
- [x] `avatar_handler.go` — 5 MB `MaxBytesReader`, `DetectContentType` allowlist (JPEG/PNG/WebP), `FILE_TOO_LARGE` / `INVALID_IMAGE` codes
- [x] `auth_context.go` — shared `authenticatedUserID(r)` helper
- [x] All six routes registered in `services/appService/cmd/main.go`
- [x] Handler tests: `register_handler_test.go`, `login_handler_test.go`, `forgot_password_handler_test.go`, `reset_password_handler_test.go`, `profile_handler_test.go`, `avatar_handler_test.go`
- [x] `docs/specs/api/openapi.yaml` covers all six routes
- [ ] Re-verify `docs/specs/api/openapi.yaml` against the handlers after any further handler edit (it is hand-maintained and must not drift)

## Phase 5 — Integration & verification

- [x] `go build ./...` clean
- [x] `go vet ./...` clean
- [x] `go test -short ./...` clean — usecase, repository and handler suites all run without real AWS/GCP credentials

## Phase 6 — Open, blocks calling US-004 done

- [ ] Wire `pkg/jwt/middleware.NewHTTPAuthMiddleware(...).Authenticate(...)` around `/v1/users/me` and `/v1/users/me/avatar` in `cmd/main.go`
      <!-- design.md "Known gaps": handlers check the auth context, but nothing populates it, so
      both routes currently return 401 for every request. Highest-priority item in this spec. -->
- [ ] Replace `appConfig.MockSecretGetter` with `pkg/secrets/gcp` (or `aws`) and point `CONFIG_PATH` at a real source
      <!-- `⚠️ DEV CONFIG` block at the top of cmd/main.go -->
- [ ] Replace `mailer.NewLogMailer(logger)` with a real provider — provider not yet chosen
      <!-- `⚠️ DEV ONLY` block; see design.md Step 3 of the password-reset flow -->
- [ ] Replace `UseDefaultCredentials: true` with a service account / workload identity
      <!-- `⚠️ DEV ONLY` block on the storage config -->
- [ ] Implement login rate limiting (US-002 AC6) at middleware or gateway layer — not in the usecase
- [ ] Implement forgot-password throttling (US-003 AC5), keyed by email rather than IP
- [ ] Decide whether refresh-token rotation is in scope for this feature
      <!-- design.md "Known gaps": `pkg/jwt.Service.RefreshAccessToken` exists but no route calls
      it, and `insertSession` hardcodes `token_version = 1` without ever bumping it. Either add a
      `POST /v1/auth/refresh` route plus token_version bumping + reuse detection, or amend the
      US-001 note in requirements.md ("rotate mỗi lần dùng") to record that rotation is deferred.
      Blocks closing US-001's stated token contract. -->
- [ ] Resolve the access/refresh token lifetime contradiction recorded in requirements.md
      <!-- config.dev.yaml says 15m / 168h (7d); the original user story says 15 minutes / 30 days.
      design.md Step 4 also notes the session row's own 7-day `sessionTTL` is independent of the
      configured refresh-token lifetime. Pick the intended production values and reconcile. -->
- [ ] Harden avatar upload against decompression bombs / pixel-dimension abuse if the 5 MB byte cap alone proves insufficient
      <!-- Deliberately deferred: the byte cap plus DetectContentType allowlist is the whole guard
      today. Only re-open if a real incident or review finding justifies it. -->
- [ ] Decide the orphan-GCS-object policy for a failed `UpdateAvatarURL`
      <!-- ⚠️ Not confirmed in design.md Step 3 of the avatar flow — compensating delete vs accept
      the orphan. Blocks closing this phase. -->
- [ ] Select `avatar_url` in `FindByEmail`, or document why login's response may omit it
- [ ] Map `ErrUserNotFound` to 404 on `GET`/`PATCH /v1/users/me` instead of falling through to 500
- [ ] Add an explicit dimension/content check for avatar uploads if the 5 MB byte cap proves insufficient

## Phase 7 — Documentation follow-ups

- [ ] Refresh `docs/specs/db/register-erd.md` if any of migrations 0001-0004 change
- [ ] Note in `changelog.md` which OKF concept under `docs/knowledge/features/` needs refreshing by `bdd-okf-docs`
      <!-- No identity concept exists yet under docs/knowledge/features/ — the directory currently
      covers only the six content features. An onboarding concept would be a new addition, not a
      refresh. -->

## Deferred (separate feature, not blocking onboarding)

- Email / SMS verification of the address or phone number — `requirements.md` "Out of scope"
- Social login (Google / Apple)
- Changing email or phone number from the profile screen — needs a re-verification flow
- "Logout everywhere" and 2FA
- Account deletion
