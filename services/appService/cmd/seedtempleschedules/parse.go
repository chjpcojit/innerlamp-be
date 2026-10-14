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

// supportedFormat is the only layout of assets/data/temple-schedules.json this
// seeder understands (meta.format).
const supportedFormat = 1

// fileJSON is the data file. It is decoded strictly: an unknown key anywhere is
// an error, so a misspelt field cannot load as an empty one.
type fileJSON struct {
	Meta  metaJSON   `json:"meta"`
	Items []itemJSON `json:"items"`
}

type metaJSON struct {
	Format int `json:"format"`
	// Source documents where the data came from (e.g. "mock"). Required; not
	// stored in the database.
	Source string `json:"source"`
}

// itemJSON uses pointers for end_time so "absent" is told apart from "00:00".
type itemJSON struct {
	ID          string     `json:"id"`
	TempleID    string     `json:"temple_id"`
	Name        string     `json:"name"`
	StartTime   string     `json:"start_time"`
	EndTime     *string    `json:"end_time"`
	Weekdays    []int      `json:"weekdays"`
	Description string     `json:"description"`
	Instructor  string     `json:"instructor"`
	Location    string     `json:"location"`
	Preparation string     `json:"preparation"`
	SortOrder   int        `json:"sort_order"`
	Steps       []stepJSON `json:"steps"`
}

type stepJSON struct {
	Position int     `json:"position"`
	Time     *string `json:"time"`
	Text     string  `json:"text"`
}

// parseReport is what parsing learned beyond the catalogue itself.
type parseReport struct {
	// Normalized counts strings changed by NFC normalisation (lossless, but
	// reported rather than silent).
	Normalized int
}

// parseCatalogue decodes the data file into a domain.TempleScheduleCatalogue.
// It checks the file's own structure — strict keys, layout version, HH:MM
// times — and leaves consistency to TempleScheduleCatalogue.Verify, which the
// caller runs on the result. It repairs nothing.
func parseCatalogue(src []byte) (domain.TempleScheduleCatalogue, parseReport, error) {
	var f fileJSON
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return domain.TempleScheduleCatalogue{}, parseReport{}, fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return domain.TempleScheduleCatalogue{}, parseReport{}, errors.New("decode: unexpected data after the JSON document")
	}
	if f.Meta.Format != supportedFormat {
		return domain.TempleScheduleCatalogue{}, parseReport{}, fmt.Errorf("meta.format is %d, this seeder reads format %d", f.Meta.Format, supportedFormat)
	}
	if f.Meta.Source == "" {
		return domain.TempleScheduleCatalogue{}, parseReport{}, errors.New("meta.source is empty")
	}

	var rep parseReport
	s := func(v string) string {
		out := norm.NFC.String(v)
		if out != v {
			rep.Normalized++
		}
		return out
	}

	c := domain.TempleScheduleCatalogue{Source: f.Meta.Source}
	for i, it := range f.Items {
		label := fmt.Sprintf("items[%d] %q", i, it.ID)
		start, err := domain.ParseTimeOfDay(it.StartTime)
		if err != nil {
			return domain.TempleScheduleCatalogue{}, parseReport{}, fmt.Errorf("%s: start_time: %w", label, err)
		}
		item := domain.TempleScheduleItem{
			ID: it.ID, TempleID: it.TempleID, Name: s(it.Name), StartTime: start,
			Weekdays: it.Weekdays, Description: s(it.Description), Instructor: s(it.Instructor),
			Location: s(it.Location), Preparation: s(it.Preparation), SortOrder: it.SortOrder,
		}
		if it.EndTime != nil {
			end, err := domain.ParseTimeOfDay(*it.EndTime)
			if err != nil {
				return domain.TempleScheduleCatalogue{}, parseReport{}, fmt.Errorf("%s: end_time: %w", label, err)
			}
			item.EndTime = &end
		}
		for _, st := range it.Steps {
			step := domain.TempleScheduleStep{Position: st.Position, Text: s(st.Text)}
			if st.Time != nil {
				tm, err := domain.ParseTimeOfDay(*st.Time)
				if err != nil {
					return domain.TempleScheduleCatalogue{}, parseReport{}, fmt.Errorf("%s: step %d time: %w", label, st.Position, err)
				}
				step.Time = &tm
			}
			item.Steps = append(item.Steps, step)
		}
		c.Items = append(c.Items, item)
	}
	return c, rep, nil
}
