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
// trusted proxy appended); otherwise it is the TCP peer. Without options the
// counters are this limiter's own, in memory; WithSharedStore moves them to a
// store several limiters (and instances) share.
func NewClientIPRateLimit(max int64, window time.Duration, trustForwardedFor bool, opts ...RateLimitOption) (*ClientIPRateLimit, error) {
	o := applyRateLimitOptions(opts)
	inner, err := ratelimitmw.NewHTTPMiddleware(ratelimit.Config{
		Rate:      max,
		Burst:     max,
		Window:    window,
		Algorithm: ratelimit.SlidingWindow,
		Store:     o.store,
		KeyFunc: func(ctx context.Context) (string, error) {
			ip, _ := ctx.Value(clientIPCtxKey{}).(string)
			return o.key(ip), nil
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

// HeaderClientIP is the header the website's server uses to forward its
// visitor's address (login AC34). It is read only on a request that carries a
// valid X-API-Key; elsewhere it is attacker-controlled text.
const HeaderClientIP = "X-InnerLamp-Client-IP"

// rateLimitIP resolves the caller's address for rate limiting and device
// metadata (login AC34), in order:
//
//	(a) X-InnerLamp-Client-IP, when RequireWebAPIKey saw a valid key on this
//	    request and the header parses as exactly one IP address;
//	(b) when trustForwardedFor, the last X-Forwarded-For entry — the address
//	    the trusted proxy saw; earlier entries are client-supplied and ignored,
//	    so a caller cannot rotate a fake value to escape the limit;
//	(c) the TCP peer.
//
// An unparsable (a) falls through to (b)/(c); without a valid key the header
// is never read.
func rateLimitIP(r *http.Request, trustForwardedFor bool) string {
	if webKeyVerified(r.Context()) {
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get(HeaderClientIP))); ip != nil {
			return ip.String()
		}
	}
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
