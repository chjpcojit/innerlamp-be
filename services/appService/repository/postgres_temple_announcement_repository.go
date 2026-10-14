package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type templeAnnouncementRepository struct {
	db database.Database
}

// NewTempleAnnouncementRepository returns the Postgres-backed
// domain.TempleAnnouncementRepository.
func NewTempleAnnouncementRepository(db database.Database) domain.TempleAnnouncementRepository {
	return &templeAnnouncementRepository{db: db}
}

// activeAnnouncementWhere is the one definition of "active" (TA8). $2 is now.
const activeAnnouncementWhere = `a.temple_id = $1 AND a.revoked_at IS NULL AND a.published_at <= $2 AND a.expires_at > $2`

// ListActiveByTemple counts and then reads one page, newest published_at first,
// id as the tie-break so two reads of the same state give the same order (TA10).
func (r *templeAnnouncementRepository) ListActiveByTemple(ctx context.Context, templeID string, now time.Time, page domain.AnnouncementPageRequest) (domain.AnnouncementPage, error) {
	out := domain.AnnouncementPage{Items: []domain.TempleAnnouncement{}, Page: page.Page, PageSize: page.PageSize}

	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM temple_announcements a WHERE `+activeAnnouncementWhere, templeID, now,
	).Scan(&out.Total); err != nil {
		return out, fmt.Errorf("count active announcements: %w", err)
	}
	if out.Total == 0 {
		return out, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.temple_id, a.issuing_body_id, b.name, a.title, a.body, a.published_at, a.expires_at
		FROM temple_announcements a
		JOIN temple_issuing_bodies b ON b.id = a.issuing_body_id
		WHERE `+activeAnnouncementWhere+`
		ORDER BY a.published_at DESC, a.id
		LIMIT $3 OFFSET $4`,
		templeID, now, page.PageSize, (page.Page-1)*page.PageSize)
	if err != nil {
		return out, fmt.Errorf("query active announcements: %w", err)
	}
	items, err := scanRows(rows, "temple announcement", func(s rowScanner) (domain.TempleAnnouncement, error) {
		var a domain.TempleAnnouncement
		err := s.Scan(&a.ID, &a.TempleID, &a.IssuingBody.ID, &a.IssuingBody.Name, &a.Title, &a.Body, &a.PublishedAt, &a.ExpiresAt)
		return a, err
	})
	if err != nil {
		return out, err
	}
	out.Items = items
	return out, nil
}

type announcementPreferenceRepository struct {
	db database.Database
}

// NewAnnouncementPreferenceRepository returns the Postgres-backed
// domain.AnnouncementPreferenceRepository (the users.receive_temple_announcements column).
func NewAnnouncementPreferenceRepository(db database.Database) domain.AnnouncementPreferenceRepository {
	return &announcementPreferenceRepository{db: db}
}

// GetSettings reads the switch and the current temple in one statement (R-2).
func (r *announcementPreferenceRepository) GetSettings(ctx context.Context, userID int64) (domain.AnnouncementSettings, error) {
	var (
		s            domain.AnnouncementSettings
		templeID, nm sql.NullString
	)
	err := r.db.QueryRow(ctx, `
		SELECT u.receive_temple_announcements, u.ordination_temple_id, t.name
		FROM users u LEFT JOIN temples t ON t.id = u.ordination_temple_id
		WHERE u.id = $1`, userID).Scan(&s.Receive, &templeID, &nm)
	if errors.Is(err, sql.ErrNoRows) {
		return s, domain.ErrUserNotFound
	}
	if err != nil {
		return s, fmt.Errorf("get announcement settings: %w", err)
	}
	if templeID.Valid {
		s.Temple = &domain.TempleRef{ID: templeID.String, Name: nm.String}
	}
	return s, nil
}

// SetReceive writes only the switch column; ordination_temple_id is untouched (TA15).
func (r *announcementPreferenceRepository) SetReceive(ctx context.Context, userID int64, receive bool) error {
	res, err := r.db.Exec(ctx,
		`UPDATE users SET receive_temple_announcements = $2, updated_at = now() WHERE id = $1`, userID, receive)
	if err != nil {
		return fmt.Errorf("set announcement receive: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
