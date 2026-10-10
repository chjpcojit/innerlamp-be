package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockLampUsecase implements lampUsecase for handler tests.
type mockLampUsecase struct {
	mock.Mock
}

func (m *mockLampUsecase) Light(ctx context.Context, userID int64, in domain.LampInput) (*domain.Lamp, bool, error) {
	args := m.Called(ctx, userID, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.Lamp), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

func (m *mockLampUsecase) Garden(ctx context.Context, f domain.GardenFilter) (*domain.GardenPage, error) {
	args := m.Called(ctx, f)
	if r := args.Get(0); r != nil {
		return r.(*domain.GardenPage), args.Error(1)
	}
	return nil, args.Error(1)
}
