package domain

import (
	"fmt"
	"unicode/utf8"
)

// Column limits of the calendar_* tables, counted in characters.
const (
	maxCalendarEventCode = 64
	maxCalendarGroupCode = 32
	maxCalendarName      = 255
	maxCalendarGroupName = 100
	maxCalendarImageURL  = 500
	maxCalendarLang      = 8
)

// Verify checks the catalogue's own consistency before it is loaded and
// returns every problem found (nil when it is sound): codes present, unique and
// within column limits; texts non-empty; every rule valid for its kind; every
// event in at least one existing group, listed once; sort orders >= 1.
func (c CalendarCatalogue) Verify() []string {
	var issues []string
	add := func(format string, a ...any) { issues = append(issues, fmt.Sprintf(format, a...)) }
	tooLong := func(s string, max int) bool { return utf8.RuneCountInString(s) > max }

	if c.Lang == "" {
		add("lang is empty")
	} else if tooLong(c.Lang, maxCalendarLang) {
		add("lang %q is longer than %d characters", c.Lang, maxCalendarLang)
	}
	if len(c.Events) == 0 {
		add("no events")
	}

	groups := make(map[string]bool, len(c.Groups))
	for i, g := range c.Groups {
		switch {
		case g.Code == "":
			add("groups[%d]: code is empty", i)
			continue
		case tooLong(g.Code, maxCalendarGroupCode):
			add("group %q: code is longer than %d characters", g.Code, maxCalendarGroupCode)
		}
		if groups[g.Code] {
			add("group %q: duplicate code", g.Code)
		}
		groups[g.Code] = true
		if g.Name == "" {
			add("group %q: name is empty", g.Code)
		} else if tooLong(g.Name, maxCalendarGroupName) {
			add("group %q: name is longer than %d characters", g.Code, maxCalendarGroupName)
		}
		if g.SortOrder < 1 {
			add("group %q: sort_order must be >= 1, got %d", g.Code, g.SortOrder)
		}
	}

	seen := make(map[string]bool, len(c.Events))
	for i, e := range c.Events {
		if e.Code == "" {
			add("events[%d]: code is empty", i)
			continue
		}
		if tooLong(e.Code, maxCalendarEventCode) {
			add("event %q: code is longer than %d characters", e.Code, maxCalendarEventCode)
		}
		if seen[e.Code] {
			add("event %q: duplicate code", e.Code)
		}
		seen[e.Code] = true

		if e.Name == "" {
			add("event %q: name is empty", e.Code)
		} else if tooLong(e.Name, maxCalendarName) {
			add("event %q: name is longer than %d characters", e.Code, maxCalendarName)
		}
		if e.ShortDescription == "" {
			add("event %q: short_description is empty", e.Code)
		}
		if e.LongDescription == "" {
			add("event %q: long_description is empty", e.Code)
		}
		if e.Guidance == "" {
			add("event %q: guidance is empty", e.Code)
		}
		if e.ImageURL != nil && (*e.ImageURL == "" || tooLong(*e.ImageURL, maxCalendarImageURL)) {
			add("event %q: image_url must be null or 1..%d characters", e.Code, maxCalendarImageURL)
		}
		if e.SortOrder < 1 {
			add("event %q: sort_order must be >= 1, got %d", e.Code, e.SortOrder)
		}
		if err := e.Rule.Validate(); err != nil {
			add("event %q: %v", e.Code, err)
		}
		if e.Rule.IsLeapMonth && e.Rule.Kind != KindLunar {
			add("event %q: is_leap_month is only valid for kind %q", e.Code, KindLunar)
		}
		if len(e.Groups) == 0 {
			add("event %q: belongs to no group", e.Code)
		}
		inEvent := map[string]bool{}
		for _, g := range e.Groups {
			if !groups[g] {
				add("event %q: group %q does not exist", e.Code, g)
			}
			if inEvent[g] {
				add("event %q: group %q listed twice", e.Code, g)
			}
			inEvent[g] = true
		}
	}

	// BR-CAL-05 / E25: Mùng 1 and Rằm come last on a day that also has a named event, so every
	// lunar_monthly event must sort after every other event.
	maxOther := 0
	for _, e := range c.Events {
		if e.Rule.Kind != KindLunarMonthly && e.SortOrder > maxOther {
			maxOther = e.SortOrder
		}
	}
	for _, e := range c.Events {
		if e.Rule.Kind == KindLunarMonthly && e.SortOrder <= maxOther {
			add("event %q: lunar_monthly sort_order %d must be greater than every other event's (highest is %d)", e.Code, e.SortOrder, maxOther)
		}
	}
	return issues
}
