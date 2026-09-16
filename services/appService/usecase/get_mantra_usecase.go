package usecase

import (
	"context"
	"errors"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GetMantraUsecase implements US-008 (Xem chi tiết một thần chú).
type GetMantraUsecase struct {
	mantras      domain.MantraRepository
	bodhisattvas domain.BodhisattvaRepository
	logger       logging.Logger
}

// NewGetMantraUsecase wires the get-mantra use-case with its dependencies.
func NewGetMantraUsecase(mantras domain.MantraRepository, bodhisattvas domain.BodhisattvaRepository, logger logging.Logger) *GetMantraUsecase {
	return &GetMantraUsecase{mantras: mantras, bodhisattvas: bodhisattvas, logger: logger}
}

// GetMantraBySlug returns the full mantra for slug plus its related bodhisattva —
// AC1 (title/body/sect/bodhisattva info), AC2 (audio_url present or empty, both handled uniformly
// by the client). Serving cache first when offline (AC3) is a client-side concern.
func (uc *GetMantraUsecase) GetMantraBySlug(ctx context.Context, slug string) (*domain.Mantra, *domain.Bodhisattva, error) {
	mantra, err := uc.mantras.GetMantraBySlug(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	var bodhisattva *domain.Bodhisattva
	if mantra.BodhisattvaID != "" && uc.bodhisattvas != nil {
		b, err := uc.bodhisattvas.GetBodhisattvaByID(ctx, mantra.BodhisattvaID)
		if err != nil {
			if !errors.Is(err, domain.ErrBodhisattvaNotFound) {
				uc.logger.Warn("failed to fetch related bodhisattva for mantra", "mantra_id", mantra.ID, "bodhisattva_id", mantra.BodhisattvaID, "error", err)
			}
		} else {
			bodhisattva = b
		}
	}

	return mantra, bodhisattva, nil
}
