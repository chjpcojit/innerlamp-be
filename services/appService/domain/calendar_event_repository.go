package domain

import "context"

// CalendarCatalogue is one complete snapshot of the event catalogue in a
// single language — the unit the seed CLI loads. Every event in Events
// carries its text (name, descriptions, guidance) in Lang.
type CalendarCatalogue struct {
	Lang   string
	Groups []CalendarGroup
	Events []CalendarEvent
}

// CalendarEventRepository reads and loads the calendar event catalogue.
type CalendarEventRepository interface {
	// ListGroups returns every group, ordered by sort_order then code.
	ListGroups(ctx context.Context) ([]CalendarGroup, error)

	// ListActiveEvents returns the active events that have text in lang,
	// ordered by sort_order then code. Each event's Groups holds its group
	// codes ordered by the group's sort_order then code; an event with no
	// group has an empty (non-nil) slice. An event with no translation in
	// lang is not returned.
	ListActiveEvents(ctx context.Context, lang string) ([]CalendarEvent, error)

	// ReplaceCatalogue upserts the groups, events, their text in c.Lang and
	// their group links in one transaction: all of it or none. It never
	// deletes an event or group absent from c, and never changes an event's
	// is_active flag (back-office owns it). An event's group links are
	// replaced by exactly the groups listed for it. It does not verify c; the
	// caller does.
	ReplaceCatalogue(ctx context.Context, c CalendarCatalogue) error
}
