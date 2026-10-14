package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
)

type templeAnnouncementRepository struct {
	db database.Database
}

// NewTempleAnnouncementRepository returns the Postgres-backed
// domain.AnnouncementRepository.
func NewTempleAnnouncementRepository(db database.Database) domain.AnnouncementRepository {
	return &templeAnnouncementRepository{db: db}
}

type issuingBodyRepository struct {
	db database.Database
}

// NewIssuingBodyRepository returns the Postgres-backed domain.IssuingBodyRepository.
func NewIssuingBodyRepository(db database.Database) domain.IssuingBodyRepository {
	return &issuingBodyRepository{db: db}
}

const announcementSelect = `SELECT a.id, a.temple_id, a.issuing_body_id, b.name, a.title, a.body, a.source,
	a.published_at, a.expires_at, a.revoked_at, a.created_by
	FROM temple_announcements a JOIN temple_issuing_bodies b ON b.id = a.issuing_body_id`

func scanAnnouncement(row rowScanner) (*domain.Announcement, error) {
	var a domain.Announcement
	var revokedAt sql.NullTime
	var createdBy sql.NullString
	if err := row.Scan(&a.ID, &a.TempleID, &a.IssuingBodyID, &a.IssuingBodyName, &a.Title, &a.Body, &a.Source,
		&a.PublishedAt, &a.ExpiresAt, &revokedAt, &createdBy); err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		a.RevokedAt = &revokedAt.Time
	}
	a.CreatedBy = nullStr(createdBy)
	return &a, nil
}

func pgCode(err error) string {
	if pe, ok := errors.AsType[*pq.Error](err); ok {
		return string(pe.Code)
	}
	return ""
}

func (r *templeAnnouncementRepository) TempleExists(ctx context.Context, templeID string) (bool, error) {
	var ok bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM temples WHERE id = $1)`, templeID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check temple exists: %w", err)
	}
	return ok, nil
}

func (r *templeAnnouncementRepository) IssuingBodyBelongs(ctx context.Context, issuingBodyID, templeID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM temple_issuing_bodies WHERE id = $1 AND temple_id = $2)`,
		issuingBodyID, templeID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check issuing body: %w", err)
	}
	return ok, nil
}

func (r *templeAnnouncementRepository) Insert(ctx context.Context, a domain.Announcement) (*domain.Announcement, error) {
	id := "tann_" + uuid.NewString()
	_, err := r.db.Exec(ctx, `
		INSERT INTO temple_announcements (id, temple_id, issuing_body_id, title, body, source, published_at, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, a.TempleID, a.IssuingBodyID, a.Title, a.Body, a.Source, a.PublishedAt, a.ExpiresAt, a.CreatedBy)
	if err != nil {
		// The temple was deleted between the existence check and the insert.
		if pgCode(err) == pgForeignKeyViolation {
			return nil, domain.ErrTempleNotFound
		}
		return nil, fmt.Errorf("insert temple announcement: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *templeAnnouncementRepository) FindByID(ctx context.Context, id string) (*domain.Announcement, error) {
	a, err := scanAnnouncement(r.db.QueryRow(ctx, announcementSelect+` WHERE a.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAnnouncementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find temple announcement: %w", err)
	}
	return a, nil
}

func (r *templeAnnouncementRepository) Update(ctx context.Context, a domain.Announcement) (*domain.Announcement, error) {
	res, err := r.db.Exec(ctx, `
		UPDATE temple_announcements
		SET title = $2, body = $3, issuing_body_id = $4, expires_at = $5, updated_at = now()
		WHERE id = $1`, a.ID, a.Title, a.Body, a.IssuingBodyID, a.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("update temple announcement: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return nil, domain.ErrAnnouncementNotFound
	}
	return r.FindByID(ctx, a.ID)
}

func (r *templeAnnouncementRepository) Revoke(ctx context.Context, id string, at time.Time) (*domain.Announcement, error) {
	res, err := r.db.Exec(ctx, `
		UPDATE temple_announcements
		SET revoked_at = COALESCE(revoked_at, $2), updated_at = CASE WHEN revoked_at IS NULL THEN now() ELSE updated_at END
		WHERE id = $1`, id, at)
	if err != nil {
		return nil, fmt.Errorf("revoke temple announcement: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return nil, domain.ErrAnnouncementNotFound
	}
	return r.FindByID(ctx, id)
}

// ListByTemple: count and page are two statements; a row added between them
// can make total one ahead of the items, which an admin listing tolerates.
func (r *templeAnnouncementRepository) ListByTemple(ctx context.Context, templeID string, offset, limit int) ([]domain.Announcement, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM temple_announcements WHERE temple_id = $1`, templeID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count temple announcements: %w", err)
	}
	rows, err := r.db.Query(ctx,
		announcementSelect+` WHERE a.temple_id = $1 ORDER BY a.published_at DESC, a.id LIMIT $2 OFFSET $3`,
		templeID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list temple announcements: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := []domain.Announcement{}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan temple announcement: %w", err)
		}
		items = append(items, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list temple announcements: %w", err)
	}
	return items, total, nil
}

func (r *issuingBodyRepository) Insert(ctx context.Context, templeID, name string) (*domain.IssuingBody, error) {
	b := domain.IssuingBody{ID: "tib_" + uuid.NewString(), TempleID: templeID, Name: name}
	err := r.db.QueryRow(ctx,
		`INSERT INTO temple_issuing_bodies (id, temple_id, name) VALUES ($1, $2, $3) RETURNING created_at`,
		b.ID, templeID, name).Scan(&b.CreatedAt)
	switch pgCode(err) {
	case "":
	case pgUniqueViolation:
		return nil, domain.ErrIssuingBodyExists
	case pgForeignKeyViolation:
		return nil, domain.ErrTempleNotFound
	default:
		return nil, fmt.Errorf("insert issuing body: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("insert issuing body: %w", err)
	}
	return &b, nil
}

func (r *issuingBodyRepository) ListByTemple(ctx context.Context, templeID string) ([]domain.IssuingBody, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, temple_id, name, created_at FROM temple_issuing_bodies WHERE temple_id = $1 ORDER BY name, id`, templeID)
	if err != nil {
		return nil, fmt.Errorf("list issuing bodies: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := []domain.IssuingBody{}
	for rows.Next() {
		var b domain.IssuingBody
		if err := rows.Scan(&b.ID, &b.TempleID, &b.Name, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan issuing body: %w", err)
		}
		items = append(items, b)
	}
	return items, rows.Err()
}
