package http

import (
	"context"
	"errors"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// getSutraUsecase is the subset of *usecase.GetSutraUsecase this handler
// depends on, defined here (the consumer) so it can be unit tested with a
// mock instead of the real use-case.
type getSutraUsecase interface {
	GetSutraBySlug(ctx context.Context, slug string) (*domain.Sutra, error)
}

// SutraDetailHandler implements GET /v1/content/sutras/{slug} (US-006).
// Public read — no authentication required.
type SutraDetailHandler struct {
	usecase getSutraUsecase
}

// NewSutraDetailHandler wires the HTTP handler to its use-case.
func NewSutraDetailHandler(uc getSutraUsecase) *SutraDetailHandler {
	return &SutraDetailHandler{usecase: uc}
}

func (h *SutraDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		utils.WriteValidationError(w, "MISSING_SLUG", "slug is required", map[string]any{"field": "slug"})
		return
	}

	sutra, err := h.usecase.GetSutraBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, domain.ErrSutraNotFound) {
			utils.WriteBusinessError(w, http.StatusNotFound, "SUTRA_NOT_FOUND", "sutra not found", nil)
			return
		}
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	utils.WriteSuccessResponse(w, newSutraResponse(sutra))
}
