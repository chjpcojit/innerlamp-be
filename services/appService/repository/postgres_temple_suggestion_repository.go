package repository

import (
	"context"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type templeSuggestionRepository struct {
	db database.Database
}

// NewTempleSuggestionRepository returns the Postgres-backed
// domain.TempleSuggestionRepository.
func NewTempleSuggestionRepository(db database.Database) domain.TempleSuggestionRepository {
	return &templeSuggestionRepository{db: db}
}

// Create inserts the suggestion as pending. status and the review columns take
// their column defaults, so the row always satisfies chk_temple_suggestions_review.
func (r *templeSuggestionRepository) Create(ctx context.Context, s *domain.TempleSuggestion) (*domain.TempleSuggestion, error) {
	var out domain.TempleSuggestion
	err := r.db.QueryRow(ctx, `
		INSERT INTO temple_suggestions (name, province, address)
		VALUES ($1, $2, $3)
		RETURNING id, name, province, address, status, created_at`,
		s.Name, s.Province, s.Address,
	).Scan(&out.ID, &out.Name, &out.Province, &out.Address, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert temple suggestion: %w", err)
	}
	return &out, nil
}
