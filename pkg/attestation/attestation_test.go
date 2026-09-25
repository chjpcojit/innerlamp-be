package attestation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubVerifier struct{}

func (stubVerifier) VerifyAndroid(ctx context.Context, token, challenge string) (*AndroidResult, error) {
	return &AndroidResult{}, nil
}

func (stubVerifier) VerifyIOSAttestation(ctx context.Context, keyID string, attestation []byte, challenge string) (*IOSAttestationResult, error) {
	return &IOSAttestationResult{}, nil
}

func (stubVerifier) VerifyIOSAssertion(ctx context.Context, publicKey []byte, assertion, clientData []byte, storedCounter int64) (int64, error) {
	return storedCounter + 1, nil
}

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	require.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNew_MissingProvider(t *testing.T) {
	_, err := New(&Config{})
	require.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNew_UnregisteredProvider(t *testing.T) {
	_, err := New(&Config{Provider: ProviderType("does-not-exist")})
	require.ErrorIs(t, err, ErrUnsupportedProvider)
}

func TestNew_UsesRegisteredFactory(t *testing.T) {
	const testProvider ProviderType = "test-stub"
	Register(testProvider, func(cfg *Config) (Verifier, error) {
		return stubVerifier{}, nil
	})

	v, err := New(&Config{Provider: testProvider})
	require.NoError(t, err)
	result, err := v.VerifyAndroid(context.Background(), "token", "challenge")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestNew_FactoryErrorPropagates(t *testing.T) {
	const testProvider ProviderType = "test-stub-failing"
	wantErr := errors.New("boom")
	Register(testProvider, func(cfg *Config) (Verifier, error) {
		return nil, wantErr
	})

	_, err := New(&Config{Provider: testProvider})
	require.ErrorIs(t, err, wantErr)
}
