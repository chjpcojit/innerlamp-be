package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func mkTemple(id, rank string, year int) domain.Temple {
	prio, ok := domain.HeritagePriority(rank)
	if !ok {
		panic("unknown rank " + rank)
	}
	return domain.Temple{
		ID: id, Name: "Chùa " + id, Type: "CHUA", Region: "north", Sect: "BẮC_TÔNG",
		Established: year, Heritage: rank, HeritagePriority: prio,
		Latitude: 21.0, Longitude: 105.8, Address: "Quận Ba Đình, Hà Nội", Province: "Hà Nội",
		Aliases: []string{"a1", "a2"},
	}
}

func newTempleRepo(t *testing.T) domain.TempleRepository {
	t.Helper()
	return NewTempleRepository(connectTestDB(t))
}

func TestTempleRepository_UpsertAndFind(t *testing.T) {
	repo := newTempleRepo(t)
	ctx := context.Background()

	tp := mkTemple("t1", "DI_TICH_QUOC_GIA", 1000)
	tp.Amenities = domain.TempleAmenities{ParkingCar: true, VegetarianMeals: true}
	tp.OperatingHours = "07:00 - 18:00"
	tp.Introduction = "Giới thiệu"
	tp.ArchitectureHeritage = "Kiến trúc"
	tp.VisitingNotes = []string{"Lưu ý 1", "Lưu ý 2"}
	require.NoError(t, repo.Upsert(ctx, []domain.Temple{tp}))

	got, err := repo.FindByID(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, tp, *got)

	// Upsert again with a change: updates in place, no duplicate row.
	tp.Name = "Renamed"
	tp.Aliases = nil
	require.NoError(t, repo.Upsert(ctx, []domain.Temple{tp}))
	got, err = repo.FindByID(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Name)
	assert.Empty(t, got.Aliases)
	_, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestTempleRepository_FindByID_NotFound(t *testing.T) {
	repo := newTempleRepo(t)
	_, err := repo.FindByID(context.Background(), "missing")
	assert.ErrorIs(t, err, domain.ErrTempleNotFound)
}

func TestTempleRepository_Upsert_IsAtomic(t *testing.T) {
	repo := newTempleRepo(t)
	ctx := context.Background()

	bad := mkTemple("bad", "DI_TICH_QUOC_GIA", 1)
	bad.Type = "NOT_A_TYPE" // violates chk_temples_type
	err := repo.Upsert(ctx, []domain.Temple{mkTemple("ok", "DI_TICH_QUOC_GIA", 1), bad})
	require.Error(t, err)

	_, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total, "first row must roll back with the failing one")
}

func TestTempleRepository_Ordering(t *testing.T) {
	repo := newTempleRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, []domain.Temple{
		mkTemple("d", "CHUA_XEP_HANG", 500),
		mkTemple("b", "DI_TICH_QUOC_GIA_DAC_BIET", 1049),
		mkTemple("a", "DI_TICH_QUOC_GIA_DAC_BIET", 541),
		mkTemple("z", "DI_SAN_THE_GIOI", 1900),
		mkTemple("c", "DI_TICH_QUOC_GIA_DAC_BIET", 1049), // same rank+year as b → id decides
	}))

	top, err := repo.ListTop(ctx, 3)
	require.NoError(t, err)
	ids := []string{}
	for _, x := range top {
		ids = append(ids, x.ID)
	}
	assert.Equal(t, []string{"z", "a", "b"}, ids, "world heritage first, then special national by oldest, id as tiebreak")

	all, _, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, all, 5)
	assert.Equal(t, "d", all[4].ID)
}

func TestTempleRepository_ListPaging(t *testing.T) {
	repo := newTempleRepo(t)
	ctx := context.Background()

	var ts []domain.Temple
	for i := 0; i < 45; i++ {
		ts = append(ts, mkTemple(fmt.Sprintf("t%02d", i), "DI_TICH_QUOC_GIA", 1000+i))
	}
	require.NoError(t, repo.Upsert(ctx, ts))

	seen := map[string]bool{}
	for page, want := range []int{20, 20, 5, 0} {
		items, total, err := repo.List(ctx, page*20, 20)
		require.NoError(t, err)
		assert.Equal(t, 45, total)
		assert.Len(t, items, want, "page %d", page+1)
		for _, it := range items {
			assert.False(t, seen[it.ID], "%s repeated across pages", it.ID)
			seen[it.ID] = true
		}
	}
	assert.Len(t, seen, 45)
}

func TestTempleRepository_ListOptions(t *testing.T) {
	repo := newTempleRepo(t)
	ctx := context.Background()

	named := func(id, rank string, year int, name, province string) domain.Temple {
		x := mkTemple(id, rank, year)
		x.Name, x.Province = name, province
		return x
	}

	t.Run("empty table is an empty, non-nil slice with total 0", func(t *testing.T) {
		got, total, err := repo.ListOptions(ctx, "", 0, 10)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
		assert.Equal(t, 0, total)
	})

	require.NoError(t, repo.Upsert(ctx, []domain.Temple{
		named("d", "CHUA_XEP_HANG", 500, "Chùa Đồng Quang", "Hồ Chí Minh"),
		named("b", "DI_TICH_QUOC_GIA_DAC_BIET", 1049, "Chùa Bích Động", "Ninh Bình"),
		named("a", "DI_TICH_QUOC_GIA_DAC_BIET", 541, "Chùa Một Cột", "Hà Nội"),
		named("z", "DI_SAN_THE_GIOI", 1900, "Chùa 100% Hương_Sơn", "Cần Thơ"),
	}))

	t.Run("returns id, name and the stored province in ranking order, paged, with the full total", func(t *testing.T) {
		page1, total, err := repo.ListOptions(ctx, "", 0, 3)
		require.NoError(t, err)
		assert.Equal(t, 4, total, "total counts the whole table, not the page")
		assert.Equal(t, []domain.TempleOption{
			{ID: "z", Name: "Chùa 100% Hương_Sơn", Province: "Cần Thơ"},
			{ID: "a", Name: "Chùa Một Cột", Province: "Hà Nội"},
			{ID: "b", Name: "Chùa Bích Động", Province: "Ninh Bình"},
		}, page1, "same order as top and the paged list")

		page2, total, err := repo.ListOptions(ctx, "", 3, 3)
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Equal(t, []domain.TempleOption{{ID: "d", Name: "Chùa Đồng Quang", Province: "Hồ Chí Minh"}}, page2)

		past, total, err := repo.ListOptions(ctx, "", 30, 3)
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Empty(t, past)
	})

	t.Run("province is the stored column, not parsed from address", func(t *testing.T) {
		x := named("p", "CHUA_XEP_HANG", 1, "Chùa P", "Tên Lưu Riêng")
		x.Address = "Phường 1, Tỉnh Khác Hẳn"
		require.NoError(t, repo.Upsert(ctx, []domain.Temple{x}))

		got, _, err := repo.ListOptions(ctx, "chua p", 0, 10)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Tên Lưu Riêng", got[0].Province)
	})

	search := func(t *testing.T, q string) ([]string, int) {
		t.Helper()
		got, total, err := repo.ListOptions(ctx, q, 0, 10)
		require.NoError(t, err)
		ids := []string{}
		for _, o := range got {
			ids = append(ids, o.ID)
		}
		return ids, total
	}

	t.Run("matches ignoring case and Vietnamese diacritics", func(t *testing.T) {
		for _, q := range []string{"bich dong", "BÍCH ĐỘNG", "Bích Động", "bích dong", "ich do"} {
			ids, total := search(t, q)
			assert.Equal(t, []string{"b"}, ids, q)
			assert.Equal(t, 1, total, q)
		}
	})

	t.Run("đ and d match each other in both directions", func(t *testing.T) {
		ids, _ := search(t, "dong quang")
		assert.Equal(t, []string{"d"}, ids)
		ids, _ = search(t, "đồng quang")
		assert.Equal(t, []string{"d"}, ids)
	})

	t.Run("total and paging apply to the matches, ranking order is kept", func(t *testing.T) {
		got, total, err := repo.ListOptions(ctx, "chua", 0, 2)
		require.NoError(t, err)
		assert.Equal(t, 5, total, "all five names contain 'chùa'")
		assert.Len(t, got, 2)
		assert.Equal(t, "z", got[0].ID)
	})

	t.Run("LIKE wildcards in the term are matched literally", func(t *testing.T) {
		ids, total := search(t, "100%")
		assert.Equal(t, []string{"z"}, ids)
		assert.Equal(t, 1, total)

		ids, _ = search(t, "huong_son")
		assert.Equal(t, []string{"z"}, ids)

		ids, total = search(t, "%")
		assert.Equal(t, []string{"z"}, ids, "a lone % is a literal percent sign, not 'match everything'")
		assert.Equal(t, 1, total)

		ids, _ = search(t, "_")
		assert.Equal(t, []string{"z"}, ids, "a lone _ is a literal underscore")

		ids, _ = search(t, `\`)
		assert.Empty(t, ids)
	})

	t.Run("no match is an empty non-nil slice with total 0", func(t *testing.T) {
		got, total, err := repo.ListOptions(ctx, "khong co chua nay", 0, 10)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
		assert.Equal(t, 0, total)
	})
}

// TestTempleRepository_AddressCodes covers spec temple v1.7 (AC 32, 37): the
// structured address round-trips; a temple without codes reads back "" (stored
// as NULL); ListOptions carries province_code; and the all-or-none CHECK
// refuses a half-filled address.
func TestTempleRepository_AddressCodes(t *testing.T) {
	repo := newTempleRepo(t)
	ctx := context.Background()

	withCodes := mkTemple("c1", "DI_TICH_QUOC_GIA", 1000)
	withCodes.Address = "73 Mai Thị Lựu"
	withCodes.Province = "Hồ Chí Minh"
	withCodes.ProvinceCode = "79"
	withCodes.WardCode = "26737"
	withCodes.Ward = "Phường Tân Định"
	withCodes.VisitingNotes = []string{}               // the repository reads an empty array, never nil
	legacy := mkTemple("c2", "DI_TICH_QUOC_GIA", 1100) // no codes: not re-seeded yet
	require.NoError(t, repo.Upsert(ctx, []domain.Temple{withCodes, legacy}))

	got, err := repo.FindByID(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, withCodes, *got)
	assert.Equal(t, "73 Mai Thị Lựu, Phường Tân Định, Hồ Chí Minh", got.FullAddress())

	got, err = repo.FindByID(ctx, "c2")
	require.NoError(t, err)
	assert.Empty(t, got.ProvinceCode)
	assert.Empty(t, got.WardCode)
	assert.Empty(t, got.Ward)
	assert.Equal(t, legacy.Address, got.FullAddress())

	opts, _, err := repo.ListOptions(ctx, "", 0, 10)
	require.NoError(t, err)
	require.Len(t, opts, 2)
	assert.Equal(t, domain.TempleOption{ID: "c1", Name: "Chùa c1", Province: "Hồ Chí Minh", ProvinceCode: "79"}, opts[0])
	assert.Empty(t, opts[1].ProvinceCode)

	// Re-seeding a temple back to no codes clears them (upsert overwrites every column).
	withCodes.ProvinceCode, withCodes.WardCode, withCodes.Ward = "", "", ""
	require.NoError(t, repo.Upsert(ctx, []domain.Temple{withCodes}))
	got, err = repo.FindByID(ctx, "c1")
	require.NoError(t, err)
	assert.Empty(t, got.ProvinceCode)

	// Half-filled address is refused by chk_temples_address_codes and nothing is written.
	half := mkTemple("c3", "DI_TICH_QUOC_GIA", 1200)
	half.ProvinceCode = "79"
	require.Error(t, repo.Upsert(ctx, []domain.Temple{half}))
	_, err = repo.FindByID(ctx, "c3")
	assert.ErrorIs(t, err, domain.ErrTempleNotFound)
}
