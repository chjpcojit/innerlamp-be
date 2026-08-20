package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// ProfileUsecase implements US-004 (Cập nhật hồ sơ cá nhân).
type ProfileUsecase struct {
	users  domain.UserRepository
	logger logging.Logger
}

// NewProfileUsecase wires the profile use-case with its dependencies.
func NewProfileUsecase(users domain.UserRepository, logger logging.Logger) *ProfileUsecase {
	return &ProfileUsecase{users: users, logger: logger}
}

// GetProfile returns the full profile for an authenticated user — AC1.
func (uc *ProfileUsecase) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	return uc.users.FindByID(ctx, userID)
}

// UpdateProfile overwrites dharma_name/ordination_place for userID — AC2.
// AC5: an empty field in input clears that column, since input is always
// written as given rather than merged against the existing row. AC4 (email
// and phone number are not editable here) is enforced structurally: input
// has no fields for them.
func (uc *ProfileUsecase) UpdateProfile(ctx context.Context, userID int64, input domain.ProfileUpdateInput) (*domain.User, error) {
	user, err := uc.users.UpdateProfile(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	uc.logger.WithContext(ctx).WithFields(logging.Fields{"user_id": userID}).Info("profile updated")

	return user, nil
}
