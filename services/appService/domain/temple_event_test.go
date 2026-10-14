package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEvent() TempleEvent {
	return evt("e1", at(2026, 11, 1, 8, 0), at(2026, 11, 1, 10, 0))
}

func TestTempleEventCatalogueVerify(t *testing.T) {
	t.Run("valid, including recurring kinds and multi-day all-day", func(t *testing.T) {
		weekly := validEvent()
		weekly.ID = "w"
		weekly.Recurrence = RecurrenceWeekly
		weekly.Weekday = intp(0)
		lunar := validEvent()
		lunar.ID = "l"
		lunar.Recurrence = RecurrenceLunarMonthly
		lunar.LunarDays = []int{14, 30}
		retreat := evt("r", at(2026, 12, 1, 0, 0), at(2026, 12, 3, 0, 0))
		retreat.IsAllDay = true
		night := evt("n", at(2026, 11, 7, 22, 0), at(2026, 11, 8, 2, 0))
		c := TempleEventCatalogue{Source: "mock", Events: []TempleEvent{validEvent(), weekly, lunar, retreat, night}}
		assert.NoError(t, c.Verify(vnZone))
	})

	mut := func(f func(*TempleEvent)) TempleEventCatalogue {
		e := validEvent()
		f(&e)
		return TempleEventCatalogue{Source: "mock", Events: []TempleEvent{e}}
	}
	until := func(y int, m time.Month, d int) *time.Time { v := day(y, m, d); return &v }

	tests := []struct {
		name string
		c    TempleEventCatalogue
		want string
	}{
		{"no source", TempleEventCatalogue{Events: []TempleEvent{validEvent()}}, "source is empty"},
		{"no events", TempleEventCatalogue{Source: "m"}, "no events"},
		{"duplicate id", TempleEventCatalogue{Source: "m", Events: []TempleEvent{validEvent(), validEvent()}}, "duplicate id"},
		{"empty id", mut(func(e *TempleEvent) { e.ID = "" }), "id is empty"},
		{"empty temple", mut(func(e *TempleEvent) { e.TempleID = "" }), "temple_id is empty"},
		{"blank title", mut(func(e *TempleEvent) { e.Title = " " }), "title is empty"},
		{"unknown kind", mut(func(e *TempleEvent) { e.Kind = "X" }), `kind "X"`},
		{"unknown status", mut(func(e *TempleEvent) { e.Status = "X" }), `status "X"`},
		{"unknown recurrence", mut(func(e *TempleEvent) { e.Recurrence = "DAILY" }), `recurrence "DAILY"`},
		{"ends before starts", mut(func(e *TempleEvent) { e.EndsAt = e.StartsAt.Add(-time.Hour) }), "ends_at is before starts_at"},
		{"year out of range", mut(func(e *TempleEvent) {
			e.StartsAt, e.EndsAt = at(2101, 1, 1, 8, 0), at(2101, 1, 1, 9, 0)
		}), "outside 1900..2100"},
		{"NONE with lunar days", mut(func(e *TempleEvent) { e.LunarDays = []int{1} }), "NONE takes no"},
		{"NONE with until", mut(func(e *TempleEvent) { e.Until = until(2026, 12, 1) }), "NONE takes no"},
		{"lunar monthly without days", mut(func(e *TempleEvent) { e.Recurrence = RecurrenceLunarMonthly }), "needs lunar_days"},
		{"lunar day 31", mut(func(e *TempleEvent) { e.Recurrence = RecurrenceLunarMonthly; e.LunarDays = []int{31} }), "outside 1..30"},
		{"lunar monthly with weekday", mut(func(e *TempleEvent) {
			e.Recurrence = RecurrenceLunarMonthly
			e.LunarDays = []int{1}
			e.Weekday = intp(1)
		}), "takes no weekday"},
		{"weekly without weekday", mut(func(e *TempleEvent) { e.Recurrence = RecurrenceWeekly }), "needs weekday"},
		{"weekly weekday 7", mut(func(e *TempleEvent) { e.Recurrence = RecurrenceWeekly; e.Weekday = intp(7) }), "needs weekday"},
		{"weekly with lunar days", mut(func(e *TempleEvent) {
			e.Recurrence = RecurrenceWeekly
			e.Weekday = intp(1)
			e.LunarDays = []int{1}
		}), "takes no lunar_days"},
		{"until before first date", mut(func(e *TempleEvent) {
			e.Recurrence = RecurrenceWeekly
			e.Weekday = intp(0)
			e.Until = until(2026, 10, 1)
		}), "until is before"},
		{"recurring lasting a day or more", mut(func(e *TempleEvent) {
			e.Recurrence = RecurrenceWeekly
			e.Weekday = intp(0)
			e.EndsAt = e.StartsAt.Add(24 * time.Hour)
		}), "less than 24 hours"},
		{"all-day not at midnight", mut(func(e *TempleEvent) { e.IsAllDay = true }), "00:00"},
		{"all-day weekly lasting several days", mut(func(e *TempleEvent) {
			e.IsAllDay = true
			e.StartsAt, e.EndsAt = at(2026, 11, 1, 0, 0), at(2026, 11, 3, 0, 0)
			e.Recurrence = RecurrenceWeekly
			e.Weekday = intp(0)
		}), "one day"},
		{"empty image url", mut(func(e *TempleEvent) { s := ""; e.ImageURL = &s }), "image_url"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Verify(vnZone)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidTempleEvent)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestInvalidDateFieldError(t *testing.T) {
	err := error(&InvalidDateFieldError{Field: "from"})
	assert.ErrorIs(t, err, ErrInvalidDate)
	assert.Contains(t, err.Error(), "from")
}
