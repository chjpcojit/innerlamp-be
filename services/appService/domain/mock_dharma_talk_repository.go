package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ DharmaTalkRepository = (*MockDharmaTalkRepository)(nil)

// MockDharmaTalkRepository is a hand-written mock of DharmaTalkRepository (see
// mock_temple_repository.go for the pattern).
type MockDharmaTalkRepository struct {
	mock.Mock
}

// List mocks the List method.
func (m *MockDharmaTalkRepository) List(ctx context.Context, offset, limit int) ([]DharmaTalk, int, error) {
	args := m.Called(ctx, offset, limit)
	if a := args.Get(0); a != nil {
		return a.([]DharmaTalk), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// FindByID mocks the FindByID method.
func (m *MockDharmaTalkRepository) FindByID(ctx context.Context, id string) (*DharmaTalk, error) {
	args := m.Called(ctx, id)
	if a := args.Get(0); a != nil {
		return a.(*DharmaTalk), args.Error(1)
	}
	return nil, args.Error(1)
}

// Count mocks the Count method.
func (m *MockDharmaTalkRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// InsertMissing mocks the InsertMissing method.
func (m *MockDharmaTalkRepository) InsertMissing(ctx context.Context, talks []DharmaTalk) (int, error) {
	args := m.Called(ctx, talks)
	return args.Int(0), args.Error(1)
}
