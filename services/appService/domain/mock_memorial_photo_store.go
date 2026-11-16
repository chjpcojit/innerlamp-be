package domain

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

var _ MemorialPhotoStore = (*MockMemorialPhotoStore)(nil)

// MockMemorialPhotoStore is a testify mock of MemorialPhotoStore.
type MockMemorialPhotoStore struct {
	mock.Mock
}

// Put mocks the Put method.
func (m *MockMemorialPhotoStore) Put(ctx context.Context, key string, r io.Reader, contentType string, size int64) error {
	return m.Called(ctx, key, r, contentType, size).Error(0)
}

// Delete mocks the Delete method.
func (m *MockMemorialPhotoStore) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

// SignedURL mocks the SignedURL method.
func (m *MockMemorialPhotoStore) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	args := m.Called(ctx, key, ttl)
	return args.String(0), args.Error(1)
}
