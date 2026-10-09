package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockSutraUsecase implements sutraUsecase for handler tests.
type mockSutraUsecase struct {
	mock.Mock
}

func (m *mockSutraUsecase) List(ctx context.Context, in domain.SutraFilterInput, page int) (*domain.SutraPage, error) {
	args := m.Called(ctx, in, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.SutraPage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSutraUsecase) Get(ctx context.Context, id string) (*domain.Sutra, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSutraUsecase) Lines(ctx context.Context, sutraID, key string, page int, pageSize *int) (*domain.SutraLinesPage, error) {
	args := m.Called(ctx, sutraID, key, page, pageSize)
	if r := args.Get(0); r != nil {
		return r.(*domain.SutraLinesPage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSutraUsecase) ListByDeity(ctx context.Context, deityID string, role *string, page int) (*domain.DeitySutraPage, error) {
	args := m.Called(ctx, deityID, role, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.DeitySutraPage), args.Error(1)
	}
	return nil, args.Error(1)
}
