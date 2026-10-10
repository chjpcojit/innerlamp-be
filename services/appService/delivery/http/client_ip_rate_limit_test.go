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
