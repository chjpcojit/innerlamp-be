package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func mkDeity(id, typ string, order int) domain.Deity {
	return domain.Deity{
		ID: id, Type: typ, SortOrder: order, NameVI: "Tên " + id,
		LunarMemorialDays: []domain.MemorialDay{{LunarDay: 15, LunarMonth: 4, EventName: "Vía"}},
		Iconography:       domain.Iconography{Posture: "Ngồi", SacredArtifacts: []string{"Bát"}},
		GreatVows:         []string{"v1", "v2"},
		Mantras:           domain.Mantras{ShortSanskrit: "Om"},
	}
}

func newDeityRepo(t *testing.T) domain.DeityRepository {
	t.Helper()
	return NewDeityRepository(connectTestDB(t))
}

func TestDeityRepository_UpsertAndFind(t *testing.T) {
	repo := newDeityRepo(t)
	ctx := context.Background()

	d := mkDeity("d1", domain.DeityBuddha, 0)
	require.NoError(t, repo.Upsert(ctx, []domain.Deity{d}))

	got, err := repo.FindByID(ctx, "d1")
	require.NoError(t, err)
	assert.Equal(t, d, *got)

	// Upsert again with a change: updates in place, no duplicate row.
	d.NameVI = "Renamed"
	d.GreatVows = nil
	d.LunarMemorialDays = nil
	require.NoError(t, repo.Upsert(ctx, []domain.Deity{d}))
	got, err = repo.FindByID(ctx, "d1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.NameVI)
	assert.Empty(t, got.GreatVows)
	assert.Empty(t, got.LunarMemorialDays)
	_, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestDeityRepository_FindByID_NotFound(t *testing.T) {
	repo := newDeityRepo(t)
	_, err := repo.FindByID(context.Background(), "missing")
	assert.ErrorIs(t, err, domain.ErrDeityNotFound)
}

func TestDeityRepository_Upsert_IsAtomic(t *testing.T) {
	repo := newDeityRepo(t)
	ctx := context.Background()

	bad := mkDeity("bad", "NOT_A_TYPE", 1) // violates chk_deities_type
	err := repo.Upsert(ctx, []domain.Deity{mkDeity("ok", domain.DeityBuddha, 0), bad})
	require.Error(t, err)

	_, findErr := repo.FindByID(ctx, "ok")
	assert.ErrorIs(t, findErr, domain.ErrDeityNotFound, "rolled back: earlier good row must not persist")
}

func TestDeityRepository_Ordering(t *testing.T) {
	repo := newDeityRepo(t)
	ctx := context.Background()

	// Inserted out of order; sort_order decides, id breaks the tie.
	require.NoError(t, repo.Upsert(ctx, []domain.Deity{
		mkDeity("c", domain.DeityBodhisattva, 2),
		mkDeity("b2", domain.DeityBodhisattva, 1),
		mkDeity("a", domain.DeityBuddha, 0),
		mkDeity("b1", domain.DeityBuddha, 1),
	}))

	top, err := repo.ListTop(ctx, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b1", "b2"}, deityIDs(top), "top mixes types")
}

func TestDeityRepository_List_Paging(t *testing.T) {
	repo := newDeityRepo(t)
	ctx := context.Background()

	var all []domain.Deity
	all = append(all, mkDeity("buddha", domain.DeityBuddha, 0))
	for i := 1; i <= 5; i++ {
		all = append(all, mkDeity(fmt.Sprintf("bo_%d", i), domain.DeityBodhisattva, i))
	}
	require.NoError(t, repo.Upsert(ctx, all))

	p1, total, err := repo.List(ctx, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 6, total, "total counts every type")
	assert.Equal(t, []string{"buddha", "bo_1"}, deityIDs(p1), "Buddhas and bodhisattvas share one ordered list")

	p3, _, err := repo.List(ctx, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"bo_4", "bo_5"}, deityIDs(p3))

	past, total, err := repo.List(ctx, 100, 2)
	require.NoError(t, err)
	assert.Empty(t, past)
	assert.NotNil(t, past)
	assert.Equal(t, 6, total)
}

func deityIDs(ds []domain.Deity) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.ID)
	}
	return out
}
