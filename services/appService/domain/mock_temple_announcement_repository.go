package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

var (
	_ TempleAnnouncementRepository     = (*MockTempleAnnouncementRepository)(nil)
	_ AnnouncementPreferenceRepository = (*MockAnnouncementPreferenceRepository)(nil)
)

// MockTempleAnnouncementRepository is a hand-written mock of TempleAnnouncementRepository.
type MockTempleAnnouncementRepository struct {
	mock.Mock
}

// ListActiveByTemple mocks the ListActiveByTemple method.
func (m *MockTempleAnnouncementRepository) ListActiveByTemple(ctx context.Context, templeID string, now time.Time, page AnnouncementPageRequest) (AnnouncementPage, error) {
	args := m.Called(ctx, templeID, now, page)
	return args.Get(0).(AnnouncementPage), args.Error(1)
}

// MockAnnouncementPreferenceRepository is a hand-written mock of AnnouncementPreferenceRepository.
type MockAnnouncementPreferenceRepository struct {
	mock.Mock
}

// GetSettings mocks the GetSettings method.
func (m *MockAnnouncementPreferenceRepository) GetSettings(ctx context.Context, userID int64) (AnnouncementSettings, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(AnnouncementSettings), args.Error(1)
}

// SetReceive mocks the SetReceive method.
func (m *MockAnnouncementPreferenceRepository) SetReceive(ctx context.Context, userID int64, receive bool) error {
	args := m.Called(ctx, userID, receive)
	return args.Error(0)
}
