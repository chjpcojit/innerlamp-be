---
type: Package
title: pkg/attestation
description: Provider-agnostic device-attestation verification — one Verifier interface over three drivers (mock, playintegrity for Android server-to-server decode, appattest for iOS local crypto) selected through the Register/New factory registry.
resource: pkg/attestation/attestation.go
tags: [infra, package, security, attestation, architecture]
status: draft
generated: { by: claude-code, at: 2026-09-25T11:32:41+07:00 }
sources:
  - path: docs/specs/features/attestation/design.md
    kind: spec
  - path: docs/specs/features/attestation/requirements.md
    kind: spec
---

# Summary

`pkg/attestation` hides platform attestation verification behind one `Verifier` interface, so a
service can prove "this is the real, untampered app on a real device" without importing a platform
SDK directly. It follows the same provider-agnostic registry pattern as `pkg/sms` and `pkg/database`:
this package declares the interface and the factory map, each driver subpackage self-registers in its
`init()`, and a consumer activates a driver by importing its subpackage and building through `New`.

The interface encodes the two-platform asymmetry of attestation directly:

- `VerifyAndroid` is **server-to-server**: the provider decodes a Play Integrity token on Google's
  servers (only Google can — the token is a JWE of a JWS).
- `VerifyIOSAttestation` and `VerifyIOSAssertion` are **local crypto**: the provider verifies an App
  Attest attestation object or an ES256 assertion itself, never calling Apple.

The concrete crypto/CBOR and the Google call live entirely in the driver subpackages; the root
package has no dependency on any service's domain types. Converting the generic result shapes
(`AndroidResult`, `IOSAttestationResult`) and the provider errors into a service's domain types is
`services/appService/repository`'s job, done by the `attestation_verifier.go` adapter — not this
package's.

The three drivers are all implemented. `mock` is the only one the service wires today (dev, ⚠️ DEV
ONLY); `playintegrity` and `appattest` are implemented and unit-tested but not yet selected by config
(both `attestation.*.provider` are `"mock"` in `config.dev.yaml`).

# Interface

| Member | Signature | Behaviour |
|---|---|---|
| `Verifier` | `type Verifier interface { VerifyAndroid(ctx, token, challenge string) (*AndroidResult, error); VerifyIOSAttestation(ctx, keyID string, attestation []byte, challenge string) (*IOSAttestationResult, error); VerifyIOSAssertion(ctx, publicKey []byte, assertion, clientData []byte, storedCounter int64) (int64, error) }` | What a consumer gets: three methods mapping one-to-one onto the three operations the attestation feature needs. An error from `VerifyAndroid` means the decode could not be performed (unreachable/timeout/non-2xx); from `VerifyIOSAttestation` a failed crypto check; from `VerifyIOSAssertion` either `ErrCounterNotIncreased` or `ErrAssertionInvalid`. |
| `Register` | `func Register(provider ProviderType, factory FactoryFunc)` | Binds a provider string to a factory in the package-level map. Called only from a driver subpackage's `init()`, never from this file — adding a provider must not require editing `pkg/attestation`. |
| `New` | `func New(config *Config) (Verifier, error)` | Validates config (`nil` or empty `Provider` → `ErrInvalidConfig`; unregistered → `ErrUnsupportedProvider`), then calls the registered factory. Does **not** do network I/O. |
| `Config` | `type Config struct { Provider ProviderType; Options map[string]string }` | `Options` carries provider-specific settings as a string map (e.g. `package_name`, `app_id`, `credentials_json`, `root_cert_pem`); a provider needing more structure is free to ignore the map. |

`ProviderType` is a string enum with three registered values: `ProviderMock` (`"mock"`),
`ProviderPlayIntegrity` (`"playintegrity"`), `ProviderAppAttest` (`"appattest"`). Common errors:
`ErrUnsupportedProvider`, `ErrInvalidConfig`, `ErrCounterNotIncreased`, `ErrAssertionInvalid`,
`ErrPlatformNotSupported` (a provider answering a platform it does not implement — e.g. the
playintegrity provider asked for an iOS assertion).

# Drivers / providers

| Driver | Subpackage | Status |
|---|---|---|
| `mock` | `pkg/attestation/mock/` | Implemented (⚠️ DEV ONLY). Fakes every result; echoes the configured package name / verdict / nonce back so a dev round-trip produces `attested:true` with no Google or Apple. Modes: `pass` / `fail` / `upstream_error` (android) and `pass` / `invalid` / `replay` (assertion). Registers `ProviderMock`. |
| `playintegrity` | `pkg/attestation/playintegrity/` | Implemented. Android server-to-server: calls `playintegrity/v1.DecodeIntegrityToken` with a service account (scope `playintegrity`); auth via `credentials_json` / `credentials_file`, falling back to ADC. iOS methods return `ErrPlatformNotSupported`. Registers `ProviderPlayIntegrity`. |
| `appattest` | `pkg/attestation/appattest/` | Implemented. iOS local crypto: decodes CBOR, verifies the `x5c` chain against Apple's App Attest root (embedded; overridable via `root_cert_pem`), nonce/RP ID/counter/aaguid, and the ES256 assertion signature. Android method returns `ErrPlatformNotSupported`. Registers `ProviderAppAttest`. |

Consumers activate a driver by importing its subpackage (whose `init()` registers it) and building
through `attestation.New`. `services/appService/cmd/main.go` imports all three and builds **two**
verifiers — one per platform — because the backends are asymmetric: `android` selects
`playintegrity`/`mock`, `ios` selects `appattest`/`mock`. A single `Verifier` is expected to answer
only its own platform(s), so the composition root wraps the two in
`repository.NewAttestationVerifier(android, ios)`.

# Lifecycle

Construction is synchronous and validates only — the repo's `New`-does-not-dial convention:
`mock.New` reads options with pass-mode defaults; `playintegrity.New` requires `package_name` and
stores the credentials but builds the Google service lazily per `VerifyAndroid` call;
`appattest.New` requires `app_id` and parses the root certificate eagerly (config validation) but
does no network I/O. There is no `Connect(ctx)`/`Start(ctx)` — a `Verifier` is ready to use as soon
as `New` returns, and the network call happens inside `playintegrity.VerifyAndroid` on each
invocation. Instances are stateless apart from immutable config, so one verifier is safe to reuse
concurrently (the playintegrity service is rebuilt per call; the appattest verifier holds only an
`appID` string and a parsed `*x509.Certificate`).

# Dependencies

- Internal `pkg/` imports: **none**. `pkg/attestation/attestation.go` imports only stdlib
  (`context`, `errors`, `fmt`); it is a stdlib-only leaf, unlike `middleware → utils` or `sms →
  logging`. The three driver subpackages import only their parent `pkg/attestation` plus external
  deps — they never reach into another `pkg/` package.
- External (driver subpackages only): `google.golang.org/api/option` +
  `google.golang.org/api/playintegrity/v1` (playintegrity); `github.com/fxamacker/cbor/v2`
  (appattest, the CBOR codec).
- No config struct, environment variable, or secret read by this package itself — `Config.Options`
  is the only input channel, and credentials (the Play service-account JSON) arrive as an option
  string from the composition root.

# Consumers

- `services/appService/cmd/main.go` — imports all three drivers, builds two `attestation.Verifier`s
  (`attestationAndroid`, `attestationIOS`) via `attestation.New`, and hands them to
  `repository.NewAttestationVerifier`. In dev both select `"mock"`.
- `services/appService/repository/attestation_verifier.go` — the adapter that implements
  `domain.AttestationVerifier` over two `pkg/attestation.Verifier`s, converting `AndroidResult`/
  `IOSAttestationResult` into `domain.AndroidVerdict`/`domain.IOSAttestationResult` and mapping
  `ErrCounterNotIncreased`/`ErrAssertionInvalid` onto the domain sentinels. This is the only place
  that translates the generic shapes into `services/appService` domain types.
- The real `playintegrity`/`appattest` drivers have **no live consumer yet** — the service selects
  `mock` for both platforms. Their only exercise today is their own unit tests.

# Verification

- `go test -count=1 ./pkg/attestation/...` — passes:
  - `attestation_test.go`: registry behaviour (nil config, missing provider, unregistered provider,
    registered factory is used, factory error propagates).
  - `mock/mock_test.go`: pass/fail/upstream-error android, ios attest pass/fail, assertion
    pass/replay/invalid.
  - `playintegrity/playintegrity_test.go`: registration, `package_name` required, iOS methods
    `ErrPlatformNotSupported`, and `mapPayload` (full payload, nil sub-structs, missing fields).
  - `appattest/appattest_test.go`: builds a synthetic cert chain + CBOR fixtures and exercises
    `VerifyIOSAttestation` (success, development environment, wrong fmt / counter≠0 / key-id
    mismatch / wrong app-id) and `VerifyIOSAssertion` (success returns the counter,
    counter-not-increased, tampered client data, wrong app-id), plus `New` validation.
- The `playintegrity` driver's actual Google decode (`VerifyAndroid`'s `buildService` +
  `DecodeIntegrityToken` call) is **not** exercised by tests — only the pure `mapPayload` is, which
  is why it is deliberately separated out. That call needs a real service account.

# Pending work

- The `playintegrity` driver's server-to-server decode is unverified: no test reaches Google, and no
  deployment has selected it. That untested network path is the main reason this concept is `draft`
  rather than `stable` — the package as a whole is not a pure library (the playintegrity half does
  network I/O), so "its own test suite passes" is not sufficient evidence of working behaviour.
- The real drivers are not consumed end-to-end: `services/appService` selects `mock` for both
  platforms. Promote to `stable` once `attestation.*.provider` points at `playintegrity`/`appattest`
  and the verify flow is exercised against a real backend.
- `pkg/attestation/README.md` does not exist, though the repo convention calls for one on any
  `pkg/` package handling non-trivial logic.
- The embedded Apple App Attest root CA (`appleRootPEM`) is hardcoded; it will eventually need
  rotation, which is why `root_cert_pem` exists as an override — but no process documents when/how.

# Related concepts

- [features/attestation.md](../features/attestation.md) — the feature that consumes this package:
  `AttestationUsecase` depends on the `domain.AttestationVerifier` port, whose concrete
  implementation is `repository/attestation_verifier.go` over this package's two `Verifier`s.
- No `Package` concept exists yet for `pkg/sms` — the package whose `Register`/`New` registry pattern
  this one copies (per `docs/specs/features/attestation/design.md`). Not yet documented as an OKF
  concept in this bundle.
- No `Architecture Pattern` concept exists yet for the provider-agnostic
  "interface + `Register(type, FactoryFunc)` + driver `init()`" pattern that this package, `pkg/sms`,
  `pkg/database` and `pkg/storage` all follow. Not yet documented as an OKF concept in this bundle.
