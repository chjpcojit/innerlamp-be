package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockPasswordResetUsecase implements passwordResetUsecase.
type mockPasswordResetUsecase struct {
	mock.Mock
}

func (m *mockPasswordResetUsecase) RequestOTP(ctx context.Context, input domain.RequestPasswordResetInput) (*domain.RequestPasswordResetResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.RequestPasswordResetResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPasswordResetUsecase) Confirm(ctx context.Context, input domain.ConfirmPasswordResetInput) error {
	return m.Called(ctx, input).Error(0)
}
