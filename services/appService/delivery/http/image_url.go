package http

import (
	"net/http"
	"strings"
)

// imageHost turns the image locations stored in the database into the absolute URLs
// the API returns. The database keeps a root-relative path ("/static/images/..."), so
// moving to another network, host or CDN never touches stored data; an already
// absolute URL (http:// or https://) is returned untouched.
type imageHost struct {
	// base is the origin the relative paths are appended to, e.g. "https://cdn.example.com";
	// empty = the request's own origin (dev only, see config.ServerConfig.PublicBaseURL).
	base string
}

// ImageOption configures how a handler builds image URLs.
type ImageOption func(*imageHost)

// WithImageBase sets the origin relative image paths are resolved against.
func WithImageBase(base string) ImageOption {
	return func(h *imageHost) { h.base = strings.TrimRight(base, "/") }
}

func newImageHost(opts []ImageOption) imageHost {
	var h imageHost
	for _, o := range opts {
		o(&h)
	}
	return h
}

// url resolves one stored value; "" stays "".
func (h imageHost) url(r *http.Request, raw string) string {
	if !strings.HasPrefix(raw, "/") {
		return raw // "", an absolute URL, or anything that is not a root-relative path
	}
	return h.origin(r) + raw
}

// ptr is url for the nullable fields: "" becomes JSON null.
func (h imageHost) ptr(r *http.Request, raw string) *string {
	if raw == "" {
		return nil
	}
	u := h.url(r, raw)
	return &u
}

func (h imageHost) origin(r *http.Request) string {
	if h.base != "" {
		return h.base
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
