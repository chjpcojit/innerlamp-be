package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain/lunarcal"
)

var vnZone = time.FixedZone("ICT", 7*3600)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, vnZone)
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func win(from, to time.Time) TempleEventWindow { return TempleEventWindow{From: from, To: to} }

func evt(id string, start, end time.Time) TempleEvent {
	return TempleEvent{
		ID: id, TempleID: "t", Title: id, Kind: TempleEventKhac, Status: TempleEventScheduled,
		StartsAt: start, EndsAt: end, Recurrence: RecurrenceNone,
	}
}

func intp(v int) *int { return &v }

func templeDates(occ []TempleEventOccurrence) []string {
	out := make([]string, len(occ))
	for i, o := range occ {
		out[i] = o.SolarDate.Format("2006-01-02")
	}
	return out
}

// 2026-10-13 is a Tuesday.
var nowFixed = at(2026, time.October, 13, 9, 0)

func TestResolve_OneOff(t *testing.T) {
	w := win(day(2026, 10, 13), day(2027, 1, 13))

	t.Run("single occurrence with its lunar date", func(t *testing.T) {
		e := evt("a", at(2026, 10, 20, 19, 0), at(2026, 10, 20, 21, 0))
		got := ResolveTempleOccurrences([]TempleEvent{e}, w, nowFixed, vnZone)
		require.Len(t, got, 1)
		assert.Equal(t, "a", got[0].EventID)
		assert.True(t, at(2026, 10, 20, 19, 0).Equal(got[0].StartsAt))
		assert.True(t, at(2026, 10, 20, 21, 0).Equal(got[0].EndsAt))
		assert.Equal(t, day(2026, 10, 20), got[0].SolarDate)
		want := lunarcal.SolarToLunar(day(2026, 10, 20))
		assert.Equal(t, CalendarLunarDate{Day: want.Day, Month: want.Month, Year: want.Year, IsLeapMonth: want.IsLeapMonth}, got[0].LunarDate)
	})

	t.Run("multi-day event shows for any window that overlaps its span", func(t *testing.T) {
		e := evt("a", at(2026, 11, 1, 6, 0), at(2026, 11, 3, 17, 0))
		got := ResolveTempleOccurrences([]TempleEvent{e}, win(day(2026, 11, 2), day(2026, 11, 2)), nowFixed, vnZone)
		require.Len(t, got, 1)
		assert.Equal(t, day(2026, 11, 1), got[0].SolarDate, "solar_date is the start date")
	})

	t.Run("midnight crossing is one occurrence", func(t *testing.T) {
		e := evt("a", at(2026, 11, 7, 22, 0), at(2026, 11, 8, 2, 0))
		got := ResolveTempleOccurrences([]TempleEvent{e}, win(day(2026, 11, 8), day(2026, 11, 8)), nowFixed, vnZone)
		require.Len(t, got, 1)
		assert.Equal(t, day(2026, 11, 7), got[0].SolarDate)
		assert.True(t, at(2026, 11, 8, 2, 0).Equal(got[0].EndsAt))
	})

	t.Run("outside the window is dropped", func(t *testing.T) {
		e := evt("a", at(2027, 5, 1, 6, 0), at(2027, 5, 1, 8, 0))
		assert.Empty(t, ResolveTempleOccurrences([]TempleEvent{e}, w, nowFixed, vnZone))
	})

	t.Run("all-day event lasts through its last day", func(t *testing.T) {
		e := evt("a", at(2026, 12, 5, 0, 0), at(2026, 12, 7, 0, 0))
		e.IsAllDay = true
		got := ResolveTempleOccurrences([]TempleEvent{e}, win(day(2026, 12, 7), day(2026, 12, 7)), nowFixed, vnZone)
		require.Len(t, got, 1)
		assert.True(t, at(2026, 12, 7, 23, 59).Add(59*time.Second).Equal(got[0].EndsAt))
	})
}

func TestResolve_PastAndInProgress(t *testing.T) {
	today := win(day(2026, 10, 13), day(2026, 10, 13))

	ended := evt("ended", at(2026, 10, 13, 6, 0), at(2026, 10, 13, 8, 0))
	inProgress := evt("live", at(2026, 10, 13, 8, 0), at(2026, 10, 13, 11, 0))
	later := evt("later", at(2026, 10, 13, 19, 0), at(2026, 10, 13, 20, 0))
	got := ResolveTempleOccurrences([]TempleEvent{ended, inProgress, later}, today, nowFixed, vnZone)

	ids := []string{}
	for _, o := range got {
		ids = append(ids, o.EventID)
	}
	assert.Equal(t, []string{"live", "later"}, ids)

	t.Run("window entirely in the past is empty", func(t *testing.T) {
		e := evt("a", at(2026, 9, 1, 6, 0), at(2026, 9, 1, 8, 0))
		assert.Empty(t, ResolveTempleOccurrences([]TempleEvent{e}, win(day(2026, 9, 1), day(2026, 9, 30)), nowFixed, vnZone))
	})
}

func TestResolve_Weekly(t *testing.T) {
	e := evt("w", at(2026, 10, 4, 8, 0), at(2026, 10, 4, 10, 0)) // a Sunday
	e.Recurrence = RecurrenceWeekly
	e.Weekday = intp(0)

	t.Run("every matching weekday at the same time of day", func(t *testing.T) {
		got := ResolveTempleOccurrences([]TempleEvent{e}, win(day(2026, 10, 13), day(2026, 11, 8)), nowFixed, vnZone)
		assert.Equal(t, []string{"2026-10-18", "2026-10-25", "2026-11-01", "2026-11-08"}, templeDates(got))
		assert.True(t, at(2026, 10, 18, 8, 0).Equal(got[0].StartsAt))
		assert.True(t, at(2026, 10, 18, 10, 0).Equal(got[0].EndsAt))
	})

	t.Run("recurrence_until is inclusive", func(t *testing.T) {
		u := day(2026, 10, 25)
		e2 := e
		e2.Until = &u
		got := ResolveTempleOccurrences([]TempleEvent{e2}, win(day(2026, 10, 13), day(2026, 12, 31)), nowFixed, vnZone)
		assert.Equal(t, []string{"2026-10-18", "2026-10-25"}, templeDates(got))
	})

	t.Run("nothing before the first date", func(t *testing.T) {
		e2 := evt("w", at(2026, 11, 8, 8, 0), at(2026, 11, 8, 10, 0))
		e2.Recurrence = RecurrenceWeekly
		e2.Weekday = intp(0)
		got := ResolveTempleOccurrences([]TempleEvent{e2}, win(day(2026, 10, 13), day(2026, 11, 15)), nowFixed, vnZone)
		assert.Equal(t, []string{"2026-11-08", "2026-11-15"}, templeDates(got))
	})

	t.Run("a Saturday-night weekly event that runs past midnight shows on Sunday's window", func(t *testing.T) {
		n := evt("n", at(2026, 10, 3, 22, 0), at(2026, 10, 4, 1, 0))
		n.Recurrence = RecurrenceWeekly
		n.Weekday = intp(6)
		got := ResolveTempleOccurrences([]TempleEvent{n}, win(day(2026, 10, 18), day(2026, 10, 18)), nowFixed, vnZone)
		require.Len(t, got, 1)
		assert.Equal(t, day(2026, 10, 17), got[0].SolarDate)
	})
}

func TestResolve_LunarMonthly(t *testing.T) {
	base := func(days ...int) TempleEvent {
		e := evt("l", at(2026, 1, 1, 19, 0), at(2026, 1, 1, 21, 0))
		e.Recurrence = RecurrenceLunarMonthly
		e.LunarDays = days
		return e
	}
	yearWin := win(day(2026, 1, 1), day(2026, 12, 31))
	past := at(2025, 12, 31, 0, 0)

	t.Run("day 15 falls on lunar day 15 of every month in the window", func(t *testing.T) {
		got := ResolveTempleOccurrences([]TempleEvent{base(15)}, yearWin, past, vnZone)
		require.NotEmpty(t, got)
		for _, o := range got {
			assert.Equal(t, 15, o.LunarDate.Day, o.SolarDate)
			assert.Equal(t, 19, o.StartsAt.Hour())
		}
		// Known reference from the calendar feature: 15/1 of 2026 is 2026-03-03.
		assert.Contains(t, templeDates(got), "2026-03-03")
		want := 0
		for _, ly := range []int{2025, 2026} {
			for _, m := range lunarcal.LunarMonths(ly) {
				if m.Start.AddDate(0, 0, 14).Year() == 2026 {
					want++
				}
			}
		}
		assert.Len(t, got, want)
	})

	t.Run("day 30 in a 29-day month lands on day 29, once, even with 29 and 30 listed", func(t *testing.T) {
		var short lunarcal.Month
		for _, m := range lunarcal.LunarMonths(2026) {
			if m.Days == 29 {
				short = m
				break
			}
		}
		require.Equal(t, 29, short.Days, "2026 has a 29-day lunar month")
		lastDay := short.Start.AddDate(0, 0, 28)
		w := win(lastDay, lastDay)

		for _, days := range [][]int{{30}, {29, 30}} {
			got := ResolveTempleOccurrences([]TempleEvent{base(days...)}, w, past, vnZone)
			require.Len(t, got, 1, "days %v", days)
			assert.Equal(t, lastDay, got[0].SolarDate)
			assert.Equal(t, 29, got[0].LunarDate.Day)
		}
	})

	t.Run("day 30 in a 30-day month stays on day 30", func(t *testing.T) {
		var long lunarcal.Month
		for _, m := range lunarcal.LunarMonths(2026) {
			if m.Days == 30 {
				long = m
				break
			}
		}
		require.Equal(t, 30, long.Days)
		d := long.Start.AddDate(0, 0, 29)
		got := ResolveTempleOccurrences([]TempleEvent{base(30)}, win(d, d), past, vnZone)
		require.Len(t, got, 1)
		assert.Equal(t, 30, got[0].LunarDate.Day)
	})

	t.Run("leap month gives an extra occurrence flagged as leap", func(t *testing.T) {
		require.Equal(t, 6, lunarcal.LeapMonth(2025))
		e := base(15)
		e.StartsAt, e.EndsAt = at(2025, 1, 1, 19, 0), at(2025, 1, 1, 21, 0)
		got := ResolveTempleOccurrences([]TempleEvent{e}, win(day(2025, 1, 1), day(2025, 12, 31)), at(2024, 12, 1, 0, 0), vnZone)
		leap, month6 := 0, 0
		for _, o := range got {
			if o.LunarDate.Month == 6 && o.LunarDate.Year == 2025 {
				month6++
				if o.LunarDate.IsLeapMonth {
					leap++
				}
			}
		}
		assert.Equal(t, 2, month6)
		assert.Equal(t, 1, leap)
	})

	t.Run("29/12 of the previous lunar year can land in January", func(t *testing.T) {
		// Last day of lunar month 12 of lunar year 2025 falls in solar Jan/Feb 2026.
		var m12 lunarcal.Month
		for _, m := range lunarcal.LunarMonths(2025) {
			if m.Month == 12 && !m.Leap {
				m12 = m
			}
		}
		d := m12.Start.AddDate(0, 0, m12.Days-1)
		require.Equal(t, 2026, d.Year(), "reference: last day of 12th month 2025 is in 2026")
		got := ResolveTempleOccurrences([]TempleEvent{base(30)}, win(d, d), past, vnZone)
		require.Len(t, got, 1)
		assert.Equal(t, 2025, got[0].LunarDate.Year)
		assert.Equal(t, 12, got[0].LunarDate.Month)
	})

	t.Run("recurrence_until is inclusive and starts_at date is the lower bound", func(t *testing.T) {
		e := base(15)
		u := day(2026, 3, 3)
		e.Until = &u
		got := ResolveTempleOccurrences([]TempleEvent{e}, yearWin, past, vnZone)
		assert.Equal(t, "2026-03-03", templeDates(got)[len(got)-1])

		e2 := base(15)
		e2.StartsAt, e2.EndsAt = at(2026, 3, 3, 19, 0), at(2026, 3, 3, 21, 0)
		got = ResolveTempleOccurrences([]TempleEvent{e2}, yearWin, past, vnZone)
		assert.Equal(t, "2026-03-03", templeDates(got)[0])
	})
}

func TestResolve_Ordering(t *testing.T) {
	b := evt("b", at(2026, 11, 1, 8, 0), at(2026, 11, 1, 9, 0))
	a := evt("a", at(2026, 11, 1, 8, 0), at(2026, 11, 1, 9, 0))
	c := evt("c", at(2026, 10, 30, 8, 0), at(2026, 10, 30, 9, 0))
	w := win(day(2026, 10, 13), day(2026, 12, 31))
	first := ResolveTempleOccurrences([]TempleEvent{b, a, c}, w, nowFixed, vnZone)
	second := ResolveTempleOccurrences([]TempleEvent{c, a, b}, w, nowFixed, vnZone)

	ids := []string{first[0].EventID, first[1].EventID, first[2].EventID}
	assert.Equal(t, []string{"c", "a", "b"}, ids)
	assert.Equal(t, first, second)
}

func TestResolve_SkipsInvalidEvent(t *testing.T) {
	bad := evt("bad", at(2026, 11, 1, 8, 0), at(2026, 11, 1, 9, 0))
	bad.Kind = "NOPE"
	good := evt("good", at(2026, 11, 2, 8, 0), at(2026, 11, 2, 9, 0))
	got := ResolveTempleOccurrences([]TempleEvent{bad, good}, win(day(2026, 10, 13), day(2026, 12, 31)), nowFixed, vnZone)
	require.Len(t, got, 1)
	assert.Equal(t, "good", got[0].EventID)
}
