package http

import (
	"context"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mantraResponse mirrors domain.Mantra's client-facing fields without
// omitempty — same convention as sutraResponse, so the client's delta-sync
// cache schema stays uniform.
type mantraResponse struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Body          string `json:"body"`
	AudioURL      string `json:"audio_url"`
	BodhisattvaID string `json:"bodhisattva_id"`
	Version       int    `json:"version"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func newMantraResponse(m *domain.Mantra) mantraResponse {
	return mantraResponse{
		ID:            m.ID,
		Slug:          m.Slug,
		Title:         m.Title,
		Body:          m.Body,
		AudioURL:      m.AudioURL,
		BodhisattvaID: m.BodhisattvaID,
		Version:       m.Version,
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     m.UpdatedAt.Format(time.RFC3339),
	}
}

// listMantrasUsecase is the subset of *usecase.ListMantrasUsecase this
// handler depends on, defined here (the consumer) so it can be unit tested
// with a mock instead of the real use-case.
type listMantrasUsecase interface {
	ListMantras(ctx context.Context, input domain.ListMantrasInput) ([]*domain.Mantra, error)
}

// MantrasHandler implements GET /v1/content/mantras (US-007). Public read —
// no authentication required.
type MantrasHandler struct {
	usecase listMantrasUsecase
}

// NewMantrasHandler wires the HTTP handler to its use-case.
func NewMantrasHandler(uc listMantrasUsecase) *MantrasHandler {
	return &MantrasHandler{usecase: uc}
}

func (h *MantrasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	query := r.URL.Query()
	input := domain.ListMantrasInput{BodhisattvaID: query.Get("bodhisattva_id")}

	if raw := query.Get("updated_since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			utils.WriteValidationError(w, "INVALID_UPDATED_SINCE", "updated_since must be RFC3339", map[string]any{"field": "updated_since"})
			return
		}
		input.UpdatedSince = parsed
	}

	mantras, err := h.usecase.ListMantras(r.Context(), input)
	if err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	responses := make([]mantraResponse, 0, len(mantras))
	for _, m := range mantras {
		responses = append(responses, newMantraResponse(m))
	}
	utils.WriteSuccessResponse(w, responses)
}
