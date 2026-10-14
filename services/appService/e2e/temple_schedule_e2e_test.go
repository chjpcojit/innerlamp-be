package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type schedStep struct {
	Position int     `json:"position"`
	Time     *string `json:"time"`
	Text     string  `json:"text"`
}

type schedItem struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	StartTime   string      `json:"start_time"`
	EndTime     *string     `json:"end_time"`
	Weekdays    []int       `json:"weekdays"`
	Description string      `json:"description"`
	Instructor  string      `json:"instructor"`
	Location    string      `json:"location"`
	Preparation string      `json:"preparation"`
	Steps       []schedStep `json:"steps"`
}

type schedData struct {
	TempleID string      `json:"temple_id"`
	Items    []schedItem `json:"items"`
}

// templeScheduleFile has three items for temple_a (given out of start order, one
// crossing midnight, one with every optional field absent) and one for temple_b.
const templeScheduleFile = `{
  "meta": {"format": 1, "source": "e2e"},
  "items": [
    {"id": "late", "temple_id": "temple_a", "name": "Khoá lễ đêm", "start_time": "23:00", "end_time": "01:00",
     "weekdays": [6, 0, 6], "description": "d", "instructor": "Sư A", "location": "Chánh điện", "preparation": "Áo tràng",
     "sort_order": 1,
     "steps": [{"position": 3, "time": "00:15", "text": "Hồi hướng"}, {"position": 1, "time": "23:00", "text": "Niệm Phật"},
               {"position": 2, "time": "23:45", "text": "Tụng kinh"}, {"position": 4, "time": null, "text": "Thiền"}]},
    {"id": "early", "temple_id": "temple_a", "name": "Công phu sáng", "start_time": "04:00", "weekdays": [0,1,2,3,4,5,6]},
    {"id": "off", "temple_id": "temple_a", "name": "Buổi sẽ bị tắt", "start_time": "09:00"},
    {"id": "other", "temple_id": "temple_b", "name": "Chùa khác", "start_time": "05:00"}
  ]
}`

func seedTempleSchedules(t *testing.T, cfgPath, file string) runResult {
	t.Helper()
	cmd := exec.Command(seederBinaryPath(t, "seedtempleschedules"), "-file", file)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.Truef(t, ok, "running seedtempleschedules: %v", err)
		code = exitErr.ExitCode()
	}
	return runResult{stderr: stderr.String(), exitCode: code}
}

// TestTempleScheduleFlow_E2E drives docs/specs/features/temple-schedule
// (US-TSC-001) against the real binary and a fresh Postgres, with the data
// loaded by the real seed command.
//
// What is here and not in the per-layer tests: that cmd/main.go puts the route
// behind the access-token middleware (unlike the public /v1/temples routes next
// to it), and that the seed command and the service agree on the schema.
func TestTempleScheduleFlow_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	db := openDB(t, host, port)

	const cols = `(id, name, type, region, sect, established_year, heritage_rank, heritage_priority,
		latitude, longitude, address, province)`
	for _, id := range []string{"temple_a", "temple_b", "temple_empty"} {
		execSQL(t, db, `INSERT INTO temples `+cols+` VALUES ($1, $2, 'CHUA', 'north', 'BẮC_TÔNG', 1900,
			'CHUA_XEP_HANG', 8, 21.0, 105.8, 'Hà Nội', 'Hà Nội')`, id, "Chùa "+id)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "temple-schedules.json")
	require.NoError(t, os.WriteFile(file, []byte(templeScheduleFile), 0o600))

	cfgPath := writeTempConfig(t, host, port, "0", nil)
	seed := seedTempleSchedules(t, cfgPath, file)
	require.Equal(t, 0, seed.exitCode, "seed failed:\n%s", seed.stderr)

	baseURL, stop := startServer(t, host, port, nil)
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	token := api.login(api.registerUser(), deviceA).AccessToken

	get := func(id string) (*apiResponse, schedData) {
		res := api.do(http.MethodGet, "/v1/temples/"+id+"/schedule", nil, token)
		var d schedData
		if res.status == http.StatusOK {
			res.data(&d)
		}
		return res, d
	}

	t.Run("TS2: the route is behind the access-token middleware", func(t *testing.T) {
		api.t = t
		pair := api.login(api.registerUser(), deviceA)

		none := api.do(http.MethodGet, "/v1/temples/temple_a/schedule", nil, "")
		assert.Equal(t, http.StatusUnauthorized, none.status, none.raw)
		assert.NotContains(t, none.raw, "items")

		refresh := api.do(http.MethodGet, "/v1/temples/temple_a/schedule", nil, pair.RefreshToken)
		assert.Equal(t, http.StatusUnauthorized, refresh.status, "a refresh token is not an access token")

		// The neighbouring temple detail route stays public.
		detail := api.do(http.MethodGet, "/v1/temples/temple_a", nil, "")
		assert.Equal(t, http.StatusOK, detail.status, detail.raw)
	})

	t.Run("TS3: unknown temple is 404", func(t *testing.T) {
		api.t = t
		res, _ := get("temple_nope")
		assert.Equal(t, http.StatusNotFound, res.status, res.raw)
		assert.Equal(t, "TEMPLE_NOT_FOUND", res.errorCode())
	})

	t.Run("TS4: temple with no schedule is 200 with empty items", func(t *testing.T) {
		api.t = t
		res, d := get("temple_empty")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "temple_empty", d.TempleID)
		assert.Contains(t, res.raw, `"items":[]`)
	})

	t.Run("TS5-TS9/TS12: items ordered, only this temple, normalised, steps by position", func(t *testing.T) {
		api.t = t
		res, d := get("temple_a")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		require.Len(t, d.Items, 3, res.raw)
		assert.Equal(t, []string{"early", "off", "late"}, []string{d.Items[0].ID, d.Items[1].ID, d.Items[2].ID})

		early := d.Items[0]
		assert.Equal(t, "04:00", early.StartTime)
		assert.Nil(t, early.EndTime)
		assert.Equal(t, []int{}, early.Weekdays, "all seven days collapse to every day")
		assert.Equal(t, "", early.Instructor)
		assert.Equal(t, []schedStep{}, early.Steps)
		assert.Contains(t, res.raw, `"end_time":null`)

		late := d.Items[2]
		assert.Equal(t, "23:00", late.StartTime)
		require.NotNil(t, late.EndTime)
		assert.Equal(t, "01:00", *late.EndTime)
		assert.Equal(t, []int{0, 6}, late.Weekdays)
		assert.Equal(t, "Sư A", late.Instructor)
		assert.Equal(t, "Chánh điện", late.Location)
		assert.Equal(t, "Áo tràng", late.Preparation)
		require.Len(t, late.Steps, 4)
		for i, s := range late.Steps {
			assert.Equal(t, i+1, s.Position)
		}
		assert.Equal(t, "Hồi hướng", late.Steps[2].Text) // 00:15 stays third: position, not time
		assert.Nil(t, late.Steps[3].Time)
	})

	t.Run("TS11/TS16/TS18: switched-off item is hidden and stays off after a re-seed; steps are replaced", func(t *testing.T) {
		api.t = t
		execSQL(t, db, `UPDATE temple_schedules SET is_active = false WHERE id = 'off'`)
		res, d := get("temple_a")
		require.Equal(t, http.StatusOK, res.status)
		require.Len(t, d.Items, 2)
		assert.Equal(t, []string{"early", "late"}, []string{d.Items[0].ID, d.Items[1].ID})

		const replaced = `{"meta":{"format":1,"source":"e2e"},"items":[
		  {"id":"late","temple_id":"temple_a","name":"Khoá lễ đêm","start_time":"23:00","end_time":"01:00",
		   "steps":[{"position":1,"text":"Chỉ còn một bước"}]},
		  {"id":"off","temple_id":"temple_a","name":"Buổi sẽ bị tắt","start_time":"09:00"}]}`
		require.NoError(t, os.WriteFile(file, []byte(replaced), 0o600))
		seed := seedTempleSchedules(t, cfgPath, file)
		require.Equal(t, 0, seed.exitCode, "re-seed failed:\n%s", seed.stderr)

		_, d = get("temple_a")
		require.Len(t, d.Items, 2, "'off' stays off after the re-seed")
		var late schedItem
		for _, it := range d.Items {
			if it.ID == "late" {
				late = it
			}
		}
		require.Len(t, late.Steps, 1)
		assert.Equal(t, "Chỉ còn một bước", late.Steps[0].Text)
		assert.Equal(t, []int{}, late.Weekdays, "weekdays absent in the file overwrite the old value")
	})

	t.Run("TS17: an invalid file writes nothing", func(t *testing.T) {
		api.t = t
		bad := filepath.Join(dir, "bad.json")
		require.NoError(t, os.WriteFile(bad, []byte(`{"meta":{"format":1,"source":"e2e"},"items":[
		  {"id":"x","temple_id":"temple_a","name":"Hỏng","start_time":"04:00","end_time":"04:00"}]}`), 0o600))
		res := seedTempleSchedules(t, cfgPath, bad)
		assert.NotEqual(t, 0, res.exitCode)
		assert.Contains(t, res.stderr, `"x"`)

		ghost := filepath.Join(dir, "ghost.json")
		require.NoError(t, os.WriteFile(ghost, []byte(`{"meta":{"format":1,"source":"e2e"},"items":[
		  {"id":"g","temple_id":"temple_ghost","name":"Chùa không có","start_time":"04:00"}]}`), 0o600))
		res = seedTempleSchedules(t, cfgPath, ghost)
		assert.NotEqual(t, 0, res.exitCode)
		assert.Equal(t, 1, countRowsWhere(t, db, "temple_schedules", "id = 'other'"))
		assert.Equal(t, 0, countRowsWhere(t, db, "temple_schedules", "id IN ('x', 'g')"))
	})
}

func countRowsWhere(t *testing.T, db interface {
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
}, table, where string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(context.Background(), "SELECT count(*) FROM "+table+" WHERE "+where).Scan(&n))
	return n
}
