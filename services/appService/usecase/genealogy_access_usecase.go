package usecase

import (
	"context"
	"strings"

	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GenealogyAccessUsecase is the owner's side of sharing a tree: invite a
// member by phone number, see and cancel invitations, list and remove viewers
// (US-GEN-06). A viewer may only remove themself (leave).
type GenealogyAccessUsecase struct {
	genealogies domain.GenealogyRepository
	viewers     domain.ViewerRepository
	invitations domain.InvitationRepository
	cfg         config.GenealogyConfig
}

// NewGenealogyAccessUsecase wires the use case.
func NewGenealogyAccessUsecase(
	genealogies domain.GenealogyRepository,
	viewers domain.ViewerRepository,
	invitations domain.InvitationRepository,
	cfg config.GenealogyConfig,
) *GenealogyAccessUsecase {
	return &GenealogyAccessUsecase{genealogies: genealogies, viewers: viewers, invitations: invitations, cfg: cfg}
}

// Invite creates a pending invitation tied to a member (AC44–AC47, AC60). It
// never looks the number up in the user store, so the answer is the same
// whether or not the number has an account (AC45). Owner only.
func (u *GenealogyAccessUsecase) Invite(ctx context.Context, userID, genealogyID int64, in domain.InviteInput) (*domain.Invitation, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	if in.MemberID <= 0 {
		return nil, genInvalid("member_id", "member_id is required")
	}
	phone := ""
	if raw := strings.TrimSpace(in.PhoneNumber); raw != "" {
		phone = validation.NormalizePhone(raw)
		if phone == "" {
			return nil, genInvalid("phone_number", "phone_number is not a valid phone number")
		}
	}
	return u.invitations.Create(ctx, genealogyID, userID,
		domain.InviteInput{MemberID: in.MemberID, PhoneNumber: phone},
		u.cfg.MaxViewers, u.cfg.InvitationTTL.Duration())
}

// ListInvitations returns the tree's invitations with their status (AC73).
// Owner only.
func (u *GenealogyAccessUsecase) ListInvitations(ctx context.Context, userID, genealogyID int64) ([]domain.Invitation, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	items, err := u.invitations.ListByGenealogy(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.Invitation{}
	}
	return items, nil
}

// CancelInvitation cancels a pending invitation (AC74). Owner only.
func (u *GenealogyAccessUsecase) CancelInvitation(ctx context.Context, userID, genealogyID, invitationID int64) error {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return err
	}
	if invitationID <= 0 {
		return domain.ErrInvitationNotFound
	}
	return u.invitations.Cancel(ctx, genealogyID, invitationID)
}

// ListViewers returns the viewers with the member each one stands at (AC48).
// Owner only.
func (u *GenealogyAccessUsecase) ListViewers(ctx context.Context, userID, genealogyID int64) ([]domain.Viewer, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	items, err := u.viewers.List(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.Viewer{}
	}
	return items, nil
}

// RemoveViewer removes a viewer's access (AC49, AC50). The owner may remove any
// viewer; a viewer may remove only themself, which is leaving the tree. The
// owner cannot leave: asking for themself is "not a viewer".
func (u *GenealogyAccessUsecase) RemoveViewer(ctx context.Context, userID, genealogyID, targetUserID int64) error {
	role, err := resolveRole(ctx, u.genealogies, userID, genealogyID)
	if err != nil {
		return err
	}
	switch role {
	case domain.RoleOwner:
		if targetUserID == userID {
			return domain.ErrViewerNotFound
		}
	default:
		if targetUserID != userID {
			return domain.ErrGenealogyForbidden
		}
	}
	return u.viewers.Remove(ctx, genealogyID, targetUserID)
}
