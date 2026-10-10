package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockTempleUsecase implements templeUsecase for handler tests.
type mockTempleUsecase struct {
	mock.Mock
}

func (m *mockTempleUsecase) Top(ctx context.Context) ([]domain.Temple, error) {
	args := m.Called(ctx)
	if r := args.Get(0); r != nil {
		return r.([]domain.Temple), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTempleUsecase) Options(ctx context.Context, query *string, page int) (*domain.TempleOptionPage, error) {
	args := m.Called(ctx, query, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.TempleOptionPage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTempleUsecase) List(ctx context.Context, page int) (*domain.TemplePage, error) {
	args := m.Called(ctx, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.TemplePage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTempleUsecase) Get(ctx context.Context, id string) (*domain.Temple, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.Temple), args.Error(1)
	}
	return nil, args.Error(1)
}
