package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// ListMantrasUsecase implements US-007 (Xem danh sách thần chú).
type ListMantrasUsecase struct {
	mantras domain.MantraRepository
	logger  logging.Logger
}

// NewListMantrasUsecase wires the list-mantras use-case with its dependencies.
func NewListMantrasUsecase(mantras domain.MantraRepository, logger logging.Logger) *ListMantrasUsecase {
	return &ListMantrasUsecase{mantras: mantras, logger: logger}
}

// ListMantras returns mantras matching input — AC1 (list), AC2 (no
// bodhisattva_id passes through without error), AC3 (bodhisattva filter),
// AC5 (delta sync when input.UpdatedSince is non-zero). Serving cache first
// and tolerating offline (AC4) is entirely a client-side concern.
func (uc *ListMantrasUsecase) ListMantras(ctx context.Context, input domain.ListMantrasInput) ([]*domain.Mantra, error) {
	return uc.mantras.ListMantras(ctx, input)
}
