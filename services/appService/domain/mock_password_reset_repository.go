package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ PasswordResetRepository = (*MockPasswordResetRepository)(nil)

// MockPasswordResetRepository is a hand-written mock of PasswordResetRepository.
type MockPasswordResetRepository struct {
	mock.Mock
}

func (m *MockPasswordResetRepository) FindVerifiedUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPasswordResetRepository) ResetPassword(ctx context.Context, userID int64, newPasswordHash string, otpID int64) error {
	return m.Called(ctx, userID, newPasswordHash, otpID).Error(0)
}
