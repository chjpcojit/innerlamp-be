package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/text/unicode/norm"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// supportedFormat is the only layout of assets/data/calendar-events.json this
// seeder understands (meta.format).
const supportedFormat = 1

// fileJSON is the data file. It is decoded strictly: an unknown key anywhere
// but inside meta is an error, so a misspelt parameter cannot load as zero.
type fileJSON struct {
	Meta   json.RawMessage `json:"meta"`
	Groups []groupJSON     `json:"groups"`
	Events []eventJSON     `json:"events"`
}

type groupJSON struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// eventJSON uses pointers for the rule parameters so "absent" is told apart
// from 0 — 0° is a real solar term (Xuân phân).
type eventJSON struct {
	Code               string `json:"code"`
	Kind               string `json:"kind"`
	SolarMonth         *int   `json:"solar_month"`
	SolarDay           *int   `json:"solar_day"`
	LunarMonth         *int   `json:"lunar_month"`
	LunarDay           *int   `json:"lunar_day"`
	IsLeapMonth        bool   `json:"is_leap_month"`
	SolarTermLongitude *int   `json:"solar_term_longitude"`
	IsPublicHoliday    bool   `json:"is_public_holiday"`
	// IsMajor is optional; absent means false (BR-CAL-10).
	IsMajor          bool     `json:"is_major"`
	SortOrder        int      `json:"sort_order"`
	Groups           []string `json:"groups"`
	Name             string   `json:"name"`
	ShortDescription string   `json:"short_description"`
	LongDescription  string   `json:"long_description"`
	Guidance         string   `json:"guidance"`
	ImageURL         *string  `json:"image_url"`
	// Source documents where the event's date and text came from, for review.
	// It is required but not loaded into the database.
	Source string `json:"source"`
}

type metaJSON struct {
	Format int    `json:"format"`
	Lang   string `json:"lang"`
}

// parseReport is what parsing learned beyond the catalogue itself.
type parseReport struct {
	// Normalized counts strings changed by NFC normalisation (lossless, but
	// reported rather than silent).
	Normalized int
}

// paramsByKind lists which rule parameters a kind requires. Every other
// parameter must be absent: a stray one is a typo or a wrong kind, not data.
var paramsByKind = map[string][]string{
	"solar":          {"solar_month", "solar_day"},
	"lunar":          {"lunar_month", "lunar_day"},
	"lunar_monthly":  {"lunar_day"},
	"lunar_last_day": {"lunar_month"},
	"solar_term":     {"solar_term_longitude"},
}

// parseCatalogue decodes the data file into a domain.CalendarCatalogue. It
// checks the file's own structure — layout version, language, required
// parameters per kind — and leaves consistency to CalendarCatalogue.Verify,
// which the caller runs on the result. It repairs nothing.
func parseCatalogue(src []byte) (domain.CalendarCatalogue, parseReport, error) {
	var f fileJSON
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return domain.CalendarCatalogue{}, parseReport{}, fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return domain.CalendarCatalogue{}, parseReport{}, errors.New("decode: unexpected data after the JSON document")
	}
	if len(f.Meta) == 0 {
		return domain.CalendarCatalogue{}, parseReport{}, errors.New(`missing "meta"`)
	}
	var meta metaJSON
	if err := json.Unmarshal(f.Meta, &meta); err != nil {
		return domain.CalendarCatalogue{}, parseReport{}, fmt.Errorf("decode meta: %w", err)
	}
	if meta.Format != supportedFormat {
		return domain.CalendarCatalogue{}, parseReport{}, fmt.Errorf("meta.format is %d, this seeder reads format %d", meta.Format, supportedFormat)
	}
	if meta.Lang == "" {
		return domain.CalendarCatalogue{}, parseReport{}, errors.New("meta.lang is empty")
	}

	var rep parseReport
	s := func(v string) string {
		out := norm.NFC.String(v)
		if out != v {
			rep.Normalized++
		}
		return out
	}

	c := domain.CalendarCatalogue{Lang: meta.Lang}
	for _, g := range f.Groups {
		c.Groups = append(c.Groups, domain.CalendarGroup{Code: g.Code, Name: s(g.Name), SortOrder: g.SortOrder})
	}

	var problems []string
	for i, e := range f.Events {
		label := fmt.Sprintf("events[%d] %q", i, e.Code)
		if e.Source == "" {
			problems = append(problems, label+": source is empty")
		}
		present := map[string]bool{
			"solar_month": e.SolarMonth != nil, "solar_day": e.SolarDay != nil,
			"lunar_month": e.LunarMonth != nil, "lunar_day": e.LunarDay != nil,
			"solar_term_longitude": e.SolarTermLongitude != nil,
		}
		required, known := paramsByKind[e.Kind]
		if !known {
			problems = append(problems, fmt.Sprintf("%s: unknown kind %q", label, e.Kind))
		} else {
			want := map[string]bool{}
			for _, p := range required {
				want[p] = true
				if !present[p] {
					problems = append(problems, fmt.Sprintf("%s: kind %q requires %s", label, e.Kind, p))
				}
			}
			for p, ok := range present {
				if ok && !want[p] {
					problems = append(problems, fmt.Sprintf("%s: kind %q does not take %s", label, e.Kind, p))
				}
			}
		}
		val := func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		}
		var image *string
		if e.ImageURL != nil {
			v := *e.ImageURL
			image = &v
		}
		c.Events = append(c.Events, domain.CalendarEvent{
			Code:             e.Code,
			Name:             s(e.Name),
			ShortDescription: s(e.ShortDescription),
			LongDescription:  s(e.LongDescription),
			Guidance:         s(e.Guidance),
			ImageURL:         image,
			Groups:           e.Groups,
			IsPublicHoliday:  e.IsPublicHoliday,
			IsMajor:          e.IsMajor,
			SortOrder:        e.SortOrder,
			Rule: domain.RecurrenceRule{
				Kind:               domain.CalendarEventKind(e.Kind),
				SolarMonth:         val(e.SolarMonth),
				SolarDay:           val(e.SolarDay),
				LunarMonth:         val(e.LunarMonth),
				LunarDay:           val(e.LunarDay),
				IsLeapMonth:        e.IsLeapMonth,
				SolarTermLongitude: val(e.SolarTermLongitude),
			},
		})
	}
	if len(problems) > 0 {
		return domain.CalendarCatalogue{}, parseReport{}, fmt.Errorf("%d malformed event(s), first: %s", len(problems), problems[0])
	}
	return c, rep, nil
}
