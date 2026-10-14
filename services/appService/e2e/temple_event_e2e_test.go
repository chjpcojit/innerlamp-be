package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tevLunar struct {
	Day         int  `json:"day"`
	Month       int  `json:"month"`
	Year        int  `json:"year"`
	IsLeapMonth bool `json:"is_leap_month"`
}

type tevRecurrence struct {
	Kind      string  `json:"kind"`
	LunarDays []int   `json:"lunar_days"`
	Weekday   *int    `json:"weekday"`
	Until     *string `json:"until"`
}

type tevEvent struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Kind         string        `json:"kind"`
	IsAllDay     bool          `json:"is_all_day"`
	Status       string        `json:"status"`
	AudienceNote string        `json:"audience_note"`
	ImageURL     *string       `json:"image_url"`
	Recurrence   tevRecurrence `json:"recurrence"`
}

type tevOccurrence struct {
	EventID   string   `json:"event_id"`
	StartsAt  string   `json:"starts_at"`
	EndsAt    string   `json:"ends_at"`
	SolarDate string   `json:"solar_date"`
	LunarDate tevLunar `json:"lunar_date"`
}

type tevData struct {
	TempleID    string          `json:"temple_id"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	Events      []tevEvent      `json:"events"`
	Occurrences []tevOccurrence `json:"occurrences"`
}

func (d tevData) on(id string) []tevOccurrence {
	var out []tevOccurrence
	for _, o := range d.Occurrences {
		if o.EventID == id {
			out = append(out, o)
		}
	}
	return out
}

func (d tevData) event(id string) (tevEvent, bool) {
	for _, e := range d.Events {
		if e.ID == id {
			return e, true
		}
	}
	return tevEvent{}, false
}

func seedTempleEvents(t *testing.T, cfgPath, file string) runResult {
	t.Helper()
	cmd := exec.Command(seederBinaryPath(t, "seedtempleevents"), "-file", file)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.Truef(t, ok, "running seedtempleevents: %v", err)
		code = exitErr.ExitCode()
	}
	return runResult{stderr: stderr.String(), exitCode: code}
}

// TestTempleEventFlow_E2E drives docs/specs/features/temple-event (US-TEV-001)
// against the real binary and a fresh Postgres, with the data loaded by the
// real seed command. The data file is generated relative to the real clock
// (the service reads "now" itself), so the test does not rot.
func TestTempleEventFlow_E2E(t *testing.T) {
	requireE2E(t)
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	require.NoError(t, err)
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	rfc := func(t time.Time) string { return t.Format(time.RFC3339) }
	dateOnly := func(t time.Time) string { return t.Format("2006-01-02") }
	clock := func(day time.Time, h, m int) time.Time {
		return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, loc)
	}

	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	db := openDB(t, host, port)
	for _, id := range []string{"temple_a", "temple_b", "temple_empty"} {
		execSQL(t, db, `INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority,
			latitude, longitude, address, province) VALUES ($1, $2, 'CHUA', 'north', 'BẮC_TÔNG', 1900,
			'CHUA_XEP_HANG', 8, 21.0, 105.8, 'Hà Nội', 'Hà Nội')`, id, "Chùa "+id)
	}

	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)
	nextWeek := today.AddDate(0, 0, 7)
	file := filepath.Join(t.TempDir(), "temple-events.json")
	data := fmt.Sprintf(`{"meta":{"format":1,"source":"e2e"},"events":[
	  {"id":"oneoff","temple_id":"temple_a","title":"Khoá tu","kind":"KHOA_TU","starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","audience_note":"thiếu nhi","recurrence":"NONE"},
	  {"id":"past","temple_id":"temple_a","title":"Đã qua","kind":"KHAC","starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","recurrence":"NONE"},
	  {"id":"cancelled","temple_id":"temple_a","title":"Bị huỷ","kind":"PHAP_HOI","starts_at":%q,"ends_at":%q,
	   "status":"CANCELLED","recurrence":"NONE"},
	  {"id":"weekly","temple_id":"temple_a","title":"Lễ hằng tuần","kind":"LE_CAU_AN","starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","recurrence":"WEEKLY","recurrence_weekday":%d,"recurrence_until":%q},
	  {"id":"lunar","temple_id":"temple_a","title":"Sám hối","kind":"KHAC","starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","recurrence":"LUNAR_MONTHLY","recurrence_lunar_days":[14,15,30]},
	  {"id":"allday","temple_id":"temple_a","title":"Cả ngày","kind":"LE_HOI","is_all_day":true,"starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","recurrence":"NONE"},
	  {"id":"off","temple_id":"temple_a","title":"Bị tắt","kind":"KHAC","starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","recurrence":"NONE"},
	  {"id":"other","temple_id":"temple_b","title":"Chùa khác","kind":"KHAC","starts_at":%q,"ends_at":%q,
	   "status":"SCHEDULED","recurrence":"NONE"}]}`,
		rfc(clock(tomorrow, 19, 0)), rfc(clock(tomorrow, 21, 0)),
		rfc(clock(yesterday, 8, 0)), rfc(clock(yesterday, 9, 0)),
		rfc(clock(tomorrow, 8, 0)), rfc(clock(tomorrow, 9, 0)),
		rfc(clock(today.AddDate(0, 0, -21), 8, 0)), rfc(clock(today.AddDate(0, 0, -21), 10, 0)),
		int(today.AddDate(0, 0, -21).Weekday()), dateOnly(today.AddDate(0, 0, 40)),
		rfc(clock(today, 19, 0)), rfc(clock(today, 21, 0)),
		rfc(clock(nextWeek, 0, 0)), rfc(clock(nextWeek.AddDate(0, 0, 1), 0, 0)),
		rfc(clock(tomorrow, 10, 0)), rfc(clock(tomorrow, 11, 0)),
		rfc(clock(tomorrow, 12, 0)), rfc(clock(tomorrow, 13, 0)))
	require.NoError(t, os.WriteFile(file, []byte(data), 0o600))

	cfgPath := writeTempConfig(t, host, port, "0", nil)
	seed := seedTempleEvents(t, cfgPath, file)
	require.Equal(t, 0, seed.exitCode, "seed failed:\n%s", seed.stderr)
	execSQL(t, db, `UPDATE temple_events SET is_active = false WHERE id = 'off'`)

	baseURL, stop := startServer(t, host, port, nil)
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	token := api.login(api.registerUser(), deviceA).AccessToken

	get := func(id, query string) (*apiResponse, tevData) {
		path := "/v1/temples/" + id + "/events"
		if query != "" {
			path += "?" + query
		}
		res := api.do(http.MethodGet, path, nil, token)
		var d tevData
		if res.status == http.StatusOK {
			res.data(&d)
		}
		return res, d
	}

	t.Run("TE2: the route is behind the access-token middleware", func(t *testing.T) {
		api.t = t
		pair := api.login(api.registerUser(), deviceA)
		none := api.do(http.MethodGet, "/v1/temples/temple_a/events", nil, "")
		assert.Equal(t, http.StatusUnauthorized, none.status, none.raw)
		assert.NotContains(t, none.raw, "occurrences")
		refresh := api.do(http.MethodGet, "/v1/temples/temple_a/events", nil, pair.RefreshToken)
		assert.Equal(t, http.StatusUnauthorized, refresh.status, "a refresh token is not an access token")
	})

	t.Run("TE3/TE4: unknown temple is 404; temple with no event is 200 and empty", func(t *testing.T) {
		api.t = t
		res, _ := get("temple_nope", "")
		assert.Equal(t, http.StatusNotFound, res.status, res.raw)
		assert.Equal(t, "TEMPLE_NOT_FOUND", res.errorCode())

		res, d := get("temple_empty", "")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Contains(t, res.raw, `"events":[]`)
		assert.Contains(t, res.raw, `"occurrences":[]`)
		assert.Equal(t, "temple_empty", d.TempleID)
	})

	t.Run("TE1/TE5-TE8/TE14-TE17: default window, own temple only, past and inactive hidden, status kept", func(t *testing.T) {
		api.t = t
		res, d := get("temple_a", "")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "no-cache", res.header.Get("Cache-Control"))
		assert.Equal(t, dateOnly(today), d.From)
		assert.Equal(t, dateOnly(today.AddDate(0, 3, 0)), d.To)

		one := d.on("oneoff")
		require.Len(t, one, 1)
		assert.Equal(t, dateOnly(tomorrow), one[0].SolarDate)
		assert.Contains(t, one[0].StartsAt, "T19:00:00+07:00")
		assert.NotZero(t, one[0].LunarDate.Day, "lunar date of the start date is present")

		for _, hidden := range []string{"past", "off", "other"} {
			assert.Empty(t, d.on(hidden), hidden)
			_, ok := d.event(hidden)
			assert.False(t, ok, hidden)
		}
		cancelled, ok := d.event("cancelled")
		require.True(t, ok)
		assert.Equal(t, "CANCELLED", cancelled.Status)
		assert.Len(t, d.on("cancelled"), 1)

		oneEvt, _ := d.event("oneoff")
		assert.Equal(t, "thiếu nhi", oneEvt.AudienceNote)
		assert.Nil(t, oneEvt.ImageURL)
		assert.Equal(t, "NONE", oneEvt.Recurrence.Kind)
		assert.Equal(t, []int{}, oneEvt.Recurrence.LunarDays)
	})

	t.Run("TE10/TE13: weekly occurrences stop at recurrence_until", func(t *testing.T) {
		api.t = t
		res, d := get("temple_a", "")
		require.Equal(t, http.StatusOK, res.status)
		occ := d.on("weekly")
		require.NotEmpty(t, occ)
		last := occ[len(occ)-1].SolarDate
		assert.LessOrEqual(t, last, dateOnly(today.AddDate(0, 0, 40)))
		for _, o := range occ {
			day, err := time.Parse("2006-01-02", o.SolarDate)
			require.NoError(t, err)
			assert.Equal(t, today.AddDate(0, 0, -21).Weekday(), day.Weekday())
			assert.Contains(t, o.StartsAt, "T08:00:00+07:00")
		}
		ev, _ := d.event("weekly")
		require.NotNil(t, ev.Recurrence.Weekday)
		require.NotNil(t, ev.Recurrence.Until)
	})

	t.Run("TE11/TE12: lunar-monthly gives lunar days 14, 15 and the last day, each at most once a month", func(t *testing.T) {
		api.t = t
		res, d := get("temple_a", "")
		require.Equal(t, http.StatusOK, res.status)
		occ := d.on("lunar")
		require.NotEmpty(t, occ)
		perMonth := map[string]int{}
		for _, o := range occ {
			assert.Contains(t, []int{14, 15, 29, 30}, o.LunarDate.Day, o.SolarDate)
			perMonth[fmt.Sprintf("%d-%d-%v-%d", o.LunarDate.Year, o.LunarDate.Month, o.LunarDate.IsLeapMonth, o.LunarDate.Day)]++
		}
		for k, n := range perMonth {
			assert.Equal(t, 1, n, "duplicate occurrence %s", k)
		}
	})

	t.Run("TE8: an all-day event covers its last day", func(t *testing.T) {
		api.t = t
		q := "from=" + dateOnly(nextWeek.AddDate(0, 0, 1)) + "&to=" + dateOnly(nextWeek.AddDate(0, 0, 1))
		res, d := get("temple_a", q)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		occ := d.on("allday")
		require.Len(t, occ, 1)
		assert.Equal(t, dateOnly(nextWeek), occ[0].SolarDate)
		assert.Contains(t, occ[0].EndsAt, "T23:59:59+07:00")
		ev, _ := d.event("allday")
		assert.True(t, ev.IsAllDay)
	})

	t.Run("TE21/TE22: occurrences are sorted and stable", func(t *testing.T) {
		api.t = t
		_, a := get("temple_a", "")
		_, b := get("temple_a", "")
		assert.Equal(t, a, b)
		for i := 1; i < len(a.Occurrences); i++ {
			assert.LessOrEqual(t, a.Occurrences[i-1].StartsAt[:10], a.Occurrences[i].StartsAt[:10])
		}
	})

	t.Run("TE20: a window that ended before today is empty", func(t *testing.T) {
		api.t = t
		q := "from=" + dateOnly(today.AddDate(0, 0, -30)) + "&to=" + dateOnly(today.AddDate(0, 0, -20))
		res, d := get("temple_a", q)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Empty(t, d.Occurrences)
	})

	t.Run("TE18/TE19: bad windows are 400 with the right code", func(t *testing.T) {
		api.t = t
		res, _ := get("temple_a", "from=2026-02-30")
		assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "INVALID_DATE", res.errorCode())
		assert.Contains(t, res.raw, `"field":"from"`)

		res, _ = get("temple_a", "to=nope")
		assert.Equal(t, "INVALID_DATE", res.errorCode())
		assert.Contains(t, res.raw, `"field":"to"`)

		res, _ = get("temple_a", "from=2026-11-10&to=2026-11-01")
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "INVALID_DATE_RANGE", res.errorCode())

		res, _ = get("temple_a", "from=2026-11-01&to=2027-06-01")
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "DATE_RANGE_TOO_LARGE", res.errorCode())
	})

	t.Run("TE24: an invalid file writes nothing", func(t *testing.T) {
		api.t = t
		before := countRowsWhere(t, db, "temple_events", "TRUE")
		bad := filepath.Join(t.TempDir(), "bad.json")
		require.NoError(t, os.WriteFile(bad, []byte(fmt.Sprintf(`{"meta":{"format":1,"source":"e2e"},"events":[
		  {"id":"x","temple_id":"temple_a","title":"Thiếu thứ","kind":"KHAC","starts_at":%q,"ends_at":%q,
		   "status":"SCHEDULED","recurrence":"WEEKLY"}]}`, rfc(clock(tomorrow, 8, 0)), rfc(clock(tomorrow, 9, 0)))), 0o600))
		res := seedTempleEvents(t, cfgPath, bad)
		assert.NotEqual(t, 0, res.exitCode)
		assert.Contains(t, res.stderr, `"x"`)

		ghost := filepath.Join(t.TempDir(), "ghost.json")
		require.NoError(t, os.WriteFile(ghost, []byte(fmt.Sprintf(`{"meta":{"format":1,"source":"e2e"},"events":[
		  {"id":"g","temple_id":"temple_ghost","title":"Chùa lạ","kind":"KHAC","starts_at":%q,"ends_at":%q,
		   "status":"SCHEDULED","recurrence":"NONE"}]}`, rfc(clock(tomorrow, 8, 0)), rfc(clock(tomorrow, 9, 0)))), 0o600))
		res = seedTempleEvents(t, cfgPath, ghost)
		assert.NotEqual(t, 0, res.exitCode)
		assert.Equal(t, before, countRowsWhere(t, db, "temple_events", "TRUE"))
	})
}
