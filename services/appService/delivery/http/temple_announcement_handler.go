package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// templeAnnouncementUsecase is the subset of *usecase.TempleAnnouncementUsecase
// this handler depends on, declared here (the consumer) so it can be mocked.
type templeAnnouncementUsecase interface {
	ListForUser(ctx context.Context, userID int64, page, pageSize int) (domain.UserAnnouncements, error)
	GetReceive(ctx context.Context, userID int64) (bool, error)
	SetReceive(ctx context.Context, userID int64, receive bool) (bool, error)
}

// TempleAnnouncementHandler serves the caller's temple announcements and the
// receive switch. The user is the token's; no route takes a temple or user id.
// cmd/main.go puts auth and the dedicated rate limit in front of every route.
type TempleAnnouncementHandler struct {
	usecase templeAnnouncementUsecase
}

// NewTempleAnnouncementHandler wires the handler.
func NewTempleAnnouncementHandler(usecase templeAnnouncementUsecase) *TempleAnnouncementHandler {
	return &TempleAnnouncementHandler{usecase: usecase}
}

type announcementIssuingBodyResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type templeAnnouncementResponse struct {
	ID          string                          `json:"id"`
	Title       string                          `json:"title"`
	Body        string                          `json:"body"`
	PublishedAt string                          `json:"published_at"`
	ExpiresAt   string                          `json:"expires_at"`
	IssuingBody announcementIssuingBodyResponse `json:"issuing_body"`
}

type announcementPageResponse struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type announcementTempleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userAnnouncementsResponse struct {
	Receive bool                         `json:"receive"`
	Temple  *announcementTempleResponse  `json:"temple"`
	Items   []templeAnnouncementResponse `json:"items"`
	Page    announcementPageResponse     `json:"page"`
}

type announcementSettingsResponse struct {
	Receive bool `json:"receive"`
}

// announcementSettingsRequest uses *bool so a missing key is told from false.
type announcementSettingsRequest struct {
	Receive *bool `json:"receive"`
}

// List serves GET /v1/users/me/temple-announcements?page=&page_size= (R-1..R-6).
func (h *TempleAnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	page, ok := positiveQueryInt(w, r, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := positiveQueryInt(w, r, "page_size", 0) // 0: the usecase applies the configured default
	if !ok {
		return
	}

	res, err := h.usecase.ListForUser(r.Context(), userID, page, pageSize)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	out := userAnnouncementsResponse{
		Receive: res.Receive,
		Items:   make([]templeAnnouncementResponse, 0, len(res.Page.Items)),
		Page:    announcementPageResponse{Page: res.Page.Page, PageSize: res.Page.PageSize, Total: res.Page.Total},
	}
	if res.Temple != nil {
		out.Temple = &announcementTempleResponse{ID: res.Temple.ID, Name: res.Temple.Name}
	}
	for _, a := range res.Page.Items {
		out.Items = append(out.Items, templeAnnouncementResponse{
			ID: a.ID, Title: a.Title, Body: a.Body,
			PublishedAt: a.PublishedAt.UTC().Format(time.RFC3339),
			ExpiresAt:   a.ExpiresAt.UTC().Format(time.RFC3339),
			IssuingBody: announcementIssuingBodyResponse{ID: a.IssuingBody.ID, Name: a.IssuingBody.Name},
		})
	}
	w.Header().Set("Cache-Control", "no-store")
	utils.WriteSuccessResponse(w, out)
}

// GetSettings serves GET /v1/users/me/temple-announcement-settings.
func (h *TempleAnnouncementHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	receive, err := h.usecase.GetReceive(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	utils.WriteSuccessResponse(w, announcementSettingsResponse{Receive: receive})
}

// PutSettings serves PUT /v1/users/me/temple-announcement-settings. The body is
// exactly {"receive": <bool>}: a missing key, a non-boolean or an extra key is 400.
func (h *TempleAnnouncementHandler) PutSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req announcementSettingsRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Receive == nil {
		utils.WriteValidationError(w, "VALIDATION_ERROR", "receive is required", map[string]any{"field": "receive"})
		return
	}
	receive, err := h.usecase.SetReceive(r.Context(), userID, *req.Receive)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	utils.WriteSuccessResponse(w, announcementSettingsResponse{Receive: receive})
}

// positiveQueryInt reads an optional query integer that must be >= 1 when
// present; a malformed or non-positive value answers 400 INVALID_PAGE.
func positiveQueryInt(w http.ResponseWriter, r *http.Request, key string, def int) (int, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		writeDomainError(w, domain.ErrInvalidPage)
		return 0, false
	}
	return n, true
}
