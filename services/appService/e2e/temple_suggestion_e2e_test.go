package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTempleSuggestion_E2E drives the public "suggest a temple" route
// (docs/specs/features/temple-suggestion/, AC1-AC8, AC3a-AC3b) against the real
// binary and a fresh Postgres.
//
// What is here and not in the per-layer tests: that cmd/main.go registers the
// route, puts the per-IP limit in front of it and the config's 5/hour reaches the
// limiter, that the use case really reads the administrative-units tables the
// seeder fills, that the stored row carries the dataset's names rather than
// anything the client sent, and that no read route for suggestions exists.
func TestTempleSuggestion_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, cleanupSrv := startServer(t, host, port, nil)
	t.Cleanup(cleanupSrv)
	client := &http.Client{Timeout: 5 * time.Second}
	api := &apiClient{t: t, baseURL: baseURL, http: client}
	db := openDB(t, host, port)

	// The seeder is covered by TestAdministrativeUnits_E2E; here a two-province
	// dataset is enough to tell "unknown", "wrong province" and "right pair" apart.
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO addr_dataset_versions (version, source_note) VALUES ('e2e', 'temple suggestion e2e')`,
		`INSERT INTO addr_new_provinces (code, name, full_name, type, sort_order) VALUES
			('79', 'Hồ Chí Minh', 'Thành phố Hồ Chí Minh', 'city', 1),
			('01', 'Hà Nội', 'Thành phố Hà Nội', 'city', 2)`,
		`INSERT INTO addr_new_wards (code, province_code, name, full_name, type, sort_order) VALUES
			('26737', '79', 'Tân Định', 'Phường Tân Định', 'ward', 1),
			('00004', '01', 'Ba Đình', 'Phường Ba Đình', 'ward', 1)`,
	} {
		_, err := db.Exec(ctx, q)
		require.NoError(t, err, q)
	}
	valid := map[string]any{
		"name": "Chùa Mới", "province_code": "79", "ward_code": "26737", "address": "12 Nguyễn Huệ",
	}

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

	// AC5: the codes are stored as sent and the names come from the dataset.
	var provinceCode, province, wardCode, ward, address string
	require.NoError(t, db.QueryRow(ctx,
		`SELECT province_code, province, ward_code, ward, address FROM temple_suggestions WHERE id = $1`, created.ID,
	).Scan(&provinceCode, &province, &wardCode, &ward, &address))
	assert.Equal(t, "79", provinceCode)
	assert.Equal(t, "Hồ Chí Minh", province)
	assert.Equal(t, "26737", wardCode)
	assert.Equal(t, "Phường Tân Định", ward)
	assert.Equal(t, "12 Nguyễn Huệ", address)

	// AC6: the same suggestion again is accepted, not de-duplicated.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", valid, "")
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	var again struct {
		ID int64 `json:"id"`
	}
	res.data(&again)
	assert.NotEqual(t, created.ID, again.ID)

	// AC3a: a province that does not exist names province_code.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", map[string]any{
		"name": "Chùa", "province_code": "999", "ward_code": "26737", "address": "x",
	}, "")
	assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
	assert.Equal(t, "VALIDATION_ERROR", res.errorCode())
	assert.Equal(t, "province_code", res.details()["field"])

	// AC3a: a real ward of another province is refused as ward_code.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", map[string]any{
		"name": "Chùa", "province_code": "01", "ward_code": "26737", "address": "x",
	}, "")
	assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
	assert.Equal(t, "VALIDATION_ERROR", res.errorCode())
	assert.Equal(t, "ward_code", res.details()["field"])

	// AC3b: the pre-v3.0 free-text province field is an unknown field now.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", map[string]any{
		"name": "Chùa", "province": "Quảng Ninh", "address": "x",
	}, "")
	assert.Equal(t, http.StatusBadRequest, res.status, res.raw)

	var stored int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_suggestions`).Scan(&stored))
	assert.Equal(t, 2, stored, "every refused request stored nothing")

	// AC7: 5 per IP per hour (dev config). Five requests above counted (the
	// wrong-key call did not, and neither does a refusal before the limiter), so
	// the sixth is limited. A valid request is refused here too: the limit is on
	// the caller, not on the content.
	res = api.do(http.MethodPost, "/v1/temple-suggestions", valid, "")
	assert.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
	assert.Equal(t, "RATE_LIMITED", res.errorCode())
}
