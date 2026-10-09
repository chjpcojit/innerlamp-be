package domain

import (
	"context"
	"errors"
	"time"
)

// Article is one Phật pháp reading. It maps the articles table.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's.
type Article struct {
	ID             string
	SortOrder      int
	Title          string
	Category       string
	Author         string
	AuthorInitials string
	// PublishedDate is nil when the source has no date (most articles).
	PublishedDate  *time.Time
	ReadMinutes    int
	Summary        string
	BodyParagraphs []string
	Quote          string
	QuoteSource    string
	Tags           []string
	Language       string
	// ImageURL is "" when the article has no image.
	ImageURL string
	// ThumbnailURL is the small card image this service hosts, "" when there is none.
	ThumbnailURL string
	SourceName   string
	SourceURL    string
	License      string
}

// ArticlePage is one page of the article list plus what a client needs to know
// whether another page exists.
type ArticlePage struct {
	Items      []Article
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// ErrArticleNotFound — no article has the requested id.
var ErrArticleNotFound = errors.New("article not found")

// ArticleRepository is the port the article use case reads and seeds through.
//
// Ordering is part of the contract: every list read returns articles by
// sort_order ASC, id ASC — deterministic, so paging never repeats or skips a row.
type ArticleRepository interface {
	// ListTop returns the first limit articles in order.
	ListTop(ctx context.Context, limit int) ([]Article, error)

	// List returns limit articles starting at offset, and the total article
	// count. An offset past the end returns an empty slice, not an error.
	List(ctx context.Context, offset, limit int) ([]Article, int, error)

	// FindByID returns the article or ErrArticleNotFound.
	FindByID(ctx context.Context, id string) (*Article, error)

	// Count returns the number of stored articles.
	Count(ctx context.Context) (int, error)

	// InsertMissing inserts the articles whose id is not stored yet, atomically,
	// and returns how many were inserted. It never modifies an existing row, so
	// two instances seeding at once cannot clobber each other or later edits.
	InsertMissing(ctx context.Context, articles []Article) (int, error)

	// FillMissingImages sets image_url and thumbnail_url from the given articles for
	// stored rows where that column is NULL, and returns how many rows changed. It
	// ignores a source value that is empty, never overwrites an existing value, and
	// touches no other column.
	FillMissingImages(ctx context.Context, articles []Article) (int, error)
}
