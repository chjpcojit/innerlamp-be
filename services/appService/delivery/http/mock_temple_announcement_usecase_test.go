package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockTempleAnnouncementUsecase implements templeAnnouncementUsecase for handler tests.
type mockTempleAnnouncementUsecase struct {
	mock.Mock
}

func (m *mockTempleAnnouncementUsecase) ListForUser(ctx context.Context, userID int64, page, pageSize int) (domain.UserAnnouncements, error) {
	args := m.Called(ctx, userID, page, pageSize)
	return args.Get(0).(domain.UserAnnouncements), args.Error(1)
}

func (m *mockTempleAnnouncementUsecase) GetReceive(ctx context.Context, userID int64) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *mockTempleAnnouncementUsecase) SetReceive(ctx context.Context, userID int64, receive bool) (bool, error) {
	args := m.Called(ctx, userID, receive)
	return args.Bool(0), args.Error(1)
}
