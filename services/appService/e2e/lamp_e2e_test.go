package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
)

type lampData struct {
	ID           int64   `json:"id"`
	LampType     string  `json:"lamp_type"`
	Intention    string  `json:"intention"`
	Recipient    string  `json:"recipient"`
	Prayer       *string `json:"prayer"`
	DurationDays int     `json:"duration_days"`
	IsAnonymous  bool    `json:"is_anonymous"`
	LighterName  string  `json:"lighter_name"`
	LitAt        string  `json:"lit_at"`
	EndAt        string  `json:"end_at"`
}

type gardenLampData struct {
	ID          int64  `json:"id"`
	LampType    string `json:"lamp_type"`
	Recipient   string `json:"recipient"`
	LighterName string `json:"lighter_name"`
	EndAt       string `json:"end_at"`
}

type gardenData struct {
	Items      []gardenLampData `json:"items"`
	Total      int              `json:"total"`
	Counts     map[string]int   `json:"counts"`
	HasMore    bool             `json:"has_more"`
	NextCursor *string          `json:"next_cursor"`
}

func lampBody(recipient, lampType string, days int) map[string]any {
	return map[string]any{
		"lamp_type": lampType, "intention": "GRATITUDE", "recipient": recipient,
		"prayer": "lời nguyện riêng tư của " + recipient, "duration_days": days,
	}
}

// userIDOf reads the numeric user id out of an access token, so the test can set
// state (a dharma name) the API has no route for.
func userIDOf(t *testing.T, token string) string {
	t.Helper()
	id, _ := decodeClaims(t, token)["user_id"].(string)
	require.NotEmpty(t, id, "access token has no user_id claim")
	return id
}

func (c *apiClient) garden(query, bearer string) (*apiResponse, gardenData) {
	c.t.Helper()
	path := "/v1/lamps"
	if query != "" {
		path += "?" + query
	}
	res := c.do(http.MethodGet, path, nil, bearer)
	var g gardenData
	if res.status == http.StatusOK {
		res.data(&g)
	}
	return res, g
}

// TestLampFlow_E2E drives docs/specs/features/lamp against the real binary and a
// fresh Postgres.
//
// What is here and not in the per-layer tests: that cmd/main.go wraps both lamp
// routes in the access-token middleware (a visitor sees neither the garden nor
// its counts), that the cap, expiry and the double-tap window hold with the real
// config and real rows, and that the garden never carries the intention or the
// prayer of anyone.
func TestLampFlow_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	// A cap of 3 keeps the cap test short; every other subtest stays under it per user.
	baseURL, stop := startServer(t, host, port, configOverrides{"lamp": {"max_active_per_user": 3}})
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	db := openDB(t, host, port)

	t.Run("AC2/AC14: both routes are behind the access-token middleware", func(t *testing.T) {
		api.t = t
		pair := api.login(api.registerUser(), deviceA)

		for _, rt := range []struct{ method, path string }{
			{http.MethodPost, "/v1/lamps"},
			{http.MethodGet, "/v1/lamps"},
		} {
			var body any
			if rt.method == http.MethodPost {
				body = lampBody("x", "OIL_LAMP", 1)
			}
			none := api.do(rt.method, rt.path, body, "")
			assert.Equal(t, http.StatusUnauthorized, none.status, "%s %s without a token: %s", rt.method, rt.path, none.raw)
			assert.Equal(t, "UNAUTHORIZED", none.errorCode())
			assert.NotContains(t, none.raw, `"total"`, "a visitor learns no count")

			garbage := api.do(rt.method, rt.path, body, "not-a-jwt")
			assert.Equal(t, http.StatusUnauthorized, garbage.status, "%s %s with garbage", rt.method, rt.path)

			wrongType := api.do(rt.method, rt.path, body, pair.RefreshToken)
			assert.Equal(t, http.StatusUnauthorized, wrongType.status, "%s %s with a refresh token", rt.method, rt.path)
		}
		assert.Equal(t, 0, countRows(t, db, "lamps"), "nothing was created by an unauthenticated call")
	})

	t.Run("AC1/AC12/AC13/AC15/AC16/AC18: light, then every user sees it in the garden", func(t *testing.T) {
		api.t = t
		monk := api.login(api.registerUser(), deviceA).AccessToken
		execSQL(t, db, `UPDATE users SET dharma_name = $2 WHERE id = $1`, userIDOf(t, monk), "Thích Minh An")
		lay := api.login(api.registerUser(), deviceA).AccessToken
		before := countRows(t, db, "lamps")

		res := api.do(http.MethodPost, "/v1/lamps", lampBody("Bà ngoại", "OIL_LAMP", 7), monk)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var lit lampData
		res.data(&lit)
		assert.NotZero(t, lit.ID)
		assert.Equal(t, "OIL_LAMP", lit.LampType)
		assert.Equal(t, "Thích Minh An", lit.LighterName)
		require.NotNil(t, lit.Prayer)
		litAt, err := time.Parse(time.RFC3339, lit.LitAt)
		require.NoError(t, err)
		endAt, err := time.Parse(time.RFC3339, lit.EndAt)
		require.NoError(t, err)
		assert.Equal(t, 7*24*time.Hour, endAt.Sub(litAt), "end_at = lit_at + 7 x 24h")
		assert.True(t, strings.HasSuffix(lit.EndAt, "Z"), "times are UTC: %s", lit.EndAt)

		// A user with no dharma name, lighting anonymously a different lamp.
		anon := lampBody("Thầy", "LOTUS_CANDLE", 1)
		anon["is_anonymous"] = true
		res = api.do(http.MethodPost, "/v1/lamps", anon, monk)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var hidden lampData
		res.data(&hidden)
		assert.True(t, hidden.IsAnonymous)
		assert.Equal(t, "Vô danh", hidden.LighterName)

		res = api.do(http.MethodPost, "/v1/lamps", lampBody("Mẹ", "FLOWER_LANTERN", 21), lay)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var noName lampData
		res.data(&noName)
		assert.Equal(t, "Vô danh", noName.LighterName, "no dharma name: shown as Vô danh")

		assert.Equal(t, before+3, countRows(t, db, "lamps"))

		// Another user's garden shows all three, newest first.
		res, g := api.garden("", lay)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		require.GreaterOrEqual(t, len(g.Items), 3)
		assert.Equal(t, noName.ID, g.Items[0].ID, "newest lit first")
		assert.Equal(t, hidden.ID, g.Items[1].ID)
		assert.Equal(t, lit.ID, g.Items[2].ID)
		assert.Equal(t, "Thích Minh An", g.Items[2].LighterName)
		assert.Equal(t, "Vô danh", g.Items[1].LighterName)

		// AC15: nothing private reaches the garden.
		for _, secret := range []string{"lời nguyện riêng tư", "GRATITUDE", `"intention"`, `"prayer"`, `"user_id"`, `"is_anonymous"`} {
			assert.NotContains(t, res.raw, secret, "garden leaks %q", secret)
		}

		// AC18: the counts cover every lit lamp and one key per type.
		assert.Len(t, g.Counts, 3)
		assert.Equal(t, g.Total, g.Counts["OIL_LAMP"]+g.Counts["LOTUS_CANDLE"]+g.Counts["FLOWER_LANTERN"])
		assert.GreaterOrEqual(t, g.Counts["OIL_LAMP"], 1)
	})

	t.Run("AC17/AC18: the type filter narrows the list but never the counts", func(t *testing.T) {
		api.t = t
		tok := api.login(api.registerUser(), deviceA).AccessToken
		require.Equal(t, http.StatusCreated, api.do(http.MethodPost, "/v1/lamps", lampBody("a", "OIL_LAMP", 1), tok).status)
		require.Equal(t, http.StatusCreated, api.do(http.MethodPost, "/v1/lamps", lampBody("b", "LOTUS_CANDLE", 1), tok).status)

		_, all := api.garden("", tok)
		res, lotus := api.garden("lamp_type=LOTUS_CANDLE", tok)
		require.Equal(t, http.StatusOK, res.status, res.raw)

		require.NotEmpty(t, lotus.Items)
		for _, it := range lotus.Items {
			assert.Equal(t, "LOTUS_CANDLE", it.LampType)
		}
		assert.Equal(t, all.Total, lotus.Total, "total ignores the filter")
		assert.Equal(t, all.Counts, lotus.Counts, "counts ignore the filter")

		bad := api.do(http.MethodGet, "/v1/lamps?lamp_type=CANDLE", nil, tok)
		assert.Equal(t, http.StatusBadRequest, bad.status)
		assert.Equal(t, "INVALID_FILTER", bad.errorCode())
		assert.Equal(t, "lamp_type", bad.details()["field"])

		empty := api.do(http.MethodGet, "/v1/lamps?lamp_type=", nil, tok)
		assert.Equal(t, http.StatusBadRequest, empty.status, "an empty lamp_type is not 'all'")
	})

	t.Run("AC19/AC20/AC21: load more with a cursor never repeats or skips", func(t *testing.T) {
		api.t = t
		// A fresh user per three lamps keeps every user under the cap of 3.
		var want []int64
		for i := 0; i < 3; i++ {
			tok := api.login(api.registerUser(), deviceA).AccessToken
			for j := 0; j < 3; j++ {
				res := api.do(http.MethodPost, "/v1/lamps", lampBody(fmt.Sprintf("page-%d-%d", i, j), "OIL_LAMP", 7), tok)
				require.Equal(t, http.StatusCreated, res.status, res.raw)
				var l lampData
				res.data(&l)
				want = append([]int64{l.ID}, want...) // newest first
			}
		}
		tok := api.login(api.registerUser(), deviceA).AccessToken

		var seen []int64
		query := "limit=4"
		for pages := 0; pages < 100; pages++ {
			res, g := api.garden(query, tok)
			require.Equal(t, http.StatusOK, res.status, res.raw)
			require.LessOrEqual(t, len(g.Items), 4)
			for _, it := range g.Items {
				seen = append(seen, it.ID)
			}
			if !g.HasMore {
				assert.Nil(t, g.NextCursor, "no more: no cursor")
				break
			}
			require.NotNil(t, g.NextCursor)
			query = "limit=4&cursor=" + *g.NextCursor
		}

		// Every lamp lit above appears exactly once, in order, among everything lit.
		seenSet := map[int64]int{}
		for _, id := range seen {
			seenSet[id]++
		}
		for id, n := range seenSet {
			assert.Equal(t, 1, n, "lamp %d listed %d times", id, n)
		}
		for _, id := range want {
			assert.Equal(t, 1, seenSet[id], "lamp %d missing", id)
		}

		assertBad := func(query, code, field string) {
			res := api.do(http.MethodGet, "/v1/lamps?"+query, nil, tok)
			assert.Equal(t, http.StatusBadRequest, res.status, query)
			assert.Equal(t, code, res.errorCode(), query)
			assert.Equal(t, field, res.details()["field"], query)
		}
		assertBad("cursor=not-a-cursor", "INVALID_CURSOR", "cursor")
		assertBad("limit=0", "INVALID_LIMIT", "limit")
		assertBad("limit=-1", "INVALID_LIMIT", "limit")
		assertBad("limit=abc", "INVALID_LIMIT", "limit")

		res, g := api.garden("limit=500", tok)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.LessOrEqual(t, len(g.Items), 12, "a limit above the page size is clamped to it")
	})

	t.Run("AC9/AC23: the cap counts lit lamps only, and an expired lamp frees a place", func(t *testing.T) {
		api.t = t
		tok := api.login(api.registerUser(), deviceA).AccessToken
		var ids []int64
		for i := 0; i < 3; i++ {
			res := api.do(http.MethodPost, "/v1/lamps", lampBody(fmt.Sprintf("cap-%d", i), "OIL_LAMP", 7), tok)
			require.Equal(t, http.StatusCreated, res.status, res.raw)
			var l lampData
			res.data(&l)
			ids = append(ids, l.ID)
		}

		over := api.do(http.MethodPost, "/v1/lamps", lampBody("cap-over", "OIL_LAMP", 7), tok)
		assert.Equal(t, http.StatusConflict, over.status, over.raw)
		assert.Equal(t, "LAMP_LIMIT_REACHED", over.errorCode())

		// Let the first lamp run out: pull both of its times 10 days into the past.
		expireLamp(t, db, ids[0])
		_, g := api.garden("", tok)
		for _, it := range g.Items {
			assert.NotEqual(t, ids[0], it.ID, "an expired lamp is not in the garden")
		}

		again := api.do(http.MethodPost, "/v1/lamps", lampBody("cap-over", "OIL_LAMP", 7), tok)
		assert.Equal(t, http.StatusCreated, again.status, "the expired lamp freed a place: %s", again.raw)

		// A different user is not affected by this user's cap.
		other := api.login(api.registerUser(), deviceA).AccessToken
		assert.Equal(t, http.StatusCreated, api.do(http.MethodPost, "/v1/lamps", lampBody("free", "OIL_LAMP", 1), other).status)
	})

	t.Run("AC10: a double tap lights one lamp and the replay is 200", func(t *testing.T) {
		api.t = t
		tok := api.login(api.registerUser(), deviceA).AccessToken
		body := lampBody("Ông nội", "FLOWER_LANTERN", 1)

		first := api.do(http.MethodPost, "/v1/lamps", body, tok)
		require.Equal(t, http.StatusCreated, first.status, first.raw)
		var a lampData
		first.data(&a)

		second := api.do(http.MethodPost, "/v1/lamps", body, tok)
		require.Equal(t, http.StatusOK, second.status, "the replay is 200, not 201: %s", second.raw)
		var b lampData
		second.data(&b)
		assert.Equal(t, a.ID, b.ID)
		assert.Equal(t, a.EndAt, b.EndAt, "the replay does not move the end time")

		assert.Equal(t, 1, countRows(t, db, fmt.Sprintf("lamps WHERE user_id = %s", userIDOf(t, tok))))
	})

	t.Run("AC3-AC8/AC11: validation reaches the client with the field name", func(t *testing.T) {
		api.t = t
		tok := api.login(api.registerUser(), deviceA).AccessToken

		cases := []struct {
			name  string
			edit  func(m map[string]any)
			field string
		}{
			{"lamp type", func(m map[string]any) { m["lamp_type"] = "CANDLE" }, "lamp_type"},
			{"intention", func(m map[string]any) { m["intention"] = "WEALTH" }, "intention"},
			{"empty recipient", func(m map[string]any) { m["recipient"] = "   " }, "recipient"},
			{"long recipient", func(m map[string]any) { m["recipient"] = strings.Repeat("ạ", 101) }, "recipient"},
			{"long prayer", func(m map[string]any) { m["prayer"] = strings.Repeat("ế", 301) }, "prayer"},
			{"duration", func(m map[string]any) { m["duration_days"] = 5 }, "duration_days"},
			{"anonymous flag not a bool", func(m map[string]any) { m["is_anonymous"] = "yes" }, "is_anonymous"},
		}
		for _, c := range cases {
			body := lampBody("ok", "OIL_LAMP", 1)
			c.edit(body)
			res := api.do(http.MethodPost, "/v1/lamps", body, tok)
			assert.Equal(t, http.StatusBadRequest, res.status, "%s: %s", c.name, res.raw)
			assert.Equal(t, "VALIDATION_ERROR", res.errorCode(), c.name)
			assert.Equal(t, c.field, res.details()["field"], c.name)
		}

		for _, forbidden := range []string{"id", "user_id", "lit_at", "end_at", "lighter_name"} {
			body := lampBody("ok", "OIL_LAMP", 1)
			body[forbidden] = "x"
			res := api.do(http.MethodPost, "/v1/lamps", body, tok)
			assert.Equal(t, http.StatusBadRequest, res.status, "server-owned %s must be rejected: %s", forbidden, res.raw)
		}

		// 100 characters exactly is fine; the prayer may be omitted or blank.
		ok := lampBody(strings.Repeat("ạ", 100), "OIL_LAMP", 1)
		ok["prayer"] = "   "
		res := api.do(http.MethodPost, "/v1/lamps", ok, tok)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var l lampData
		res.data(&l)
		assert.Nil(t, l.Prayer, "a blank prayer is stored as null")
	})
}

// expireLamp moves a lamp's lit and end times ten days back, so it is out as of now.
func expireLamp(t *testing.T, db database.Database, id int64) {
	t.Helper()
	execSQL(t, db, `UPDATE lamps SET lit_at = lit_at - interval '10 days', end_at = end_at - interval '10 days' WHERE id = $1`, id)
}
