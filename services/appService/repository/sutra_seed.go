package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"strings"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sourceSutraIndex is the part of assets/data/sutras/index.json the seed needs:
// the list of ids, in curated order. The index also repeats each sutra's metadata
// for readers, but the per-sutra file is the authority.
type sourceSutraIndex struct {
	Sutras []struct {
		ID string `json:"id"`
	} `json:"sutras"`
}

// sourceSutra mirrors one assets/data/sutras/<id>.json file.
type sourceSutra struct {
	ID         string `json:"id"`
	Tradition  string `json:"tradition"`
	Kind       string `json:"kind"`
	Collection string `json:"collection"`
	Reference  string `json:"reference"`
	// The next six are optional; a missing field reads as "" (HanTranslationYear: raw,
	// so null / missing can be told from a bad value).
	OriginalTitle      string            `json:"originalTitle"`
	Author             string            `json:"author"`
	HanTranslator      string            `json:"hanTranslator"`
	HanTranslationYear json.RawMessage   `json:"hanTranslationYear"`
	ReferenceNote      string            `json:"referenceNote"`
	Summary            string            `json:"summary"`
	Titles             map[string]string `json:"titles"`
	Deities            []struct {
		DeityID string `json:"deity_id"`
		Role    string `json:"role"`
	} `json:"deities"`
	// DailyChant is a pointer so a missing field can be told from false.
	DailyChant *bool `json:"dailyChant"`
	// Purposes is nil when the field is missing, non-nil (possibly empty) when present.
	Purposes []string `json:"purposes"`
	// DurationMinutes is raw so a missing field (len 0) can be told from null.
	DurationMinutes json.RawMessage `json:"durationMinutes"`
	ReviewStatus    string          `json:"reviewStatus"`
	Translations    []struct {
		Key         string `json:"key"`
		Lang        string `json:"lang"`
		Variant     string `json:"variant"`
		Translator  string `json:"translator"`
		SourceName  string `json:"sourceName"`
		SourceURL   string `json:"sourceUrl"`
		License     string `json:"license"`
		LicenseURL  string `json:"licenseUrl"`
		LicenseNote string `json:"licenseNote"`
		Note        string `json:"note"`
		Lines       []struct {
			N       int    `json:"n"`
			Text    string `json:"text"`
			Section string `json:"section"`
		} `json:"lines"`
	} `json:"translations"`
}

// ParseSutraSeed reads index.json and every <id>.json it lists from fsys and turns
// them into domain seeds, validating everything the database would otherwise reject
// piecemeal — or worse, accept: unique ids that match their file names, values from
// the closed sets, the four fields the API depends on (key, dailyChant, purposes,
// durationMinutes), a key per translation that is unique within its sutra, line
// numbers running 1..N without gaps, and no deity listed twice in one sutra. It
// stops at the first problem and names the sutra. SortOrder is the position in
// index.json (see migrations/0013).
//
// fsys is the data directory itself (assets.SutrasFS rooted at data/sutras, or
// os.DirFS), so the same parser serves the embedded data and a directory on disk.
func ParseSutraSeed(fsys fs.FS) ([]domain.SutraSeed, error) {
	raw, err := fs.ReadFile(fsys, "index.json")
	if err != nil {
		return nil, fmt.Errorf("read index.json: %w", err)
	}
	var index sourceSutraIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("decode index.json: %w", err)
	}
	if len(index.Sutras) == 0 {
		return nil, fmt.Errorf("no sutras in index.json")
	}

	seen := make(map[string]bool, len(index.Sutras))
	out := make([]domain.SutraSeed, 0, len(index.Sutras))
	for i, entry := range index.Sutras {
		id := entry.ID
		if id == "" {
			return nil, fmt.Errorf("index.json entry %d: empty id", i)
		}
		if strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
			return nil, fmt.Errorf("sutra %q: id is not a plain file name", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate id %q in index.json", id)
		}
		seen[id] = true

		data, err := fs.ReadFile(fsys, id+".json")
		if err != nil {
			return nil, fmt.Errorf("sutra %q: read %s.json: %w", id, id, err)
		}
		seed, err := parseSutraFile(id, i, data)
		if err != nil {
			return nil, fmt.Errorf("sutra %q: %w", id, err)
		}
		out = append(out, *seed)
	}
	return out, nil
}

func parseSutraFile(fileID string, sortOrder int, data []byte) (*domain.SutraSeed, error) {
	var s sourceSutra
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // a typo'd key would otherwise read as "missing", or worse, be ignored
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if s.ID != fileID {
		return nil, fmt.Errorf("id %q does not match the file name", s.ID)
	}
	tradition, kind := domain.Tradition(s.Tradition), domain.Kind(s.Kind)
	if !tradition.Valid() {
		return nil, fmt.Errorf("unknown tradition %q", s.Tradition)
	}
	if !kind.Valid() {
		return nil, fmt.Errorf("unknown kind %q", s.Kind)
	}
	if strings.TrimSpace(s.Titles["vi"]) == "" {
		return nil, fmt.Errorf("titles.vi is empty")
	}
	if s.DailyChant == nil {
		return nil, fmt.Errorf("dailyChant is missing or not a boolean")
	}
	if s.Purposes == nil {
		return nil, fmt.Errorf("purposes is missing (use [] when none applies)")
	}
	purposes := make([]domain.Purpose, 0, len(s.Purposes))
	for _, p := range s.Purposes {
		if !domain.Purpose(p).Valid() {
			return nil, fmt.Errorf("unknown purpose %q", p)
		}
		purposes = append(purposes, domain.Purpose(p))
	}
	duration, err := parseDuration(s.DurationMinutes)
	if err != nil {
		return nil, err
	}
	hanYear, err := parseOptionalYear(s.HanTranslationYear)
	if err != nil {
		return nil, err
	}
	reviewStatus := s.ReviewStatus
	if reviewStatus == "" {
		reviewStatus = "UNREVIEWED"
	}

	deities := make([]domain.SutraDeity, 0, len(s.Deities))
	seenDeity := map[string]bool{}
	for _, d := range s.Deities {
		if d.DeityID == "" {
			return nil, fmt.Errorf("a deity link has an empty deity_id")
		}
		if seenDeity[d.DeityID] {
			return nil, fmt.Errorf("deity %q is listed twice", d.DeityID)
		}
		seenDeity[d.DeityID] = true
		if !domain.DeityRole(d.Role).Valid() {
			return nil, fmt.Errorf("deity %q: unknown role %q", d.DeityID, d.Role)
		}
		deities = append(deities, domain.SutraDeity{DeityID: d.DeityID, Role: domain.DeityRole(d.Role)})
	}

	if len(s.Translations) == 0 {
		return nil, fmt.Errorf("no translations")
	}
	translations := make([]domain.SutraTranslationSeed, 0, len(s.Translations))
	seenKey := map[string]bool{}
	for ti, t := range s.Translations {
		if t.Key == "" {
			return nil, fmt.Errorf("translation %d has no key", ti)
		}
		if seenKey[t.Key] {
			return nil, fmt.Errorf("translation key %q is used twice", t.Key)
		}
		seenKey[t.Key] = true
		if t.Lang == "" {
			return nil, fmt.Errorf("translation %q: empty lang", t.Key)
		}
		if !domain.Variant(t.Variant).Valid() {
			return nil, fmt.Errorf("translation %q: unknown variant %q", t.Key, t.Variant)
		}
		if !domain.ValidLicense(t.License) {
			return nil, fmt.Errorf("translation %q: unknown license %q", t.Key, t.License)
		}
		if len(t.Lines) == 0 {
			return nil, fmt.Errorf("translation %q: no lines", t.Key)
		}
		lines := make([]domain.SutraLine, len(t.Lines))
		for li, l := range t.Lines {
			if l.N != li+1 {
				return nil, fmt.Errorf("translation %q: line %d has n=%d, want %d (n must run 1..N without gaps)", t.Key, li, l.N, li+1)
			}
			lines[li] = domain.SutraLine{N: l.N, Text: l.Text, Section: l.Section}
		}
		translations = append(translations, domain.SutraTranslationSeed{
			SutraTranslation: domain.SutraTranslation{
				Key: t.Key, Lang: t.Lang, Variant: domain.Variant(t.Variant),
				Translator: t.Translator, SourceName: t.SourceName, SourceURL: t.SourceURL,
				License: t.License, LicenseURL: t.LicenseURL, LicenseNote: t.LicenseNote, Note: t.Note, LineCount: len(lines), Sections: sectionsOf(lines),
			},
			Lines: lines,
		})
	}

	return &domain.SutraSeed{
		Sutra: domain.Sutra{
			ID: s.ID, SortOrder: sortOrder, Tradition: tradition, Kind: kind,
			Collection: s.Collection, Reference: s.Reference, Titles: s.Titles,
			OriginalTitle: s.OriginalTitle, Author: s.Author, HanTranslator: s.HanTranslator,
			HanTranslationYear: hanYear, ReferenceNote: s.ReferenceNote, Summary: s.Summary,
			DailyChant: *s.DailyChant, Purposes: purposes, DurationMinutes: duration,
			ReviewStatus: reviewStatus, Deities: deities,
		},
		Translations: translations,
	}, nil
}

// parseDuration reads durationMinutes: it must be present, and either null or an
// integer >= 1. nil result means null.
func parseDuration(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("durationMinutes is missing (use null when unknown)")
	}
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, fmt.Errorf("durationMinutes must be null or an integer: %w", err)
	}
	if n < 1 || n > math.MaxInt32 {
		return nil, fmt.Errorf("durationMinutes must be >= 1, got %d", n)
	}
	return &n, nil
}

// parseOptionalYear reads hanTranslationYear: missing or null means unknown (nil);
// otherwise it must be an integer >= 1.
func parseOptionalYear(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, fmt.Errorf("hanTranslationYear must be null or an integer: %w", err)
	}
	if n < 1 {
		return nil, fmt.Errorf("hanTranslationYear must be >= 1, got %d", n)
	}
	return &n, nil
}

// sectionsOf builds the table of contents of a translation: one entry for each run
// of consecutive lines that share the same non-empty section name. Lines with no
// section belong to no entry, and a name that comes back after another section
// starts a new entry. StartN is the N of the run's first line, so a client can turn
// it into a page with floor((StartN-1)/pageSize)+1.
func sectionsOf(lines []domain.SutraLine) []domain.SutraSection {
	var out []domain.SutraSection
	for _, l := range lines {
		if l.Section == "" {
			continue
		}
		if last := len(out) - 1; last >= 0 && out[last].Name == l.Section && out[last].StartN+out[last].LineCount == l.N {
			out[last].LineCount++
			continue
		}
		out = append(out, domain.SutraSection{Name: l.Section, StartN: l.N, LineCount: 1})
	}
	return out
}
