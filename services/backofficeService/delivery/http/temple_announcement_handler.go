package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// templeAnnouncementUsecase is the subset of *usecase.TempleAnnouncementUsecase
// the handler needs, declared here (the consumer) so it can be tested with a mock.
type templeAnnouncementUsecase interface {
	Publish(ctx context.Context, templeID, adminID string, in domain.NewAnnouncement) (*domain.Announcement, error)
	Edit(ctx context.Context, id string, p domain.AnnouncementPatch) (*domain.Announcement, error)
	Revoke(ctx context.Context, id string) (*domain.Announcement, error)
	List(ctx context.Context, templeID string, page, pageSize int) (*domain.AnnouncementPage, error)
	CreateIssuingBody(ctx context.Context, templeID, name string) (*domain.IssuingBody, error)
	ListIssuingBodies(ctx context.Context, templeID string) ([]domain.IssuingBody, error)
}

// TempleAnnouncementHandler serves the announcement routes. Authorization is
// the route wrapper's; the handler only reads the caller's id from the claims.
type TempleAnnouncementHandler struct {
	usecase templeAnnouncementUsecase
}

// NewTempleAnnouncementHandler wires the handler.
func NewTempleAnnouncementHandler(usecase templeAnnouncementUsecase) *TempleAnnouncementHandler {
	return &TempleAnnouncementHandler{usecase: usecase}
}

type issuingBodyRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type announcementResponse struct {
	ID          string                 `json:"id"`
	TempleID    string                 `json:"temple_id"`
	IssuingBody issuingBodyRefResponse `json:"issuing_body"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Source      string                 `json:"source"`
	State       string                 `json:"state"`
	PublishedAt time.Time              `json:"published_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	RevokedAt   *time.Time             `json:"revoked_at"`
	CreatedBy   *string                `json:"created_by"`
}

func toAnnouncement(a domain.Announcement) announcementResponse {
	return announcementResponse{
		ID: a.ID, TempleID: a.TempleID, IssuingBody: issuingBodyRefResponse{ID: a.IssuingBodyID, Name: a.IssuingBodyName},
		Title: a.Title, Body: a.Body, Source: a.Source, State: string(a.State),
		PublishedAt: a.PublishedAt, ExpiresAt: a.ExpiresAt, RevokedAt: a.RevokedAt, CreatedBy: a.CreatedBy,
	}
}

type announcementPageResponse struct {
	Items      []announcementResponse `json:"items"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	Total      int                    `json:"total"`
	TotalPages int                    `json:"total_pages"`
}

type issuingBodyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type publishRequest struct {
	Title         string `json:"title"`
	Body          string `json:"body"`
	IssuingBodyID string `json:"issuing_body_id"`
	ExpiresAt     string `json:"expires_at"`
}

type editRequest struct {
	Title         *string `json:"title"`
	Body          *string `json:"body"`
	IssuingBodyID *string `json:"issuing_body_id"`
	ExpiresAt     *string `json:"expires_at"`
}

type issuingBodyRequest struct {
	Name string `json:"name"`
}

// parseExpiresAt reads an RFC 3339 timestamp; blank is the zero time (missing).
func parseExpiresAt(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, &domain.AnnouncementInputError{Field: "expires_at", Message: "expires_at must be an RFC 3339 timestamp"}
	}
	return t, nil
}

// Publish serves POST /v1/admin/temples/{id}/announcements.
func (h *TempleAnnouncementHandler) Publish(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return
	}
	var req publishRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	exp, err := parseExpiresAt(req.ExpiresAt)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	a, err := h.usecase.Publish(r.Context(), r.PathValue("id"), claims.UserID, domain.NewAnnouncement{
		IssuingBodyID: req.IssuingBodyID, Title: req.Title, Body: req.Body, ExpiresAt: exp,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, toAnnouncement(*a))
}

// Edit serves PATCH /v1/admin/temple-announcements/{id}.
func (h *TempleAnnouncementHandler) Edit(w http.ResponseWriter, r *http.Request) {
	var req editRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	patch := domain.AnnouncementPatch{Title: req.Title, Body: req.Body, IssuingBodyID: req.IssuingBodyID}
	if req.ExpiresAt != nil {
		exp, err := parseExpiresAt(*req.ExpiresAt)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		patch.ExpiresAt = &exp // a blank value is the zero time: rejected as required
	}
	a, err := h.usecase.Edit(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, toAnnouncement(*a))
}

// Revoke serves POST /v1/admin/temple-announcements/{id}/revoke.
func (h *TempleAnnouncementHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	a, err := h.usecase.Revoke(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, toAnnouncement(*a))
}

// List serves GET /v1/admin/temples/{id}/announcements?page=&page_size=.
func (h *TempleAnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize := 1, 0
	for key, dst := range map[string]*int{"page": &page, "page_size": &pageSize} {
		if raw := r.URL.Query().Get(key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				writeDomainError(w, domain.ErrInvalidPage)
				return
			}
			*dst = n
		}
	}
	res, err := h.usecase.List(r.Context(), r.PathValue("id"), page, pageSize)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	items := make([]announcementResponse, 0, len(res.Items))
	for _, a := range res.Items {
		items = append(items, toAnnouncement(a))
	}
	utils.WriteSuccessResponse(w, announcementPageResponse{
		Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total, TotalPages: res.TotalPages,
	})
}

// CreateIssuingBody serves POST /v1/admin/temples/{id}/issuing-bodies.
func (h *TempleAnnouncementHandler) CreateIssuingBody(w http.ResponseWriter, r *http.Request) {
	var req issuingBodyRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	b, err := h.usecase.CreateIssuingBody(r.Context(), r.PathValue("id"), req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, issuingBodyResponse{ID: b.ID, Name: b.Name, CreatedAt: b.CreatedAt})
}

// ListIssuingBodies serves GET /v1/admin/temples/{id}/issuing-bodies.
func (h *TempleAnnouncementHandler) ListIssuingBodies(w http.ResponseWriter, r *http.Request) {
	bodies, err := h.usecase.ListIssuingBodies(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	items := make([]issuingBodyResponse, 0, len(bodies))
	for _, b := range bodies {
		items = append(items, issuingBodyResponse{ID: b.ID, Name: b.Name, CreatedAt: b.CreatedAt})
	}
	utils.WriteSuccessResponse(w, map[string]any{"items": items})
}
