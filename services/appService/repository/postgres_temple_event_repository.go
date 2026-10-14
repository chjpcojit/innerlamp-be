package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type templeEventRepository struct {
	db database.Database
}

// NewTempleEventRepository returns the Postgres-backed domain.TempleEventRepository.
func NewTempleEventRepository(db database.Database) domain.TempleEventRepository {
	return &templeEventRepository{db: db}
}

const templeEventDateLayout = "2006-01-02"

// ListActiveByTemple is a coarse filter, generous by a day or two on each side
// so no time zone has to be known here: the resolver does the exact test. A
// one-off event is kept when it ends on or after the day before the window; a
// recurring one when its first occurrence starts before the window ends and its
// recurrence_until (if any) is not before the day before the window.
func (r *templeEventRepository) ListActiveByTemple(ctx context.Context, templeID string, w domain.TempleEventWindow) ([]domain.TempleEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, temple_id, title, description, kind, starts_at, ends_at, is_all_day, status,
		       audience_note, contact_info, image_url, recurrence, recurrence_lunar_days,
		       recurrence_weekday, recurrence_until
		FROM temple_events
		WHERE temple_id = $1 AND is_active
		  AND starts_at < $2
		  AND (recurrence <> 'NONE' OR ends_at >= $3)
		  AND (recurrence_until IS NULL OR recurrence_until >= $4::date)
		ORDER BY starts_at, id`,
		templeID, w.To.AddDate(0, 0, 2), w.From.AddDate(0, 0, -1), w.From.AddDate(0, 0, -1).Format(templeEventDateLayout))
	if err != nil {
		return nil, fmt.Errorf("query temple events: %w", err)
	}
	return scanRows(rows, "temple event", func(s rowScanner) (domain.TempleEvent, error) {
		var (
			e         domain.TempleEvent
			kind      string
			status    string
			recur     string
			image     sql.NullString
			lunarDays pq.Int64Array
			weekday   sql.NullInt64
			until     sql.NullTime
		)
		if err := s.Scan(&e.ID, &e.TempleID, &e.Title, &e.Description, &kind, &e.StartsAt, &e.EndsAt,
			&e.IsAllDay, &status, &e.AudienceNote, &e.ContactInfo, &image, &recur, &lunarDays,
			&weekday, &until); err != nil {
			return e, err
		}
		e.Kind = domain.TempleEventKind(kind)
		e.Status = domain.TempleEventStatus(status)
		e.Recurrence = domain.TempleRecurrence(recur)
		if image.Valid {
			e.ImageURL = &image.String
		}
		e.LunarDays = make([]int, len(lunarDays))
		for i, d := range lunarDays {
			e.LunarDays[i] = int(d)
		}
		if weekday.Valid {
			v := int(weekday.Int64)
			e.Weekday = &v
		}
		if until.Valid {
			y, m, d := until.Time.Date()
			t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
			e.Until = &t
		}
		return e, nil
	})
}

// ReplaceCatalogue upserts the events in one transaction after checking every
// temple_id exists. is_active is deliberately not written: a new event takes the
// column default (true) and an existing one keeps what back-office set.
func (r *templeEventRepository) ReplaceCatalogue(ctx context.Context, c domain.TempleEventCatalogue) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace temple events: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	known, err := templeIDSet(ctx, tx)
	if err != nil {
		return err
	}
	for _, e := range c.Events {
		if !known[e.TempleID] {
			return fmt.Errorf("temple event %q: temple %q does not exist", e.ID, e.TempleID)
		}
	}

	if err := upsertBatched(ctx, tx, upsertSpec{"temple_events",
		[]string{"id", "temple_id", "title", "description", "kind", "starts_at", "ends_at", "is_all_day", "status",
			"audience_note", "contact_info", "image_url", "recurrence", "recurrence_lunar_days",
			"recurrence_weekday", "recurrence_until"},
		[]string{"id"}},
		c.Events, templeEventRow); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace temple events: %w", err)
	}
	return nil
}

func templeEventRow(e domain.TempleEvent) []any {
	var image, weekday, until any
	if e.ImageURL != nil {
		image = *e.ImageURL
	}
	if e.Weekday != nil {
		weekday = *e.Weekday
	}
	if e.Until != nil {
		until = e.Until.Format(templeEventDateLayout)
	}
	days := make([]int64, len(e.LunarDays))
	for i, d := range e.LunarDays {
		days[i] = int64(d)
	}
	return []any{e.ID, e.TempleID, e.Title, e.Description, string(e.Kind), e.StartsAt, e.EndsAt, e.IsAllDay,
		string(e.Status), e.AudienceNote, e.ContactInfo, image, string(e.Recurrence), pq.Array(days), weekday, until}
}
