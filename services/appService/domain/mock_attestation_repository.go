package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// The mock must satisfy the port, and this line is what makes that true at
// compile time rather than at first use in a test. Without it, a method added
// to AttestationRepository leaves the mock silently short until some test
// happens to exercise the new method — see the "when a port's interface
// changes" note in the package README.
var _ AttestationRepository = (*MockAttestationRepository)(nil)

// MockAttestationRepository is a mock implementation of AttestationRepository
// for testing. Hand-written, in the same package as the interface (the repo has
// no mockgen/mockery codegen step).
type MockAttestationRepository struct {
	mock.Mock
}

// SaveChallenge mocks the SaveChallenge method.
func (m *MockAttestationRepository) SaveChallenge(ctx context.Context, challenge string, platform Platform, purpose ChallengePurpose, expiresAt time.Time) error {
	return m.Called(ctx, challenge, platform, purpose, expiresAt).Error(0)
}

// FindChallenge mocks the FindChallenge method.
func (m *MockAttestationRepository) FindChallenge(ctx context.Context, challenge string, purpose ChallengePurpose) (*AttestationChallenge, error) {
	args := m.Called(ctx, challenge, purpose)
	if c := args.Get(0); c != nil {
		return c.(*AttestationChallenge), args.Error(1)
	}
	return nil, args.Error(1)
}

// MarkChallengeUsed mocks the MarkChallengeUsed method.
func (m *MockAttestationRepository) MarkChallengeUsed(ctx context.Context, challenge string) (bool, error) {
	args := m.Called(ctx, challenge)
	return args.Bool(0), args.Error(1)
}

// SaveKey mocks the SaveKey method.
func (m *MockAttestationRepository) SaveKey(ctx context.Context, key *AttestationKey, challenge string) error {
	return m.Called(ctx, key, challenge).Error(0)
}

// FindKey mocks the FindKey method.
func (m *MockAttestationRepository) FindKey(ctx context.Context, keyID string) (*AttestationKey, error) {
	args := m.Called(ctx, keyID)
	if k := args.Get(0); k != nil {
		return k.(*AttestationKey), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateKeyCounter mocks the UpdateKeyCounter method.
func (m *MockAttestationRepository) UpdateKeyCounter(ctx context.Context, keyID string, newCounter int64, challenge string) error {
	return m.Called(ctx, keyID, newCounter, challenge).Error(0)
}
