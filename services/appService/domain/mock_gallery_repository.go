package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ GalleryRepository = (*MockGalleryRepository)(nil)

// MockGalleryRepository is a hand-written mock of GalleryRepository (see
// mock_temple_repository.go for the pattern).
type MockGalleryRepository struct {
	mock.Mock
}

// ListTop mocks the ListTop method.
func (m *MockGalleryRepository) ListTop(ctx context.Context, limit int) ([]GalleryImage, error) {
	args := m.Called(ctx, limit)
	if g := args.Get(0); g != nil {
		return g.([]GalleryImage), args.Error(1)
	}
	return nil, args.Error(1)
}

// List mocks the List method.
func (m *MockGalleryRepository) List(ctx context.Context, imageType string, offset, limit int) ([]GalleryImage, int, error) {
	args := m.Called(ctx, imageType, offset, limit)
	if g := args.Get(0); g != nil {
		return g.([]GalleryImage), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// FindByID mocks the FindByID method.
func (m *MockGalleryRepository) FindByID(ctx context.Context, id string) (*GalleryImage, error) {
	args := m.Called(ctx, id)
	if g := args.Get(0); g != nil {
		return g.(*GalleryImage), args.Error(1)
	}
	return nil, args.Error(1)
}

// Upsert mocks the Upsert method.
func (m *MockGalleryRepository) Upsert(ctx context.Context, images []GalleryImage) error {
	return m.Called(ctx, images).Error(0)
}
