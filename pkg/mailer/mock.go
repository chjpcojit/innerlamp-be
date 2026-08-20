package mailer

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockMailer is a mock implementation of Mailer for testing.
type MockMailer struct {
	mock.Mock
}

// SendPasswordResetEmail mocks the SendPasswordResetEmail method.
func (m *MockMailer) SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error {
	args := m.Called(ctx, toEmail, resetToken)
	return args.Error(0)
}
