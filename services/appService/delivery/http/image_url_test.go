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
