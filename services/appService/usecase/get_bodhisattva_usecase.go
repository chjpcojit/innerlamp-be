package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GetBodhisattvaUsecase implements US-010 (Xem chi tiết một vị Phật/Bồ Tát).
type GetBodhisattvaUsecase struct {
	bodhisattvas domain.BodhisattvaRepository
	mantras      domain.MantraRepository
	logger       logging.Logger
}

// NewGetBodhisattvaUsecase wires the get-bodhisattva use-case with its dependencies.
func NewGetBodhisattvaUsecase(bodhisattvas domain.BodhisattvaRepository, mantras domain.MantraRepository, logger logging.Logger) *GetBodhisattvaUsecase {
	return &GetBodhisattvaUsecase{bodhisattvas: bodhisattvas, mantras: mantras, logger: logger}
}

// GetBodhisattvaBySlug returns the bodhisattva plus its related mantras —
// AC1 (name/description/image + related mantras). AC2 (no related mantras
// isn't an error, just an empty list — the handler decides how to render
// that) holds because ListMantras always returns a non-nil slice on no
// rows. AC3 (navigating to a mantra's detail) is a client-side concern.
// Related mantras are fetched via MantraRepository rather than
// BodhisattvaRepository doing a join, so a not-found bodhisattva short-
// circuits before any mantra lookup happens.
func (uc *GetBodhisattvaUsecase) GetBodhisattvaBySlug(ctx context.Context, slug string) (*domain.Bodhisattva, []*domain.Mantra, error) {
	bodhisattva, err := uc.bodhisattvas.GetBodhisattvaBySlug(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	mantras, err := uc.mantras.ListMantras(ctx, domain.ListMantrasInput{BodhisattvaID: bodhisattva.ID})
	if err != nil {
		return nil, nil, err
	}

	return bodhisattva, mantras, nil
}
