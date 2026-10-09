package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ SutraRepository = (*MockSutraRepository)(nil)

// MockSutraRepository is a hand-written mock of SutraRepository (see
// mock_temple_repository.go for the pattern).
type MockSutraRepository struct {
	mock.Mock
}

// Count mocks the Count method.
func (m *MockSutraRepository) Count(ctx context.Context, filter SutraFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// List mocks the List method.
func (m *MockSutraRepository) List(ctx context.Context, filter SutraFilter, offset, limit int) ([]Sutra, error) {
	args := m.Called(ctx, filter, offset, limit)
	if s := args.Get(0); s != nil {
		return s.([]Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindByID mocks the FindByID method.
func (m *MockSutraRepository) FindByID(ctx context.Context, id string) (*Sutra, error) {
	args := m.Called(ctx, id)
	if s := args.Get(0); s != nil {
		return s.(*Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindTranslation mocks the FindTranslation method.
func (m *MockSutraRepository) FindTranslation(ctx context.Context, sutraID, key string) (*SutraTranslation, error) {
	args := m.Called(ctx, sutraID, key)
	if t := args.Get(0); t != nil {
		return t.(*SutraTranslation), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListLines mocks the ListLines method.
func (m *MockSutraRepository) ListLines(ctx context.Context, translationID int64, fromN, toN int64) ([]SutraLine, error) {
	args := m.Called(ctx, translationID, fromN, toN)
	if l := args.Get(0); l != nil {
		return l.([]SutraLine), args.Error(1)
	}
	return nil, args.Error(1)
}

// CountByDeity mocks the CountByDeity method.
func (m *MockSutraRepository) CountByDeity(ctx context.Context, deityID string, role *DeityRole) (int, error) {
	args := m.Called(ctx, deityID, role)
	return args.Int(0), args.Error(1)
}

// ListByDeity mocks the ListByDeity method.
func (m *MockSutraRepository) ListByDeity(ctx context.Context, deityID string, role *DeityRole, offset, limit int) ([]DeitySutra, error) {
	args := m.Called(ctx, deityID, role, offset, limit)
	if s := args.Get(0); s != nil {
		return s.([]DeitySutra), args.Error(1)
	}
	return nil, args.Error(1)
}

// MissingDeities mocks the MissingDeities method.
func (m *MockSutraRepository) MissingDeities(ctx context.Context, ids []string) ([]string, error) {
	args := m.Called(ctx, ids)
	if s := args.Get(0); s != nil {
		return s.([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

// InsertAll mocks the InsertAll method.
func (m *MockSutraRepository) InsertAll(ctx context.Context, seeds []SutraSeed) (SutraSeedResult, error) {
	args := m.Called(ctx, seeds)
	if r := args.Get(0); r != nil {
		return r.(SutraSeedResult), args.Error(1)
	}
	return SutraSeedResult{}, args.Error(1)
}
