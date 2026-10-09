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

// SendAccountInviteEmail mocks the SendAccountInviteEmail method.
func (m *MockMailer) SendAccountInviteEmail(ctx context.Context, toEmail, name, tempPassword string) error {
	return m.Called(ctx, toEmail, name, tempPassword).Error(0)
}
