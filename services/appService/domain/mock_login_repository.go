package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// The mock must satisfy the port, and this line is what makes that true at
// compile time rather than at first use in a test. Without it, a method added
// to LoginRepository leaves the mock silently short until some test happens to
// exercise the new method — see the "when a port's interface changes" note in
// the package README.
var _ LoginRepository = (*MockLoginRepository)(nil)

// MockLoginRepository is a mock implementation of LoginRepository for testing.
// Hand-written, in the same package as the interface (the repo has no
// mockgen/mockery codegen step): see mock_registration_repository.go for the
// same pattern.
type MockLoginRepository struct {
	mock.Mock
}

// FindUserByPhone mocks the FindUserByPhone method.
func (m *MockLoginRepository) FindUserByPhone(ctx context.Context, phoneNumber string) (*User, error) {
	args := m.Called(ctx, phoneNumber)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindUserByID mocks the FindUserByID method.
func (m *MockLoginRepository) FindUserByID(ctx context.Context, userID int64) (*User, error) {
	args := m.Called(ctx, userID)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindLoginAttempt mocks the FindLoginAttempt method.
func (m *MockLoginRepository) FindLoginAttempt(ctx context.Context, phoneNumber string) (*LoginAttempt, error) {
	args := m.Called(ctx, phoneNumber)
	if a := args.Get(0); a != nil {
		return a.(*LoginAttempt), args.Error(1)
	}
	return nil, args.Error(1)
}

// RecordLoginFailure mocks the RecordLoginFailure method.
func (m *MockLoginRepository) RecordLoginFailure(ctx context.Context, phoneNumber string, maxAttempts int, lockDuration, resetWindow time.Duration) (*LoginAttempt, error) {
	args := m.Called(ctx, phoneNumber, maxAttempts, lockDuration, resetWindow)
	if a := args.Get(0); a != nil {
		return a.(*LoginAttempt), args.Error(1)
	}
	return nil, args.Error(1)
}

// CreateLoginSession mocks the CreateLoginSession method.
func (m *MockLoginRepository) CreateLoginSession(ctx context.Context, user *User, device DeviceInfo, sessionTTL time.Duration, issueTokens TokenIssuer) (*LoginResult, error) {
	args := m.Called(ctx, user, device, sessionTTL, issueTokens)
	if r := args.Get(0); r != nil {
		return r.(*LoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindSessionByRefreshHash mocks the FindSessionByRefreshHash method.
func (m *MockLoginRepository) FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (*Session, bool, error) {
	args := m.Called(ctx, refreshTokenHash)
	if s := args.Get(0); s != nil {
		return s.(*Session), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

// RotateSession mocks the RotateSession method.
func (m *MockLoginRepository) RotateSession(ctx context.Context, sessionRowID int64, oldHash string, sessionTTL time.Duration, issueTokens TokenIssuer) (*RefreshResult, error) {
	args := m.Called(ctx, sessionRowID, oldHash, sessionTTL, issueTokens)
	if r := args.Get(0); r != nil {
		return r.(*RefreshResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// RevokeSession mocks the RevokeSession method.
func (m *MockLoginRepository) RevokeSession(ctx context.Context, userID int64, sessionID, reason string) error {
	args := m.Called(ctx, userID, sessionID, reason)
	return args.Error(0)
}
