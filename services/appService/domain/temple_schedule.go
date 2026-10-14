package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

// TimeOfDay is a wall-clock time in Vietnam, as minutes since 00:00 (0..1439).
// It carries no date and no zone: a schedule item repeats every day or on given
// weekdays, so there is nothing to convert.
type TimeOfDay int

// ParseTimeOfDay reads exactly "HH:MM", 24-hour, zero-padded ("24:00", "7:5",
// "" and "08:00:00" are all rejected).
func ParseTimeOfDay(s string) (TimeOfDay, error) {
	if len(s) != 5 || s[2] != ':' {
		return 0, fmt.Errorf("time %q: want HH:MM", s)
	}
	digits := func(a, b byte) (int, bool) {
		if a < '0' || a > '9' || b < '0' || b > '9' {
			return 0, false
		}
		return int(a-'0')*10 + int(b-'0'), true
	}
	h, okH := digits(s[0], s[1])
	m, okM := digits(s[3], s[4])
	if !okH || !okM || h > 23 || m > 59 {
		return 0, fmt.Errorf("time %q: want HH:MM in 00:00..23:59", s)
	}
	return TimeOfDay(h*60 + m), nil
}

// String formats as zero-padded "HH:MM".
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", int(t)/60, int(t)%60)
}

// Column limits of the temple_schedules table, counted in characters.
const (
	maxTempleScheduleID       = 64
	maxTempleScheduleName     = 255
	maxTempleScheduleInstruct = 255
	maxTempleScheduleLocation = 255
	maxTempleScheduleSource   = 255
)

// TempleScheduleItem is one recurring practice session of a temple (công phu
// sáng, lễ Dược Sư, ...). It maps temple_schedules; its Steps map
// temple_schedule_steps.
//
// Plain data: no json/db tag. Weekdays follow time.Weekday (0 = Sunday ..
// 6 = Saturday); empty means every day. EndTime may be earlier than StartTime
// (a session that crosses midnight). Optional text is "" and an absent end time
// is nil — delivery decides how each is shown.
type TempleScheduleItem struct {
	ID          string
	TempleID    string
	Name        string
	StartTime   TimeOfDay
	EndTime     *TimeOfDay
	Weekdays    []int
	Description string
	Instructor  string
	Location    string
	Preparation string
	SortOrder   int
	Steps       []TempleScheduleStep
}

// TempleScheduleStep is one line of a session's running order. Position (>= 1)
// alone decides the order; Time is informational and may be nil.
type TempleScheduleStep struct {
	Position int
	Time     *TimeOfDay
	Text     string
}

// TempleScheduleCatalogue is one complete seed file: the unit the seed CLI
// loads. Source records where the data came from (required, not stored).
type TempleScheduleCatalogue struct {
	Source string
	Items  []TempleScheduleItem
}

// ErrInvalidTempleSchedule — a seed catalogue failed Verify.
var ErrInvalidTempleSchedule = errors.New("invalid temple schedule")

// TempleScheduleRepository reads a temple's schedule and loads the seed.
type TempleScheduleRepository interface {
	// ListActiveByTemple returns the active items of one temple with their
	// steps (items with is_active = false are left out together with their
	// steps). Order is not guaranteed: callers apply SortTempleSchedule. A
	// temple with no active item returns an empty slice, not an error.
	ListActiveByTemple(ctx context.Context, templeID string) ([]TempleScheduleItem, error)

	// ReplaceCatalogue upserts the items by id and replaces the steps of every
	// item in c with exactly the steps listed, in one transaction: all of it or
	// none. It never deletes an item absent from c and never writes is_active
	// (back-office owns it). A temple_id that does not exist fails the whole
	// call. It does not verify c; the caller does.
	ReplaceCatalogue(ctx context.Context, c TempleScheduleCatalogue) error
}

// SortTempleSchedule normalises items in place and returns them in response
// order: items by StartTime, SortOrder, ID; steps by Position (never by Time);
// weekdays de-duplicated and ascending, with all seven days collapsed to empty
// ("every day"). A nil Steps/Weekdays becomes an empty non-nil slice.
func SortTempleSchedule(items []TempleScheduleItem) []TempleScheduleItem {
	for i := range items {
		it := &items[i]
		it.Weekdays = normaliseWeekdays(it.Weekdays)
		steps := slices.Clone(it.Steps)
		if steps == nil {
			steps = []TempleScheduleStep{}
		}
		sort.SliceStable(steps, func(a, b int) bool { return steps[a].Position < steps[b].Position })
		it.Steps = steps
	}
	sort.SliceStable(items, func(a, b int) bool {
		x, y := items[a], items[b]
		if x.StartTime != y.StartTime {
			return x.StartTime < y.StartTime
		}
		if x.SortOrder != y.SortOrder {
			return x.SortOrder < y.SortOrder
		}
		return x.ID < y.ID
	})
	return items
}

func normaliseWeekdays(in []int) []int {
	set := map[int]bool{}
	for _, d := range in {
		set[d] = true
	}
	if len(set) == 0 || len(set) == 7 {
		return []int{}
	}
	out := make([]int, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Ints(out)
	return out
}

// Verify checks the catalogue's own consistency before it is loaded. It
// returns nil, or an error wrapping ErrInvalidTempleSchedule whose message
// lists every problem found (each names the offending item id).
func (c TempleScheduleCatalogue) Verify() error {
	var issues []string
	add := func(format string, a ...any) { issues = append(issues, fmt.Sprintf(format, a...)) }
	tooLong := func(s string, max int) bool { return utf8.RuneCountInString(s) > max }
	blank := func(s string) bool { return strings.TrimSpace(s) == "" }
	validTime := func(t TimeOfDay) bool { return t >= 0 && t < 24*60 }

	if blank(c.Source) {
		add("source is empty")
	} else if tooLong(c.Source, maxTempleScheduleSource) {
		add("source is longer than %d characters", maxTempleScheduleSource)
	}
	if len(c.Items) == 0 {
		add("no items")
	}

	seen := make(map[string]bool, len(c.Items))
	for i, it := range c.Items {
		if it.ID == "" {
			add("items[%d]: id is empty", i)
			continue
		}
		if tooLong(it.ID, maxTempleScheduleID) {
			add("item %q: id is longer than %d characters", it.ID, maxTempleScheduleID)
		}
		if seen[it.ID] {
			add("item %q: duplicate id", it.ID)
		}
		seen[it.ID] = true

		if it.TempleID == "" {
			add("item %q: temple_id is empty", it.ID)
		}
		if blank(it.Name) {
			add("item %q: name is empty", it.ID)
		} else if tooLong(it.Name, maxTempleScheduleName) {
			add("item %q: name is longer than %d characters", it.ID, maxTempleScheduleName)
		}
		if tooLong(it.Instructor, maxTempleScheduleInstruct) {
			add("item %q: instructor is longer than %d characters", it.ID, maxTempleScheduleInstruct)
		}
		if tooLong(it.Location, maxTempleScheduleLocation) {
			add("item %q: location is longer than %d characters", it.ID, maxTempleScheduleLocation)
		}
		if !validTime(it.StartTime) {
			add("item %q: start_time out of range", it.ID)
		}
		if it.EndTime != nil {
			switch {
			case !validTime(*it.EndTime):
				add("item %q: end_time out of range", it.ID)
			case *it.EndTime == it.StartTime:
				add("item %q: end_time equals start_time", it.ID)
			}
		}
		for _, d := range it.Weekdays {
			if d < 0 || d > 6 {
				add("item %q: weekday %d is outside 0..6", it.ID, d)
			}
		}
		positions := map[int]bool{}
		for _, s := range it.Steps {
			if s.Position < 1 {
				add("item %q: step position must be >= 1, got %d", it.ID, s.Position)
			}
			if positions[s.Position] {
				add("item %q: two steps share position %d", it.ID, s.Position)
			}
			positions[s.Position] = true
			if blank(s.Text) {
				add("item %q: step %d text is empty", it.ID, s.Position)
			}
			if s.Time != nil && !validTime(*s.Time) {
				add("item %q: step %d time out of range", it.ID, s.Position)
			}
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %d issue(s): %s", ErrInvalidTempleSchedule, len(issues), strings.Join(issues, "; "))
}
