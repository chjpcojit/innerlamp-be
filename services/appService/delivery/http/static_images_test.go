package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func newStaticImagesTestServer(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "temples", "full"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "temples", "full", "a.jpg"), []byte("jpg-bytes"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("dotfile-content-xyz"), 0o600))

	h, err := NewStaticImages(dir)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle("GET /static/images/", http.StripPrefix("/static/images", h))
	return mux
}

func TestStaticImages_ServesFileWithCacheHeader(t *testing.T) {
	h := newStaticImagesTestServer(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/images/temples/full/a.jpg", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "jpg-bytes", rec.Body.String())
	require.Contains(t, rec.Header().Get("Cache-Control"), "max-age")
}

func TestStaticImages_NotServed(t *testing.T) {
	h := newStaticImagesTestServer(t)

	for name, path := range map[string]string{
		"directory listing":  "/static/images/temples/",
		"directory no slash": "/static/images/temples/full",
		"dotfile":            "/static/images/.gitignore",
		"missing file":       "/static/images/temples/full/nope.jpg",
		"traversal":          "/static/images/../secret",
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			require.NotEqual(t, http.StatusOK, rec.Code)
			require.NotContains(t, rec.Body.String(), "a.jpg")
			require.NotContains(t, rec.Body.String(), "dotfile-content-xyz")
		})
	}
}

func TestNewStaticImages_RejectsMissingDir(t *testing.T) {
	_, err := NewStaticImages(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}
