package ratelimit

import (
	"context"
	"time"
)

// Algorithm defines the rate limiting algorithm to use
type Algorithm string

const (
	// TokenBucket algorithm
	TokenBucket Algorithm = "token_bucket"
	// SlidingWindow algorithm
	SlidingWindow Algorithm = "sliding_window"
	// FixedWindow algorithm
	FixedWindow Algorithm = "fixed_window"
	// LeakyBucket algorithm
	LeakyBucket Algorithm = "leaky_bucket"
)

// Config defines the configuration for the rate limiter
type Config struct {
	// Rate is the number of requests allowed per time window
	Rate int64
	// Burst is the maximum number of requests allowed at once (for Token Bucket)
	Burst int64
	// Window is the time window for the rate limit
	Window time.Duration
	// Algorithm is the rate limiting algorithm to use
	Algorithm Algorithm
	// Store is the storage backend to use
	Store Store
	// KeyFunc is the function to extract the key from the request
	KeyFunc func(ctx context.Context) (string, error)
}

// Result represents the result of a rate limit check
type Result struct {
	// Allowed indicates if the request is allowed
	Allowed bool
	// Remaining is the number of requests remaining in the current window
	Remaining int64
	// ResetTime is the time when the rate limit will reset
	ResetTime time.Time
	// RetryAfter is the duration to wait before retrying (if Allowed is false)
	RetryAfter time.Duration
}

// Store defines the interface for rate limit storage backends
type Store interface {
	// Allow checks if a request is allowed and updates the state
	Allow(ctx context.Context, key string, limit int64, window time.Duration, burst int64, algo Algorithm) (*Result, error)
}
