package usecase

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func photoConfig() config.MemorialConfig {
	return config.MemorialConfig{
		MaxPerUser:    100,
		PhotoMaxBytes: 5 << 20,
		PhotoURLTTL:   config.Duration(15 * time.Minute),
		PhotoPrefix:   "memorials/",
	}
}

func TestMemorialUsecase_SetPhoto(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	store := new(domain.MockMemorialPhotoStore)
	m := &domain.Memorial{ID: 9, UserID: 7, FullName: "A"}

	repo.On("FindByIDAndUser", mock.Anything, int64(9), int64(7)).Return(m, nil).Twice()
	store.On("Put", mock.Anything, mock.MatchedBy(func(k string) bool {
		return len(k) > len("memorials/7/9/") && k[:len("memorials/7/9/")] == "memorials/7/9/"
	}), mock.Anything, "image/png", int64(4)).Return(nil)
	repo.On("SetPhotoKey", mock.Anything, int64(9), int64(7), mock.Anything).Return(nil, nil)

	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(store)
	_, err := uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader([]byte("png!")), "image/png", 4)

	require.NoError(t, err)
	store.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestMemorialUsecase_SetPhoto_StoreFailureKeepsKey(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	store := new(domain.MockMemorialPhotoStore)
	repo.On("FindByIDAndUser", mock.Anything, int64(9), int64(7)).Return(&domain.Memorial{ID: 9, UserID: 7}, nil)
	store.On("Put", mock.Anything, mock.Anything, mock.Anything, "image/jpeg", int64(3)).Return(assert.AnError)

	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(store)
	_, err := uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader([]byte("jpg")), "image/jpeg", 3)

	require.Error(t, err)
	repo.AssertNotCalled(t, "SetPhotoKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMemorialUsecase_SetPhoto_RejectsLargeAndWrongType(t *testing.T) {
	uc := NewMemorialUsecase(new(domain.MockMemorialRepository), photoConfig()).WithPhotoStore(new(domain.MockMemorialPhotoStore))

	_, err := uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader(nil), "image/png", 6<<20)
	assert.ErrorIs(t, err, domain.ErrPhotoTooLarge)

	_, err = uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader(nil), "application/pdf", 10)
	assert.ErrorIs(t, err, domain.ErrPhotoUnsupportedType)
}

func TestMemorialUsecase_RemovePhotoDeletesOldObject(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	store := new(domain.MockMemorialPhotoStore)
	old := "memorials/7/9/old.jpg"

	repo.On("SetPhotoKey", mock.Anything, int64(9), int64(7), (*string)(nil)).Return(&old, nil)
	store.On("Delete", mock.Anything, old).Return(nil)
	repo.On("FindByIDAndUser", mock.Anything, int64(9), int64(7)).Return(&domain.Memorial{ID: 9, UserID: 7}, nil)

	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(store)
	_, err := uc.RemovePhoto(context.Background(), 7, 9)

	require.NoError(t, err)
	store.AssertExpectations(t)
}

func TestMemorialUsecase_DeleteRemovesPhotoObject(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	store := new(domain.MockMemorialPhotoStore)
	key := "memorials/7/9/a.jpg"
	repo.On("Delete", mock.Anything, int64(9), int64(7)).Return(&key, nil)
	store.On("Delete", mock.Anything, key).Return(nil)

	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(store)
	require.NoError(t, uc.Delete(context.Background(), 7, 9))
	store.AssertExpectations(t)
}
