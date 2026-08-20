package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// ListSutrasUsecase implements US-005 (Xem danh sách kinh điển).
type ListSutrasUsecase struct {
	sutras domain.SutraRepository
	logger logging.Logger
}

// NewListSutrasUsecase wires the list-sutras use-case with its dependencies.
func NewListSutrasUsecase(sutras domain.SutraRepository, logger logging.Logger) *ListSutrasUsecase {
	return &ListSutrasUsecase{sutras: sutras, logger: logger}
}

// ListSutras returns sutras matching input — AC1 (list), AC2 (full sync when
// input.UpdatedSince is zero), AC3/AC6 (delta sync when non-zero), AC5
// (sect filter). Serving cache first and tolerating offline (AC3/AC4) are
// entirely a client-side concern (SQLite/Hive cache); this use-case is the
// server half — the "silently call the API to sync in the background" part.
func (uc *ListSutrasUsecase) ListSutras(ctx context.Context, input domain.ListSutrasInput) ([]*domain.Sutra, error) {
	return uc.sutras.ListSutras(ctx, input)
}
