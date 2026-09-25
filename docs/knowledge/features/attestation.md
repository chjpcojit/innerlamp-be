---
type: Feature
title: Attestation
description: US-006's device attestation flow — mint single-use challenge nonces and verify Play Integrity (android) / App Attest (ios) evidence, returning a Verdict — plus a RequireAttestation middleware that gates sensitive routes behind a per-request iOS assertion.
resource: services/appService/
tags: [auth, identity, security, attestation]
status: draft
generated: { by: claude-code, at: 2026-09-25T11:13:05+07:00 }
sources:
  - path: docs/specs/features/attestation/requirements.md
    kind: spec
  - path: docs/specs/features/attestation/design.md
    kind: spec
  - path: docs/specs/features/attestation/tasks.md
    kind: spec
  - path: docs/specs/features/attestation/changelog.md
    kind: spec
  - path: docs/specs/api/openapi.yaml
    kind: openapi
  - path: migrations/0004_attestation.sql
    kind: migration
  - path: services/appService/domain/README.md
    kind: implementation-notes
  - path: services/appService/usecase/README.md
    kind: implementation-notes
---

# Summary

Three public routes implement US-006 (device attestation) in the `attestation` module. They let a
client prove to the backend that it is the real, untampered app running on a real device:

- `POST /v1/attestation/challenge` mints a single-use nonce (`purpose = 'attest'`) for either
  `android` or `ios`.
- `POST /v1/assertion/challenge` mints a single-use nonce (`purpose = 'assert'`) for `ios` only —
  the per-request assertion flow.
- `POST /v1/attestation/verify` verifies the evidence bound to a challenge and returns a `Verdict`:
  for `android` a Play Integrity token is decoded server-to-server by Google; for `ios` an App
  Attest attestation object (the one-time `attest` key-registration phase) is verified locally.

A fourth piece, `RequireAttestation`, is a middleware (not a route) that gates sensitive routes
behind a per-request iOS App Attest assertion: it reads three headers, rebuilds `clientData` from
the real request, verifies the ES256 assertion against the stored key, and passes or rejects `401`.

Two design decisions shape the whole feature. First, a **negative verdict is not an HTTP error**:
`verify` always answers `200`, with `attested:false` plus a `reason` on failure, because the caller
is allowed to learn "your app is not recognized" but must not be able to distinguish a probeable
failure (design.md decision #10). Only a challenge-state problem (`400 ATTESTATION_CHALLENGE_INVALID`)
and an upstream failure (`502 ATTESTATION_UPSTREAM_ERROR`) are errors. Second, the two platforms are
**deliberately asymmetric**: Android has no durable key — every verify is a fresh integrity token
bound to a challenge, and only the `attestation_challenges` table is involved — while iOS has a
durable per-device key (`key_id`) registered once via `attest` and then used for many `assert`
operations, persisted in `attestation_keys`. Android verification is server-to-server (Google
decodes); iOS verification is local crypto (no call to Apple).

**Current limitation**: in dev, both platform drivers are `pkg/attestation/mock` (⚠️ DEV ONLY) —
the mock echoes the configured package name / verdict back so the flow round-trips without Google or
Apple. The real drivers (`pkg/attestation/playintegrity`, `pkg/attestation/appattest`) are
implemented and tested but not yet selected by config, and `RequireAttestation` is built and tested
but not wired to any route (choosing which routes to gate is a deferred business decision).

# Architecture

Clean Architecture inside `services/appService/`: `delivery/http` → `usecase` → `domain`, and
`repository` → `domain`. `domain` imports nothing from the other layers; the concrete crypto/CBOR
and the Google server call live in `pkg/attestation`, and `domain.AttestationVerifier` is the seam
to it — the usecase depends on the port, not on the package. The usecase owns the challenge/verdict
policy through `config.AttestationConfig` (an intentional import of `services/appService/config`,
like `OTPConfig`). Each atomic write is one repository method wrapping one transaction: `SaveKey`
and `UpdateKeyCounter` both consume the challenge in the same transaction, so a key is never
persisted (or a counter never advanced) against a still-unused nonce.

| Layer | Type | File |
|---|---|---|
| Domain / Entities | Domain Entity | `services/appService/domain/attestation.go` — `AttestationChallenge`, `AttestationKey`, `Verdict`, `AndroidVerdict`, `IOSAttestationResult`; enums `Platform` (`android`/`ios`) and `ChallengePurpose` (`attest`/`assert`); the `Reason*` constants (`app_not_recognized`, `attestation_invalid`, `unknown_key`, `counter_not_increased`, `assertion_invalid`, `missing_evidence`). Plain data, no json/db tags |
| Domain / Ports | interfaces | `services/appService/domain/attestation.go` — `AttestationRepository` (`SaveChallenge`, `FindChallenge`, `MarkChallengeUsed`, `SaveKey`, `FindKey`, `UpdateKeyCounter`) and `AttestationVerifier` (`VerifyAndroid`, `VerifyIOSAttestation`, `VerifyIOSAssertion`) |
| Domain / Errors | sentinel errors | `services/appService/domain/errors.go` — `ErrAttestationChallengeInvalid`, `ErrAttestationUpstreamError`, `ErrAttestationKeyNotFound`, `ErrAttestationCounterNotIncreased`, `ErrAttestationAssertionInvalid` |
| Usecase | UseCase | `services/appService/usecase/attestation_usecase.go` — `RequestChallenge` (crypto/rand nonce → `SaveChallenge`), `Verify` (branches android/ios), `VerifyAssertion` (for the middleware). Holds the Android identity checks (package/nonce/freshness/verdict) and the freshness window |
| Repository | Repository | `services/appService/repository/postgres_attestation_repository.go` — all SQL + transactions for `attestation_challenges` / `attestation_keys`; `services/appService/repository/attestation_verifier.go` — adapter mapping `pkg/attestation.Verifier` to `domain.AttestationVerifier` (generic shapes + provider errors → domain types/sentinels) |
| Delivery / Handler | Handler | `services/appService/delivery/http/attestation_handler.go` — 3 handlers + DTOs; `platform` oneof validation; cross-field rules in `validateVerifyFields`; base64-decodes the ios `attestation` field here (delivery owns the wire format) |
| Delivery / Middleware | Middleware | `services/appService/delivery/http/attestation_middleware.go` — `RequireAttestation`: reads the three headers, tees the body, builds `clientData = challenge ‖ method ‖ path ‖ SHA256(body)`, calls `VerifyAssertion`, passes or `401` |
| Wiring | composition root | `services/appService/cmd/main.go` — builds one `pkg/attestation` driver per platform, wraps them in `repository.NewAttestationVerifier`, registers the 3 public routes; `RequireAttestation` is built but deliberately NOT wired (comment in source) |
| External package | Package | `pkg/attestation/attestation.go` — `Verifier` interface + `Register`/`New` registry; drivers `pkg/attestation/mock`, `pkg/attestation/playintegrity`, `pkg/attestation/appattest` (each self-registers via `init()`) |
| Migration | schema | `migrations/0004_attestation.sql` — `attestation_challenges` + `attestation_keys` (see below) |

`pkg/` dependencies: `pkg/attestation` (the new provider-agnostic verification package), `pkg/database`
(transactions), `pkg/utils` (`DecodeJSONBodyStrict`, `ApiResponse` writers), `pkg/validation`
(`platform` oneof via the shared validator), `pkg/logging`. `pkg/attestation/appattest` additionally
pulls `github.com/fxamacker/cbor/v2` (CBOR) and `playintegrity` pulls `google.golang.org/api/playintegrity/v1`.

Policy comes from `config.dev.yaml`'s `attestation:` block (`config.AttestationConfig`, passed to
`NewAttestationUsecase`):

| Setting | Value | Meaning |
|---|---|---|
| `challenge_ttl` | 5m | how long a minted nonce can be used (AC1/AC3) |
| `freshness_window` | 5m | how recent a Play Integrity token's `timestampMillis` must be (AC5) |
| `android.provider` | `mock` | Play Integrity driver — `playintegrity` in a real deployment |
| `android.package_name` | `com.innerlamp.app` | decoded token's `requestPackageName` must match (AC5) |
| `android.required_app_recognition_verdict` | `PLAY_RECOGNIZED` | config-driven threshold (design.md decision #11) |
| `ios.provider` | `mock` | App Attest driver — `appattest` in a real deployment |
| `ios.app_id` | `TEAMID.com.innerlamp.app` | RP ID hash = `SHA256(app_id)` |

`android.credentials_secret` (optional) names the Play Integrity service-account secret; resolved via
`SecretGetter` into `Credentials` at load time, passed to the driver as `credentials_json`.

# API surface

| Method | Path | Auth | Handler |
|---|---|---|---|
| `POST` | `/v1/attestation/challenge` | public | `AttestationHandler.Challenge` |
| `POST` | `/v1/assertion/challenge` | public | `AttestationHandler.AssertionChallenge` |
| `POST` | `/v1/attestation/verify` | public | `AttestationHandler.Verify` |

All three are public in `cmd/main.go` — the caller has no account yet, so there is nothing to
authenticate; this is by design, not a missing auth gap. The generic
`Recovery`/`CORS`/`Logger` chain ([packages/middleware.md](../packages/middleware.md)) wraps them but
does not authenticate.

Wire contract: `docs/specs/api/openapi.yaml` (3 paths + assertion-header schemas). No per-endpoint
`api/` concepts exist yet.

The `RequireAttestation` middleware is **not** in the API-surface table because it is not an
endpoint and is not wired to any route today; its header contract is:

```text
X-Attestation-Key-Id:     <key_id base64>
X-Attestation-Assertion:  <assertion base64>
X-Attestation-Challenge:  <nonce>
```

# Request flow

```
POST /v1/attestation/challenge                         (public)
  └─ AttestationHandler.Challenge                      (delivery/http)
       ├─ DecodeJSONBodyStrict + validate (platform oneof=android ios)
       └─ AttestationUsecase.RequestChallenge(platform, purpose=attest)
            ├─ generateNonce()  (crypto/rand, 32 bytes → base64)
            └─ AttestationRepository.SaveChallenge     (repository, 1 INSERT)
                 → 200 { challenge, expires_in_seconds }

POST /v1/assertion/challenge                           (public)
  └─ AttestationHandler.AssertionChallenge
       ├─ DecodeJSONBodyStrict + validate (platform oneof=ios)
       └─ AttestationUsecase.RequestChallenge(platform, purpose=assert)
            └─ SaveChallenge (purpose='assert')         → 200 { challenge, expires_in_seconds }

POST /v1/attestation/verify                            (public)
  └─ AttestationHandler.Verify                         (delivery/http)
       ├─ DecodeJSONBodyStrict + validate + validateVerifyFields (cross-field)
       ├─ decodeAttestation (base64 → []byte; malformed → 400 VALIDATION_ERROR)
       └─ AttestationUsecase.Verify(input)             (usecase)
            ├─ [android] FindChallenge(challenge, purpose=attest)  (400 if bad)
            │    → Verifier.VerifyAndroid(token, challenge)  → pkg/attestation → Google
            │        error → 502 ATTESTATION_UPSTREAM_ERROR (challenge NOT consumed)
            │    → androidVerdictReason(): package / nonce / freshness / verdict check
            │    → MarkChallengeUsed(challenge)  (0 rows → 400 challenge already used)
            │    → 200 { attested: reason=="", platform, signals?, reason? }
            └─ [ios] FindChallenge(challenge, purpose=attest)  (400 if bad)
                 → Verifier.VerifyIOSAttestation(keyID, attestation, challenge)  (local crypto)
                     error → MarkChallengeUsed + 200 { attested:false, reason:attestation_invalid }
                 → SaveKey(key, challenge)   (repository, 1 tx: upsert key + consume challenge)
                 → 200 { attested:true, platform:ios, signals:{environment} }

(sensitive route, future) — RequireAttestation        (middleware, NOT wired today)
  └─ Wrap(next)
       ├─ read 3 headers; any missing → 401 ATTESTATION_FAILED (reason=missing_evidence)
       ├─ tee body (MaxBytesReader 1 MB) + base64-decode assertion
       ├─ buildClientData = challenge ‖ method ‖ path ‖ SHA256(body)
       └─ AttestationUsecase.VerifyAssertion(input)    (usecase)
            ├─ FindChallenge(challenge, purpose=assert)   → 401 ATTESTATION_CHALLENGE_INVALID
            ├─ FindKey(keyID)                              → 401 (reason=unknown_key)
            ├─ Verifier.VerifyIOSAssertion(publicKey, assertion, clientData, storedCounter)
            │    → ErrCounterNotIncreased → 401 (reason=counter_not_increased)
            │    → ErrAssertionInvalid     → 401 (reason=assertion_invalid)
            └─ UpdateKeyCounter(keyID, newCounter, challenge)  (repository, 1 tx)
                 → next.ServeHTTP
```

# Validation rules

- Body shape (all three routes) — `utils.DecodeJSONBodyStrict` via `writeDecodeError`: unknown field
  or malformed JSON → `400 VALIDATION_ERROR`; oversized body → `413 PAYLOAD_TOO_LARGE`.
- `platform` — `challengeRequest.Platform` tag `required,oneof=android ios`;
  `assertionChallengeRequest.Platform` tag `required,oneof=ios` (only iOS has assertions). A wrong
  value is a struct-tag `400 VALIDATION_ERROR` and no challenge row is written (AC2/AC4).
- `verify` cross-field rules (AC15/AC16) — not struct tags; `validateVerifyFields` in the handler:
  `android` requires `token` (else `400` `token:required`) and forbids `key_id`/`attestation` (else
  `400` `key_id:excluded`); `ios` requires `key_id` + `attestation` (else `400`
  `key_id:required`/`attestation:required`) and forbids `token`. All are `400 VALIDATION_ERROR`
  before any Google/Apple call.
- `attestation` (ios verify) — base64-decoded in the handler (`decodeAttestation`); a malformed
  value is `400 VALIDATION_ERROR` (`attestation: base64`), not a crypto failure.
- `challenge` — `verifyRequest.Challenge` tag `required`. State validity (known, unused, unexpired)
  is checked in the usecase via `FindChallenge` and surfaces as `ErrAttestationChallengeInvalid`
  (`400 ATTESTATION_CHALLENGE_INVALID` on verify, `401` on the middleware) — one code for
  unknown/expired/used so the caller cannot probe which state (AC6/AC14).

Status/code mapping lives in `delivery/http/errors.go` `writeDomainError`
(`ErrAttestationChallengeInvalid` → `400 ATTESTATION_CHALLENGE_INVALID`,
`ErrAttestationUpstreamError` → `502 ATTESTATION_UPSTREAM_ERROR`) and in
`attestation_middleware.go` `writeAttestationError` (the assertion sentinels → `401`).

# Verification

- `go test -count=1 ./services/appService/usecase/` — `TestAttestationUsecase_*` table-driven tests
  over `domain/mock_attestation_repository.go` + `domain/mock_attestation_verifier.go`: challenge
  (attest + assert), android verify pass/fail per verdict branch, challenge-invalid, upstream error
  leaving the challenge unconsumed, ios attest pass/verify-fail, ios assert counter
  increase/no-increase/unknown-key/challenge-invalid/signature-fail.
- `go test -count=1 ./services/appService/repository/` against Postgres 16 (not skipped):
  `TestAttestationRepository_*` race two `MarkChallengeUsed` on the same challenge (only one wins),
  `FindChallenge` filters by `purpose`, key upsert is idempotent, counter update; and
  `TestAttestationVerifier_*` map the provider errors onto the domain sentinels.
- `go test -count=1 ./services/appService/delivery/http/` — `attestation_handler_test.go` +
  `attestation_middleware_test.go`: success per branch, every error code, malformed bodies, iOS
  fields on an android request (and vice versa) rejected, middleware pass/`401` per reason.
- `go test -count=1 ./pkg/attestation/...` — per-driver tests plus `attestation_test.go` (registry:
  nil config, missing provider, unregistered provider, factory error).
- Phase 6 (per `tasks.md`): `go build ./...`, `go vet ./...`, `go test -short ./...` clean, and the
  flow was driven end-to-end with the **mock** driver + a real Postgres via curl (challenge → verify
  android/ios → re-using a challenge → `400`).
- There is **no automated black-box E2E test** for attestation (`services/appService/e2e/` holds only
  `login_e2e_test.go` and `register_e2e_test.go`), and the real drivers have never been exercised —
  only their unit tests run. This is the main reason the concept is `draft`, not `stable`.

# Pending work

- `cmd/main.go` selects `cfg.Attestation.Android.Provider` / `cfg.Attestation.IOS.Provider`, which are
  both `"mock"` in `config.dev.yaml` (⚠️ DEV ONLY). The real drivers are implemented but never wired:
  flip to `playintegrity` / `appattest` once the Play service-account secret and the published
  `app_id` (`{team_id}.{bundle_id}`) exist. Until then the verify path round-trips through the mock.
- `RequireAttestation` is built (`delivery/http/attestation_middleware.go`) but **not wired to any
  route** in `cmd/main.go`. Choosing which routes to gate, and how to handle a rejection, is a
  business decision deferred to a future feature (design.md "Out of scope"). Promote the middleware
  to a live gate only as part of that feature.
- No automated E2E test for attestation (see Verification) — the `golang-e2e-verification` AC audit
  has not been run for US-006, and there is no `attestation_e2e_test.go` next to the register/login
  ones.
- Fraud-risk receipt is deferred (design.md decision #7): `attestation_keys.receipt` is stored on a
  successful iOS attestation but never sent to `data-*.devicecheck.apple.com` for a risk metric.
- No cleanup job for expired `attestation_challenges` rows (the `idx_attestation_challenges_expires`
  index exists for it). Same deferred-cleanup situation as `otp_codes`/`pending_registration`.
- No iOS key-lifecycle policy: a key lives until overwritten or the user is deleted
  (`ON DELETE SET NULL`). Re-attestation/revocation is out of scope for this version.
- `pkg/attestation` has no `README.md`, and `services/appService/repository/`,
  `delivery/http/`, and `cmd/` still have no `README.md` (pre-existing, not attestation-specific).
- No `migrations/0004-attestation.md`, `api/`, `entities/`, or `usecases/` concepts exist yet for this
  feature's slice (the `Package` concept does exist — [packages/attestation.md](../packages/attestation.md)).
- Promote to `stable` once the flow is exercised end-to-end against the **real** drivers (or a
  staging backend) and the ACs are audited via `golang-e2e-verification`.

# Known gaps vs. design

- The one `⚠️ Not confirmed` marker in `design.md` (iOS attest Step 3) is still open: whether
  production should reject an `environment=development` key (a possible
  `ios.reject_development_keys` config flag). Not a blocker; it would be an added config flag.
- Everything else in `design.md`'s "Deliberately stubbed" / "Known gaps" list is tracked in Pending
  work above rather than being an undocumented divergence.

# Related concepts

- [Register](register.md) — US-001. `attestation_keys.user_id` is a nullable FK to `users` with
  `ON DELETE SET NULL`; the two features share `delivery/http/errors.go`'s `writeDomainError` and the
  `Recovery`/`CORS`/`Logger` chain from [packages/middleware.md](../packages/middleware.md).
- [Login](login.md) — US-002. Same shared handler-error surface and middleware chain; no direct
  dependency, but the `RequireAttestation` middleware would sit alongside its `pkg/jwt/middleware`
  logout wrapper on any future gated route.
- [packages/middleware.md](../packages/middleware.md) — the generic HTTP middleware chain; distinct
  from `RequireAttestation`, which lives in `delivery/http` (business attestation, not generic infra).
- [packages/attestation.md](../packages/attestation.md) — the `pkg/attestation` package this feature
  consumes: interface `Verifier` + `Register`/`New` registry and the `mock`/`playintegrity`/
  `appattest` drivers, with the provider-agnostic pattern it shares with `pkg/sms`.
- No `Migration` concept exists yet for `migrations/0004_attestation.sql` (the
  `attestation_challenges` + `attestation_keys` tables).
- No `Domain Entity` or `UseCase` concepts exist yet for `AttestationChallenge`, `AttestationKey`,
  `Verdict`, or `AttestationUsecase` as standalone files — each is documented only inside this
  feature's Architecture table, per `concept_types.md`'s "Feature vs UseCase" rule.
