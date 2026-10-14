package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeity_TypeFilter_E2E drives GET /v1/deities?type= (docs/specs/features/deity/,
// AC 20-22) against the real binary and a fresh Postgres seeded with the real
// deity data (startPostgres).
//
// What is here and not in the per-layer tests: that the `type` query parameter
// reaches the usecase through the real mux, that the 400 travels as the shared
// envelope with details.field = "type", and that the counts match the seeded rows.
func TestDeity_TypeFilter_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, cleanupSrv := startServer(t, host, port, nil)
	t.Cleanup(cleanupSrv)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	db := openDB(t, host, port)

	type page struct {
		Items []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"items"`
		Page       int `json:"page"`
		PageSize   int `json:"page_size"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	}
	count := func(typ string) int {
		var n int
		q := `SELECT count(*) FROM deities`
		args := []any{}
		if typ != "" {
			q += ` WHERE type = $1`
			args = append(args, typ)
		}
		require.NoError(t, db.QueryRow(context.Background(), q, args...).Scan(&n))
		return n
	}

	t.Run("AC20 each type returns only that type and counts only that type", func(t *testing.T) {
		api.t = t
		for _, typ := range []string{"BUDDHA", "BODHISATTVA", "DEVA"} {
			want := count(typ)
			res := api.get("/v1/deities?type=" + typ)
			require.Equal(t, http.StatusOK, res.status, res.raw)
			var p page
			res.data(&p)
			assert.Equal(t, want, p.Total, "type=%s total", typ)
			require.NotEmpty(t, p.Items, "type=%s: the seed has at least one", typ)
			for _, it := range p.Items {
				assert.Equal(t, typ, it.Type, "type=%s leaked %s", typ, it.ID)
			}
		}
	})

	t.Run("AC4 no type or empty type lists every type", func(t *testing.T) {
		api.t = t
		all := count("")
		for _, path := range []string{"/v1/deities", "/v1/deities?type="} {
			res := api.get(path)
			require.Equal(t, http.StatusOK, res.status, res.raw)
			var p page
			res.data(&p)
			assert.Equal(t, all, p.Total, path)
		}
	})

	t.Run("AC22 page applies within the filtered list", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/deities?type=BUDDHA&page=2")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var p page
		res.data(&p)
		assert.Equal(t, 2, p.Page)
		assert.Empty(t, p.Items, "BUDDHA fits one page, so page 2 is empty")
		assert.Equal(t, count("BUDDHA"), p.Total)
		var raw struct {
			Items json.RawMessage `json:"items"`
		}
		res.data(&raw)
		assert.JSONEq(t, `[]`, string(raw.Items), "items must be [] not null")
	})

	t.Run("AC21 an unknown type is 400 INVALID_DEITY_TYPE", func(t *testing.T) {
		api.t = t
		for _, typ := range []string{"buddha", "Buddha", "X", "BUDDHA,DEVA"} {
			res := api.get("/v1/deities?type=" + typ)
			assert.Equal(t, http.StatusBadRequest, res.status, "type=%q: %s", typ, res.raw)
			assert.Equal(t, "INVALID_DEITY_TYPE", res.errorCode(), "type=%q", typ)
			assert.Equal(t, "type", res.details()["field"], "type=%q", typ)
		}
	})

	t.Run("AC22 an invalid page is reported before an invalid type", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/deities?type=X&page=0")
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "INVALID_PAGE", res.errorCode())
	})
}
