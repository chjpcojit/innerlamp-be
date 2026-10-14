package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ DeityRepository = (*MockDeityRepository)(nil)

// MockDeityRepository is a hand-written mock of DeityRepository (see
// mock_temple_repository.go for the pattern).
type MockDeityRepository struct {
	mock.Mock
}

// ListTop mocks the ListTop method.
func (m *MockDeityRepository) ListTop(ctx context.Context, limit int) ([]Deity, error) {
	args := m.Called(ctx, limit)
	if d := args.Get(0); d != nil {
		return d.([]Deity), args.Error(1)
	}
	return nil, args.Error(1)
}

// List mocks the List method.
func (m *MockDeityRepository) List(ctx context.Context, deityType string, offset, limit int) ([]Deity, int, error) {
	args := m.Called(ctx, deityType, offset, limit)
	if d := args.Get(0); d != nil {
		return d.([]Deity), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// FindByID mocks the FindByID method.
func (m *MockDeityRepository) FindByID(ctx context.Context, id string) (*Deity, error) {
	args := m.Called(ctx, id)
	if d := args.Get(0); d != nil {
		return d.(*Deity), args.Error(1)
	}
	return nil, args.Error(1)
}

// Upsert mocks the Upsert method.
func (m *MockDeityRepository) Upsert(ctx context.Context, deities []Deity) error {
	return m.Called(ctx, deities).Error(0)
}
