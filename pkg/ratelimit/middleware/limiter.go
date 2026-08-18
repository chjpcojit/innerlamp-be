package middleware

import (
	"context"
	"errors"

	"base_project/pkg/ratelimit"
	"base_project/pkg/ratelimit/store"
)

// Limiter controls rate limiting logic
type Limiter struct {
	config ratelimit.Config
}

// NewLimiter creates a new Limiter
func NewLimiter(config ratelimit.Config) (*Limiter, error) {
	if config.Store == nil {
		config.Store = store.NewMemoryStore()
	}
	if config.Rate <= 0 {
		return nil, errors.New("rate must be positive")
	}
	if config.Window <= 0 {
		return nil, errors.New("window must be positive")
	}
	if config.Algorithm == "" {
		config.Algorithm = ratelimit.TokenBucket
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate // Default burst to rate
	}

	return &Limiter{
		config: config,
	}, nil
}

// Allow checks if the request is allowed
func (l *Limiter) Allow(ctx context.Context) (*ratelimit.Result, error) {
	key := ""
	if l.config.KeyFunc != nil {
		var err error
		key, err = l.config.KeyFunc(ctx)
		if err != nil {
			return nil, err
		}
	}

	// If no key, maybe skip or use default?
	// User requirement: "Rate limit by IP address, API key, or User ID"
	// If KeyFunc returns empty, we probably shouldn't limit or limit globally?
	// Let's assume empty key means "global" or "unknown".
	if key == "" {
		key = "global"
	}

	return l.config.Store.Allow(ctx, key, l.config.Rate, l.config.Window, l.config.Burst, l.config.Algorithm)
}
