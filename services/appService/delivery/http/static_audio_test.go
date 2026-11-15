package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticFiles_ServesMP3WithRange(t *testing.T) {
	dir := t.TempDir()
	body := strings.Repeat("ID3", 400)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.mp3"), []byte(body), 0o600))

	h, err := NewStaticFiles(dir)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/a.mp3", nil)
	req.Header.Set("Range", "bytes=0-99")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusPartialContent, rec.Code)
	require.Equal(t, "audio/mpeg", rec.Header().Get("Content-Type"))
	require.Equal(t, "bytes", rec.Header().Get("Accept-Ranges"))
	require.Equal(t, "cross-origin", rec.Header().Get("Cross-Origin-Resource-Policy"))
	require.Len(t, rec.Body.Bytes(), 100)
}

func TestStaticFiles_HeadProbesMP3(t *testing.T) {
	dir := t.TempDir()
	body := strings.Repeat("ID3", 400)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.mp3"), []byte(body), 0o600))

	h, err := NewStaticFiles(dir)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodHead, "/a.mp3", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "audio/mpeg", rec.Header().Get("Content-Type"))
	require.Equal(t, "bytes", rec.Header().Get("Accept-Ranges"))
	require.Equal(t, "1200", rec.Header().Get("Content-Length"))
	require.Empty(t, rec.Body.Bytes(), "HEAD must not send a body")
}

func TestStaticFiles_UnsatisfiableRangeIs416(t *testing.T) {
	dir := t.TempDir()
	body := strings.Repeat("ID3", 400)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.mp3"), []byte(body), 0o600))

	h, err := NewStaticFiles(dir)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/a.mp3", nil)
	req.Header.Set("Range", "bytes=9999-10000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestedRangeNotSatisfiable, rec.Code)
	require.Equal(t, "bytes */1200", rec.Header().Get("Content-Range"))
}
