package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// fakeClient implements only the five pkg/storage.Client methods the adapter
// uses; embedding the interface makes any other call panic, which is the point:
// the adapter must not reach for anything else.
type fakeClient struct {
	pkgstorage.Client

	signed       *signedCall
	signErr      error
	info         *pkgstorage.ObjectInfo
	infoErr      error
	copied       *copyCall
	copyErr      error
	deleted      []string
	deleteErr    error
	seenInfoCall string
}

type signedCall struct {
	bucket, object string
	opts           pkgstorage.SignedURLOptions
}

type copyCall struct{ srcBucket, srcObject, dstBucket, dstObject string }

func (f *fakeClient) SignedURL(_ context.Context, bucket, object string, opts *pkgstorage.SignedURLOptions) (string, error) {
	f.signed = &signedCall{bucket: bucket, object: object, opts: *opts}
	if f.signErr != nil {
		return "", f.signErr
	}
	return "https://storage.example/" + bucket + "/" + object, nil
}

func (f *fakeClient) GetObjectInfo(_ context.Context, bucket, object string) (*pkgstorage.ObjectInfo, error) {
	f.seenInfoCall = bucket + "/" + object
	return f.info, f.infoErr
}

func (f *fakeClient) Copy(_ context.Context, sb, so, db, do string, _ *pkgstorage.CopyOptions) (*pkgstorage.ObjectInfo, error) {
	f.copied = &copyCall{sb, so, db, do}
	return nil, f.copyErr
}

func (f *fakeClient) Delete(_ context.Context, bucket, object string) error {
	f.deleted = append(f.deleted, bucket+"/"+object)
	return f.deleteErr
}

func newAdapter(c *fakeClient) *AvatarStorage {
	return NewAvatarStorage(c, "bucket-dev")
}

func TestAvatarStorage_SignUpload_BindsMethodTypeAndLength(t *testing.T) {
	c := &fakeClient{}

	up, err := newAdapter(c).SignUpload(context.Background(),
		"avatars/tmp/42/u1", "image/jpeg", 5242880, 5*time.Minute)

	require.NoError(t, err)
	assert.Equal(t, "https://storage.example/bucket-dev/avatars/tmp/42/u1", up.URL)
	assert.Equal(t, "PUT", up.Method)
	require.NotNil(t, c.signed)
	assert.Equal(t, "bucket-dev", c.signed.bucket)
	assert.Equal(t, "avatars/tmp/42/u1", c.signed.object)
	assert.Equal(t, pkgstorage.SignedURLMethodPUT, c.signed.opts.Method)
	assert.Equal(t, 5*time.Minute, c.signed.opts.Expires)
	assert.Equal(t, "image/jpeg", c.signed.opts.ContentType)
	assert.Equal(t, "0,5242880", c.signed.opts.Headers["x-goog-content-length-range"])
	// What the client must send is exactly what was signed.
	assert.Equal(t, map[string]string{
		"Content-Type":                "image/jpeg",
		"x-goog-content-length-range": "0,5242880",
	}, up.Headers)
}

func TestAvatarStorage_SignUpload_Error(t *testing.T) {
	c := &fakeClient{signErr: errors.New("no key")}

	_, err := newAdapter(c).SignUpload(context.Background(), "k", "image/png", 10, time.Minute)

	assert.Error(t, err)
}

// Emulator mode is the dev/test path: nothing is signed, because the emulator
// neither needs nor checks a signature. The same limits are therefore NOT
// enforced by the store (TD-001); the use case still re-checks the recorded
// type and size on confirm.
func TestAvatarStorage_EmulatorMode_UploadIsAnUnsignedPostToTheEmulator(t *testing.T) {
	c := &fakeClient{}
	a := NewAvatarStorage(c, "bucket-dev", WithEmulator("localhost:4443"))

	up, err := a.SignUpload(context.Background(), "avatars/tmp/42/u1", "image/png", 5242880, time.Minute)

	require.NoError(t, err)
	assert.Equal(t,
		"http://localhost:4443/upload/storage/v1/b/bucket-dev/o?name=avatars%2Ftmp%2F42%2Fu1&uploadType=media", up.URL)
	assert.Equal(t, "POST", up.Method, "the emulator's JSON upload endpoint takes POST, not PUT")
	assert.Equal(t, map[string]string{"Content-Type": "image/png"}, up.Headers)
	assert.Nil(t, c.signed, "nothing may be signed in emulator mode")
}

func TestAvatarStorage_EmulatorMode_ReadIsAnUnsignedDownloadURL(t *testing.T) {
	c := &fakeClient{}
	a := NewAvatarStorage(c, "bucket-dev", WithEmulator("localhost:4443"))

	url, err := a.SignRead(context.Background(), "avatars/42/a b.jpg", time.Minute)

	require.NoError(t, err)
	assert.Equal(t,
		"http://localhost:4443/download/storage/v1/b/bucket-dev/o/avatars%2F42%2Fa%20b.jpg?alt=media", url)
	assert.Nil(t, c.signed)
}

func TestAvatarStorage_EmulatorMode_StatCopyDeleteStillGoThroughTheClient(t *testing.T) {
	c := &fakeClient{info: &pkgstorage.ObjectInfo{ContentType: "image/png", Size: 3}}
	a := NewAvatarStorage(c, "bucket-dev", WithEmulator("localhost:4443"))

	_, err := a.Stat(context.Background(), "k")
	require.NoError(t, err)
	require.NoError(t, a.Copy(context.Background(), "a", "b"))
	require.NoError(t, a.Delete(context.Background(), "k"))
	assert.Equal(t, &copyCall{"bucket-dev", "a", "bucket-dev", "b"}, c.copied)
}

func TestAvatarStorage_SignRead(t *testing.T) {
	c := &fakeClient{}

	url, err := newAdapter(c).SignRead(context.Background(), "avatars/42/a.jpg", 15*time.Minute)

	require.NoError(t, err)
	assert.Equal(t, "https://storage.example/bucket-dev/avatars/42/a.jpg", url)
	assert.Equal(t, pkgstorage.SignedURLMethodGET, c.signed.opts.Method)
	assert.Equal(t, 15*time.Minute, c.signed.opts.Expires)
}

func TestAvatarStorage_Stat(t *testing.T) {
	t.Run("maps recorded type and size", func(t *testing.T) {
		c := &fakeClient{info: &pkgstorage.ObjectInfo{ContentType: "image/webp", Size: 1234}}
		meta, err := newAdapter(c).Stat(context.Background(), "avatars/tmp/42/u1")
		require.NoError(t, err)
		assert.Equal(t, &domain.ObjectMeta{ContentType: "image/webp", Size: 1234}, meta)
		assert.Equal(t, "bucket-dev/avatars/tmp/42/u1", c.seenInfoCall)
	})
	t.Run("missing object becomes the domain sentinel", func(t *testing.T) {
		c := &fakeClient{infoErr: pkgstorage.ErrObjectNotFound}
		_, err := newAdapter(c).Stat(context.Background(), "k")
		assert.ErrorIs(t, err, domain.ErrObjectNotFound)
	})
	t.Run("other errors are not mistaken for missing", func(t *testing.T) {
		c := &fakeClient{infoErr: errors.New("boom")}
		_, err := newAdapter(c).Stat(context.Background(), "k")
		require.Error(t, err)
		assert.NotErrorIs(t, err, domain.ErrObjectNotFound)
	})
}

func TestAvatarStorage_Copy_StaysInTheBucket(t *testing.T) {
	c := &fakeClient{}

	err := newAdapter(c).Copy(context.Background(), "avatars/tmp/42/u1", "avatars/42/n.jpg")

	require.NoError(t, err)
	assert.Equal(t, &copyCall{"bucket-dev", "avatars/tmp/42/u1", "bucket-dev", "avatars/42/n.jpg"}, c.copied)
}

func TestAvatarStorage_Copy_MissingSourceIsTheDomainSentinel(t *testing.T) {
	c := &fakeClient{copyErr: pkgstorage.ErrObjectNotFound}

	err := newAdapter(c).Copy(context.Background(), "a", "b")

	assert.ErrorIs(t, err, domain.ErrObjectNotFound)
}

func TestAvatarStorage_Delete(t *testing.T) {
	t.Run("deletes in the bucket", func(t *testing.T) {
		c := &fakeClient{}
		require.NoError(t, newAdapter(c).Delete(context.Background(), "avatars/42/a.jpg"))
		assert.Equal(t, []string{"bucket-dev/avatars/42/a.jpg"}, c.deleted)
	})
	t.Run("an already missing object is not an error", func(t *testing.T) {
		c := &fakeClient{deleteErr: pkgstorage.ErrObjectNotFound}
		assert.NoError(t, newAdapter(c).Delete(context.Background(), "k"))
	})
	t.Run("a real failure is reported", func(t *testing.T) {
		c := &fakeClient{deleteErr: errors.New("boom")}
		assert.Error(t, newAdapter(c).Delete(context.Background(), "k"))
	})
}
