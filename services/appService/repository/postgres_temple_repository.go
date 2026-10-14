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

// templeColumns / templeOrder are shared by every read so top, list and detail
// scan identically and paging order can never diverge from the top-N order.
const (
	templeColumns = `id, name, type, region, image_url, thumbnail_url, aliases, sect, founder,
		current_abbot, established_year, heritage_rank, heritage_priority, latitude, longitude,
		address, province, COALESCE(province_code, ''), COALESCE(ward_code, ''), COALESCE(ward, ''),
		operating_hours, parking_car, parking_motorcycle, vegetarian_meals, summer_retreat,
		introduction, architecture_heritage, visiting_notes`

	// id is the final tiebreaker: without it two temples of the same rank and
	// year could swap between pages.
	templeOrder = `ORDER BY heritage_priority ASC, established_year ASC, id ASC`
)

type templeRepository struct {
	db database.Database
}

// NewTempleRepository returns the Postgres-backed domain.TempleRepository.
func NewTempleRepository(db database.Database) domain.TempleRepository {
	return &templeRepository{db: db}
}

// ListTop returns the first limit temples in ranking order.
func (r *templeRepository) ListTop(ctx context.Context, limit int) ([]domain.Temple, error) {
	rows, err := r.db.Query(ctx, `SELECT `+templeColumns+` FROM temples `+templeOrder+` LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query top temples: %w", err)
	}
	return scanTemples(rows)
}

// List returns one window of the ranked catalogue plus the total row count.
func (r *templeRepository) List(ctx context.Context, offset, limit int) ([]domain.Temple, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM temples`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count temples: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT `+templeColumns+` FROM temples `+templeOrder+` OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query temples: %w", err)
	}
	items, err := scanTemples(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListOptions returns one window of the ranked catalogue as id/name/province
// plus the total number of matching rows. It reuses templeOrder so it can never
// diverge from top and the paged list, and reads only id, name, province and
// province_code (no editorial text or array; NULL code, i.e. a temple not
// re-seeded since migration 0032, reads as ""). A non-empty query keeps temples whose name contains
// it, ignoring case and Vietnamese diacritics (unaccent, enabled by migration
// 0019, as for GET /v1/sutras?q=); LIKE wildcards in it are escaped so "100%" or
// "a_b" match literally.
func (r *templeRepository) ListOptions(ctx context.Context, query string, offset, limit int) ([]domain.TempleOption, int, error) {
	where := ""
	var args []any
	if query != "" {
		where = `WHERE unaccent(lower(name)) LIKE unaccent(lower($1)) ESCAPE '\' `
		args = append(args, likeContains(query))
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM temples `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count temple options: %w", err)
	}

	args = append(args, offset, limit)
	rows, err := r.db.Query(ctx,
		`SELECT id, name, province, COALESCE(province_code, '') FROM temples `+where+templeOrder+
			fmt.Sprintf(` OFFSET $%d LIMIT $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query temple options: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := []domain.TempleOption{}
	for rows.Next() {
		var o domain.TempleOption
		if err := rows.Scan(&o.ID, &o.Name, &o.Province, &o.ProvinceCode); err != nil {
			return nil, 0, fmt.Errorf("scan temple option: %w", err)
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate temple options: %w", err)
	}
	return items, total, nil
}

// FindByID returns the temple or domain.ErrTempleNotFound.
func (r *templeRepository) FindByID(ctx context.Context, id string) (*domain.Temple, error) {
	row := r.db.QueryRow(ctx, `SELECT `+templeColumns+` FROM temples WHERE id = $1`, id)
	t, err := scanTemple(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTempleNotFound
		}
		return nil, fmt.Errorf("find temple: %w", err)
	}
	return t, nil
}

// Upsert writes every temple in one transaction: all rows or none, so a bad row
// in the source file cannot leave the catalogue half-seeded.
func (r *templeRepository) Upsert(ctx context.Context, temples []domain.Temple) error {
	const query = `
		INSERT INTO temples (id, name, type, region, image_url, thumbnail_url, aliases, sect, founder,
			current_abbot, established_year, heritage_rank, heritage_priority, latitude, longitude,
			address, province, operating_hours, parking_car, parking_motorcycle, vegetarian_meals, summer_retreat,
			introduction, architecture_heritage, visiting_notes, province_code, ward_code, ward)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,
			NULLIF($26,''), NULLIF($27,''), NULLIF($28,''))
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, type = EXCLUDED.type, region = EXCLUDED.region,
			image_url = EXCLUDED.image_url, thumbnail_url = EXCLUDED.thumbnail_url,
			aliases = EXCLUDED.aliases, sect = EXCLUDED.sect, founder = EXCLUDED.founder,
			current_abbot = EXCLUDED.current_abbot, established_year = EXCLUDED.established_year,
			heritage_rank = EXCLUDED.heritage_rank, heritage_priority = EXCLUDED.heritage_priority,
			latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude, address = EXCLUDED.address,
			province = EXCLUDED.province, operating_hours = EXCLUDED.operating_hours, parking_car = EXCLUDED.parking_car,
			parking_motorcycle = EXCLUDED.parking_motorcycle, vegetarian_meals = EXCLUDED.vegetarian_meals,
			summer_retreat = EXCLUDED.summer_retreat, introduction = EXCLUDED.introduction,
			architecture_heritage = EXCLUDED.architecture_heritage, visiting_notes = EXCLUDED.visiting_notes,
			province_code = EXCLUDED.province_code, ward_code = EXCLUDED.ward_code, ward = EXCLUDED.ward,
			updated_at = now()
	`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert temples: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, t := range temples {
		aliases := t.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		notes := t.VisitingNotes
		if notes == nil {
			notes = []string{}
		}
		if _, err := tx.ExecContext(ctx, query,
			t.ID, t.Name, t.Type, t.Region, t.ImageURL, t.Thumbnail, pq.Array(aliases), t.Sect, t.Founder,
			t.Abbot, t.Established, t.Heritage, t.HeritagePriority, t.Latitude, t.Longitude,
			t.Address, t.Province, t.OperatingHours, t.Amenities.ParkingCar, t.Amenities.ParkingMotorcycle,
			t.Amenities.VegetarianMeals, t.Amenities.SummerRetreat,
			t.Introduction, t.ArchitectureHeritage, pq.Array(notes),
			t.ProvinceCode, t.WardCode, t.Ward,
		); err != nil {
			return fmt.Errorf("upsert temple %q: %w", t.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert temples: %w", err)
	}
	return nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanTemple(row rowScanner) (*domain.Temple, error) {
	var t domain.Temple
	if err := row.Scan(
		&t.ID, &t.Name, &t.Type, &t.Region, &t.ImageURL, &t.Thumbnail, pq.Array(&t.Aliases), &t.Sect,
		&t.Founder, &t.Abbot, &t.Established, &t.Heritage, &t.HeritagePriority, &t.Latitude, &t.Longitude,
		&t.Address, &t.Province, &t.ProvinceCode, &t.WardCode, &t.Ward,
		&t.OperatingHours, &t.Amenities.ParkingCar, &t.Amenities.ParkingMotorcycle,
		&t.Amenities.VegetarianMeals, &t.Amenities.SummerRetreat,
		&t.Introduction, &t.ArchitectureHeritage, pq.Array(&t.VisitingNotes),
	); err != nil {
		return nil, err
	}
	return &t, nil
}

func scanTemples(rows *sql.Rows) ([]domain.Temple, error) {
	defer func() { _ = rows.Close() }()
	items := []domain.Temple{}
	for rows.Next() {
		t, err := scanTemple(rows)
		if err != nil {
			return nil, fmt.Errorf("scan temple: %w", err)
		}
		items = append(items, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate temples: %w", err)
	}
	return items, nil
}
