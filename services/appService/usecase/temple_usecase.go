package usecase

import (
	"context"
	"fmt"
	"math"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// TempleUsecase serves the read-only temple catalogue: a short "top" list, a
// paged list and a detail lookup. Ranking lives in the repository contract
// (domain.TempleRepository); this layer owns paging arithmetic and the page
// sizes it is given by config.
type TempleUsecase struct {
	temples domain.TempleRepository
	cfg     config.TempleConfig
}

// NewTempleUsecase wires the use case. cfg is the whole TempleConfig so a new
// tunable does not change the constructor.
func NewTempleUsecase(temples domain.TempleRepository, cfg config.TempleConfig) *TempleUsecase {
	return &TempleUsecase{temples: temples, cfg: cfg}
}

// Top returns the highest-ranked temples, cfg.TopLimit of them.
func (u *TempleUsecase) Top(ctx context.Context) ([]domain.Temple, error) {
	items, err := u.temples.ListTop(ctx, u.cfg.TopLimit)
	if err != nil {
		return nil, fmt.Errorf("list top temples: %w", err)
	}
	return items, nil
}

// List returns one 1-based page of cfg.PageSize temples. A page past the end
// is a valid, empty page (Total still reports the real count) so a client that
// pages until it sees an empty page terminates cleanly. page < 1, or so large
// that the offset would overflow, is ErrInvalidPage.
func (u *TempleUsecase) List(ctx context.Context, page int) (*domain.TemplePage, error) {
	size := u.cfg.PageSize
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	items, total, err := u.temples.List(ctx, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list temples: %w", err)
	}
	if items == nil {
		items = []domain.Temple{}
	}
	return &domain.TemplePage{
		Items:      items,
		Page:       page,
		PageSize:   size,
		Total:      total,
		TotalPages: (total + size - 1) / size,
	}, nil
}

// Get returns one temple by id, or domain.ErrTempleNotFound.
func (u *TempleUsecase) Get(ctx context.Context, id string) (*domain.Temple, error) {
	t, err := u.temples.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get temple %q: %w", id, err)
	}
	return t, nil
}
