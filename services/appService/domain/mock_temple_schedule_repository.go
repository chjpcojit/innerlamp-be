package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ TempleScheduleRepository = (*MockTempleScheduleRepository)(nil)

// MockTempleScheduleRepository is a hand-written mock of
// TempleScheduleRepository (see mock_administrative_unit_repository.go for the
// pattern).
type MockTempleScheduleRepository struct {
	mock.Mock
}

// ListActiveByTemple mocks the ListActiveByTemple method.
func (m *MockTempleScheduleRepository) ListActiveByTemple(ctx context.Context, templeID string) ([]TempleScheduleItem, error) {
	args := m.Called(ctx, templeID)
	if v := args.Get(0); v != nil {
		return v.([]TempleScheduleItem), args.Error(1)
	}
	return nil, args.Error(1)
}

// ReplaceCatalogue mocks the ReplaceCatalogue method.
func (m *MockTempleScheduleRepository) ReplaceCatalogue(ctx context.Context, c TempleScheduleCatalogue) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
