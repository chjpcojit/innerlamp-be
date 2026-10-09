package http

import (
	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockCalendarUsecase implements calendarUsecase for handler tests.
type mockCalendarUsecase struct {
	mock.Mock
}

func (m *mockCalendarUsecase) ImageForDate(date string) (string, domain.CalendarImage, error) {
	args := m.Called(date)
	return args.String(0), args.Get(1).(domain.CalendarImage), args.Error(2)
}
