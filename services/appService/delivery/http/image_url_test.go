package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageHost_URL(t *testing.T) {
	req := func(host string, mut func(*http.Request)) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
		r.Host = host
		if mut != nil {
			mut(r)
		}
		return r
	}
	https := func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") }

	tests := []struct {
		name string
		base string
		r    *http.Request
		raw  string
		want string
	}{
		{"empty stays empty", "", req("a:8080", nil), "", ""},
		{"relative uses request host", "", req("192.168.1.9:8080", nil), "/static/images/a.jpg", "http://192.168.1.9:8080/static/images/a.jpg"},
		{"relative follows forwarded https", "", req("api.x.vn", https), "/static/images/a.jpg", "https://api.x.vn/static/images/a.jpg"},
		{"relative follows forwarded host", "", req("internal:8082", func(r *http.Request) {
			r.Header.Set("X-Forwarded-Host", "api.x.vn")
			r.Header.Set("X-Forwarded-Proto", "https")
		}), "/static/images/a.jpg", "https://api.x.vn/static/images/a.jpg"},
		{"forwarded list takes the first hop", "", req("internal:8082", func(r *http.Request) {
			r.Header.Set("X-Forwarded-Host", "api.x.vn, proxy.internal")
			r.Header.Set("X-Forwarded-Proto", "https, http")
		}), "/static/images/a.jpg", "https://api.x.vn/static/images/a.jpg"},
		{"configured base wins", "https://cdn.x.vn", req("a:8080", nil), "/static/images/a.jpg", "https://cdn.x.vn/static/images/a.jpg"},
		{"configured base trailing slash", "https://cdn.x.vn/", req("a:8080", nil), "/static/images/a.jpg", "https://cdn.x.vn/static/images/a.jpg"},
		{"absolute http passes through", "https://cdn.x.vn", req("a", nil), "http://old/x.jpg", "http://old/x.jpg"},
		{"bare name untouched", "", req("a", nil), "q.jpg", "q.jpg"},
		{"absolute https passes through", "", req("a", nil), "https://img/x.jpg", "https://img/x.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := imageHost{}
			WithImageBase(tt.base)(&h)
			assert.Equal(t, tt.want, h.url(tt.r, tt.raw))
		})
	}
}

func TestImageHost_Ptr(t *testing.T) {
	h := imageHost{}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Host = "h"
	assert.Nil(t, h.ptr(r, ""))
	got := h.ptr(r, "/static/images/a.jpg")
	if assert.NotNil(t, got) {
		assert.Equal(t, "http://h/static/images/a.jpg", *got)
	}
}

func TestImageHost_AudioURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/dharma-talks/x", nil)
	req.Host = "192.168.1.9:8082"

	tests := []struct {
		name      string
		imageBase string
		audioBase string
		raw       string
		want      string
	}{
		{"audio base wins", "https://cdn.x.vn", "https://audio.x.vn", "/static/audio/a.mp3", "https://audio.x.vn/static/audio/a.mp3"},
		{"audio base trailing slash", "", "https://audio.x.vn/", "/static/audio/a.mp3", "https://audio.x.vn/static/audio/a.mp3"},
		{"falls back to image base", "https://cdn.x.vn", "", "/static/audio/a.mp3", "https://cdn.x.vn/static/audio/a.mp3"},
		{"falls back to request host", "", "", "/static/audio/a.mp3", "http://192.168.1.9:8082/static/audio/a.mp3"},
		{"absolute passes through", "https://cdn.x.vn", "https://audio.x.vn", "https://old/a.mp3", "https://old/a.mp3"},
		{"empty stays empty", "", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := imageHost{}
			WithImageBase(tt.imageBase)(&h)
			WithAudioBase(tt.audioBase)(&h)
			assert.Equal(t, tt.want, h.audioURL(req, tt.raw))
		})
	}

	h := imageHost{}
	assert.Nil(t, h.audioPtr(req, ""))
	got := h.audioPtr(req, "/static/audio/a.mp3")
	if assert.NotNil(t, got) {
		assert.Equal(t, "http://192.168.1.9:8082/static/audio/a.mp3", *got)
	}
}
