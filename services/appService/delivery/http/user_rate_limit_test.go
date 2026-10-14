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

func TestUserRateLimit_BlocksTheAccountOverTheLimit(t *testing.T) {
	var calls int
	rl, err := NewUserRateLimit(3, time.Hour)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h(w, memorialReq(http.MethodGet, "/v1/users/me", "", "7"))
		assert.Equal(t, http.StatusNoContent, w.Code, "request %d is within the limit", i+1)
	}

	w := httptest.NewRecorder()
	h(w, memorialReq(http.MethodGet, "/v1/users/me", "", "7"))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.Equal(t, "RATE_LIMITED", body.Error.Code)
	assert.Equal(t, 3, calls, "the handler must not run for a limited request")
}

func TestUserRateLimit_CountsPerAccountNotPerIP(t *testing.T) {
	var calls int
	rl, err := NewUserRateLimit(1, time.Hour)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	// Two accounts behind one NAT address: each has its own budget.
	reqFor := func(userID string) *http.Request {
		r := memorialReq(http.MethodGet, "/v1/users/me", "", userID)
		r.RemoteAddr = "10.0.0.1:40000"
		return r
	}
	w := httptest.NewRecorder()
	h(w, reqFor("7"))
	assert.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	h(w, reqFor("7"))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	w = httptest.NewRecorder()
	h(w, reqFor("8"))
	assert.Equal(t, http.StatusNoContent, w.Code, "another account is not limited by 7's traffic")
}

func TestUserRateLimit_SameAccountFromAnotherAddressSharesTheBudget(t *testing.T) {
	var calls int
	rl, err := NewUserRateLimit(1, time.Hour)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	first := memorialReq(http.MethodGet, "/v1/users/me", "", "7")
	first.RemoteAddr = "10.0.0.1:1"
	second := memorialReq(http.MethodGet, "/v1/users/me", "", "7")
	second.RemoteAddr = "10.0.0.2:1"

	w := httptest.NewRecorder()
	h(w, first)
	require.Equal(t, http.StatusNoContent, w.Code)
	w = httptest.NewRecorder()
	h(w, second)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestUserRateLimit_RequestWithoutAccountIsNotCounted(t *testing.T) {
	var calls int
	rl, err := NewUserRateLimit(1, time.Hour)
	require.NoError(t, err)
	h := rl.WrapFunc(okHandler(&calls))

	// Anonymous requests must neither run the handler nor eat anyone's budget,
	// and must not pile into one shared "" bucket that would 429 real users.
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h(w, memorialReq(http.MethodGet, "/v1/users/me", "", ""))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}
	assert.Zero(t, calls)

	w := httptest.NewRecorder()
	h(w, memorialReq(http.MethodGet, "/v1/users/me", "", "7"))
	assert.Equal(t, http.StatusNoContent, w.Code, "the first real request still has its whole budget")
}

func TestNewUserRateLimit_RejectsNonsenseLimits(t *testing.T) {
	_, err := NewUserRateLimit(0, time.Minute)
	assert.Error(t, err)
	_, err = NewUserRateLimit(5, 0)
	assert.Error(t, err)
}
