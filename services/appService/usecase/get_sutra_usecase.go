package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GetSutraUsecase implements US-006 (Xem chi tiết một bài kinh).
type GetSutraUsecase struct {
	sutras domain.SutraRepository
	logger logging.Logger
}

// NewGetSutraUsecase wires the get-sutra use-case with its dependencies.
func NewGetSutraUsecase(sutras domain.SutraRepository, logger logging.Logger) *GetSutraUsecase {
	return &GetSutraUsecase{sutras: sutras, logger: logger}
}

// GetSutraBySlug returns the full sutra for slug — AC1 (title/sect/body),
// AC2 (audio_url present or empty, both handled uniformly by the client).
// AC4 (always the current server version) holds structurally: this always
// hits the repository, never a server-side cache, so there is no stale
// version to accidentally serve — deciding whether to call it at all
// instead of using a local cache (AC3/AC4) is the client's job.
func (uc *GetSutraUsecase) GetSutraBySlug(ctx context.Context, slug string) (*domain.Sutra, error) {
	return uc.sutras.GetSutraBySlug(ctx, slug)
}
