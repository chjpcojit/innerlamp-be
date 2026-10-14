package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// CalendarEventUsecase answers "which events fall in this solar year". The
// catalogue holds repeat rules, not dates: each request reads the catalogue and
// computes the year's dates (domain.ResolveOccurrences), storing nothing.
type CalendarEventUsecase struct {
	repo   domain.CalendarEventRepository
	lang   string
	loc    *time.Location
	now    func() time.Time
	logger logging.Logger
}

// NewCalendarEventUsecase wires the use case. cfg.Timezone must be a valid IANA
// zone (the zone whose "this year" an omitted year means) and cfg.DefaultLanguage
// non-empty. now is the clock, injected so tests can fix "today".
func NewCalendarEventUsecase(repo domain.CalendarEventRepository, cfg config.CalendarConfig, now func() time.Time, logger logging.Logger) (*CalendarEventUsecase, error) {
	if cfg.DefaultLanguage == "" {
		return nil, fmt.Errorf("calendar default_language is required")
	}
	if cfg.Timezone == "" { // LoadLocation("") silently means UTC
		return nil, fmt.Errorf("calendar timezone is required")
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("calendar timezone %q: %w", cfg.Timezone, err)
	}
	return &CalendarEventUsecase{repo: repo, lang: cfg.DefaultLanguage, loc: loc, now: now, logger: logger}, nil
}

// EventsForYear returns the groups, events and occurrences of a solar year.
// year is the raw query value: empty means the current year in the configured
// zone; otherwise it must be a plain whole number in 1900..2100, else
// domain.ErrInvalidYear (nothing is read from the database).
func (u *CalendarEventUsecase) EventsForYear(ctx context.Context, year string) (domain.CalendarEventsResult, error) {
	y, err := u.parseYear(year)
	if err != nil {
		return domain.CalendarEventsResult{}, err
	}

	groups, err := u.repo.ListGroups(ctx)
	if err != nil {
		return domain.CalendarEventsResult{}, fmt.Errorf("list calendar groups: %w", err)
	}
	events, err := u.repo.ListActiveEvents(ctx, u.lang)
	if err != nil {
		return domain.CalendarEventsResult{}, fmt.Errorf("list calendar events: %w", err)
	}

	// A rule that fails validation (data edited by hand) must not take the whole
	// response down: skip that event and say so. The seed's Verify makes this rare.
	valid := make([]domain.CalendarEvent, 0, len(events))
	for _, e := range events {
		if err := e.Rule.Validate(); err != nil {
			u.logger.Warn("calendar event skipped: invalid recurrence rule", "event_code", e.Code, "error", err.Error())
			continue
		}
		valid = append(valid, e)
	}

	occurrences := domain.ResolveOccurrences(valid, y)

	known := make(map[string]bool, len(groups))
	for _, g := range groups {
		known[g.Code] = true
	}
	occurs := make(map[string]bool, len(occurrences))
	for _, o := range occurrences {
		occurs[o.EventCode] = true
	}
	result := domain.CalendarEventsResult{
		Year:        y,
		Groups:      groups,
		Events:      []domain.CalendarEvent{},
		Occurrences: occurrences,
	}
	if result.Groups == nil {
		result.Groups = []domain.CalendarGroup{}
	}
	for _, e := range valid {
		if !occurs[e.Code] {
			continue
		}
		// A group code missing from the catalogue (back-office changed it between
		// the two reads) is dropped, not an error.
		kept := make([]string, 0, len(e.Groups))
		for _, g := range e.Groups {
			if known[g] {
				kept = append(kept, g)
			}
		}
		e.Groups = kept
		result.Events = append(result.Events, e)
	}
	return result, nil
}

func (u *CalendarEventUsecase) parseYear(year string) (int, error) {
	if year == "" {
		return u.now().In(u.loc).Year(), nil
	}
	// Atoi accepts a sign ("+2026") and nothing else odd; require plain digits.
	for _, r := range year {
		if r < '0' || r > '9' {
			return 0, domain.ErrInvalidYear
		}
	}
	y, err := strconv.Atoi(year)
	if err != nil || y < calendarMinYear || y > calendarMaxYear {
		return 0, domain.ErrInvalidYear
	}
	return y, nil
}
