package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runWebKey(t *testing.T, configured, sent string, setHeader bool) (code int, reached, verified bool, body string) {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		verified = webKeyVerified(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	if setHeader {
		req.Header.Set(HeaderAPIKey, sent)
	}
	rec := httptest.NewRecorder()
	NewRequireWebAPIKey(configured).Wrap(next).ServeHTTP(rec, req)
	return rec.Code, reached, verified, rec.Body.String()
}

func TestRequireWebAPIKey(t *testing.T) {
	t.Run("matching key passes and is marked verified", func(t *testing.T) {
		code, reached, verified, _ := runWebKey(t, "web-key", "web-key", true)
		assert.Equal(t, http.StatusOK, code)
		assert.True(t, reached)
		assert.True(t, verified)
	})

	t.Run("no key passes as mobile, not verified", func(t *testing.T) {
		code, reached, verified, _ := runWebKey(t, "web-key", "", false)
		assert.Equal(t, http.StatusOK, code)
		assert.True(t, reached)
		assert.False(t, verified)
	})

	t.Run("wrong key is 401 INVALID_API_KEY and never reaches next", func(t *testing.T) {
		code, reached, _, body := runWebKey(t, "web-key", "nope", true)
		require.Equal(t, http.StatusUnauthorized, code)
		assert.False(t, reached)
		assert.Contains(t, body, `"code":"INVALID_API_KEY"`)
	})

	t.Run("empty header value is wrong, not absent", func(t *testing.T) {
		code, reached, _, _ := runWebKey(t, "web-key", "", true)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.False(t, reached)
	})

	t.Run("unconfigured key fails closed for any presented key", func(t *testing.T) {
		code, reached, _, _ := runWebKey(t, "", "", true)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.False(t, reached)
		code, reached, _, _ = runWebKey(t, "", "anything", true)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.False(t, reached)
	})
}
