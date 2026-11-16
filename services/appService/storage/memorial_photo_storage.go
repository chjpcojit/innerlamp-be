package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// MemorialPhotoStorage implements domain.MemorialPhotoStore on a pkg/storage
// client. It owns the bucket; the use case passes the full object key.
type MemorialPhotoStorage struct {
	client       pkgstorage.Client
	bucket       string
	emulatorHost string
}

var _ domain.MemorialPhotoStore = (*MemorialPhotoStorage)(nil)

// NewMemorialPhotoStorage wires the adapter. emulatorHost ("host:port", empty
// outside dev) makes SignedURL return a plain emulator URL, because the emulator
// does not verify V4 signatures.
func NewMemorialPhotoStorage(client pkgstorage.Client, bucket, emulatorHost string) *MemorialPhotoStorage {
	return &MemorialPhotoStorage{client: client, bucket: bucket, emulatorHost: emulatorHost}
}

// Put writes size bytes of r at key.
//
// The object is private and only ever reached through a 15-minute signed URL,
// so a shared cache must not store or transform it: private, max-age=900 (the
// signed URL lifetime) and no-transform keep proxies and the app's own HTTP
// cache from serving one user's photo to another.
func (s *MemorialPhotoStorage) Put(ctx context.Context, key string, r io.Reader, contentType string, size int64) error {
	if _, err := s.client.Upload(ctx, s.bucket, key, r, &pkgstorage.UploadOptions{
		ContentType:  contentType,
		CacheControl: "private, max-age=900, no-transform",
		Metadata:     map[string]string{"size": fmt.Sprintf("%d", size)},
	}); err != nil {
		return fmt.Errorf("upload %q: %w", key, err)
	}
	return nil
}

// Delete removes key; a key that is already gone is success.
func (s *MemorialPhotoStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.Delete(ctx, s.bucket, key); err != nil && !errors.Is(err, pkgstorage.ErrObjectNotFound) {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	return nil
}

// SignedURL returns a GET address for key valid for ttl.
func (s *MemorialPhotoStorage) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.emulatorHost != "" {
		return fmt.Sprintf("http://%s/download/storage/v1/b/%s/o/%s?alt=media",
			s.emulatorHost, url.PathEscape(s.bucket), url.PathEscape(key)), nil
	}
	signed, err := s.client.SignedURL(ctx, s.bucket, key, &pkgstorage.SignedURLOptions{
		Method:  pkgstorage.SignedURLMethodGET,
		Expires: ttl,
	})
	if err != nil {
		if pkgstorage.IsExpiredSignatureError(err) {
			// The store rejected the signature as stale: the caller must get a
			// fresh URL (403), not "not found".
			return "", fmt.Errorf("%w: %v", domain.ErrPhotoURLExpired, err)
		}
		return "", fmt.Errorf("sign read %q: %w", key, err)
	}
	return signed, nil
}
