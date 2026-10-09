package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

var _ AdminRepository = (*MockAdminRepository)(nil)

// MockAdminRepository is a hand-written mock of AdminRepository.
type MockAdminRepository struct {
	mock.Mock
}

// FindByEmail mocks the FindByEmail method.
func (m *MockAdminRepository) FindByEmail(ctx context.Context, email string) (*Admin, error) {
	args := m.Called(ctx, email)
	if a := args.Get(0); a != nil {
		return a.(*Admin), args.Error(1)
	}
	return nil, args.Error(1)
}

// TouchLastLogin mocks the TouchLastLogin method.
func (m *MockAdminRepository) TouchLastLogin(ctx context.Context, id int64, at time.Time) error {
	return m.Called(ctx, id, at).Error(0)
}

// Create mocks the Create method.
func (m *MockAdminRepository) Create(ctx context.Context, email, passwordHash, name string) (*Admin, error) {
	args := m.Called(ctx, email, passwordHash, name)
	if a := args.Get(0); a != nil {
		return a.(*Admin), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindByID mocks the FindByID method.
func (m *MockAdminRepository) FindByID(ctx context.Context, id int64) (*Admin, error) {
	args := m.Called(ctx, id)
	if a := args.Get(0); a != nil {
		return a.(*Admin), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdatePassword mocks the UpdatePassword method.
func (m *MockAdminRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	return m.Called(ctx, id, passwordHash).Error(0)
}

// CreateBootstrapIfEmpty mocks the CreateBootstrapIfEmpty method.
func (m *MockAdminRepository) CreateBootstrapIfEmpty(ctx context.Context, email, passwordHash, name string) (bool, error) {
	args := m.Called(ctx, email, passwordHash, name)
	return args.Bool(0), args.Error(1)
}

// CreateAccount mocks the CreateAccount method.
func (m *MockAdminRepository) CreateAccount(ctx context.Context, email, passwordHash, name, role string, tempExpiresAt *time.Time) (*Admin, error) {
	args := m.Called(ctx, email, passwordHash, name, role, tempExpiresAt)
	if a := args.Get(0); a != nil {
		return a.(*Admin), args.Error(1)
	}
	return nil, args.Error(1)
}

// ResetTempPassword mocks the ResetTempPassword method.
func (m *MockAdminRepository) ResetTempPassword(ctx context.Context, id int64, passwordHash string, tempExpiresAt *time.Time) (*Admin, error) {
	args := m.Called(ctx, id, passwordHash, tempExpiresAt)
	if a := args.Get(0); a != nil {
		return a.(*Admin), args.Error(1)
	}
	return nil, args.Error(1)
}

// DeleteByID mocks the DeleteByID method.
func (m *MockAdminRepository) DeleteByID(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
