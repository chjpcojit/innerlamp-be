package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ TempleSuggestionRepository = (*MockTempleSuggestionRepository)(nil)

// MockTempleSuggestionRepository is a hand-written mock of
// TempleSuggestionRepository (see mock_temple_repository.go for the pattern).
type MockTempleSuggestionRepository struct {
	mock.Mock
}

// Create mocks the Create method.
func (m *MockTempleSuggestionRepository) Create(ctx context.Context, s *TempleSuggestion) (*TempleSuggestion, error) {
	args := m.Called(ctx, s)
	if r := args.Get(0); r != nil {
		return r.(*TempleSuggestion), args.Error(1)
	}
	return nil, args.Error(1)
}
