// Package storage holds appService's adapters over pkg/storage: the one place
// that knows which bucket the service's objects live in and how a domain port
// maps onto the generic storage client.
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// contentLengthRangeHeader makes the storage service itself refuse an upload
// outside the signed byte range. It is part of the signature, so the client must
// send it with exactly the value returned in the ticket.
const contentLengthRangeHeader = "x-goog-content-length-range"

// AvatarStorage implements domain.AvatarStorage on a pkg/storage.Client for one
// bucket. The client is built and connected by the composition root.
//
// It has two modes. By default URLs are signed and point at the real bucket.
// With WithEmulator they are plain URLs to a storage emulator (dev and tests),
// which needs no signature — so none is made.
type AvatarStorage struct {
	client pkgstorage.Client
	bucket string
	// emulatorHost is "host:port" of a plain-HTTP emulator; empty = real bucket.
	emulatorHost string
}

var _ domain.AvatarStorage = (*AvatarStorage)(nil)

// Option configures an AvatarStorage.
type Option func(*AvatarStorage)

// WithEmulator makes SignUpload and SignRead return unsigned URLs to the
// emulator at host ("localhost:4443", plain HTTP) instead of signing for a real
// bucket. Stat, Copy and Delete are unaffected: they go through the client,
// which reaches the same emulator via STORAGE_EMULATOR_HOST.
//
// ⚠️ DEV ONLY. Unsigned means the store enforces neither the 5 MB limit, the
// content type nor the URL lifetime. The use case still checks the recorded type
// and size on confirm, but a real deploy needs real signing (TD-001).
func WithEmulator(host string) Option {
	return func(a *AvatarStorage) { a.emulatorHost = host }
}

// NewAvatarStorage wires the adapter. bucket is profile.bucket.
func NewAvatarStorage(client pkgstorage.Client, bucket string, opts ...Option) *AvatarStorage {
	a := &AvatarStorage{client: client, bucket: bucket}
	for _, o := range opts {
		o(a)
	}
	return a
}

// SignUpload returns how to upload key.
//
// Real bucket: a V4-signed PUT limited to contentType and maxBytes.
// TD-001: only the emulator has exercised this so far, and the emulator does not
// verify V4 signatures, signed headers, length ranges or expiry, so the real
// behaviour must be checked on a real bucket before the first deploy.
//
// Emulator: an unsigned POST to its JSON upload endpoint (the emulator has no
// PUT upload), so the caller must use the returned Method, not assume PUT.
func (a *AvatarStorage) SignUpload(ctx context.Context, key, contentType string, maxBytes int64, ttl time.Duration) (*domain.SignedUpload, error) {
	if a.emulatorHost != "" {
		q := url.Values{"uploadType": {"media"}, "name": {key}}
		return &domain.SignedUpload{
			URL:     fmt.Sprintf("http://%s/upload/storage/v1/b/%s/o?%s", a.emulatorHost, url.PathEscape(a.bucket), q.Encode()),
			Method:  "POST",
			Headers: map[string]string{"Content-Type": contentType},
		}, nil
	}

	headers := map[string]string{
		"Content-Type":           contentType,
		contentLengthRangeHeader: "0," + strconv.FormatInt(maxBytes, 10),
	}
	signed, err := a.client.SignedURL(ctx, a.bucket, key, &pkgstorage.SignedURLOptions{
		Method:      pkgstorage.SignedURLMethodPUT,
		Expires:     ttl,
		ContentType: contentType,
		// Content-Type travels as ContentType; only the extra header is listed
		// here so it is not signed twice.
		Headers: map[string]string{contentLengthRangeHeader: headers[contentLengthRangeHeader]},
	})
	if err != nil {
		return nil, fmt.Errorf("sign upload %q: %w", key, err)
	}
	return &domain.SignedUpload{URL: signed, Method: "PUT", Headers: headers}, nil
}

// SignRead returns a GET address for key: signed for a real bucket (see
// TD-001), a plain download URL on the emulator.
func (a *AvatarStorage) SignRead(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if a.emulatorHost != "" {
		return fmt.Sprintf("http://%s/download/storage/v1/b/%s/o/%s?alt=media",
			a.emulatorHost, url.PathEscape(a.bucket), url.PathEscape(key)), nil
	}
	signed, err := a.client.SignedURL(ctx, a.bucket, key, &pkgstorage.SignedURLOptions{
		Method:  pkgstorage.SignedURLMethodGET,
		Expires: ttl,
	})
	if err != nil {
		return "", fmt.Errorf("sign read %q: %w", key, err)
	}
	return signed, nil
}

// Stat returns the content type and size storage recorded for key.
func (a *AvatarStorage) Stat(ctx context.Context, key string) (*domain.ObjectMeta, error) {
	info, err := a.client.GetObjectInfo(ctx, a.bucket, key)
	if err != nil {
		if errors.Is(err, pkgstorage.ErrObjectNotFound) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, fmt.Errorf("stat %q: %w", key, err)
	}
	return &domain.ObjectMeta{ContentType: info.ContentType, Size: info.Size}, nil
}

// Copy duplicates srcKey to dstKey inside the bucket.
func (a *AvatarStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	if _, err := a.client.Copy(ctx, a.bucket, srcKey, a.bucket, dstKey, nil); err != nil {
		if errors.Is(err, pkgstorage.ErrObjectNotFound) {
			return domain.ErrObjectNotFound
		}
		return fmt.Errorf("copy %q to %q: %w", srcKey, dstKey, err)
	}
	return nil
}

// Delete removes key. A key that is already gone is success: the caller wants
// the object absent, and it is.
//
// TD-002: callers treat a failure here as "leave the file behind", not as a
// reason to fail the request; nothing cleans such files up yet.
func (a *AvatarStorage) Delete(ctx context.Context, key string) error {
	if err := a.client.Delete(ctx, a.bucket, key); err != nil && !errors.Is(err, pkgstorage.ErrObjectNotFound) {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	return nil
}
