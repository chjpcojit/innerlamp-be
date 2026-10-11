package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ TempleSuggestionRepository = (*MockTempleSuggestionRepository)(nil)

// MockTempleSuggestionRepository is a hand-written mock of
// TempleSuggestionRepository (see mock_admin_repository.go for the pattern).
type MockTempleSuggestionRepository struct {
	mock.Mock
}

// List mocks the List method.
func (m *MockTempleSuggestionRepository) List(ctx context.Context, status string, offset, limit int) ([]TempleSuggestion, int, error) {
	args := m.Called(ctx, status, offset, limit)
	if r := args.Get(0); r != nil {
		return r.([]TempleSuggestion), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// FindByID mocks the FindByID method.
func (m *MockTempleSuggestionRepository) FindByID(ctx context.Context, id int64) (*TempleSuggestion, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*TempleSuggestion), args.Error(1)
	}
	return nil, args.Error(1)
}

// CreateTemple mocks the CreateTemple method.
func (m *MockTempleSuggestionRepository) CreateTemple(ctx context.Context, suggestionID, adminID int64, role string, t NewTemple, priority int) (*CreatedTemple, error) {
	args := m.Called(ctx, suggestionID, adminID, role, t, priority)
	if r := args.Get(0); r != nil {
		return r.(*CreatedTemple), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ WardResolver = (*MockWardResolver)(nil)

// MockWardResolver is a hand-written mock of WardResolver.
type MockWardResolver struct {
	mock.Mock
}

// ResolveCurrentWard mocks the ResolveCurrentWard method.
func (m *MockWardResolver) ResolveCurrentWard(ctx context.Context, provinceCode, wardCode string) (*ResolvedWard, error) {
	args := m.Called(ctx, provinceCode, wardCode)
	if r := args.Get(0); r != nil {
		return r.(*ResolvedWard), args.Error(1)
	}
	return nil, args.Error(1)
}
