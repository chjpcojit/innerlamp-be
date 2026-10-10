package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ TempleRepository = (*MockTempleRepository)(nil)

// MockTempleRepository is a hand-written mock of TempleRepository (see
// mock_login_repository.go for the pattern).
type MockTempleRepository struct {
	mock.Mock
}

// ListTop mocks the ListTop method.
func (m *MockTempleRepository) ListTop(ctx context.Context, limit int) ([]Temple, error) {
	args := m.Called(ctx, limit)
	if t := args.Get(0); t != nil {
		return t.([]Temple), args.Error(1)
	}
	return nil, args.Error(1)
}

// List mocks the List method.
func (m *MockTempleRepository) List(ctx context.Context, offset, limit int) ([]Temple, int, error) {
	args := m.Called(ctx, offset, limit)
	if t := args.Get(0); t != nil {
		return t.([]Temple), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// ListOptions mocks the ListOptions method.
func (m *MockTempleRepository) ListOptions(ctx context.Context, query string, offset, limit int) ([]TempleOption, int, error) {
	args := m.Called(ctx, query, offset, limit)
	if t := args.Get(0); t != nil {
		return t.([]TempleOption), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// FindByID mocks the FindByID method.
func (m *MockTempleRepository) FindByID(ctx context.Context, id string) (*Temple, error) {
	args := m.Called(ctx, id)
	if t := args.Get(0); t != nil {
		return t.(*Temple), args.Error(1)
	}
	return nil, args.Error(1)
}

// Upsert mocks the Upsert method.
func (m *MockTempleRepository) Upsert(ctx context.Context, temples []Temple) error {
	return m.Called(ctx, temples).Error(0)
}
