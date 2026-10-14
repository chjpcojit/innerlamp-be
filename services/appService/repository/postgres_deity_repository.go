package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// deityColumns / deityOrder are shared by every read so top, list and detail
// scan identically and paging order can never diverge from the top-N order.
const (
	deityColumns = `id, type, sort_order, name_vi, name_sanskrit, name_chinese, title_honorific,
		thumbnail_url, image_url, canonical_source, core_virtue, pure_land, lunar_memorial_days,
		iconography_posture, iconography_mount_or_throne, sacred_artifacts, symbolic_meaning,
		biography_summary, great_vows, short_mantra_sanskrit, short_mantra_sino_vietnamese,
		long_dharani_name, dharani_phonetic_snippet`

	// id is the final tiebreaker so equal sort_order rows cannot swap between pages.
	deityOrder = `ORDER BY sort_order ASC, id ASC`
)

// memorialDayJSON is the stored shape of one lunar_memorial_days element.
type memorialDayJSON struct {
	LunarDay   int    `json:"lunar_day"`
	LunarMonth int    `json:"lunar_month"`
	EventName  string `json:"event_name"`
}

type deityRepository struct {
	db database.Database
}

// NewDeityRepository returns the Postgres-backed domain.DeityRepository.
func NewDeityRepository(db database.Database) domain.DeityRepository {
	return &deityRepository{db: db}
}

// ListTop returns the first limit deities of any type in order.
func (r *deityRepository) ListTop(ctx context.Context, limit int) ([]domain.Deity, error) {
	rows, err := r.db.Query(ctx, `SELECT `+deityColumns+` FROM deities `+deityOrder+` LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query top deities: %w", err)
	}
	return scanDeities(rows)
}

// List returns one window of the ordered list of deities plus the total row
// count. deityType "" lists every type; otherwise both the count and the window
// are restricted to that type.
func (r *deityRepository) List(ctx context.Context, deityType string, offset, limit int) ([]domain.Deity, int, error) {
	where, args := "", []any{}
	if deityType != "" {
		where, args = `WHERE type = $1 `, []any{deityType}
	}
	n := len(args)

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM deities `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deities: %w", err)
	}
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(`SELECT `+deityColumns+` FROM deities `+where+deityOrder+` OFFSET $%d LIMIT $%d`, n+1, n+2),
		append(args, offset, limit)...)
	if err != nil {
		return nil, 0, fmt.Errorf("query deities: %w", err)
	}
	items, err := scanDeities(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID returns the deity or domain.ErrDeityNotFound.
func (r *deityRepository) FindByID(ctx context.Context, id string) (*domain.Deity, error) {
	row := r.db.QueryRow(ctx, `SELECT `+deityColumns+` FROM deities WHERE id = $1`, id)
	d, err := scanDeity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeityNotFound
		}
		return nil, fmt.Errorf("find deity: %w", err)
	}
	return d, nil
}

// Upsert writes every deity in one transaction: all rows or none, so a bad row
// in the source file cannot leave the catalogue half-seeded.
func (r *deityRepository) Upsert(ctx context.Context, deities []domain.Deity) error {
	const query = `
		INSERT INTO deities (id, type, sort_order, name_vi, name_sanskrit, name_chinese, title_honorific,
			thumbnail_url, image_url, canonical_source, core_virtue, pure_land, lunar_memorial_days,
			iconography_posture, iconography_mount_or_throne, sacred_artifacts, symbolic_meaning,
			biography_summary, great_vows, short_mantra_sanskrit, short_mantra_sino_vietnamese,
			long_dharani_name, dharani_phonetic_snippet)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type, sort_order = EXCLUDED.sort_order, name_vi = EXCLUDED.name_vi,
			name_sanskrit = EXCLUDED.name_sanskrit, name_chinese = EXCLUDED.name_chinese,
			title_honorific = EXCLUDED.title_honorific, thumbnail_url = EXCLUDED.thumbnail_url,
			image_url = EXCLUDED.image_url, canonical_source = EXCLUDED.canonical_source,
			core_virtue = EXCLUDED.core_virtue, pure_land = EXCLUDED.pure_land,
			lunar_memorial_days = EXCLUDED.lunar_memorial_days,
			iconography_posture = EXCLUDED.iconography_posture,
			iconography_mount_or_throne = EXCLUDED.iconography_mount_or_throne,
			sacred_artifacts = EXCLUDED.sacred_artifacts, symbolic_meaning = EXCLUDED.symbolic_meaning,
			biography_summary = EXCLUDED.biography_summary, great_vows = EXCLUDED.great_vows,
			short_mantra_sanskrit = EXCLUDED.short_mantra_sanskrit,
			short_mantra_sino_vietnamese = EXCLUDED.short_mantra_sino_vietnamese,
			long_dharani_name = EXCLUDED.long_dharani_name,
			dharani_phonetic_snippet = EXCLUDED.dharani_phonetic_snippet, updated_at = now()
	`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert deities: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, d := range deities {
		days := make([]memorialDayJSON, 0, len(d.LunarMemorialDays))
		for _, m := range d.LunarMemorialDays {
			days = append(days, memorialDayJSON{LunarDay: m.LunarDay, LunarMonth: m.LunarMonth, EventName: m.EventName})
		}
		daysJSON, err := json.Marshal(days)
		if err != nil {
			return fmt.Errorf("marshal memorial days of %q: %w", d.ID, err)
		}
		artifacts, vows := d.Iconography.SacredArtifacts, d.GreatVows
		if artifacts == nil {
			artifacts = []string{}
		}
		if vows == nil {
			vows = []string{}
		}
		if _, err := tx.ExecContext(ctx, query,
			d.ID, d.Type, d.SortOrder, d.NameVI, d.NameSanskrit, d.NameChinese, d.TitleHonorific,
			d.ThumbnailURL, d.ImageURL, d.CanonicalSource, d.CoreVirtue, d.PureLand, daysJSON,
			d.Iconography.Posture, d.Iconography.MountOrThrone, pq.Array(artifacts), d.Iconography.SymbolicMeaning,
			d.BiographySummary, pq.Array(vows), d.Mantras.ShortSanskrit, d.Mantras.ShortSinoVietnamese,
			d.Mantras.LongDharaniName, d.Mantras.DharaniPhoneticSnippet,
		); err != nil {
			return fmt.Errorf("upsert deity %q: %w", d.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert deities: %w", err)
	}
	return nil
}

func scanDeity(row rowScanner) (*domain.Deity, error) {
	var d domain.Deity
	var daysJSON []byte
	if err := row.Scan(
		&d.ID, &d.Type, &d.SortOrder, &d.NameVI, &d.NameSanskrit, &d.NameChinese, &d.TitleHonorific,
		&d.ThumbnailURL, &d.ImageURL, &d.CanonicalSource, &d.CoreVirtue, &d.PureLand, &daysJSON,
		&d.Iconography.Posture, &d.Iconography.MountOrThrone, pq.Array(&d.Iconography.SacredArtifacts),
		&d.Iconography.SymbolicMeaning, &d.BiographySummary, pq.Array(&d.GreatVows),
		&d.Mantras.ShortSanskrit, &d.Mantras.ShortSinoVietnamese, &d.Mantras.LongDharaniName,
		&d.Mantras.DharaniPhoneticSnippet,
	); err != nil {
		return nil, err
	}
	var days []memorialDayJSON
	if err := json.Unmarshal(daysJSON, &days); err != nil {
		return nil, fmt.Errorf("decode memorial days of %q: %w", d.ID, err)
	}
	for _, m := range days {
		d.LunarMemorialDays = append(d.LunarMemorialDays, domain.MemorialDay{
			LunarDay: m.LunarDay, LunarMonth: m.LunarMonth, EventName: m.EventName,
		})
	}
	return &d, nil
}

func scanDeities(rows *sql.Rows) ([]domain.Deity, error) {
	defer func() { _ = rows.Close() }()
	items := []domain.Deity{}
	for rows.Next() {
		d, err := scanDeity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan deity: %w", err)
		}
		items = append(items, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deities: %w", err)
	}
	return items, nil
}
