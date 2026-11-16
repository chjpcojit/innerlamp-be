package http

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockMemorialUsecase implements memorialUsecase for handler tests.
type mockMemorialUsecase struct {
	mock.Mock
}

func (m *mockMemorialUsecase) Create(ctx context.Context, userID int64, in domain.MemorialInput) (*domain.Memorial, error) {
	args := m.Called(ctx, userID, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMemorialUsecase) List(ctx context.Context, userID int64) ([]domain.Memorial, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.([]domain.Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMemorialUsecase) Get(ctx context.Context, userID, id int64) (*domain.Memorial, error) {
	args := m.Called(ctx, userID, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMemorialUsecase) Update(ctx context.Context, userID, id int64, in domain.MemorialInput) (*domain.Memorial, error) {
	args := m.Called(ctx, userID, id, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMemorialUsecase) Delete(ctx context.Context, userID, id int64) error {
	return m.Called(ctx, userID, id).Error(0)
}

func (m *mockMemorialUsecase) SetPhoto(ctx context.Context, userID, id int64, r io.Reader, contentType string, size int64) (*domain.Memorial, error) {
	args := m.Called(ctx, userID, id, r, contentType, size)
	if v := args.Get(0); v != nil {
		return v.(*domain.Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMemorialUsecase) RemovePhoto(ctx context.Context, userID, id int64) (*domain.Memorial, error) {
	args := m.Called(ctx, userID, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Memorial), args.Error(1)
	}
	return nil, args.Error(1)
}
