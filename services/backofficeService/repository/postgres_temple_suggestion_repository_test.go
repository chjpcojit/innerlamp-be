package repository

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// connectTestDB opens a scratch Postgres for this package's SQL tests. These
// tests TRUNCATE temples and temple_suggestions, so they refuse to run unless
// TEST_DB_NAME names a database other than the dev one (innerlamp) and skip
// under -short or when no Postgres answers.
func connectTestDB(t *testing.T) database.Database {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping postgres integration test in -short mode")
	}
	name := os.Getenv("TEST_DB_NAME")
	if name == "" || name == "innerlamp" {
		t.Skip("set TEST_DB_NAME to a scratch database (not innerlamp): these tests truncate temples and temple_suggestions")
	}
	port := 5432
	if v := os.Getenv("TEST_DB_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		require.NoError(t, err)
		port = n
	}
	db, err := database.New(&database.Config{
		Type: database.PostgreSQL, Host: envOr("TEST_DB_HOST", "localhost"), Port: port,
		Username: envOr("TEST_DB_USER", "postgres"), Password: envOr("TEST_DB_PASSWORD", "postgres"),
		Database: name, SSLMode: envOr("TEST_DB_SSLMODE", "disable"), Timeout: 5 * time.Second,
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Connect(ctx); err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, f := range []string{
		"0005_temples.sql", "0006_admins.sql", "0011_temple_content.sql", "0022_temple_suggestions.sql",
		"0024_temple_province.sql", "0026_administrative_units.sql", "0030_temple_suggestions_address_codes.sql",
		"0032_temple_address_codes.sql", "0001_register_onboarding.sql", "0036_temple_announcements.sql",
	} {
		b, err := os.ReadFile("../../../migrations/" + f)
		require.NoError(t, err)
		_, err = db.Exec(ctx, string(b))
		require.NoError(t, err, f)
	}
	_, err = db.Exec(ctx, `TRUNCATE temple_suggestions, temples CASCADE`)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `TRUNCATE addr_new_wards, addr_new_provinces CASCADE`)
	require.NoError(t, err)
	return db
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type suggestionEnv struct {
	repo    domain.TempleSuggestionRepository
	db      database.Database
	adminID int64
}

func newSuggestionEnv(t *testing.T) *suggestionEnv {
	t.Helper()
	db := connectTestDB(t)
	ctx := context.Background()
	_, err := db.Exec(ctx, `INSERT INTO addr_new_provinces (code, name, full_name, type, sort_order) VALUES
		('79', 'Hồ Chí Minh', 'Thành phố Hồ Chí Minh', 'city', 1), ('01', 'Hà Nội', 'Thành phố Hà Nội', 'city', 2)`)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO addr_new_wards (code, province_code, name, full_name, type, sort_order) VALUES
		('26740', '79', 'Sài Gòn', 'Phường Sài Gòn', 'ward', 1), ('00004', '01', 'Ba Đình', 'Phường Ba Đình', 'ward', 1)`)
	require.NoError(t, err)

	var adminID int64
	require.NoError(t, db.QueryRow(ctx, `INSERT INTO admins (email, password_hash, name)
		VALUES ('reviewer@example.com', 'x', 'Reviewer') ON CONFLICT (email) DO UPDATE SET name = 'Reviewer' RETURNING id`).Scan(&adminID))
	return &suggestionEnv{repo: NewTempleSuggestionRepository(db), db: db, adminID: adminID}
}

func (e *suggestionEnv) addSuggestion(t *testing.T, name string, structured bool) int64 {
	t.Helper()
	var id int64
	var err error
	if structured {
		err = e.db.QueryRow(context.Background(), `INSERT INTO temple_suggestions (name, province_code, province, ward_code, ward, address)
			VALUES ($1, '79', 'Hồ Chí Minh', '26740', 'Phường Sài Gòn', '12 Nguyễn Huệ') RETURNING id`, name).Scan(&id)
	} else {
		err = e.db.QueryRow(context.Background(), `INSERT INTO temple_suggestions (name, province, address)
			VALUES ($1, 'Quảng Ninh', 'Thôn 1') RETURNING id`, name).Scan(&id)
	}
	require.NoError(t, err)
	return id
}

func validTemple(region string) domain.NewTemple {
	return domain.NewTemple{
		Name: "Chùa Mới", Type: "CHUA", Region: region, Sect: "Bắc tông", EstablishedYear: 1990,
		HeritageRank: "DI_TICH_CAP_TINH", Latitude: 10.77, Longitude: 106.69,
		Address: "12 Nguyễn Huệ", ProvinceCode: "79", WardCode: "26740", Province: "Hồ Chí Minh", Ward: "Phường Sài Gòn",
		Aliases: []string{"Chùa X"}, VisitingNotes: []string{},
	}
}

func TestTempleSuggestionRepository_ListAndFind(t *testing.T) {
	e := newSuggestionEnv(t)
	ctx := context.Background()
	a := e.addSuggestion(t, "A", true)
	b := e.addSuggestion(t, "B", false)
	_, err := e.repo.CreateTemple(ctx, a, e.adminID, domain.RoleAdmin, validTemple("south"), 6)
	require.NoError(t, err)

	pending, total, err := e.repo.List(ctx, domain.TempleSuggestionPending, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, pending, 1)
	assert.Equal(t, b, pending[0].ID)
	// legacy row: no codes, province as typed.
	assert.Nil(t, pending[0].ProvinceCode)
	assert.Nil(t, pending[0].WardCode)
	assert.Nil(t, pending[0].Ward)
	assert.Equal(t, "Quảng Ninh", pending[0].Province)

	created, _, err := e.repo.List(ctx, domain.TempleSuggestionCreated, 0, 10)
	require.NoError(t, err)
	require.Len(t, created, 1)
	assert.Equal(t, a, created[0].ID)
	require.NotNil(t, created[0].ProvinceCode)
	assert.Equal(t, "79", *created[0].ProvinceCode)
	require.NotNil(t, created[0].TempleID)
	assert.Equal(t, "temple_south_001", *created[0].TempleID)
	require.NotNil(t, created[0].ReviewedBy)
	assert.Equal(t, e.adminID, *created[0].ReviewedBy)

	got, err := e.repo.FindByID(ctx, a)
	require.NoError(t, err)
	assert.Equal(t, "A", got.Name)
	_, err = e.repo.FindByID(ctx, 999999)
	assert.ErrorIs(t, err, domain.ErrSuggestionNotFound)
}

func TestTempleSuggestionRepository_ListOrderAndPaging(t *testing.T) {
	e := newSuggestionEnv(t)
	ctx := context.Background()
	var ids []int64
	for _, n := range []string{"1", "2", "3"} {
		ids = append(ids, e.addSuggestion(t, n, true))
	}
	// Same created_at: id breaks the tie so paging is stable.
	_, err := e.db.Exec(ctx, `UPDATE temple_suggestions SET created_at = '2026-01-01'`)
	require.NoError(t, err)

	p1, total, err := e.repo.List(ctx, domain.TempleSuggestionPending, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, p1, 2)
	assert.Equal(t, ids[0], p1[0].ID)
	assert.Equal(t, ids[1], p1[1].ID)
	p2, _, err := e.repo.List(ctx, domain.TempleSuggestionPending, 2, 2)
	require.NoError(t, err)
	require.Len(t, p2, 1)
	assert.Equal(t, ids[2], p2[0].ID)
	p3, _, err := e.repo.List(ctx, domain.TempleSuggestionPending, 4, 2)
	require.NoError(t, err)
	assert.Empty(t, p3)
}

func TestTempleSuggestionRepository_CreateTemple(t *testing.T) {
	e := newSuggestionEnv(t)
	ctx := context.Background()
	id := e.addSuggestion(t, "Chùa Mới", true)

	res, err := e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleContentEditor, validTemple("south"), 6)
	require.NoError(t, err)
	assert.Equal(t, "temple_south_001", res.Temple.ID)
	assert.Equal(t, domain.TempleSuggestionCreated, res.Suggestion.Status)

	var provinceCode, wardCode, ward, province, address string
	var priority int
	require.NoError(t, e.db.QueryRow(ctx, `SELECT province_code, ward_code, ward, province, address, heritage_priority FROM temples WHERE id = $1`,
		res.Temple.ID).Scan(&provinceCode, &wardCode, &ward, &province, &address, &priority))
	assert.Equal(t, "79", provinceCode)
	assert.Equal(t, "26740", wardCode)
	assert.Equal(t, "Phường Sài Gòn", ward)
	assert.Equal(t, "Hồ Chí Minh", province)
	assert.Equal(t, "12 Nguyễn Huệ", address)
	assert.Equal(t, 6, priority)
	assert.Equal(t, "12 Nguyễn Huệ, Phường Sài Gòn, Hồ Chí Minh", res.Temple.FullAddress())

	// The sequence continues per region: a second temple in south is 002, the
	// first in north is 001.
	s2 := e.addSuggestion(t, "Hai", true)
	r2, err := e.repo.CreateTemple(ctx, s2, e.adminID, domain.RoleAdmin, validTemple("south"), 6)
	require.NoError(t, err)
	assert.Equal(t, "temple_south_002", r2.Temple.ID)
	s3 := e.addSuggestion(t, "Ba", true)
	r3, err := e.repo.CreateTemple(ctx, s3, e.adminID, domain.RoleAdmin, validTemple("north"), 6)
	require.NoError(t, err)
	assert.Equal(t, "temple_north_001", r3.Temple.ID)
}

func TestTempleSuggestionRepository_CreateTemple_StateRules(t *testing.T) {
	e := newSuggestionEnv(t)
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		_, err := e.repo.CreateTemple(ctx, 424242, e.adminID, domain.RoleAdmin, validTemple("south"), 6)
		assert.ErrorIs(t, err, domain.ErrSuggestionNotFound)
	})

	t.Run("already created leaves one temple", func(t *testing.T) {
		id := e.addSuggestion(t, "X", true)
		_, err := e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleAdmin, validTemple("central"), 6)
		require.NoError(t, err)
		_, err = e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleAdmin, validTemple("central"), 6)
		assert.ErrorIs(t, err, domain.ErrSuggestionAlreadyCreated)
		var n int
		require.NoError(t, e.db.QueryRow(ctx, `SELECT count(*) FROM temples WHERE region = 'central'`).Scan(&n))
		assert.Equal(t, 1, n)
	})

	t.Run("rejected: content_editor forbidden, admin may and reason is cleared", func(t *testing.T) {
		id := e.addSuggestion(t, "R", true)
		_, err := e.db.Exec(ctx, `UPDATE temple_suggestions SET status='rejected', reviewed_by=$2, reviewed_at=now(), reject_reason='rác' WHERE id=$1`, id, e.adminID)
		require.NoError(t, err)

		_, err = e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleContentEditor, validTemple("north"), 6)
		assert.ErrorIs(t, err, domain.ErrReopenForbidden)
		s, err := e.repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, domain.TempleSuggestionRejected, s.Status)

		res, err := e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleAdmin, validTemple("north"), 6)
		require.NoError(t, err)
		assert.Nil(t, res.Suggestion.RejectReason)
		assert.Equal(t, domain.TempleSuggestionCreated, res.Suggestion.Status)
	})

	t.Run("insert failure rolls back and suggestion stays pending", func(t *testing.T) {
		id := e.addSuggestion(t, "Bad", true)
		bad := validTemple("south")
		bad.HeritageRank = "NOT_A_RANK" // violates chk_temples_heritage_rank
		_, err := e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleAdmin, bad, 6)
		require.Error(t, err)
		s, err := e.repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, domain.TempleSuggestionPending, s.Status)
		assert.Nil(t, s.TempleID)
	})
}

func TestTempleSuggestionRepository_CreateTemple_RaceOneWins(t *testing.T) {
	e := newSuggestionEnv(t)
	ctx := context.Background()
	id := e.addSuggestion(t, "Race", true)

	const n = 4
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = e.repo.CreateTemple(ctx, id, e.adminID, domain.RoleAdmin, validTemple("south"), 6)
		}()
	}
	wg.Wait()

	ok, conflict := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case assert.ErrorIs(t, err, domain.ErrSuggestionAlreadyCreated):
			conflict++
		}
	}
	assert.Equal(t, 1, ok)
	assert.Equal(t, n-1, conflict)
	var temples int
	require.NoError(t, e.db.QueryRow(ctx, `SELECT count(*) FROM temples`).Scan(&temples))
	assert.Equal(t, 1, temples)
}

func TestTempleSuggestionRepository_ResolveCurrentWard(t *testing.T) {
	e := newSuggestionEnv(t)
	r := e.repo.(domain.WardResolver)
	ctx := context.Background()

	got, err := r.ResolveCurrentWard(ctx, "79", "26740")
	require.NoError(t, err)
	assert.Equal(t, "Hồ Chí Minh", got.ProvinceName)
	assert.Equal(t, "Phường Sài Gòn", got.WardFullName)

	_, err = r.ResolveCurrentWard(ctx, "99", "26740")
	assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
	_, err = r.ResolveCurrentWard(ctx, "01", "26740") // ward of another province
	assert.ErrorIs(t, err, domain.ErrWardNotInProvince)
	_, err = r.ResolveCurrentWard(ctx, "79", "00000")
	assert.ErrorIs(t, err, domain.ErrWardNotInProvince)
}
