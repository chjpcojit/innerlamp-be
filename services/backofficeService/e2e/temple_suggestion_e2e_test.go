package e2e

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const e2ePassword = "Correct-Horse-9"

type tsEnv struct {
	base string
	// one logged-in client per role
	admin, editor, viewer *apiClient
	anon                  *apiClient
	dbExec                func(q string, args ...any)
}

// newTempleSuggestionEnv starts Postgres and the real binary, seeds a minimal
// administrative dataset and three active accounts (one per role, password
// already changed), and logs each of them in through the real login route.
func newTempleSuggestionEnv(t *testing.T) *tsEnv {
	t.Helper()
	requireE2E(t)
	host, port, stop := startPostgres(t)
	t.Cleanup(stop)
	db := openDB(t, host, port)

	exec1(t, db, `INSERT INTO addr_new_provinces (code, name, full_name, type, sort_order) VALUES
		('79', 'Hồ Chí Minh', 'Thành phố Hồ Chí Minh', 'city', 1), ('01', 'Hà Nội', 'Thành phố Hà Nội', 'city', 2)`)
	exec1(t, db, `INSERT INTO addr_new_wards (code, province_code, name, full_name, type, sort_order) VALUES
		('26740', '79', 'Sài Gòn', 'Phường Sài Gòn', 'ward', 1), ('00004', '01', 'Ba Đình', 'Phường Ba Đình', 'ward', 1)`)

	// Accounts exist before the server starts, so the bootstrap admin is skipped.
	hash, err := bcrypt.GenerateFromPassword([]byte(e2ePassword), bcrypt.MinCost)
	require.NoError(t, err)
	for _, a := range []struct{ email, role string }{
		{"admin@e2e.test", "admin"}, {"editor@e2e.test", "content_editor"}, {"viewer@e2e.test", "viewer"},
	} {
		exec1(t, db, `INSERT INTO admins (email, password_hash, name, role) VALUES ($1, $2, $3, $4)`,
			a.email, string(hash), a.role, a.role)
	}

	base, kill := startServer(t, host, port)
	t.Cleanup(kill)

	login := func(email string) *apiClient {
		c := &apiClient{base: base}
		status, env := c.do(t, http.MethodPost, "/v1/admin/auth/login", map[string]string{"email": email, "password": e2ePassword})
		require.Equal(t, http.StatusOK, status, "login %s: %+v", email, env)
		var d struct {
			AccessToken string `json:"access_token"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &d))
		c.token = d.AccessToken
		return c
	}
	return &tsEnv{
		base: base, anon: &apiClient{base: base},
		admin: login("admin@e2e.test"), editor: login("editor@e2e.test"), viewer: login("viewer@e2e.test"),
		dbExec: func(q string, args ...any) { exec1(t, db, q, args...) },
	}
}

func (e *tsEnv) addStructured(t *testing.T, name string) int64 {
	t.Helper()
	// Through SQL: the submission route belongs to appService and has its own E2E.
	e.dbExec(`INSERT INTO temple_suggestions (name, province_code, province, ward_code, ward, address)
		VALUES ($1, '79', 'Hồ Chí Minh', '26740', 'Phường Sài Gòn', '12 Nguyễn Huệ')`, name)
	return e.lastID(t)
}

func (e *tsEnv) addLegacy(t *testing.T, name string) int64 {
	t.Helper()
	e.dbExec(`INSERT INTO temple_suggestions (name, province, address) VALUES ($1, 'Quảng Ninh', 'Thôn 1, xã A')`, name)
	return e.lastID(t)
}

func (e *tsEnv) lastID(t *testing.T) int64 {
	t.Helper()
	status, env := e.admin.do(t, http.MethodGet, "/v1/admin/temple-suggestions?status=pending&page=1", nil)
	require.Equal(t, http.StatusOK, status)
	var page struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &page))
	require.NotEmpty(t, page.Items)
	return page.Items[len(page.Items)-1].ID // oldest first, so the last one is the newest
}

func createBody(overrides map[string]any) map[string]any {
	b := map[string]any{
		"name": "Chùa Mới", "type": "CHUA", "region": "south", "sect": "Bắc tông", "established_year": 1990,
		"heritage_rank": "DI_TICH_CAP_TINH", "latitude": 10.77, "longitude": 106.69,
		"province_code": "79", "ward_code": "26740", "address": "12 Nguyễn Huệ",
		"image_url": "/static/images/x.jpg",
	}
	for k, v := range overrides {
		if v == nil {
			delete(b, k)
		} else {
			b[k] = v
		}
	}
	return b
}

func TestTempleSuggestion_E2E_Access(t *testing.T) {
	e := newTempleSuggestionEnv(t)

	for _, p := range []struct{ method, path string }{
		{http.MethodGet, "/v1/admin/temple-suggestions"},
		{http.MethodGet, "/v1/admin/temple-suggestions/1"},
		{http.MethodPost, "/v1/admin/temple-suggestions/1/create-temple"},
	} {
		status, env := e.anon.do(t, p.method, p.path, nil)
		assert.Equal(t, http.StatusUnauthorized, status, p.path)
		assert.False(t, env.Success)

		status, env = e.viewer.do(t, p.method, p.path, nil)
		assert.Equal(t, http.StatusForbidden, status, "viewer %s", p.path)
		assert.Equal(t, "FORBIDDEN", env.Error.Code)
	}
}

func TestTempleSuggestion_E2E_QueueAndDetail(t *testing.T) {
	e := newTempleSuggestionEnv(t)
	structured := e.addStructured(t, "Chùa có mã")
	legacy := e.addLegacy(t, "Chùa cũ")

	status, env := e.editor.do(t, http.MethodGet, "/v1/admin/temple-suggestions", nil)
	require.Equal(t, http.StatusOK, status)
	var page struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &page))
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 2)
	// oldest first
	assert.EqualValues(t, structured, page.Items[0]["id"])
	assert.Equal(t, "79", page.Items[0]["province_code"])
	assert.Equal(t, "Phường Sài Gòn", page.Items[0]["ward"])
	assert.EqualValues(t, legacy, page.Items[1]["id"])
	assert.Nil(t, page.Items[1]["province_code"])
	assert.Nil(t, page.Items[1]["ward"])
	assert.Equal(t, "Quảng Ninh", page.Items[1]["province"])

	for _, tc := range []struct{ query, code string }{
		{"?status=bogus", "INVALID_STATUS"}, {"?page=0", "INVALID_PAGE"}, {"?page=abc", "INVALID_PAGE"},
	} {
		status, env = e.editor.do(t, http.MethodGet, "/v1/admin/temple-suggestions"+tc.query, nil)
		assert.Equal(t, http.StatusBadRequest, status, tc.query)
		assert.Equal(t, tc.code, env.Error.Code, tc.query)
	}

	// beyond the last page: 200 with no items
	status, env = e.editor.do(t, http.MethodGet, "/v1/admin/temple-suggestions?page=9", nil)
	require.Equal(t, http.StatusOK, status)
	var past struct {
		Items []any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &past))
	assert.Empty(t, past.Items)

	status, env = e.editor.do(t, http.MethodGet, "/v1/admin/temple-suggestions/999", nil)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "TEMPLE_SUGGESTION_NOT_FOUND", env.Error.Code)
}

func TestTempleSuggestion_E2E_CreateTemple(t *testing.T) {
	e := newTempleSuggestionEnv(t)
	id := e.addStructured(t, "Chùa Mới")
	path := "/v1/admin/temple-suggestions/" + itoa(id) + "/create-temple"

	// Validation failures leave the suggestion pending.
	for _, tc := range []struct {
		name  string
		over  map[string]any
		field string
	}{
		{"unknown province", map[string]any{"province_code": "99"}, "province_code"},
		{"ward of another province", map[string]any{"province_code": "01"}, "ward_code"},
		{"missing ward_code", map[string]any{"ward_code": nil}, "ward_code"},
		{"missing address", map[string]any{"address": nil}, "address"},
		{"bad type", map[string]any{"type": "NHA_THO"}, "type"},
		{"missing latitude", map[string]any{"latitude": nil}, "latitude"},
	} {
		status, env := e.editor.do(t, http.MethodPost, path, createBody(tc.over))
		assert.Equal(t, http.StatusBadRequest, status, tc.name)
		assert.Equal(t, "VALIDATION_ERROR", env.Error.Code, tc.name)
		assert.Equal(t, tc.field, env.Details["field"], tc.name)
	}
	status, env := e.editor.do(t, http.MethodGet, "/v1/admin/temple-suggestions/"+itoa(id), nil)
	require.Equal(t, http.StatusOK, status)
	var d map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &d))
	assert.Equal(t, "pending", d["status"])

	// The names in the request are ignored: they come from the dataset.
	status, env = e.editor.do(t, http.MethodPost, path, createBody(map[string]any{"province": "Hà Nội"}))
	assert.Equal(t, http.StatusBadRequest, status, "unknown field is refused by the strict decoder")

	status, env = e.editor.do(t, http.MethodPost, path, createBody(nil))
	require.Equal(t, http.StatusCreated, status, "%+v", env)
	var created struct {
		Suggestion struct {
			ID       int64  `json:"id"`
			Status   string `json:"status"`
			TempleID string `json:"temple_id"`
		} `json:"suggestion"`
		Temple struct {
			ID       string `json:"id"`
			ImageURL string `json:"image_url"`
			Location struct {
				Address      string `json:"address"`
				Province     string `json:"province"`
				ProvinceCode string `json:"province_code"`
				WardCode     string `json:"ward_code"`
				Ward         string `json:"ward"`
				FullAddress  string `json:"full_address"`
			} `json:"location"`
		} `json:"temple"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &created))
	assert.Equal(t, "created", created.Suggestion.Status)
	assert.Equal(t, "temple_south_001", created.Temple.ID)
	assert.Equal(t, created.Temple.ID, created.Suggestion.TempleID)
	assert.Equal(t, "https://cdn.test/static/images/x.jpg", created.Temple.ImageURL)
	loc := created.Temple.Location
	assert.Equal(t, "79", loc.ProvinceCode)
	assert.Equal(t, "26740", loc.WardCode)
	assert.Equal(t, "Phường Sài Gòn", loc.Ward)
	assert.Equal(t, "Hồ Chí Minh", loc.Province)
	assert.Equal(t, "12 Nguyễn Huệ", loc.Address)
	assert.Equal(t, "12 Nguyễn Huệ, Phường Sài Gòn, Hồ Chí Minh", loc.FullAddress)

	// Second create → 409; the suggestion keeps its original data (AC26).
	status, env = e.editor.do(t, http.MethodPost, path, createBody(nil))
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, "SUGGESTION_ALREADY_CREATED", env.Error.Code)

	// It left the pending queue and shows under created.
	status, env = e.editor.do(t, http.MethodGet, "/v1/admin/temple-suggestions?status=created", nil)
	require.Equal(t, http.StatusOK, status)
	var cp struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &cp))
	assert.Equal(t, 1, cp.Total)

	status, env = e.editor.do(t, http.MethodPost, "/v1/admin/temple-suggestions/424242/create-temple", createBody(nil))
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "TEMPLE_SUGGESTION_NOT_FOUND", env.Error.Code)
}

func TestTempleSuggestion_E2E_LegacySuggestionTakesCodesFromRequest(t *testing.T) {
	e := newTempleSuggestionEnv(t)
	id := e.addLegacy(t, "Chùa cũ")
	status, env := e.admin.do(t, http.MethodPost, "/v1/admin/temple-suggestions/"+itoa(id)+"/create-temple",
		createBody(map[string]any{"region": "north", "province_code": "01", "ward_code": "00004", "address": "Ngõ 5"}))
	require.Equal(t, http.StatusCreated, status, "%+v", env)
	var d struct {
		Temple struct {
			ID       string `json:"id"`
			Location struct {
				FullAddress string `json:"full_address"`
			} `json:"location"`
		} `json:"temple"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &d))
	assert.Equal(t, "temple_north_001", d.Temple.ID)
	assert.Equal(t, "Ngõ 5, Phường Ba Đình, Hà Nội", d.Temple.Location.FullAddress)
}

func TestTempleSuggestion_E2E_RejectedReopen(t *testing.T) {
	e := newTempleSuggestionEnv(t)
	id := e.addStructured(t, "Bị từ chối")
	e.dbExec(`UPDATE temple_suggestions SET status='rejected', reviewed_by=(SELECT id FROM admins LIMIT 1),
		reviewed_at=now(), reject_reason='rác' WHERE id=$1`, id)
	path := "/v1/admin/temple-suggestions/" + itoa(id) + "/create-temple"

	status, env := e.editor.do(t, http.MethodPost, path, createBody(nil))
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, "FORBIDDEN", env.Error.Code)

	status, _ = e.admin.do(t, http.MethodPost, path, createBody(nil))
	assert.Equal(t, http.StatusCreated, status)

	status, env = e.admin.do(t, http.MethodGet, "/v1/admin/temple-suggestions/"+itoa(id), nil)
	require.Equal(t, http.StatusOK, status)
	var d map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &d))
	assert.Equal(t, "created", d["status"])
	assert.Nil(t, d["reject_reason"])
}

func TestTempleSuggestion_E2E_RaceOneWins(t *testing.T) {
	e := newTempleSuggestionEnv(t)
	id := e.addStructured(t, "Đua")
	path := "/v1/admin/temple-suggestions/" + itoa(id) + "/create-temple"

	const n = 4
	statuses := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &apiClient{base: e.base, token: e.editor.token}
			statuses[i], _ = c.do(t, http.MethodPost, path, createBody(nil))
		}()
	}
	wg.Wait()

	created, conflict := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		}
	}
	assert.Equal(t, 1, created, "statuses: %v", statuses)
	assert.Equal(t, n-1, conflict, "statuses: %v", statuses)
}
