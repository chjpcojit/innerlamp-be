package http

import (
	"context"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sutraResponse mirrors domain.Sutra's client-facing fields without
// omitempty — the client's delta-sync cache (SQLite/Hive) needs every field
// present even when empty (e.g. audio_url), same convention as profileResponse.
type sutraResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Sect      string `json:"sect"`
	Language  string `json:"language"`
	Body      string `json:"body"`
	AudioURL  string `json:"audio_url"`
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func newSutraResponse(s *domain.Sutra) sutraResponse {
	return sutraResponse{
		ID:        s.ID,
		Slug:      s.Slug,
		Title:     s.Title,
		Sect:      s.Sect,
		Language:  s.Language,
		Body:      s.Body,
		AudioURL:  s.AudioURL,
		Version:   s.Version,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
}

// listSutrasUsecase is the subset of *usecase.ListSutrasUsecase this handler
// depends on, defined here (the consumer) so it can be unit tested with a
// mock instead of the real use-case.
type listSutrasUsecase interface {
	ListSutras(ctx context.Context, input domain.ListSutrasInput) ([]*domain.Sutra, error)
}

// SutrasHandler implements GET /v1/content/sutras (US-005). Public read —
// no authentication required, unlike the other handlers in this package.
type SutrasHandler struct {
	usecase listSutrasUsecase
}

// NewSutrasHandler wires the HTTP handler to its use-case.
func NewSutrasHandler(uc listSutrasUsecase) *SutrasHandler {
	return &SutrasHandler{usecase: uc}
}

func (h *SutrasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	query := r.URL.Query()
	input := domain.ListSutrasInput{Sect: query.Get("sect")}

	if raw := query.Get("updated_since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			utils.WriteValidationError(w, "INVALID_UPDATED_SINCE", "updated_since must be RFC3339", map[string]any{"field": "updated_since"})
			return
		}
		input.UpdatedSince = parsed
	}

	sutras, err := h.usecase.ListSutras(r.Context(), input)
	if err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	responses := make([]sutraResponse, 0, len(sutras))
	for _, s := range sutras {
		responses = append(responses, newSutraResponse(s))
	}
	utils.WriteSuccessResponse(w, responses)
}
