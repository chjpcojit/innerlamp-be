package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockDharmaTalkUsecase implements dharmaTalkUsecase for handler tests.
type mockDharmaTalkUsecase struct {
	mock.Mock
}

func (m *mockDharmaTalkUsecase) List(ctx context.Context, page int) (*domain.DharmaTalkPage, error) {
	args := m.Called(ctx, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.DharmaTalkPage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDharmaTalkUsecase) Get(ctx context.Context, id string) (*domain.DharmaTalk, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.DharmaTalk), args.Error(1)
	}
	return nil, args.Error(1)
}
