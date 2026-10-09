package domain

import (
	"context"
	"errors"
)

// Deity types as stored in deities.type and the source data.
const (
	DeityBuddha      = "BUDDHA"
	DeityBodhisattva = "BODHISATTVA"
	DeityDeva        = "DEVA"
)

// Deity is one Phật or Bồ tát in the browse catalogue. It maps the deities table.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's.
type Deity struct {
	ID                string
	Type              string
	SortOrder         int
	NameVI            string
	NameSanskrit      string
	NameChinese       string
	TitleHonorific    string
	ThumbnailURL      string
	ImageURL          string
	CanonicalSource   string
	CoreVirtue        string
	PureLand          string
	LunarMemorialDays []MemorialDay
	Iconography       Iconography
	BiographySummary  string
	GreatVows         []string
	Mantras           Mantras
}

// MemorialDay is one lunar-calendar observance (vía, đản sanh, thành đạo...).
type MemorialDay struct {
	LunarDay   int
	LunarMonth int
	EventName  string
}

// Iconography describes how the deity is depicted.
type Iconography struct {
	Posture         string
	MountOrThrone   string
	SacredArtifacts []string
	SymbolicMeaning string
}

// Mantras are the deity's recitation texts.
type Mantras struct {
	ShortSanskrit          string
	ShortSinoVietnamese    string
	LongDharaniName        string
	DharaniPhoneticSnippet string
}

// DeityPage is one page of a deity list plus what a client needs to know
// whether another page exists.
type DeityPage struct {
	Items      []Deity
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// ErrDeityNotFound — no deity has the requested id.
var ErrDeityNotFound = errors.New("deity not found")

// DeityRepository is the port the deity use case reads through, and the seeder
// writes through.
//
// Ordering is part of the contract: every read returns deities by
// sort_order ASC, id ASC — deterministic, so paging never repeats or skips a row.
type DeityRepository interface {
	// ListTop returns the first limit deities of any type in order.
	ListTop(ctx context.Context, limit int) ([]Deity, error)

	// List returns limit deities of any type (Buddhas and bodhisattvas in one
	// ordered list) starting at offset, and the total number of deities. An
	// offset past the end returns an empty slice, not an error.
	List(ctx context.Context, offset, limit int) ([]Deity, int, error)

	// FindByID returns the deity or ErrDeityNotFound.
	FindByID(ctx context.Context, id string) (*Deity, error)

	// Upsert inserts or updates deities keyed by id, idempotently, atomically.
	Upsert(ctx context.Context, deities []Deity) error
}
