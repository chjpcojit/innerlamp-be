package usecase

import (
	"context"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// TempleAnnouncementUsecase serves the announcements of the caller's current
// ordination temple and the per-user receive switch. Nothing is stored per user
// besides the switch: the temple is read at call time (TA3, TA6).
type TempleAnnouncementUsecase struct {
	prefs         domain.AnnouncementPreferenceRepository
	announcements domain.TempleAnnouncementRepository
	cfg           config.TempleAnnouncementConfig
	now           func() time.Time
}

// NewTempleAnnouncementUsecase wires the use case. cfg must pass Validate; now
// is the clock, injected so tests can fix the active/expired boundary.
func NewTempleAnnouncementUsecase(prefs domain.AnnouncementPreferenceRepository, announcements domain.TempleAnnouncementRepository,
	cfg config.TempleAnnouncementConfig, now func() time.Time) (*TempleAnnouncementUsecase, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &TempleAnnouncementUsecase{prefs: prefs, announcements: announcements, cfg: cfg, now: now}, nil
}

// ListForUser implements flow R-1..R-5. page >= 1; pageSize 0 means the
// configured default, larger than the configured max is capped.
func (u *TempleAnnouncementUsecase) ListForUser(ctx context.Context, userID int64, page, pageSize int) (domain.UserAnnouncements, error) {
	if page < 1 || pageSize < 0 {
		return domain.UserAnnouncements{}, domain.ErrInvalidPage
	}
	if pageSize == 0 {
		pageSize = u.cfg.PageSize
	}
	if pageSize > u.cfg.MaxPageSize {
		pageSize = u.cfg.MaxPageSize
	}
	empty := domain.AnnouncementPage{Items: []domain.TempleAnnouncement{}, Page: page, PageSize: pageSize}

	s, err := u.prefs.GetSettings(ctx, userID)
	if err != nil {
		return domain.UserAnnouncements{}, fmt.Errorf("get announcement settings of user %d: %w", userID, err)
	}
	if !s.Receive {
		return domain.UserAnnouncements{Receive: false, Page: empty}, nil
	}
	if s.Temple == nil {
		return domain.UserAnnouncements{Receive: true, Page: empty}, nil
	}

	res, err := u.announcements.ListActiveByTemple(ctx, s.Temple.ID, u.now(),
		domain.AnnouncementPageRequest{Page: page, PageSize: pageSize})
	if err != nil {
		return domain.UserAnnouncements{}, fmt.Errorf("list announcements of temple %q: %w", s.Temple.ID, err)
	}
	if res.Items == nil {
		res.Items = []domain.TempleAnnouncement{}
	}
	res.Page, res.PageSize = page, pageSize
	return domain.UserAnnouncements{Receive: true, Temple: s.Temple, Page: res}, nil
}

// GetReceive returns the user's stored switch (TA16).
func (u *TempleAnnouncementUsecase) GetReceive(ctx context.Context, userID int64) (bool, error) {
	s, err := u.prefs.GetSettings(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get announcement settings of user %d: %w", userID, err)
	}
	return s.Receive, nil
}

// SetReceive stores the switch and returns the stored value (TA13).
func (u *TempleAnnouncementUsecase) SetReceive(ctx context.Context, userID int64, receive bool) (bool, error) {
	if err := u.prefs.SetReceive(ctx, userID, receive); err != nil {
		return false, fmt.Errorf("set announcement receive of user %d: %w", userID, err)
	}
	return receive, nil
}
