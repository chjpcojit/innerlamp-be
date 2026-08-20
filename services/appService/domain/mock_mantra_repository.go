package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockMantraRepository is a mock implementation of MantraRepository for testing.
type MockMantraRepository struct {
	mock.Mock
}

// ListMantras mocks the ListMantras method.
func (m *MockMantraRepository) ListMantras(ctx context.Context, input ListMantrasInput) ([]*Mantra, error) {
	args := m.Called(ctx, input)
	if s := args.Get(0); s != nil {
		return s.([]*Mantra), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetMantraBySlug mocks the GetMantraBySlug method.
func (m *MockMantraRepository) GetMantraBySlug(ctx context.Context, slug string) (*Mantra, error) {
	args := m.Called(ctx, slug)
	if s := args.Get(0); s != nil {
		return s.(*Mantra), args.Error(1)
	}
	return nil, args.Error(1)
}
