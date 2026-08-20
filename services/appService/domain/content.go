package domain

import (
	"context"
	"time"
)

// Sutra maps to the sutras table (docs/db/content-erd.md).
type Sutra struct {
	ID        string
	Slug      string
	Title     string
	Sect      string
	Language  string
	Body      string
	AudioURL  string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListSutrasInput is the use-case input for US-005. UpdatedSince zero-valued
// means a full sync — AC2 (first open, no cache yet); a non-zero value
// requests only sutras changed since then — AC3/AC6 (background delta sync
// against an existing cache). Sect empty means no filter; non-empty applies
// the tông phái filter — AC5.
type ListSutrasInput struct {
	UpdatedSince time.Time
	Sect         string
}

// SutraRepository lists and looks up sutras — US-005/US-006.
type SutraRepository interface {
	// ListSutras returns sutras matching input, ordered by updated_at
	// ascending so a delta-sync caller can persist the last row's
	// updated_at as its next sync cursor — AC2/AC3/AC5/AC6.
	ListSutras(ctx context.Context, input ListSutrasInput) ([]*Sutra, error)

	// GetSutraBySlug returns the full sutra (including body) for slug —
	// US-006 AC1. Always returns the current server version (AC4 — the
	// client decides cache-vs-refetch itself by comparing local
	// version/updated_at before calling this). Returns ErrSutraNotFound if
	// no sutra has that slug.
	GetSutraBySlug(ctx context.Context, slug string) (*Sutra, error)
}

// Mantra maps to the mantras table (docs/db/content-erd.md). BodhisattvaID
// is empty when the mantra isn't gắn với any vị — AC2 of US-007/US-008.
type Mantra struct {
	ID            string
	Slug          string
	Title         string
	Body          string
	AudioURL      string
	BodhisattvaID string
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ListMantrasInput is the use-case input for US-007. UpdatedSince zero-valued
// means a full sync, mirroring ListSutrasInput. BodhisattvaID empty means no
// filter; non-empty applies the "thần chú của vị này" filter — AC3 — and is
// reused as-is by GetBodhisattvaUsecase (US-010) to fetch a vị's related
// mantras.
type ListMantrasInput struct {
	UpdatedSince  time.Time
	BodhisattvaID string
}

// MantraRepository lists and looks up mantras — US-007/US-008.
type MantraRepository interface {
	// ListMantras returns mantras matching input, ordered by updated_at
	// ascending — same delta-sync convention as SutraRepository.ListSutras.
	ListMantras(ctx context.Context, input ListMantrasInput) ([]*Mantra, error)

	// GetMantraBySlug returns the full mantra (including body) for slug —
	// US-008 AC1. Returns ErrMantraNotFound if no mantra has that slug.
	GetMantraBySlug(ctx context.Context, slug string) (*Mantra, error)
}

// Bodhisattva maps to the bodhisattvas table (docs/db/content-erd.md).
type Bodhisattva struct {
	ID          string
	Slug        string
	Name        string
	Description string
	ImageURL    string
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListBodhisattvasInput is the use-case input for US-009. UpdatedSince
// zero-valued means a full sync, mirroring ListSutrasInput.
type ListBodhisattvasInput struct {
	UpdatedSince time.Time
}

// BodhisattvaRepository lists and looks up bodhisattvas — US-009/US-010.
type BodhisattvaRepository interface {
	// ListBodhisattvas returns bodhisattvas matching input, ordered by
	// updated_at ascending — same delta-sync convention as
	// SutraRepository.ListSutras.
	ListBodhisattvas(ctx context.Context, input ListBodhisattvasInput) ([]*Bodhisattva, error)

	// GetBodhisattvaBySlug returns the bodhisattva for slug — US-010 AC1.
	// Related mantras are fetched separately via
	// MantraRepository.ListMantras(BodhisattvaID: ...) rather than joined
	// here, keeping this repository single-table. Returns
	// ErrBodhisattvaNotFound if no bodhisattva has that slug.
	GetBodhisattvaBySlug(ctx context.Context, slug string) (*Bodhisattva, error)
}
