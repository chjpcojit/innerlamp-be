package usecase

import (
	"context"
	"fmt"
	"io"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// AvatarStorage is the subset of an object-storage client this use-case
// needs, defined here (the consumer) so it can be unit tested with a mock
// instead of depending on the full pkg/storage.Client interface.
type AvatarStorage interface {
	// Upload stores reader's content under objectName and returns the URL
	// the uploaded object is reachable at.
	Upload(ctx context.Context, objectName string, reader io.Reader, contentType string) (url string, err error)
}

// AvatarUsecase implements US-004 AC3 (avatar upload).
type AvatarUsecase struct {
	users   domain.UserRepository
	storage AvatarStorage
	logger  logging.Logger
}

// NewAvatarUsecase wires the avatar use-case with its dependencies.
func NewAvatarUsecase(users domain.UserRepository, storage AvatarStorage, logger logging.Logger) *AvatarUsecase {
	return &AvatarUsecase{users: users, storage: storage, logger: logger}
}

// UpdateAvatar uploads a new avatar image and persists its URL — AC3. The
// object key is deterministic per user (avatars/{userID}), so re-uploading
// naturally overwrites the previous image in storage rather than leaving
// orphaned files behind.bạn 
func (uc *AvatarUsecase) UpdateAvatar(ctx context.Context, userID int64, reader io.Reader, contentType string) (*domain.User, error) {
	objectName := fmt.Sprintf("avatars/%d", userID)

	url, err := uc.storage.Upload(ctx, objectName, reader, contentType)
	if err != nil {
		return nil, fmt.Errorf("upload avatar: %w", err)
	}

	user, err := uc.users.UpdateAvatarURL(ctx, userID, url)
	if err != nil {
		return nil, err
	}

	uc.logger.WithContext(ctx).WithFields(logging.Fields{"user_id": userID}).Info("avatar updated")

	return user, nil
}
