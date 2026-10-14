package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newCalendarEventRepo returns a repository over empty calendar_* tables. It
// clears only those tables, but connectTestDB truncates others: run with
// TEST_DB_NAME=innerlamp_test, never against the dev database.
func newCalendarEventRepo(t *testing.T) (domain.CalendarEventRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	_, err := db.Exec(context.Background(),
		`TRUNCATE TABLE calendar_event_groups, calendar_event_translations, calendar_events, calendar_groups`)
	require.NoError(t, err)
	return NewCalendarEventRepository(db), db
}

// calendarCatalogue has one event of each kind. Group sort_order is the reverse
// of code order, so a passing group-order test proves ordering by sort_order.
func calendarCatalogue() domain.CalendarCatalogue {
	return domain.CalendarCatalogue{
		Lang: "vi",
		Groups: []domain.CalendarGroup{
			{Code: "buddhist", Name: "Phật giáo", SortOrder: 2},
			{Code: "folk", Name: "Dân gian", SortOrder: 1},
			{Code: "vietnam", Name: "Việt Nam", SortOrder: 3},
		},
		Events: []domain.CalendarEvent{
			{Code: "giai-phong", Name: "Giải phóng miền Nam", ShortDescription: "s1", LongDescription: "l1", Guidance: "g1",
				Groups: []string{"vietnam"}, IsPublicHoliday: true, SortOrder: 2,
				Rule: domain.RecurrenceRule{Kind: domain.KindSolar, SolarMonth: 4, SolarDay: 30}},
			{Code: "phat-dan", Name: "Lễ Phật đản", ShortDescription: "s2", LongDescription: "l2", Guidance: "g2",
				Groups: []string{"vietnam", "buddhist", "folk"}, SortOrder: 1, ImageURL: calendarStr("https://cdn.example/phat-dan.jpg"),
				Rule: domain.RecurrenceRule{Kind: domain.KindLunar, LunarMonth: 4, LunarDay: 15}},
			{Code: "ram", Name: "Rằm", ShortDescription: "s3", LongDescription: "l3", Guidance: "g3",
				Groups: []string{"buddhist"}, SortOrder: 3,
				Rule: domain.RecurrenceRule{Kind: domain.KindLunarMonthly, LunarDay: 15}},
			{Code: "giao-thua", Name: "Giao thừa", ShortDescription: "s4", LongDescription: "l4", Guidance: "g4",
				Groups: []string{"folk"}, SortOrder: 4,
				Rule: domain.RecurrenceRule{Kind: domain.KindLunarLastDay, LunarMonth: 12}},
			{Code: "xuan-phan", Name: "Xuân phân", ShortDescription: "s5", LongDescription: "l5", Guidance: "g5",
				Groups: []string{"folk"}, SortOrder: 5,
				Rule: domain.RecurrenceRule{Kind: domain.KindSolarTerm, SolarTermLongitude: 0}},
		},
	}
}

func TestCalendarReplaceAndList(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))

	groups, err := repo.ListGroups(ctx)
	require.NoError(t, err)
	assert.Equal(t, []domain.CalendarGroup{
		{Code: "folk", Name: "Dân gian", SortOrder: 1},
		{Code: "buddhist", Name: "Phật giáo", SortOrder: 2},
		{Code: "vietnam", Name: "Việt Nam", SortOrder: 3},
	}, groups)

	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	require.Len(t, events, 5)

	// ordered by sort_order, then code
	var codes []string
	for _, e := range events {
		codes = append(codes, e.Code)
	}
	assert.Equal(t, []string{"phat-dan", "giai-phong", "ram", "giao-thua", "xuan-phan"}, codes)

	byCode := map[string]domain.CalendarEvent{}
	for _, e := range events {
		byCode[e.Code] = e
	}
	pd := byCode["phat-dan"]
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindLunar, LunarMonth: 4, LunarDay: 15}, pd.Rule)
	assert.Equal(t, []string{"folk", "buddhist", "vietnam"}, pd.Groups, "groups ordered by the group's sort_order")
	require.NotNil(t, pd.ImageURL)
	assert.Equal(t, "https://cdn.example/phat-dan.jpg", *pd.ImageURL)
	assert.Equal(t, "Lễ Phật đản", pd.Name)
	assert.Equal(t, "g2", pd.Guidance)
	assert.False(t, pd.IsPublicHoliday)

	assert.True(t, byCode["giai-phong"].IsPublicHoliday)
	assert.Nil(t, byCode["giai-phong"].ImageURL)
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindSolar, SolarMonth: 4, SolarDay: 30}, byCode["giai-phong"].Rule)
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindLunarMonthly, LunarDay: 15}, byCode["ram"].Rule)
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindLunarLastDay, LunarMonth: 12}, byCode["giao-thua"].Rule)
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindSolarTerm, SolarTermLongitude: 0}, byCode["xuan-phan"].Rule,
		"0° is a valid longitude and must round-trip as a solar term")
}

func TestCalendarReplaceIsIdempotent(t *testing.T) {
	repo, db := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))

	for table, want := range map[string]int{
		"calendar_groups": 3, "calendar_events": 5, "calendar_event_translations": 5, "calendar_event_groups": 7,
	} {
		var n int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n))
		assert.Equal(t, want, n, table)
	}
}

func TestCalendarReplaceOverwritesContent(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))

	c := calendarCatalogue()
	c.Events[1].Name = "Phật đản (sửa)"
	c.Events[1].Guidance = "mới"
	c.Events[1].ImageURL = nil
	c.Events[1].IsPublicHoliday = true
	c.Events[1].Groups = []string{"buddhist"} // two links removed
	c.Events[1].Rule = domain.RecurrenceRule{Kind: domain.KindLunar, LunarMonth: 4, LunarDay: 8}
	c.Groups[0].Name = "Phật giáo (sửa)"
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	var pd domain.CalendarEvent
	for _, e := range events {
		if e.Code == "phat-dan" {
			pd = e
		}
	}
	assert.Equal(t, "Phật đản (sửa)", pd.Name)
	assert.Equal(t, "mới", pd.Guidance)
	assert.Nil(t, pd.ImageURL)
	assert.True(t, pd.IsPublicHoliday)
	assert.Equal(t, []string{"buddhist"}, pd.Groups)
	assert.Equal(t, 8, pd.Rule.LunarDay)

	groups, err := repo.ListGroups(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Phật giáo (sửa)", groups[1].Name)
}

func TestCalendarInactiveEventNotListed(t *testing.T) {
	repo, db := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))
	_, err := db.Exec(ctx, `UPDATE calendar_events SET is_active = false WHERE code = 'ram'`)
	require.NoError(t, err)

	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	assert.Len(t, events, 4)
	for _, e := range events {
		assert.NotEqual(t, "ram", e.Code)
	}

	// Re-running the seed must not switch back-office's decision off again.
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))
	events, err = repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	assert.Len(t, events, 4)
}

func TestCalendarEventAbsentFromFileIsKept(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))

	c := calendarCatalogue()
	c.Events = c.Events[:1] // only giai-phong
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	assert.Len(t, events, 5)
}

func TestCalendarLanguageFilter(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))

	events, err := repo.ListActiveEvents(ctx, "en")
	require.NoError(t, err)
	assert.Empty(t, events)

	// A second language adds text without touching the events.
	en := calendarCatalogue()
	en.Lang = "en"
	for i := range en.Events {
		en.Events[i].Name = "EN " + en.Events[i].Code
	}
	require.NoError(t, repo.ReplaceCatalogue(ctx, en))
	events, err = repo.ListActiveEvents(ctx, "en")
	require.NoError(t, err)
	require.Len(t, events, 5)
	assert.Equal(t, "EN phat-dan", events[0].Name)
	vi, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	assert.Equal(t, "Lễ Phật đản", vi[0].Name)
}

func TestCalendarEventWithoutGroupsHasEmptySlice(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	c := calendarCatalogue()
	c.Events[0].Groups = nil
	require.NoError(t, repo.ReplaceCatalogue(ctx, c))

	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	for _, e := range events {
		require.NotNil(t, e.Groups, e.Code)
		if e.Code == "giai-phong" {
			assert.Empty(t, e.Groups)
		}
	}
}

// A failure part-way (an event naming a group that does not exist) must leave
// nothing behind, including the groups and events written before it.
func TestCalendarReplaceRollsBackOnError(t *testing.T) {
	repo, db := newCalendarEventRepo(t)
	ctx := context.Background()

	c := calendarCatalogue()
	c.Events[4].Groups = []string{"no-such-group"}
	err := repo.ReplaceCatalogue(ctx, c)
	require.Error(t, err)

	for _, table := range []string{"calendar_groups", "calendar_events", "calendar_event_translations", "calendar_event_groups"} {
		var n int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n))
		assert.Zero(t, n, table)
	}
}

func TestCalendarReplaceRollbackKeepsExistingData(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.ReplaceCatalogue(ctx, calendarCatalogue()))

	bad := calendarCatalogue()
	bad.Events[0].Name = "SHOULD NOT PERSIST"
	bad.Events[4].Groups = []string{"no-such-group"}
	require.Error(t, repo.ReplaceCatalogue(ctx, bad))

	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	for _, e := range events {
		assert.NotEqual(t, "SHOULD NOT PERSIST", e.Name)
	}
	assert.Len(t, events, 5)
}

func TestCalendarReplaceRequiresLang(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	c := calendarCatalogue()
	c.Lang = ""
	assert.Error(t, repo.ReplaceCatalogue(context.Background(), c))
}

func TestCalendarListOnEmptyDatabase(t *testing.T) {
	repo, _ := newCalendarEventRepo(t)
	ctx := context.Background()
	groups, err := repo.ListGroups(ctx)
	require.NoError(t, err)
	assert.Empty(t, groups)
	events, err := repo.ListActiveEvents(ctx, "vi")
	require.NoError(t, err)
	assert.Empty(t, events)
}

func calendarStr(s string) *string { return &s }
