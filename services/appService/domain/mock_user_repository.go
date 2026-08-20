package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	mock.Mock
}

// Register mocks the Register method.
func (m *MockUserRepository) Register(ctx context.Context, user *User, device *Device, issueTokens TokenIssuer) (*Session, error) {
	args := m.Called(ctx, user, device, issueTokens)
	if s := args.Get(0); s != nil {
		return s.(*Session), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindByEmail mocks the FindByEmail method.
func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

// CreateSession mocks the CreateSession method.
func (m *MockUserRepository) CreateSession(ctx context.Context, userID int64, device *Device, issueTokens TokenIssuer) (*Session, error) {
	args := m.Called(ctx, userID, device, issueTokens)
	if s := args.Get(0); s != nil {
		return s.(*Session), args.Error(1)
	}
	return nil, args.Error(1)
}

// CreatePasswordResetToken mocks the CreatePasswordResetToken method.
func (m *MockUserRepository) CreatePasswordResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error) {
	args := m.Called(ctx, userID, tokenHash, expiresAt)
	if t := args.Get(0); t != nil {
		return t.(*PasswordResetToken), args.Error(1)
	}
	return nil, args.Error(1)
}

// ResetPassword mocks the ResetPassword method.
func (m *MockUserRepository) ResetPassword(ctx context.Context, tokenHash, newPasswordHash string) error {
	args := m.Called(ctx, tokenHash, newPasswordHash)
	return args.Error(0)
}

// FindByID mocks the FindByID method.
func (m *MockUserRepository) FindByID(ctx context.Context, userID int64) (*User, error) {
	args := m.Called(ctx, userID)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateProfile mocks the UpdateProfile method.
func (m *MockUserRepository) UpdateProfile(ctx context.Context, userID int64, input ProfileUpdateInput) (*User, error) {
	args := m.Called(ctx, userID, input)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateAvatarURL mocks the UpdateAvatarURL method.
func (m *MockUserRepository) UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) (*User, error) {
	args := m.Called(ctx, userID, avatarURL)
	if u := args.Get(0); u != nil {
		return u.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}
