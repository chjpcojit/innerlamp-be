package http

import (
	"net/http"
	"strings"
)

// imageHost turns the media locations stored in the database into the absolute URLs
// the API returns. The database keeps a root-relative path ("/static/images/...",
// "/static/audio/..."), so moving to another network, host or CDN never touches
// stored data; an already absolute URL (http:// or https://) is returned untouched.
type imageHost struct {
	// base is the origin relative image paths are appended to, e.g.
	// "https://cdn.example.com"; empty = the request's own origin (dev only, see
	// config.ServerConfig.PublicBaseURL).
	base string
	// audioBase overrides base for audio paths only, so audio can live on its own
	// object-storage or CDN origin (config.ServerConfig.AudioBaseURL).
	audioBase string
}

// ImageOption configures how a handler builds image URLs.
type ImageOption func(*imageHost)

// WithImageBase sets the origin relative image paths are resolved against.
func WithImageBase(base string) ImageOption {
	return func(h *imageHost) { h.base = strings.TrimRight(base, "/") }
}

// WithAudioBase sets a separate origin for audio paths. Empty falls back to
// WithImageBase, then the request origin.
func WithAudioBase(base string) ImageOption {
	return func(h *imageHost) { h.audioBase = strings.TrimRight(base, "/") }
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

// audioURL is url for a playable audio file, preferring the audio origin.
func (h imageHost) audioURL(r *http.Request, raw string) string {
	if !strings.HasPrefix(raw, "/") {
		return raw
	}
	if h.audioBase != "" {
		return h.audioBase + raw
	}
	return h.url(r, raw)
}

// audioPtr is audioURL for the nullable audio field: "" becomes JSON null.
func (h imageHost) audioPtr(r *http.Request, raw string) *string {
	if raw == "" {
		return nil
	}
	u := h.audioURL(r, raw)
	return &u
}

func (h imageHost) origin(r *http.Request) string {
	if h.base != "" {
		return h.base
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(firstForwarded(r.Header.Get("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	host := r.Host
	if forwarded := firstForwarded(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		host = forwarded
	}
	return scheme + "://" + host
}

// firstForwarded takes the first value of a comma-separated proxy header and
// trims it. Proxies append their own hop, so the first value is the client-facing
// one; an empty or malformed header leaves the request's own value in place.
func firstForwarded(raw string) string {
	first, _, _ := strings.Cut(raw, ",")
	return strings.TrimSpace(first)
}
