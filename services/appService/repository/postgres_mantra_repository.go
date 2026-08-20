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

type mantraRepository struct {
	db database.Database
}

// NewMantraRepository returns a PostgreSQL-backed domain.MantraRepository.
func NewMantraRepository(db database.Database) domain.MantraRepository {
	return &mantraRepository{db: db}
}

// ListMantras returns mantras matching input, ordered by updated_at
// ascending — US-007 AC1/AC2/AC3/AC5.
func (r *mantraRepository) ListMantras(ctx context.Context, input domain.ListMantrasInput) ([]*domain.Mantra, error) {
	var conditions []string
	var args []any

	if !input.UpdatedSince.IsZero() {
		args = append(args, input.UpdatedSince)
		conditions = append(conditions, fmt.Sprintf("updated_at > $%d", len(args)))
	}
	if input.BodhisattvaID != "" {
		args = append(args, input.BodhisattvaID)
		conditions = append(conditions, fmt.Sprintf("bodhisattva_id = $%d", len(args)))
	}

	query := `SELECT id, slug, title, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY updated_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mantras: %w", err)
	}
	defer rows.Close()

	mantras := []*domain.Mantra{}
	for rows.Next() {
		m := &domain.Mantra{}
		var audioURL, bodhisattvaID sql.NullString
		if err := rows.Scan(&m.ID, &m.Slug, &m.Title, &m.Body, &audioURL, &bodhisattvaID, &m.Version, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan mantra: %w", err)
		}
		m.AudioURL = audioURL.String
		m.BodhisattvaID = bodhisattvaID.String
		mantras = append(mantras, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mantras: %w", err)
	}

	return mantras, nil
}

// GetMantraBySlug returns the full mantra for slug — US-008 AC1.
func (r *mantraRepository) GetMantraBySlug(ctx context.Context, slug string) (*domain.Mantra, error) {
	m := &domain.Mantra{}
	var audioURL, bodhisattvaID sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, slug, title, body, audio_url, bodhisattva_id, version, created_at, updated_at
		 FROM mantras WHERE slug = $1`,
		slug,
	)
	err := row.Scan(&m.ID, &m.Slug, &m.Title, &m.Body, &audioURL, &bodhisattvaID, &m.Version, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMantraNotFound
		}
		return nil, fmt.Errorf("get mantra by slug: %w", err)
	}
	m.AudioURL = audioURL.String
	m.BodhisattvaID = bodhisattvaID.String

	return m, nil
}
