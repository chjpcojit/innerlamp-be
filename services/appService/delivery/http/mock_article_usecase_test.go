package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockArticleUsecase implements articleUsecase for handler tests.
type mockArticleUsecase struct {
	mock.Mock
}

func (m *mockArticleUsecase) Top(ctx context.Context) ([]domain.Article, error) {
	args := m.Called(ctx)
	if r := args.Get(0); r != nil {
		return r.([]domain.Article), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockArticleUsecase) List(ctx context.Context, page int) (*domain.ArticlePage, error) {
	args := m.Called(ctx, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.ArticlePage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockArticleUsecase) Get(ctx context.Context, id string) (*domain.Article, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.Article), args.Error(1)
	}
	return nil, args.Error(1)
}
