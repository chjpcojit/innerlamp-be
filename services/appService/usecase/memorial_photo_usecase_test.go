package usecase

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strings"
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

// testJPEG is a real JPEG, because SetPhoto now decodes what it is given: a
// handful of fake bytes would be rejected as unsupported.
func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	return buf.Bytes()
}

func TestMemorialUsecase_SetPhoto(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	store := new(domain.MockMemorialPhotoStore)
	m := &domain.Memorial{ID: 9, UserID: 7, FullName: "A"}

	var stored []byte
	repo.On("FindByIDAndUser", mock.Anything, int64(9), int64(7)).Return(m, nil).Twice()
	store.On("Put", mock.Anything, mock.MatchedBy(func(k string) bool {
		return strings.HasPrefix(k, "memorials/7/9/") && strings.HasSuffix(k, ".jpg")
	}), mock.Anything, "image/jpeg", mock.AnythingOfType("int64")).
		Run(func(args mock.Arguments) {
			stored, _ = io.ReadAll(args.Get(2).(io.Reader))
		}).Return(nil)
	repo.On("SetPhotoKey", mock.Anything, int64(9), int64(7), mock.Anything).Return(nil, nil)

	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(store)
	_, err := uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader(testJPEG(t, 200, 100)))

	require.NoError(t, err)
	require.NotEmpty(t, stored, "the use case must hand the store the re-encoded image")
	assert.True(t, bytes.HasPrefix(stored, []byte{0xFF, 0xD8, 0xFF}), "stored bytes must be a JPEG")
	assert.NotContains(t, string(stored), "Exif")
	store.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestMemorialUsecase_SetPhoto_StoreFailureKeepsKey(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	store := new(domain.MockMemorialPhotoStore)
	repo.On("FindByIDAndUser", mock.Anything, int64(9), int64(7)).Return(&domain.Memorial{ID: 9, UserID: 7}, nil)
	store.On("Put", mock.Anything, mock.Anything, mock.Anything, "image/jpeg", mock.Anything).Return(assert.AnError)

	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(store)
	_, err := uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader(testJPEG(t, 60, 40)))

	require.Error(t, err)
	repo.AssertNotCalled(t, "SetPhotoKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMemorialUsecase_SetPhoto_RejectsLargeAndWrongType(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	repo.On("FindByIDAndUser", mock.Anything, int64(9), int64(7)).Return(&domain.Memorial{ID: 9, UserID: 7}, nil).Twice()
	uc := NewMemorialUsecase(repo, photoConfig()).WithPhotoStore(new(domain.MockMemorialPhotoStore))

	_, err := uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader(make([]byte, 5<<20+1)))
	assert.ErrorIs(t, err, domain.ErrPhotoTooLarge)

	_, err = uc.SetPhoto(context.Background(), 7, 9, bytes.NewReader([]byte("application/pdf")))
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
