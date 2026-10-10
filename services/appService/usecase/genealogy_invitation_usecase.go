package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GenealogyInvitationUsecase is the invitee's side: the inbox, accept and
// decline (AC70–AC72). It does not go through role resolution — the caller has
// no role yet — and instead matches invitations by the caller's own phone
// number (AC75).
type GenealogyInvitationUsecase struct {
	invitations domain.InvitationRepository
	phones      domain.UserPhoneLookup
	cfg         config.GenealogyConfig
}

// NewGenealogyInvitationUsecase wires the use case.
func NewGenealogyInvitationUsecase(
	invitations domain.InvitationRepository,
	phones domain.UserPhoneLookup,
	cfg config.GenealogyConfig,
) *GenealogyInvitationUsecase {
	return &GenealogyInvitationUsecase{invitations: invitations, phones: phones, cfg: cfg}
}

// ListMine returns the pending, unexpired invitations addressed to the caller's
// own phone number (AC70).
func (u *GenealogyInvitationUsecase) ListMine(ctx context.Context, userID int64) ([]domain.Invitation, error) {
	phone, err := u.phones.FindPhoneByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	items, err := u.invitations.ListMine(ctx, phone)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.Invitation{}
	}
	return items, nil
}

// Accept makes the caller a viewer of the invited tree and marks the
// invitation accepted (AC71).
func (u *GenealogyInvitationUsecase) Accept(ctx context.Context, userID, invitationID int64) (*domain.AcceptResult, error) {
	if invitationID <= 0 {
		return nil, domain.ErrInvitationNotFound
	}
	phone, err := u.phones.FindPhoneByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return u.invitations.Accept(ctx, invitationID, userID, phone, u.cfg.MaxViewers)
}

// Decline marks the invitation declined without granting anything (AC72).
func (u *GenealogyInvitationUsecase) Decline(ctx context.Context, userID, invitationID int64) error {
	if invitationID <= 0 {
		return domain.ErrInvitationNotFound
	}
	phone, err := u.phones.FindPhoneByID(ctx, userID)
	if err != nil {
		return err
	}
	return u.invitations.Decline(ctx, invitationID, phone)
}
