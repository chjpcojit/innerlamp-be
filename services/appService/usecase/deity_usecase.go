package usecase

import (
	"context"
	"fmt"
	"math"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// DeityUsecase serves the read-only Phật / Bồ tát catalogue: a short "top"
// list across both types, a paged list of all deities and a detail lookup.
// Ordering lives in the repository contract (domain.DeityRepository); this
// layer owns paging arithmetic and the sizes it is given by config.
type DeityUsecase struct {
	deities domain.DeityRepository
	cfg     config.DeityConfig
}

// NewDeityUsecase wires the use case.
func NewDeityUsecase(deities domain.DeityRepository, cfg config.DeityConfig) *DeityUsecase {
	return &DeityUsecase{deities: deities, cfg: cfg}
}

// Top returns the first cfg.TopLimit deities (Buddhas and bodhisattvas).
func (u *DeityUsecase) Top(ctx context.Context) ([]domain.Deity, error) {
	items, err := u.deities.ListTop(ctx, u.cfg.TopLimit)
	if err != nil {
		return nil, fmt.Errorf("list top deities: %w", err)
	}
	return items, nil
}

// ListDeities returns one 1-based page of cfg.PageSize deities, Buddhas and
// bodhisattvas together. A page past the end is a valid, empty page. page < 1,
// or so large that the offset would overflow, is domain.ErrInvalidPage.
func (u *DeityUsecase) ListDeities(ctx context.Context, page int) (*domain.DeityPage, error) {
	size := u.cfg.PageSize
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	items, total, err := u.deities.List(ctx, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list deities: %w", err)
	}
	if items == nil {
		items = []domain.Deity{}
	}
	return &domain.DeityPage{
		Items: items, Page: page, PageSize: size, Total: total,
		TotalPages: (total + size - 1) / size,
	}, nil
}

// Get returns one deity (Buddha or bodhisattva) by id, or domain.ErrDeityNotFound.
func (u *DeityUsecase) Get(ctx context.Context, id string) (*domain.Deity, error) {
	d, err := u.deities.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get deity %q: %w", id, err)
	}
	return d, nil
}
