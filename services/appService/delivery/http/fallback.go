package http

import (
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
)

// WithAPIFallback wraps mux so the two replies ServeMux writes on its own —
// 404 for a path no route matches, 405 for a known path with the wrong method —
// use the utils.ApiResponse envelope every handler replies with, instead of
// ServeMux's plain-text "404 page not found" / "Method Not Allowed".
//
// Only requests no route matched (mux.Handler returns an empty pattern) are
// intercepted, so a real handler's response is never touched. ServeMux's own
// fallback still runs, so the 404/405 decision and the Allow header stay
// exactly ServeMux's; only the body is replaced. A path-cleaning redirect also
// has an empty pattern and passes through unchanged.
func WithAPIFallback(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}

		fw := &fallbackWriter{ResponseWriter: w}
		mux.ServeHTTP(fw, r)

		switch fw.intercepted {
		case http.StatusNotFound:
			utils.WriteBusinessError(w, http.StatusNotFound, "NOT_FOUND", "route not found", nil)
		case http.StatusMethodNotAllowed:
			utils.WriteBusinessError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		}
	})
}

// fallbackWriter swallows a 404/405 status and its plain-text body so
// WithAPIFallback can write the envelope instead; any other status (a
// redirect) is forwarded as is. Headers ServeMux set (Allow) stay on the
// underlying writer and go out with the envelope.
type fallbackWriter struct {
	http.ResponseWriter
	intercepted int
}

func (fw *fallbackWriter) WriteHeader(code int) {
	if code == http.StatusNotFound || code == http.StatusMethodNotAllowed {
		fw.intercepted = code
		return
	}
	fw.ResponseWriter.WriteHeader(code)
}

func (fw *fallbackWriter) Write(b []byte) (int, error) {
	if fw.intercepted != 0 {
		return len(b), nil
	}
	return fw.ResponseWriter.Write(b)
}
