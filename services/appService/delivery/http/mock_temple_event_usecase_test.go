package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockTempleEventUsecase implements templeEventUsecase for handler tests.
type mockTempleEventUsecase struct {
	mock.Mock
}

func (m *mockTempleEventUsecase) EventsForTemple(ctx context.Context, templeID, from, to string) (domain.TempleEventsResult, error) {
	args := m.Called(ctx, templeID, from, to)
	return args.Get(0).(domain.TempleEventsResult), args.Error(1)
}
