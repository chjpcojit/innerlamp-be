package e2e

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bytes"
	"encoding/json"
)

// calendarEventsDataPath is the catalogue the CLI loads, relative to this package.
const calendarEventsDataPath = "../../../assets/data/calendar-events.json"

type calGroup struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type calEvent struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	ShortDescription string   `json:"short_description"`
	LongDescription  string   `json:"long_description"`
	Guidance         string   `json:"guidance"`
	ImageURL         *string  `json:"image_url"`
	Groups           []string `json:"groups"`
	IsPublicHoliday  bool     `json:"is_public_holiday"`
}

type calLunar struct {
	Day         int  `json:"day"`
	Month       int  `json:"month"`
	Year        int  `json:"year"`
	IsLeapMonth bool `json:"is_leap_month"`
}

type calOccurrence struct {
	EventCode string   `json:"event_code"`
	SolarDate string   `json:"solar_date"`
	LunarDate calLunar `json:"lunar_date"`
}

type calEventsData struct {
	Year        int             `json:"year"`
	Groups      []calGroup      `json:"groups"`
	Events      []calEvent      `json:"events"`
	Occurrences []calOccurrence `json:"occurrences"`
}

func (d calEventsData) on(code string) []calOccurrence {
	var out []calOccurrence
	for _, o := range d.Occurrences {
		if o.EventCode == code {
			out = append(out, o)
		}
	}
	return out
}

func (d calEventsData) event(code string) (calEvent, bool) {
	for _, e := range d.Events {
		if e.Code == code {
			return e, true
		}
	}
	return calEvent{}, false
}

// seedCalendar runs cmd/seedcalendarevents against the database cfgPath points at
// and returns its result. file is passed as an absolute path.
func seedCalendar(t *testing.T, cfgPath, file string, extra ...string) runResult {
	t.Helper()
	abs, err := filepath.Abs(file)
	require.NoError(t, err)
	cmd := exec.Command(seederBinaryPath(t, "seedcalendarevents"), append([]string{"-file", abs}, extra...)...)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.Truef(t, ok, "running seedcalendarevents: %v", err)
		code = exitErr.ExitCode()
	}
	return runResult{stderr: stderr.String(), exitCode: code}
}

// TestCalendarEventsFlow_E2E drives docs/specs/features/calendar (US-CAL-002)
// against the real binary and a fresh Postgres, with the catalogue loaded by the
// real seed command.
//
// What is here and not in the per-layer tests: that cmd/main.go puts the route
// behind the access-token middleware, that the seed command and the service agree
// on the schema, and that the dates the API computes for a real year are the ones
// the published calendar gives.
func TestCalendarEventsFlow_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)

	cfgPath := writeTempConfig(t, host, port, "0", nil)
	seed := seedCalendar(t, cfgPath, calendarEventsDataPath)
	require.Equal(t, 0, seed.exitCode, "seed failed:\n%s", seed.stderr)

	baseURL, stop := startServer(t, host, port, nil)
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	db := openDB(t, host, port)

	token := api.login(api.registerUser(), deviceA).AccessToken
	get := func(year string) (*apiResponse, calEventsData) {
		api.t = t
		path := "/v1/calendar/events"
		if year != "" {
			path += "?year=" + year
		}
		res := api.do(http.MethodGet, path, nil, token)
		var d calEventsData
		if res.status == http.StatusOK {
			res.data(&d)
		}
		return res, d
	}

	t.Run("E4: the route is behind the access-token middleware", func(t *testing.T) {
		api.t = t
		pair := api.login(api.registerUser(), deviceA)

		none := api.do(http.MethodGet, "/v1/calendar/events?year=2026", nil, "")
		assert.Equal(t, http.StatusUnauthorized, none.status, none.raw)
		assert.Equal(t, "UNAUTHORIZED", none.errorCode())
		assert.NotContains(t, none.raw, "occurrences")

		garbage := api.do(http.MethodGet, "/v1/calendar/events?year=2026", nil, "not-a-jwt")
		assert.Equal(t, http.StatusUnauthorized, garbage.status)

		refresh := api.do(http.MethodGet, "/v1/calendar/events?year=2026", nil, pair.RefreshToken)
		assert.Equal(t, http.StatusUnauthorized, refresh.status, "a refresh token is not an access token")
	})

	t.Run("E1/E5/E13/E14: a year returns groups, events and occurrences with both dates", func(t *testing.T) {
		res, d := get("2026")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, 2026, d.Year)

		groups := map[string]string{}
		for _, g := range d.Groups {
			groups[g.Code] = g.Name
		}
		assert.Equal(t, map[string]string{"buddhist": "Phật giáo", "vietnam": "Việt Nam", "folk": "Dân gian"}, groups)

		tet, ok := d.event("tet-nguyen-dan")
		require.True(t, ok, "Tết is in the response")
		assert.Equal(t, "Tết Nguyên Đán", tet.Name)
		assert.NotEmpty(t, tet.ShortDescription)
		assert.NotEmpty(t, tet.LongDescription)
		assert.NotEmpty(t, tet.Guidance)
		assert.Nil(t, tet.ImageURL)
		assert.True(t, tet.IsPublicHoliday)
		assert.ElementsMatch(t, []string{"vietnam", "folk"}, tet.Groups)
		assert.Contains(t, res.raw, `"image_url":null`)

		occ := d.on("tet-nguyen-dan")
		require.Len(t, occ, 1)
		assert.Equal(t, "2026-02-17", occ[0].SolarDate)
		assert.Equal(t, calLunar{Day: 1, Month: 1, Year: 2026}, occ[0].LunarDate)

		for _, o := range d.Occurrences {
			_, known := d.event(o.EventCode)
			assert.True(t, known, "occurrence of %q has no entry in events", o.EventCode)
			assert.True(t, strings.HasPrefix(o.SolarDate, "2026-"), "%s %s is outside 2026", o.EventCode, o.SolarDate)
			assert.NotZero(t, o.LunarDate.Day)
		}
	})

	t.Run("E6/E10: solar date and solar term", func(t *testing.T) {
		_, d := get("2026")
		lib := d.on("giai-phong-mien-nam")
		require.Len(t, lib, 1)
		assert.Equal(t, "2026-04-30", lib[0].SolarDate)
		e, _ := d.event("giai-phong-mien-nam")
		assert.True(t, e.IsPublicHoliday)

		tm := d.on("tiet-thanh-minh")
		require.Len(t, tm, 1)
		assert.Equal(t, "2026-04-05", tm[0].SolarDate)
		xp := d.on("tiet-xuan-phan")
		require.Len(t, xp, 1)
		assert.Equal(t, "2026-03-20", xp[0].SolarDate, "0° is a valid longitude")
	})

	t.Run("E7: leap month 2025 gives the extra Rằm (E8) and Táo quân lands in January 2025 (E11)", func(t *testing.T) {
		_, d := get("2025")
		ram := d.on("ram-am-lich")
		assert.GreaterOrEqual(t, len(ram), 12)
		leap := 0
		for _, o := range ram {
			assert.Equal(t, 15, o.LunarDate.Day)
			if o.LunarDate.IsLeapMonth {
				leap++
				assert.Equal(t, 6, o.LunarDate.Month)
			}
		}
		assert.Equal(t, 1, leap, "2025 has a leap month 6")

		tao := d.on("ong-cong-ong-tao")
		require.Len(t, tao, 1)
		assert.True(t, strings.HasPrefix(tao[0].SolarDate, "2025-01"), tao[0].SolarDate)
		assert.Equal(t, 2024, tao[0].LunarDate.Year, "the lunar year is the previous one")
		assert.Equal(t, 12, tao[0].LunarDate.Month)
		assert.Equal(t, 23, tao[0].LunarDate.Day)
	})

	t.Run("E9: Giao thừa is the day before mùng 1 Tết", func(t *testing.T) {
		for _, y := range []string{"2024", "2025", "2026"} {
			_, d := get(y)
			gt, tet := d.on("giao-thua"), d.on("tet-nguyen-dan")
			require.Len(t, gt, 1, y)
			require.Len(t, tet, 1, y)
			g, err := time.Parse("2006-01-02", gt[0].SolarDate)
			require.NoError(t, err)
			assert.Equal(t, tet[0].SolarDate, g.AddDate(0, 0, 1).Format("2006-01-02"), "year %s", y)
			assert.Contains(t, []int{29, 30}, gt[0].LunarDate.Day)
			assert.Equal(t, 12, gt[0].LunarDate.Month)
		}
	})

	t.Run("E12/E21: occurrences are sorted and a year answers the same twice", func(t *testing.T) {
		first, d := get("2026")
		for i := 1; i < len(d.Occurrences); i++ {
			assert.LessOrEqual(t, d.Occurrences[i-1].SolarDate, d.Occurrences[i].SolarDate)
		}
		second, _ := get("2026")
		assert.Equal(t, first.raw, second.raw)
	})

	t.Run("E3: a bad year is a 400 INVALID_YEAR", func(t *testing.T) {
		for _, y := range []string{"abc", "2026.5", "1899", "2101", "-1", "+2026"} {
			res, _ := get(y)
			assert.Equal(t, http.StatusBadRequest, res.status, "year=%q: %s", y, res.raw)
			assert.Equal(t, "INVALID_YEAR", res.errorCode(), "year=%q", y)
			assert.Equal(t, "year", res.details()["field"])
			assert.NotContains(t, res.raw, "occurrences")
		}
	})

	t.Run("E2/E17: omitted year is this year and not cached; explicit year is cached privately", func(t *testing.T) {
		omitted, d := get("")
		require.Equal(t, http.StatusOK, omitted.status, omitted.raw)
		assert.Equal(t, time.Now().In(time.FixedZone("ICT", 7*3600)).Year(), d.Year)
		assert.Equal(t, "no-cache", omitted.header.Get("Cache-Control"))

		explicit, _ := get("2027")
		assert.Equal(t, "private, max-age=86400", explicit.header.Get("Cache-Control"))
	})

	t.Run("E16: an inactive event is not returned, and re-seeding does not switch it back on", func(t *testing.T) {
		execSQL(t, db, `UPDATE calendar_events SET is_active = false WHERE code = 'quoc-te-lao-dong'`)
		_, d := get("2026")
		_, present := d.event("quoc-te-lao-dong")
		assert.False(t, present)
		assert.Empty(t, d.on("quoc-te-lao-dong"))

		again := seedCalendar(t, cfgPath, calendarEventsDataPath)
		require.Equal(t, 0, again.exitCode, again.stderr)
		_, d = get("2026")
		_, present = d.event("quoc-te-lao-dong")
		assert.False(t, present, "the seed must not undo back-office")
	})

	t.Run("E18: the seed is idempotent and refuses an invalid file without writing", func(t *testing.T) {
		events := countRows(t, db, "calendar_events")
		links := countRows(t, db, "calendar_event_groups")
		require.Greater(t, events, 40)

		again := seedCalendar(t, cfgPath, calendarEventsDataPath)
		require.Equal(t, 0, again.exitCode, again.stderr)
		assert.Equal(t, events, countRows(t, db, "calendar_events"))
		assert.Equal(t, links, countRows(t, db, "calendar_event_groups"))

		// A file with a duplicate code and an unknown group must be refused whole.
		raw, err := os.ReadFile(calendarEventsDataPath)
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		evs := doc["events"].([]any)
		dup := map[string]any{}
		for k, v := range evs[0].(map[string]any) {
			dup[k] = v
		}
		dup["groups"] = []any{"no-such-group"}
		doc["events"] = append(evs, dup)
		bad, err := json.Marshal(doc)
		require.NoError(t, err)
		badPath := filepath.Join(t.TempDir(), "bad.json")
		require.NoError(t, os.WriteFile(badPath, bad, 0o600))

		res := seedCalendar(t, cfgPath, badPath)
		assert.Equal(t, 1, res.exitCode, res.stderr)
		assert.Contains(t, res.stderr, "duplicate code")
		assert.Contains(t, res.stderr, "does not exist")
		assert.Equal(t, events, countRows(t, db, "calendar_events"))

		dry := seedCalendar(t, cfgPath, calendarEventsDataPath, "-dry-run")
		assert.Equal(t, 0, dry.exitCode, dry.stderr)
	})
}
