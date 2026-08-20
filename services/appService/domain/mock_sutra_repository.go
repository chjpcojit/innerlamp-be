package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockSutraRepository is a mock implementation of SutraRepository for testing.
type MockSutraRepository struct {
	mock.Mock
}

// ListSutras mocks the ListSutras method.
func (m *MockSutraRepository) ListSutras(ctx context.Context, input ListSutrasInput) ([]*Sutra, error) {
	args := m.Called(ctx, input)
	if s := args.Get(0); s != nil {
		return s.([]*Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetSutraBySlug mocks the GetSutraBySlug method.
func (m *MockSutraRepository) GetSutraBySlug(ctx context.Context, slug string) (*Sutra, error) {
	args := m.Called(ctx, slug)
	if s := args.Get(0); s != nil {
		return s.(*Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}
