package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockTempleScheduleUsecase implements templeScheduleUsecase for handler tests.
type mockTempleScheduleUsecase struct {
	mock.Mock
}

func (m *mockTempleScheduleUsecase) ScheduleForTemple(ctx context.Context, templeID string) ([]domain.TempleScheduleItem, error) {
	args := m.Called(ctx, templeID)
	if v := args.Get(0); v != nil {
		return v.([]domain.TempleScheduleItem), args.Error(1)
	}
	return nil, args.Error(1)
}
