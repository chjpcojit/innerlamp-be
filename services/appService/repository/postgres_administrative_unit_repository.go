package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type administrativeUnitRepository struct {
	db database.Database
}

// NewAdministrativeUnitRepository returns the Postgres-backed
// domain.AdministrativeUnitRepository.
func NewAdministrativeUnitRepository(db database.Database) domain.AdministrativeUnitRepository {
	return &administrativeUnitRepository{db: db}
}

// Every list is ordered by the precomputed Vietnamese alphabetical rank
// (sort_order) with code as the tie-break, so the order never depends on the
// database collation and two reads never disagree.

// ListCurrentProvinces returns every current-scheme province.
func (r *administrativeUnitRepository) ListCurrentProvinces(ctx context.Context) ([]domain.CurrentProvince, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, name, full_name, type, sort_order FROM addr_new_provinces ORDER BY sort_order, code`)
	if err != nil {
		return nil, fmt.Errorf("query current provinces: %w", err)
	}
	return scanRows(rows, "current province", scanCurrentProvince)
}

// ListCurrentWards returns every ward of a current-scheme province. An empty
// result means the province does not exist: every real province has wards.
func (r *administrativeUnitRepository) ListCurrentWards(ctx context.Context, provinceCode string) ([]domain.CurrentWard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT code, province_code, name, full_name, type, sort_order
		FROM addr_new_wards WHERE province_code = $1 ORDER BY sort_order, code`, provinceCode)
	if err != nil {
		return nil, fmt.Errorf("query current wards: %w", err)
	}
	items, err := scanRows(rows, "current ward", scanCurrentWard)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_new_provinces", provinceCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ListLegacyProvinces returns every legacy-scheme province.
func (r *administrativeUnitRepository) ListLegacyProvinces(ctx context.Context) ([]domain.LegacyProvince, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, name, full_name, type, sort_order FROM addr_old_provinces ORDER BY sort_order, code`)
	if err != nil {
		return nil, fmt.Errorf("query legacy provinces: %w", err)
	}
	return scanRows(rows, "legacy province", scanLegacyProvince)
}

// ListLegacyDistricts returns every district of a legacy-scheme province. An
// empty result means the province does not exist.
func (r *administrativeUnitRepository) ListLegacyDistricts(ctx context.Context, provinceCode string) ([]domain.LegacyDistrict, error) {
	rows, err := r.db.Query(ctx, `
		SELECT code, province_code, name, full_name, type, sort_order
		FROM addr_old_districts WHERE province_code = $1 ORDER BY sort_order, code`, provinceCode)
	if err != nil {
		return nil, fmt.Errorf("query legacy districts: %w", err)
	}
	items, err := scanRows(rows, "legacy district", scanLegacyDistrict)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_old_provinces", provinceCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ListLegacyWards returns every ward of a legacy-scheme district. A district
// that exists but has no wards (an island district) is an empty slice; a
// district that does not exist is ErrAdministrativeUnitNotFound.
func (r *administrativeUnitRepository) ListLegacyWards(ctx context.Context, districtCode string) ([]domain.LegacyWard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT w.code, w.district_code, d.province_code, w.name, w.full_name, w.type, w.sort_order
		FROM addr_old_wards w
		JOIN addr_old_districts d ON d.code = w.district_code
		WHERE w.district_code = $1 ORDER BY w.sort_order, w.code`, districtCode)
	if err != nil {
		return nil, fmt.Errorf("query legacy wards: %w", err)
	}
	items, err := scanRows(rows, "legacy ward", scanLegacyWard)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_old_districts", districtCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ConvertLegacyWard returns the current wards a legacy ward maps to, with each
// one's province and the mapping's type and note.
func (r *administrativeUnitRepository) ConvertLegacyWard(ctx context.Context, wardCode string) ([]domain.CurrentWardCandidate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT nw.code, nw.province_code, nw.name, nw.full_name, nw.type, nw.sort_order,
		       np.code, np.name, np.full_name, np.type, np.sort_order,
		       m.mapping_type, m.note
		FROM addr_ward_mappings m
		JOIN addr_new_wards nw ON nw.code = m.new_ward_code
		JOIN addr_new_provinces np ON np.code = nw.province_code
		WHERE m.old_ward_code = $1
		ORDER BY np.sort_order, np.code, nw.sort_order, nw.code`, wardCode)
	if err != nil {
		return nil, fmt.Errorf("query conversion of legacy ward: %w", err)
	}
	items, err := scanRows(rows, "current ward candidate", func(s rowScanner) (domain.CurrentWardCandidate, error) {
		var c domain.CurrentWardCandidate
		err := s.Scan(
			&c.Ward.Code, &c.Ward.ProvinceCode, &c.Ward.Name, &c.Ward.FullName, &c.Ward.Type, &c.Ward.SortOrder,
			&c.Province.Code, &c.Province.Name, &c.Province.FullName, &c.Province.Type, &c.Province.SortOrder,
			&c.MappingType, &c.Note)
		return c, err
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_old_wards", wardCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ConvertCurrentWard returns the legacy wards that map to a current ward, each
// with its district and province. The result is empty for a current ward that
// is the target of no ward mapping (the special zones that replaced whole
// island districts); it is ErrAdministrativeUnitNotFound only when the ward
// itself does not exist.
func (r *administrativeUnitRepository) ConvertCurrentWard(ctx context.Context, wardCode string) ([]domain.LegacyWardCandidate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ow.code, ow.district_code, d.province_code, ow.name, ow.full_name, ow.type, ow.sort_order,
		       d.code, d.province_code, d.name, d.full_name, d.type, d.sort_order,
		       lp.code, lp.name, lp.full_name, lp.type, lp.sort_order,
		       m.mapping_type, m.note
		FROM addr_ward_mappings m
		JOIN addr_old_wards ow ON ow.code = m.old_ward_code
		JOIN addr_old_districts d ON d.code = ow.district_code
		JOIN addr_old_provinces lp ON lp.code = d.province_code
		WHERE m.new_ward_code = $1
		ORDER BY lp.sort_order, lp.code, d.sort_order, d.code, ow.sort_order, ow.code`, wardCode)
	if err != nil {
		return nil, fmt.Errorf("query conversion of current ward: %w", err)
	}
	items, err := scanRows(rows, "legacy ward candidate", func(s rowScanner) (domain.LegacyWardCandidate, error) {
		var c domain.LegacyWardCandidate
		err := s.Scan(
			&c.Ward.Code, &c.Ward.DistrictCode, &c.Ward.ProvinceCode, &c.Ward.Name, &c.Ward.FullName, &c.Ward.Type, &c.Ward.SortOrder,
			&c.District.Code, &c.District.ProvinceCode, &c.District.Name, &c.District.FullName, &c.District.Type, &c.District.SortOrder,
			&c.Province.Code, &c.Province.Name, &c.Province.FullName, &c.Province.Type, &c.Province.SortOrder,
			&c.MappingType, &c.Note)
		return c, err
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_new_wards", wardCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ConvertLegacyProvince returns the current provinces the wards of a legacy
// province map into. It is derived from the ward mappings: there is no
// province-level table.
func (r *administrativeUnitRepository) ConvertLegacyProvince(ctx context.Context, provinceCode string) ([]domain.CurrentProvince, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT np.code, np.name, np.full_name, np.type, np.sort_order
		FROM addr_old_districts d
		JOIN addr_old_wards ow ON ow.district_code = d.code
		JOIN addr_ward_mappings m ON m.old_ward_code = ow.code
		JOIN addr_new_wards nw ON nw.code = m.new_ward_code
		JOIN addr_new_provinces np ON np.code = nw.province_code
		WHERE d.province_code = $1
		ORDER BY np.sort_order, np.code`, provinceCode)
	if err != nil {
		return nil, fmt.Errorf("query conversion of legacy province: %w", err)
	}
	items, err := scanRows(rows, "current province", scanCurrentProvince)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_old_provinces", provinceCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ConvertCurrentProvince returns the legacy provinces whose wards map into a
// current province, derived from the ward mappings.
func (r *administrativeUnitRepository) ConvertCurrentProvince(ctx context.Context, provinceCode string) ([]domain.LegacyProvince, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT lp.code, lp.name, lp.full_name, lp.type, lp.sort_order
		FROM addr_new_wards nw
		JOIN addr_ward_mappings m ON m.new_ward_code = nw.code
		JOIN addr_old_wards ow ON ow.code = m.old_ward_code
		JOIN addr_old_districts d ON d.code = ow.district_code
		JOIN addr_old_provinces lp ON lp.code = d.province_code
		WHERE nw.province_code = $1
		ORDER BY lp.sort_order, lp.code`, provinceCode)
	if err != nil {
		return nil, fmt.Errorf("query conversion of current province: %w", err)
	}
	items, err := scanRows(rows, "legacy province", scanLegacyProvince)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		if err := r.requireUnit(ctx, "addr_new_provinces", provinceCode); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// ReplaceDataset upserts the whole dataset in one transaction: all rows or
// none, so a bad row cannot leave the units half-loaded. Parents are written
// before children and the ward mappings last, so foreign keys hold at every
// step. It never deletes — a unit absent from ds stays as it was — and loading
// the same dataset again changes nothing.
//
// It does not verify ds; the seeder calls domain.Dataset.Verify first. The
// foreign keys still reject an orphan, but a duplicate code would silently
// become last-write-wins.
func (r *administrativeUnitRepository) ReplaceDataset(ctx context.Context, ds domain.Dataset) error {
	if ds.Version == "" {
		return errors.New("replace administrative units: dataset version is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace administrative units: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	steps := []func() error{
		func() error {
			return upsertBatched(ctx, tx, upsertSpec{"addr_new_provinces",
				[]string{"code", "name", "full_name", "type", "sort_order"}, []string{"code"}},
				ds.CurrentProvinces, func(p domain.CurrentProvince) []any {
					return []any{p.Code, p.Name, p.FullName, p.Type, p.SortOrder}
				})
		},
		func() error {
			return upsertBatched(ctx, tx, upsertSpec{"addr_new_wards",
				[]string{"code", "province_code", "name", "full_name", "type", "sort_order"}, []string{"code"}},
				ds.CurrentWards, func(w domain.CurrentWard) []any {
					return []any{w.Code, w.ProvinceCode, w.Name, w.FullName, w.Type, w.SortOrder}
				})
		},
		func() error {
			return upsertBatched(ctx, tx, upsertSpec{"addr_old_provinces",
				[]string{"code", "name", "full_name", "type", "sort_order"}, []string{"code"}},
				ds.LegacyProvinces, func(p domain.LegacyProvince) []any {
					return []any{p.Code, p.Name, p.FullName, p.Type, p.SortOrder}
				})
		},
		func() error {
			return upsertBatched(ctx, tx, upsertSpec{"addr_old_districts",
				[]string{"code", "province_code", "name", "full_name", "type", "sort_order"}, []string{"code"}},
				ds.LegacyDistricts, func(d domain.LegacyDistrict) []any {
					return []any{d.Code, d.ProvinceCode, d.Name, d.FullName, d.Type, d.SortOrder}
				})
		},
		func() error {
			return upsertBatched(ctx, tx, upsertSpec{"addr_old_wards",
				[]string{"code", "district_code", "name", "full_name", "type", "sort_order"}, []string{"code"}},
				ds.LegacyWards, func(w domain.LegacyWard) []any {
					return []any{w.Code, w.DistrictCode, w.Name, w.FullName, w.Type, w.SortOrder}
				})
		},
		func() error {
			return upsertBatched(ctx, tx, upsertSpec{"addr_ward_mappings",
				[]string{"old_ward_code", "new_ward_code", "mapping_type", "note"}, []string{"old_ward_code", "new_ward_code"}},
				ds.WardMappings, func(m domain.WardMapping) []any {
					return []any{m.LegacyWardCode, m.CurrentWardCode, m.MappingType, m.Note}
				})
		},
	}
	for _, run := range steps {
		if err := run(); err != nil {
			return err
		}
	}

	// District mappings are part of the dataset but not stored yet (district-level
	// conversion is deferred), so nothing is written for ds.DistrictMappings.

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO addr_dataset_versions (version, source_note) VALUES ($1, $2)
		ON CONFLICT (version) DO NOTHING`, ds.Version, ds.SourceNote); err != nil {
		return fmt.Errorf("record dataset version %q: %w", ds.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace administrative units: %w", err)
	}
	return nil
}

// requireUnit returns ErrAdministrativeUnitNotFound when no row of table has the
// given code. table is always one of this file's own string literals, never
// caller input.
func (r *administrativeUnitRepository) requireUnit(ctx context.Context, table, code string) error {
	var one int
	err := r.db.QueryRow(ctx, `SELECT 1 FROM `+table+` WHERE code = $1`, code).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.ErrAdministrativeUnitNotFound
	case err != nil:
		return fmt.Errorf("check %s %q: %w", table, code, err)
	}
	return nil
}

// upsertBatchSize is how many rows go into one multi-row INSERT. The widest
// table has six columns, so a batch is at most 6,000 of Postgres's 65,535 bind
// parameters. One statement per row costs a network round trip each (about 22 s
// for the real 24,000-row dataset against a local container); batching it makes
// the same load a fraction of a second.
const upsertBatchSize = 1000

// upsertSpec names a table, its columns in bind order, and its conflict key.
// All three are this file's own string literals, never caller input.
type upsertSpec struct {
	table string
	cols  []string
	key   []string
}

// upsertBatched writes rows with INSERT ... ON CONFLICT (key) DO UPDATE, a batch
// of upsertBatchSize rows per statement, inside tx. Columns outside the key are
// overwritten from the incoming row. A key repeated within one batch is an
// error from Postgres, not a silent last-write-wins — callers verify first.
func upsertBatched[T any](ctx context.Context, tx *sql.Tx, spec upsertSpec, rows []T, args func(T) []any) error {
	isKey := make(map[string]bool, len(spec.key))
	for _, k := range spec.key {
		isKey[k] = true
	}
	var update []string
	for _, c := range spec.cols {
		if !isKey[c] {
			update = append(update, c+" = EXCLUDED."+c)
		}
	}
	conflict := " ON CONFLICT (" + strings.Join(spec.key, ", ") + ")"
	if len(update) > 0 {
		conflict += " DO UPDATE SET " + strings.Join(update, ", ")
	} else {
		conflict += " DO NOTHING"
	}

	for start := 0; start < len(rows); start += upsertBatchSize {
		end := min(start+upsertBatchSize, len(rows))
		var sb strings.Builder
		sb.WriteString("INSERT INTO " + spec.table + " (" + strings.Join(spec.cols, ", ") + ") VALUES ")
		params := make([]any, 0, (end-start)*len(spec.cols))
		for i, row := range rows[start:end] {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteByte('(')
			for j := range spec.cols {
				if j > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString("$" + strconv.Itoa(len(params)+j+1))
			}
			sb.WriteByte(')')
			params = append(params, args(row)...)
		}
		sb.WriteString(conflict)
		if _, err := tx.ExecContext(ctx, sb.String(), params...); err != nil {
			return fmt.Errorf("upsert %s rows %d-%d: %w", spec.table, start, end-1, err)
		}
	}
	return nil
}

// scanRows drains rows into a non-nil slice with the given per-row scanner.
func scanRows[T any](rows *sql.Rows, what string, scan func(rowScanner) (T, error)) ([]T, error) {
	defer func() { _ = rows.Close() }()
	items := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", what, err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %ss: %w", what, err)
	}
	return items, nil
}

func scanCurrentProvince(s rowScanner) (domain.CurrentProvince, error) {
	var p domain.CurrentProvince
	err := s.Scan(&p.Code, &p.Name, &p.FullName, &p.Type, &p.SortOrder)
	return p, err
}

func scanCurrentWard(s rowScanner) (domain.CurrentWard, error) {
	var w domain.CurrentWard
	err := s.Scan(&w.Code, &w.ProvinceCode, &w.Name, &w.FullName, &w.Type, &w.SortOrder)
	return w, err
}

func scanLegacyProvince(s rowScanner) (domain.LegacyProvince, error) {
	var p domain.LegacyProvince
	err := s.Scan(&p.Code, &p.Name, &p.FullName, &p.Type, &p.SortOrder)
	return p, err
}

func scanLegacyDistrict(s rowScanner) (domain.LegacyDistrict, error) {
	var d domain.LegacyDistrict
	err := s.Scan(&d.Code, &d.ProvinceCode, &d.Name, &d.FullName, &d.Type, &d.SortOrder)
	return d, err
}

func scanLegacyWard(s rowScanner) (domain.LegacyWard, error) {
	var w domain.LegacyWard
	err := s.Scan(&w.Code, &w.DistrictCode, &w.ProvinceCode, &w.Name, &w.FullName, &w.Type, &w.SortOrder)
	return w, err
}
