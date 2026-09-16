package http

import (
	"context"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// bodhisattvaResponse mirrors domain.Bodhisattva's client-facing fields
// without omitempty — same convention as sutraResponse/mantraResponse.
type bodhisattvaResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Sect        string `json:"sect"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func newBodhisattvaResponse(b *domain.Bodhisattva) bodhisattvaResponse {
	return bodhisattvaResponse{
		ID:          b.ID,
		Slug:        b.Slug,
		Name:        b.Name,
		Sect:        b.Sect,
		Description: b.Description,
		ImageURL:    b.ImageURL,
		Version:     b.Version,
		CreatedAt:   b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   b.UpdatedAt.Format(time.RFC3339),
	}
}

// listBodhisattvasUsecase is the subset of *usecase.ListBodhisattvasUsecase
// this handler depends on, defined here (the consumer) so it can be unit
// tested with a mock instead of the real use-case.
type listBodhisattvasUsecase interface {
	ListBodhisattvas(ctx context.Context, input domain.ListBodhisattvasInput) ([]*domain.Bodhisattva, error)
}

// BodhisattvasHandler implements GET /v1/content/bodhisattvas (US-009).
// Public read — no authentication required.
type BodhisattvasHandler struct {
	usecase listBodhisattvasUsecase
}

// NewBodhisattvasHandler wires the HTTP handler to its use-case.
func NewBodhisattvasHandler(uc listBodhisattvasUsecase) *BodhisattvasHandler {
	return &BodhisattvasHandler{usecase: uc}
}

func (h *BodhisattvasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	query := r.URL.Query()
	input := domain.ListBodhisattvasInput{
		Sect: query.Get("sect"),
	}

	if raw := query.Get("updated_since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			utils.WriteValidationError(w, "INVALID_UPDATED_SINCE", "updated_since must be RFC3339", map[string]any{"field": "updated_since"})
			return
		}
		input.UpdatedSince = parsed
	}

	bodhisattvas, err := h.usecase.ListBodhisattvas(r.Context(), input)
	if err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	responses := make([]bodhisattvaResponse, 0, len(bodhisattvas))
	for _, b := range bodhisattvas {
		responses = append(responses, newBodhisattvaResponse(b))
	}
	utils.WriteSuccessResponse(w, responses)
}
