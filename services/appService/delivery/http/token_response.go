package http

import (
	"net/http"
	"time"
)

// HandlerOption configures the handlers that issue tokens (RegisterHandler's
// VerifyOTP, LoginHandler's Login and Refresh).
type HandlerOption func(*tokenSettings)

type tokenSettings struct {
	accessTokenTTL time.Duration

	// trustForwardedFor is auth_rate_limit.trust_forwarded_for: LoginHandler
	// resolves the device's IP with the same rule the rate limiter uses
	// (login AC34).
	trustForwardedFor bool
}

// WithTrustForwardedFor makes the handler resolve client IPs from the last
// X-Forwarded-For entry when no web API key vouches for X-InnerLamp-Client-IP.
func WithTrustForwardedFor(trust bool) HandlerOption {
	return func(s *tokenSettings) { s.trustForwardedFor = trust }
}

// WithAccessTokenTTL sets the access token lifetime reported as `expires_in`.
// It must be the same value the JWT service mints tokens with
// (config.JWTConfig.AccessTokenExpiration), passed in from cmd/main.go rather
// than read back out of a token.
func WithAccessTokenTTL(d time.Duration) HandlerOption {
	return func(s *tokenSettings) { s.accessTokenTTL = d }
}

func newTokenSettings(opts []HandlerOption) tokenSettings {
	var s tokenSettings
	for _, o := range opts {
		o(&s)
	}
	return s
}

// expiresIn is the `expires_in` value of a token response: whole seconds of the
// access token's lifetime (OAuth 2.0 meaning; the refresh token is not included).
func (s tokenSettings) expiresIn() int64 {
	return int64(s.accessTokenTTL / time.Second)
}

// NoStore marks every response of the routes it wraps as uncacheable. It is set
// before the handler runs, so error responses carry the headers too: a token
// response must never be stored by a browser, proxy or CDN (AC20).
type NoStore struct{}

// WrapFunc returns a handler that sets the headers and then delegates to next.
func (NoStore) WrapFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next(w, r)
	}
}
