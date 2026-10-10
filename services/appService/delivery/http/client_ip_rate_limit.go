package http

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	ratelimitmw "chjpcoj.com.innerlamp/pkg/ratelimit/middleware"
)

type clientIPCtxKey struct{}

// ClientIPRateLimit limits a route per client IP using pkg/ratelimit. It exists
// because that package's HTTP middleware hands KeyFunc only the path and method:
// this wrapper resolves the IP first, puts it on the context, and keys on it.
//
// The counters live in memory, so the limit is per service instance. That is a
// deliberate floor, not an exact quota: it stops one address from flooding a
// single instance, and a shared store (pkg/ratelimit/store redis) can replace it
// without changing callers.
type ClientIPRateLimit struct {
	inner             *ratelimitmw.HTTPMiddleware
	trustForwardedFor bool
}

// NewClientIPRateLimit allows max requests per client IP per window. With
// trustForwardedFor the IP is the last X-Forwarded-For entry (the one the
// trusted proxy appended); otherwise it is the TCP peer.
func NewClientIPRateLimit(max int64, window time.Duration, trustForwardedFor bool) (*ClientIPRateLimit, error) {
	inner, err := ratelimitmw.NewHTTPMiddleware(ratelimit.Config{
		Rate:      max,
		Burst:     max,
		Window:    window,
		Algorithm: ratelimit.SlidingWindow,
		KeyFunc: func(ctx context.Context) (string, error) {
			ip, _ := ctx.Value(clientIPCtxKey{}).(string)
			return ip, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return &ClientIPRateLimit{inner: inner, trustForwardedFor: trustForwardedFor}, nil
}

// WrapFunc guards next. A request over the limit gets 429 RATE_LIMITED (the
// shared envelope, written by pkg/ratelimit) and next never runs.
func (l *ClientIPRateLimit) WrapFunc(next http.HandlerFunc) http.HandlerFunc {
	limited := l.inner.Handler(next)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), clientIPCtxKey{}, rateLimitIP(r, l.trustForwardedFor))
		limited.ServeHTTP(w, r.WithContext(ctx))
	}
}

// rateLimitIP resolves the caller's address for rate limiting. Behind a trusted
// proxy the last X-Forwarded-For entry is the address that proxy saw; earlier
// entries are client-supplied and ignored, so a caller cannot rotate a fake
// value to escape the limit.
func rateLimitIP(r *http.Request, trustForwardedFor bool) string {
	if trustForwardedFor {
		if parts := strings.Split(r.Header.Get("X-Forwarded-For"), ","); len(parts) > 0 {
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
