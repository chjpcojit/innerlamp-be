package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ MemorialRepository = (*MockMemorialRepository)(nil)

// MockMemorialRepository is a hand-written mock of MemorialRepository (see
// mock_temple_repository.go for the pattern).
type MockMemorialRepository struct {
	mock.Mock
}

// Create mocks the Create method.
func (m *MockMemorialRepository) Create(ctx context.Context, mem *Memorial, max int) (*Memorial, error) {
	args := m.Called(ctx, mem, max)
	if r := args.Get(0); r != nil {
		return r.(*Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListByUser mocks the ListByUser method.
func (m *MockMemorialRepository) ListByUser(ctx context.Context, userID int64) ([]Memorial, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.([]Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindByIDAndUser mocks the FindByIDAndUser method.
func (m *MockMemorialRepository) FindByIDAndUser(ctx context.Context, id, userID int64) (*Memorial, error) {
	args := m.Called(ctx, id, userID)
	if r := args.Get(0); r != nil {
		return r.(*Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

// Update mocks the Update method.
func (m *MockMemorialRepository) Update(ctx context.Context, mem *Memorial) (*Memorial, error) {
	args := m.Called(ctx, mem)
	if r := args.Get(0); r != nil {
		return r.(*Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockMemorialRepository) Delete(ctx context.Context, id, userID int64) error {
	return m.Called(ctx, id, userID).Error(0)
}
