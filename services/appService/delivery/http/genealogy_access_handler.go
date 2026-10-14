package http

import (
	"context"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// genealogyAccessUsecase is the subset of *usecase.GenealogyAccessUsecase this
// handler uses.
type genealogyAccessUsecase interface {
	Invite(ctx context.Context, userID, genealogyID int64, in domain.InviteInput) (*domain.Invitation, error)
	ListInvitations(ctx context.Context, userID, genealogyID int64) ([]domain.Invitation, error)
	CancelInvitation(ctx context.Context, userID, genealogyID, invitationID int64) error
	ListViewers(ctx context.Context, userID, genealogyID int64) ([]domain.Viewer, error)
	RemoveViewer(ctx context.Context, userID, genealogyID, targetUserID int64) error
}

// genealogyInvitationUsecase is the subset of *usecase.GenealogyInvitationUsecase.
type genealogyInvitationUsecase interface {
	ListMine(ctx context.Context, userID int64) ([]domain.Invitation, error)
	Accept(ctx context.Context, userID, invitationID int64) (*domain.AcceptResult, error)
	Decline(ctx context.Context, userID, invitationID int64) error
}

// GenealogyAccessHandler serves the owner's side of sharing: invitations and
// viewers.
type GenealogyAccessHandler struct {
	usecase genealogyAccessUsecase
}

// NewGenealogyAccessHandler wires the handler.
func NewGenealogyAccessHandler(usecase genealogyAccessUsecase) *GenealogyAccessHandler {
	return &GenealogyAccessHandler{usecase: usecase}
}

// GenealogyInvitationHandler serves the invitee's side: inbox, accept, decline.
type GenealogyInvitationHandler struct {
	usecase genealogyInvitationUsecase
}

// NewGenealogyInvitationHandler wires the handler.
func NewGenealogyInvitationHandler(usecase genealogyInvitationUsecase) *GenealogyInvitationHandler {
	return &GenealogyInvitationHandler{usecase: usecase}
}

type inviteRequest struct {
	MemberID    int64   `json:"member_id"`
	PhoneNumber *string `json:"phone_number"`
}

// invitationResponse is the owner's Invitation shape (AC44, AC73).
type invitationResponse struct {
	ID             int64      `json:"id"`
	MemberID       int64      `json:"member_id"`
	MemberFullName string     `json:"member_full_name"`
	PhoneNumber    string     `json:"phone_number"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RespondedAt    *time.Time `json:"responded_at"`
}

func newInvitationResponse(i domain.Invitation) invitationResponse {
	return invitationResponse{
		ID: i.ID, MemberID: i.MemberID, MemberFullName: i.MemberFullName, PhoneNumber: i.PhoneNumber,
		Status: string(i.Status), CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt, RespondedAt: i.RespondedAt,
	}
}

// myInvitationResponse is the invitee's inbox item (AC70): the tree name, the
// member they were invited as and who invited them — nothing else of the tree.
type myInvitationResponse struct {
	ID             int64     `json:"id"`
	GenealogyName  string    `json:"genealogy_name"`
	MemberFullName string    `json:"member_full_name"`
	InvitedByPhone string    `json:"invited_by_phone"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type viewerResponse struct {
	UserID      int64     `json:"user_id"`
	PhoneNumber string    `json:"phone_number"`
	MemberID    *int64    `json:"member_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type acceptResponse struct {
	GenealogyID int64  `json:"genealogy_id"`
	MemberID    int64  `json:"member_id"`
	Role        string `json:"role"`
}

// Invite serves POST /v1/genealogies/{id}/invitations (owner only). The answer
// is the same whether or not the number has an account (AC45).
func (h *GenealogyAccessHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req inviteRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	in := domain.InviteInput{MemberID: req.MemberID}
	if req.PhoneNumber != nil {
		in.PhoneNumber = *req.PhoneNumber
	}
	inv, err := h.usecase.Invite(r.Context(), userID, pathID(r, "id"), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, newInvitationResponse(*inv))
}

// ListInvitations serves GET /v1/genealogies/{id}/invitations (owner only).
func (h *GenealogyAccessHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.usecase.ListInvitations(r.Context(), userID, pathID(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]invitationResponse, 0, len(items))
	for _, i := range items {
		out = append(out, newInvitationResponse(i))
	}
	utils.WriteSuccessResponse(w, newList(out))
}

// CancelInvitation serves DELETE /v1/genealogies/{id}/invitations/{invitationId}.
func (h *GenealogyAccessHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.usecase.CancelInvitation(r.Context(), userID, pathID(r, "id"), pathID(r, "invitationId")); err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, nil)
}

// ListViewers serves GET /v1/genealogies/{id}/viewers (owner only).
func (h *GenealogyAccessHandler) ListViewers(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.usecase.ListViewers(r.Context(), userID, pathID(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]viewerResponse, 0, len(items))
	for _, v := range items {
		out = append(out, viewerResponse{UserID: v.UserID, PhoneNumber: v.PhoneNumber, MemberID: v.MemberID, CreatedAt: v.CreatedAt})
	}
	utils.WriteSuccessResponse(w, newList(out))
}

// RemoveViewer serves DELETE /v1/genealogies/{id}/viewers/{userId}: the owner
// removes a viewer, or a viewer removes themself (leaves).
func (h *GenealogyAccessHandler) RemoveViewer(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	// A malformed {userId} becomes 0: the use case still resolves the caller's
	// role first, then finds no such viewer.
	if err := h.usecase.RemoveViewer(r.Context(), userID, pathID(r, "id"), pathID(r, "userId")); err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, nil)
}

// ListMine serves GET /v1/invitations: the invitee's inbox.
func (h *GenealogyInvitationHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.usecase.ListMine(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]myInvitationResponse, 0, len(items))
	for _, i := range items {
		out = append(out, myInvitationResponse{
			ID: i.ID, GenealogyName: i.GenealogyName, MemberFullName: i.MemberFullName,
			InvitedByPhone: i.InvitedByPhone, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt,
		})
	}
	utils.WriteSuccessResponse(w, newList(out))
}

// Accept serves POST /v1/invitations/{id}/accept.
func (h *GenealogyInvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	res, err := h.usecase.Accept(r.Context(), userID, pathID(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, acceptResponse{GenealogyID: res.GenealogyID, MemberID: res.MemberID, Role: string(res.Role)})
}

// Decline serves POST /v1/invitations/{id}/decline.
func (h *GenealogyInvitationHandler) Decline(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.usecase.Decline(r.Context(), userID, pathID(r, "id")); err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, nil)
}
