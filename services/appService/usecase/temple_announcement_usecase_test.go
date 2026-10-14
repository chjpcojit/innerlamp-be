package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var announcementNow = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func announcementTestConfig() config.TempleAnnouncementConfig {
	return config.TempleAnnouncementConfig{
		PageSize: 20, MaxPageSize: 50,
		RateLimit: config.TempleAnnouncementRateLimit{Max: 60, Window: config.Duration(time.Minute)},
	}
}

type announcementFixture struct {
	uc    *TempleAnnouncementUsecase
	prefs *domain.MockAnnouncementPreferenceRepository
	repo  *domain.MockTempleAnnouncementRepository
}

func newAnnouncementFixture(t *testing.T) *announcementFixture {
	t.Helper()
	f := &announcementFixture{
		prefs: &domain.MockAnnouncementPreferenceRepository{},
		repo:  &domain.MockTempleAnnouncementRepository{},
	}
	uc, err := NewTempleAnnouncementUsecase(f.prefs, f.repo, announcementTestConfig(), func() time.Time { return announcementNow })
	require.NoError(t, err)
	f.uc = uc
	t.Cleanup(func() {
		f.prefs.AssertExpectations(t)
		f.repo.AssertExpectations(t)
	})
	return f
}

func TestNewTempleAnnouncementUsecase_RejectsInvalidConfig(t *testing.T) {
	_, err := NewTempleAnnouncementUsecase(&domain.MockAnnouncementPreferenceRepository{}, &domain.MockTempleAnnouncementRepository{},
		config.TempleAnnouncementConfig{}, time.Now)
	assert.Error(t, err)
}

func TestTempleAnnouncementUsecase_ListForUser(t *testing.T) {
	ctx := context.Background()
	templeA := &domain.TempleRef{ID: "tmp_a", Name: "Chùa A"}
	templeB := &domain.TempleRef{ID: "tmp_b", Name: "Chùa B"}
	item := domain.TempleAnnouncement{ID: "tann_1", TempleID: "tmp_a", Title: "Ghi danh"}

	t.Run("no temple returns empty without reading announcements (TA5)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true}, nil)

		got, err := f.uc.ListForUser(ctx, 7, 1, 0)
		require.NoError(t, err)
		assert.True(t, got.Receive)
		assert.Nil(t, got.Temple)
		assert.Empty(t, got.Page.Items)
		assert.NotNil(t, got.Page.Items)
		assert.Equal(t, 1, got.Page.Page)
		assert.Equal(t, 20, got.Page.PageSize)
		f.repo.AssertNotCalled(t, "ListActiveByTemple", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("switch off returns empty and no temple (TA7)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: false, Temple: templeA}, nil)

		got, err := f.uc.ListForUser(ctx, 7, 1, 10)
		require.NoError(t, err)
		assert.False(t, got.Receive)
		assert.Nil(t, got.Temple)
		assert.Empty(t, got.Page.Items)
		assert.Equal(t, 10, got.Page.PageSize)
		f.repo.AssertNotCalled(t, "ListActiveByTemple", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("lists the temple's page at the injected now (TA1, TA8)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true, Temple: templeA}, nil)
		f.repo.On("ListActiveByTemple", ctx, "tmp_a", announcementNow, domain.AnnouncementPageRequest{Page: 2, PageSize: 20}).
			Return(domain.AnnouncementPage{Items: []domain.TempleAnnouncement{item}, Total: 21, Page: 2, PageSize: 20}, nil)

		got, err := f.uc.ListForUser(ctx, 7, 2, 0)
		require.NoError(t, err)
		assert.True(t, got.Receive)
		assert.Equal(t, templeA, got.Temple)
		assert.Equal(t, 21, got.Page.Total)
		assert.Equal(t, []domain.TempleAnnouncement{item}, got.Page.Items)
	})

	t.Run("page size is capped at max_page_size (TA10)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true, Temple: templeA}, nil)
		f.repo.On("ListActiveByTemple", ctx, "tmp_a", announcementNow, domain.AnnouncementPageRequest{Page: 1, PageSize: 50}).
			Return(domain.AnnouncementPage{Page: 1, PageSize: 50}, nil)

		got, err := f.uc.ListForUser(ctx, 7, 1, 500)
		require.NoError(t, err)
		assert.Equal(t, 50, got.Page.PageSize)
		assert.NotNil(t, got.Page.Items)
	})

	t.Run("temple A to B to A follows the current temple each read (TA6, TA15)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		page := domain.AnnouncementPageRequest{Page: 1, PageSize: 20}
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true, Temple: templeA}, nil).Once()
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true, Temple: templeB}, nil).Once()
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true, Temple: templeA}, nil).Once()
		f.repo.On("ListActiveByTemple", ctx, "tmp_a", announcementNow, page).Return(domain.AnnouncementPage{Items: []domain.TempleAnnouncement{item}, Total: 1}, nil).Twice()
		f.repo.On("ListActiveByTemple", ctx, "tmp_b", announcementNow, page).Return(domain.AnnouncementPage{}, nil).Once()

		for _, want := range []string{"tmp_a", "tmp_b", "tmp_a"} {
			got, err := f.uc.ListForUser(ctx, 7, 1, 0)
			require.NoError(t, err)
			assert.Equal(t, want, got.Temple.ID)
		}
	})

	t.Run("invalid page numbers fail before any read (R-1)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		for _, c := range [][2]int{{0, 0}, {-1, 0}, {1, -1}} {
			_, err := f.uc.ListForUser(ctx, 7, c[0], c[1])
			assert.ErrorIs(t, err, domain.ErrInvalidPage)
		}
		f.prefs.AssertNotCalled(t, "GetSettings", mock.Anything, mock.Anything)
	})

	t.Run("missing user surfaces ErrUserNotFound (R-2)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{}, domain.ErrUserNotFound)
		_, err := f.uc.ListForUser(ctx, 7, 1, 0)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("repository error is wrapped", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		boom := errors.New("db down")
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: true, Temple: templeA}, nil)
		f.repo.On("ListActiveByTemple", ctx, "tmp_a", announcementNow, mock.Anything).Return(domain.AnnouncementPage{}, boom)
		_, err := f.uc.ListForUser(ctx, 7, 1, 0)
		assert.ErrorIs(t, err, boom)
	})
}

func TestTempleAnnouncementUsecase_Receive(t *testing.T) {
	ctx := context.Background()

	t.Run("GetReceive returns the stored flag (TA16)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("GetSettings", ctx, int64(7)).Return(domain.AnnouncementSettings{Receive: false}, nil)
		got, err := f.uc.GetReceive(ctx, 7)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("SetReceive stores and echoes the value (TA13)", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("SetReceive", ctx, int64(7), false).Return(nil)
		got, err := f.uc.SetReceive(ctx, 7, false)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("SetReceive propagates ErrUserNotFound", func(t *testing.T) {
		f := newAnnouncementFixture(t)
		f.prefs.On("SetReceive", ctx, int64(7), true).Return(domain.ErrUserNotFound)
		_, err := f.uc.SetReceive(ctx, 7, true)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
