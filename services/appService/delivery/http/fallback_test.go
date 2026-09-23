package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func newFallbackTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	return mux
}

func TestWithAPIFallback_UnknownPathIs404Envelope(t *testing.T) {
	h := WithAPIFallback(newFallbackTestMux())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/nope", nil))

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	got := decodeErrorBody(t, rec)
	require.Equal(t, "NOT_FOUND", got.Code)
}

func TestWithAPIFallback_WrongMethodIs405EnvelopeWithAllow(t *testing.T) {
	h := WithAPIFallback(newFallbackTestMux())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/auth/login", nil))

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Allow"), http.MethodPost)
	got := decodeErrorBody(t, rec)
	require.Equal(t, "METHOD_NOT_ALLOWED", got.Code)
}

func TestWithAPIFallback_MatchedRouteUntouched(t *testing.T) {
	h := WithAPIFallback(newFallbackTestMux())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil))

	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestWithAPIFallback_PathCleanRedirectUntouched(t *testing.T) {
	h := WithAPIFallback(newFallbackTestMux())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/auth/../auth/login", nil))

	require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	require.Equal(t, "/v1/auth/login", rec.Header().Get("Location"))
}
