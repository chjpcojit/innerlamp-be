package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockDeityUsecase implements deityUsecase for handler tests.
type mockDeityUsecase struct {
	mock.Mock
}

func (m *mockDeityUsecase) Top(ctx context.Context) ([]domain.Deity, error) {
	args := m.Called(ctx)
	if r := args.Get(0); r != nil {
		return r.([]domain.Deity), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeityUsecase) ListDeities(ctx context.Context, page int) (*domain.DeityPage, error) {
	args := m.Called(ctx, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.DeityPage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeityUsecase) Get(ctx context.Context, id string) (*domain.Deity, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.Deity), args.Error(1)
	}
	return nil, args.Error(1)
}
