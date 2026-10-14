package usecase

import (
	"context"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

const (
	templeEventDateLayout = "2006-01-02"
	templeEventMinYear    = 1900
	templeEventMaxYear    = 2100
)

// TempleEventUsecase answers "which events does this temple hold in this
// window". Events are stored with repeat rules; each request reads the temple's
// events and computes the occurrences in the window, storing nothing.
type TempleEventUsecase struct {
	temples domain.TempleRepository
	events  domain.TempleEventRepository
	cfg     config.TempleEventConfig
	loc     *time.Location
	now     func() time.Time
	logger  logging.Logger
}

// NewTempleEventUsecase wires the use case. cfg must pass Validate (a valid
// IANA zone and 1 <= default <= max window months). now is the clock, injected
// so tests can fix "today" and "already ended".
func NewTempleEventUsecase(temples domain.TempleRepository, events domain.TempleEventRepository,
	cfg config.TempleEventConfig, now func() time.Time, logger logging.Logger) (*TempleEventUsecase, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("temple_event timezone %q: %w", cfg.Timezone, err)
	}
	return &TempleEventUsecase{temples: temples, events: events, cfg: cfg, loc: loc, now: now, logger: logger}, nil
}

// EventsForTemple returns the events of a temple that have an occurrence in the
// window [from, to] (YYYY-MM-DD, both inclusive) and have not ended yet. An
// empty from means today and an empty to means from + default_window_months,
// both in the configured zone. Bad dates, a reversed window or a window over
// max_window_months fail before anything is read.
func (u *TempleEventUsecase) EventsForTemple(ctx context.Context, templeID, from, to string) (domain.TempleEventsResult, error) {
	w, err := u.window(from, to)
	if err != nil {
		return domain.TempleEventsResult{}, err
	}

	if _, err := u.temples.FindByID(ctx, templeID); err != nil {
		return domain.TempleEventsResult{}, fmt.Errorf("find temple %q: %w", templeID, err)
	}
	events, err := u.events.ListActiveByTemple(ctx, templeID, w)
	if err != nil {
		return domain.TempleEventsResult{}, fmt.Errorf("list events of temple %q: %w", templeID, err)
	}

	// Data edited by hand can be inconsistent: skip that event and say so
	// rather than failing the whole response. The seed's Verify makes this rare.
	valid := make([]domain.TempleEvent, 0, len(events))
	for _, e := range events {
		if err := e.Validate(u.loc); err != nil {
			u.logger.Warn("temple event skipped: invalid data", "event_id", e.ID, "error", err.Error())
			continue
		}
		valid = append(valid, e)
	}

	occurrences := domain.ResolveTempleOccurrences(valid, w, u.now(), u.loc)
	has := make(map[string]bool, len(occurrences))
	for _, o := range occurrences {
		has[o.EventID] = true
	}
	res := domain.TempleEventsResult{
		TempleID:    templeID,
		From:        w.From,
		To:          w.To,
		Events:      []domain.TempleEvent{},
		Occurrences: occurrences,
	}
	if res.Occurrences == nil {
		res.Occurrences = []domain.TempleEventOccurrence{}
	}
	for _, e := range valid {
		if has[e.ID] {
			res.Events = append(res.Events, e)
		}
	}
	return res, nil
}

// window turns the raw query values into a checked window.
func (u *TempleEventUsecase) window(from, to string) (domain.TempleEventWindow, error) {
	y, m, d := u.now().In(u.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	start := today
	if from != "" {
		t, err := parseTempleEventDate("from", from)
		if err != nil {
			return domain.TempleEventWindow{}, err
		}
		start = t
	}
	var end time.Time
	if to != "" {
		t, err := parseTempleEventDate("to", to)
		if err != nil {
			return domain.TempleEventWindow{}, err
		}
		end = t
	} else {
		end = start.AddDate(0, u.cfg.DefaultWindowMonths, 0)
	}

	if end.Before(start) {
		return domain.TempleEventWindow{}, domain.ErrInvalidDateRange
	}
	if end.After(start.AddDate(0, u.cfg.MaxWindowMonths, 0)) {
		return domain.TempleEventWindow{}, domain.ErrDateRangeTooLarge
	}
	return domain.TempleEventWindow{From: start, To: end}, nil
}

// parseTempleEventDate reads a plain YYYY-MM-DD in the supported years.
func parseTempleEventDate(field, s string) (time.Time, error) {
	t, err := time.Parse(templeEventDateLayout, s)
	if err != nil || t.Year() < templeEventMinYear || t.Year() > templeEventMaxYear {
		return time.Time{}, &domain.InvalidDateFieldError{Field: field}
	}
	return t, nil
}
