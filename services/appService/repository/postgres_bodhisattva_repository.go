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

type bodhisattvaRepository struct {
	db database.Database
}

// NewBodhisattvaRepository returns a PostgreSQL-backed domain.BodhisattvaRepository.
func NewBodhisattvaRepository(db database.Database) domain.BodhisattvaRepository {
	return &bodhisattvaRepository{db: db}
}

// ListBodhisattvas returns bodhisattvas matching input, ordered by
// updated_at ascending — US-009 AC1/AC2/AC2b/AC4.
func (r *bodhisattvaRepository) ListBodhisattvas(ctx context.Context, input domain.ListBodhisattvasInput) ([]*domain.Bodhisattva, error) {
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

	query := `SELECT id, slug, name, sect, description, image_url, version, created_at, updated_at FROM bodhisattvas`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
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
		var sect, imageURL sql.NullString
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &sect, &b.Description, &imageURL, &b.Version, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bodhisattva: %w", err)
		}
		b.Sect = sect.String
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
	var sect, imageURL sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, slug, name, sect, description, image_url, version, created_at, updated_at
		 FROM bodhisattvas WHERE slug = $1`,
		slug,
	)
	err := row.Scan(&b.ID, &b.Slug, &b.Name, &sect, &b.Description, &imageURL, &b.Version, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBodhisattvaNotFound
		}
		return nil, fmt.Errorf("get bodhisattva by slug: %w", err)
	}
	b.Sect = sect.String
	b.ImageURL = imageURL.String

	return b, nil
}

// GetBodhisattvaByID returns the bodhisattva for id — US-008 AC1.
func (r *bodhisattvaRepository) GetBodhisattvaByID(ctx context.Context, id string) (*domain.Bodhisattva, error) {
	b := &domain.Bodhisattva{}
	var sect, imageURL sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, slug, name, sect, description, image_url, version, created_at, updated_at
		 FROM bodhisattvas WHERE id = $1`,
		id,
	)
	err := row.Scan(&b.ID, &b.Slug, &b.Name, &sect, &b.Description, &imageURL, &b.Version, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBodhisattvaNotFound
		}
		return nil, fmt.Errorf("get bodhisattva by id: %w", err)
	}
	b.Sect = sect.String
	b.ImageURL = imageURL.String

	return b, nil
}
