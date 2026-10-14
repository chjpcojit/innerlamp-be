package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// profileUsecase is the subset of *usecase.ProfileUsecase this handler depends
// on, declared here (the consumer) so the handler is tested with a mock.
type profileUsecase interface {
	Get(ctx context.Context, userID int64) (*domain.ProfileView, error)
	Update(ctx context.Context, userID int64, patch domain.ProfilePatch) (*domain.ProfileView, error)
	RequestAvatarUpload(ctx context.Context, userID int64, req domain.AvatarUploadRequest) (*domain.AvatarUploadTicket, error)
	ConfirmAvatar(ctx context.Context, userID int64, uploadID string) (*domain.ProfileView, error)
	RemoveAvatar(ctx context.Context, userID int64) (*domain.ProfileView, error)
}

// ProfileHandler serves a user's own profile under /v1/users/me. Every route
// must be wrapped by the JWT auth middleware: the owner is read from the token's
// claims, never from the path, query or body, and there is no route that takes a
// user id.
type ProfileHandler struct {
	usecase profileUsecase
}

// NewProfileHandler wires the handler.
func NewProfileHandler(usecase profileUsecase) *ProfileHandler {
	return &ProfileHandler{usecase: usecase}
}

// optionalString tells the three states of a PATCH field apart, which a plain
// *string cannot: key absent (UnmarshalJSON never runs, set stays false), `null`
// (set, value nil) and a string (set, value non-nil).
type optionalString struct {
	set   bool
	value *string
}

// UnmarshalJSON implements json.Unmarshaler. Any JSON type other than string or
// null is an error, so `{"dharma_name": 5}` is a 400, not a silent "5".
func (o *optionalString) UnmarshalJSON(b []byte) error {
	o.set = true
	if string(b) == "null" {
		o.value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.value = &s
	return nil
}

func (o optionalString) domain() domain.OptionalString {
	return domain.OptionalString{Set: o.set, Value: o.value}
}

// profilePatchRequest is the body of PATCH /v1/users/me. Only the editable
// fields exist, so the strict decoder rejects email, phone_number, id and every
// other key (AC15) before the use case runs.
type profilePatchRequest struct {
	FullName           optionalString `json:"full_name"`
	DharmaName         optionalString `json:"dharma_name"`
	OrdinationTempleID optionalString `json:"ordination_temple_id"`
}

type avatarUploadRequest struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type avatarConfirmRequest struct {
	UploadID string `json:"upload_id"`
}

type templeRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// profileResponse is the Profile wire shape. Keys that have no value are
// omitted, never null or empty (AC2). email, email_verified and phone_number
// are always present. The storage object key is deliberately not here: only the
// signed address is (AC5, AC6).
type profileResponse struct {
	Email              string             `json:"email"`
	EmailVerified      bool               `json:"email_verified"`
	PhoneNumber        string             `json:"phone_number"`
	FullName           *string            `json:"full_name,omitempty"`
	DharmaName         *string            `json:"dharma_name,omitempty"`
	OrdinationTemple   *templeRefResponse `json:"ordination_temple,omitempty"`
	AvatarURL          *string            `json:"avatar_url,omitempty"`
	AvatarURLExpiresAt *time.Time         `json:"avatar_url_expires_at,omitempty"`
}

func newProfileResponse(v *domain.ProfileView) profileResponse {
	p := v.Profile
	r := profileResponse{
		Email: p.Email, EmailVerified: p.EmailVerified, PhoneNumber: p.PhoneNumber,
		FullName: p.FullName, DharmaName: p.DharmaName, AvatarURL: v.AvatarURL,
	}
	if p.OrdinationTemple != nil {
		r.OrdinationTemple = &templeRefResponse{ID: p.OrdinationTemple.ID, Name: p.OrdinationTemple.Name}
	}
	if v.AvatarURLExpiresAt != nil {
		t := v.AvatarURLExpiresAt.UTC()
		r.AvatarURLExpiresAt = &t
	}
	return r
}

type avatarUploadTicketResponse struct {
	UploadID  string            `json:"upload_id"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// writeProfileError answers a use case error. An account that no longer exists
// is the token being stale, so it is a 401 exactly like refresh does, not a 404
// that would confirm which ids exist.
func writeProfileError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, domain.ErrUserNotFound) {
		WriteAuthError(w, r, err)
		return
	}
	writeDomainError(w, err)
}

func (h *ProfileHandler) respond(w http.ResponseWriter, r *http.Request, v *domain.ProfileView, err error) {
	if err != nil {
		writeProfileError(w, r, err)
		return
	}
	utils.WriteSuccessResponse(w, newProfileResponse(v))
}

// Get serves GET /v1/users/me.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	v, err := h.usecase.Get(r.Context(), userID)
	h.respond(w, r, v, err)
}

// Patch serves PATCH /v1/users/me: absent keeps, null or blank clears.
func (h *ProfileHandler) Patch(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req profilePatchRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	v, err := h.usecase.Update(r.Context(), userID, domain.ProfilePatch{
		FullName:           req.FullName.domain(),
		DharmaName:         req.DharmaName.domain(),
		OrdinationTempleID: req.OrdinationTempleID.domain(),
	})
	h.respond(w, r, v, err)
}

// AvatarUploadURL serves POST /v1/users/me/avatar/upload-url.
func (h *ProfileHandler) AvatarUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req avatarUploadRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	t, err := h.usecase.RequestAvatarUpload(r.Context(), userID, domain.AvatarUploadRequest{
		ContentType: req.ContentType, SizeBytes: req.SizeBytes,
	})
	if err != nil {
		writeProfileError(w, r, err)
		return
	}
	utils.WriteSuccessResponse(w, avatarUploadTicketResponse{
		UploadID: t.UploadID, UploadURL: t.URL, Method: t.Method, Headers: t.Headers, ExpiresAt: t.ExpiresAt.UTC(),
	})
}

// AvatarConfirm serves POST /v1/users/me/avatar/confirm.
func (h *ProfileHandler) AvatarConfirm(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req avatarConfirmRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	v, err := h.usecase.ConfirmAvatar(r.Context(), userID, req.UploadID)
	h.respond(w, r, v, err)
}

// AvatarDelete serves DELETE /v1/users/me/avatar.
func (h *ProfileHandler) AvatarDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	v, err := h.usecase.RemoveAvatar(r.Context(), userID)
	h.respond(w, r, v, err)
}
