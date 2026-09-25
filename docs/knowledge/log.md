# Knowledge Bundle Update Log

## 2026-09-25 (2)

- Added `packages/attestation.md` (`type: Package`, `status: draft`) for `pkg/attestation`, the
  provider-agnostic verification package the attestation feature consumes.
  - Read the root (`pkg/attestation/attestation.go`) and all three drivers (`mock`, `playintegrity`,
    `appattest`) plus their tests, and confirmed the registry shape: `Verifier` interface +
    `Register(ProviderType, FactoryFunc)` / `New(Config)` backed by a package-level map, drivers
    self-registering in `init()`, the same pattern as `pkg/sms`/`pkg/database`.
  - Established the dependency graph from the real imports: the root is a **stdlib-only leaf** (no
    internal `pkg/` imports); the drivers import only their parent + external deps
    (`google.golang.org/api/playintegrity/v1` for playintegrity, `github.com/fxamacker/cbor/v2` for
    appattest). No `pkg/utils`/`pkg/logging` dependency, unlike `middleware`/`sms`.
  - Marked `draft` because the package is **not a pure library**: `playintegrity.VerifyAndroid` does
    the untested Google `DecodeIntegrityToken` call (only `mapPayload` is unit-tested), and no
    deployment selects the real drivers — `services/appService` wires `mock` for both platforms. The
    "own test suite passing" shortcut to `stable` therefore does not apply.
  - `index.md`: appended the `packages/attestation.md` row.
  - `features/attestation.md`: rewrote the Related-concepts bullet that previously said "no Package
    concept exists yet" into a link to the new file, and corrected its Pending-work list accordingly.
- This closes the scope question from the previous entry: the feature and package concepts are now
  both written.

## 2026-09-25

- Documented US-006 and added `features/attestation.md` (`type: Feature`, `status: draft`). This is
  the concept the spec's `changelog.md` has been asking for since v1.0 ("concept `attestation` cần
  `il-okf-writer` tạo sau khi flow chạy end-to-end").
  - Read the whole vertical slice against the code: `domain/attestation.go` (entities, enums, the
    `AttestationRepository` + `AttestationVerifier` ports, `Reason*` constants), `domain/errors.go`
    (the five attestation sentinels), `usecase/attestation_usecase.go`, `repository/
    postgres_attestation_repository.go` + `attestation_verifier.go` (adapter), `delivery/http/
    attestation_handler.go` + `attestation_middleware.go`, `cmd/main.go` wiring, `config/config.go`
    + `config.dev.yaml` (`attestation:` block), `pkg/attestation/*` (interface + `mock` /
    `playintegrity` / `appattest` drivers), `migrations/0004_attestation.sql`, and the
    `docs/specs/features/attestation/` spec (requirements/design/tasks/changelog).
  - Marked `draft` because the flow has only been exercised with the **mock** driver: both
    `attestation.android.provider` and `attestation.ios.provider` are `"mock"` in `config.dev.yaml`
    (⚠️ DEV ONLY), the real `playintegrity`/`appattest` drivers are implemented and unit-tested but
    never wired or run, and `RequireAttestation` is built but deliberately not attached to any route
    (gate policy is a deferred business decision). There is also no automated E2E test for
    attestation (`services/appService/e2e/` has only register/login). Promote to `stable` once the
    flow runs end-to-end against the real drivers / a staging backend and the ACs are audited.
  - Recorded the deferred items (fraud-risk receipt, `attestation_challenges` cleanup, iOS key
    lifecycle) and the still-open "⚠️ Not confirmed" design question (`environment=development`
    rejection) in Pending work / Known gaps.
  - `index.md`: appended the `features/attestation.md` row to the concept table.
- Scope note: wrote the **Feature** concept only. `pkg/attestation` (interface + 3 drivers, the
  provider-agnostic registry pattern shared with `pkg/sms`) deserves its own `packages/attestation.md`
  `Package` concept — flagged in the concept's Related concepts as not yet written.

## 2026-09-22 (4)

- Added `packages/middleware.md` (`type: Package`, `status: draft`) for `pkg/middleware`, which
  became wired into the server by `http-middleware-wiring` (implemented 2026-09-22). This populates
  the `packages/` section for the first time.
  - Documented the real HTTP surface (`Logger`, `CORS`, `Recovery`) and the gRPC interceptors, the
    `pkg/utils`-only internal dependency, and the consumer chain
    `Recovery(CORS(Logger(WithAPIFallback(mux))))` built by `newHandler` in
    `services/appService/cmd/main.go`, including why the order is load-bearing (Recovery catches
    every layer; CORS answers preflight before Logger, so preflight is not logged).
  - Marked `draft` because the chain and the new preflight envelope are proven by unit tests
    (`pkg/middleware/middleware_test.go`, `services/appService/cmd/main_test.go`) but were **not**
    exercised on a running binary + Postgres (the curl checks in the spec's Phase 3 are unrun);
    the gRPC `Recovery`/`StreamRecovery` interceptors are untested and unused. Promote to `stable`
    once a real server is exercised.
  - `index.md`: moved `packages/` from the "Planned" list into the active sections and added the
    concept row.
- Fixed drift the change created in the two feature concepts, after re-reading `cmd/main.go`:
  - `features/register.md`: its Wiring row still described `Recovery(WithAPIFallback(mux))`; rewritten
    to `newHandler(mux)` = `Recovery(CORS(Logger(WithAPIFallback(mux))))`, the "missing-middleware"
    sentence clarified to mean auth only, and a Related-concepts link added.
  - `features/login.md`: Related-concepts link added (its Wiring row was not wrong, so left as is).
  - Both `generated.at` timestamps refreshed; both keep their existing `status` (`register` draft,
    `login` stable) since the request/response status and bodies they document did not change.
- Sources read to write it: `pkg/middleware/{middleware,grpc}.go`, `pkg/middleware/README.md`,
  `pkg/middleware/middleware_test.go`, `services/appService/cmd/main.go` +
  `main_test.go`, and the `http-middleware-wiring` / `api-response-envelope` specs. No `resource:`
  path is guessed; `pkg/middleware/` was confirmed on disk and no `verified:` field was written.

## 2026-09-22 (3)

- Removed the `docs/ARCHITECTURE.md` reference from the bundle. That file has been deleted repo-wide
  (it was stale: §3–§4 still described removed US-003…US-010 work).
  - `index.md`: the introduction no longer points at it; it now tells the reader to use the code and
    the per-package/layer `README`s and to verify relationships from real imports.
  - `features/register.md` and `features/login.md`: dropped it from `sources:` and removed the body
    references (§4 gap list, §3.2 story table).
  - Also refreshed `features/register.md`'s "Known gaps vs. design" to cite onboarding v5.2 and
    corrected `appConfig.MockSecretGetter` → `appConfig.NewMockSecretGetterFromYAML` in `login.md`.
- Repo-wide, `AGENTS.md`, `CLAUDE.md` and the root `README.md` were updated to the current tree and
  no longer reference the deleted file.

## 2026-09-22 (2)

- Fixed two factual slips found in a full bundle-vs-code audit:
  - The send-cap wording in `features/register.md` (Summary + "Send policy") said "the fourth
    counted send" — only three sends are counted. The fourth send overall (the third counted) is
    still delivered and sets the 30-minute `otp_send_limits.locked_until`; the fifth is refused.
    The same wording is corrected in this log's `## 2026-09-22` entry below.
  - `features/register.md`'s Domain/Errors row omitted the input-validation sentinels
    (`ErrInvalidPhone`, `ErrInvalidEmail`, `ErrInvalidPassword`) and `ErrUnsupportedOTPPurpose`.
- Reworded the `index.md` introduction: `../ARCHITECTURE.md` §2.2 is the verified `pkg/` import
  graph, but §3–§4 have drifted, so the bundle no longer calls the whole file the source of truth.
- Corrected this log's `## 2026-09-21` entry: the login lock is recorded by the 5th consecutive
  failure (the 6th attempt is the one rejected), not "lockout on the 6th attempt".

## 2026-09-22

- Updated `features/register.md` after onboarding v5.0 (Phase 9) + v5.1 (Phase 10), re-reading the
  whole vertical slice against the code. What changed:
  - The send cap is no longer `otp_codes.resend_count` carried forward; it is a per-phone
    `otp_send_limits` row (`applySendLimits`, migration `0003`). One attempt gets four sends (first
    free + three counted), the 4th send (the 3rd counted) is still delivered and sets a 30-minute
    `locked_until`, and `request-otp`
    spends from the same counter as `resend-otp`. Removed the stale "resend_count never resets / SMS
    cap pending a decision" item — that was the bug v5.0 fixed.
  - Documented the two locks as independent despite both being 30 minutes: the send lock
    (`otp_send_limits.locked_until`) must not block `verify-otp`, while the wrong-guess lock
    (`otp_codes.locked_until`) does.
  - `pending_registration_ttl` is now 24h (was 90s). A pending row for the same phone no longer
    blocks `request-otp` (`FindConflicts` checks only `users` for the phone, plus another phone's
    live pending for the email); `resend-otp` issues a new code even when the previous code expired,
    and leaves the payload untouched. Added AC15 to the summary.
  - Refreshed the config table (`send_cooldown`/`max_send_attempts`/`send_lock_duration`,
    `lock_duration` 30m, `pending_registration_ttl` 24h), the Architecture table
    (`domain/otp_send_limit.go`, `ErrOTPSendLimited` replacing `ErrOTPResendExhausted`, five tables),
    all three Request-flow diagrams, the Validation rules (AC6), and Verification.
  - Added Verification for the three 2026-09-22 E2E tests and the new repository tests.
- Status stays `draft`: the OTP (`"111111"`) and SMS (`pkg/sms/logsms`) placeholders remain, the
  cleanup job is still deferred (now more load-bearing because the payload lives 24h), and the US-001
  ACs have not been audited end-to-end. Promote to `stable` once (a) the placeholders are replaced
  and (b) `golang-e2e-verification` maps the ACs to tests.
- Updated the `features/register.md` row in `index.md`.

## 2026-09-21 (5)

- Updated `features/register.md` after onboarding v4.5 (Phase 8): `FindConflicts` now counts only
  unexpired `pending_registration` rows, so an expired registration restarts via `request-otp`
  instead of returning `409` forever.
- Removed the corresponding Known gap and replaced the "expired rows are not inert" pending item.
- Added the new E2E test `register_e2e_test.go` to Verification, and noted the unresolved
  `resend_count` reset / SMS cap decision.
- Status stays `draft`: OTP/SMS placeholders remain, and the US-001 ACs are not fully audited.

## 2026-09-21 (4)

- Re-verified `features/register.md` and `features/login.md` line by line against the code.
  Both stay at their current `status`.
- `register.md` corrections:
  - A malformed `otp_code` is a `400` `VALIDATION_ERROR` from the struct tag. It does not reach
    the usecase and does not count as an attempt (the concept previously said it became
    `OTP_INVALID`).
  - Added the missing steps: `NormalizePhone` in resend/verify; no active OTP → `OTP_EXPIRED`;
    `ReissueOTP` extends `pending_registration.expires_at`; `SavePendingAndIssueOTP` carries
    `resend_count` forward; `LockOTP` also sets `used_at`.
  - Added the status/code on every branch, the `otp:` config values, and the phone-before-email
    conflict order.
- New Known gap in `register.md`, reproduced on the real binary: an expired `pending_registration`
  row is never deleted, and `FindConflicts` ignores `expires_at`. An abandoned registration
  therefore blocks its phone and email with `409` `PHONE_TAKEN` indefinitely, contradicting
  `design.md` flow 3 Step 2 and AC12's "làm lại từ bước 1". This also corrects the earlier claim
  (in this bundle and in the onboarding Deferred note) that expired rows are inert.
- `login.md` corrections: the lock is set on the 5th failure (not "6th failure"); documented the
  `RecordLoginFailure` reset rules, `FindSessionByRefreshHash` matching current or previous hash,
  and the `RotateSession` re-check.

## 2026-09-21 (3)

- Refreshed `features/register.md` from the code (`status: draft`, unchanged). Drift fixed:
  - `repository` does not import `usecase`.
  - `TokenIssuer` is now `func(userID, sessionID)` and the `verify-otp` access token carries `sid`.
  - `password` has a `required` tag.
  - `verify-otp` answers `201`.
  - The dev secret resolver is `NewMockSecretGetterFromYAML`.
  - Envelope, fallback and recovery wiring added.
- Added Verification: registration repository tests ran against Postgres and pass, and the E2E
  harness exercises `request-otp`/`verify-otp` as login setup.
- Dropped the resolved items: the `pkg/utils` failures are fixed, and the cleanup job was accepted
  for MVP.
- Linked [Login](features/login.md).
- Still `draft`: OTP/SMS are dev placeholders, and the US-001 ACs have no E2E audit yet.

## 2026-09-21 (2)

- Promoted `features/login.md` from `draft` to `stable`. The condition
  `docs/specs/features/login/requirements.md` set is now met: the `golang-e2e-verification` AC
  audit is done. `services/appService/e2e/login_e2e_test.go` runs the real binary against
  testcontainers Postgres and passes for AC1–AC15 (see
  `docs/specs/features/login/test-coverage-report.md`).
- In the concept: added a Verification bullet for the E2E test and the middleware-removal check,
  dropped the "no automated E2E" pending item, and updated the index row.

## 2026-09-21

- Documented US-002 (login, refresh, logout) and added `features/login.md` (`status: draft`).
  Checked against `domain/login.go`, `domain/session_token.go`, `usecase/login_usecase.go`,
  `repository/postgres_login_repository.go`, `delivery/http/login_handler.go` + `errors.go`,
  `cmd/main.go` wiring, and `migrations/0002_login_devices_and_rate_limit.sql`.
- Kept `draft` because `docs/specs/features/login/requirements.md` requires it until the AC audit
  (`golang-e2e-verification`) is done. The whole flow was run end-to-end in this pass against the real binary
  and Postgres 16 (Docker `innerlamp-pg`), using curl:
  - login and refresh, including rotation and reuse-revoke
  - same-device supersede
  - lockout: the 5th consecutive failure records the 15-minute lock, so the 6th attempt is rejected
    with `429 LOGIN_LOCKED`
  - logout returning `200` `{"success": true}`, idempotent
  - refresh after logout, and logout without a token, both rejected
- `TestLoginRepository_*` also ran against that database (not skipped) and passed.
- Still pending, listed in the concept: an automated E2E audit (`golang-e2e-verification`,
  blocked on the `testcontainers-go` decision), a cleanup job for `login_attempts`/sessions, and
  missing folder READMEs.
- Added the concept to `index.md`, and removed stray `****` markers from its heading and from the
  `api/` bullet.

## 2026-09-17

- Documented US-001's registration flow (`request-otp`/`resend-otp`/`verify-otp`) and added
  `features/register.md` (`status: draft`). Marked `draft` because `usecase/register_usecase.go`'s
  `generateOTP()` returns a hardcoded `"111111"` instead of a random code, `cmd/main.go` wires
  `pkg/sms/logsms` (logs the OTP rather than sending it) and `appConfig.MockSecretGetter` (both
  `⚠️ DEV ONLY`/`⚠️ DEV CONFIG`), there is no cleanup job for expired `pending_registration`/
  `otp_codes` rows, and this pass did not itself run `postgres_registration_repository_test.go`'s
  integration test against a live Postgres to confirm the end-to-end claim
  `docs/specs/features/onboarding/design.md` (v3.3) makes. Promote to `stable` once the OTP/SMS
  placeholders are replaced with production equivalents and the flow is independently re-exercised
  end-to-end against a real/staging Postgres.
- Created the `features/` section entry and the bundle intro/sections list in `index.md` (both
  files were zero-byte before this pass) and added `features/register.md` to its concept table.
