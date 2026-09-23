package middleware

import (
	"context"
	"net/http"
	"strconv"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/utils"
)

// Context keys
type contextKey string

const (
	// ContextKeyHTTPPath is the key for the HTTP path in the context
	ContextKeyHTTPPath contextKey = "ratelimit_http_path"
	// ContextKeyHTTPMethod is the key for the HTTP method in the context
	ContextKeyHTTPMethod contextKey = "ratelimit_http_method"
)

// HTTPMiddleware provides HTTP middleware for rate limiting
type HTTPMiddleware struct {
	limiter *Limiter
}

// NewHTTPMiddleware creates a new HTTP middleware
func NewHTTPMiddleware(config ratelimit.Config) (*HTTPMiddleware, error) {
	limiter, err := NewLimiter(config)
	if err != nil {
		return nil, err
	}
	return &HTTPMiddleware{
		limiter: limiter,
	}, nil
}

// Handler wraps an http.Handler with rate limiting
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Inject path and method into context
		ctx := context.WithValue(r.Context(), ContextKeyHTTPPath, r.URL.Path)
		ctx = context.WithValue(ctx, ContextKeyHTTPMethod, r.Method)

		res, err := m.limiter.Allow(ctx)
		if err != nil {
			// Fail open on error
			next.ServeHTTP(w, r)
			return
		}

		// Set headers
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(m.limiter.config.Rate, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(res.Remaining, 10))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(res.ResetTime.Unix(), 10))

		if !res.Allowed {
			retryAfter := int64(res.RetryAfter.Seconds())
			w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
			utils.WriteBusinessError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests",
				map[string]any{"retry_after_seconds": retryAfter})
			return
		}

		next.ServeHTTP(w, r)
	})
}
