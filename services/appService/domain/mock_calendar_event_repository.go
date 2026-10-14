package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ CalendarEventRepository = (*MockCalendarEventRepository)(nil)

// MockCalendarEventRepository is a hand-written mock of CalendarEventRepository
// (see mock_administrative_unit_repository.go for the pattern).
type MockCalendarEventRepository struct {
	mock.Mock
}

// ListGroups mocks the ListGroups method.
func (m *MockCalendarEventRepository) ListGroups(ctx context.Context) ([]CalendarGroup, error) {
	args := m.Called(ctx)
	if v := args.Get(0); v != nil {
		return v.([]CalendarGroup), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListActiveEvents mocks the ListActiveEvents method.
func (m *MockCalendarEventRepository) ListActiveEvents(ctx context.Context, lang string) ([]CalendarEvent, error) {
	args := m.Called(ctx, lang)
	if v := args.Get(0); v != nil {
		return v.([]CalendarEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

// ReplaceCatalogue mocks the ReplaceCatalogue method.
func (m *MockCalendarEventRepository) ReplaceCatalogue(ctx context.Context, c CalendarCatalogue) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
