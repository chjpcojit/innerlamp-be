package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

const (
	// sutraColumns / sutraOrder are shared by every list read so the list and the
	// by-deity list scan identically and paging order can never diverge.
	sutraColumns = `s.id, s.sort_order, s.tradition, s.kind, s.collection, s.reference, s.titles,
		s.daily_chant, s.purposes, s.duration_minutes, s.audio_url, s.review_status, s.summary`

	// sutraDetailColumns are read by FindByID only: the list keeps its card shape, so it
	// never loads the split reference fields (summary is on the card, in sutraColumns).
	sutraDetailColumns = `s.original_title, s.author, s.han_translator, s.han_translation_year,
		s.reference_note`

	// id is the final tiebreaker so equal sort_order rows cannot swap between pages.
	sutraOrder = `ORDER BY s.sort_order ASC, s.id ASC`

	// deityRoleRank puts the deity's most specific relationship first: the sutras that
	// are about it, then the ones it speaks, then the ones that merely mention it.
	deityRoleRank = `CASE %s WHEN 'SUBJECT' THEN 0 WHEN 'SPEAKER' THEN 1 ELSE 2 END`

	// insertLinesChunk is how many lines go into one multi-row INSERT: 4 parameters
	// each, well under Postgres's 65535-parameter limit.
	insertLinesChunk = 2000
)

type sutraRepository struct {
	db database.Database
}

// NewSutraRepository returns the Postgres-backed domain.SutraRepository.
func NewSutraRepository(db database.Database) domain.SutraRepository {
	return &sutraRepository{db: db}
}

// sectionJSON is the stored shape of one table-of-contents entry
// (sutra_translations.sections).
type sectionJSON struct {
	Name      string `json:"name"`
	StartN    int    `json:"start_n"`
	LineCount int    `json:"line_count"`
}

// sutraWhere turns the validated filter into a WHERE clause (empty for the zero
// filter) and its arguments, numbered from $1. Every value is a parameter; nothing
// from the request is concatenated into the SQL.
func sutraWhere(f domain.SutraFilter) (string, []any) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.Tradition != nil {
		add("s.tradition = ?", string(*f.Tradition))
	}
	if f.Kind != nil {
		add("s.kind = ?", string(*f.Kind))
	}
	if f.Collection != nil {
		add("s.collection = ?", *f.Collection)
	}
	if f.Purpose != nil {
		add("s.purposes @> ARRAY[?]::text[]", string(*f.Purpose))
	}
	if f.Daily != nil {
		add("s.daily_chant = ?", *f.Daily)
	}
	if f.MaxMinutes != nil {
		// NULL <= n is NULL, so a sutra with no estimate drops out, as specified.
		add("s.duration_minutes <= ?", *f.MaxMinutes)
	}
	if f.Lang != nil {
		add("EXISTS (SELECT 1 FROM sutra_translations t WHERE t.sutra_id = s.id AND t.lang = ?)", *f.Lang)
	}
	if f.Query != nil {
		// Substring match, ignoring case and Vietnamese diacritics, on every title
		// (any language), the collection and the reference. The term is a parameter;
		// LIKE wildcards in it are escaped so "50%" or "a_b" match literally.
		add(`(unaccent(lower(s.collection || ' ' || s.reference)) LIKE unaccent(lower(?)) ESCAPE '\'
			OR EXISTS (SELECT 1 FROM jsonb_each_text(s.titles) ti WHERE unaccent(lower(ti.value)) LIKE unaccent(lower(?)) ESCAPE '\'))`,
			likeContains(*f.Query))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

// likeContains turns a search term into a LIKE pattern that matches it anywhere,
// with \, % and _ escaped so the term is always taken literally.
func likeContains(term string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term) + "%"
}

// Count returns the number of sutras matching the filter.
func (r *sutraRepository) Count(ctx context.Context, filter domain.SutraFilter) (int, error) {
	where, args := sutraWhere(filter)
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM sutras s `+where, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count sutras: %w", err)
	}
	return n, nil
}

// List returns one window of the ordered, filtered list in list shape.
func (r *sutraRepository) List(ctx context.Context, filter domain.SutraFilter, offset, limit int) ([]domain.Sutra, error) {
	where, args := sutraWhere(filter)
	args = append(args, offset, limit)
	query := `SELECT ` + sutraColumns + ` FROM sutras s ` + where + ` ` + sutraOrder +
		fmt.Sprintf(` OFFSET $%d LIMIT $%d`, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sutras: %w", err)
	}
	items, err := scanSutras(rows)
	if err != nil {
		return nil, err
	}
	if err := r.attach(ctx, items, false); err != nil {
		return nil, err
	}
	return items, nil
}

// FindByID returns the sutra in detail shape or domain.ErrSutraNotFound.
func (r *sutraRepository) FindByID(ctx context.Context, id string) (*domain.Sutra, error) {
	row := r.db.QueryRow(ctx, `SELECT `+sutraColumns+`, `+sutraDetailColumns+` FROM sutras s WHERE s.id = $1`, id)
	var originalTitle, author, hanTranslator, referenceNote string
	var hanYear sql.NullInt64
	s, err := scanSutraWith(row, &originalTitle, &author, &hanTranslator, &hanYear, &referenceNote)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSutraNotFound
		}
		return nil, fmt.Errorf("find sutra: %w", err)
	}
	s.OriginalTitle, s.Author, s.HanTranslator = originalTitle, author, hanTranslator
	s.ReferenceNote = referenceNote
	if hanYear.Valid {
		y := int(hanYear.Int64)
		s.HanTranslationYear = &y
	}
	one := []domain.Sutra{*s}
	if err := r.attach(ctx, one, true); err != nil {
		return nil, err
	}
	return &one[0], nil
}

// FindTranslation checks the sutra first, then the key, so the caller can tell an
// unknown sutra from an unknown translation of a known one.
func (r *sutraRepository) FindTranslation(ctx context.Context, sutraID, key string) (*domain.SutraTranslation, error) {
	var one int
	if err := r.db.QueryRow(ctx, `SELECT 1 FROM sutras WHERE id = $1`, sutraID).Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSutraNotFound
		}
		return nil, fmt.Errorf("check sutra: %w", err)
	}

	var t domain.SutraTranslation
	var variant string
	err := r.db.QueryRow(ctx,
		`SELECT id, key, lang, variant, line_count FROM sutra_translations WHERE sutra_id = $1 AND key = $2`,
		sutraID, key,
	).Scan(&t.ID, &t.Key, &t.Lang, &variant, &t.LineCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSutraTranslationNotFound
		}
		return nil, fmt.Errorf("find translation: %w", err)
	}
	t.Variant = domain.Variant(variant)
	return &t, nil
}

// ListLines returns the lines with fromN < n <= toN. n runs 1..line_count without
// gaps, so this range IS the page; the primary key (translation_id, n) serves it.
// The ::bigint casts matter: n is an INTEGER, so Postgres would otherwise type the
// parameters as integer and reject a range beyond 2^31 on a far-past-the-end page.
func (r *sutraRepository) ListLines(ctx context.Context, translationID int64, fromN, toN int64) ([]domain.SutraLine, error) {
	rows, err := r.db.Query(ctx,
		`SELECT n, text, section FROM sutra_lines
		 WHERE translation_id = $1 AND n > $2::bigint AND n <= $3::bigint ORDER BY n ASC`,
		translationID, fromN, toN)
	if err != nil {
		return nil, fmt.Errorf("query sutra lines: %w", err)
	}
	defer func() { _ = rows.Close() }()

	lines := []domain.SutraLine{}
	for rows.Next() {
		var l domain.SutraLine
		var section sql.NullString
		if err := rows.Scan(&l.N, &l.Text, &section); err != nil {
			return nil, fmt.Errorf("scan sutra line: %w", err)
		}
		l.Section = section.String
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sutra lines: %w", err)
	}
	return lines, nil
}

// CountByDeity counts the sutras a deity is linked to, optionally only with one role.
func (r *sutraRepository) CountByDeity(ctx context.Context, deityID string, role *domain.DeityRole) (int, error) {
	query, args := `SELECT count(*) FROM sutra_deities WHERE deity_id = $1`, []any{deityID}
	if role != nil {
		query += ` AND role = $2`
		args = append(args, string(*role))
	}
	var n int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count sutras of deity: %w", err)
	}
	return n, nil
}

// ListByDeity returns one window of the sutras linked to a deity, in list shape,
// each carrying the deity's role.
func (r *sutraRepository) ListByDeity(ctx context.Context, deityID string, role *domain.DeityRole, offset, limit int) ([]domain.DeitySutra, error) {
	args := []any{deityID}
	where := `WHERE sd.deity_id = $1`
	if role != nil {
		args = append(args, string(*role))
		where += ` AND sd.role = $2`
	}
	args = append(args, offset, limit)
	query := `SELECT ` + sutraColumns + `, sd.role FROM sutras s
		JOIN sutra_deities sd ON sd.sutra_id = s.id ` + where +
		` ORDER BY ` + fmt.Sprintf(deityRoleRank, "sd.role") + `, s.sort_order ASC, s.id ASC` +
		fmt.Sprintf(` OFFSET $%d LIMIT $%d`, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sutras of deity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sutras []domain.Sutra
	var roles []domain.DeityRole
	for rows.Next() {
		var linkRole string
		s, err := scanSutraWith(rows, &linkRole)
		if err != nil {
			return nil, fmt.Errorf("scan sutra of deity: %w", err)
		}
		sutras = append(sutras, *s)
		roles = append(roles, domain.DeityRole(linkRole))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sutras of deity: %w", err)
	}
	_ = rows.Close()

	if err := r.attach(ctx, sutras, false); err != nil {
		return nil, err
	}
	out := make([]domain.DeitySutra, len(sutras))
	for i := range sutras {
		out[i] = domain.DeitySutra{Sutra: sutras[i], Role: roles[i]}
	}
	return out, nil
}

// attach fills in the translations and deity links of sutras. detail selects the
// translation columns: false is the list shape (no sources, no sections), true adds
// them. Both lookups use one query for the whole slice (= ANY), never one per sutra.
func (r *sutraRepository) attach(ctx context.Context, sutras []domain.Sutra, detail bool) error {
	if len(sutras) == 0 {
		return nil
	}
	ids := make([]string, len(sutras))
	at := make(map[string]int, len(sutras))
	for i, s := range sutras {
		ids[i] = s.ID
		at[s.ID] = i
	}

	trs, err := r.loadTranslations(ctx, ids, detail)
	if err != nil {
		return err
	}
	deities, err := r.loadDeities(ctx, ids)
	if err != nil {
		return err
	}
	for i := range sutras {
		sutras[i].Translations = trs[sutras[i].ID]
		sutras[i].Deities = deities[sutras[i].ID]
	}
	return nil
}

func (r *sutraRepository) loadTranslations(ctx context.Context, ids []string, detail bool) (map[string][]domain.SutraTranslation, error) {
	cols := `sutra_id, key, lang, variant, translator, license, line_count`
	if detail {
		cols += `, source_name, source_url, license_url, license_note, note, sections`
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+cols+` FROM sutra_translations WHERE sutra_id = ANY($1) ORDER BY sutra_id, ordinal`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("query sutra translations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string][]domain.SutraTranslation, len(ids))
	for rows.Next() {
		var sutraID, variant string
		var t domain.SutraTranslation
		dest := []any{&sutraID, &t.Key, &t.Lang, &variant, &t.Translator, &t.License, &t.LineCount}
		var sections []byte
		if detail {
			dest = append(dest, &t.SourceName, &t.SourceURL, &t.LicenseURL, &t.LicenseNote, &t.Note, &sections)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan sutra translation: %w", err)
		}
		t.Variant = domain.Variant(variant)
		if detail {
			var raw []sectionJSON
			if err := json.Unmarshal(sections, &raw); err != nil {
				return nil, fmt.Errorf("decode sections of %s/%s: %w", sutraID, t.Key, err)
			}
			for _, sec := range raw {
				t.Sections = append(t.Sections, domain.SutraSection{Name: sec.Name, StartN: sec.StartN, LineCount: sec.LineCount})
			}
		}
		out[sutraID] = append(out[sutraID], t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sutra translations: %w", err)
	}
	return out, nil
}

func (r *sutraRepository) loadDeities(ctx context.Context, ids []string) (map[string][]domain.SutraDeity, error) {
	rows, err := r.db.Query(ctx,
		`SELECT sd.sutra_id, sd.deity_id, sd.role, d.name_vi, d.thumbnail_url
		 FROM sutra_deities sd JOIN deities d ON d.id = sd.deity_id
		 WHERE sd.sutra_id = ANY($1)
		 ORDER BY sd.sutra_id, `+fmt.Sprintf(deityRoleRank, "sd.role")+`, sd.deity_id`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("query sutra deities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string][]domain.SutraDeity, len(ids))
	for rows.Next() {
		var sutraID, role string
		var d domain.SutraDeity
		if err := rows.Scan(&sutraID, &d.DeityID, &role, &d.NameVI, &d.ThumbnailURL); err != nil {
			return nil, fmt.Errorf("scan sutra deity: %w", err)
		}
		d.Role = domain.DeityRole(role)
		out[sutraID] = append(out[sutraID], d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sutra deities: %w", err)
	}
	return out, nil
}

// MissingDeities returns those of ids that the deities table does not hold, in the
// order given.
func (r *sutraRepository) MissingDeities(ctx context.Context, ids []string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM deities WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("query deities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	have := make(map[string]bool, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan deity id: %w", err)
		}
		have[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deities: %w", err)
	}
	var missing []string
	for _, id := range ids {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

// InsertAll inserts every sutra with its translations, lines and deity links in one
// transaction: all rows or none, so a bad row in the seed cannot leave the library
// half-seeded. ON CONFLICT (id) DO NOTHING on sutras keeps concurrent seeders (two
// instances starting together) and existing rows safe: when the sutra row is not
// inserted its children are skipped, never duplicated or merged into the old row.
func (r *sutraRepository) InsertAll(ctx context.Context, seeds []domain.SutraSeed) (domain.SutraSeedResult, error) {
	var res domain.SutraSeedResult
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return res, fmt.Errorf("begin insert sutras: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, seed := range seeds {
		s := seed.Sutra
		titles, err := json.Marshal(s.Titles)
		if err != nil {
			return res, fmt.Errorf("encode titles of %q: %w", s.ID, err)
		}
		purposes := make([]string, len(s.Purposes))
		for i, p := range s.Purposes {
			purposes[i] = string(p)
		}
		var duration any // NULL, not 0, when no estimate was made
		if s.DurationMinutes != nil {
			duration = *s.DurationMinutes
		}
		var hanYear any // NULL, not 0, when the year is unknown
		if s.HanTranslationYear != nil {
			hanYear = *s.HanTranslationYear
		}
		reviewStatus := s.ReviewStatus
		if reviewStatus == "" {
			reviewStatus = "UNREVIEWED"
		}

		var insertedID string
		err = tx.QueryRowContext(ctx, `
			INSERT INTO sutras (id, sort_order, tradition, kind, collection, reference, titles,
				daily_chant, purposes, duration_minutes, review_status,
				original_title, author, han_translator, han_translation_year, reference_note, summary)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			ON CONFLICT (id) DO NOTHING
			RETURNING id`,
			s.ID, s.SortOrder, string(s.Tradition), string(s.Kind), s.Collection, s.Reference, titles,
			s.DailyChant, pq.Array(purposes), duration, reviewStatus,
			s.OriginalTitle, s.Author, s.HanTranslator, hanYear, s.ReferenceNote, s.Summary,
		).Scan(&insertedID)
		if errors.Is(err, sql.ErrNoRows) {
			continue // already stored: leave it, and its children, untouched
		}
		if err != nil {
			return res, fmt.Errorf("insert sutra %q: %w", s.ID, err)
		}
		res.Sutras++

		for ordinal, t := range seed.Translations {
			sections := make([]sectionJSON, len(t.Sections))
			for i, sec := range t.Sections {
				sections[i] = sectionJSON{Name: sec.Name, StartN: sec.StartN, LineCount: sec.LineCount}
			}
			sectionsJSON, err := json.Marshal(sections)
			if err != nil {
				return res, fmt.Errorf("encode sections of %q/%q: %w", s.ID, t.Key, err)
			}
			var translationID int64
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO sutra_translations (sutra_id, key, ordinal, lang, variant, translator,
					source_name, source_url, license, license_url, license_note, note, line_count, sections)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
				RETURNING id`,
				s.ID, t.Key, ordinal, t.Lang, string(t.Variant), t.Translator,
				t.SourceName, t.SourceURL, t.License, t.LicenseURL, t.LicenseNote, t.Note, len(t.Lines), sectionsJSON,
			).Scan(&translationID); err != nil {
				return res, fmt.Errorf("insert translation %q of sutra %q: %w", t.Key, s.ID, err)
			}
			res.Translations++

			n, err := insertLines(ctx, tx, translationID, t.Lines)
			if err != nil {
				return res, fmt.Errorf("insert lines of %q/%q: %w", s.ID, t.Key, err)
			}
			res.Lines += n
		}

		for _, d := range s.Deities {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO sutra_deities (sutra_id, deity_id, role) VALUES ($1,$2,$3)`,
				s.ID, d.DeityID, string(d.Role)); err != nil {
				return res, fmt.Errorf("link sutra %q to deity %q: %w", s.ID, d.DeityID, err)
			}
			res.DeityLinks++
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.SutraSeedResult{}, fmt.Errorf("commit insert sutras: %w", err)
	}
	return res, nil
}

// insertLines writes a translation's lines with multi-row INSERTs and returns how
// many rows were written. A line with no section stores NULL, not ”.
func insertLines(ctx context.Context, tx *sql.Tx, translationID int64, lines []domain.SutraLine) (int, error) {
	for start := 0; start < len(lines); start += insertLinesChunk {
		end := min(start+insertLinesChunk, len(lines))
		var sb strings.Builder
		sb.WriteString(`INSERT INTO sutra_lines (translation_id, n, text, section) VALUES `)
		args := make([]any, 0, (end-start)*4)
		for i, l := range lines[start:end] {
			if i > 0 {
				sb.WriteByte(',')
			}
			p := len(args)
			fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d)", p+1, p+2, p+3, p+4)
			var section any
			if l.Section != "" {
				section = l.Section
			}
			args = append(args, translationID, l.N, l.Text, section)
		}
		if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
			return 0, err
		}
	}
	return len(lines), nil
}

func scanSutra(row rowScanner) (*domain.Sutra, error) { return scanSutraWith(row) }

// scanSutraWith scans the sutra columns, then any extra destinations the caller's
// query appended (the by-deity list adds the role).
func scanSutraWith(row rowScanner, extra ...any) (*domain.Sutra, error) {
	var s domain.Sutra
	var tradition, kind string
	var titles []byte
	var purposes []string
	var duration sql.NullInt64
	dest := append([]any{
		&s.ID, &s.SortOrder, &tradition, &kind, &s.Collection, &s.Reference, &titles,
		&s.DailyChant, pq.Array(&purposes), &duration, &s.AudioURL, &s.ReviewStatus, &s.Summary,
	}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	s.Tradition, s.Kind = domain.Tradition(tradition), domain.Kind(kind)
	if err := json.Unmarshal(titles, &s.Titles); err != nil {
		return nil, fmt.Errorf("decode titles of %s: %w", s.ID, err)
	}
	s.Purposes = make([]domain.Purpose, len(purposes))
	for i, p := range purposes {
		s.Purposes[i] = domain.Purpose(p)
	}
	if duration.Valid {
		d := int(duration.Int64)
		s.DurationMinutes = &d
	}
	return &s, nil
}

func scanSutras(rows *sql.Rows) ([]domain.Sutra, error) {
	defer func() { _ = rows.Close() }()
	items := []domain.Sutra{}
	for rows.Next() {
		s, err := scanSutra(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sutra: %w", err)
		}
		items = append(items, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sutras: %w", err)
	}
	return items, nil
}
