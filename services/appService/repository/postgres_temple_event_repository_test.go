package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var teRepoZone = time.FixedZone("ICT", 7*3600)

// newTempleEventRepo returns a repository over empty tables with temples t1 and
// t2 present. connectTestDB truncates many tables: run with
// TEST_DB_NAME=innerlamp_test, never against the dev database.
func newTempleEventRepo(t *testing.T) (domain.TempleEventRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	require.NoError(t, NewTempleRepository(db).Upsert(context.Background(), []domain.Temple{
		mkTemple("t1", "DI_TICH_QUOC_GIA", 1000),
		mkTemple("t2", "DI_TICH_QUOC_GIA", 1100),
	}))
	return NewTempleEventRepository(db), db
}

func teWindow(from, to time.Time) domain.TempleEventWindow {
	return domain.TempleEventWindow{From: from, To: to}
}

func teDate(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func teAt(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, teRepoZone) }

func templeEventCatalogue() domain.TempleEventCatalogue {
	until := teDate(2027, 6, 30)
	sunday, image := 0, "https://cdn.example/a.jpg"
	return domain.TempleEventCatalogue{Source: "mock", Events: []domain.TempleEvent{
		{ID: "one", TempleID: "t1", Title: "Khoá tu", Description: "d", Kind: domain.TempleEventKhoaTu,
			StartsAt: teAt(2026, 11, 7, 6), EndsAt: teAt(2026, 11, 8, 17), Status: domain.TempleEventScheduled,
			AudienceNote: "người lớn", ContactInfo: "0123", ImageURL: &image, Recurrence: domain.RecurrenceNone},
		{ID: "weekly", TempleID: "t1", Title: "Lễ Dược Sư", Kind: domain.TempleEventLeCauAn,
			StartsAt: teAt(2026, 10, 4, 8), EndsAt: teAt(2026, 10, 4, 11), Status: domain.TempleEventScheduled,
			Recurrence: domain.RecurrenceWeekly, Weekday: &sunday, Until: &until},
		{ID: "lunar", TempleID: "t1", Title: "Sám hối", Kind: domain.TempleEventKhac,
			StartsAt: teAt(2026, 10, 1, 19), EndsAt: teAt(2026, 10, 1, 21), Status: domain.TempleEventPostponed,
			Recurrence: domain.RecurrenceLunarMonthly, LunarDays: []int{14, 30}},
		{ID: "allday", TempleID: "t1", Title: "Cả ngày", Kind: domain.TempleEventLeHoi, IsAllDay: true,
			StartsAt: teAt(2026, 12, 1, 0), EndsAt: teAt(2026, 12, 3, 0), Status: domain.TempleEventCancelled,
			Recurrence: domain.RecurrenceNone},
		{ID: "other", TempleID: "t2", Title: "Chùa khác", Kind: domain.TempleEventKhac,
			StartsAt: teAt(2026, 11, 9, 8), EndsAt: teAt(2026, 11, 9, 9), Status: domain.TempleEventScheduled,
			Recurrence: domain.RecurrenceNone},
	}}
}

func byID(events []domain.TempleEvent) map[string]domain.TempleEvent {
	m := map[string]domain.TempleEvent{}
	for _, e := range events {
		m[e.ID] = e
	}
	return m
}

func TestTempleEventReplaceAndList(t *testing.T) {
	repo, _ := newTempleEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, templeEventCatalogue()))

	got, err := repo.ListActiveByTemple(ctx, "t1", teWindow(teDate(2026, 10, 13), teDate(2027, 1, 13)))
	require.NoError(t, err)
	m := byID(got)
	require.Len(t, m, 4, "only temple t1")

	one := m["one"]
	assert.True(t, teAt(2026, 11, 7, 6).Equal(one.StartsAt))
	assert.True(t, teAt(2026, 11, 8, 17).Equal(one.EndsAt))
	assert.Equal(t, domain.TempleEventKhoaTu, one.Kind)
	assert.Equal(t, domain.TempleEventScheduled, one.Status)
	assert.Equal(t, "người lớn", one.AudienceNote)
	assert.Equal(t, "0123", one.ContactInfo)
	require.NotNil(t, one.ImageURL)
	assert.Equal(t, "https://cdn.example/a.jpg", *one.ImageURL)
	assert.Equal(t, domain.RecurrenceNone, one.Recurrence)
	assert.Empty(t, one.LunarDays)
	assert.Nil(t, one.Weekday)
	assert.Nil(t, one.Until)

	weekly := m["weekly"]
	require.NotNil(t, weekly.Weekday)
	assert.Equal(t, 0, *weekly.Weekday)
	require.NotNil(t, weekly.Until)
	assert.Equal(t, teDate(2027, 6, 30), *weekly.Until)

	lunar := m["lunar"]
	assert.Equal(t, []int{14, 30}, lunar.LunarDays)
	assert.Equal(t, domain.TempleEventPostponed, lunar.Status)
	assert.Nil(t, lunar.ImageURL)

	assert.True(t, m["allday"].IsAllDay)
	assert.Equal(t, domain.TempleEventCancelled, m["allday"].Status)
}

func TestTempleEventListFilters(t *testing.T) {
	repo, db := newTempleEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, templeEventCatalogue()))
	ids := func(w domain.TempleEventWindow) []string {
		got, err := repo.ListActiveByTemple(ctx, "t1", w)
		require.NoError(t, err)
		out := []string{}
		for _, e := range got {
			out = append(out, e.ID)
		}
		return out
	}

	t.Run("one-off outside the window is left out, recurring kept", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"weekly", "lunar"}, ids(teWindow(teDate(2026, 10, 13), teDate(2026, 10, 31))))
	})
	t.Run("one-off that ended before the window is left out", func(t *testing.T) {
		assert.NotContains(t, ids(teWindow(teDate(2026, 12, 20), teDate(2027, 1, 13))), "one")
	})
	t.Run("recurring event whose until is before the window is left out", func(t *testing.T) {
		assert.NotContains(t, ids(teWindow(teDate(2027, 8, 1), teDate(2027, 9, 1))), "weekly")
	})
	t.Run("recurring event that starts after the window is left out", func(t *testing.T) {
		assert.NotContains(t, ids(teWindow(teDate(2026, 9, 1), teDate(2026, 9, 20))), "weekly")
	})
	t.Run("inactive events are left out", func(t *testing.T) {
		_, err := db.Exec(ctx, `UPDATE temple_events SET is_active = false WHERE id = 'lunar'`)
		require.NoError(t, err)
		assert.NotContains(t, ids(teWindow(teDate(2026, 10, 13), teDate(2027, 1, 13))), "lunar")
	})
	t.Run("unknown temple returns an empty non-nil slice", func(t *testing.T) {
		got, err := repo.ListActiveByTemple(ctx, "nope", teWindow(teDate(2026, 10, 13), teDate(2027, 1, 13)))
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestTempleEventReplaceIsIdempotentAndKeepsIsActive(t *testing.T) {
	repo, db := newTempleEventRepo(t)
	ctx := context.Background()
	c := templeEventCatalogue()
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))
	_, err := db.Exec(ctx, `UPDATE temple_events SET is_active = false WHERE id = 'one'`)
	require.NoError(t, err)

	c.Events[0].Title = "Khoá tu (đổi tên)"
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_events`).Scan(&n))
	assert.Equal(t, 5, n)
	var active bool
	var title string
	require.NoError(t, db.QueryRow(ctx, `SELECT is_active, title FROM temple_events WHERE id = 'one'`).Scan(&active, &title))
	assert.False(t, active)
	assert.Equal(t, "Khoá tu (đổi tên)", title)
}

func TestTempleEventReplaceUnknownTempleRollsBack(t *testing.T) {
	repo, db := newTempleEventRepo(t)
	ctx := context.Background()
	c := templeEventCatalogue()
	c.Events = append(c.Events, domain.TempleEvent{ID: "bad", TempleID: "ghost", Title: "x", Kind: domain.TempleEventKhac,
		StartsAt: teAt(2026, 11, 1, 8), EndsAt: teAt(2026, 11, 1, 9), Status: domain.TempleEventScheduled,
		Recurrence: domain.RecurrenceNone})

	err := repo.ReplaceCatalogue(ctx, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_events`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestTempleEventConstraintsRejectInconsistentRows(t *testing.T) {
	repo, db := newTempleEventRepo(t)
	ctx := context.Background()
	monday := 1
	cases := map[string]domain.TempleEvent{
		"weekly without weekday": {Recurrence: domain.RecurrenceWeekly},
		"NONE with a weekday":    {Recurrence: domain.RecurrenceNone, Weekday: &monday},
		"lunar monthly no days":  {Recurrence: domain.RecurrenceLunarMonthly},
		"lunar day 31":           {Recurrence: domain.RecurrenceLunarMonthly, LunarDays: []int{31}},
		"ends before starts": {Recurrence: domain.RecurrenceNone,
			StartsAt: teAt(2026, 11, 1, 9), EndsAt: teAt(2026, 11, 1, 8)},
		"unknown kind": {Recurrence: domain.RecurrenceNone, Kind: "X"},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			e.ID, e.TempleID, e.Title = "x", "t1", "x"
			if e.Kind == "" {
				e.Kind = domain.TempleEventKhac
			}
			e.Status = domain.TempleEventScheduled
			if e.StartsAt.IsZero() {
				e.StartsAt, e.EndsAt = teAt(2026, 11, 1, 8), teAt(2026, 11, 1, 9)
			}
			err := repo.ReplaceCatalogue(ctx, domain.TempleEventCatalogue{Source: "m", Events: []domain.TempleEvent{e}})
			assert.Error(t, err)
		})
	}
	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_events`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestTempleEventDeletingTempleCascades(t *testing.T) {
	repo, db := newTempleEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, templeEventCatalogue()))
	_, err := db.Exec(ctx, `DELETE FROM temples WHERE id = 't1'`)
	require.NoError(t, err)
	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM temple_events WHERE temple_id = 't1'`).Scan(&n))
	assert.Equal(t, 0, n)
}
