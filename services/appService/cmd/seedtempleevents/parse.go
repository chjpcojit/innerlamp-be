package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/text/unicode/norm"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// supportedFormat is the only layout of assets/data/temple-events.json this
// seeder understands (meta.format).
const supportedFormat = 1

// fileJSON is the data file. It is decoded strictly: an unknown key anywhere is
// an error, so a misspelt field cannot load as an empty one.
type fileJSON struct {
	Meta   metaJSON    `json:"meta"`
	Events []eventJSON `json:"events"`
}

type metaJSON struct {
	Format int `json:"format"`
	// Source documents where the data came from (e.g. "mock"). Required; not
	// stored in the database.
	Source string `json:"source"`
}

// eventJSON uses pointers for the optional values so "absent" is told apart from
// zero (weekday 0 is Sunday).
type eventJSON struct {
	ID                  string  `json:"id"`
	TempleID            string  `json:"temple_id"`
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	Kind                string  `json:"kind"`
	StartsAt            string  `json:"starts_at"`
	EndsAt              string  `json:"ends_at"`
	IsAllDay            bool    `json:"is_all_day"`
	Status              string  `json:"status"`
	AudienceNote        string  `json:"audience_note"`
	ContactInfo         string  `json:"contact_info"`
	ImageURL            *string `json:"image_url"`
	Recurrence          string  `json:"recurrence"`
	RecurrenceLunarDays []int   `json:"recurrence_lunar_days"`
	RecurrenceWeekday   *int    `json:"recurrence_weekday"`
	RecurrenceUntil     *string `json:"recurrence_until"`
}

// parseReport is what parsing learned beyond the catalogue itself.
type parseReport struct {
	// Normalized counts strings changed by NFC normalisation (lossless, but
	// reported rather than silent).
	Normalized int
}

// parseCatalogue decodes the data file into a domain.TempleEventCatalogue.
// starts_at and ends_at must be RFC 3339 with an offset; recurrence_until is a
// plain YYYY-MM-DD. It checks the file's own structure and leaves consistency
// to TempleEventCatalogue.Verify, which the caller runs on the result. It
// repairs nothing.
func parseCatalogue(src []byte) (domain.TempleEventCatalogue, parseReport, error) {
	var zero domain.TempleEventCatalogue
	var f fileJSON
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return zero, parseReport{}, fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return zero, parseReport{}, errors.New("decode: unexpected data after the JSON document")
	}
	if f.Meta.Format != supportedFormat {
		return zero, parseReport{}, fmt.Errorf("meta.format is %d, this seeder reads format %d", f.Meta.Format, supportedFormat)
	}
	if f.Meta.Source == "" {
		return zero, parseReport{}, errors.New("meta.source is empty")
	}

	var rep parseReport
	s := func(v string) string {
		out := norm.NFC.String(v)
		if out != v {
			rep.Normalized++
		}
		return out
	}

	c := domain.TempleEventCatalogue{Source: f.Meta.Source}
	for i, e := range f.Events {
		label := fmt.Sprintf("events[%d] %q", i, e.ID)
		start, err := time.Parse(time.RFC3339, e.StartsAt)
		if err != nil {
			return zero, parseReport{}, fmt.Errorf("%s: starts_at: %w", label, err)
		}
		end, err := time.Parse(time.RFC3339, e.EndsAt)
		if err != nil {
			return zero, parseReport{}, fmt.Errorf("%s: ends_at: %w", label, err)
		}
		ev := domain.TempleEvent{
			ID: e.ID, TempleID: e.TempleID, Title: s(e.Title), Description: s(e.Description),
			Kind: domain.TempleEventKind(e.Kind), StartsAt: start, EndsAt: end, IsAllDay: e.IsAllDay,
			Status: domain.TempleEventStatus(e.Status), AudienceNote: s(e.AudienceNote), ContactInfo: s(e.ContactInfo),
			Recurrence: domain.TempleRecurrence(e.Recurrence), LunarDays: e.RecurrenceLunarDays,
			Weekday: e.RecurrenceWeekday,
		}
		if e.ImageURL != nil {
			v := *e.ImageURL
			ev.ImageURL = &v
		}
		if e.RecurrenceUntil != nil {
			u, err := time.Parse("2006-01-02", *e.RecurrenceUntil)
			if err != nil {
				return zero, parseReport{}, fmt.Errorf("%s: recurrence_until: %w", label, err)
			}
			ev.Until = &u
		}
		c.Events = append(c.Events, ev)
	}
	return c, rep, nil
}
