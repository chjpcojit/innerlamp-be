package store

import (
	"errors"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

var (
	// ErrNotImplemented is returned when a feature is not implemented by the store
	ErrNotImplemented = errors.New("not implemented")
	// ErrRateLimitExceeded is returned when the rate limit is exceeded (optional, usually handled by Result)
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// Limiter defines the interface for rate limiting algorithms (stateful)
type Limiter interface {
	Allow(limit int64, window time.Duration, burst int64) *ratelimit.Result
}
