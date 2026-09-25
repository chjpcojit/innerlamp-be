package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// nonceSize is how many random bytes each minted nonce carries (AC1 requires at
// least 16). 32 bytes → 44 base64 characters, comfortably inside the
// attestation_challenges.challenge VARCHAR(255).
const nonceSize = 32

// AttestationUsecase implements the attestation feature: minting single-use
// challenges, verifying Play Integrity (Android) and App Attest (iOS)
// attestation evidence and returning a Verdict, and verifying per-request iOS
// assertions for the RequireAttestation middleware.
//
// It owns the challenge/verify policy (config.AttestationConfig), the
// persistence port (domain.AttestationRepository) and the verification port
// (domain.AttestationVerifier). It never touches the DB or the crypto directly
// — the concrete Postgres repository and the pkg/attestation backend are
// injected.
type AttestationUsecase struct {
	repo     domain.AttestationRepository
	verifier domain.AttestationVerifier
	logger   logging.Logger
	config   config.AttestationConfig
}

// NewAttestationUsecase wires the use case. Every collaborator is a port or a
// plain config value; no concrete repository or verification backend is visible
// here.
func NewAttestationUsecase(
	repo domain.AttestationRepository,
	verifier domain.AttestationVerifier,
	logger logging.Logger,
	cfg config.AttestationConfig,
) *AttestationUsecase {
	return &AttestationUsecase{
		repo:     repo,
		verifier: verifier,
		logger:   logger,
		config:   cfg,
	}
}

// RequestChallenge mints a fresh nonce and persists it for platform/purpose —
// AC1 (attestation) and AC3 (assertion). The nonce is cryptographically random
// (crypto/rand, never math/rand — a guessable nonce lets an attacker forge a
// verdict) and expires after config.ChallengeTTL.
func (u *AttestationUsecase) RequestChallenge(ctx context.Context, platform domain.Platform, purpose domain.ChallengePurpose) (*domain.RequestChallengeResult, error) {
	nonce, err := generateNonce()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(u.config.ChallengeTTL.Duration())
	if err := u.repo.SaveChallenge(ctx, nonce, platform, purpose, expiresAt); err != nil {
		return nil, fmt.Errorf("save challenge: %w", err)
	}

	return &domain.RequestChallengeResult{
		Challenge:        nonce,
		ExpiresInSeconds: int(u.config.ChallengeTTL.Duration().Seconds()),
	}, nil
}

// Verify runs the two-branch verify flow (design.md flow 3/4). The platform
// selects which branch; delivery has already validated it to android|ios.
//
// A negative verdict (Attested == false) is NOT an error — it is returned as a
// Verdict with a Reason, and the challenge is still consumed. Only a challenge
// state problem (ErrAttestationChallengeInvalid → 400) and an upstream failure
// (ErrAttestationUpstreamError → 502) are errors, and the latter leaves the
// challenge unconsumed so the caller may retry (AC7).
func (u *AttestationUsecase) Verify(ctx context.Context, input domain.VerifyInput) (*domain.Verdict, error) {
	switch input.Platform {
	case domain.PlatformAndroid:
		return u.verifyAndroid(ctx, input)
	case domain.PlatformIOS:
		return u.verifyIOS(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported attestation platform %q", input.Platform)
	}
}

// verifyAndroid decodes the integrity token on Google's servers, compares the
// decoded identity against config and the challenge, and consumes the challenge
// regardless of the verdict (design.md flow 3).
func (u *AttestationUsecase) verifyAndroid(ctx context.Context, input domain.VerifyInput) (*domain.Verdict, error) {
	// Step 2: the challenge must be known, unused and unexpired. A bad one is
	// 400 and never reaches Google.
	if _, err := u.repo.FindChallenge(ctx, input.Challenge, domain.ChallengePurposeAttest); err != nil {
		return nil, err
	}

	// Step 3: server-to-server decode. An error here is an upstream failure —
	// surfaced as ErrAttestationUpstreamError and the challenge is left
	// unconsumed (AC7).
	result, err := u.verifier.VerifyAndroid(ctx, input.Token, input.Challenge)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAttestationUpstreamError, err)
	}

	// Step 4: compare the decoded identity, then close the challenge. Closing
	// happens first because a race-losing mark (0 rows) is a challenge-state
	// error regardless of the verdict — the nonce is one-time.
	reason := u.androidVerdictReason(result, input.Challenge)

	marked, err := u.repo.MarkChallengeUsed(ctx, input.Challenge)
	if err != nil {
		return nil, fmt.Errorf("mark challenge used: %w", err)
	}
	if !marked {
		return nil, domain.ErrAttestationChallengeInvalid
	}

	verdict := &domain.Verdict{
		Platform: domain.PlatformAndroid,
		Attested: reason == "",
		Reason:   reason,
	}
	if reason == "" {
		verdict.Signals = map[string]string{
			"app_recognition_verdict":    result.AppRecognitionVerdict,
			"device_recognition_verdict": result.DeviceRecognitionVerdict,
			"app_licensing_verdict":      result.AppLicensingVerdict,
		}
	}
	return verdict, nil
}

// androidVerdictReason compares the decoded token against config and the
// challenge, returning "" when every check passes and a Reason otherwise. The
// verdict threshold (appRecognitionVerdict) is config-driven (design.md
// decision #11), the identity checks (package/nonce/freshness) are fixed.
func (u *AttestationUsecase) androidVerdictReason(result *domain.AndroidVerdict, challenge string) string {
	if result.PackageName != u.config.Android.PackageName {
		return domain.ReasonAttestationInvalid
	}
	if result.Nonce != challenge {
		return domain.ReasonAttestationInvalid
	}
	if !u.isFresh(result.TimestampMillis, time.Now()) {
		return domain.ReasonAttestationInvalid
	}
	if result.AppRecognitionVerdict != u.config.Android.RequiredAppRecognitionVerdict {
		return domain.ReasonAppNotRecognized
	}
	return ""
}

// isFresh reports whether the token's timestamp is no older than the freshness
// window, and not in the future. A token older than the window is stale; a
// token claiming the future is treated as invalid too.
func (u *AttestationUsecase) isFresh(timestampMillis int64, now time.Time) bool {
	age := now.Sub(time.UnixMilli(timestampMillis))
	return age >= 0 && age <= u.config.FreshnessWindow.Duration()
}

// verifyIOS runs the App Attest `attest` (key registration) phase (design.md
// flow 4): verify the attestation object locally, and on success upsert the key
// and close the challenge atomically. A crypto failure is a negative Verdict,
// not an error (AC8/AC9).
func (u *AttestationUsecase) verifyIOS(ctx context.Context, input domain.VerifyInput) (*domain.Verdict, error) {
	if _, err := u.repo.FindChallenge(ctx, input.Challenge, domain.ChallengePurposeAttest); err != nil {
		return nil, err
	}

	result, err := u.verifier.VerifyIOSAttestation(ctx, input.KeyID, input.Attestation, input.Challenge)
	if err != nil {
		// The object failed crypto verification: attested:false, the challenge
		// is still consumed (AC8), and no key is persisted.
		marked, markErr := u.repo.MarkChallengeUsed(ctx, input.Challenge)
		if markErr != nil {
			return nil, fmt.Errorf("mark challenge used: %w", markErr)
		}
		if !marked {
			return nil, domain.ErrAttestationChallengeInvalid
		}
		return &domain.Verdict{
			Platform: domain.PlatformIOS,
			Attested: false,
			Reason:   domain.ReasonAttestationInvalid,
		}, nil
	}

	key := &domain.AttestationKey{
		KeyID:       input.KeyID,
		Platform:    domain.PlatformIOS,
		PublicKey:   result.PublicKey,
		Receipt:     result.Receipt,
		SignCounter: 0,
		Environment: result.Environment,
	}
	// SaveKey upserts by key_id AND consumes the challenge in one transaction
	// (AC8/AC10).
	if err := u.repo.SaveKey(ctx, key, input.Challenge); err != nil {
		return nil, fmt.Errorf("save key: %w", err)
	}

	return &domain.Verdict{
		Platform: domain.PlatformIOS,
		Attested: true,
		Signals:  map[string]string{"environment": result.Environment},
	}, nil
}

// VerifyAssertion verifies a per-request iOS assertion for the
// RequireAttestation middleware (design.md assertion flow). It looks up the
// challenge and key, verifies the assertion, and atomically advances the
// counter and closes the challenge. Errors are the assertion sentinels, which
// the middleware maps to 401.
func (u *AttestationUsecase) VerifyAssertion(ctx context.Context, input domain.VerifyAssertionInput) error {
	if _, err := u.repo.FindChallenge(ctx, input.Challenge, domain.ChallengePurposeAssert); err != nil {
		return err
	}

	key, err := u.repo.FindKey(ctx, input.KeyID)
	if err != nil {
		return err
	}

	newCounter, err := u.verifier.VerifyIOSAssertion(ctx, key.PublicKey, input.Assertion, input.ClientData, key.SignCounter)
	if err != nil {
		return err
	}

	if err := u.repo.UpdateKeyCounter(ctx, input.KeyID, newCounter, input.Challenge); err != nil {
		return fmt.Errorf("update key counter: %w", err)
	}

	return nil
}

// generateNonce returns a base64 encoding of nonceSize cryptographically-random
// bytes. crypto/rand, not math/rand: a guessable nonce would let an attacker
// forge a verdict (design.md challenge flow Step 2).
func generateNonce() (string, error) {
	buf := make([]byte, nonceSize)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}
