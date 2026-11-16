package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Upload records the options the adapter passed, so a test can assert on the
// headers that end up on the object.
func (f *fakeClient) Upload(_ context.Context, bucket, object string, r io.Reader, opts *pkgstorage.UploadOptions) (*pkgstorage.ObjectInfo, error) {
	body, _ := io.ReadAll(r)
	f.uploaded = opts
	f.uploadedBody = body
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return &pkgstorage.ObjectInfo{Bucket: bucket, Name: object}, nil
}

func TestMemorialPhotoStorage_PutSetsPrivateCacheControl(t *testing.T) {
	c := &fakeClient{}
	s := NewMemorialPhotoStorage(c, "innerlamp-avatars-dev", "")

	require.NoError(t, s.Put(context.Background(), "memorials/7/9/a.jpg", bytes.NewReader([]byte("jpeg")), "image/jpeg", 4))

	require.NotNil(t, c.uploaded)
	assert.Equal(t, "private, max-age=900, no-transform", c.uploaded.CacheControl,
		"a signed URL is the only way in, so the object must not be cached publicly")
	assert.Equal(t, "image/jpeg", c.uploaded.ContentType)
	assert.Equal(t, []byte("jpeg"), c.uploadedBody)
}

func TestMemorialPhotoStorage_PutWrapsUploadError(t *testing.T) {
	c := &fakeClient{uploadErr: pkgstorage.ErrObjectNotFound}
	s := NewMemorialPhotoStorage(c, "bucket", "")

	err := s.Put(context.Background(), "memorials/7/9/a.jpg", bytes.NewReader(nil), "image/jpeg", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkgstorage.ErrObjectNotFound)
}

// An expired V4 URL and a signature computed over different parameters both
// mean "ask for a fresh URL", which the handler answers with 403.
func TestMemorialPhotoStorage_SignedURLExpiredToken(t *testing.T) {
	c := &fakeClient{signErr: errors.New("storage: object get: ExpiredToken")}
	s := NewMemorialPhotoStorage(c, "bucket", "")

	_, err := s.SignedURL(context.Background(), "memorials/7/9/a.jpg", 15*time.Minute)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPhotoURLExpired)
}

func TestMemorialPhotoStorage_SignedURLSignatureMismatch(t *testing.T) {
	c := &fakeClient{signErr: errors.New("storage: SignatureDoesNotMatch")}
	s := NewMemorialPhotoStorage(c, "bucket", "")

	_, err := s.SignedURL(context.Background(), "memorials/7/9/a.jpg", 15*time.Minute)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPhotoURLExpired)
}

func TestMemorialPhotoStorage_SignedURLOtherErrorStaysGeneric(t *testing.T) {
	c := &fakeClient{signErr: errors.New("no key")}
	s := NewMemorialPhotoStorage(c, "bucket", "")

	_, err := s.SignedURL(context.Background(), "memorials/7/9/a.jpg", 15*time.Minute)

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrPhotoURLExpired)
}
