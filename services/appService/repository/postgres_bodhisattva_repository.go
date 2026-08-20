package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type bodhisattvaRepository struct {
	db database.Database
}

// NewBodhisattvaRepository returns a PostgreSQL-backed domain.BodhisattvaRepository.
func NewBodhisattvaRepository(db database.Database) domain.BodhisattvaRepository {
	return &bodhisattvaRepository{db: db}
}

// ListBodhisattvas returns bodhisattvas matching input, ordered by
// updated_at ascending — US-009 AC1/AC2/AC4.
func (r *bodhisattvaRepository) ListBodhisattvas(ctx context.Context, input domain.ListBodhisattvasInput) ([]*domain.Bodhisattva, error) {
	query := `SELECT id, slug, name, description, image_url, version, created_at, updated_at FROM bodhisattvas`
	var args []any
	if !input.UpdatedSince.IsZero() {
		args = append(args, input.UpdatedSince)
		query += " WHERE updated_at > $1"
	}
	query += " ORDER BY updated_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list bodhisattvas: %w", err)
	}
	defer rows.Close()

	bodhisattvas := []*domain.Bodhisattva{}
	for rows.Next() {
		b := &domain.Bodhisattva{}
		var imageURL sql.NullString
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &b.Description, &imageURL, &b.Version, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bodhisattva: %w", err)
		}
		b.ImageURL = imageURL.String
		bodhisattvas = append(bodhisattvas, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bodhisattvas: %w", err)
	}

	return bodhisattvas, nil
}

// GetBodhisattvaBySlug returns the bodhisattva for slug — US-010 AC1.
func (r *bodhisattvaRepository) GetBodhisattvaBySlug(ctx context.Context, slug string) (*domain.Bodhisattva, error) {
	b := &domain.Bodhisattva{}
	var imageURL sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, slug, name, description, image_url, version, created_at, updated_at
		 FROM bodhisattvas WHERE slug = $1`,
		slug,
	)
	err := row.Scan(&b.ID, &b.Slug, &b.Name, &b.Description, &imageURL, &b.Version, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBodhisattvaNotFound
		}
		return nil, fmt.Errorf("get bodhisattva by slug: %w", err)
	}
	b.ImageURL = imageURL.String

	return b, nil
}
