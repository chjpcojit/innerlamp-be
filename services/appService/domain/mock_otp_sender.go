package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ OTPSender = (*MockOTPSender)(nil)

// MockOTPSender is a mock implementation of OTPSender for testing. It exists so
// usecase tests can assert *what* was sent — phone number, code, purpose —
// without a network call, and so the SMS-failure path (ErrSMSSendFailed) is
// testable by returning an error here.
type MockOTPSender struct {
	mock.Mock
}

// SendOTP mocks the SendOTP method.
func (m *MockOTPSender) SendOTP(ctx context.Context, phoneNumber, otpCode string, purpose OTPPurpose) error {
	args := m.Called(ctx, phoneNumber, otpCode, purpose)
	return args.Error(0)
}
