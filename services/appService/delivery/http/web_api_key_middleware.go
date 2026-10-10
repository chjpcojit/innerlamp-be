package http

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
)

// HeaderAPIKey carries the shared key the web frontend's server sends on every
// call. It identifies the calling application (the website), not a user: users
// are identified by their own access token.
const HeaderAPIKey = "X-API-Key"

type webKeyCtxKey struct{}

// withWebKeyVerified marks ctx as coming from a caller that presented the
// correct web API key.
func withWebKeyVerified(ctx context.Context) context.Context {
	return context.WithValue(ctx, webKeyCtxKey{}, true)
}

// webKeyVerified reports whether RequireWebAPIKey saw a valid key on this request.
func webKeyVerified(ctx context.Context) bool {
	v, _ := ctx.Value(webKeyCtxKey{}).(bool)
	return v
}

// RequireWebAPIKey validates the web frontend's API key.
//
//   - header absent  → passes through unmarked: the caller is treated as mobile,
//     which cannot keep a shared secret (it would ship in the binary).
//   - header correct → passes through marked, so handlers can require it for
//     platform "web" (see LoginHandler.Login).
//   - header present but wrong or empty → 401 INVALID_API_KEY.
//
// Known gaps: docs/tech-debt/auth/TD-001.
//
// An unconfigured key never matches, so a presented key is always rejected
// rather than accepted by default.
//
// This is a shared application secret, not user authentication, and it does not
// stop a caller who simply omits the header: it only authenticates the callers
// that claim to be the website.
type RequireWebAPIKey struct {
	keyHash [sha256.Size]byte
	enabled bool
}

// NewRequireWebAPIKey wires the middleware. An empty key leaves it fail-closed.
func NewRequireWebAPIKey(key string) *RequireWebAPIKey {
	return &RequireWebAPIKey{keyHash: sha256.Sum256([]byte(key)), enabled: key != ""}
}

// Wrap returns a handler that validates the key before delegating to next.
func (m *RequireWebAPIKey) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, present := r.Header[http.CanonicalHeaderKey(HeaderAPIKey)]
		if !present {
			next.ServeHTTP(w, r)
			return
		}

		// Hashing first makes the comparison constant-time regardless of the
		// presented key's length.
		got := sha256.Sum256([]byte(values[0]))
		if !m.enabled || subtle.ConstantTimeCompare(got[:], m.keyHash[:]) != 1 {
			writeInvalidAPIKey(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(withWebKeyVerified(r.Context())))
	})
}

// WrapFunc is Wrap for a HandlerFunc, matching how cmd/main.go registers routes.
func (m *RequireWebAPIKey) WrapFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Wrap(next).ServeHTTP
}

func writeInvalidAPIKey(w http.ResponseWriter) {
	utils.WriteBusinessError(w, http.StatusUnauthorized, "INVALID_API_KEY", "invalid api key", nil)
}
