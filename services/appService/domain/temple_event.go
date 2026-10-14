package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// TempleEventKind is the category of a temple event. The set is fixed by the
// temple_events.kind CHECK; a new kind is a migration plus an entry here.
type TempleEventKind string

const (
	TempleEventKhoaTu     TempleEventKind = "KHOA_TU"
	TempleEventPhapHoi    TempleEventKind = "PHAP_HOI"
	TempleEventLeCauAn    TempleEventKind = "LE_CAU_AN"
	TempleEventLeCauSieu  TempleEventKind = "LE_CAU_SIEU"
	TempleEventLeHoi      TempleEventKind = "LE_HOI"
	TempleEventThuyetPhap TempleEventKind = "THUYET_PHAP"
	TempleEventKhac       TempleEventKind = "KHAC"
)

var templeEventKinds = map[TempleEventKind]bool{
	TempleEventKhoaTu: true, TempleEventPhapHoi: true, TempleEventLeCauAn: true,
	TempleEventLeCauSieu: true, TempleEventLeHoi: true, TempleEventThuyetPhap: true, TempleEventKhac: true,
}

// TempleEventStatus is the state of an event as a whole (not of one occurrence).
type TempleEventStatus string

const (
	TempleEventScheduled TempleEventStatus = "SCHEDULED"
	TempleEventPostponed TempleEventStatus = "POSTPONED"
	TempleEventCancelled TempleEventStatus = "CANCELLED"
)

// TempleRecurrence says how an event repeats.
type TempleRecurrence string

const (
	RecurrenceNone         TempleRecurrence = "NONE"
	RecurrenceLunarMonthly TempleRecurrence = "LUNAR_MONTHLY"
	RecurrenceWeekly       TempleRecurrence = "WEEKLY"
)

// TempleEvent is one event owned by a temple (khoá tu, pháp hội, lễ…), maybe
// recurring. It maps temple_events. Plain data: no json/db tag.
//
// StartsAt is the first (or only) start. For a recurring event EndsAt-StartsAt
// is the length of each occurrence and the clock time of StartsAt (in the
// configured zone) is the time of day of every occurrence. Until is a civil
// date at UTC midnight, inclusive; nil = repeats forever. An all-day event has
// StartsAt/EndsAt at 00:00 (configured zone) of its first and last day and
// covers the whole last day.
type TempleEvent struct {
	ID           string
	TempleID     string
	Title        string
	Description  string
	Kind         TempleEventKind
	StartsAt     time.Time
	EndsAt       time.Time
	IsAllDay     bool
	Status       TempleEventStatus
	AudienceNote string
	ContactInfo  string
	ImageURL     *string
	Recurrence   TempleRecurrence
	LunarDays    []int
	Weekday      *int
	Until        *time.Time
}

// TempleEventWindow is a span of civil dates, both inclusive, as UTC midnights.
type TempleEventWindow struct {
	From time.Time
	To   time.Time
}

// TempleEventOccurrence is one concrete happening of an event. StartsAt and
// EndsAt are in the configured zone; for an all-day event EndsAt is 23:59:59 of
// its last day. SolarDate is the start date (UTC midnight), LunarDate its lunar date.
type TempleEventOccurrence struct {
	EventID   string
	StartsAt  time.Time
	EndsAt    time.Time
	SolarDate time.Time
	LunarDate CalendarLunarDate
}

// TempleEventsResult is what the use case returns: the effective window, the
// events that have an occurrence in it, and those occurrences.
type TempleEventsResult struct {
	TempleID    string
	From        time.Time
	To          time.Time
	Events      []TempleEvent
	Occurrences []TempleEventOccurrence
}

// TempleEventCatalogue is one seed file. Source records where the data came
// from (required, not stored).
type TempleEventCatalogue struct {
	Source string
	Events []TempleEvent
}

// Sentinel errors for temple events. ErrInvalidDate and ErrTempleNotFound are
// reused from calendar and temple.
var (
	// ErrInvalidDateRange — "to" is before "from".
	ErrInvalidDateRange = errors.New("to must not be before from")

	// ErrDateRangeTooLarge — the window is longer than the configured maximum.
	ErrDateRangeTooLarge = errors.New("date range is longer than the allowed maximum")

	// ErrInvalidTempleEvent — a seed catalogue failed Verify.
	ErrInvalidTempleEvent = errors.New("invalid temple event")
)

// InvalidDateFieldError says which query parameter ("from" or "to") was not a
// valid date. It unwraps to ErrInvalidDate.
type InvalidDateFieldError struct{ Field string }

func (e *InvalidDateFieldError) Error() string { return fmt.Sprintf("%s: %s", e.Field, ErrInvalidDate) }
func (e *InvalidDateFieldError) Unwrap() error { return ErrInvalidDate }

// Supported solar years of the lunar converter.
const (
	templeEventMinYear = 1900
	templeEventMaxYear = 2100
)

// TempleEventRepository reads a temple's events and loads the seed.
type TempleEventRepository interface {
	// ListActiveByTemple returns the active events of one temple that can have
	// an occurrence in w. It is a coarse filter by design: the resolver does
	// the exact one, so it may return events that end up with none. Order is
	// not guaranteed. A temple with no such event returns an empty slice.
	ListActiveByTemple(ctx context.Context, templeID string, w TempleEventWindow) ([]TempleEvent, error)

	// ReplaceCatalogue upserts the events by id in one transaction: all of it
	// or none. It never deletes an event absent from c and never writes
	// is_active (back-office owns it). A temple_id that does not exist fails
	// the whole call. It does not verify c; the caller does.
	ReplaceCatalogue(ctx context.Context, c TempleEventCatalogue) error
}

// Validate reports why the event's own data cannot be expanded (unknown enum,
// missing or inconsistent recurrence parameters, a recurring span of a day or
// more, years outside the converter's range). loc is the configured zone.
func (e TempleEvent) Validate(loc *time.Location) error {
	if !templeEventKinds[e.Kind] {
		return fmt.Errorf("kind %q is not known", e.Kind)
	}
	switch e.Status {
	case TempleEventScheduled, TempleEventPostponed, TempleEventCancelled:
	default:
		return fmt.Errorf("status %q is not known", e.Status)
	}
	if e.EndsAt.Before(e.StartsAt) {
		return errors.New("ends_at is before starts_at")
	}
	for _, t := range []time.Time{e.StartsAt, e.EndsAt} {
		if y := t.In(loc).Year(); y < templeEventMinYear || y > templeEventMaxYear {
			return fmt.Errorf("year %d is outside %d..%d", y, templeEventMinYear, templeEventMaxYear)
		}
	}
	first := civilDate(e.StartsAt, loc)

	switch e.Recurrence {
	case RecurrenceNone:
		if len(e.LunarDays) != 0 || e.Weekday != nil || e.Until != nil {
			return errors.New("recurrence NONE takes no lunar_days, weekday or until")
		}
	case RecurrenceLunarMonthly:
		if len(e.LunarDays) == 0 {
			return errors.New("recurrence LUNAR_MONTHLY needs lunar_days")
		}
		for _, d := range e.LunarDays {
			if d < 1 || d > 30 {
				return fmt.Errorf("lunar day %d is outside 1..30", d)
			}
		}
		if e.Weekday != nil {
			return errors.New("recurrence LUNAR_MONTHLY takes no weekday")
		}
	case RecurrenceWeekly:
		if e.Weekday == nil || *e.Weekday < 0 || *e.Weekday > 6 {
			return errors.New("recurrence WEEKLY needs weekday 0..6")
		}
		if len(e.LunarDays) != 0 {
			return errors.New("recurrence WEEKLY takes no lunar_days")
		}
	default:
		return fmt.Errorf("recurrence %q is not known", e.Recurrence)
	}

	if e.Until != nil {
		if e.Until.Before(first) {
			return errors.New("until is before the first date")
		}
		if y := e.Until.Year(); y > templeEventMaxYear {
			return fmt.Errorf("until year %d is outside %d..%d", y, templeEventMinYear, templeEventMaxYear)
		}
	}
	if e.Recurrence != RecurrenceNone {
		if e.IsAllDay {
			if !civilDate(e.EndsAt, loc).Equal(first) {
				return errors.New("a recurring all-day event must last one day")
			}
		} else if e.EndsAt.Sub(e.StartsAt) >= 24*time.Hour {
			return errors.New("a recurring event must last less than 24 hours")
		}
	}
	if e.IsAllDay {
		for _, t := range []time.Time{e.StartsAt, e.EndsAt} {
			if h, m, s := t.In(loc).Clock(); h != 0 || m != 0 || s != 0 || t.Nanosecond() != 0 {
				return errors.New("an all-day event starts and ends at 00:00")
			}
		}
	}
	return nil
}

// civilDate is the calendar date of t in loc as a UTC midnight.
func civilDate(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Column limits of temple_events, counted in characters.
const (
	maxTempleEventID    = 64
	maxTempleEventTitle = 255
)

// Verify checks the catalogue's own consistency before it is loaded. It
// returns nil, or an error wrapping ErrInvalidTempleEvent whose message lists
// every problem found (each names the event id). loc is the configured zone.
func (c TempleEventCatalogue) Verify(loc *time.Location) error {
	var issues []string
	add := func(format string, a ...any) { issues = append(issues, fmt.Sprintf(format, a...)) }

	if strings.TrimSpace(c.Source) == "" {
		add("source is empty")
	}
	if len(c.Events) == 0 {
		add("no events")
	}
	seen := make(map[string]bool, len(c.Events))
	for i, e := range c.Events {
		if e.ID == "" {
			add("events[%d]: id is empty", i)
			continue
		}
		if utf8.RuneCountInString(e.ID) > maxTempleEventID {
			add("event %q: id is longer than %d characters", e.ID, maxTempleEventID)
		}
		if seen[e.ID] {
			add("event %q: duplicate id", e.ID)
		}
		seen[e.ID] = true
		if e.TempleID == "" {
			add("event %q: temple_id is empty", e.ID)
		}
		if strings.TrimSpace(e.Title) == "" {
			add("event %q: title is empty", e.ID)
		} else if utf8.RuneCountInString(e.Title) > maxTempleEventTitle {
			add("event %q: title is longer than %d characters", e.ID, maxTempleEventTitle)
		}
		if e.ImageURL != nil && *e.ImageURL == "" {
			add("event %q: image_url must be null or non-empty", e.ID)
		}
		if err := e.Validate(loc); err != nil {
			add("event %q: %v", e.ID, err)
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %d issue(s): %s", ErrInvalidTempleEvent, len(issues), strings.Join(issues, "; "))
}
