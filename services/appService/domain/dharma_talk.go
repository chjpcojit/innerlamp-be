package domain

import (
	"context"
	"errors"
	"time"
)

// MediaType says how a dharma talk is consumed; the client picks its layout
// from it. It mirrors the CHECK on dharma_talks.media_type (migrations/0014).
type MediaType string

const (
	MediaYouTube MediaType = "youtube" // embedded via YouTubeID
	MediaAudio   MediaType = "audio"   // played from AudioURL
	MediaText    MediaType = "text"    // read-only, no media
)

// DharmaTalkChapter is a seekable section of a talk.
type DharmaTalkChapter struct {
	StartSeconds int
	Title        string
}

// FollowUpMeditation is the "sit after listening" card shown under a talk.
type FollowUpMeditation struct {
	Title        string
	Minutes      int
	MeditationID string // "" when not linked to a meditation yet
}

// DharmaTalk is one pháp thoại (a teacher's talk — not a sutra). It maps the
// dharma_talks table.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's.
type DharmaTalk struct {
	ID              string
	SortOrder       int
	Title           string
	Speaker         string
	SpeakerInitials string
	Category        string
	// Season is "" when the talk is not tied to an occasion (Vu Lan, Tết...).
	Season string
	// PublishedDate is nil when unknown.
	PublishedDate *time.Time
	// DurationMinutes is 0 when unknown.
	DurationMinutes int
	Summary         string
	KeyPoints       []string
	Quote           string
	QuoteSource     string
	Tags            []string
	Language        string
	MediaType       MediaType
	// YouTubeID is set only for MediaYouTube, AudioURL only for MediaAudio.
	YouTubeID string
	AudioURL  string
	// Chapters is ordered by StartSeconds; empty when none are known.
	Chapters []DharmaTalkChapter
	// FollowUpMeditation is nil when the talk has no follow-up card.
	FollowUpMeditation *FollowUpMeditation
	// NeedsReview hides the talk from every read until a human clears it.
	NeedsReview bool
	ReviewNote  string
	SourceName  string
	SourceURL   string
	License     string
}

// DharmaTalkPage is one page of the list plus what a client needs to know
// whether another page exists.
type DharmaTalkPage struct {
	Items      []DharmaTalk
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// ErrDharmaTalkNotFound — no displayable talk has the requested id (a talk
// flagged NeedsReview counts as absent).
var ErrDharmaTalkNotFound = errors.New("dharma talk not found")

// DharmaTalkRepository is the port the dharma talk use case reads and seeds through.
//
// Ordering is part of the contract: List returns talks by sort_order ASC,
// id ASC. List and FindByID only see talks with NeedsReview == false; Count and
// InsertMissing see every row, so the seed never mistakes "all hidden" for "empty".
type DharmaTalkRepository interface {
	// List returns limit displayable talks starting at offset, and the total
	// number of displayable talks. An offset past the end returns an empty
	// slice, not an error.
	List(ctx context.Context, offset, limit int) ([]DharmaTalk, int, error)

	// FindByID returns the displayable talk or ErrDharmaTalkNotFound.
	FindByID(ctx context.Context, id string) (*DharmaTalk, error)

	// Count returns the number of stored talks, hidden ones included.
	Count(ctx context.Context) (int, error)

	// InsertMissing inserts the talks whose id is not stored yet, atomically,
	// and returns how many were inserted. It never modifies an existing row.
	InsertMissing(ctx context.Context, talks []DharmaTalk) (int, error)
}
