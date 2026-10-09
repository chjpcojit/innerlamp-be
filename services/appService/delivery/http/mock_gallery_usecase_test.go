package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockGalleryUsecase implements galleryUsecase for handler tests.
type mockGalleryUsecase struct {
	mock.Mock
}

func (m *mockGalleryUsecase) Top(ctx context.Context) ([]domain.GalleryImage, error) {
	args := m.Called(ctx)
	if r := args.Get(0); r != nil {
		return r.([]domain.GalleryImage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGalleryUsecase) List(ctx context.Context, imageType string, page int) (*domain.GalleryPage, error) {
	args := m.Called(ctx, imageType, page)
	if r := args.Get(0); r != nil {
		return r.(*domain.GalleryPage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGalleryUsecase) Get(ctx context.Context, id string) (*domain.GalleryImage, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.GalleryImage), args.Error(1)
	}
	return nil, args.Error(1)
}
