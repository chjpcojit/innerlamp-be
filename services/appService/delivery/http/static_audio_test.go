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
	require.Len(t, rec.Body.Bytes(), 100)
}
