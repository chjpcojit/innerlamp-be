package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

var _ AvatarStorage = (*MockAvatarStorage)(nil)

// MockAvatarStorage is a hand-written mock of AvatarStorage.
type MockAvatarStorage struct {
	mock.Mock
}

// SignUpload mocks the SignUpload method.
func (m *MockAvatarStorage) SignUpload(ctx context.Context, key, contentType string, maxBytes int64, ttl time.Duration) (*SignedUpload, error) {
	args := m.Called(ctx, key, contentType, maxBytes, ttl)
	if r := args.Get(0); r != nil {
		return r.(*SignedUpload), args.Error(1)
	}
	return nil, args.Error(1)
}

// SignRead mocks the SignRead method.
func (m *MockAvatarStorage) SignRead(ctx context.Context, key string, ttl time.Duration) (string, error) {
	args := m.Called(ctx, key, ttl)
	return args.String(0), args.Error(1)
}

// Stat mocks the Stat method.
func (m *MockAvatarStorage) Stat(ctx context.Context, key string) (*ObjectMeta, error) {
	args := m.Called(ctx, key)
	if r := args.Get(0); r != nil {
		return r.(*ObjectMeta), args.Error(1)
	}
	return nil, args.Error(1)
}

// Copy mocks the Copy method.
func (m *MockAvatarStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	return m.Called(ctx, srcKey, dstKey).Error(0)
}

// Delete mocks the Delete method.
func (m *MockAvatarStorage) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
