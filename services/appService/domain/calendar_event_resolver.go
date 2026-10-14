package domain

import (
	"sort"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain/lunarcal"
)

// ResolveOccurrences returns every occurrence of events whose solar date
// falls in the Gregorian year, sorted by solar date, then event SortOrder,
// then event code. It is pure: same input, same output, no clock, no I/O.
//
// Events whose rule fails RecurrenceRule.Validate are skipped (the caller can
// Validate first to report them). A lunar date that does not exist in a given
// year (day 30 in a 29-day month, a leap month the year lacks) simply yields
// no occurrence that year.
func ResolveOccurrences(events []CalendarEvent, year int) []CalendarOccurrence {
	type keyed struct {
		occ  CalendarOccurrence
		sort int
	}
	var all []keyed
	for _, ev := range events {
		if ev.Rule.Validate() != nil {
			continue
		}
		for _, occ := range resolveRule(ev.Rule, year) {
			occ.EventCode = ev.Code
			all = append(all, keyed{occ: occ, sort: ev.SortOrder})
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if !a.occ.SolarDate.Equal(b.occ.SolarDate) {
			return a.occ.SolarDate.Before(b.occ.SolarDate)
		}
		if a.sort != b.sort {
			return a.sort < b.sort
		}
		return a.occ.EventCode < b.occ.EventCode
	})
	out := make([]CalendarOccurrence, len(all))
	for i, k := range all {
		out[i] = k.occ
	}
	return out
}

func toCalendarLunarDate(d lunarcal.Date) CalendarLunarDate {
	return CalendarLunarDate{Day: d.Day, Month: d.Month, Year: d.Year, IsLeapMonth: d.IsLeapMonth}
}

func occurrenceOn(solar time.Time) CalendarOccurrence {
	return CalendarOccurrence{SolarDate: solar, LunarDate: toCalendarLunarDate(lunarcal.SolarToLunar(solar))}
}

// resolveRule expands one valid rule into occurrences (EventCode unset) whose
// solar year is year.
func resolveRule(r RecurrenceRule, year int) []CalendarOccurrence {
	var out []CalendarOccurrence
	switch r.Kind {
	case KindSolar:
		d := time.Date(year, time.Month(r.SolarMonth), r.SolarDay, 0, 0, 0, 0, time.UTC)
		if d.Month() == time.Month(r.SolarMonth) { // 29/2 in a common year rolls into March
			out = append(out, occurrenceOn(d))
		}
	case KindSolarTerm:
		if d, ok := lunarcal.SolarTermDate(year, r.SolarTermLongitude); ok {
			out = append(out, occurrenceOn(d))
		}
	case KindLunar:
		// A solar year overlaps lunar years year-1 (its tháng Chạp) and year.
		for _, ly := range []int{year - 1, year} {
			ld := lunarcal.Date{Day: r.LunarDay, Month: r.LunarMonth, Year: ly, IsLeapMonth: r.IsLeapMonth}
			solar, err := lunarcal.LunarToSolar(ld)
			if err != nil || solar.Year() != year {
				continue
			}
			out = append(out, CalendarOccurrence{SolarDate: solar, LunarDate: toCalendarLunarDate(ld)})
		}
	case KindLunarMonthly:
		for _, ly := range []int{year - 1, year} {
			for _, m := range lunarcal.LunarMonths(ly) {
				if m.Days < r.LunarDay {
					continue
				}
				solar := m.Start.AddDate(0, 0, r.LunarDay-1)
				if solar.Year() != year {
					continue
				}
				out = append(out, CalendarOccurrence{
					SolarDate: solar,
					LunarDate: CalendarLunarDate{Day: r.LunarDay, Month: m.Month, Year: ly, IsLeapMonth: m.Leap},
				})
			}
		}
	case KindLunarLastDay:
		for _, ly := range []int{year - 1, year} {
			for _, m := range lunarcal.LunarMonths(ly) {
				if m.Month != r.LunarMonth || m.Leap {
					continue
				}
				solar := m.Start.AddDate(0, 0, m.Days-1)
				if solar.Year() != year {
					continue
				}
				out = append(out, CalendarOccurrence{
					SolarDate: solar,
					LunarDate: CalendarLunarDate{Day: m.Days, Month: m.Month, Year: ly},
				})
			}
		}
	}
	return out
}
