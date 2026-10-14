package main

import (
	"context"
	"fmt"
	"time"

	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
)

// emulatorBootstrapTimeout bounds ensureBucket. It has its own budget on purpose:
// the GCS client retries a refused connection until its context ends, so sharing
// main's startup context would let a missing emulator eat the time the database
// seeds still need. Dev-only, hence a constant rather than config (TD-008 style).
const emulatorBootstrapTimeout = 3 * time.Second

// ensureBucket creates bucket on the storage emulator if it is missing. A real
// bucket is created by infrastructure, never by the service, so this is only
// called in emulator mode.
func ensureBucket(c pkgstorage.Client, bucket string) error {
	ctx, cancel := context.WithTimeout(context.Background(), emulatorBootstrapTimeout)
	defer cancel()
	exists, err := c.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", bucket, err)
	}
	if exists {
		return nil
	}
	if err := c.CreateBucket(ctx, bucket, &pkgstorage.BucketCreateOptions{}); err != nil {
		return fmt.Errorf("create bucket %q: %w", bucket, err)
	}
	return nil
}
