package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// galleryColumns / galleryOrder are shared by every read so top, list and detail
// scan identically and paging order can never diverge from the top-N order.
const (
	galleryColumns = `id, type, sort_order, title, location, thumbnail_url, image_url, author, license, source_url`

	// id is the final tiebreaker so equal sort_order rows cannot swap between pages.
	galleryOrder = `ORDER BY sort_order ASC, id ASC`
)

type galleryRepository struct {
	db database.Database
}

// NewGalleryRepository returns the Postgres-backed domain.GalleryRepository.
func NewGalleryRepository(db database.Database) domain.GalleryRepository {
	return &galleryRepository{db: db}
}

// ListTop returns the first limit images of any type in order.
func (r *galleryRepository) ListTop(ctx context.Context, limit int) ([]domain.GalleryImage, error) {
	rows, err := r.db.Query(ctx, `SELECT `+galleryColumns+` FROM gallery_images `+galleryOrder+` LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query top gallery images: %w", err)
	}
	return scanGalleryImages(rows)
}

// List returns one window of the ordered list (optionally one type) plus the
// total number of matching rows. The type filter is a parameter, never spliced
// into the SQL: "$1 = ” OR type = $1" keeps one query shape for both cases.
func (r *galleryRepository) List(ctx context.Context, imageType string, offset, limit int) ([]domain.GalleryImage, int, error) {
	const where = ` WHERE ($1 = '' OR type = $1) `
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM gallery_images`+where, imageType).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count gallery images: %w", err)
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+galleryColumns+` FROM gallery_images`+where+galleryOrder+` OFFSET $2 LIMIT $3`,
		imageType, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query gallery images: %w", err)
	}
	items, err := scanGalleryImages(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID returns the image or domain.ErrGalleryImageNotFound.
func (r *galleryRepository) FindByID(ctx context.Context, id string) (*domain.GalleryImage, error) {
	row := r.db.QueryRow(ctx, `SELECT `+galleryColumns+` FROM gallery_images WHERE id = $1`, id)
	g, err := scanGalleryImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrGalleryImageNotFound
		}
		return nil, fmt.Errorf("find gallery image: %w", err)
	}
	return g, nil
}

// Upsert writes every image in one transaction: all rows or none, so a bad row
// in the source file cannot leave the gallery half-seeded.
func (r *galleryRepository) Upsert(ctx context.Context, images []domain.GalleryImage) error {
	const query = `
		INSERT INTO gallery_images (id, type, sort_order, title, location, thumbnail_url, image_url,
			author, license, source_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type, sort_order = EXCLUDED.sort_order, title = EXCLUDED.title,
			location = EXCLUDED.location, thumbnail_url = EXCLUDED.thumbnail_url,
			image_url = EXCLUDED.image_url, author = EXCLUDED.author, license = EXCLUDED.license,
			source_url = EXCLUDED.source_url, updated_at = now()
	`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert gallery images: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, g := range images {
		if _, err := tx.ExecContext(ctx, query,
			g.ID, g.Type, g.SortOrder, g.Title, g.Location, g.ThumbnailURL, g.ImageURL,
			g.Author, g.License, g.SourceURL,
		); err != nil {
			return fmt.Errorf("upsert gallery image %q: %w", g.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert gallery images: %w", err)
	}
	return nil
}

func scanGalleryImage(row rowScanner) (*domain.GalleryImage, error) {
	var g domain.GalleryImage
	if err := row.Scan(&g.ID, &g.Type, &g.SortOrder, &g.Title, &g.Location, &g.ThumbnailURL,
		&g.ImageURL, &g.Author, &g.License, &g.SourceURL); err != nil {
		return nil, err
	}
	return &g, nil
}

func scanGalleryImages(rows *sql.Rows) ([]domain.GalleryImage, error) {
	defer func() { _ = rows.Close() }()
	items := []domain.GalleryImage{}
	for rows.Next() {
		g, err := scanGalleryImage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan gallery image: %w", err)
		}
		items = append(items, *g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gallery images: %w", err)
	}
	return items, nil
}
