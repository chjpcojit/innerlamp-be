package repository

import (
	"context"
	"fmt"
	"io"

	"chjpcoj.com.innerlamp/pkg/storage"
	"chjpcoj.com.innerlamp/services/appService/usecase"
)

// gcsAvatarStorage adapts a pkg/storage.Client to the usecase.AvatarStorage
// port (US-004 AC3), uploading avatar images to a fixed bucket with
// public-read ACL.
//
// Assumption: avatar images are served directly from the bucket's public
// object URL rather than through signed URLs or a CDN — the simplest option
// consistent with this repo's GCS-primary cloud bias (CLAUDE.md) and not
// contradicted by the spec, which only says "upload lên Cloud Storage, lưu
// avatar_url". Revisit if a CDN/signed-URL requirement shows up later.
type gcsAvatarStorage struct {
	client storage.Client
	bucket string
}

// NewGCSAvatarStorage returns a usecase.AvatarStorage backed by client,
// storing objects in bucket.
func NewGCSAvatarStorage(client storage.Client, bucket string) usecase.AvatarStorage {
	return &gcsAvatarStorage{client: client, bucket: bucket}
}

func (s *gcsAvatarStorage) Upload(ctx context.Context, objectName string, reader io.Reader, contentType string) (string, error) {
	_, err := s.client.Upload(ctx, s.bucket, objectName, reader, &storage.UploadOptions{
		ContentType: contentType,
		ACL:         storage.ACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("upload avatar object: %w", err)
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucket, objectName), nil
}
