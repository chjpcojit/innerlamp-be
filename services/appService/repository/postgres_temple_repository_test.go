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
		Latitude: 21.0, Longitude: 105.8, Address: "Hà Nội",
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
