package usecase

import (
	"context"
	"fmt"
	"math"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// DharmaTalkUsecase serves the read-only pháp thoại catalogue: a paged list and
// a detail lookup, plus the one-time first-run seed. Which talks are visible
// (needs_review) and their order live in the repository contract
// (domain.DharmaTalkRepository); this layer owns paging arithmetic and the page
// size it is given by config.
type DharmaTalkUsecase struct {
	talks domain.DharmaTalkRepository
	cfg   config.DharmaTalkConfig
}

// NewDharmaTalkUsecase wires the use case.
func NewDharmaTalkUsecase(talks domain.DharmaTalkRepository, cfg config.DharmaTalkConfig) *DharmaTalkUsecase {
	return &DharmaTalkUsecase{talks: talks, cfg: cfg}
}

// List returns one 1-based page of cfg.PageSize talks. A page past the end is a
// valid, empty page. page < 1, or so large that the offset would overflow, is
// domain.ErrInvalidPage.
func (u *DharmaTalkUsecase) List(ctx context.Context, page int) (*domain.DharmaTalkPage, error) {
	size := u.cfg.PageSize
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	items, total, err := u.talks.List(ctx, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list dharma talks: %w", err)
	}
	if items == nil {
		items = []domain.DharmaTalk{}
	}
	return &domain.DharmaTalkPage{
		Items: items, Page: page, PageSize: size, Total: total,
		TotalPages: (total + size - 1) / size,
	}, nil
}

// Get returns one talk by id, or domain.ErrDharmaTalkNotFound.
func (u *DharmaTalkUsecase) Get(ctx context.Context, id string) (*domain.DharmaTalk, error) {
	t, err := u.talks.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dharma talk %q: %w", id, err)
	}
	return t, nil
}

// SeedIfEmpty loads the seed talks on the service's first run: only when the
// table holds no row at all — hidden (needs_review) rows count, so a table whose
// every talk is under review is not mistaken for empty and reseeded over the
// reviewers' decisions. load is called only when seeding is needed. It returns
// how many rows were inserted (0 when the table already had data).
func (u *DharmaTalkUsecase) SeedIfEmpty(ctx context.Context, load func() ([]domain.DharmaTalk, error)) (int, error) {
	n, err := u.talks.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count dharma talks: %w", err)
	}
	if n > 0 {
		return 0, nil
	}
	seed, err := load()
	if err != nil {
		return 0, fmt.Errorf("load dharma talk seed: %w", err)
	}
	inserted, err := u.talks.InsertMissing(ctx, seed)
	if err != nil {
		return 0, fmt.Errorf("seed dharma talks: %w", err)
	}
	return inserted, nil
}
