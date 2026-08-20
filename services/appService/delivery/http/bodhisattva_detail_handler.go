package http

import (
	"context"
	"errors"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// bodhisattvaDetailResponse embeds the related mantras in the same response
// (US-010's Ghi chú kỹ thuật) rather than making the client call
// GET /v1/content/mantras?bodhisattva_id=... separately.
type bodhisattvaDetailResponse struct {
	bodhisattvaResponse
	Mantras []mantraResponse `json:"mantras"`
}

func newBodhisattvaDetailResponse(b *domain.Bodhisattva, mantras []*domain.Mantra) bodhisattvaDetailResponse {
	resp := bodhisattvaDetailResponse{
		bodhisattvaResponse: newBodhisattvaResponse(b),
		Mantras:             make([]mantraResponse, 0, len(mantras)),
	}
	for _, m := range mantras {
		resp.Mantras = append(resp.Mantras, newMantraResponse(m))
	}
	return resp
}

// getBodhisattvaUsecase is the subset of *usecase.GetBodhisattvaUsecase this
// handler depends on, defined here (the consumer) so it can be unit tested
// with a mock instead of the real use-case.
type getBodhisattvaUsecase interface {
	GetBodhisattvaBySlug(ctx context.Context, slug string) (*domain.Bodhisattva, []*domain.Mantra, error)
}

// BodhisattvaDetailHandler implements GET /v1/content/bodhisattvas/{slug}
// (US-010). Public read — no authentication required.
type BodhisattvaDetailHandler struct {
	usecase getBodhisattvaUsecase
}

// NewBodhisattvaDetailHandler wires the HTTP handler to its use-case.
func NewBodhisattvaDetailHandler(uc getBodhisattvaUsecase) *BodhisattvaDetailHandler {
	return &BodhisattvaDetailHandler{usecase: uc}
}

func (h *BodhisattvaDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		utils.WriteValidationError(w, "MISSING_SLUG", "slug is required", map[string]any{"field": "slug"})
		return
	}

	bodhisattva, mantras, err := h.usecase.GetBodhisattvaBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, domain.ErrBodhisattvaNotFound) {
			utils.WriteBusinessError(w, http.StatusNotFound, "BODHISATTVA_NOT_FOUND", "bodhisattva not found", nil)
			return
		}
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	utils.WriteSuccessResponse(w, newBodhisattvaDetailResponse(bodhisattva, mantras))
}
