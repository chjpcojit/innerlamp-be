package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ ProfileRepository = (*MockProfileRepository)(nil)

// MockProfileRepository is a hand-written mock of ProfileRepository (see
// mock_temple_repository.go for the pattern).
type MockProfileRepository struct {
	mock.Mock
}

// Get mocks the Get method.
func (m *MockProfileRepository) Get(ctx context.Context, userID int64) (*Profile, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.(*Profile), args.Error(1)
	}
	return nil, args.Error(1)
}

// Update mocks the Update method.
func (m *MockProfileRepository) Update(ctx context.Context, userID int64, patch ProfilePatch) (*Profile, error) {
	args := m.Called(ctx, userID, patch)
	if r := args.Get(0); r != nil {
		return r.(*Profile), args.Error(1)
	}
	return nil, args.Error(1)
}

// SwapAvatar mocks the SwapAvatar method.
func (m *MockProfileRepository) SwapAvatar(ctx context.Context, userID int64, newObject string) (*string, error) {
	args := m.Called(ctx, userID, newObject)
	if r := args.Get(0); r != nil {
		return r.(*string), args.Error(1)
	}
	return nil, args.Error(1)
}

// ClearAvatar mocks the ClearAvatar method.
func (m *MockProfileRepository) ClearAvatar(ctx context.Context, userID int64) (*string, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.(*string), args.Error(1)
	}
	return nil, args.Error(1)
}

// TempleExists mocks the TempleExists method.
func (m *MockProfileRepository) TempleExists(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
