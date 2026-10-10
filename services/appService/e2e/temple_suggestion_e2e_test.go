package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTempleSuggestion_E2E drives the public "suggest a temple" route
// (docs/specs/features/temple-suggestion/, AC1-AC8) against the real binary and
// a fresh Postgres.
//
// What is here and not in the per-layer tests: that cmd/main.go registers the
// route, puts the per-IP limit in front of it and the config's 5/hour reaches the
// limiter, and that no read route for suggestions exists.
func TestTempleSuggestion_E2E(t *testing.T) {
	baseURL, client := newE2EServer(t)
	api := &apiClient{t: t, baseURL: baseURL, http: client}
	valid := map[string]any{"name": "Chùa Mới", "province": "Quảng Ninh", "address": "Thôn 1, xã A"}

	// A wrong client key is refused before it can spend any of the rate budget.
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/temple-suggestions", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", "not-the-key")
	resp, err := client.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// AC8: the app has no way to read suggestions back.
	for _, m := range []string{http.MethodGet, http.MethodDelete} {
		res := api.do(m, "/v1/temple-suggestions", nil, "")
		assert.GreaterOrEqual(t, res.status, 400, "%s must not be served: %s", m, res.raw)
		assert.False(t, res.envelope.Success)
	}
	res := api.do(http.MethodGet, "/v1/temple-suggestions/1", nil, "")
	assert.GreaterOrEqual(t, res.status, 400)

	// AC1: anonymous submit (no Authorization) is accepted as pending.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", valid, "")
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	var created struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	res.data(&created)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "pending", created.Status)

	// AC6: the same suggestion again is accepted, not de-duplicated.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", valid, "")
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	var again struct {
		ID int64 `json:"id"`
	}
	res.data(&again)
	assert.NotEqual(t, created.ID, again.ID)

	// AC2: a blank value names the field; AC4: an unknown field is refused.
	res = api.do(http.MethodPost, "/v1/temple-suggestions",
		map[string]any{"name": "Chùa", "province": "  ", "address": "x"}, "")
	assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
	assert.Equal(t, "VALIDATION_ERROR", res.errorCode())
	assert.Equal(t, "province", res.details()["field"])

	res = api.do(http.MethodPost, "/v1/temple-suggestions",
		map[string]any{"name": "Chùa", "province": "p", "address": "a", "status": "created"}, "")
	assert.Equal(t, http.StatusBadRequest, res.status, res.raw)

	// AC7: 5 per IP per hour (dev config). Four requests above counted (the wrong-key
	// call did not), the fifth is the last allowed, the sixth is limited.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", valid, "")
	assert.Equal(t, http.StatusCreated, res.status, res.raw)
	res = api.do(http.MethodPost, "/v1/temple-suggestions", valid, "")
	assert.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
	assert.Equal(t, "RATE_LIMITED", res.errorCode())
}
