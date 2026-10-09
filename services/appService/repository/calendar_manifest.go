package repository

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// calendarFileName is the only shape of file name the manifest may list: a plain
// name with an image extension, so it can be appended to a URL path as-is.
var calendarFileName = regexp.MustCompile(`^[A-Za-z0-9_-]+\.jpg$`)

type sourceCalendarImage struct {
	File       string `json:"file"`
	Author     string `json:"author"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url"`
	SourceURL  string `json:"source_url"`
}

// ParseCalendarManifest turns assets/images/home-calendar/manifest.json into the
// image list the calendar use case picks from. It stops at the first problem —
// an empty list, an unsafe or duplicate file name, a missing licence or author —
// and names the entry, so a bad manifest fails at startup rather than per request.
// The result is sorted by file name, so the date → image mapping does not depend on
// the order of entries in the manifest.
func ParseCalendarManifest(raw []byte) ([]domain.CalendarImage, error) {
	var entries []sourceCalendarImage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("decode calendar manifest: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("calendar manifest is empty")
	}

	seen := make(map[string]bool, len(entries))
	out := make([]domain.CalendarImage, 0, len(entries))
	for i, e := range entries {
		if !calendarFileName.MatchString(e.File) {
			return nil, fmt.Errorf("calendar manifest entry %d: invalid file name %q", i, e.File)
		}
		if seen[e.File] {
			return nil, fmt.Errorf("calendar manifest entry %d: duplicate file %q", i, e.File)
		}
		seen[e.File] = true
		if e.Author == "" || e.License == "" || e.SourceURL == "" {
			return nil, fmt.Errorf("calendar manifest %q: author, license and source_url are required", e.File)
		}
		out = append(out, domain.CalendarImage{
			ID: strings.TrimSuffix(e.File, ".jpg"), File: e.File,
			Author: e.Author, License: e.License, LicenseURL: e.LicenseURL, SourceURL: e.SourceURL,
		})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].File < out[b].File })
	return out, nil
}
