package repository

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sourceDharmaTalk mirrors one object of assets/data/dharma_talks.json.
type sourceDharmaTalk struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Speaker         string   `json:"speaker"`
	SpeakerInitials string   `json:"speakerInitials"`
	Category        string   `json:"category"`
	Season          string   `json:"season"`
	PublishedDate   *string  `json:"publishedDate"`
	DurationMinutes *int     `json:"durationMinutes"`
	Summary         string   `json:"summary"`
	KeyPoints       []string `json:"keyPoints"`
	Quote           string   `json:"quote"`
	QuoteSource     string   `json:"quoteSource"`
	Tags            []string `json:"tags"`
	Language        string   `json:"language"`
	// MediaType is "video" (a YouTube video), "audio" or "text" in the file.
	MediaType string  `json:"mediaType"`
	MediaURL  *string `json:"mediaUrl"`
	Chapters  []struct {
		StartSeconds int    `json:"startSeconds"`
		Title        string `json:"title"`
	} `json:"chapters"`
	NeedsReview bool   `json:"needsReview"`
	ReviewNote  string `json:"reviewNote"`
	SourceName  string `json:"sourceName"`
	SourceURL   string `json:"sourceUrl"`
	License     string `json:"license"`
}

// youtubeIDPattern pulls the 11-character id out of youtube.com/watch?v=<id> or
// youtu.be/<id>; the lookahead rejects ids that are longer than 11 characters.
var youtubeIDPattern = regexp.MustCompile(`^https?://(?:(?:www\.|m\.)?youtube\.com/watch\?(?:[^#]*&)?v=|youtu\.be/)([A-Za-z0-9_-]{11})(?:[^A-Za-z0-9_-]|$)`)

// ParseDharmaTalkSeed turns the source JSON ({"talks":[...]}) into domain
// talks, validating what the database would otherwise reject one row at a time
// (unique non-empty ids, consistent media columns, sane chapters). The file's
// "video" becomes domain.MediaYouTube with the id extracted from mediaUrl.
// SortOrder is the position in the file (see migrations/0014).
func ParseDharmaTalkSeed(src []byte) ([]domain.DharmaTalk, error) {
	var doc struct {
		Talks []sourceDharmaTalk `json:"talks"`
	}
	if err := json.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("decode dharma talks: %w", err)
	}
	if len(doc.Talks) == 0 {
		return nil, fmt.Errorf("no dharma talks in source")
	}

	seen := make(map[string]bool, len(doc.Talks))
	out := make([]domain.DharmaTalk, 0, len(doc.Talks))
	for i, s := range doc.Talks {
		if s.ID == "" {
			return nil, fmt.Errorf("entry %d: empty id", i)
		}
		if seen[s.ID] {
			return nil, fmt.Errorf("duplicate id %q", s.ID)
		}
		seen[s.ID] = true
		if strings.TrimSpace(s.Title) == "" {
			return nil, fmt.Errorf("dharma talk %q: empty title", s.ID)
		}
		if strings.TrimSpace(s.Speaker) == "" {
			return nil, fmt.Errorf("dharma talk %q: empty speaker", s.ID)
		}

		t := domain.DharmaTalk{
			ID: s.ID, SortOrder: i, Title: s.Title, Speaker: s.Speaker, SpeakerInitials: s.SpeakerInitials,
			Category: s.Category, Season: s.Season, Summary: s.Summary, KeyPoints: s.KeyPoints,
			Quote: s.Quote, QuoteSource: s.QuoteSource, Tags: s.Tags, Language: s.Language,
			NeedsReview: s.NeedsReview, ReviewNote: s.ReviewNote,
			SourceName: s.SourceName, SourceURL: s.SourceURL, License: s.License,
		}

		if err := applySeedMedia(&t, s); err != nil {
			return nil, err
		}
		if s.DurationMinutes != nil {
			if *s.DurationMinutes <= 0 {
				return nil, fmt.Errorf("dharma talk %q: durationMinutes must be > 0, got %d", s.ID, *s.DurationMinutes)
			}
			t.DurationMinutes = *s.DurationMinutes
		}
		if s.PublishedDate != nil && *s.PublishedDate != "" {
			d, err := dateOnly(*s.PublishedDate)
			if err != nil {
				return nil, fmt.Errorf("dharma talk %q: bad publishedDate %q: %w", s.ID, *s.PublishedDate, err)
			}
			t.PublishedDate = &d
		}
		prev := 0
		for j, c := range s.Chapters {
			if c.StartSeconds < 0 {
				return nil, fmt.Errorf("dharma talk %q: chapter %d: negative startSeconds %d", s.ID, j, c.StartSeconds)
			}
			if strings.TrimSpace(c.Title) == "" {
				return nil, fmt.Errorf("dharma talk %q: chapter %d: empty title", s.ID, j)
			}
			if c.StartSeconds < prev {
				return nil, fmt.Errorf("dharma talk %q: chapter %d starts at %ds, before the previous chapter (%ds)", s.ID, j, c.StartSeconds, prev)
			}
			prev = c.StartSeconds
			t.Chapters = append(t.Chapters, domain.DharmaTalkChapter{StartSeconds: c.StartSeconds, Title: c.Title})
		}
		out = append(out, t)
	}
	return out, nil
}

// applySeedMedia maps the file's mediaType/mediaUrl onto MediaType plus the one
// media column that type uses.
func applySeedMedia(t *domain.DharmaTalk, s sourceDharmaTalk) error {
	url := ""
	if s.MediaURL != nil {
		url = *s.MediaURL
	}
	switch s.MediaType {
	case "video":
		m := youtubeIDPattern.FindStringSubmatch(url)
		if m == nil {
			return fmt.Errorf("dharma talk %q: mediaUrl %q is not a YouTube watch URL with an 11-character id", s.ID, url)
		}
		t.MediaType, t.YouTubeID = domain.MediaYouTube, m[1]
	case "audio":
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return fmt.Errorf("dharma talk %q: audio mediaUrl %q must be an http(s) URL", s.ID, url)
		}
		t.MediaType, t.AudioURL = domain.MediaAudio, url
	case "text":
		t.MediaType = domain.MediaText
	default:
		return fmt.Errorf("dharma talk %q: unknown mediaType %q (want video, audio or text)", s.ID, s.MediaType)
	}
	return nil
}
