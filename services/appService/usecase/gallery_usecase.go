package usecase

import (
	"context"
	"fmt"
	"math"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GalleryUsecase serves the read-only Buddhist photo gallery: a short "top"
// list, a paged list with an optional type filter and a detail lookup. Ordering
// lives in the repository contract (domain.GalleryRepository); this layer owns
// paging arithmetic, type validation and the sizes it is given by config.
type GalleryUsecase struct {
	gallery domain.GalleryRepository
	cfg     config.GalleryConfig
}

// NewGalleryUsecase wires the use case.
func NewGalleryUsecase(gallery domain.GalleryRepository, cfg config.GalleryConfig) *GalleryUsecase {
	return &GalleryUsecase{gallery: gallery, cfg: cfg}
}

// Top returns the first cfg.TopLimit images.
func (u *GalleryUsecase) Top(ctx context.Context) ([]domain.GalleryImage, error) {
	items, err := u.gallery.ListTop(ctx, u.cfg.TopLimit)
	if err != nil {
		return nil, fmt.Errorf("list top gallery images: %w", err)
	}
	return items, nil
}

// List returns one 1-based page of cfg.PageSize images, optionally filtered to
// one type ("" = every type). A page past the end is a valid, empty page.
// page < 1, or so large that the offset would overflow, is domain.ErrInvalidPage;
// a non-empty imageType that is not a gallery type is domain.ErrInvalidGalleryType.
func (u *GalleryUsecase) List(ctx context.Context, imageType string, page int) (*domain.GalleryPage, error) {
	size := u.cfg.PageSize
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	if imageType != "" && !domain.IsValidGalleryType(imageType) {
		return nil, domain.ErrInvalidGalleryType
	}
	items, total, err := u.gallery.List(ctx, imageType, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list gallery images: %w", err)
	}
	if items == nil {
		items = []domain.GalleryImage{}
	}
	return &domain.GalleryPage{
		Items: items, Page: page, PageSize: size, Total: total,
		TotalPages: (total + size - 1) / size,
	}, nil
}

// Get returns one image by id, or domain.ErrGalleryImageNotFound.
func (u *GalleryUsecase) Get(ctx context.Context, id string) (*domain.GalleryImage, error) {
	g, err := u.gallery.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get gallery image %q: %w", id, err)
	}
	return g, nil
}
