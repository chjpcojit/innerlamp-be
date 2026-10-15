package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

var _ AuthMailer = (*MockAuthMailer)(nil)

// MockAuthMailer is a mock implementation of AuthMailer for testing. Usecase
// tests use it to assert which email went out (register code vs account-exists
// notice) and to read the clear code a test then submits.
type MockAuthMailer struct {
	mock.Mock
}

// SendRegisterCode mocks the SendRegisterCode method.
func (m *MockAuthMailer) SendRegisterCode(ctx context.Context, to, code string, ttl time.Duration) error {
	return m.Called(ctx, to, code, ttl).Error(0)
}

// SendAccountExistsNotice mocks the SendAccountExistsNotice method.
func (m *MockAuthMailer) SendAccountExistsNotice(ctx context.Context, to string) error {
	return m.Called(ctx, to).Error(0)
}

// SendPasswordResetCode mocks the SendPasswordResetCode method.
func (m *MockAuthMailer) SendPasswordResetCode(ctx context.Context, to, code string, ttl time.Duration) error {
	return m.Called(ctx, to, code, ttl).Error(0)
}
