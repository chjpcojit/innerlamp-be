package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// adminUnitsDataPath is the dataset the CLI loads, relative to this package. The
// file is untracked by default (.gitignore: *.json), so the tests skip without it.
const adminUnitsDataPath = "../../../assets/data/administrative-units.json"

var (
	seedAdminUnitsOnce sync.Once
	seedAdminUnitsPath string
	seedAdminUnitsErr  error
)

// seedAdminUnitsBinaryPath builds cmd/seedadminunits once, like binaryPath does for
// the service, so the data reaches the database the way an operator loads it.
func seedAdminUnitsBinaryPath(t *testing.T) string {
	t.Helper()
	dir := sharedBuildDir(t)
	seedAdminUnitsOnce.Do(func() {
		seedAdminUnitsPath = filepath.Join(dir, "seedadminunits")
		seedAdminUnitsErr = goBuild(filepath.Join(cmdDir, "seedadminunits"), seedAdminUnitsPath)
	})
	require.NoError(t, seedAdminUnitsErr)
	return seedAdminUnitsPath
}

// runCLI runs the seeder with args and waits for it to exit.
func runCLI(t *testing.T, cfgPath string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(seedAdminUnitsBinaryPath(t), args...)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	require.NoError(t, cmd.Start())
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		code := 0
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			require.Truef(t, ok, "waiting for seedadminunits: %v", err)
			code = exitErr.ExitCode()
		}
		return runResult{stderr: stderr.String(), exitCode: code}
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		<-done
		return runResult{stderr: stderr.String(), exitCode: -1, timedOut: true}
	}
}

// devWebAPIKey is the shared web key the dev config resolves, so the tests can
// send the one a real web client would.
func devWebAPIKey(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(baseConfigPath)
	require.NoError(t, err)
	var doc struct {
		Server struct {
			WebAPIKeySecret string `yaml:"web_api_key_secret"`
		} `yaml:"server"`
		DevSecrets map[string]string `yaml:"dev_secrets"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	key := doc.DevSecrets[doc.Server.WebAPIKeySecret]
	require.NotEmpty(t, key, "dev config has no web API key")
	return key
}

// getKeyed does a GET, optionally with an X-API-Key header, and never an
// Authorization header: these routes need no account.
func (c *apiClient) getKeyed(path string, apiKey *string) *apiResponse {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	require.NoError(c.t, err)
	if apiKey != nil {
		req.Header.Set("X-API-Key", *apiKey)
	}
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)
	res := &apiResponse{t: c.t, status: resp.StatusCode, header: resp.Header, contentType: resp.Header.Get("Content-Type"), raw: string(raw)}
	require.NoErrorf(c.t, json.Unmarshal(raw, &res.envelope), "%s %s: body is not the JSON envelope: %q", path, resp.Status, raw)
	return res
}

type adminUnit struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	ProvinceCode string `json:"province_code"`
	DistrictCode string `json:"district_code"`
}

type adminItems struct {
	Items []adminUnit `json:"items"`
}

type adminCandidate struct {
	Ward        adminUnit `json:"ward"`
	District    adminUnit `json:"district"`
	Province    adminUnit `json:"province"`
	MappingType string    `json:"mapping_type"`
	Note        string    `json:"note"`
}

type adminWardConversion struct {
	Unique     bool             `json:"unique"`
	Candidates []adminCandidate `json:"candidates"`
}

type adminProvinceConversion struct {
	Unique     bool        `json:"unique"`
	Candidates []adminUnit `json:"candidates"`
}

func names(us []adminUnit) []string {
	out := make([]string, 0, len(us))
	for _, u := range us {
		out = append(out, u.Name)
	}
	return out
}

const adminBase = "/v1/administrative-units"

// TestAdministrativeUnits_E2E drives the nine public lookup routes, and the
// seeder that fills them, against the real binaries and a fresh Postgres
// (docs/specs/features/administrative-units/, AC 1–12, 14–20, 23, 24).
//
// What is here and not in the per-layer tests: that cmd/main.go registers every
// route, puts the web API key and the per-IP limit in front of them, that the
// config's cache lifetime reaches the response, and that the CLI an operator
// runs really fills the tables the routes read.
func TestAdministrativeUnits_E2E(t *testing.T) {
	requireE2E(t)
	if _, err := os.Stat(adminUnitsDataPath); os.IsNotExist(err) {
		t.Skip("assets/data/administrative-units.json is not in this checkout (it is gitignored)")
	}
	dataPath, err := filepath.Abs(adminUnitsDataPath)
	require.NoError(t, err)

	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	db := openDB(t, host, port)
	cfg := writeTempConfig(t, host, port, "0", nil)

	baseURL, cleanupSrv := startServer(t, host, port, nil)
	t.Cleanup(cleanupSrv)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
	webKey := devWebAPIKey(t)

	t.Run("before the dataset is loaded a list is empty, not an error", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/new/provinces", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var got adminItems
		res.data(&got)
		assert.Empty(t, got.Items)
		assert.Equal(t, 0, countRows(t, db, "addr_new_provinces"))
	})

	t.Run("CLI: a dry run checks the file and writes nothing", func(t *testing.T) {
		res := runCLI(t, cfg, "-file", dataPath, "-dry-run")
		require.False(t, res.timedOut, res.stderr)
		assert.Equal(t, 0, res.exitCode, res.stderr)
		assert.Contains(t, res.stderr, "34 provinces")
		assert.Contains(t, res.stderr, "nothing was written")
		assert.Equal(t, 0, countRows(t, db, "addr_new_wards"))
	})

	t.Run("CLI: a corrupted file is refused whole and nothing is written", func(t *testing.T) {
		var doc map[string]any
		raw, err := os.ReadFile(dataPath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &doc))
		// Put the original labelling defect back: the Tân An ward under Tiền Giang.
		for _, w := range doc["old_wards"].([]any) {
			if m := w.(map[string]any); m["code"] == "27691" {
				m["province_code"] = "82"
			}
		}
		bad := filepath.Join(t.TempDir(), "corrupt.json")
		out, err := json.Marshal(doc)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(bad, out, 0o600))

		res := runCLI(t, cfg, "-file", bad)

		require.False(t, res.timedOut, res.stderr)
		assert.Equal(t, 1, res.exitCode, res.stderr)
		assert.Contains(t, res.stderr, "ward_province_mismatch")
		assert.Contains(t, res.stderr, "27691")
		assert.Equal(t, 0, countRows(t, db, "addr_new_provinces"), "refused before the database was touched")
		assert.Equal(t, 0, countRows(t, db, "addr_dataset_versions"))
	})

	t.Run("CLI: load fills every table, and loading again changes nothing", func(t *testing.T) {
		for run := 1; run <= 2; run++ {
			res := runCLI(t, cfg, "-file", dataPath)
			require.False(t, res.timedOut, res.stderr)
			require.Equal(t, 0, res.exitCode, "run %d: %s", run, res.stderr)
			assert.Contains(t, res.stderr, "seeded administrative units")
		}
		assert.Equal(t, 34, countRows(t, db, "addr_new_provinces"))
		assert.Equal(t, 3321, countRows(t, db, "addr_new_wards"))
		assert.Equal(t, 63, countRows(t, db, "addr_old_provinces"))
		assert.Equal(t, 696, countRows(t, db, "addr_old_districts"))
		assert.Equal(t, 10033, countRows(t, db, "addr_old_wards"))
		assert.Equal(t, 10565, countRows(t, db, "addr_ward_mappings"))
		assert.Equal(t, 1, countRows(t, db, "addr_dataset_versions"), "one dataset version, however often it is loaded")
	})

	t.Run("access: no account needed, the web key is checked, a wrong key is refused", func(t *testing.T) {
		// AC 16/17: header absent (mobile) and correct key (web) are both served,
		// and neither sends an Authorization header.
		res := api.getKeyed(adminBase+"/new/provinces", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "public, max-age=86400", res.header.Get("Cache-Control"), "AC 24: dev cache_max_age is 24h")

		res = api.getKeyed(adminBase+"/new/provinces", &webKey)
		assert.Equal(t, http.StatusOK, res.status, res.raw)

		wrong := "not-the-key"
		res = api.getKeyed(adminBase+"/new/provinces", &wrong)
		assert.Equal(t, http.StatusUnauthorized, res.status, res.raw)
		assert.Equal(t, "INVALID_API_KEY", res.errorCode())
		assert.Empty(t, res.header.Get("Cache-Control"))
	})

	t.Run("new scheme: provinces, then the wards of one province", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/new/provinces", nil)
		var provs adminItems
		res.data(&provs)
		require.Len(t, provs.Items, 34, "AC 1")
		assert.Equal(t, "An Giang", provs.Items[0].Name, "Vietnamese alphabetical order")
		var hcm string
		for _, p := range provs.Items {
			if p.Name == "Hồ Chí Minh" {
				hcm = p.Code
				assert.Equal(t, "Thành phố Hồ Chí Minh", p.FullName)
			}
			assert.NotEqual(t, "Bình Dương", p.Name, "merged away: not a current province")
		}
		require.NotEmpty(t, hcm)

		res = api.getKeyed(adminBase+"/new/provinces/"+hcm+"/wards", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "public, max-age=86400", res.header.Get("Cache-Control"))
		var wards adminItems
		res.data(&wards)
		assert.Len(t, wards.Items, 168, "AC 2: every ward in one response, no paging")
		for _, w := range wards.Items {
			assert.Equal(t, hcm, w.ProvinceCode, "no ward of another province")
		}
		assert.Contains(t, names(wards.Items), "Tân Định")
		assert.Contains(t, names(wards.Items), "Thủ Dầu Một", "former Bình Dương")
		assert.Contains(t, names(wards.Items), "Vũng Tàu", "former Bà Rịa - Vũng Tàu")

		res = api.getKeyed(adminBase+"/new/provinces/no-such/wards", nil)
		assert.Equal(t, http.StatusNotFound, res.status, "AC 3")
		assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", res.errorCode())
		assert.Empty(t, res.header.Get("Cache-Control"), "an error is never cached")
	})

	t.Run("old scheme: provinces, districts, wards", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/old/provinces", nil)
		var provs adminItems
		res.data(&provs)
		require.Len(t, provs.Items, 63, "AC 4")
		var oldHCM string
		for _, p := range provs.Items {
			if p.Name == "Hồ Chí Minh" {
				oldHCM = p.Code
			}
		}
		require.NotEmpty(t, oldHCM)

		res = api.getKeyed(adminBase+"/old/provinces/"+oldHCM+"/districts", nil)
		var dists adminItems
		res.data(&dists)
		assert.Len(t, dists.Items, 22, "AC 5")
		assert.Equal(t, "Bình Chánh", dists.Items[0].Name)

		// District 774 has numbered wards: numbers compare as numbers.
		res = api.getKeyed(adminBase+"/old/districts/774/wards", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var wards adminItems
		res.data(&wards)
		order := names(wards.Items)
		i9, i11 := indexOf(order, "Phường 9"), indexOf(order, "Phường 11")
		require.GreaterOrEqual(t, i9, 0)
		require.GreaterOrEqual(t, i11, 0)
		assert.Less(t, i9, i11, `"Phường 9" before "Phường 11"`)

		// AC 6: an island district exists but has no wards: 200 and [].
		res = api.getKeyed(adminBase+"/old/districts/318/wards", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.JSONEq(t, `{"items":[]}`, string(res.envelope.Data))

		// AC 7
		res = api.getKeyed(adminBase+"/old/districts/no-such/wards", nil)
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", res.errorCode())
	})

	t.Run("the same code in both schemes names different units (AC 8)", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/old/districts/760/wards", nil)
		var legacy adminItems
		res.data(&legacy)
		var legacyName string
		for _, w := range legacy.Items {
			if w.Code == "26737" {
				legacyName = w.Name
			}
		}
		assert.Equal(t, "Đa Kao", legacyName, "old ward 26737")

		res = api.getKeyed(adminBase+"/old/wards/26737/conversion", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var conv adminWardConversion
		res.data(&conv)
		var current []string
		for _, c := range conv.Candidates {
			current = append(current, c.Ward.Code+":"+c.Ward.Name)
		}
		assert.Contains(t, current, "26737:Tân Định", "new ward 26737 is a different unit")
	})

	t.Run("conversion old → new (AC 9, 11)", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/old/wards/26761/conversion", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "public, max-age=86400", res.header.Get("Cache-Control"))
		var conv adminWardConversion
		res.data(&conv)
		assert.True(t, conv.Unique)
		require.Len(t, conv.Candidates, 1)
		c := conv.Candidates[0]
		assert.Equal(t, "Phường Cầu Ông Lãnh", c.Ward.FullName)
		assert.Equal(t, "Thành phố Hồ Chí Minh", c.Province.FullName)
		assert.Equal(t, "merged", c.MappingType)
		assert.Equal(t, "Nhập toàn bộ", c.Note)

		// A ward split between two new wards: both come back, nothing is chosen.
		res = api.getKeyed(adminBase+"/old/wards/26737/conversion", nil)
		res.data(&conv)
		assert.False(t, conv.Unique)
		assert.Len(t, conv.Candidates, 2)
		for _, c := range conv.Candidates {
			assert.Equal(t, "ambiguous", c.MappingType)
		}

		res = api.getKeyed(adminBase+"/old/wards/no-such/conversion", nil)
		assert.Equal(t, http.StatusNotFound, res.status, "AC 14")
		assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", res.errorCode())
	})

	t.Run("conversion new → old (AC 10)", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/new/wards/26758/conversion", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var conv adminWardConversion
		res.data(&conv)
		assert.False(t, conv.Unique)
		var got []string
		for _, c := range conv.Candidates {
			got = append(got, c.Ward.Name)
			assert.NotEmpty(t, c.District.FullName, "carries its old district")
			assert.Equal(t, "Hồ Chí Minh", c.Province.Name, "and its old province")
		}
		assert.ElementsMatch(t, []string{"Cầu Kho", "Cầu Ông Lãnh", "Cô Giang", "Nguyễn Cư Trinh"}, got)

		// A special zone that replaced a whole island district has no ward mapping:
		// 200 with no candidates, not a 404 (the unit exists).
		res = api.getKeyed(adminBase+"/new/wards/11948/conversion", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		res.data(&conv)
		assert.False(t, conv.Unique)
		assert.Empty(t, conv.Candidates)
	})

	t.Run("province conversion is derived from the wards (AC 12)", func(t *testing.T) {
		var conv adminProvinceConversion

		// Tiền Giang (old 82) → Đồng Tháp only: the Tân An ward that the source
		// dataset mislabelled must not make it two.
		res := api.getKeyed(adminBase+"/old/provinces/82/conversion", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		res.data(&conv)
		assert.True(t, conv.Unique)
		assert.Equal(t, []string{"Đồng Tháp"}, names(conv.Candidates))

		// new Hồ Chí Minh ← three old provinces.
		res = api.getKeyed(adminBase+"/new/provinces/79/conversion", nil)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		res.data(&conv)
		assert.False(t, conv.Unique)
		assert.ElementsMatch(t, []string{"Bà Rịa - Vũng Tàu", "Bình Dương", "Hồ Chí Minh"}, names(conv.Candidates))

		res = api.getKeyed(adminBase+"/new/provinces/no-such/conversion", nil)
		assert.Equal(t, http.StatusNotFound, res.status)
	})

	t.Run("ward search inside a province ignores case, diacritics, tone placement and apostrophes (AC 25, 26)", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/new/provinces", nil)
		var provs adminItems
		res.data(&provs)
		codeOf := func(name string) string {
			for _, p := range provs.Items {
				if p.Name == name {
					return p.Code
				}
			}
			t.Fatalf("no current province %q", name)
			return ""
		}
		search := func(province, q string) (*apiResponse, adminItems) {
			t.Helper()
			r := api.getKeyed(adminBase+"/new/provinces/"+province+"/wards?q="+url.QueryEscape(q), nil)
			var items adminItems
			if r.status == http.StatusOK {
				r.data(&items)
			}
			return r, items
		}

		// Tone-mark placement and case: Hóa / Hoá / hoa / HÓA are one search.
		thanh := codeOf("Thanh Hóa")
		_, base := search(thanh, "hoa")
		require.NotEmpty(t, base.Items)
		assert.Contains(t, names(base.Items), "Hoằng Hóa")
		for _, q := range []string{"Hóa", "Hoá", "HÓA", "hoá"} {
			r, got := search(thanh, q)
			require.Equal(t, http.StatusOK, r.status, r.raw)
			assert.Equal(t, names(base.Items), names(got.Items), "q=%q", q)
		}
		for _, w := range base.Items {
			assert.Equal(t, thanh, w.ProvinceCode, "never a ward of another province")
		}

		// An apostrophe of either kind finds the stored one (the data uses ’).
		lamDong := codeOf("Lâm Đồng")
		for _, q := range []string{"b'lao", "b’lao", "B’Lao"} {
			r, got := search(lamDong, q)
			require.Equal(t, http.StatusOK, r.status, r.raw)
			assert.Contains(t, names(got.Items), "B’Lao", "q=%q", q)
		}

		// Unaccented input finds accented names, and the full name is searched.
		r, got := search(codeOf("Hồ Chí Minh"), "phuong tan dinh")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		assert.Equal(t, "public, max-age=86400", r.header.Get("Cache-Control"))
		assert.Contains(t, names(got.Items), "Tân Định")

		// No match: 200 and [].
		r, got = search(thanh, "zzzzzz")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		assert.JSONEq(t, `{"items":[]}`, string(r.envelope.Data))
		assert.Empty(t, got.Items)

		// AC 26: a present-but-blank term is a 400, never "no search"; an over-long one too.
		for _, q := range []string{"", "   ", strings.Repeat("a", 101)} {
			r, _ := search(thanh, q)
			assert.Equal(t, http.StatusBadRequest, r.status, "q=%q: %s", q, r.raw)
			assert.Equal(t, "INVALID_FILTER", r.errorCode())
			assert.Equal(t, "q", r.details()["field"])
			assert.Empty(t, r.header.Get("Cache-Control"))
		}

		// An unknown province is 404 whatever the term.
		r, _ = search("no-such", "hoa")
		assert.Equal(t, http.StatusNotFound, r.status)
		assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", r.errorCode())

		// Without q nothing changed: every ward comes back.
		res = api.getKeyed(adminBase+"/new/provinces/"+thanh+"/wards", nil)
		var all adminItems
		res.data(&all)
		assert.Greater(t, len(all.Items), len(base.Items))
	})

	t.Run("reading and converting write nothing (AC 15)", func(t *testing.T) {
		// Every call above was a read; the tables must hold exactly what the CLI loaded.
		assert.Equal(t, 34, countRows(t, db, "addr_new_provinces"))
		assert.Equal(t, 3321, countRows(t, db, "addr_new_wards"))
		assert.Equal(t, 63, countRows(t, db, "addr_old_provinces"))
		assert.Equal(t, 696, countRows(t, db, "addr_old_districts"))
		assert.Equal(t, 10033, countRows(t, db, "addr_old_wards"))
		assert.Equal(t, 10565, countRows(t, db, "addr_ward_mappings"))
		assert.Equal(t, 1, countRows(t, db, "addr_dataset_versions"))
	})

	t.Run("an unknown route or method is the shared envelope, not a lookup", func(t *testing.T) {
		res := api.getKeyed(adminBase+"/new/nothing", nil)
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "NOT_FOUND", res.errorCode())

		req, err := http.NewRequest(http.MethodPost, baseURL+adminBase+"/new/provinces", nil)
		require.NoError(t, err)
		resp, err := api.http.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "read-only")
	})
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// TestAdministrativeUnits_RateLimit_E2E proves AC 23 against the real binary: the
// config's budget reaches the limiter, all nine routes share it, and a refused
// key does not spend it.
func TestAdministrativeUnits_RateLimit_E2E(t *testing.T) {
	baseURL, client := newE2EServerWithConfig(t, configOverrides{
		"administrative_units": {
			"rate_limit": map[string]any{"max": 3, "window": "1m", "trust_forwarded_for": false},
		},
	})
	api := &apiClient{t: t, baseURL: baseURL, http: client}
	wrong := "not-the-key"

	// A refused key is turned away before the limiter, so it spends nothing.
	for range 5 {
		res := api.getKeyed(adminBase+"/new/provinces", &wrong)
		require.Equal(t, http.StatusUnauthorized, res.status, res.raw)
	}

	// Three different routes share one budget of 3.
	for _, path := range []string{"/new/provinces", "/old/provinces", "/old/districts/318/wards"} {
		res := api.getKeyed(adminBase+path, nil)
		assert.NotEqual(t, http.StatusTooManyRequests, res.status, "%s inside the budget: %s", path, res.raw)
	}
	res := api.getKeyed(adminBase+"/new/provinces", nil)
	require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
	assert.Equal(t, "RATE_LIMITED", res.errorCode())
	assert.Empty(t, res.header.Get("Cache-Control"), "a limited answer is not cacheable")
}
