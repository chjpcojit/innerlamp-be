package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ TempleEventRepository = (*MockTempleEventRepository)(nil)

// MockTempleEventRepository is a hand-written mock of TempleEventRepository
// (see mock_administrative_unit_repository.go for the pattern).
type MockTempleEventRepository struct {
	mock.Mock
}

// ListActiveByTemple mocks the ListActiveByTemple method.
func (m *MockTempleEventRepository) ListActiveByTemple(ctx context.Context, templeID string, w TempleEventWindow) ([]TempleEvent, error) {
	args := m.Called(ctx, templeID, w)
	if v := args.Get(0); v != nil {
		return v.([]TempleEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

// ReplaceCatalogue mocks the ReplaceCatalogue method.
func (m *MockTempleEventRepository) ReplaceCatalogue(ctx context.Context, c TempleEventCatalogue) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
