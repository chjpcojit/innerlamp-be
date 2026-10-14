package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tanItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	IssuingBody struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"issuing_body"`
}

type tanList struct {
	Receive bool `json:"receive"`
	Temple  *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"temple"`
	Items []tanItem `json:"items"`
	Page  struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
		Total    int `json:"total"`
	} `json:"page"`
}

func (l tanList) ids() []string {
	out := []string{}
	for _, i := range l.Items {
		out = append(out, i.ID)
	}
	return out
}

// TestTempleAnnouncementFlow_E2E drives docs/specs/features/temple-announcement
// (app side) against the real binary and a fresh Postgres: auth + the dedicated
// rate limiter wiring, the temple read at call time, and the receive switch.
func TestTempleAnnouncementFlow_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	const limitMax = 20
	baseURL, stop := startServer(t, host, port, configOverrides{
		"temple_announcement": {"rate_limit": map[string]any{"max": limitMax, "window": "1m"}},
	})
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	db := openDB(t, host, port)

	for _, id := range []string{"tan_a", "tan_b"} {
		execSQL(t, db, `INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority, latitude, longitude, address)
			VALUES ($1, $2, 'CHUA', 'north', 'Thiền', 1000, 'CHUA_XEP_HANG', 7, 21.0, 105.8, 'Hà Nội')`, id, "Chùa "+id)
		execSQL(t, db, `INSERT INTO temple_issuing_bodies (id, temple_id, name) VALUES ($1, $2, 'Ban Trị sự')`, "tib_"+id, id)
	}
	ann := func(id, temple, body string, pubAgo, expIn time.Duration, revoked bool) {
		var rev any
		if revoked {
			rev = time.Now().Add(-time.Minute)
		}
		execSQL(t, db, `INSERT INTO temple_announcements (id, temple_id, issuing_body_id, title, body, published_at, expires_at, revoked_at)
			VALUES ($1, $2, $3, $4, 'nội dung', $5, $6, $7)`,
			id, temple, body, "Tiêu đề "+id, time.Now().Add(-pubAgo), time.Now().Add(expIn), rev)
	}
	ann("tann_a1", "tan_a", "tib_tan_a", 2*time.Hour, 48*time.Hour, false)
	ann("tann_a2", "tan_a", "tib_tan_a", time.Hour, 48*time.Hour, false)
	ann("tann_a_expired", "tan_a", "tib_tan_a", 3*time.Hour, -time.Minute, false)
	ann("tann_a_revoked", "tan_a", "tib_tan_a", 3*time.Hour, 48*time.Hour, true)
	ann("tann_b1", "tan_b", "tib_tan_b", time.Hour, 48*time.Hour, false)

	const list = "/v1/users/me/temple-announcements"
	const settings = "/v1/users/me/temple-announcement-settings"
	deviceB := map[string]any{"platform": "android", "device_model": "Pixel 8", "os_version": "14"}

	read := func(token string) tanList {
		t.Helper()
		res := api.do(http.MethodGet, list, nil, token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var l tanList
		res.data(&l)
		return l
	}

	t.Run("TA2: all three routes need an access token", func(t *testing.T) {
		api.t = t
		pair := api.login(api.registerUser(), deviceA)
		for _, rt := range []struct {
			method, path string
			body         any
		}{
			{http.MethodGet, list, nil},
			{http.MethodGet, settings, nil},
			{http.MethodPut, settings, map[string]any{"receive": true}},
		} {
			none := api.do(rt.method, rt.path, rt.body, "")
			assert.Equal(t, http.StatusUnauthorized, none.status, "%s %s: %s", rt.method, rt.path, none.raw)
			refresh := api.do(rt.method, rt.path, rt.body, pair.RefreshToken)
			assert.Equal(t, http.StatusUnauthorized, refresh.status, "%s %s with a refresh token", rt.method, rt.path)
		}
	})

	t.Run("TA5/TA12: a user without a temple gets empty items and receive on", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		l := read(token)

		assert.True(t, l.Receive)
		assert.Nil(t, l.Temple)
		assert.Empty(t, l.Items)
		assert.Equal(t, 0, l.Page.Total)
	})

	t.Run("TA1/TA4/TA6/TA8/TA15: temple read at call time, only active ones, switch kept across a temple change", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		patch := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"ordination_temple_id": "tan_a"}, token)
		require.Equal(t, http.StatusOK, patch.status, patch.raw)
		l := read(token)
		require.NotNil(t, l.Temple)
		assert.Equal(t, "tan_a", l.Temple.ID)
		assert.Equal(t, []string{"tann_a2", "tann_a1"}, l.ids(), "newest first; expired and revoked are gone")
		assert.Equal(t, 2, l.Page.Total)
		assert.Equal(t, "Ban Trị sự", l.Items[0].IssuingBody.Name)

		off := api.do(http.MethodPut, settings, map[string]any{"receive": false}, token)
		require.Equal(t, http.StatusOK, off.status, off.raw)

		api.do(http.MethodPatch, "/v1/users/me", map[string]any{"ordination_temple_id": "tan_b"}, token)
		get := api.do(http.MethodGet, settings, nil, token)
		var s struct {
			Receive bool `json:"receive"`
		}
		get.data(&s)
		assert.False(t, s.Receive, "TA15: changing the temple keeps the switch")
		off2 := read(token)
		assert.False(t, off2.Receive)
		assert.Empty(t, off2.Items, "TA7")
		assert.Nil(t, off2.Temple)

		api.do(http.MethodPut, settings, map[string]any{"receive": true}, token)
		assert.Equal(t, []string{"tann_b1"}, read(token).ids(), "TA6: next read follows the new temple")

		api.do(http.MethodPatch, "/v1/users/me", map[string]any{"ordination_temple_id": nil}, token)
		assert.Empty(t, read(token).Items)
	})

	t.Run("TA12: two devices of one user see the same list and switch", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		a := api.login(phone, deviceA).AccessToken
		b := api.login(phone, deviceB).AccessToken
		api.do(http.MethodPatch, "/v1/users/me", map[string]any{"ordination_temple_id": "tan_a"}, a)

		assert.Equal(t, read(a).ids(), read(b).ids())
		api.do(http.MethodPut, settings, map[string]any{"receive": false}, b)
		assert.False(t, read(a).Receive)
	})

	t.Run("TA10/R-1: paging and INVALID_PAGE", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken
		api.do(http.MethodPatch, "/v1/users/me", map[string]any{"ordination_temple_id": "tan_a"}, token)

		res := api.do(http.MethodGet, list+"?page=2&page_size=1", nil, token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var l tanList
		res.data(&l)
		assert.Equal(t, []string{"tann_a1"}, l.ids())
		assert.Equal(t, 2, l.Page.Total)

		bad := api.do(http.MethodGet, list+"?page=0", nil, token)
		assert.Equal(t, http.StatusBadRequest, bad.status)
		assert.Equal(t, "INVALID_PAGE", bad.errorCode())
	})

	t.Run("TA14: bad switch bodies are 400 and change nothing", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken
		for _, body := range []any{
			map[string]any{}, map[string]any{"receive": "no"}, map[string]any{"receive": true, "x": 1},
		} {
			res := api.do(http.MethodPut, settings, body, token)
			assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
			assert.Equal(t, "VALIDATION_ERROR", res.errorCode())
		}
		get := api.do(http.MethodGet, settings, nil, token)
		var s struct {
			Receive bool `json:"receive"`
		}
		get.data(&s)
		assert.True(t, s.Receive)
	})

	t.Run("TA32/TA33: own limiter, 429 RATE_LIMITED, profile budget untouched", func(t *testing.T) {
		api.t = t
		busy := api.login(api.registerUser(), deviceA).AccessToken
		other := api.login(api.registerUser(), deviceA).AccessToken

		for i := 1; i <= limitMax; i++ {
			res := api.do(http.MethodGet, list, nil, busy)
			require.Equal(t, http.StatusOK, res.status, "request %d: %s", i, res.raw)
		}
		limited := api.do(http.MethodGet, list, nil, busy)
		assert.Equal(t, http.StatusTooManyRequests, limited.status, limited.raw)
		assert.Equal(t, "RATE_LIMITED", limited.errorCode())
		assert.Equal(t, http.StatusTooManyRequests, api.do(http.MethodGet, settings, nil, busy).status, "settings share the budget")

		profile := api.do(http.MethodGet, "/v1/users/me", nil, busy)
		assert.Equal(t, http.StatusOK, profile.status, "the profile budget is separate: %s", profile.raw)
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, list, nil, other).status, "another account has its own budget")
	})
}
