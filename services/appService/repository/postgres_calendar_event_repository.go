package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type calendarEventRepository struct {
	db database.Database
}

// NewCalendarEventRepository returns the Postgres-backed
// domain.CalendarEventRepository.
func NewCalendarEventRepository(db database.Database) domain.CalendarEventRepository {
	return &calendarEventRepository{db: db}
}

// ListGroups returns every group ordered by sort_order, then code.
func (r *calendarEventRepository) ListGroups(ctx context.Context) ([]domain.CalendarGroup, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, name, sort_order FROM calendar_groups ORDER BY sort_order, code`)
	if err != nil {
		return nil, fmt.Errorf("query calendar groups: %w", err)
	}
	return scanRows(rows, "calendar group", func(s rowScanner) (domain.CalendarGroup, error) {
		var g domain.CalendarGroup
		err := s.Scan(&g.Code, &g.Name, &g.SortOrder)
		return g, err
	})
}

// ListActiveEvents reads the active events with their text in lang and their
// group codes in one SELECT. The inner join on the translation drops events
// that have no text in lang; the left joins keep events with no group.
func (r *calendarEventRepository) ListActiveEvents(ctx context.Context, lang string) ([]domain.CalendarEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.code, e.kind, e.solar_month, e.solar_day, e.lunar_month, e.lunar_day,
		       e.is_leap_month, e.solar_term_longitude, e.is_public_holiday, e.image_url, e.sort_order,
		       t.name, t.short_description, t.long_description, t.guidance,
		       COALESCE(string_agg(eg.group_code, ',' ORDER BY cg.sort_order, eg.group_code), '')
		FROM calendar_events e
		JOIN calendar_event_translations t ON t.event_code = e.code AND t.lang = $1
		LEFT JOIN calendar_event_groups eg ON eg.event_code = e.code
		LEFT JOIN calendar_groups cg ON cg.code = eg.group_code
		WHERE e.is_active
		GROUP BY e.code, t.event_code, t.lang
		ORDER BY e.sort_order, e.code`, lang)
	if err != nil {
		return nil, fmt.Errorf("query calendar events: %w", err)
	}
	return scanRows(rows, "calendar event", func(s rowScanner) (domain.CalendarEvent, error) {
		var (
			ev                                         domain.CalendarEvent
			kind                                       string
			solarMonth, solarDay, lunarMonth, lunarDay sql.NullInt32
			longitude                                  sql.NullInt32
			image                                      sql.NullString
			groups                                     string
		)
		err := s.Scan(&ev.Code, &kind, &solarMonth, &solarDay, &lunarMonth, &lunarDay,
			&ev.Rule.IsLeapMonth, &longitude, &ev.IsPublicHoliday, &image, &ev.SortOrder,
			&ev.Name, &ev.ShortDescription, &ev.LongDescription, &ev.Guidance, &groups)
		if err != nil {
			return ev, err
		}
		ev.Rule.Kind = domain.CalendarEventKind(kind)
		ev.Rule.SolarMonth = int(solarMonth.Int32)
		ev.Rule.SolarDay = int(solarDay.Int32)
		ev.Rule.LunarMonth = int(lunarMonth.Int32)
		ev.Rule.LunarDay = int(lunarDay.Int32)
		ev.Rule.SolarTermLongitude = int(longitude.Int32)
		if image.Valid {
			ev.ImageURL = &image.String
		}
		ev.Groups = []string{}
		if groups != "" {
			ev.Groups = strings.Split(groups, ",")
		}
		return ev, nil
	})
}

// ReplaceCatalogue upserts the catalogue in one transaction. Order: groups,
// events, translations, then each event's links (replaced wholesale), so the
// foreign keys hold at every step. is_active is deliberately not written: a
// new event takes the column default (true) and an existing one keeps what
// back-office set. Nothing is ever deleted.
func (r *calendarEventRepository) ReplaceCatalogue(ctx context.Context, c domain.CalendarCatalogue) error {
	if c.Lang == "" {
		return errors.New("replace calendar catalogue: lang is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace calendar catalogue: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := upsertBatched(ctx, tx, upsertSpec{"calendar_groups",
		[]string{"code", "name", "sort_order"}, []string{"code"}},
		c.Groups, func(g domain.CalendarGroup) []any { return []any{g.Code, g.Name, g.SortOrder} }); err != nil {
		return err
	}

	if err := upsertBatched(ctx, tx, upsertSpec{"calendar_events",
		[]string{"code", "kind", "solar_month", "solar_day", "lunar_month", "lunar_day",
			"is_leap_month", "solar_term_longitude", "is_public_holiday", "image_url", "sort_order"},
		[]string{"code"}},
		c.Events, eventRow); err != nil {
		return err
	}

	if err := upsertBatched(ctx, tx, upsertSpec{"calendar_event_translations",
		[]string{"event_code", "lang", "name", "short_description", "long_description", "guidance"},
		[]string{"event_code", "lang"}},
		c.Events, func(e domain.CalendarEvent) []any {
			return []any{e.Code, c.Lang, e.Name, e.ShortDescription, e.LongDescription, e.Guidance}
		}); err != nil {
		return err
	}

	if len(c.Events) > 0 {
		codes := make([]any, len(c.Events))
		marks := make([]string, len(c.Events))
		for i, e := range c.Events {
			codes[i] = e.Code
			marks[i] = "$" + strconv.Itoa(i+1)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM calendar_event_groups WHERE event_code IN (`+strings.Join(marks, ",")+`)`, codes...); err != nil {
			return fmt.Errorf("clear calendar event groups: %w", err)
		}
	}
	type link struct{ event, group string }
	var links []link
	for _, e := range c.Events {
		for _, g := range e.Groups {
			links = append(links, link{e.Code, g})
		}
	}
	if err := upsertBatched(ctx, tx, upsertSpec{"calendar_event_groups",
		[]string{"event_code", "group_code"}, []string{"event_code", "group_code"}},
		links, func(l link) []any { return []any{l.event, l.group} }); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace calendar catalogue: %w", err)
	}
	return nil
}

// eventRow binds one event to the calendar_events columns. Only the parameters
// the rule's kind uses are written; the others are NULL, as the table's CHECK
// requires (a solar-term longitude of 0° is valid, so it is selected by kind,
// not by being non-zero).
func eventRow(e domain.CalendarEvent) []any {
	var solarMonth, solarDay, lunarMonth, lunarDay, longitude any
	switch e.Rule.Kind {
	case domain.KindSolar:
		solarMonth, solarDay = e.Rule.SolarMonth, e.Rule.SolarDay
	case domain.KindLunar:
		lunarMonth, lunarDay = e.Rule.LunarMonth, e.Rule.LunarDay
	case domain.KindLunarMonthly:
		lunarDay = e.Rule.LunarDay
	case domain.KindLunarLastDay:
		lunarMonth = e.Rule.LunarMonth
	case domain.KindSolarTerm:
		longitude = e.Rule.SolarTermLongitude
	}
	var image any
	if e.ImageURL != nil {
		image = *e.ImageURL
	}
	return []any{e.Code, string(e.Rule.Kind), solarMonth, solarDay, lunarMonth, lunarDay,
		e.Rule.IsLeapMonth, longitude, e.IsPublicHoliday, image, e.SortOrder}
}
