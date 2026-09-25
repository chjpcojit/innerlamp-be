package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/attestation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// stubPkgVerifier is a hand-written pkg/attestation.Verifier whose returns are
// configured per test, so the adapter can be exercised in isolation.
type stubPkgVerifier struct {
	android        *attestation.AndroidResult
	androidErr     error
	iosAttest      *attestation.IOSAttestationResult
	iosAttestErr   error
	iosAssertCount int64
	iosAssertErr   error
}

func (s *stubPkgVerifier) VerifyAndroid(_ context.Context, _, _ string) (*attestation.AndroidResult, error) {
	return s.android, s.androidErr
}

func (s *stubPkgVerifier) VerifyIOSAttestation(_ context.Context, _ string, _ []byte, _ string) (*attestation.IOSAttestationResult, error) {
	return s.iosAttest, s.iosAttestErr
}

func (s *stubPkgVerifier) VerifyIOSAssertion(_ context.Context, _ []byte, _, _ []byte, _ int64) (int64, error) {
	return s.iosAssertCount, s.iosAssertErr
}

// newAdapter wires a domain.AttestationVerifier over the given android/ios
// stubs (defaulting to a stub that errors, so a test only stubs what it needs).
func newAdapter(android, ios *stubPkgVerifier) domain.AttestationVerifier {
	return NewAttestationVerifier(android, ios)
}

// errAndroidUnreachable and errIOSInvalid stand in for the two stubs' distinct
// failures, so a routing test can tell which verifier was consulted.
var (
	errAndroidUnreachable = errors.New("android unreachable")
	errIOSInvalid         = errors.New("ios invalid")
)

func TestAttestationVerifier_RoutesByPlatform(t *testing.T) {
	// A call on the iOS side must hit the ios verifier, not the android one, and
	// vice versa. Each stub records nothing; we assert by giving the two stubs
	// different results and checking which one comes back.
	androidStub := &stubPkgVerifier{androidErr: errAndroidUnreachable}
	iosStub := &stubPkgVerifier{iosAttestErr: errIOSInvalid}

	adapter := newAdapter(androidStub, iosStub)

	_, err := adapter.VerifyAndroid(context.Background(), "token", "nonce")
	require.ErrorIs(t, err, errAndroidUnreachable)

	_, err = adapter.VerifyIOSAttestation(context.Background(), "key", []byte("obj"), "nonce")
	require.ErrorIs(t, err, errIOSInvalid)
}

func TestAttestationVerifier_VerifyAndroid_ConvertsShape(t *testing.T) {
	android := &stubPkgVerifier{android: &attestation.AndroidResult{
		PackageName:              "com.example.app",
		Nonce:                    "nonce",
		TimestampMillis:          1234,
		AppRecognitionVerdict:    "PLAY_RECOGNIZED",
		DeviceRecognitionVerdict: "MEETS_DEVICE_INTEGRITY",
		AppLicensingVerdict:      "LICENSED",
	}}
	adapter := newAdapter(android, &stubPkgVerifier{})

	result, err := adapter.VerifyAndroid(context.Background(), "token", "nonce")
	require.NoError(t, err)
	require.Equal(t, &domain.AndroidVerdict{
		PackageName:              "com.example.app",
		Nonce:                    "nonce",
		TimestampMillis:          1234,
		AppRecognitionVerdict:    "PLAY_RECOGNIZED",
		DeviceRecognitionVerdict: "MEETS_DEVICE_INTEGRITY",
		AppLicensingVerdict:      "LICENSED",
	}, result)
}

func TestAttestationVerifier_VerifyAndroid_PassesError(t *testing.T) {
	wantErr := errors.New("upstream down")
	adapter := newAdapter(&stubPkgVerifier{androidErr: wantErr}, &stubPkgVerifier{})

	_, err := adapter.VerifyAndroid(context.Background(), "token", "nonce")
	require.ErrorIs(t, err, wantErr)
}

func TestAttestationVerifier_VerifyIOSAttestation_ConvertsShape(t *testing.T) {
	ios := &stubPkgVerifier{iosAttest: &attestation.IOSAttestationResult{
		PublicKey:   []byte("pk"),
		Receipt:     []byte("rcpt"),
		Environment: "production",
	}}
	adapter := newAdapter(&stubPkgVerifier{}, ios)

	result, err := adapter.VerifyIOSAttestation(context.Background(), "key", []byte("obj"), "nonce")
	require.NoError(t, err)
	require.Equal(t, []byte("pk"), result.PublicKey)
	require.Equal(t, []byte("rcpt"), result.Receipt)
	require.Equal(t, "production", result.Environment)
}

func TestAttestationVerifier_VerifyIOSAssertion_MapsErrors(t *testing.T) {
	t.Run("counter not increased", func(t *testing.T) {
		adapter := newAdapter(&stubPkgVerifier{}, &stubPkgVerifier{iosAssertErr: attestation.ErrCounterNotIncreased})
		_, err := adapter.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 5)
		require.ErrorIs(t, err, domain.ErrAttestationCounterNotIncreased)
	})

	t.Run("assertion invalid", func(t *testing.T) {
		adapter := newAdapter(&stubPkgVerifier{}, &stubPkgVerifier{iosAssertErr: attestation.ErrAssertionInvalid})
		_, err := adapter.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 5)
		require.ErrorIs(t, err, domain.ErrAttestationAssertionInvalid)
	})

	t.Run("success returns the new counter", func(t *testing.T) {
		adapter := newAdapter(&stubPkgVerifier{}, &stubPkgVerifier{iosAssertCount: 7})
		counter, err := adapter.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 5)
		require.NoError(t, err)
		require.Equal(t, int64(7), counter)
	})
}
