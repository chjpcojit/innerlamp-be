package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// lampColumns is shared by every read of a whole lamp so they scan identically.
const lampColumns = `id, user_id, lamp_type, intention, recipient, prayer, duration_days, is_anonymous,
		lighter_name, lit_at, end_at`

type lampRepository struct {
	db database.Database
}

// NewLampRepository returns the Postgres-backed domain.LampRepository.
func NewLampRepository(db database.Database) domain.LampRepository {
	return &lampRepository{db: db}
}

// Light stores a lamp in one transaction (docs/specs/features/lamp design.md,
// Step L-3).
//
// The users row is locked first. That serializes concurrent Light calls of the
// same user, so neither the cap count nor the double-tap lookup below can go
// stale between the check and the insert: without the lock two requests at 19
// lamps would both see 19 and both insert, and two double taps would both miss
// each other. The same statement reads the dharma name, so the copy put on the
// lamp is the one in force inside this transaction.
func (r *lampRepository) Light(ctx context.Context, p domain.LightLampParams) (*domain.Lamp, bool, error) {
	l := p.Lamp

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var dharmaName sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT dharma_name FROM users WHERE id = $1 FOR UPDATE`, l.UserID).Scan(&dharmaName); err != nil {
		return nil, false, fmt.Errorf("lock user %d: %w", l.UserID, err)
	}

	// An identical lamp lit within the window is the same submission: hand it
	// back instead of writing a second one. This runs before the cap check so a
	// double tap at the cap still gets its lamp rather than a 409. recipient and
	// prayer may be NULL, so they compare with IS NOT DISTINCT FROM.
	// VI: recipient và prayer có thể NULL nên so bằng IS NOT DISTINCT FROM; dùng
	// "=" thì hai lần bấm đúp không có người nhận sẽ tạo hai đèn.
	existing, err := scanLamp(tx.QueryRowContext(ctx, `
		SELECT `+lampColumns+` FROM lamps
		WHERE user_id = $1 AND lit_at > $2
		  AND lamp_type = $3 AND intention = $4 AND recipient IS NOT DISTINCT FROM $5
		  AND prayer IS NOT DISTINCT FROM $6
		  AND duration_days = $7 AND is_anonymous = $8
		ORDER BY lit_at DESC, id DESC
		LIMIT 1`,
		l.UserID, l.LitAt.Add(-p.DedupeWindow),
		string(l.LampType), string(l.Intention), nullableStringPtr(l.Recipient), nullableStringPtr(l.Prayer),
		l.DurationDays, l.IsAnonymous))
	switch {
	case err == nil:
		if err := tx.Commit(); err != nil {
			return nil, false, fmt.Errorf("commit: %w", err)
		}
		return existing, false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return nil, false, fmt.Errorf("find identical lamp: %w", err)
	}

	var active int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM lamps WHERE user_id = $1 AND end_at > $2`, l.UserID, l.LitAt).Scan(&active); err != nil {
		return nil, false, fmt.Errorf("count lit lamps: %w", err)
	}
	if active >= p.MaxActive {
		return nil, false, domain.ErrLampLimitReached
	}

	// An anonymous lamp keeps no name; otherwise copy the dharma name (NULL when
	// the user has none).
	// VI: Đèn ẩn danh không lưu tên; ngược lại chép pháp danh (NULL khi chưa có).
	var dharma *string
	if dharmaName.Valid {
		dharma = &dharmaName.String
	}
	name := domain.LighterNameAtLighting(dharma, l.IsAnonymous)

	created, err := scanLamp(tx.QueryRowContext(ctx, `
		INSERT INTO lamps
			(user_id, lamp_type, intention, recipient, prayer, duration_days, is_anonymous, lighter_name, lit_at, end_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+lampColumns,
		l.UserID, string(l.LampType), string(l.Intention), nullableStringPtr(l.Recipient), nullableStringPtr(l.Prayer),
		l.DurationDays, l.IsAnonymous, nullableStringPtr(name), l.LitAt, l.EndAt))
	if err != nil {
		return nil, false, fmt.Errorf("insert lamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit: %w", err)
	}
	return created, true, nil
}

// Garden reads the per-type counts and one page in a single read-only
// REPEATABLE READ transaction, so the totals and the page describe the same
// moment (design.md, Step G-3). It fetches Limit+1 rows; the extra row only
// tells the caller there is a next page.
func (r *lampRepository) Garden(ctx context.Context, q domain.GardenQuery) (*domain.GardenPage, error) {
	sqlDB, err := r.db.GetSQLDB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	tx, err := sqlDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, fmt.Errorf("begin read tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	page := &domain.GardenPage{Counts: make(map[domain.LampType]int, len(domain.AllLampTypes))}

	countRows, err := tx.QueryContext(ctx,
		`SELECT lamp_type, count(*) FROM lamps WHERE end_at > $1 GROUP BY lamp_type`, q.Now)
	if err != nil {
		return nil, fmt.Errorf("count lit lamps: %w", err)
	}
	for countRows.Next() {
		var t string
		var n int
		if err := countRows.Scan(&t, &n); err != nil {
			_ = countRows.Close()
			return nil, fmt.Errorf("scan count: %w", err)
		}
		page.Counts[domain.LampType(t)] = n
		page.Total += n
	}
	if err := countRows.Err(); err != nil {
		_ = countRows.Close()
		return nil, fmt.Errorf("iterate counts: %w", err)
	}
	_ = countRows.Close()

	// is_mine is computed in SQL so user_id never leaves the database.
	// VI: is_mine được tính trong SQL để user_id không bao giờ ra khỏi database.
	query := `SELECT id, lamp_type, recipient, lighter_name, lit_at, end_at, user_id = $2 AS is_mine
		FROM lamps WHERE end_at > $1`
	args := []any{q.Now, q.ViewerID}
	if q.LampType != nil {
		args = append(args, string(*q.LampType))
		query += fmt.Sprintf(" AND lamp_type = $%d", len(args))
	}
	if q.After != nil {
		args = append(args, q.After.LitAt, q.After.ID)
		query += fmt.Sprintf(" AND (lit_at, id) < ($%d::timestamptz, $%d::bigint)", len(args)-1, len(args))
	}
	args = append(args, q.Limit+1)
	query += fmt.Sprintf(" ORDER BY lit_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query garden: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			g         domain.GardenLamp
			t         string
			recipient sql.NullString
			name      sql.NullString
		)
		if err := rows.Scan(&g.ID, &t, &recipient, &name, &g.LitAt, &g.EndAt, &g.IsMine); err != nil {
			return nil, fmt.Errorf("scan garden lamp: %w", err)
		}
		g.LampType = domain.LampType(t)
		if recipient.Valid {
			s := recipient.String
			g.Recipient = &s
		}
		if name.Valid {
			s := name.String
			g.LighterName = &s
		}
		g.LitAt, g.EndAt = g.LitAt.UTC(), g.EndAt.UTC()
		page.Items = append(page.Items, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate garden: %w", err)
	}
	return page, nil
}

func scanLamp(row rowScanner) (*domain.Lamp, error) {
	var (
		l          domain.Lamp
		lampType   string
		intention  string
		recipient  sql.NullString
		prayer     sql.NullString
		lighter    sql.NullString
		litAt, end time.Time
	)
	if err := row.Scan(&l.ID, &l.UserID, &lampType, &intention, &recipient, &prayer, &l.DurationDays,
		&l.IsAnonymous, &lighter, &litAt, &end); err != nil {
		return nil, err
	}
	l.LampType = domain.LampType(lampType)
	l.Intention = domain.Intention(intention)
	if recipient.Valid {
		s := recipient.String
		l.Recipient = &s
	}
	if prayer.Valid {
		s := prayer.String
		l.Prayer = &s
	}
	if lighter.Valid {
		s := lighter.String
		l.LighterName = &s
	}
	l.LitAt, l.EndAt = litAt.UTC(), end.UTC()
	return &l, nil
}
