package domain

import (
	"context"
	"time"
)

// Platform discriminates which attestation backend a challenge or verify
// targets. It maps the attestation_challenges.platform column, whose CHECK
// constraint accepts exactly these two values.
//
// The two platforms are deliberately asymmetric (design.md §Overview): Android
// has no durable key — every verify is a fresh integrity token bound to a
// challenge — while iOS has a per-device durable key (key_id) registered once
// via the `attest` phase and then used for many `assert` operations.
type Platform string

const (
	PlatformAndroid Platform = "android"
	PlatformIOS     Platform = "ios"
)

// ChallengePurpose discriminates what a nonce is minted for. It maps the
// attestation_challenges.purpose column, whose CHECK constraint accepts exactly
// these two values.
//
// A nonce minted for `attest` (the verify endpoint) must not be consumable by
// the assertion middleware, and vice versa, so FindChallenge always filters on
// purpose.
type ChallengePurpose string

const (
	ChallengePurposeAttest ChallengePurpose = "attest"
	ChallengePurposeAssert ChallengePurpose = "assert"
)

// AttestationChallenge is one minted nonce. It maps the attestation_challenges
// table. The nonce is a base64 encoding of ≥16 cryptographically-random bytes,
// is single-use (UsedAt set on first consume), and expires after the configured
// challenge TTL.
//
// Plain data: no json/db tag — the wire shape belongs to delivery's response
// structs and the DB shape to repository's SQL.
type AttestationChallenge struct {
	ID        int64
	Challenge string
	Platform  Platform
	Purpose   ChallengePurpose
	UsedAt    *time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

// AttestationKey is one registered iOS App Attest key. It maps the
// attestation_keys table. Keys only exist for iOS — Android has no durable key,
// so there is no android key row (design.md §Schema).
//
// KeyID is the base64 credentialId the client presents (it is the hash of the
// public key), PublicKey is the COSE-encoded ES256 public key used to verify
// later assertions, Receipt is stored for the (deferred) fraud-risk lookup, and
// SignCounter is the largest authenticator counter seen so far — the value that
// makes replay/rollback detectable (AC11/AC12).
type AttestationKey struct {
	ID          int64
	KeyID       string
	Platform    Platform
	PublicKey   []byte
	Receipt     []byte
	SignCounter int64
	Environment string
	UserID      *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Verdict is the outcome of a verify. It is deliberately NOT an error: a
// negative verdict (Attested == false) is a 200 response with a reason, because
// the caller is allowed to learn "your app is not recognized" but must not be
// able to distinguish a probeable failure (design.md decision #10).
//
// Signals carries platform-specific extra information on success: for Android
// the Play Integrity verdicts, for iOS the App Attest environment.
type Verdict struct {
	Attested bool
	Platform Platform
	Signals  map[string]string
	Reason   string
}

// Reason values carried by a negative Verdict (or, for the assertion path, by a
// 401 body). They are domain constants rather than free-form strings so the
// client-facing contract has one definition. The assertion middleware's reasons
// (unknown_key, counter_not_increased, assertion_invalid) come from the
// sentinel errors below and are mapped in delivery.
const (
	// ReasonAppNotRecognized — Play Integrity decoded the token but the app's
	// recognition verdict did not meet the configured threshold (AC5).
	ReasonAppNotRecognized = "app_not_recognized"

	// ReasonAttestationInvalid — the decoded token/attestation failed one of
	// the identity checks (package name, nonce, freshness) or the iOS
	// attestation object failed crypto verification (AC9).
	ReasonAttestationInvalid = "attestation_invalid"

	// ReasonUnknownKey — the assertion named a key_id the server has no
	// registration for (AC13).
	ReasonUnknownKey = "unknown_key"

	// ReasonCounterNotIncreased — the assertion's authenticator counter was not
	// strictly greater than the stored one (replay/rollback, AC12).
	ReasonCounterNotIncreased = "counter_not_increased"

	// ReasonAssertionInvalid — the assertion signature or RP ID did not verify
	// against the stored public key (design.md assertion flow Step 3).
	ReasonAssertionInvalid = "assertion_invalid"

	// ReasonMissingEvidence — a protected request arrived without one of the
	// three attestation headers (design.md assertion flow Step 1).
	ReasonMissingEvidence = "missing_evidence"
)

// AndroidVerdict is what a Play Integrity decode returns: the identity fields
// the use case compares against config and the challenge (package name, nonce,
// timestamp) plus the three verdicts Google reports.
type AndroidVerdict struct {
	PackageName              string
	Nonce                    string
	TimestampMillis          int64
	AppRecognitionVerdict    string
	DeviceRecognitionVerdict string
	AppLicensingVerdict      string
}

// IOSAttestationResult is what a successful iOS attestation-object verification
// returns: the public key to store for later assertions, the receipt to keep
// for the (deferred) fraud-risk lookup, and the environment derived from the
// attestation's aaguid.
type IOSAttestationResult struct {
	PublicKey   []byte
	Receipt     []byte
	Environment string
}

// AttestationVerifier is the seam to the attestation backend (pkg/attestation).
// The use case depends on this port rather than importing pkg/attestation
// directly, so domain stays free of concrete infrastructure and tests can
// assert a verdict without touching Google or Apple.
//
// The asymmetry of the two platforms is visible in the split:
//   - VerifyAndroid is server-to-server: Google decodes the integrity token.
//   - VerifyIOSAttestation and VerifyIOSAssertion are local crypto: the server
//     verifies the CBOR attestation object / the ES256 assertion itself and
//     never calls Apple.
type AttestationVerifier interface {
	// VerifyAndroid decodes the Play Integrity token for token against the
	// challenge nonce. An error means the decode could not be performed
	// (unreachable, timeout) — the use case surfaces it as
	// ErrAttestationUpstreamError and does not consume the challenge (AC7).
	VerifyAndroid(ctx context.Context, token, challenge string) (*AndroidVerdict, error)

	// VerifyIOSAttestation verifies an App Attest attestation object (the
	// `attest` phase). An error means the object failed crypto verification —
	// the use case surfaces it as a negative Verdict, not an HTTP error (AC9).
	VerifyIOSAttestation(ctx context.Context, keyID string, attestation []byte, challenge string) (*IOSAttestationResult, error)

	// VerifyIOSAssertion verifies an App Attest assertion over clientData
	// against publicKey (the stored COSE ES256 key), and returns the
	// authenticator counter it read. It returns ErrAttestationCounterNotIncreased
	// when counter <= storedCounter and ErrAttestationAssertionInvalid when the
	// signature/RP ID does not verify. publicKey is the raw key the repository
	// stored at registration — the keyID alone is only a hash of it and cannot
	// verify a signature.
	VerifyIOSAssertion(ctx context.Context, publicKey []byte, assertion, clientData []byte, storedCounter int64) (int64, error)
}

// AttestationRepository persists the attestation flow's own state: the single-
// use challenge nonces and (for iOS) the registered keys.
//
// As in US-001, every write that must not be observed half-done is one method
// wrapping one transaction. SaveKey and UpdateKeyCounter both take the
// challenge to consume, so "register a key" / "advance the counter" and "close
// the challenge" are atomic — otherwise a crash between two calls would leave a
// key registered against a still-unused challenge, which a caller could replay
// to mint a second key (design.md iOS flow Step 4 / assertion flow Step 4).
type AttestationRepository interface {
	// SaveChallenge mints a row for a fresh nonce, expiring at expiresAt —
	// AC1/AC3. expiresAt is computed by the use case from config.ChallengeTTL,
	// never defaulted in SQL, so the stored expiry and the value reported to
	// the client are the same number by construction.
	SaveChallenge(ctx context.Context, challenge string, platform Platform, purpose ChallengePurpose, expiresAt time.Time) error

	// FindChallenge loads the row for challenge with the given purpose,
	// requiring it to be unused and unexpired. Returns
	// ErrAttestationChallengeInvalid for a row that is unknown, expired, or
	// already used — one sentinel for all three so the caller cannot probe
	// which state (AC6/AC14).
	FindChallenge(ctx context.Context, challenge string, purpose ChallengePurpose) (*AttestationChallenge, error)

	// MarkChallengeUsed consumes challenge with a single UPDATE ... WHERE
	// used_at IS NULL. The bool reports whether a row was actually marked
	// (false means a concurrent request won the race), which the use case
	// maps to ErrAttestationChallengeInvalid (AC6/AC7).
	MarkChallengeUsed(ctx context.Context, challenge string) (bool, error)

	// SaveKey upserts the iOS key by key_id and marks challenge used, in one
	// transaction — AC8/AC10 (idempotent re-registration).
	SaveKey(ctx context.Context, key *AttestationKey, challenge string) error

	// FindKey loads the registered key for keyID. Returns
	// ErrAttestationKeyNotFound when no row exists (AC13).
	FindKey(ctx context.Context, keyID string) (*AttestationKey, error)

	// UpdateKeyCounter advances the key's stored counter to newCounter and
	// marks challenge used, in one transaction — AC11/AC12.
	UpdateKeyCounter(ctx context.Context, keyID string, newCounter int64, challenge string) error
}

// RequestChallengeResult reports the minted nonce and how long it lives, so the
// handler can put both in the 200 body without hardcoding the TTL.
type RequestChallengeResult struct {
	Challenge        string
	ExpiresInSeconds int
}

// VerifyInput is the use-case input for POST /v1/attestation/verify. The
// platform selects which of the two branches is meaningful: Token for android,
// KeyID + Attestation for ios. Attestation is already base64-decoded by the
// delivery layer, which owns the wire format.
type VerifyInput struct {
	Platform    Platform
	Challenge   string
	Token       string
	KeyID       string
	Attestation []byte
}

// VerifyAssertionInput is the use-case input for the assertion middleware. The
// middleware builds ClientData from the real request (challenge || method ||
// path || SHA256(body)) and passes the raw assertion bytes; the use case looks
// up the challenge/key and verifies.
type VerifyAssertionInput struct {
	KeyID      string
	Assertion  []byte
	ClientData []byte
	Challenge  string
}
