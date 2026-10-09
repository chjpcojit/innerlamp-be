package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// memorialUsecase is the subset of *usecase.MemorialUsecase this handler depends
// on, declared here (the consumer) so the handler is tested with a mock.
type memorialUsecase interface {
	Create(ctx context.Context, userID int64, in domain.MemorialInput) (*domain.Memorial, error)
	List(ctx context.Context, userID int64) ([]domain.Memorial, error)
	Get(ctx context.Context, userID, id int64) (*domain.Memorial, error)
	Update(ctx context.Context, userID, id int64, in domain.MemorialInput) (*domain.Memorial, error)
	Delete(ctx context.Context, userID, id int64) error
}

// MemorialHandler serves a user's own death-anniversary book. Every route must be
// wrapped by the JWT auth middleware: the owner is read from the token's claims,
// never from the path or body.
type MemorialHandler struct {
	usecase memorialUsecase
}

// NewMemorialHandler wires the handler.
func NewMemorialHandler(usecase memorialUsecase) *MemorialHandler {
	return &MemorialHandler{usecase: usecase}
}

// reminderRequest / reminderResponse are the reminder object on the wire.
// In a request, days_before is a pointer so a missing value is not read as 0 and
// timezone may be omitted.
type reminderRequest struct {
	DaysBefore *int   `json:"days_before"`
	Time       string `json:"time"`
	Timezone   string `json:"timezone"`
}

type reminderResponse struct {
	DaysBefore int    `json:"days_before"`
	Time       string `json:"time"`
	Timezone   string `json:"timezone"`
}

// memorialRequest is the body of POST and PUT. Server-owned fields (id,
// photo_url, timestamps) are deliberately absent, so the strict decoder rejects
// them as unknown fields.
type memorialRequest struct {
	FullName        string           `json:"full_name"`
	Relationship    string           `json:"relationship"`
	DeathCalendar   string           `json:"death_calendar"`
	DeathDate       *string          `json:"death_date"`
	DeathDay        *int             `json:"death_day"`
	DeathMonth      *int             `json:"death_month"`
	DeathLeapMonth  *bool            `json:"death_leap_month"`
	Note            *string          `json:"note"`
	Reminder        *reminderRequest `json:"reminder"`
	AddToOSReminder bool             `json:"add_to_os_reminder"`
}

// memorialResponse is the Memorial wire shape. Optional values are JSON null,
// never absent or empty strings. photo_url is always null until the photo
// feature exists (TD-001, memorial; tasks.md Phase 6).
type memorialResponse struct {
	ID              int64             `json:"id"`
	FullName        string            `json:"full_name"`
	Relationship    string            `json:"relationship"`
	DeathDate       *string           `json:"death_date"`
	DeathCalendar   string            `json:"death_calendar"`
	DeathDay        *int              `json:"death_day"`
	DeathMonth      *int              `json:"death_month"`
	DeathLeapMonth  *bool             `json:"death_leap_month"`
	Note            *string           `json:"note"`
	PhotoURL        *string           `json:"photo_url"`
	Reminder        *reminderResponse `json:"reminder"`
	AddToOSReminder bool              `json:"add_to_os_reminder"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type memorialListResponse struct {
	Items []memorialResponse `json:"items"`
	Total int                `json:"total"`
}

func newMemorialResponse(m *domain.Memorial) memorialResponse {
	r := memorialResponse{
		ID: m.ID, FullName: m.FullName, Relationship: string(m.Relationship), DeathCalendar: string(m.DeathCalendar),
		DeathDay: m.DeathDay, DeathMonth: m.DeathMonth, DeathLeapMonth: m.DeathLeapMonth, Note: m.Note,
		AddToOSReminder: m.AddToOSReminder, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
	if m.DeathDate != nil {
		d := m.DeathDate.Format(time.DateOnly)
		r.DeathDate = &d
	}
	if m.Reminder != nil {
		r.Reminder = &reminderResponse{DaysBefore: m.Reminder.DaysBefore, Time: m.Reminder.Time, Timezone: m.Reminder.Timezone}
	}
	return r
}

func (req memorialRequest) toInput() domain.MemorialInput {
	in := domain.MemorialInput{
		FullName: req.FullName, Relationship: req.Relationship, DeathCalendar: req.DeathCalendar,
		DeathDay: req.DeathDay, DeathMonth: req.DeathMonth, DeathLeapMonth: req.DeathLeapMonth,
		Note: req.Note, AddToOSReminder: req.AddToOSReminder,
	}
	if req.DeathDate != nil {
		in.DeathDate = *req.DeathDate
	}
	if req.Reminder != nil {
		in.Reminder = &domain.ReminderInput{
			DaysBefore: req.Reminder.DaysBefore, Time: req.Reminder.Time, Timezone: req.Reminder.Timezone,
		}
	}
	return in
}

// currentUserID reads the owner from the access token's claims. It writes the 401
// itself and reports false when there is no usable user id.
func currentUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return 0, false
	}
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		WriteAuthError(w, r, errors.New("user_id claim is not numeric"))
		return 0, false
	}
	return userID, true
}

// memorialID parses the {id} path value. Anything that is not a positive integer
// is reported as 404 — the same answer as an id that does not exist.
func memorialID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeDomainError(w, domain.ErrMemorialNotFound)
		return 0, false
	}
	return id, true
}

// Create serves POST /v1/memorials.
func (h *MemorialHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req memorialRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	m, err := h.usecase.Create(r.Context(), userID, req.toInput())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, newMemorialResponse(m))
}

// List serves GET /v1/memorials: the caller's own memorials, newest first.
func (h *MemorialHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.usecase.List(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]memorialResponse, 0, len(items))
	for i := range items {
		out = append(out, newMemorialResponse(&items[i]))
	}
	utils.WriteSuccessResponse(w, memorialListResponse{Items: out, Total: len(out)})
}

// Get serves GET /v1/memorials/{id}.
func (h *MemorialHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := memorialID(w, r)
	if !ok {
		return
	}
	m, err := h.usecase.Get(r.Context(), userID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newMemorialResponse(m))
}

// Update serves PUT /v1/memorials/{id}: a full replacement of the editable fields.
func (h *MemorialHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := memorialID(w, r)
	if !ok {
		return
	}
	var req memorialRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	m, err := h.usecase.Update(r.Context(), userID, id, req.toInput())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newMemorialResponse(m))
}

// Delete serves DELETE /v1/memorials/{id}: 200 {"success": true} with no data.
func (h *MemorialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := memorialID(w, r)
	if !ok {
		return
	}
	if err := h.usecase.Delete(r.Context(), userID, id); err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, nil)
}
