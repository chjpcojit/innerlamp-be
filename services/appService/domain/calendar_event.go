package domain

import (
	"errors"
	"fmt"
	"time"
)

// CalendarEventKind says how an event's date is derived for a given year.
type CalendarEventKind string

const (
	// KindSolar is a fixed solar date (SolarMonth/SolarDay), e.g. 30/4.
	KindSolar CalendarEventKind = "solar"
	// KindLunar is a fixed lunar date (LunarMonth/LunarDay, optionally the
	// leap month), e.g. Phật đản 15/4 ÂL.
	KindLunar CalendarEventKind = "lunar"
	// KindLunarMonthly repeats every lunar month on LunarDay (mùng 1, Rằm).
	KindLunarMonthly CalendarEventKind = "lunar_monthly"
	// KindLunarLastDay is the last day (29 or 30) of LunarMonth, e.g. Giao thừa.
	KindLunarLastDay CalendarEventKind = "lunar_last_day"
	// KindSolarTerm is the day the Sun reaches SolarTermLongitude (tiết khí).
	KindSolarTerm CalendarEventKind = "solar_term"
)

// ErrInvalidYear — the year query is not a whole number between 1900 and 2100.
var ErrInvalidYear = errors.New("year must be a whole number between 1900 and 2100")

// ErrInvalidRecurrenceRule — a rule is missing the parameters its kind needs,
// or they are out of range.
var ErrInvalidRecurrenceRule = errors.New("invalid recurrence rule")

// RecurrenceRule is the repeat rule of an event. Which fields are used depends
// on Kind (see the Kind constants); the others are zero.
type RecurrenceRule struct {
	Kind               CalendarEventKind
	SolarMonth         int
	SolarDay           int
	LunarMonth         int
	LunarDay           int
	IsLeapMonth        bool
	SolarTermLongitude int
}

// maxSolarDay holds the longest each solar month can be (29/2 is allowed).
var maxSolarDay = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// Validate reports whether the rule has the parameters its kind requires,
// each in range. The error wraps ErrInvalidRecurrenceRule.
func (r RecurrenceRule) Validate() error {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidRecurrenceRule, fmt.Sprintf(format, a...))
	}
	switch r.Kind {
	case KindSolar:
		if r.SolarMonth < 1 || r.SolarMonth > 12 {
			return bad("solar_month %d not in 1..12", r.SolarMonth)
		}
		if r.SolarDay < 1 || r.SolarDay > maxSolarDay[r.SolarMonth] {
			return bad("solar_day %d not valid for month %d", r.SolarDay, r.SolarMonth)
		}
	case KindLunar:
		if r.LunarMonth < 1 || r.LunarMonth > 12 {
			return bad("lunar_month %d not in 1..12", r.LunarMonth)
		}
		if r.LunarDay < 1 || r.LunarDay > 30 {
			return bad("lunar_day %d not in 1..30", r.LunarDay)
		}
	case KindLunarMonthly:
		if r.LunarDay < 1 || r.LunarDay > 30 {
			return bad("lunar_day %d not in 1..30", r.LunarDay)
		}
	case KindLunarLastDay:
		if r.LunarMonth < 1 || r.LunarMonth > 12 {
			return bad("lunar_month %d not in 1..12", r.LunarMonth)
		}
	case KindSolarTerm:
		if r.SolarTermLongitude < 0 || r.SolarTermLongitude > 345 || r.SolarTermLongitude%15 != 0 {
			return bad("solar_term_longitude %d is not a multiple of 15 in 0..345", r.SolarTermLongitude)
		}
	default:
		return bad("unknown kind %q", r.Kind)
	}
	return nil
}

// CalendarGroup is a label an event can belong to (Phật giáo, Việt Nam, Dân gian…).
type CalendarGroup struct {
	Code      string
	Name      string
	SortOrder int
}

// CalendarEvent is one catalogue entry: its content in a single language and
// the rule that places it on the calendar.
type CalendarEvent struct {
	Code             string
	Name             string
	ShortDescription string
	LongDescription  string
	Guidance         string
	// ImageURL is nil until the owner supplies an image.
	ImageURL *string
	// Groups holds group codes, in display order.
	Groups          []string
	IsPublicHoliday bool
	SortOrder       int
	Rule            RecurrenceRule
}

// CalendarLunarDate is a lunar date, named apart from genealogy.go's year-less
// LunarDate. Year is the lunar year, which can differ from
// the solar year of the same day (lunar 23/12 falls in January–February).
type CalendarLunarDate struct {
	Day         int
	Month       int
	Year        int
	IsLeapMonth bool
}

// CalendarOccurrence is one event on one solar day, with its lunar date.
type CalendarOccurrence struct {
	EventCode string
	// SolarDate is a UTC-midnight time; only its year, month and day matter.
	SolarDate time.Time
	LunarDate CalendarLunarDate
}

// CalendarEventsResult is the answer for one solar year: every group, the
// events that occur in that year (each once), and their occurrences in
// order. Occurrences point at Events by EventCode.
type CalendarEventsResult struct {
	Year        int
	Groups      []CalendarGroup
	Events      []CalendarEvent
	Occurrences []CalendarOccurrence
}
