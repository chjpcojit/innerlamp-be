package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockAvatarStorage struct {
	mock.Mock
}

func (m *mockAvatarStorage) Upload(ctx context.Context, objectName string, reader io.Reader, contentType string) (string, error) {
	args := m.Called(ctx, objectName, reader, contentType)
	return args.String(0), args.Error(1)
}

// TestUpdateAvatar_Success covers AC3: the image is uploaded under a
// deterministic per-user object key and the returned URL is persisted.
func TestUpdateAvatar_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	storage := &mockAvatarStorage{}
	logger := newLogger()

	reader := strings.NewReader("fake-image-bytes")
	storage.On("Upload", mock.Anything, "avatars/7", reader, "image/jpeg").
		Return("https://cdn.example.com/avatars/7", nil)
	repo.On("UpdateAvatarURL", mock.Anything, int64(7), "https://cdn.example.com/avatars/7").
		Return(&domain.User{ID: 7, AvatarURL: "https://cdn.example.com/avatars/7"}, nil)

	uc := NewAvatarUsecase(repo, storage, logger)

	got, err := uc.UpdateAvatar(context.Background(), 7, reader, "image/jpeg")
	if err != nil {
		t.Fatalf("UpdateAvatar() unexpected error: %v", err)
	}
	if got.AvatarURL != "https://cdn.example.com/avatars/7" {
		t.Errorf("UpdateAvatar() = %+v, want AvatarURL set", got)
	}
	storage.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestUpdateAvatar_UploadFailure ensures a storage failure is wrapped and
// propagated without touching the repository.
func TestUpdateAvatar_UploadFailure(t *testing.T) {
	repo := &domain.MockUserRepository{}
	storage := &mockAvatarStorage{}
	logger := newLogger()

	storage.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", errors.New("bucket unreachable"))

	uc := NewAvatarUsecase(repo, storage, logger)

	if _, err := uc.UpdateAvatar(context.Background(), 7, strings.NewReader(""), "image/jpeg"); err == nil {
		t.Fatalf("UpdateAvatar() expected error when upload fails")
	}
	repo.AssertNotCalled(t, "UpdateAvatarURL", mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateAvatar_RepositoryErrorPropagates ensures a persistence failure
// after a successful upload still surfaces as an error.
func TestUpdateAvatar_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockUserRepository{}
	storage := &mockAvatarStorage{}
	logger := newLogger()

	storage.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("https://cdn.example.com/avatars/7", nil)
	repo.On("UpdateAvatarURL", mock.Anything, int64(7), "https://cdn.example.com/avatars/7").
		Return(nil, domain.ErrUserNotFound)

	uc := NewAvatarUsecase(repo, storage, logger)

	_, err := uc.UpdateAvatar(context.Background(), 7, strings.NewReader(""), "image/jpeg")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("UpdateAvatar() error = %v, want %v", err, domain.ErrUserNotFound)
	}
}
