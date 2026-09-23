package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockLoginUsecase is a hand-written testify.Mock implementing loginUsecase —
// this package's own consumer-defined interface — so LoginHandler is tested
// without a real use case, repository or JWT service. See mocking_convention.md.
type mockLoginUsecase struct {
	mock.Mock
}

func (m *mockLoginUsecase) Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.LoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockLoginUsecase) Refresh(ctx context.Context, input domain.RefreshInput) (*domain.RefreshResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.RefreshResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockLoginUsecase) Logout(ctx context.Context, input domain.LogoutInput) error {
	return m.Called(ctx, input).Error(0)
}
