package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// articleColumns / articleOrder are shared by every read so top, list and
// detail scan identically and paging order can never diverge from the top-N order.
const (
	articleColumns = `id, sort_order, title, category, author, author_initials, published_date,
		read_minutes, summary, body_paragraphs, quote, quote_source, tags, language, image_url,
		thumbnail_url, source_name, source_url, license`

	// id is the final tiebreaker so equal sort_order rows cannot swap between pages.
	articleOrder = `ORDER BY sort_order ASC, id ASC`
)

type articleRepository struct {
	db database.Database
}

// NewArticleRepository returns the Postgres-backed domain.ArticleRepository.
func NewArticleRepository(db database.Database) domain.ArticleRepository {
	return &articleRepository{db: db}
}

// ListTop returns the first limit articles in order.
func (r *articleRepository) ListTop(ctx context.Context, limit int) ([]domain.Article, error) {
	rows, err := r.db.Query(ctx, `SELECT `+articleColumns+` FROM articles `+articleOrder+` LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query top articles: %w", err)
	}
	return scanArticles(rows)
}

// List returns one window of the ordered list plus the total row count.
func (r *articleRepository) List(ctx context.Context, offset, limit int) ([]domain.Article, int, error) {
	total, err := r.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+articleColumns+` FROM articles `+articleOrder+` OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query articles: %w", err)
	}
	items, err := scanArticles(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID returns the article or domain.ErrArticleNotFound.
func (r *articleRepository) FindByID(ctx context.Context, id string) (*domain.Article, error) {
	row := r.db.QueryRow(ctx, `SELECT `+articleColumns+` FROM articles WHERE id = $1`, id)
	a, err := scanArticle(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrArticleNotFound
		}
		return nil, fmt.Errorf("find article: %w", err)
	}
	return a, nil
}

// Count returns the number of stored articles.
func (r *articleRepository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM articles`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count articles: %w", err)
	}
	return n, nil
}

// InsertMissing inserts articles whose id is absent, in one transaction: all
// rows or none, so a bad row in the seed cannot leave the catalogue half-seeded.
// ON CONFLICT DO NOTHING keeps concurrent seeders (two instances starting
// together) and existing rows safe.
func (r *articleRepository) InsertMissing(ctx context.Context, articles []domain.Article) (int, error) {
	const query = `
		INSERT INTO articles (id, sort_order, title, category, author, author_initials, published_date,
			read_minutes, summary, body_paragraphs, quote, quote_source, tags, language, image_url,
			thumbnail_url, source_name, source_url, license)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		ON CONFLICT (id) DO NOTHING
	`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin insert articles: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	inserted := 0
	for _, a := range articles {
		body, tags := a.BodyParagraphs, a.Tags
		if body == nil {
			body = []string{}
		}
		if tags == nil {
			tags = []string{}
		}
		var published, image, thumb any // NULL, not '' / zero date, when the source has none
		if a.PublishedDate != nil {
			published = *a.PublishedDate
		}
		if a.ImageURL != "" {
			image = a.ImageURL
		}
		if a.ThumbnailURL != "" {
			thumb = a.ThumbnailURL
		}
		res, err := tx.ExecContext(ctx, query,
			a.ID, a.SortOrder, a.Title, a.Category, a.Author, a.AuthorInitials, published,
			a.ReadMinutes, a.Summary, pq.Array(body), a.Quote, a.QuoteSource, pq.Array(tags), a.Language, image,
			thumb, a.SourceName, a.SourceURL, a.License,
		)
		if err != nil {
			return 0, fmt.Errorf("insert article %q: %w", a.ID, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("rows affected for article %q: %w", a.ID, err)
		}
		inserted += int(n)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit insert articles: %w", err)
	}
	return inserted, nil
}

// FillMissingImages back-fills image_url and thumbnail_url on existing rows that
// have none, in one transaction. COALESCE keeps a value set earlier (by hand or by
// another source), each column is filled independently, and re-running is a no-op.
// A source value that is empty is passed as NULL, so it can never blank a column.
func (r *articleRepository) FillMissingImages(ctx context.Context, articles []domain.Article) (int, error) {
	const query = `UPDATE articles
		SET image_url = COALESCE(image_url, $2), thumbnail_url = COALESCE(thumbnail_url, $3)
		WHERE id = $1
		  AND ((image_url IS NULL AND $2::text IS NOT NULL) OR (thumbnail_url IS NULL AND $3::text IS NOT NULL))`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin fill article images: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	updated := 0
	for _, a := range articles {
		var image, thumb any // NULL when the source has none
		if a.ImageURL != "" {
			image = a.ImageURL
		}
		if a.ThumbnailURL != "" {
			thumb = a.ThumbnailURL
		}
		if image == nil && thumb == nil {
			continue
		}
		res, err := tx.ExecContext(ctx, query, a.ID, image, thumb)
		if err != nil {
			return 0, fmt.Errorf("fill image for article %q: %w", a.ID, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("rows affected for article %q: %w", a.ID, err)
		}
		updated += int(n)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit fill article images: %w", err)
	}
	return updated, nil
}

func scanArticle(row rowScanner) (*domain.Article, error) {
	var a domain.Article
	var published sql.NullTime
	var image, thumb sql.NullString
	if err := row.Scan(
		&a.ID, &a.SortOrder, &a.Title, &a.Category, &a.Author, &a.AuthorInitials, &published,
		&a.ReadMinutes, &a.Summary, pq.Array(&a.BodyParagraphs), &a.Quote, &a.QuoteSource,
		pq.Array(&a.Tags), &a.Language, &image, &thumb, &a.SourceName, &a.SourceURL, &a.License,
	); err != nil {
		return nil, err
	}
	if published.Valid {
		t := published.Time
		a.PublishedDate = &t
	}
	a.ImageURL = image.String
	a.ThumbnailURL = thumb.String
	return &a, nil
}

func scanArticles(rows *sql.Rows) ([]domain.Article, error) {
	defer func() { _ = rows.Close() }()
	items := []domain.Article{}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		items = append(items, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}
	return items, nil
}
