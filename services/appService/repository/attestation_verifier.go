package repository

import (
	"context"
	"errors"

	"chjpcoj.com.innerlamp/pkg/attestation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// attestationVerifier implements domain.AttestationVerifier by routing to two
// platform-specific pkg/attestation.Verifier instances: one for Android
// (playintegrity or mock) and one for iOS (appattest or mock).
//
// pkg/attestation is a generic "verify platform evidence" package with no
// knowledge of this service's domain types — that is what keeps it reusable.
// Turning its generic result shapes and provider errors into the domain types
// and sentinels the use case consumes is a service-level concern, so it lives
// here, next to every other domain-port implementation this package holds.
type attestationVerifier struct {
	android attestation.Verifier
	ios     attestation.Verifier
}

// NewAttestationVerifier returns the domain.AttestationVerifier backed by two
// platform verifiers. android answers VerifyAndroid; ios answers the two App
// Attest methods.
func NewAttestationVerifier(android, ios attestation.Verifier) domain.AttestationVerifier {
	return &attestationVerifier{android: android, ios: ios}
}

// VerifyAndroid converts the generic decode result into a domain.AndroidVerdict.
// An error passes through unchanged — the use case wraps it as
// ErrAttestationUpstreamError.
func (a *attestationVerifier) VerifyAndroid(ctx context.Context, token, challenge string) (*domain.AndroidVerdict, error) {
	result, err := a.android.VerifyAndroid(ctx, token, challenge)
	if err != nil {
		return nil, err
	}
	return &domain.AndroidVerdict{
		PackageName:              result.PackageName,
		Nonce:                    result.Nonce,
		TimestampMillis:          result.TimestampMillis,
		AppRecognitionVerdict:    result.AppRecognitionVerdict,
		DeviceRecognitionVerdict: result.DeviceRecognitionVerdict,
		AppLicensingVerdict:      result.AppLicensingVerdict,
	}, nil
}

// VerifyIOSAttestation converts the generic result into a
// domain.IOSAttestationResult. An error passes through unchanged — the use case
// treats any error as a negative Verdict.
func (a *attestationVerifier) VerifyIOSAttestation(ctx context.Context, keyID string, attestationObj []byte, challenge string) (*domain.IOSAttestationResult, error) {
	result, err := a.ios.VerifyIOSAttestation(ctx, keyID, attestationObj, challenge)
	if err != nil {
		return nil, err
	}
	return &domain.IOSAttestationResult{
		PublicKey:   result.PublicKey,
		Receipt:     result.Receipt,
		Environment: result.Environment,
	}, nil
}

// VerifyIOSAssertion maps the provider's two assertion errors onto the domain
// sentinels the middleware distinguishes, and passes the new counter through.
func (a *attestationVerifier) VerifyIOSAssertion(ctx context.Context, publicKey []byte, assertion, clientData []byte, storedCounter int64) (int64, error) {
	newCounter, err := a.ios.VerifyIOSAssertion(ctx, publicKey, assertion, clientData, storedCounter)
	switch {
	case err == nil:
		return newCounter, nil
	case errors.Is(err, attestation.ErrCounterNotIncreased):
		return 0, domain.ErrAttestationCounterNotIncreased
	default:
		// Includes attestation.ErrAssertionInvalid and any other failure.
		return 0, domain.ErrAttestationAssertionInvalid
	}
}
