package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GetMantraUsecase implements US-008 (Xem chi tiết một thần chú).
type GetMantraUsecase struct {
	mantras domain.MantraRepository
	logger  logging.Logger
}

// NewGetMantraUsecase wires the get-mantra use-case with its dependencies.
func NewGetMantraUsecase(mantras domain.MantraRepository, logger logging.Logger) *GetMantraUsecase {
	return &GetMantraUsecase{mantras: mantras, logger: logger}
}

// GetMantraBySlug returns the full mantra for slug — AC1 (title/body/
// bodhisattva_id), AC2 (audio_url present or empty, both handled uniformly
// by the client). Serving cache first when offline (AC3) is a client-side
// concern.
func (uc *GetMantraUsecase) GetMantraBySlug(ctx context.Context, slug string) (*domain.Mantra, error) {
	return uc.mantras.GetMantraBySlug(ctx, slug)
}
