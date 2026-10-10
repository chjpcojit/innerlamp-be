package http

import (
	"context"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// lampUsecase is the subset of *usecase.LampUsecase this handler depends on,
// declared here (the consumer) so the handler is tested with a mock.
type lampUsecase interface {
	Light(ctx context.Context, userID int64, in domain.LampInput) (lamp *domain.Lamp, created bool, err error)
	Garden(ctx context.Context, f domain.GardenFilter) (*domain.GardenPage, error)
}

// LampHandler serves lighting a lamp and the garden of lit lamps. Both routes
// must be wrapped by the JWT auth middleware: the lighter is read from the
// token's claims, never from the body, and visitors see nothing.
type LampHandler struct {
	usecase lampUsecase
}

// NewLampHandler wires the handler.
func NewLampHandler(usecase lampUsecase) *LampHandler {
	return &LampHandler{usecase: usecase}
}

// lampRequest is the body of POST /v1/lamps. Server-owned fields (id, user_id,
// lit_at, end_at, lighter_name) are deliberately absent, so the strict decoder
// rejects them as unknown fields.
type lampRequest struct {
	LampType     string  `json:"lamp_type"`
	Intention    string  `json:"intention"`
	Recipient    string  `json:"recipient"`
	Prayer       *string `json:"prayer"`
	DurationDays int     `json:"duration_days"`
	IsAnonymous  bool    `json:"is_anonymous"`
}

func (req lampRequest) toInput() domain.LampInput {
	return domain.LampInput{
		LampType: req.LampType, Intention: req.Intention, Recipient: req.Recipient,
		Prayer: req.Prayer, DurationDays: req.DurationDays, IsAnonymous: req.IsAnonymous,
	}
}

// lampResponse is the Lamp wire shape, returned only to the lighter. Optional
// values are JSON null, never absent or empty strings.
type lampResponse struct {
	ID           int64     `json:"id"`
	LampType     string    `json:"lamp_type"`
	Intention    string    `json:"intention"`
	Recipient    string    `json:"recipient"`
	Prayer       *string   `json:"prayer"`
	DurationDays int       `json:"duration_days"`
	IsAnonymous  bool      `json:"is_anonymous"`
	LighterName  string    `json:"lighter_name"`
	LitAt        time.Time `json:"lit_at"`
	EndAt        time.Time `json:"end_at"`
}

func newLampResponse(l *domain.Lamp) lampResponse {
	return lampResponse{
		ID: l.ID, LampType: string(l.LampType), Intention: string(l.Intention), Recipient: l.Recipient,
		Prayer: l.Prayer, DurationDays: l.DurationDays, IsAnonymous: l.IsAnonymous, LighterName: l.DisplayName(),
		LitAt: l.LitAt.UTC(), EndAt: l.EndAt.UTC(),
	}
}

// gardenLampResponse is one lamp as other users see it: no intention, prayer,
// owner or anonymity flag.
type gardenLampResponse struct {
	ID          int64     `json:"id"`
	LampType    string    `json:"lamp_type"`
	Recipient   string    `json:"recipient"`
	LighterName string    `json:"lighter_name"`
	EndAt       time.Time `json:"end_at"`
}

type gardenResponse struct {
	Items      []gardenLampResponse `json:"items"`
	Total      int                  `json:"total"`
	Counts     map[string]int       `json:"counts"`
	HasMore    bool                 `json:"has_more"`
	NextCursor *string              `json:"next_cursor"`
}

func newGardenResponse(p *domain.GardenPage) gardenResponse {
	items := make([]gardenLampResponse, 0, len(p.Items))
	for _, g := range p.Items {
		items = append(items, gardenLampResponse{
			ID: g.ID, LampType: string(g.LampType), Recipient: g.Recipient,
			LighterName: g.DisplayName(), EndAt: g.EndAt.UTC(),
		})
	}
	counts := make(map[string]int, len(domain.AllLampTypes))
	for _, t := range domain.AllLampTypes {
		counts[string(t)] = p.Counts[t]
	}
	return gardenResponse{Items: items, Total: p.Total, Counts: counts, HasMore: p.HasMore, NextCursor: p.NextCursor}
}

// Light serves POST /v1/lamps: 201 with the new lamp, or 200 with the lamp the
// same user just lit when this is a double tap.
func (h *LampHandler) Light(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req lampRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	lamp, created, err := h.usecase.Light(r.Context(), userID, req.toInput())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	utils.WriteSuccessResponseWithStatus(w, status, newLampResponse(lamp))
}

// Garden serves GET /v1/lamps: every user's lit lamps, newest first, one page at
// a time, with the totals. The caller's identity is required but never used to
// filter.
func (h *LampHandler) Garden(w http.ResponseWriter, r *http.Request) {
	if _, ok := currentUserID(w, r); !ok {
		return
	}
	query := r.URL.Query()
	var f domain.GardenFilter
	if v, present := query["lamp_type"]; present {
		f.LampType = &v[0]
	}
	if v, present := query["limit"]; present {
		f.Limit = &v[0]
	}
	if v, present := query["cursor"]; present {
		f.Cursor = &v[0]
	}

	page, err := h.usecase.Garden(r.Context(), f)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newGardenResponse(page))
}
