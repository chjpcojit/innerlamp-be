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

// memorialColumns is shared by every read so list and detail scan identically.
// reminder_time is formatted in SQL as "HH:MM", exactly the string the domain stores,
// instead of scanning a TIME value into a time.Time.
const memorialColumns = `id, user_id, full_name, relationship, death_calendar, death_date, death_day,
		death_month, death_leap_month, note, photo_object_key, reminder_days_before,
		to_char(reminder_time, 'HH24:MI'), reminder_timezone, add_to_os_reminder, created_at, updated_at`

type memorialRepository struct {
	db database.Database
}

// NewMemorialRepository returns the Postgres-backed domain.MemorialRepository.
func NewMemorialRepository(db database.Database) domain.MemorialRepository {
	return &memorialRepository{db: db}
}

// Create inserts m unless the user is at max memorials, in one transaction.
//
// The users row is locked first: that serializes concurrent Creates of the same
// user, so the count below cannot go stale between the check and the insert.
// Without the lock two requests at 99 memorials would both see 99 and both insert.
func (r *memorialRepository) Create(ctx context.Context, m *domain.Memorial, max int) (*domain.Memorial, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var lockedID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, m.UserID).Scan(&lockedID); err != nil {
		return nil, fmt.Errorf("lock user %d: %w", m.UserID, err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM memorials WHERE user_id = $1`, m.UserID).Scan(&count); err != nil {
		return nil, fmt.Errorf("count memorials: %w", err)
	}
	if count >= max {
		return nil, domain.ErrMemorialLimitReached
	}

	days, rtime, zone := reminderColumns(m.Reminder)
	row := tx.QueryRowContext(ctx, `
		INSERT INTO memorials
			(user_id, full_name, relationship, death_calendar, death_date, death_day, death_month,
			 death_leap_month, note, reminder_days_before, reminder_time, reminder_timezone, add_to_os_reminder)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+memorialColumns,
		m.UserID, m.FullName, string(m.Relationship), string(m.DeathCalendar), nullableTime(m.DeathDate),
		nullableInt(m.DeathDay), nullableInt(m.DeathMonth), nullableBool(m.DeathLeapMonth), nullableStringPtr(m.Note),
		days, rtime, zone, m.AddToOSReminder)
	created, err := scanMemorial(row)
	if err != nil {
		return nil, fmt.Errorf("insert memorial: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return created, nil
}

// ListByUser returns the user's memorials, newest first.
func (r *memorialRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Memorial, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+memorialColumns+` FROM memorials WHERE user_id = $1 ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query memorials: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.Memorial
	for rows.Next() {
		m, err := scanMemorial(rows)
		if err != nil {
			return nil, fmt.Errorf("scan memorial: %w", err)
		}
		out = append(out, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memorials: %w", err)
	}
	return out, nil
}

// FindByIDAndUser returns the memorial or domain.ErrMemorialNotFound. Filtering
// on user_id here is what keeps another user's row unreadable.
func (r *memorialRepository) FindByIDAndUser(ctx context.Context, id, userID int64) (*domain.Memorial, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+memorialColumns+` FROM memorials WHERE id = $1 AND user_id = $2`, id, userID)
	m, err := scanMemorial(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMemorialNotFound
		}
		return nil, fmt.Errorf("find memorial: %w", err)
	}
	return m, nil
}

// Update replaces the editable fields. owner, created_at and photo are not in the SET list.
func (r *memorialRepository) Update(ctx context.Context, m *domain.Memorial) (*domain.Memorial, error) {
	days, rtime, zone := reminderColumns(m.Reminder)
	row := r.db.QueryRow(ctx, `
		UPDATE memorials SET
			full_name = $3, relationship = $4, death_calendar = $5, death_date = $6, death_day = $7,
			death_month = $8, death_leap_month = $9, note = $10, reminder_days_before = $11,
			reminder_time = $12, reminder_timezone = $13, add_to_os_reminder = $14, updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+memorialColumns,
		m.ID, m.UserID, m.FullName, string(m.Relationship), string(m.DeathCalendar), nullableTime(m.DeathDate),
		nullableInt(m.DeathDay), nullableInt(m.DeathMonth), nullableBool(m.DeathLeapMonth), nullableStringPtr(m.Note),
		days, rtime, zone, m.AddToOSReminder)
	updated, err := scanMemorial(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMemorialNotFound
		}
		return nil, fmt.Errorf("update memorial: %w", err)
	}
	return updated, nil
}

// Delete removes the memorial or returns domain.ErrMemorialNotFound.
func (r *memorialRepository) Delete(ctx context.Context, id, userID int64) error {
	res, err := r.db.Exec(ctx, `DELETE FROM memorials WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete memorial: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete memorial rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrMemorialNotFound
	}
	return nil
}

func scanMemorial(s rowScanner) (*domain.Memorial, error) {
	var (
		m                                   domain.Memorial
		relationship, calendar              string
		deathDate                           sql.NullTime
		day, month                          sql.NullInt64
		leap                                sql.NullBool
		note, photoKey, remindTime, remZone sql.NullString
		remindDays                          sql.NullInt64
	)
	if err := s.Scan(&m.ID, &m.UserID, &m.FullName, &relationship, &calendar, &deathDate, &day, &month, &leap,
		&note, &photoKey, &remindDays, &remindTime, &remZone, &m.AddToOSReminder, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}

	m.Relationship = domain.Relationship(relationship)
	m.DeathCalendar = domain.DeathCalendar(calendar)
	if deathDate.Valid {
		d := deathDate.Time
		m.DeathDate = &d
	}
	if day.Valid {
		v := int(day.Int64)
		m.DeathDay = &v
	}
	if month.Valid {
		v := int(month.Int64)
		m.DeathMonth = &v
	}
	if leap.Valid {
		v := leap.Bool
		m.DeathLeapMonth = &v
	}
	if note.Valid {
		m.Note = &note.String
	}
	if photoKey.Valid {
		m.PhotoObjectKey = &photoKey.String
	}
	if remindDays.Valid {
		m.Reminder = &domain.Reminder{DaysBefore: int(remindDays.Int64), Time: remindTime.String, Timezone: remZone.String}
	}
	return &m, nil
}

// reminderColumns splits a reminder into its three columns; nil → three NULLs.
func reminderColumns(rem *domain.Reminder) (days, at, zone any) {
	if rem == nil {
		return nil, nil, nil
	}
	return rem.DaysBefore, rem.Time, rem.Timezone
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullableInt(i *int) any {
	if i == nil {
		return nil
	}
	return *i
}

func nullableBool(b *bool) any {
	if b == nil {
		return nil
	}
	return *b
}

func nullableStringPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
