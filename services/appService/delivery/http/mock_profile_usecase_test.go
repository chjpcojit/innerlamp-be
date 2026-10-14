package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockProfileUsecase implements profileUsecase for handler tests.
type mockProfileUsecase struct {
	mock.Mock
}

func (m *mockProfileUsecase) viewResult(args mock.Arguments) (*domain.ProfileView, error) {
	if r := args.Get(0); r != nil {
		return r.(*domain.ProfileView), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProfileUsecase) Get(ctx context.Context, userID int64) (*domain.ProfileView, error) {
	return m.viewResult(m.Called(ctx, userID))
}

func (m *mockProfileUsecase) Update(ctx context.Context, userID int64, patch domain.ProfilePatch) (*domain.ProfileView, error) {
	return m.viewResult(m.Called(ctx, userID, patch))
}

func (m *mockProfileUsecase) RequestAvatarUpload(ctx context.Context, userID int64, req domain.AvatarUploadRequest) (*domain.AvatarUploadTicket, error) {
	args := m.Called(ctx, userID, req)
	if r := args.Get(0); r != nil {
		return r.(*domain.AvatarUploadTicket), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProfileUsecase) ConfirmAvatar(ctx context.Context, userID int64, uploadID string) (*domain.ProfileView, error) {
	return m.viewResult(m.Called(ctx, userID, uploadID))
}

func (m *mockProfileUsecase) RemoveAvatar(ctx context.Context, userID int64) (*domain.ProfileView, error) {
	return m.viewResult(m.Called(ctx, userID))
}
