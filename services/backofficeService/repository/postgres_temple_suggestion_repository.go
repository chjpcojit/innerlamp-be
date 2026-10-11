package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type templeSuggestionRepository struct {
	db database.Database
}

// NewTempleSuggestionRepository returns the Postgres-backed
// domain.TempleSuggestionRepository.
func NewTempleSuggestionRepository(db database.Database) domain.TempleSuggestionRepository {
	return &templeSuggestionRepository{db: db}
}

const suggestionColumns = `id, name, province_code, province, ward_code, ward, address, status, created_at,
	temple_id, reviewed_by, reviewed_at, reject_reason`

type rowScanner interface{ Scan(dest ...any) error }

func scanSuggestion(row rowScanner) (*domain.TempleSuggestion, error) {
	var s domain.TempleSuggestion
	var provinceCode, wardCode, ward, templeID, rejectReason sql.NullString
	var reviewedBy sql.NullInt64
	var reviewedAt sql.NullTime
	if err := row.Scan(&s.ID, &s.Name, &provinceCode, &s.Province, &wardCode, &ward, &s.Address, &s.Status, &s.CreatedAt,
		&templeID, &reviewedBy, &reviewedAt, &rejectReason); err != nil {
		return nil, err
	}
	s.ProvinceCode = nullStr(provinceCode)
	s.WardCode = nullStr(wardCode)
	s.Ward = nullStr(ward)
	s.TempleID = nullStr(templeID)
	s.RejectReason = nullStr(rejectReason)
	if reviewedBy.Valid {
		s.ReviewedBy = &reviewedBy.Int64
	}
	if reviewedAt.Valid {
		s.ReviewedAt = &reviewedAt.Time
	}
	return &s, nil
}

func nullStr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}

// List returns one page of the queue, oldest first. The count and the page are
// two statements: a row added between them can make total one ahead of the
// items, which a queue read tolerates.
func (r *templeSuggestionRepository) List(ctx context.Context, status string, offset, limit int) ([]domain.TempleSuggestion, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM temple_suggestions WHERE status = $1`, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count temple suggestions: %w", err)
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+suggestionColumns+` FROM temple_suggestions WHERE status = $1 ORDER BY created_at, id LIMIT $2 OFFSET $3`,
		status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list temple suggestions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := []domain.TempleSuggestion{}
	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan temple suggestion: %w", err)
		}
		items = append(items, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate temple suggestions: %w", err)
	}
	return items, total, nil
}

// FindByID returns the suggestion or domain.ErrSuggestionNotFound.
func (r *templeSuggestionRepository) FindByID(ctx context.Context, id int64) (*domain.TempleSuggestion, error) {
	s, err := scanSuggestion(r.db.QueryRow(ctx, `SELECT `+suggestionColumns+` FROM temple_suggestions WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSuggestionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find temple suggestion: %w", err)
	}
	return s, nil
}

// CreateTemple runs the whole conversion in one transaction (design Step C-4..C-7):
// lock the suggestion row, generate the id, insert the temple, mark the
// suggestion created. Anything failing rolls back all of it.
func (r *templeSuggestionRepository) CreateTemple(ctx context.Context, suggestionID, adminID int64, role string, t domain.NewTemple, priority int) (*domain.CreatedTemple, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create temple: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// C-4: lock and check. Concurrent requests for the same suggestion queue
	// here; the second sees 'created' once the first commits (AC22).
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM temple_suggestions WHERE id = $1 FOR UPDATE`, suggestionID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSuggestionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock temple suggestion: %w", err)
	}
	switch status {
	case domain.TempleSuggestionCreated:
		return nil, domain.ErrSuggestionAlreadyCreated
	case domain.TempleSuggestionRejected:
		if role != domain.RoleAdmin {
			return nil, domain.ErrReopenForbidden
		}
	}

	// C-5: next id of the region. The advisory lock keeps two creations in the
	// same region (different suggestions, so different row locks) from reading
	// the same maximum.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "temples:"+t.Region); err != nil {
		return nil, fmt.Errorf("lock temple id sequence: %w", err)
	}
	var next int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(substring(id from '^temple_'||$1||'_([0-9]+)$')::int), 0) + 1
		FROM temples WHERE id ~ ('^temple_'||$1||'_[0-9]+$')`, t.Region).Scan(&next)
	if err != nil {
		return nil, fmt.Errorf("next temple id: %w", err)
	}
	id := fmt.Sprintf("temple_%s_%03d", t.Region, next)

	// C-6: the temple. Columns are listed explicitly; province, ward and the
	// codes arrive together (chk_temples_address_codes).
	aliases, notes := t.Aliases, t.VisitingNotes
	if aliases == nil {
		aliases = []string{}
	}
	if notes == nil {
		notes = []string{}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO temples (id, name, type, region, image_url, thumbnail_url, aliases, sect, founder,
			current_abbot, established_year, heritage_rank, heritage_priority, latitude, longitude,
			address, province, operating_hours, parking_car, parking_motorcycle, vegetarian_meals, summer_retreat,
			introduction, architecture_heritage, visiting_notes, province_code, ward_code, ward)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)`,
		id, t.Name, t.Type, t.Region, t.ImageURL, t.ThumbnailURL, pq.Array(aliases), t.Sect, t.Founder,
		t.CurrentAbbot, t.EstablishedYear, t.HeritageRank, priority, t.Latitude, t.Longitude,
		t.Address, t.Province, t.OperatingHours, t.Amenities.ParkingCar, t.Amenities.ParkingMotorcycle,
		t.Amenities.VegetarianMeals, t.Amenities.SummerRetreat,
		t.Introduction, t.ArchitectureHeritage, pq.Array(notes), t.ProvinceCode, t.WardCode, t.Ward)
	if err != nil {
		return nil, fmt.Errorf("insert temple: %w", err)
	}

	// C-7: mark the suggestion. reject_reason is cleared (the CHECK for
	// 'created' requires NULL), so reopening a rejected one loses the reason.
	reviewedAt := time.Now()
	row := tx.QueryRowContext(ctx, `
		UPDATE temple_suggestions
		SET status = 'created', temple_id = $2, reviewed_by = $3, reviewed_at = $4, reject_reason = NULL, updated_at = $4
		WHERE id = $1
		RETURNING `+suggestionColumns, suggestionID, id, adminID, reviewedAt)
	s, err := scanSuggestion(row)
	if err != nil {
		return nil, fmt.Errorf("mark temple suggestion created: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create temple: %w", err)
	}

	return &domain.CreatedTemple{Suggestion: *s, Temple: domain.Temple{
		ID: id, Name: t.Name, Type: t.Type, Region: t.Region, ImageURL: t.ImageURL, ThumbnailURL: t.ThumbnailURL,
		Aliases: aliases, Sect: t.Sect, Founder: t.Founder, CurrentAbbot: t.CurrentAbbot,
		EstablishedYear: t.EstablishedYear, HeritageRank: t.HeritageRank, HeritagePriority: priority,
		Latitude: t.Latitude, Longitude: t.Longitude, Address: t.Address,
		ProvinceCode: t.ProvinceCode, Province: t.Province, WardCode: t.WardCode, Ward: t.Ward,
		OperatingHours: t.OperatingHours, Amenities: t.Amenities, Introduction: t.Introduction,
		ArchitectureHeritage: t.ArchitectureHeritage, VisitingNotes: notes,
	}}, nil
}

// ResolveCurrentWard checks a (province, ward) pair of the current scheme in one
// query. The LEFT JOIN keeps the province row when the ward is missing or sits
// in another province, which is how "no such province" is told apart from "no
// such ward in this province". Same statement as appService's repository (a
// deliberate copy, see design "Known gaps").
func (r *templeSuggestionRepository) ResolveCurrentWard(ctx context.Context, provinceCode, wardCode string) (*domain.ResolvedWard, error) {
	var provinceName string
	var wardName sql.NullString
	err := r.db.QueryRow(ctx, `
		SELECT p.name, w.full_name
		FROM addr_new_provinces p
		LEFT JOIN addr_new_wards w ON w.code = $2 AND w.province_code = p.code
		WHERE p.code = $1`, provinceCode, wardCode).Scan(&provinceName, &wardName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAdministrativeUnitNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resolve current ward: %w", err)
	}
	if !wardName.Valid {
		return nil, domain.ErrWardNotInProvince
	}
	return &domain.ResolvedWard{ProvinceName: provinceName, WardFullName: wardName.String}, nil
}
