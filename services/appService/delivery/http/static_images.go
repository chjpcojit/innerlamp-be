package http

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// NewStaticFiles serves the files under dir (assets/images in dev) so image_url can
// point at this service while there is no object store / CDN yet. It lists no
// directories and refuses any path segment that starts with "." (.gitignore, ...).
// Mount it behind http.StripPrefix so a request path maps onto dir.
// DEV-ONLY by intent: enabled only when server.static_images_dir is set in config.
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
		w.Header().Set("Cache-Control", "public, max-age=86400")
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
