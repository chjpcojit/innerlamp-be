package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type sutraRepository struct {
	db database.Database
}

// NewSutraRepository returns a PostgreSQL-backed domain.SutraRepository.
func NewSutraRepository(db database.Database) domain.SutraRepository {
	return &sutraRepository{db: db}
}

// ListSutras returns sutras matching input, ordered by updated_at ascending
// — US-005 AC1/AC2/AC3/AC5/AC6. UpdatedSince and Sect are optional filters;
// either, both, or neither may be applied depending on the query.
func (r *sutraRepository) ListSutras(ctx context.Context, input domain.ListSutrasInput) ([]*domain.Sutra, error) {
	var conditions []string
	var args []any

	if !input.UpdatedSince.IsZero() {
		args = append(args, input.UpdatedSince)
		conditions = append(conditions, fmt.Sprintf("updated_at > $%d", len(args)))
	}
	if input.Sect != "" {
		args = append(args, input.Sect)
		conditions = append(conditions, fmt.Sprintf("sect = $%d", len(args)))
	}

	query := `SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY updated_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sutras: %w", err)
	}
	defer rows.Close()

	sutras := []*domain.Sutra{}
	for rows.Next() {
		s := &domain.Sutra{}
		var audioURL sql.NullString
		if err := rows.Scan(&s.ID, &s.Slug, &s.Title, &s.Sect, &s.Language, &s.Body, &audioURL, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sutra: %w", err)
		}
		s.AudioURL = audioURL.String
		sutras = append(sutras, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sutras: %w", err)
	}

	return sutras, nil
}

// GetSutraBySlug returns the full sutra for slug — US-006 AC1.
func (r *sutraRepository) GetSutraBySlug(ctx context.Context, slug string) (*domain.Sutra, error) {
	s := &domain.Sutra{}
	var audioURL sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at
		 FROM sutras WHERE slug = $1`,
		slug,
	)
	err := row.Scan(&s.ID, &s.Slug, &s.Title, &s.Sect, &s.Language, &s.Body, &audioURL, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSutraNotFound
		}
		return nil, fmt.Errorf("get sutra by slug: %w", err)
	}
	s.AudioURL = audioURL.String

	return s, nil
}
