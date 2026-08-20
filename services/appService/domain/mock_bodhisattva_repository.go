package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockBodhisattvaRepository is a mock implementation of BodhisattvaRepository for testing.
type MockBodhisattvaRepository struct {
	mock.Mock
}

// ListBodhisattvas mocks the ListBodhisattvas method.
func (m *MockBodhisattvaRepository) ListBodhisattvas(ctx context.Context, input ListBodhisattvasInput) ([]*Bodhisattva, error) {
	args := m.Called(ctx, input)
	if s := args.Get(0); s != nil {
		return s.([]*Bodhisattva), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetBodhisattvaBySlug mocks the GetBodhisattvaBySlug method.
func (m *MockBodhisattvaRepository) GetBodhisattvaBySlug(ctx context.Context, slug string) (*Bodhisattva, error) {
	args := m.Called(ctx, slug)
	if s := args.Get(0); s != nil {
		return s.(*Bodhisattva), args.Error(1)
	}
	return nil, args.Error(1)
}
