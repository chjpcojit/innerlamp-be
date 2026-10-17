package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okHandler(calls *int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		*calls++
		w.WriteHeader(http.StatusNoContent)
	}
}

func fromIP(ip, xff string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/temple-suggestions", nil)
	r.RemoteAddr = ip + ":40000"
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func TestClientIPRateLimit_BlocksOverLimitPerIP(t *testing.T) {
	var calls int
	rl, err := NewClientIPRateLimit(2, time.Hour, false)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h(w, fromIP("10.0.0.1", ""))
		assert.Equal(t, http.StatusNoContent, w.Code, "request %d is within the limit", i+1)
	}

	w := httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", ""))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.Equal(t, "RATE_LIMITED", body.Error.Code)
	assert.Equal(t, 2, calls, "the handler must not run for a limited request")

	// A different address has its own bucket.
	w = httptest.NewRecorder()
	h(w, fromIP("10.0.0.2", ""))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestClientIPRateLimit_IgnoresForwardedForUnlessTrusted(t *testing.T) {
	var calls int
	rl, err := NewClientIPRateLimit(1, time.Hour, false)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	// Rotating a fake X-Forwarded-For must not escape the limit.
	w := httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", "1.1.1.1"))
	assert.Equal(t, http.StatusNoContent, w.Code)
	w = httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", "2.2.2.2"))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestClientIPRateLimit_TrustedProxyUsesLastForwardedEntry(t *testing.T) {
	var calls int
	rl, err := NewClientIPRateLimit(1, time.Hour, true)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	// Same proxy peer, different real clients: separate buckets.
	w := httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", "9.9.9.9, 203.0.113.5"))
	assert.Equal(t, http.StatusNoContent, w.Code)
	w = httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", "9.9.9.9, 203.0.113.6"))
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Same last entry again: limited, whatever the spoofed leading values are.
	w = httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", "7.7.7.7, 203.0.113.5"))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestClientIPRateLimit_TrustedProxyFallsBackToPeerWithoutHeader(t *testing.T) {
	var calls int
	rl, err := NewClientIPRateLimit(1, time.Hour, true)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	w := httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", ""))
	assert.Equal(t, http.StatusNoContent, w.Code)
	w = httptest.NewRecorder()
	h(w, fromIP("10.0.0.1", ""))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestNewClientIPRateLimit_RejectsInvalidLimit(t *testing.T) {
	_, err := NewClientIPRateLimit(0, time.Hour, false)
	assert.Error(t, err)
	_, err = NewClientIPRateLimit(5, 0, false)
	assert.Error(t, err)
}

// TestRateLimitIP_Resolution covers login AC34's order: (a) the forwarded
// header only with a valid key and a parsable single IP, (b) last XFF entry when
// trusted, (c) the connection.
func TestRateLimitIP_Resolution(t *testing.T) {
	build := func(keyOK bool, clientIPHeader, xff string) *http.Request {
		r := fromIP("10.0.0.9", xff)
		if clientIPHeader != "" {
			r.Header.Set(HeaderClientIP, clientIPHeader)
		}
		if keyOK {
			r = r.WithContext(withWebKeyVerified(r.Context()))
		}
		return r
	}
	cases := []struct {
		name  string
		r     *http.Request
		trust bool
		want  string
	}{
		{"valid key + header wins over everything", build(true, "203.0.113.7", "198.51.100.1, 198.51.100.2"), true, "203.0.113.7"},
		{"IPv6 header is canonicalised", build(true, "2001:DB8::1", ""), false, "2001:db8::1"},
		{"no key: header ignored, falls to peer", build(false, "203.0.113.7", ""), false, "10.0.0.9"},
		{"no key: header ignored, falls to XFF when trusted", build(false, "203.0.113.7", "198.51.100.1, 198.51.100.2"), true, "198.51.100.2"},
		{"garbage header with key falls to XFF when trusted", build(true, "not-an-ip", "198.51.100.1, 198.51.100.2"), true, "198.51.100.2"},
		{"garbage header with key falls to peer otherwise", build(true, "not-an-ip", "198.51.100.1"), false, "10.0.0.9"},
		{"two addresses in the header are not one address", build(true, "203.0.113.7, 203.0.113.8", ""), false, "10.0.0.9"},
		{"XFF ignored when not trusted", build(false, "", "198.51.100.1"), false, "10.0.0.9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, rateLimitIP(tc.r, tc.trust))
		})
	}
}

// TestClientIPRateLimit_ForwardedHeaderGivesVisitorsOwnBuckets: behind the
// website's server every request shares one TCP peer; with a valid key the
// visitors' forwarded addresses must be counted separately (AC34 (a)).
func TestClientIPRateLimit_ForwardedHeaderGivesVisitorsOwnBuckets(t *testing.T) {
	var calls int
	rl, err := NewClientIPRateLimit(1, time.Hour, false)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	send := func(visitor string) int {
		r := fromIP("10.0.0.9", "")
		r.Header.Set(HeaderClientIP, visitor)
		r = r.WithContext(withWebKeyVerified(r.Context()))
		w := httptest.NewRecorder()
		h(w, r)
		return w.Code
	}

	assert.Equal(t, http.StatusNoContent, send("203.0.113.1"))
	assert.Equal(t, http.StatusNoContent, send("203.0.113.2"), "another visitor, another bucket")
	assert.Equal(t, http.StatusTooManyRequests, send("203.0.113.1"))
}

func TestClientIPRateLimit_WebKeySkipsTheLimit(t *testing.T) {
	var calls int
	rl, err := NewClientIPRateLimit(1, time.Hour, false)
	require.NoError(t, err)
	h := rl.WrapFuncUnlessWebKey(okHandler(&calls))

	// Without the key the second request from one address is limited.
	for i, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		w := httptest.NewRecorder()
		h(w, fromIP("10.0.0.1", ""))
		assert.Equal(t, want, w.Code, "request %d without a key", i+1)
	}

	// With the verified key the same address is never counted.
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r := fromIP("10.0.0.1", "")
		r = r.WithContext(withWebKeyVerified(r.Context()))
		h(w, r)
		assert.Equal(t, http.StatusNoContent, w.Code, "request %d with the key", i+1)
	}
}
