package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockCalendarEventUsecase implements calendarEventUsecase for handler tests.
type mockCalendarEventUsecase struct {
	mock.Mock
}

func (m *mockCalendarEventUsecase) EventsForYear(ctx context.Context, year string) (domain.CalendarEventsResult, error) {
	args := m.Called(ctx, year)
	return args.Get(0).(domain.CalendarEventsResult), args.Error(1)
}
