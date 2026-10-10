package usecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

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

// maxTempleQueryRunes caps the search term; no temple name is longer than the
// 255-character column and a pick-list search never needs more than this.
const maxTempleQueryRunes = 100

// Options returns one 1-based page of cfg.OptionsPageSize temple options
// (id, name, province) in ranking order — the registration pick-list. A nil
// query lists everything; a present query keeps the temples whose name contains
// it (case- and diacritic-insensitive, done by the repository). It follows
// List's paging rules: a page past the end is a valid empty page, page < 1 or
// above MaxInt32 is ErrInvalidPage; a present-but-empty or over-long query is
// ErrInvalidTempleQuery.
func (u *TempleUsecase) Options(ctx context.Context, query *string, page int) (*domain.TempleOptionPage, error) {
	size := u.cfg.OptionsPageSize
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	term := ""
	if query != nil {
		term = strings.TrimSpace(*query)
		if term == "" || utf8.RuneCountInString(term) > maxTempleQueryRunes {
			return nil, domain.ErrInvalidTempleQuery
		}
	}
	items, total, err := u.temples.ListOptions(ctx, term, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list temple options: %w", err)
	}
	if items == nil {
		items = []domain.TempleOption{}
	}
	return &domain.TempleOptionPage{
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
