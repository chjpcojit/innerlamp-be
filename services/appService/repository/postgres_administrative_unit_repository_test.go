package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// adminUnitDataset is a small dataset with the shapes the real one has: a
// merge of whole provinces (L1, L2 → current 79), a province that stays alone
// (L3 → 01), a ward spread over two current wards (W2), a legacy ward that
// changes province's name only, and an island district with no wards whose
// current ward is reached only through a district mapping (C-ISLAND).
//
// sort_order is deliberately the reverse of the code order in several places,
// so a test that passes proves the repository orders by sort_order, not code.
func adminUnitDataset() domain.Dataset {
	return domain.Dataset{
		Version:    "test-1",
		SourceNote: "repository test",
		CurrentProvinces: []domain.CurrentProvince{
			{Code: "01", Name: "Hà Nội", FullName: "Thành phố Hà Nội", Type: "city", SortOrder: 2},
			{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 1},
		},
		CurrentWards: []domain.CurrentWard{
			{Code: "C1", ProvinceCode: "79", Name: "Tân Định", FullName: "Phường Tân Định", Type: "ward", SortOrder: 2},
			{Code: "C2", ProvinceCode: "79", Name: "Sài Gòn", FullName: "Phường Sài Gòn", Type: "ward", SortOrder: 1},
			{Code: "C3", ProvinceCode: "01", Name: "Ba Đình", FullName: "Phường Ba Đình", Type: "ward", SortOrder: 1},
			{Code: "C-ISLAND", ProvinceCode: "01", Name: "Bạch Long Vĩ", FullName: "Đặc khu Bạch Long Vĩ", Type: "special_zone", SortOrder: 2},
		},
		LegacyProvinces: []domain.LegacyProvince{
			{Code: "L1", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 2},
			{Code: "L2", Name: "Bình Dương", FullName: "Tỉnh Bình Dương", Type: "province", SortOrder: 1},
			{Code: "L3", Name: "Hà Nội", FullName: "Thành phố Hà Nội", Type: "city", SortOrder: 3},
		},
		LegacyDistricts: []domain.LegacyDistrict{
			{Code: "D1", ProvinceCode: "L1", Name: "Quận 1", FullName: "Quận 1", Type: "urban_district", SortOrder: 2},
			{Code: "D2", ProvinceCode: "L1", Name: "Quận 3", FullName: "Quận 3", Type: "urban_district", SortOrder: 1},
			{Code: "D3", ProvinceCode: "L2", Name: "Thủ Dầu Một", FullName: "Thành phố Thủ Dầu Một", Type: "city", SortOrder: 1},
			{Code: "D4", ProvinceCode: "L3", Name: "Ba Đình", FullName: "Quận Ba Đình", Type: "urban_district", SortOrder: 1},
			{Code: "D5", ProvinceCode: "L3", Name: "Bạch Long Vĩ", FullName: "Huyện Bạch Long Vĩ", Type: "district", SortOrder: 2},
		},
		LegacyWards: []domain.LegacyWard{
			{Code: "W1", DistrictCode: "D1", ProvinceCode: "L1", Name: "Cầu Kho", FullName: "Phường Cầu Kho", Type: "ward", SortOrder: 2},
			{Code: "W2", DistrictCode: "D1", ProvinceCode: "L1", Name: "Đa Kao", FullName: "Phường Đa Kao", Type: "ward", SortOrder: 1},
			{Code: "W3", DistrictCode: "D3", ProvinceCode: "L2", Name: "Hiệp Thành", FullName: "Phường Hiệp Thành", Type: "ward", SortOrder: 1},
			{Code: "W4", DistrictCode: "D4", ProvinceCode: "L3", Name: "Phúc Xá", FullName: "Phường Phúc Xá", Type: "ward", SortOrder: 1},
		},
		WardMappings: []domain.WardMapping{
			{LegacyWardCode: "W1", CurrentWardCode: "C1", MappingType: "merged", Note: "Nhập toàn bộ"},
			{LegacyWardCode: "W2", CurrentWardCode: "C1", MappingType: "ambiguous", Note: "Nhập một phần"},
			{LegacyWardCode: "W2", CurrentWardCode: "C2", MappingType: "ambiguous", Note: "Nhập một phần"},
			{LegacyWardCode: "W3", CurrentWardCode: "C2", MappingType: "province_changed", Note: "Nhập toàn bộ"},
			{LegacyWardCode: "W4", CurrentWardCode: "C3", MappingType: "unchanged", Note: "Giữ nguyên"},
		},
		DistrictMappings: []domain.DistrictMapping{
			{LegacyDistrictCode: "D5", CurrentWardCode: "C-ISLAND", MappingType: "merged", Note: "island"},
		},
	}
}

// newAdminUnitRepo returns a repository over a database whose addr_* tables
// are empty. It clears only those tables, so other repository tests running in
// the same database are not disturbed.
func newAdminUnitRepo(t *testing.T) (domain.AdministrativeUnitRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)

	// These tests TRUNCATE the addr_* tables. A database that holds a real
	// dataset (its version is a date such as 2026-09-25, set by cmd/seedadminunits)
	// must not be wiped by a plain `go test`: point TEST_DB_NAME at a scratch
	// database instead.
	var realVersions int
	require.NoError(t, db.QueryRow(context.Background(),
		`SELECT count(*) FROM addr_dataset_versions WHERE version ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'`).Scan(&realVersions))
	if realVersions > 0 {
		t.Skip("this database holds a real administrative-units dataset; set TEST_DB_NAME to a scratch database to run this test")
	}

	_, err := db.Exec(context.Background(), `TRUNCATE TABLE addr_ward_mappings, addr_old_wards,
		addr_old_districts, addr_old_provinces, addr_new_wards, addr_new_provinces, addr_dataset_versions`)
	require.NoError(t, err)
	return NewAdministrativeUnitRepository(db), db
}

func loadedAdminUnitRepo(t *testing.T) (domain.AdministrativeUnitRepository, database.Database) {
	t.Helper()
	repo, db := newAdminUnitRepo(t)
	require.NoError(t, repo.ReplaceDataset(context.Background(), adminUnitDataset()))
	return repo, db
}

func countRows(t *testing.T, db database.Database, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&n))
	return n
}

func codesOf[T any](items []T, code func(T) string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, code(it))
	}
	return out
}

func TestAdministrativeUnitRepository_ListsCurrentScheme(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	provs, err := repo.ListCurrentProvinces(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"79", "01"},
		codesOf(provs, func(p domain.CurrentProvince) string { return p.Code }),
		"ordered by sort_order, not by code")
	assert.Equal(t, domain.CurrentProvince{
		Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 1,
	}, provs[0], "every field round-trips, Vietnamese text intact")

	wards, err := repo.ListCurrentWards(ctx, "79")
	require.NoError(t, err)
	assert.Equal(t, []string{"C2", "C1"},
		codesOf(wards, func(w domain.CurrentWard) string { return w.Code }))
	for _, w := range wards {
		assert.Equal(t, "79", w.ProvinceCode, "no ward of another province")
	}

	_, err = repo.ListCurrentWards(ctx, "nope")
	assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
}

func TestAdministrativeUnitRepository_ResolveCurrentWard(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	got, err := repo.ResolveCurrentWard(ctx, "79", "C1")
	require.NoError(t, err)
	assert.Equal(t, &domain.ResolvedWard{ProvinceName: "Hồ Chí Minh", WardFullName: "Phường Tân Định"}, got,
		"province short name + ward full name, Vietnamese text intact")

	// A ward that exists but sits in another province is not the province's ward.
	_, err = repo.ResolveCurrentWard(ctx, "01", "C1")
	assert.ErrorIs(t, err, domain.ErrWardNotInProvince)

	// A ward code that does not exist.
	_, err = repo.ResolveCurrentWard(ctx, "79", "nope")
	assert.ErrorIs(t, err, domain.ErrWardNotInProvince)

	// A legacy code is not a current-scheme code, whichever side it is on.
	_, err = repo.ResolveCurrentWard(ctx, "L1", "C1")
	assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
	_, err = repo.ResolveCurrentWard(ctx, "79", "W1")
	assert.ErrorIs(t, err, domain.ErrWardNotInProvince)

	// An unknown province is told apart from an unknown ward.
	_, err = repo.ResolveCurrentWard(ctx, "nope", "C1")
	assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
}

func TestAdministrativeUnitRepository_ListsLegacyScheme(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	provs, err := repo.ListLegacyProvinces(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"L2", "L1", "L3"},
		codesOf(provs, func(p domain.LegacyProvince) string { return p.Code }))

	dists, err := repo.ListLegacyDistricts(ctx, "L1")
	require.NoError(t, err)
	assert.Equal(t, []string{"D2", "D1"},
		codesOf(dists, func(d domain.LegacyDistrict) string { return d.Code }))

	wards, err := repo.ListLegacyWards(ctx, "D1")
	require.NoError(t, err)
	assert.Equal(t, []string{"W2", "W1"},
		codesOf(wards, func(w domain.LegacyWard) string { return w.Code }))
	assert.Equal(t, "L1", wards[0].ProvinceCode, "province comes from the district")
	assert.Equal(t, "D1", wards[0].DistrictCode)

	_, err = repo.ListLegacyDistricts(ctx, "nope")
	assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
}

func TestAdministrativeUnitRepository_ListLegacyWards_DistrictWithoutWardsIsEmptyNotMissing(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	got, err := repo.ListLegacyWards(ctx, "D5") // island district, exists, no wards
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)

	_, err = repo.ListLegacyWards(ctx, "no-such-district")
	assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
}

func TestAdministrativeUnitRepository_ConvertLegacyWard(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	t.Run("a whole merge gives one candidate with province, type and note", func(t *testing.T) {
		got, err := repo.ConvertLegacyWard(ctx, "W1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		c := got[0]
		assert.Equal(t, "C1", c.Ward.Code)
		assert.Equal(t, "Phường Tân Định", c.Ward.FullName)
		assert.Equal(t, "79", c.Ward.ProvinceCode)
		assert.Equal(t, "79", c.Province.Code)
		assert.Equal(t, "Thành phố Hồ Chí Minh", c.Province.FullName)
		assert.Equal(t, "merged", c.MappingType)
		assert.Equal(t, "Nhập toàn bộ", c.Note)
	})

	t.Run("a ward spread over two wards returns both, ordered by sort_order", func(t *testing.T) {
		got, err := repo.ConvertLegacyWard(ctx, "W2")
		require.NoError(t, err)
		assert.Equal(t, []string{"C2", "C1"},
			codesOf(got, func(c domain.CurrentWardCandidate) string { return c.Ward.Code }))
	})

	t.Run("an unknown ward is not found", func(t *testing.T) {
		_, err := repo.ConvertLegacyWard(ctx, "nope")
		assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
	})
}

func TestAdministrativeUnitRepository_ConvertCurrentWard(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	t.Run("a ward gathering several legacy wards returns each with district and province", func(t *testing.T) {
		got, err := repo.ConvertCurrentWard(ctx, "C1")
		require.NoError(t, err)
		assert.Equal(t, []string{"W2", "W1"},
			codesOf(got, func(c domain.LegacyWardCandidate) string { return c.Ward.Code }))
		first := got[0]
		assert.Equal(t, "D1", first.District.Code)
		assert.Equal(t, "Quận 1", first.District.FullName)
		assert.Equal(t, "L1", first.Province.Code)
		assert.Equal(t, "L1", first.Ward.ProvinceCode)
		assert.Equal(t, "ambiguous", first.MappingType)
	})

	t.Run("an island special zone reached only by a district mapping returns no candidates, not an error", func(t *testing.T) {
		got, err := repo.ConvertCurrentWard(ctx, "C-ISLAND")
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("an unknown ward is not found", func(t *testing.T) {
		_, err := repo.ConvertCurrentWard(ctx, "nope")
		assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
	})
}

func TestAdministrativeUnitRepository_ProvinceConversionIsDerivedFromWards(t *testing.T) {
	repo, _ := loadedAdminUnitRepo(t)
	ctx := context.Background()

	t.Run("legacy → current is exactly one", func(t *testing.T) {
		for _, legacy := range []string{"L1", "L2"} {
			got, err := repo.ConvertLegacyProvince(ctx, legacy)
			require.NoError(t, err)
			assert.Equal(t, []string{"79"},
				codesOf(got, func(p domain.CurrentProvince) string { return p.Code }), "legacy %s", legacy)
		}
	})

	t.Run("current → legacy lists every merged province", func(t *testing.T) {
		got, err := repo.ConvertCurrentProvince(ctx, "79")
		require.NoError(t, err)
		assert.Equal(t, []string{"L2", "L1"},
			codesOf(got, func(p domain.LegacyProvince) string { return p.Code }),
			"Hồ Chí Minh gathers Bình Dương and the old Hồ Chí Minh, ordered by sort_order")
	})

	t.Run("a province that stays alone maps to itself", func(t *testing.T) {
		got, err := repo.ConvertLegacyProvince(ctx, "L3")
		require.NoError(t, err)
		assert.Equal(t, []string{"01"}, codesOf(got, func(p domain.CurrentProvince) string { return p.Code }))
	})

	t.Run("unknown provinces are not found in both directions", func(t *testing.T) {
		_, err := repo.ConvertLegacyProvince(ctx, "nope")
		assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
		_, err = repo.ConvertCurrentProvince(ctx, "nope")
		assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
	})
}

func TestAdministrativeUnitRepository_ReplaceDataset_IsIdempotent(t *testing.T) {
	repo, db := loadedAdminUnitRepo(t)
	ctx := context.Background()

	tables := []string{"addr_new_provinces", "addr_new_wards", "addr_old_provinces",
		"addr_old_districts", "addr_old_wards", "addr_ward_mappings", "addr_dataset_versions"}
	before := map[string]int{}
	for _, tb := range tables {
		before[tb] = countRows(t, db, tb)
	}
	assert.Equal(t, 2, before["addr_new_provinces"])
	assert.Equal(t, 4, before["addr_new_wards"])
	assert.Equal(t, 3, before["addr_old_provinces"])
	assert.Equal(t, 5, before["addr_old_districts"])
	assert.Equal(t, 4, before["addr_old_wards"])
	assert.Equal(t, 5, before["addr_ward_mappings"])
	assert.Equal(t, 1, before["addr_dataset_versions"])

	require.NoError(t, repo.ReplaceDataset(ctx, adminUnitDataset()))

	for _, tb := range tables {
		assert.Equal(t, before[tb], countRows(t, db, tb), "%s unchanged by a second load of the same dataset", tb)
	}
}

func TestAdministrativeUnitRepository_ReplaceDataset_UpdatesInPlaceAndNeverDeletes(t *testing.T) {
	repo, db := loadedAdminUnitRepo(t)
	ctx := context.Background()

	// A newer dataset: renames one ward, and no longer contains W4 / C3 / L3.
	next := adminUnitDataset()
	next.Version = "test-2"
	next.CurrentWards[0].Name = "Tân Định (đổi tên)"
	next.CurrentWards[0].FullName = "Phường Tân Định (đổi tên)"
	next.CurrentWards = next.CurrentWards[:2]
	next.LegacyWards = next.LegacyWards[:3]
	next.WardMappings = next.WardMappings[:4]
	require.NoError(t, repo.ReplaceDataset(ctx, next))

	wards, err := repo.ListCurrentWards(ctx, "79")
	require.NoError(t, err)
	var renamed string
	for _, w := range wards {
		if w.Code == "C1" {
			renamed = w.FullName
		}
	}
	assert.Equal(t, "Phường Tân Định (đổi tên)", renamed, "an existing unit is updated in place")

	assert.Equal(t, 4, countRows(t, db, "addr_new_wards"), "a unit absent from the new dataset is left alone")
	assert.Equal(t, 4, countRows(t, db, "addr_old_wards"))
	assert.Equal(t, 5, countRows(t, db, "addr_ward_mappings"))
	assert.Equal(t, 2, countRows(t, db, "addr_dataset_versions"), "each new version is recorded once")
}

func TestAdministrativeUnitRepository_ReplaceDataset_IsAtomic(t *testing.T) {
	repo, db := newAdminUnitRepo(t)
	ctx := context.Background()

	bad := adminUnitDataset()
	// The last write of the transaction fails its foreign key; everything
	// written before it must roll back with it.
	bad.WardMappings = append(bad.WardMappings,
		domain.WardMapping{LegacyWardCode: "W1", CurrentWardCode: "no-such-ward", MappingType: "merged"})

	err := repo.ReplaceDataset(ctx, bad)
	require.Error(t, err)

	for _, tb := range []string{"addr_new_provinces", "addr_new_wards", "addr_old_provinces",
		"addr_old_districts", "addr_old_wards", "addr_ward_mappings", "addr_dataset_versions"} {
		assert.Zero(t, countRows(t, db, tb), "%s must be empty after a rolled-back load", tb)
	}
}

func TestAdministrativeUnitRepository_ReplaceDataset_RequiresAVersion(t *testing.T) {
	repo, db := newAdminUnitRepo(t)

	noVersion := adminUnitDataset()
	noVersion.Version = ""

	require.Error(t, repo.ReplaceDataset(context.Background(), noVersion))
	assert.Zero(t, countRows(t, db, "addr_new_provinces"))
}

func TestAdministrativeUnitRepository_SameCodeInBothSchemesStaysSeparate(t *testing.T) {
	repo, _ := newAdminUnitRepo(t)
	ctx := context.Background()

	// 26737 is a legacy ward (Đa Kao) and a current ward (Tân Định) in the real
	// data: the two tables must hold it side by side without collision.
	ds := domain.Dataset{
		Version:          "collision",
		CurrentProvinces: []domain.CurrentProvince{{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 1}},
		CurrentWards:     []domain.CurrentWard{{Code: "26737", ProvinceCode: "79", Name: "Tân Định", FullName: "Phường Tân Định", Type: "ward", SortOrder: 1}},
		LegacyProvinces:  []domain.LegacyProvince{{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 1}},
		LegacyDistricts:  []domain.LegacyDistrict{{Code: "760", ProvinceCode: "79", Name: "Quận 1", FullName: "Quận 1", Type: "urban_district", SortOrder: 1}},
		LegacyWards:      []domain.LegacyWard{{Code: "26737", DistrictCode: "760", ProvinceCode: "79", Name: "Đa Kao", FullName: "Phường Đa Kao", Type: "ward", SortOrder: 1}},
		WardMappings:     []domain.WardMapping{{LegacyWardCode: "26737", CurrentWardCode: "26737", MappingType: "ambiguous"}},
	}
	require.NoError(t, repo.ReplaceDataset(ctx, ds))

	legacy, err := repo.ListLegacyWards(ctx, "760")
	require.NoError(t, err)
	require.Len(t, legacy, 1)
	assert.Equal(t, "Phường Đa Kao", legacy[0].FullName)

	current, err := repo.ListCurrentWards(ctx, "79")
	require.NoError(t, err)
	require.Len(t, current, 1)
	assert.Equal(t, "Phường Tân Định", current[0].FullName)

	conv, err := repo.ConvertLegacyWard(ctx, "26737")
	require.NoError(t, err)
	require.Len(t, conv, 1)
	assert.Equal(t, "Phường Tân Định", conv[0].Ward.FullName)
}

func TestAdministrativeUnitRepository_SearchCurrentWards(t *testing.T) {
	repo, _ := newAdminUnitRepo(t)
	ctx := context.Background()

	// Built from code points so the fixture states exactly which form each
	// name is in: tone mark on the first vowel (Hòa) or on the second (Hoà),
	// a curly right quote (U+2019) or a straight one, and a capital Đ.
	hoaOld := "H" + string(rune(0x00F2)) + "a"
	hoaNew := "Ho" + string(rune(0x00E0))
	curly := string(rune(0x2019))
	bigD := string(rune(0x0110))
	ward := func(code, province, name string, order int) domain.CurrentWard {
		return domain.CurrentWard{Code: code, ProvinceCode: province, Name: name, FullName: "Phường " + name, Type: "ward", SortOrder: order}
	}
	ds := domain.Dataset{
		Version: "search",
		CurrentProvinces: []domain.CurrentProvince{
			{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 1},
			{Code: "01", Name: "Hà Nội", FullName: "Thành phố Hà Nội", Type: "city", SortOrder: 2},
		},
		CurrentWards: []domain.CurrentWard{
			ward("C1", "79", hoaOld+" Thắng", 4),
			ward("C2", "79", hoaNew+" Bình", 2),
			ward("C3", "79", "B"+curly+"Lao", 1),
			ward("C4", "79", "D'Ran", 3),
			ward("C5", "79", bigD+"a Kao", 5),
			ward("C6", "79", "Ba_Be", 6),
			ward("C7", "79", "Cabbe", 7),
			ward("C8", "01", hoaOld+" Phát", 1), // another province: must never appear for 79
		},
	}
	require.NoError(t, repo.ReplaceDataset(ctx, ds))

	codes := func(q string) []string {
		t.Helper()
		got, err := repo.SearchCurrentWards(ctx, "79", q)
		require.NoError(t, err, q)
		return codesOf(got, func(w domain.CurrentWard) string { return w.Code })
	}

	t.Run("tone-mark placement does not matter, in either direction", func(t *testing.T) {
		want := []string{"C2", "C1"} // ordered by sort_order (2, 4), not by code
		assert.Equal(t, want, codes("hoa"))
		assert.Equal(t, want, codes(hoaOld), "typed Hòa")
		assert.Equal(t, want, codes(hoaNew), "typed Hoà")
		assert.Equal(t, want, codes("HOA"), "case does not matter")
	})

	t.Run("either apostrophe finds either apostrophe", func(t *testing.T) {
		assert.Equal(t, []string{"C3"}, codes("b'lao"), "straight term, curly stored")
		assert.Equal(t, []string{"C3"}, codes("b"+curly+"lao"), "curly term, curly stored")
		assert.Equal(t, []string{"C4"}, codes("d"+curly+"ran"), "curly term, straight stored")
		assert.Equal(t, []string{"C4"}, codes("D'Ran"), "straight term, straight stored")
	})

	t.Run("đ is found by d and d by đ", func(t *testing.T) {
		assert.Equal(t, []string{"C5"}, codes("da kao"))
		assert.Equal(t, []string{"C5"}, codes(bigD+"a kao"))
		assert.Equal(t, []string{"C5"}, codes("đa"))
	})

	t.Run("a wildcard in the term is taken literally", func(t *testing.T) {
		assert.Equal(t, []string{"C6"}, codes("a_b"), `"a_b" must not match "abb" in Cabbe`)
		assert.Empty(t, codes("%"), "a lone % is a literal percent sign, not 'everything'")
	})

	t.Run("it searches the full name, so a type prefix finds its wards", func(t *testing.T) {
		assert.Equal(t, []string{"C1"}, codes("phuong hoa thang"), "type prefix + tone-insensitive + case-insensitive")
		assert.Equal(t, []string{"C3"}, codes("PHƯỜNG b'lao"))
	})

	t.Run("the search stays inside the province", func(t *testing.T) {
		got, err := repo.SearchCurrentWards(ctx, "01", "hoa")
		require.NoError(t, err)
		assert.Equal(t, []string{"C8"}, codesOf(got, func(w domain.CurrentWard) string { return w.Code }))
	})

	t.Run("no match is an empty list, an unknown province is not found", func(t *testing.T) {
		got, err := repo.SearchCurrentWards(ctx, "79", "zzzz")
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)

		_, err = repo.SearchCurrentWards(ctx, "no-such", "hoa")
		assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
	})

	t.Run("fields round-trip", func(t *testing.T) {
		got, err := repo.SearchCurrentWards(ctx, "79", "b'lao")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, domain.CurrentWard{
			Code: "C3", ProvinceCode: "79", Name: "B" + curly + "Lao", FullName: "Phường B" + curly + "Lao", Type: "ward", SortOrder: 1,
		}, got[0])
	})
}
