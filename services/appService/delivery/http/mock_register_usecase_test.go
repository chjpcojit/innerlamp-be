package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockRegisterUsecase is a hand-written testify.Mock implementing
// registerUsecase — this package's own consumer-defined interface — so
// RegisterHandler is tested without a real use case, repository, JWT
// service, or SMS sender. See mocking_convention.md.
type mockRegisterUsecase struct {
	mock.Mock
}

func (m *mockRegisterUsecase) RequestOTP(ctx context.Context, input domain.RequestRegisterOTPInput) (*domain.RequestOTPResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.RequestOTPResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRegisterUsecase) ResendOTP(ctx context.Context, input domain.ResendOTPInput) (*domain.ResendOTPResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.ResendOTPResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRegisterUsecase) VerifyOTP(ctx context.Context, input domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.RegisterResult), args.Error(1)
	}
	return nil, args.Error(1)
}
