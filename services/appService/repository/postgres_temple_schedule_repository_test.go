package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newTempleScheduleRepo returns a repository over empty tables with temples t1
// and t2 present. connectTestDB truncates many tables: run with
// TEST_DB_NAME=innerlamp_test, never against the dev database.
func newTempleScheduleRepo(t *testing.T) (domain.TempleScheduleRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	require.NoError(t, NewTempleRepository(db).Upsert(context.Background(), []domain.Temple{
		mkTemple("t1", "DI_TICH_QUOC_GIA", 1000),
		mkTemple("t2", "DI_TICH_QUOC_GIA", 1100),
	}))
	return NewTempleScheduleRepository(db), db
}

func schedTime(minutes int) *domain.TimeOfDay { v := domain.TimeOfDay(minutes); return &v }

func scheduleCatalogue() domain.TempleScheduleCatalogue {
	return domain.TempleScheduleCatalogue{Source: "mock", Items: []domain.TempleScheduleItem{
		{ID: "s_full", TempleID: "t1", Name: "Lễ Dược Sư", StartTime: 8 * 60, EndTime: schedTime(11 * 60),
			Weekdays: []int{0}, Description: "Cầu an", Instructor: "Thượng toạ A", Location: "Chánh điện",
			Preparation: "Áo tràng", SortOrder: 2,
			Steps: []domain.TempleScheduleStep{
				{Position: 2, Time: schedTime(8*60 + 30), Text: "Tụng kinh"},
				{Position: 1, Time: schedTime(8 * 60), Text: "Niệm Phật"},
				{Position: 3, Text: "Hồi hướng"},
			}},
		{ID: "s_min", TempleID: "t1", Name: "Công phu sáng", StartTime: 4 * 60},
		{ID: "s_night", TempleID: "t1", Name: "Toạ thiền", StartTime: 23 * 60, EndTime: schedTime(60)},
		{ID: "s_other", TempleID: "t2", Name: "Chùa khác", StartTime: 5 * 60},
	}}
}

func TestTempleScheduleReplaceAndList(t *testing.T) {
	repo, _ := newTempleScheduleRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, scheduleCatalogue()))

	got, err := repo.ListActiveByTemple(ctx, "t1")
	require.NoError(t, err)
	require.Len(t, got, 3)
	// SQL order is start_time ASC (04:00, 08:00, 23:00).
	assert.Equal(t, []string{"s_min", "s_full", "s_night"}, []string{got[0].ID, got[1].ID, got[2].ID})

	min := got[0]
	assert.Nil(t, min.EndTime)
	assert.Empty(t, min.Weekdays)
	assert.Empty(t, min.Steps)
	assert.NotNil(t, min.Steps)
	assert.Equal(t, "", min.Instructor)

	full := got[1]
	assert.Equal(t, domain.TimeOfDay(480), full.StartTime)
	require.NotNil(t, full.EndTime)
	assert.Equal(t, domain.TimeOfDay(660), *full.EndTime)
	assert.Equal(t, []int{0}, full.Weekdays)
	assert.Equal(t, "Thượng toạ A", full.Instructor)
	assert.Equal(t, "Chánh điện", full.Location)
	assert.Equal(t, "Áo tràng", full.Preparation)
	assert.Equal(t, 2, full.SortOrder)
	require.Len(t, full.Steps, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{full.Steps[0].Position, full.Steps[1].Position, full.Steps[2].Position})
	assert.Equal(t, "Niệm Phật", full.Steps[0].Text)
	require.NotNil(t, full.Steps[0].Time)
	assert.Equal(t, domain.TimeOfDay(480), *full.Steps[0].Time)
	assert.Nil(t, full.Steps[2].Time)

	night := got[2]
	require.NotNil(t, night.EndTime)
	assert.Equal(t, domain.TimeOfDay(60), *night.EndTime) // earlier than start: crosses midnight
}

func TestTempleScheduleListFiltersByTempleAndActive(t *testing.T) {
	repo, db := newTempleScheduleRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, scheduleCatalogue()))

	_, err := db.Exec(ctx, `UPDATE temple_schedules SET is_active = false WHERE id = 's_full'`)
	require.NoError(t, err)

	got, err := repo.ListActiveByTemple(ctx, "t1")
	require.NoError(t, err)
	ids := []string{}
	for _, it := range got {
		ids = append(ids, it.ID)
	}
	assert.Equal(t, []string{"s_min", "s_night"}, ids)

	other, err := repo.ListActiveByTemple(ctx, "t2")
	require.NoError(t, err)
	require.Len(t, other, 1)
	assert.Equal(t, "s_other", other[0].ID)

	none, err := repo.ListActiveByTemple(ctx, "nope")
	require.NoError(t, err)
	assert.NotNil(t, none)
	assert.Empty(t, none)
}

func TestTempleScheduleReplaceIsIdempotentAndKeepsIsActive(t *testing.T) {
	repo, db := newTempleScheduleRepo(t)
	ctx := context.Background()
	c := scheduleCatalogue()
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	_, err := db.Exec(ctx, `UPDATE temple_schedules SET is_active = false WHERE id = 's_min'`)
	require.NoError(t, err)

	// Same file again, with one item renamed: updates in place, no duplicate,
	// and the hand-disabled item stays disabled.
	c.Items[1].Name = "Công phu sáng (đổi tên)"
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedules`).Scan(&n))
	assert.Equal(t, 4, n)
	var active bool
	var name string
	require.NoError(t, db.QueryRow(ctx, `SELECT is_active, name FROM temple_schedules WHERE id = 's_min'`).Scan(&active, &name))
	assert.False(t, active)
	assert.Equal(t, "Công phu sáng (đổi tên)", name)
}

func TestTempleScheduleReplaceReplacesSteps(t *testing.T) {
	repo, db := newTempleScheduleRepo(t)
	ctx := context.Background()
	c := scheduleCatalogue()
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	c.Items[0].Steps = []domain.TempleScheduleStep{{Position: 1, Text: "Chỉ còn một bước"}}
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedule_steps WHERE schedule_id = 's_full'`).Scan(&n))
	assert.Equal(t, 1, n)

	// A step list emptied in the file removes the steps.
	c.Items[0].Steps = nil
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedule_steps WHERE schedule_id = 's_full'`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestTempleScheduleReplaceUnknownTempleRollsBackEverything(t *testing.T) {
	repo, db := newTempleScheduleRepo(t)
	ctx := context.Background()
	c := scheduleCatalogue()
	c.Items = append(c.Items, domain.TempleScheduleItem{ID: "s_bad", TempleID: "ghost", Name: "x", StartTime: 60})

	err := repo.ReplaceCatalogue(ctx, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s_bad")

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedules`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestTempleScheduleReplaceDBConstraintFailureRollsBack(t *testing.T) {
	repo, db := newTempleScheduleRepo(t)
	ctx := context.Background()
	c := scheduleCatalogue()
	// Unverified catalogue: end == start violates chk_temple_schedules_end.
	c.Items[3].EndTime = &c.Items[3].StartTime

	require.Error(t, repo.ReplaceCatalogue(ctx, c))

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedules`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestTempleScheduleDeletingTempleCascades(t *testing.T) {
	repo, db := newTempleScheduleRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, scheduleCatalogue()))

	_, err := db.Exec(ctx, `DELETE FROM temples WHERE id = 't1'`)
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedules WHERE temple_id = 't1'`).Scan(&n))
	assert.Equal(t, 0, n)
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_schedule_steps`).Scan(&n))
	assert.Equal(t, 0, n)
}
