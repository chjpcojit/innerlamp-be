package http

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// NewStaticFiles serves media files under dir (assets/images and assets/audio in
// dev) so image_url and audio_url can point at this service while there is no
// object store / CDN yet. It lists no directories and refuses any path segment
// that starts with "." (.gitignore, ...). Mount it behind http.StripPrefix so a
// request path maps onto dir.
//
// Range and HEAD are handled by http.FileServer, so a player can seek and an
// iOS/Android player can probe the file before playing. The extra headers make
// the same files usable from a web page on another origin. DEV-ONLY by intent:
// enabled only when server.static_images_dir / static_audio_dir is set.
func NewStaticFiles(dir string) (http.Handler, error) {
	st, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("static files dir: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("static files dir %q is not a directory", dir)
	}

	files := http.FileServer(noListingFS{http.Dir(dir)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, seg := range strings.Split(r.URL.Path, "/") {
			if strings.HasPrefix(seg, ".") {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(w, r)
	}), nil
}

// NewStaticImages is NewStaticFiles, kept for the image route.
func NewStaticImages(dir string) (http.Handler, error) { return NewStaticFiles(dir) }

// noListingFS answers a directory as not-found so http.FileServer never renders an
// index of the folder.
type noListingFS struct{ fs http.FileSystem }

func (n noListingFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if st.IsDir() {
		_ = f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}
