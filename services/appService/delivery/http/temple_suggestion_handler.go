package http

import (
	"context"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// templeSuggestionUsecase is the subset of *usecase.TempleSuggestionUsecase this
// handler depends on, declared here (the consumer) so it is tested with a mock.
type templeSuggestionUsecase interface {
	Create(ctx context.Context, in domain.TempleSuggestionInput) (*domain.TempleSuggestion, error)
}

// TempleSuggestionHandler serves the public "suggest a temple" route. It is
// anonymous by design: wrap it with the rate limit, not the JWT middleware.
type TempleSuggestionHandler struct {
	usecase templeSuggestionUsecase
}

// NewTempleSuggestionHandler wires the handler.
func NewTempleSuggestionHandler(usecase templeSuggestionUsecase) *TempleSuggestionHandler {
	return &TempleSuggestionHandler{usecase: usecase}
}

// templeSuggestionRequest is the body of POST /v1/temple-suggestions. Server-
// owned fields (id, status, timestamps) are deliberately absent, so the strict
// decoder rejects them as unknown fields.
type templeSuggestionRequest struct {
	Name     string `json:"name"`
	Province string `json:"province"`
	Address  string `json:"address"`
}

// templeSuggestionResponse is all a submitter learns: the id and that it is
// pending. Nothing about other suggestions or the catalogue is ever returned.
type templeSuggestionResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// Create serves POST /v1/temple-suggestions.
func (h *TempleSuggestionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req templeSuggestionRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	s, err := h.usecase.Create(r.Context(), domain.TempleSuggestionInput{
		Name: req.Name, Province: req.Province, Address: req.Address,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, templeSuggestionResponse{ID: s.ID, Status: s.Status})
}
