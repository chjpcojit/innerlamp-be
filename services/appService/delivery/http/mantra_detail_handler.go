package http

import (
	"context"
	"errors"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// getMantraUsecase is the subset of *usecase.GetMantraUsecase this handler
// depends on, defined here (the consumer) so it can be unit tested with a
// mock instead of the real use-case.
type getMantraUsecase interface {
	GetMantraBySlug(ctx context.Context, slug string) (*domain.Mantra, error)
}

// MantraDetailHandler implements GET /v1/content/mantras/{slug} (US-008).
// Public read — no authentication required.
type MantraDetailHandler struct {
	usecase getMantraUsecase
}

// NewMantraDetailHandler wires the HTTP handler to its use-case.
func NewMantraDetailHandler(uc getMantraUsecase) *MantraDetailHandler {
	return &MantraDetailHandler{usecase: uc}
}

func (h *MantraDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		utils.WriteValidationError(w, "MISSING_SLUG", "slug is required", map[string]any{"field": "slug"})
		return
	}

	mantra, err := h.usecase.GetMantraBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, domain.ErrMantraNotFound) {
			utils.WriteBusinessError(w, http.StatusNotFound, "MANTRA_NOT_FOUND", "mantra not found", nil)
			return
		}
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	utils.WriteSuccessResponse(w, newMantraResponse(mantra))
}
