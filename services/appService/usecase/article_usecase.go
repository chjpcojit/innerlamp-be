package usecase

import (
	"context"
	"fmt"
	"math"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// ArticleUsecase serves the read-only Phật pháp article catalogue: a short
// "top" list, a paged list and a detail lookup, plus the one-time first-run
// seed. Ordering lives in the repository contract (domain.ArticleRepository);
// this layer owns paging arithmetic and the sizes it is given by config.
type ArticleUsecase struct {
	articles domain.ArticleRepository
	cfg      config.ArticleConfig
}

// NewArticleUsecase wires the use case.
func NewArticleUsecase(articles domain.ArticleRepository, cfg config.ArticleConfig) *ArticleUsecase {
	return &ArticleUsecase{articles: articles, cfg: cfg}
}

// Top returns the first cfg.TopLimit articles.
func (u *ArticleUsecase) Top(ctx context.Context) ([]domain.Article, error) {
	items, err := u.articles.ListTop(ctx, u.cfg.TopLimit)
	if err != nil {
		return nil, fmt.Errorf("list top articles: %w", err)
	}
	return items, nil
}

// List returns one 1-based page of cfg.PageSize articles. A page past the end
// is a valid, empty page. page < 1, or so large that the offset would overflow,
// is domain.ErrInvalidPage.
func (u *ArticleUsecase) List(ctx context.Context, page int) (*domain.ArticlePage, error) {
	size := u.cfg.PageSize
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	items, total, err := u.articles.List(ctx, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}
	if items == nil {
		items = []domain.Article{}
	}
	return &domain.ArticlePage{
		Items: items, Page: page, PageSize: size, Total: total,
		TotalPages: (total + size - 1) / size,
	}, nil
}

// Get returns one article by id, or domain.ErrArticleNotFound.
func (u *ArticleUsecase) Get(ctx context.Context, id string) (*domain.Article, error) {
	a, err := u.articles.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get article %q: %w", id, err)
	}
	return a, nil
}

// SeedIfEmpty loads the seed articles on the service's first run: only when the
// table holds no article at all, so a restart never resurrects a deleted article
// or overwrites an edited one. load is called only when seeding is needed, so a
// normal start does not parse the seed. It returns how many rows were inserted
// (0 when the table already had data).
func (u *ArticleUsecase) SeedIfEmpty(ctx context.Context, load func() ([]domain.Article, error)) (int, error) {
	n, err := u.articles.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count articles: %w", err)
	}
	if n > 0 {
		return 0, nil
	}
	seed, err := load()
	if err != nil {
		return 0, fmt.Errorf("load article seed: %w", err)
	}
	inserted, err := u.articles.InsertMissing(ctx, seed)
	if err != nil {
		return 0, fmt.Errorf("seed articles: %w", err)
	}
	return inserted, nil
}
