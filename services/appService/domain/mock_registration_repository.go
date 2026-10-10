package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// The mock must satisfy the port, and this line is what makes that true at
// compile time rather than at first use in a test. Without it, a method added
// to RegistrationRepository leaves the mock silently short until some test
// happens to exercise the new method — see the "when a port's interface
// changes" note in the package README.
var _ RegistrationRepository = (*MockRegistrationRepository)(nil)

// MockRegistrationRepository is a mock implementation of RegistrationRepository
// for testing. Hand-written, in the same package as the interface (the repo has
// no mockgen/mockery codegen step): see pkg/jwt/mock.go for the same pattern.
type MockRegistrationRepository struct {
	mock.Mock
}

// FindConflicts mocks the FindConflicts method.
func (m *MockRegistrationRepository) FindConflicts(ctx context.Context, phoneNumber, email string) (*RegistrationConflicts, error) {
	args := m.Called(ctx, phoneNumber, email)
	if c := args.Get(0); c != nil {
		return c.(*RegistrationConflicts), args.Error(1)
	}
	return nil, args.Error(1)
}

// TempleExists mocks the TempleExists method.
func (m *MockRegistrationRepository) TempleExists(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// SavePendingAndIssueOTP mocks the SavePendingAndIssueOTP method.
func (m *MockRegistrationRepository) SavePendingAndIssueOTP(ctx context.Context, pending *PendingRegistration, otpHash string, expiresAt time.Time, limits SendLimits) (*SendState, error) {
	args := m.Called(ctx, pending, otpHash, expiresAt, limits)
	if s := args.Get(0); s != nil {
		return s.(*SendState), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindPendingRegistration mocks the FindPendingRegistration method.
func (m *MockRegistrationRepository) FindPendingRegistration(ctx context.Context, phoneNumber string) (*PendingRegistration, error) {
	args := m.Called(ctx, phoneNumber)
	if p := args.Get(0); p != nil {
		return p.(*PendingRegistration), args.Error(1)
	}
	return nil, args.Error(1)
}

// CreateUserAndSession mocks the CreateUserAndSession method.
func (m *MockRegistrationRepository) CreateUserAndSession(ctx context.Context, pending *PendingRegistration, otpID int64, sessionTTL time.Duration, issueTokens TokenIssuer) (*RegisterResult, error) {
	args := m.Called(ctx, pending, otpID, sessionTTL, issueTokens)
	if r := args.Get(0); r != nil {
		return r.(*RegisterResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// DeletePendingRegistration mocks the DeletePendingRegistration method.
func (m *MockRegistrationRepository) DeletePendingRegistration(ctx context.Context, pendingID int64) error {
	args := m.Called(ctx, pendingID)
	return args.Error(0)
}
