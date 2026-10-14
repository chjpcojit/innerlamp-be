package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/ratelimit"
	ratelimitmw "chjpcoj.com.innerlamp/pkg/ratelimit/middleware"
)

type userRateLimitKey struct{}

// UserRateLimit limits a group of routes per authenticated account. It is the
// per-user sibling of ClientIPRateLimit and must sit INSIDE the auth middleware:
// the account is read from the access token's claims, so an unauthenticated
// request has no bucket to be counted in.
//
// The key is the account, not the address: several people behind one NAT do not
// starve each other, and one account moving between networks does not get a
// fresh budget. The counters are in memory, so the limit is per service
// instance, the same floor ClientIPRateLimit documents.
type UserRateLimit struct {
	inner *ratelimitmw.HTTPMiddleware
}

// NewUserRateLimit allows max requests per account per window (sliding window,
// no burst beyond max). Options as for NewClientIPRateLimit.
func NewUserRateLimit(max int64, window time.Duration, opts ...RateLimitOption) (*UserRateLimit, error) {
	if max < 1 {
		return nil, errors.New("user rate limit: max must be >= 1")
	}
	if window <= 0 {
		return nil, errors.New("user rate limit: window must be > 0")
	}
	o := applyRateLimitOptions(opts)
	inner, err := ratelimitmw.NewHTTPMiddleware(ratelimit.Config{
		Rate:      max,
		Burst:     max,
		Window:    window,
		Algorithm: ratelimit.SlidingWindow,
		Store:     o.store,
		KeyFunc: func(ctx context.Context) (string, error) {
			id, _ := ctx.Value(userRateLimitKey{}).(string)
			return o.key(id), nil
		},
	})
	if err != nil {
		return nil, err
	}
	return &UserRateLimit{inner: inner}, nil
}

// WrapFunc guards next. A request over the limit gets 429 RATE_LIMITED (the
// shared envelope, written by pkg/ratelimit) and next never runs. A request
// with no account in its context is answered 401 and not counted: collapsing
// those into one empty-keyed bucket would let anonymous traffic 429 real users.
func (l *UserRateLimit) WrapFunc(next http.HandlerFunc) http.HandlerFunc {
	limited := l.inner.Handler(next)
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := jwtcontext.GetClaims(r.Context())
		if !ok || claims.UserID == "" {
			WriteAuthError(w, r, errors.New("no account in context"))
			return
		}
		ctx := context.WithValue(r.Context(), userRateLimitKey{}, claims.UserID)
		limited.ServeHTTP(w, r.WithContext(ctx))
	}
}
