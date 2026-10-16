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
func (m *MockOTPRepository) IssueOTP(ctx context.Context, email string, purpose OTPPurpose, otpHash string, expiresAt time.Time, limits SendLimits) (*SendState, error) {
	args := m.Called(ctx, email, purpose, otpHash, expiresAt, limits)
	if s := args.Get(0); s != nil {
		return s.(*SendState), args.Error(1)
	}
	return nil, args.Error(1)
}

// ApplySendLimit mocks the ApplySendLimit method.
func (m *MockOTPRepository) ApplySendLimit(ctx context.Context, email string, purpose OTPPurpose, limits SendLimits) (*SendState, error) {
	args := m.Called(ctx, email, purpose, limits)
	if s := args.Get(0); s != nil {
		return s.(*SendState), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindActiveOTP mocks the FindActiveOTP method.
func (m *MockOTPRepository) FindActiveOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTPCode, error) {
	args := m.Called(ctx, email, purpose)
	if o := args.Get(0); o != nil {
		return o.(*OTPCode), args.Error(1)
	}
	return nil, args.Error(1)
}

// ConsumeOTPAttempt mocks the ConsumeOTPAttempt method.
func (m *MockOTPRepository) ConsumeOTPAttempt(ctx context.Context, otpID int64, maxAttempts int, lockUntil time.Time) (int, bool, error) {
	args := m.Called(ctx, otpID, maxAttempts, lockUntil)
	return args.Int(0), args.Bool(1), args.Error(2)
}

// ReleaseSendLimit mocks the ReleaseSendLimit method.
func (m *MockOTPRepository) ReleaseSendLimit(ctx context.Context, email string, purpose OTPPurpose) error {
	args := m.Called(ctx, email, purpose)
	return args.Error(0)
}

// InvalidateActiveOTP mocks the InvalidateActiveOTP method.
func (m *MockOTPRepository) InvalidateActiveOTP(ctx context.Context, email string, purpose OTPPurpose) error {
	args := m.Called(ctx, email, purpose)
	return args.Error(0)
}
