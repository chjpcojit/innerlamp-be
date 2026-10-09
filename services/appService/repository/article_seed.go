package repository

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sourceArticle mirrors one object of assets/data/buddhist_articles.json.
type sourceArticle struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Category       string   `json:"category"`
	Author         string   `json:"author"`
	AuthorInitials string   `json:"authorInitials"`
	PublishedDate  *string  `json:"publishedDate"`
	ReadMinutes    int      `json:"readMinutes"`
	Summary        string   `json:"summary"`
	BodyParagraphs []string `json:"bodyParagraphs"`
	Quote          string   `json:"quote"`
	QuoteSource    string   `json:"quoteSource"`
	Tags           []string `json:"tags"`
	Language       string   `json:"language"`
	ImageURL       *string  `json:"imageURL"`
	ThumbnailURL   *string  `json:"thumbnailURL"`
	SourceName     string   `json:"sourceName"`
	SourceURL      string   `json:"sourceURL"`
	License        string   `json:"license"`
}

// ParseArticleSeed turns the source JSON ({"articles":[...]}) into domain
// articles, validating what the database would otherwise reject one row at a
// time: unique non-empty ids, a title, a parseable date. SortOrder is the
// position in the file (see migrations/0012).
func ParseArticleSeed(src []byte) ([]domain.Article, error) {
	var doc struct {
		Articles []sourceArticle `json:"articles"`
	}
	if err := json.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("decode articles: %w", err)
	}
	if len(doc.Articles) == 0 {
		return nil, fmt.Errorf("no articles in source")
	}

	seen := make(map[string]bool, len(doc.Articles))
	out := make([]domain.Article, 0, len(doc.Articles))
	for i, s := range doc.Articles {
		if s.ID == "" {
			return nil, fmt.Errorf("entry %d: empty id", i)
		}
		if seen[s.ID] {
			return nil, fmt.Errorf("duplicate id %q", s.ID)
		}
		seen[s.ID] = true
		if strings.TrimSpace(s.Title) == "" {
			return nil, fmt.Errorf("article %q: empty title", s.ID)
		}

		a := domain.Article{
			ID: s.ID, SortOrder: i, Title: s.Title, Category: s.Category, Author: s.Author,
			AuthorInitials: s.AuthorInitials, ReadMinutes: s.ReadMinutes, Summary: s.Summary,
			BodyParagraphs: s.BodyParagraphs, Quote: s.Quote, QuoteSource: s.QuoteSource, Tags: s.Tags,
			Language: s.Language, SourceName: s.SourceName, SourceURL: s.SourceURL, License: s.License,
		}
		if s.PublishedDate != nil && *s.PublishedDate != "" {
			d, err := dateOnly(*s.PublishedDate)
			if err != nil {
				return nil, fmt.Errorf("article %q: bad publishedDate %q: %w", s.ID, *s.PublishedDate, err)
			}
			a.PublishedDate = &d
		}
		if s.ImageURL != nil {
			a.ImageURL = *s.ImageURL
		}
		if s.ThumbnailURL != nil {
			a.ThumbnailURL = *s.ThumbnailURL
		}
		out = append(out, a)
	}
	return out, nil
}

// dateOnly parses a source "YYYY-MM-DD".
func dateOnly(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }
