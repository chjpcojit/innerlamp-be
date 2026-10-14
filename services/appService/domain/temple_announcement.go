package domain

import (
	"context"
	"time"
)

// TempleAnnouncement is one notice a temple published, as the people of that
// temple read it (docs/specs/features/temple-announcement). Plain data: no tags.
type TempleAnnouncement struct {
	ID          string
	TempleID    string
	IssuingBody IssuingBodyRef
	Title       string
	Body        string
	PublishedAt time.Time
	ExpiresAt   time.Time
}

// IssuingBodyRef is the body (ban) that issued an announcement.
type IssuingBodyRef struct {
	ID   string
	Name string
}

// AnnouncementPageRequest is a checked page: Page >= 1, 1 <= PageSize <= max.
type AnnouncementPageRequest struct {
	Page     int
	PageSize int
}

// AnnouncementPage is one page of active announcements plus the total of all
// active ones of the temple.
type AnnouncementPage struct {
	Items    []TempleAnnouncement
	Total    int
	Page     int
	PageSize int
}

// AnnouncementSettings is what the user-level read needs in one SELECT:
// the receive switch and the current ordination temple (nil when none).
type AnnouncementSettings struct {
	Receive bool
	Temple  *TempleRef
}

// UserAnnouncements is the answer of "my announcements". Temple is nil when the
// switch is off or the user has no ordination temple.
type UserAnnouncements struct {
	Receive bool
	Temple  *TempleRef
	Page    AnnouncementPage
}

// TempleAnnouncementRepository is the port the read side lists through.
type TempleAnnouncementRepository interface {
	// ListActiveByTemple returns the page of announcements of templeID that are
	// active at now (not revoked, published_at <= now < expires_at), newest
	// published_at first then id, with the total of active announcements.
	ListActiveByTemple(ctx context.Context, templeID string, now time.Time, page AnnouncementPageRequest) (AnnouncementPage, error)
}

// AnnouncementPreferenceRepository reads and writes the per-user switch.
type AnnouncementPreferenceRepository interface {
	// GetSettings returns the user's switch and ordination temple (name joined).
	// Returns ErrUserNotFound if the account no longer exists.
	GetSettings(ctx context.Context, userID int64) (AnnouncementSettings, error)

	// SetReceive stores the switch and returns ErrUserNotFound if there is no row.
	SetReceive(ctx context.Context, userID int64, receive bool) error
}
