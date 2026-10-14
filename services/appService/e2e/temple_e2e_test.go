package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemple_E2E drives the public temple catalogue (docs/specs/features/temple/,
// AC 19-22 options, AC 26-27 and AC 32-33 structured address) against the real
// binary and a fresh Postgres.
//
// What is here and not in the per-layer tests: that migration 0032 is applied by the
// same glob the service relies on, that the routes are wired without any key, that
// province_code/ward_code/ward and full_address travel from the columns to the wire
// on options, top, list and detail, and that a temple which has not been re-seeded
// reads as null codes with its stored address as the full address.
func TestTemple_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, cleanupSrv := startServer(t, host, port, nil)
	t.Cleanup(cleanupSrv)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	db := openDB(t, host, port)
	ctx := context.Background()

	const cols = `(id, name, type, region, sect, established_year, heritage_rank, heritage_priority,
		latitude, longitude, address, province, province_code, ward_code, ward)`
	for _, q := range []string{
		// structured, with a street
		`INSERT INTO temples ` + cols + ` VALUES ('temple_south_002', 'Chùa Ngọc Hoàng', 'CHUA', 'south', 'BẮC_TÔNG', 1900,
			'DI_TICH_CAP_THANH_PHO', 7, 10.79, 106.69, '73 Mai Thị Lựu', 'Hồ Chí Minh', '79', '26737', 'Phường Tân Định')`,
		// structured, nothing below ward level (A-004)
		`INSERT INTO temples ` + cols + ` VALUES ('temple_north_003', 'Chùa Hương', 'CHUA', 'north', 'BẮC_TÔNG', 1600,
			'DI_TICH_QUOC_GIA_DAC_BIET', 2, 20.6, 105.7, '', 'Hà Nội', '01', '00679', 'Xã Ngọc Hồi')`,
		// not re-seeded yet: old full string, no codes
		`INSERT INTO temples ` + cols + ` VALUES ('temple_central_900', 'Chùa Cũ', 'CHUA', 'central', 'BẮC_TÔNG', 1800,
			'CHUA_XEP_HANG', 8, 16.4, 107.5, 'Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình', 'Ninh Bình', NULL, NULL, NULL)`,
	} {
		_, err := db.Exec(ctx, q)
		require.NoError(t, err, q)
	}

	type option struct {
		ID           string  `json:"id"`
		Name         string  `json:"name"`
		Province     string  `json:"province"`
		ProvinceCode *string `json:"province_code"`
	}
	type location struct {
		Address      string  `json:"address"`
		Province     string  `json:"province"`
		ProvinceCode *string `json:"province_code"`
		WardCode     *string `json:"ward_code"`
		Ward         *string `json:"ward"`
		FullAddress  string  `json:"full_address"`
	}
	type temple struct {
		ID       string   `json:"id"`
		Location location `json:"location"`
	}
	str := func(s *string) string {
		if s == nil {
			return "<null>"
		}
		return *s
	}

	// AC 19-20: public, ranked, four fields per item, province_code null for the old row.
	res := api.do(http.MethodGet, "/v1/temples/options", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	var opts struct {
		Items      []option `json:"items"`
		Total      int      `json:"total"`
		TotalPages int      `json:"total_pages"`
	}
	res.data(&opts)
	require.Len(t, opts.Items, 3)
	assert.Equal(t, 3, opts.Total)
	assert.Equal(t, []string{"temple_north_003", "temple_south_002", "temple_central_900"},
		[]string{opts.Items[0].ID, opts.Items[1].ID, opts.Items[2].ID}, "ranking order, not alphabetical")
	assert.Equal(t, "79", str(opts.Items[1].ProvinceCode))
	assert.Equal(t, "Hồ Chí Minh", opts.Items[1].Province)
	assert.Equal(t, "<null>", str(opts.Items[2].ProvinceCode), "a temple not re-seeded has no code")

	// AC 23: search still works next to the new field.
	res = api.do(http.MethodGet, "/v1/temples/options?q=ngoc%20hoang", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	res.data(&opts)
	require.Len(t, opts.Items, 1)
	assert.Equal(t, "temple_south_002", opts.Items[0].ID)

	// AC 33 on detail: structured temple with a street.
	res = api.do(http.MethodGet, "/v1/temples/temple_south_002", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	var d temple
	res.data(&d)
	assert.Equal(t, "73 Mai Thị Lựu", d.Location.Address)
	assert.Equal(t, "79", str(d.Location.ProvinceCode))
	assert.Equal(t, "26737", str(d.Location.WardCode))
	assert.Equal(t, "Phường Tân Định", str(d.Location.Ward))
	assert.Equal(t, "73 Mai Thị Lựu, Phường Tân Định, Hồ Chí Minh", d.Location.FullAddress)

	// AC 33: nothing below ward level - the empty address is not joined.
	res = api.do(http.MethodGet, "/v1/temples/temple_north_003", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	res.data(&d)
	assert.Equal(t, "", d.Location.Address)
	assert.Equal(t, "Xã Ngọc Hồi, Hà Nội", d.Location.FullAddress)

	// AC 32-33: not re-seeded - codes null, stored full string returned as is.
	res = api.do(http.MethodGet, "/v1/temples/temple_central_900", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	res.data(&d)
	assert.Equal(t, "<null>", str(d.Location.ProvinceCode))
	assert.Equal(t, "<null>", str(d.Location.WardCode))
	assert.Equal(t, "<null>", str(d.Location.Ward))
	assert.Equal(t, "Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình", d.Location.FullAddress)

	// top and list share the summary shape, so they carry the same location.
	res = api.do(http.MethodGet, "/v1/temples/top", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	var top struct {
		Items []temple `json:"items"`
	}
	res.data(&top)
	require.Len(t, top.Items, 3)
	assert.Equal(t, "Xã Ngọc Hồi, Hà Nội", top.Items[0].Location.FullAddress)

	res = api.do(http.MethodGet, "/v1/temples?page=1", nil, "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	var list struct {
		Items []temple `json:"items"`
	}
	res.data(&list)
	require.Len(t, list.Items, 3)
	assert.Equal(t, "79", str(list.Items[1].Location.ProvinceCode))

	// AC 3 / AC 9 still hold.
	res = api.do(http.MethodGet, "/v1/temples/temple_nowhere_1", nil, "")
	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, "TEMPLE_NOT_FOUND", res.errorCode())
}
