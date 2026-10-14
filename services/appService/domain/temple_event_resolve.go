package domain

import (
	"sort"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain/lunarcal"
)

// ResolveTempleOccurrences expands events into the occurrences that overlap
// the window and have not ended yet, sorted by StartsAt then EventID. It is
// pure: same input, same output; the only clock is the now argument.
//
// loc is the zone in which clock times, "today" and all-day events are read.
// An event that fails Validate(loc) is skipped (callers Validate first to
// report it). A lunar-monthly day 30 in a 29-day month lands on day 29, and
// two listed days that land on the same date give one occurrence.
func ResolveTempleOccurrences(events []TempleEvent, w TempleEventWindow, now time.Time, loc *time.Location) []TempleEventOccurrence {
	windowStart := time.Date(w.From.Year(), w.From.Month(), w.From.Day(), 0, 0, 0, 0, loc)
	windowEnd := time.Date(w.To.Year(), w.To.Month(), w.To.Day(), 23, 59, 59, 0, loc)

	var out []TempleEventOccurrence
	for _, e := range events {
		if e.Validate(loc) != nil {
			continue
		}
		for _, o := range expandTempleEvent(e, w, loc) {
			if o.EndsAt.Before(now) || o.EndsAt.Before(windowStart) || o.StartsAt.After(windowEnd) {
				continue
			}
			y := o.SolarDate.Year()
			if y < templeEventMinYear || y > templeEventMaxYear {
				continue
			}
			o.EventID = e.ID
			o.LunarDate = calendarLunarDate(lunarcal.SolarToLunar(o.SolarDate))
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].StartsAt.Equal(out[j].StartsAt) {
			return out[i].StartsAt.Before(out[j].StartsAt)
		}
		return out[i].EventID < out[j].EventID
	})
	return out
}

func calendarLunarDate(d lunarcal.Date) CalendarLunarDate {
	return CalendarLunarDate{Day: d.Day, Month: d.Month, Year: d.Year, IsLeapMonth: d.IsLeapMonth}
}

// expandTempleEvent returns every occurrence of one valid event whose start
// date is within one day of the window (the exact overlap test is the caller's).
func expandTempleEvent(e TempleEvent, w TempleEventWindow, loc *time.Location) []TempleEventOccurrence {
	first := civilDate(e.StartsAt, loc)
	if e.Recurrence == RecurrenceNone {
		return []TempleEventOccurrence{templeOccurrence(e, first, e.StartsAt.In(loc), loc)}
	}

	dur := e.EndsAt.Sub(e.StartsAt)
	h, m, s := e.StartsAt.In(loc).Clock()
	at := func(d time.Time) TempleEventOccurrence {
		start := time.Date(d.Year(), d.Month(), d.Day(), h, m, s, 0, loc)
		o := TempleEventOccurrence{StartsAt: start, EndsAt: start.Add(dur), SolarDate: d}
		if e.IsAllDay {
			o.EndsAt = endOfCivilDay(d, loc)
		}
		return o
	}

	// An occurrence that starts the day before the window can still reach into it.
	lo := w.From.AddDate(0, 0, -1)
	if first.After(lo) {
		lo = first
	}
	hi := w.To
	if e.Until != nil && e.Until.Before(hi) {
		hi = *e.Until
	}

	var out []TempleEventOccurrence
	switch e.Recurrence {
	case RecurrenceWeekly:
		for d := lo; !d.After(hi); d = d.AddDate(0, 0, 1) {
			if int(d.Weekday()) == *e.Weekday {
				out = append(out, at(d))
			}
		}
	case RecurrenceLunarMonthly:
		seen := map[time.Time]bool{}
		// 29/12 of lunar year y can fall in January of y+1, so start a year early.
		for ly := lo.Year() - 1; ly <= hi.Year(); ly++ {
			if ly < templeEventMinYear || ly > templeEventMaxYear {
				continue
			}
			for _, month := range lunarcal.LunarMonths(ly) {
				for _, day := range e.LunarDays {
					d := min(day, month.Days)
					solar := month.Start.AddDate(0, 0, d-1)
					if solar.Before(lo) || solar.After(hi) || seen[solar] {
						continue
					}
					seen[solar] = true
					out = append(out, at(solar))
				}
			}
		}
	}
	return out
}

// templeOccurrence builds the single occurrence of a non-recurring event.
func templeOccurrence(e TempleEvent, date, start time.Time, loc *time.Location) TempleEventOccurrence {
	end := e.EndsAt.In(loc)
	if e.IsAllDay {
		end = endOfCivilDay(civilDate(e.EndsAt, loc), loc)
	}
	return TempleEventOccurrence{StartsAt: start, EndsAt: end, SolarDate: date}
}

// endOfCivilDay is 23:59:59 of the date in loc.
func endOfCivilDay(d time.Time, loc *time.Location) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, loc)
}
