package repository

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"chjpcoj.com.innerlamp/pkg/storage"
)

// fakeStorageClient implements storage.Client, exercising only the Upload
// method the avatar adapter under test calls — same narrow-fake convention
// as fakeDatabase above.
type fakeStorageClient struct {
	uploadBucket  string
	uploadObject  string
	uploadOpts    *storage.UploadOptions
	uploadErr     error
	uploadCalled  bool
	uploadContent string
}

func (f *fakeStorageClient) Connect(ctx context.Context) error { return nil }
func (f *fakeStorageClient) Close() error                      { return nil }
func (f *fakeStorageClient) IsConnected() bool                 { return true }

func (f *fakeStorageClient) Upload(ctx context.Context, bucket, objectName string, reader io.Reader, opts *storage.UploadOptions) (*storage.ObjectInfo, error) {
	f.uploadCalled = true
	f.uploadBucket = bucket
	f.uploadObject = objectName
	f.uploadOpts = opts
	body, _ := io.ReadAll(reader)
	f.uploadContent = string(body)
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return &storage.ObjectInfo{Bucket: bucket, Name: objectName}, nil
}

func (f *fakeStorageClient) Download(ctx context.Context, bucket, objectName string, writer io.Writer, opts *storage.DownloadOptions) (*storage.ObjectInfo, error) {
	return nil, errors.New("fakeStorageClient.Download not implemented")
}
func (f *fakeStorageClient) Delete(ctx context.Context, bucket, objectName string) error {
	return errors.New("fakeStorageClient.Delete not implemented")
}
func (f *fakeStorageClient) GetObjectInfo(ctx context.Context, bucket, objectName string) (*storage.ObjectInfo, error) {
	return nil, errors.New("fakeStorageClient.GetObjectInfo not implemented")
}
func (f *fakeStorageClient) UpdateMetadata(ctx context.Context, bucket, objectName string, metadata map[string]string) (*storage.ObjectInfo, error) {
	return nil, errors.New("fakeStorageClient.UpdateMetadata not implemented")
}
func (f *fakeStorageClient) Copy(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, opts *storage.CopyOptions) (*storage.ObjectInfo, error) {
	return nil, errors.New("fakeStorageClient.Copy not implemented")
}
func (f *fakeStorageClient) Exists(ctx context.Context, bucket, objectName string) (bool, error) {
	return false, errors.New("fakeStorageClient.Exists not implemented")
}
func (f *fakeStorageClient) List(ctx context.Context, bucket string, opts *storage.ListOptions) (*storage.ListResult, error) {
	return nil, errors.New("fakeStorageClient.List not implemented")
}
func (f *fakeStorageClient) ListAll(ctx context.Context, bucket string, opts *storage.ListOptions) ([]*storage.ObjectInfo, error) {
	return nil, errors.New("fakeStorageClient.ListAll not implemented")
}
func (f *fakeStorageClient) CreateBucket(ctx context.Context, bucket string, opts *storage.BucketCreateOptions) error {
	return errors.New("fakeStorageClient.CreateBucket not implemented")
}
func (f *fakeStorageClient) DeleteBucket(ctx context.Context, bucket string) error {
	return errors.New("fakeStorageClient.DeleteBucket not implemented")
}
func (f *fakeStorageClient) GetBucketInfo(ctx context.Context, bucket string) (*storage.BucketInfo, error) {
	return nil, errors.New("fakeStorageClient.GetBucketInfo not implemented")
}
func (f *fakeStorageClient) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return false, errors.New("fakeStorageClient.BucketExists not implemented")
}
func (f *fakeStorageClient) ListBuckets(ctx context.Context) ([]*storage.BucketInfo, error) {
	return nil, errors.New("fakeStorageClient.ListBuckets not implemented")
}
func (f *fakeStorageClient) SignedURL(ctx context.Context, bucket, objectName string, opts *storage.SignedURLOptions) (string, error) {
	return "", errors.New("fakeStorageClient.SignedURL not implemented")
}
func (f *fakeStorageClient) GetBackend() storage.Backend { return storage.BackendGCS }
func (f *fakeStorageClient) GetConfig() *storage.Config  { return &storage.Config{} }

// TestGCSAvatarStorage_Upload covers US-004 AC3: the adapter uploads to the
// configured bucket with public-read ACL and returns the object's public URL.
func TestGCSAvatarStorage_Upload(t *testing.T) {
	client := &fakeStorageClient{}
	s := NewGCSAvatarStorage(client, "innerlamp-avatars")

	url, err := s.Upload(context.Background(), "avatars/7", strings.NewReader("fake-image-bytes"), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload() unexpected error: %v", err)
	}
	if url != "https://storage.googleapis.com/innerlamp-avatars/avatars/7" {
		t.Errorf("Upload() url = %q, want public GCS object URL", url)
	}
	if !client.uploadCalled || client.uploadBucket != "innerlamp-avatars" || client.uploadObject != "avatars/7" {
		t.Errorf("Upload() called with bucket=%q object=%q, want innerlamp-avatars/avatars/7", client.uploadBucket, client.uploadObject)
	}
	if client.uploadOpts.ContentType != "image/jpeg" || client.uploadOpts.ACL != storage.ACLPublicRead {
		t.Errorf("Upload() opts = %+v, want ContentType=image/jpeg ACL=public-read", client.uploadOpts)
	}
	if client.uploadContent != "fake-image-bytes" {
		t.Errorf("Upload() content = %q, want fake-image-bytes", client.uploadContent)
	}
}

// TestGCSAvatarStorage_UploadFailure wraps and propagates client errors.
func TestGCSAvatarStorage_UploadFailure(t *testing.T) {
	client := &fakeStorageClient{uploadErr: errors.New("bucket unreachable")}
	s := NewGCSAvatarStorage(client, "innerlamp-avatars")

	if _, err := s.Upload(context.Background(), "avatars/7", strings.NewReader(""), "image/jpeg"); err == nil {
		t.Fatalf("Upload() expected error when client upload fails")
	}
}
