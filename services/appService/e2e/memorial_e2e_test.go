package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// do sends any method with an optional JSON body and bearer token, and returns the
// decoded envelope. (apiClient.post in login_e2e_test.go is POST-only.)
func (c *apiClient) do(method, path string, body any, bearer string) *apiResponse {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	require.NoError(c.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)

	res := &apiResponse{t: c.t, status: resp.StatusCode, header: resp.Header, contentType: resp.Header.Get("Content-Type"), raw: string(raw)}
	require.NoErrorf(c.t, json.Unmarshal(raw, &res.envelope), "%s %s %s: body is not the JSON envelope: %q", method, path, resp.Status, raw)
	return res
}

func memorialBody(name string) map[string]any {
	return map[string]any{
		"full_name": name, "relationship": "FATHER", "death_calendar": "solar", "death_date": "2015-03-04",
	}
}

type memorialData struct {
	ID             int64          `json:"id"`
	FullName       string         `json:"full_name"`
	Relationship   string         `json:"relationship"`
	DeathDate      *string        `json:"death_date"`
	DeathCalendar  string         `json:"death_calendar"`
	DeathDay       *int           `json:"death_day"`
	DeathMonth     *int           `json:"death_month"`
	DeathLeapMonth *bool          `json:"death_leap_month"`
	Note           *string        `json:"note"`
	PhotoURL       *string        `json:"photo_url"`
	Reminder       map[string]any `json:"reminder"`
	AddToOS        bool           `json:"add_to_os_reminder"`
}

// TestMemorialFlow_E2E drives the sổ giỗ chạp spec (docs/specs/features/memorial/,
// v1 without photos) against the real binary and a fresh Postgres.
//
// What is here and not in the per-layer tests: that cmd/main.go wraps every
// memorial route in the access-token middleware, that ownership holds across
// real users and real DB rows, and the 100-per-user cap from the running config.
func TestMemorialFlow_E2E(t *testing.T) {
	baseURL, client := newE2EServer(t)
	api := &apiClient{t: t, baseURL: baseURL, http: client}

	routes := []struct{ method, path string }{
		{http.MethodPost, "/v1/memorials"},
		{http.MethodGet, "/v1/memorials"},
		{http.MethodGet, "/v1/memorials/1"},
		{http.MethodPut, "/v1/memorials/1"},
		{http.MethodDelete, "/v1/memorials/1"},
	}

	t.Run("AC2: every route is behind the access-token middleware", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		pair := api.login(phone, deviceA)

		for _, rt := range routes {
			var body any
			if rt.method == http.MethodPost || rt.method == http.MethodPut {
				body = memorialBody("x")
			}

			none := api.do(rt.method, rt.path, body, "")
			assert.Equal(t, http.StatusUnauthorized, none.status, "%s %s without a token: %s", rt.method, rt.path, none.raw)
			assert.Equal(t, "UNAUTHORIZED", none.errorCode())

			garbage := api.do(rt.method, rt.path, body, "not-a-jwt")
			assert.Equal(t, http.StatusUnauthorized, garbage.status, "%s %s with garbage", rt.method, rt.path)

			// A refresh token is a real, signed token but the wrong type.
			wrongType := api.do(rt.method, rt.path, body, pair.RefreshToken)
			assert.Equal(t, http.StatusUnauthorized, wrongType.status, "%s %s with a refresh token", rt.method, rt.path)
		}
	})

	t.Run("AC1/AC13-AC15/AC17/AC18/AC22: create, list, get, update, delete over real HTTP", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		tok := api.login(phone, deviceA).AccessToken

		// A full solar date with a reminder.
		full := memorialBody("Nguyễn Văn A")
		full["death_calendar"] = "lunar" // typed as lunar; the client already converted to solar
		full["note"] = "nhớ ơn"
		full["reminder"] = map[string]any{"days_before": 3, "time": "07:30"}
		full["add_to_os_reminder"] = true
		res := api.do(http.MethodPost, "/v1/memorials", full, tok)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var first memorialData
		res.data(&first)
		assert.NotZero(t, first.ID)
		require.NotNil(t, first.DeathDate)
		assert.Equal(t, "2015-03-04", *first.DeathDate)
		assert.Equal(t, "lunar", first.DeathCalendar)
		assert.Nil(t, first.DeathDay)
		assert.Nil(t, first.PhotoURL, "photo is deferred (TD-001): always null")
		assert.Equal(t, map[string]any{"days_before": float64(3), "time": "07:30", "timezone": "Asia/Ho_Chi_Minh"}, first.Reminder)
		assert.True(t, first.AddToOS)

		// A year-unknown lunar date in a leap month.
		res = api.do(http.MethodPost, "/v1/memorials", map[string]any{
			"full_name": "Bà ngoại", "relationship": "GRANDMOTHER_MATERNAL", "death_calendar": "lunar",
			"death_day": 15, "death_month": 7, "death_leap_month": true,
		}, tok)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var second memorialData
		res.data(&second)
		assert.Nil(t, second.DeathDate)
		assert.Equal(t, 15, *second.DeathDay)
		assert.True(t, *second.DeathLeapMonth)
		assert.Nil(t, second.Note)
		assert.Nil(t, second.Reminder)

		// AC13: newest first.
		res = api.do(http.MethodGet, "/v1/memorials", nil, tok)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var list struct {
			Items []memorialData `json:"items"`
			Total int            `json:"total"`
		}
		res.data(&list)
		require.Len(t, list.Items, 2)
		assert.Equal(t, 2, list.Total)
		assert.Equal(t, second.ID, list.Items[0].ID)
		assert.Equal(t, first.ID, list.Items[1].ID)

		// AC14
		res = api.do(http.MethodGet, fmt.Sprintf("/v1/memorials/%d", first.ID), nil, tok)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var got memorialData
		res.data(&got)
		assert.Equal(t, "Nguyễn Văn A", got.FullName)

		// AC18: PUT replaces; it can switch the date form and drop the reminder.
		res = api.do(http.MethodPut, fmt.Sprintf("/v1/memorials/%d", first.ID), map[string]any{
			"full_name": "Nguyễn Văn B", "relationship": "SPOUSE", "death_calendar": "solar", "death_day": 4, "death_month": 3,
		}, tok)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var updated memorialData
		res.data(&updated)
		assert.Equal(t, first.ID, updated.ID)
		assert.Equal(t, "Nguyễn Văn B", updated.FullName)
		assert.Nil(t, updated.DeathDate)
		assert.Equal(t, 4, *updated.DeathDay)
		assert.Nil(t, updated.Note, "replacement: an omitted note is cleared")
		assert.Nil(t, updated.Reminder)
		assert.False(t, updated.AddToOS)

		// AC22 + AC24: delete, then it is gone and a second delete is 404.
		res = api.do(http.MethodDelete, fmt.Sprintf("/v1/memorials/%d", first.ID), nil, tok)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.True(t, res.envelope.Success)
		assert.Empty(t, res.envelope.Data, "AC22: no data on delete")

		res = api.do(http.MethodGet, fmt.Sprintf("/v1/memorials/%d", first.ID), nil, tok)
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "MEMORIAL_NOT_FOUND", res.errorCode())
		res = api.do(http.MethodDelete, fmt.Sprintf("/v1/memorials/%d", first.ID), nil, tok)
		assert.Equal(t, http.StatusNotFound, res.status)
	})

	t.Run("AC15/AC21/AC24: another user cannot see, change or delete it, and learns nothing", func(t *testing.T) {
		api.t = t
		owner := api.login(api.registerUser(), deviceA).AccessToken
		intruder := api.login(api.registerUser(), deviceA).AccessToken

		res := api.do(http.MethodPost, "/v1/memorials", memorialBody("Của chủ"), owner)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var mine memorialData
		res.data(&mine)
		path := fmt.Sprintf("/v1/memorials/%d", mine.ID)

		missing := api.do(http.MethodGet, "/v1/memorials/999999999", nil, intruder)
		for _, rt := range []struct {
			method string
			body   any
		}{{http.MethodGet, nil}, {http.MethodPut, memorialBody("Hijacked")}, {http.MethodDelete, nil}} {
			res := api.do(rt.method, path, rt.body, intruder)
			assert.Equal(t, http.StatusNotFound, res.status, "%s someone else's: %s", rt.method, res.raw)
			assert.Equal(t, "MEMORIAL_NOT_FOUND", res.errorCode())
			assert.Equal(t, missing.raw, res.raw, "%s: byte-identical to an id that does not exist", rt.method)
		}

		res = api.do(http.MethodGet, "/v1/memorials", nil, intruder)
		var list struct {
			Items []memorialData `json:"items"`
		}
		res.data(&list)
		assert.Empty(t, list.Items, "the intruder's list holds none of the owner's rows")

		// The owner's row is exactly as it was.
		res = api.do(http.MethodGet, path, nil, owner)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var still memorialData
		res.data(&still)
		assert.Equal(t, "Của chủ", still.FullName)
	})

	t.Run("AC3-AC12: validation names the field; unknown and server-owned fields are rejected", func(t *testing.T) {
		api.t = t
		tok := api.login(api.registerUser(), deviceA).AccessToken

		with := func(k string, v any) map[string]any {
			b := memorialBody("A")
			b[k] = v
			return b
		}
		cases := []struct {
			name  string
			body  map[string]any
			field string
		}{
			{"blank name", with("full_name", "  "), "full_name"},
			{"bad relationship", with("relationship", "UNCLE"), "relationship"},
			{"bad calendar", with("death_calendar", "x"), "death_calendar"},
			{"future death_date", with("death_date", "2999-01-01"), "death_date"},
			{"reminder out of range", with("reminder", map[string]any{"days_before": 31, "time": "07:30"}), "reminder.days_before"},
			{"OS flag without reminder", with("add_to_os_reminder", true), "add_to_os_reminder"},
		}
		for _, tc := range cases {
			res := api.do(http.MethodPost, "/v1/memorials", tc.body, tok)
			assert.Equal(t, http.StatusBadRequest, res.status, "%s: %s", tc.name, res.raw)
			assert.Equal(t, "VALIDATION_ERROR", res.errorCode(), tc.name)
			assert.Equal(t, tc.field, res.details()["field"], tc.name)
		}

		for _, k := range []string{"photo_url", "id", "user_id", "created_at"} {
			res := api.do(http.MethodPost, "/v1/memorials", with(k, "x"), tok)
			assert.Equal(t, http.StatusBadRequest, res.status, "server-owned field %s must be rejected: %s", k, res.raw)
		}

		res := api.do(http.MethodGet, "/v1/memorials", nil, tok)
		var list struct {
			Items []memorialData `json:"items"`
		}
		res.data(&list)
		assert.Empty(t, list.Items, "no rejected request stored anything")
	})

	t.Run("AC11: the 101st memorial is refused with 409, per user", func(t *testing.T) {
		api.t = t
		tok := api.login(api.registerUser(), deviceA).AccessToken
		other := api.login(api.registerUser(), deviceA).AccessToken

		for i := 0; i < 100; i++ {
			res := api.do(http.MethodPost, "/v1/memorials", memorialBody(fmt.Sprint("m", i)), tok)
			require.Equal(t, http.StatusCreated, res.status, "create #%d: %s", i+1, res.raw)
		}
		res := api.do(http.MethodPost, "/v1/memorials", memorialBody("one too many"), tok)
		assert.Equal(t, http.StatusConflict, res.status, res.raw)
		assert.Equal(t, "MEMORIAL_LIMIT_REACHED", res.errorCode())

		res = api.do(http.MethodPost, "/v1/memorials", memorialBody("someone else is unaffected"), other)
		assert.Equal(t, http.StatusCreated, res.status, res.raw)
	})
}
