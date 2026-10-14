package usecase

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/backofficeService/config"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// TempleAnnouncementUsecase publishes, edits, revokes and lists a temple's
// announcements and manages its issuing bodies. Who may call it is the route's
// decision (reviewer); nothing here looks at the caller's role.
type TempleAnnouncementUsecase struct {
	announcements domain.AnnouncementRepository
	bodies        domain.IssuingBodyRepository
	cfg           config.TempleAnnouncementConfig
	now           func() time.Time
}

// NewTempleAnnouncementUsecase wires the use case with the temple_announcement
// config block.
func NewTempleAnnouncementUsecase(a domain.AnnouncementRepository, b domain.IssuingBodyRepository, cfg config.TempleAnnouncementConfig) *TempleAnnouncementUsecase {
	return &TempleAnnouncementUsecase{announcements: a, bodies: b, cfg: cfg, now: time.Now}
}

func annErr(field, msg string) error {
	return &domain.AnnouncementInputError{Field: field, Message: msg, Kind: domain.ErrInvalidAnnouncement}
}

// checkText rejects a blank value and one longer than max characters (runes).
func (u *TempleAnnouncementUsecase) checkText(field, v string, max int) error {
	switch {
	case strings.TrimSpace(v) == "":
		return annErr(field, field+" is required")
	case utf8.RuneCountInString(v) > max:
		return annErr(field, field+" is too long")
	}
	return nil
}

func (u *TempleAnnouncementUsecase) checkExpiry(t time.Time, now time.Time) error {
	if t.IsZero() {
		return annErr("expires_at", "expires_at is required")
	}
	if !t.After(now) {
		return annErr("expires_at", "expires_at must be in the future")
	}
	if t.After(now.Add(time.Duration(u.cfg.MaxValidityDays) * 24 * time.Hour)) {
		return annErr("expires_at", "expires_at is too far in the future")
	}
	return nil
}

func (u *TempleAnnouncementUsecase) withState(a *domain.Announcement, now time.Time) *domain.Announcement {
	a.State = a.StateAt(now)
	return a
}

// Publish validates the form, checks the temple and issuing body, and stores
// an announcement with source admin, published_at = now (design P-3..P-6).
// Validation runs first so a refused form touches nothing.
func (u *TempleAnnouncementUsecase) Publish(ctx context.Context, templeID, adminID string, in domain.NewAnnouncement) (*domain.Announcement, error) {
	now := u.now()
	if err := u.checkText("title", in.Title, u.cfg.TitleMaxChars); err != nil {
		return nil, err
	}
	if err := u.checkText("body", in.Body, u.cfg.BodyMaxChars); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.IssuingBodyID) == "" {
		return nil, annErr("issuing_body_id", "issuing_body_id is required")
	}
	if err := u.checkExpiry(in.ExpiresAt, now); err != nil {
		return nil, err
	}

	ok, err := u.announcements.TempleExists(ctx, templeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrTempleNotFound
	}
	ok, err = u.announcements.IssuingBodyBelongs(ctx, in.IssuingBodyID, templeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrInvalidIssuingBody
	}

	created, err := u.announcements.Insert(ctx, domain.Announcement{
		TempleID: templeID, IssuingBodyID: in.IssuingBodyID, Title: in.Title, Body: in.Body,
		Source: domain.AnnouncementSourceAdmin, PublishedAt: now, ExpiresAt: in.ExpiresAt, CreatedBy: &adminID,
	})
	if err != nil {
		return nil, err
	}
	return u.withState(created, now), nil
}

// Edit changes the given fields of a non-revoked announcement under the same
// validation as Publish; published_at and source stay.
func (u *TempleAnnouncementUsecase) Edit(ctx context.Context, id string, p domain.AnnouncementPatch) (*domain.Announcement, error) {
	now := u.now()
	cur, err := u.announcements.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.RevokedAt != nil {
		return nil, domain.ErrAnnouncementRevoked
	}

	if p.Title != nil {
		if err := u.checkText("title", *p.Title, u.cfg.TitleMaxChars); err != nil {
			return nil, err
		}
		cur.Title = *p.Title
	}
	if p.Body != nil {
		if err := u.checkText("body", *p.Body, u.cfg.BodyMaxChars); err != nil {
			return nil, err
		}
		cur.Body = *p.Body
	}
	if p.ExpiresAt != nil {
		if err := u.checkExpiry(*p.ExpiresAt, now); err != nil {
			return nil, err
		}
		cur.ExpiresAt = *p.ExpiresAt
	}
	if p.IssuingBodyID != nil {
		if strings.TrimSpace(*p.IssuingBodyID) == "" {
			return nil, annErr("issuing_body_id", "issuing_body_id is required")
		}
		ok, err := u.announcements.IssuingBodyBelongs(ctx, *p.IssuingBodyID, cur.TempleID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.ErrInvalidIssuingBody
		}
		cur.IssuingBodyID = *p.IssuingBodyID
	}

	updated, err := u.announcements.Update(ctx, *cur)
	if err != nil {
		return nil, err
	}
	return u.withState(updated, now), nil
}

// Revoke sets revoked_at once; revoking again returns the original (TA25).
func (u *TempleAnnouncementUsecase) Revoke(ctx context.Context, id string) (*domain.Announcement, error) {
	now := u.now()
	a, err := u.announcements.Revoke(ctx, id, now)
	if err != nil {
		return nil, err
	}
	return u.withState(a, now), nil
}

// List returns one page of the temple's announcements in any state, newest
// first. pageSize 0 means the configured default; larger than the configured
// max is capped.
func (u *TempleAnnouncementUsecase) List(ctx context.Context, templeID string, page, pageSize int) (*domain.AnnouncementPage, error) {
	if page < 1 || pageSize < 0 {
		return nil, domain.ErrInvalidPage
	}
	if pageSize == 0 {
		pageSize = u.cfg.PageSize
	}
	if pageSize > u.cfg.MaxPageSize {
		pageSize = u.cfg.MaxPageSize
	}
	if err := u.requireTemple(ctx, templeID); err != nil {
		return nil, err
	}
	items, total, err := u.announcements.ListByTemple(ctx, templeID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.Announcement{}
	}
	now := u.now()
	for i := range items {
		items[i].State = items[i].StateAt(now)
	}
	return &domain.AnnouncementPage{
		Items: items, Page: page, PageSize: pageSize, Total: total,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// CreateIssuingBody adds a named issuing body to the temple. A duplicate name
// (any case) is domain.ErrIssuingBodyExists from the repository's unique index.
func (u *TempleAnnouncementUsecase) CreateIssuingBody(ctx context.Context, templeID, name string) (*domain.IssuingBody, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return nil, &domain.AnnouncementInputError{Field: "name", Message: "name is required", Kind: domain.ErrInvalidIssuingBodyName}
	case utf8.RuneCountInString(name) > u.cfg.IssuingBodyNameMaxChars:
		return nil, &domain.AnnouncementInputError{Field: "name", Message: "name is too long", Kind: domain.ErrInvalidIssuingBodyName}
	}
	if err := u.requireTemple(ctx, templeID); err != nil {
		return nil, err
	}
	return u.bodies.Insert(ctx, templeID, name)
}

// ListIssuingBodies returns the temple's issuing bodies.
func (u *TempleAnnouncementUsecase) ListIssuingBodies(ctx context.Context, templeID string) ([]domain.IssuingBody, error) {
	if err := u.requireTemple(ctx, templeID); err != nil {
		return nil, err
	}
	items, err := u.bodies.ListByTemple(ctx, templeID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.IssuingBody{}
	}
	return items, nil
}

func (u *TempleAnnouncementUsecase) requireTemple(ctx context.Context, templeID string) error {
	ok, err := u.announcements.TempleExists(ctx, templeID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrTempleNotFound
	}
	return nil
}
