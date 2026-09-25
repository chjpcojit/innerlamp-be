package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// The mock must satisfy the port, and this line is what makes that true at
// compile time rather than at first use in a test.
var _ AttestationVerifier = (*MockAttestationVerifier)(nil)

// MockAttestationVerifier is a mock implementation of AttestationVerifier for
// testing. Hand-written, in the same package as the interface (the repo has no
// mockgen/mockery codegen step).
type MockAttestationVerifier struct {
	mock.Mock
}

// VerifyAndroid mocks the VerifyAndroid method.
func (m *MockAttestationVerifier) VerifyAndroid(ctx context.Context, token, challenge string) (*AndroidVerdict, error) {
	args := m.Called(ctx, token, challenge)
	if v := args.Get(0); v != nil {
		return v.(*AndroidVerdict), args.Error(1)
	}
	return nil, args.Error(1)
}

// VerifyIOSAttestation mocks the VerifyIOSAttestation method.
func (m *MockAttestationVerifier) VerifyIOSAttestation(ctx context.Context, keyID string, attestation []byte, challenge string) (*IOSAttestationResult, error) {
	args := m.Called(ctx, keyID, attestation, challenge)
	if r := args.Get(0); r != nil {
		return r.(*IOSAttestationResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// VerifyIOSAssertion mocks the VerifyIOSAssertion method.
func (m *MockAttestationVerifier) VerifyIOSAssertion(ctx context.Context, publicKey []byte, assertion, clientData []byte, storedCounter int64) (int64, error) {
	args := m.Called(ctx, publicKey, assertion, clientData, storedCounter)
	if c := args.Get(0); c != nil {
		return c.(int64), args.Error(1)
	}
	return 0, args.Error(1)
}
