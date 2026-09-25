package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// The mock must satisfy the port, and this line is what makes that true at
// compile time rather than at first use in a test. Without it, a method added
// to OTPRepository leaves the mock silently short until some test happens to
// exercise the new method — see the "when a port's interface changes" note in
// the package README.
var _ OTPRepository = (*MockOTPRepository)(nil)

// MockOTPRepository is a mock implementation of OTPRepository for testing.
// Hand-written, in the same package as the interface (the repo has no
// mockgen/mockery codegen step).
type MockOTPRepository struct {
	mock.Mock
}

// IssueOTP mocks the IssueOTP method.
func (m *MockOTPRepository) IssueOTP(ctx context.Context, phoneNumber string, purpose OTPPurpose, otpHash string, expiresAt time.Time, limits SendLimits) (*SendState, error) {
	args := m.Called(ctx, phoneNumber, purpose, otpHash, expiresAt, limits)
	if s := args.Get(0); s != nil {
		return s.(*SendState), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindActiveOTP mocks the FindActiveOTP method.
func (m *MockOTPRepository) FindActiveOTP(ctx context.Context, phoneNumber string, purpose OTPPurpose) (*OTPCode, error) {
	args := m.Called(ctx, phoneNumber, purpose)
	if o := args.Get(0); o != nil {
		return o.(*OTPCode), args.Error(1)
	}
	return nil, args.Error(1)
}

// IncrementOTPAttempts mocks the IncrementOTPAttempts method.
func (m *MockOTPRepository) IncrementOTPAttempts(ctx context.Context, otpID int64) (int, error) {
	args := m.Called(ctx, otpID)
	return args.Int(0), args.Error(1)
}

// LockOTP mocks the LockOTP method.
func (m *MockOTPRepository) LockOTP(ctx context.Context, otpID int64, lockedUntil time.Time) error {
	args := m.Called(ctx, otpID, lockedUntil)
	return args.Error(0)
}

// ReleaseSendLimit mocks the ReleaseSendLimit method.
func (m *MockOTPRepository) ReleaseSendLimit(ctx context.Context, phoneNumber string, purpose OTPPurpose) error {
	args := m.Called(ctx, phoneNumber, purpose)
	return args.Error(0)
}
