package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type templeScheduleRepository struct {
	db database.Database
}

// NewTempleScheduleRepository returns the Postgres-backed
// domain.TempleScheduleRepository.
func NewTempleScheduleRepository(db database.Database) domain.TempleScheduleRepository {
	return &templeScheduleRepository{db: db}
}

// ListActiveByTemple reads the temple's active sessions, then the steps of
// those sessions in a second query (skipped when there is no session). Times
// are selected as "HH24:MI" text and parsed by the domain.
func (r *templeScheduleRepository) ListActiveByTemple(ctx context.Context, templeID string) ([]domain.TempleScheduleItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, temple_id, name, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       weekdays, description, instructor, location, preparation, sort_order
		FROM temple_schedules
		WHERE temple_id = $1 AND is_active
		ORDER BY start_time, sort_order, id`, templeID)
	if err != nil {
		return nil, fmt.Errorf("query temple schedules: %w", err)
	}
	items, err := scanRows(rows, "temple schedule", func(s rowScanner) (domain.TempleScheduleItem, error) {
		var (
			it       domain.TempleScheduleItem
			start    string
			end      sql.NullString
			weekdays pq.Int64Array
		)
		if err := s.Scan(&it.ID, &it.TempleID, &it.Name, &start, &end, &weekdays,
			&it.Description, &it.Instructor, &it.Location, &it.Preparation, &it.SortOrder); err != nil {
			return it, err
		}
		st, err := domain.ParseTimeOfDay(start)
		if err != nil {
			return it, err
		}
		it.StartTime = st
		if end.Valid {
			et, err := domain.ParseTimeOfDay(end.String)
			if err != nil {
				return it, err
			}
			it.EndTime = &et
		}
		it.Weekdays = make([]int, len(weekdays))
		for i, d := range weekdays {
			it.Weekdays[i] = int(d)
		}
		it.Steps = []domain.TempleScheduleStep{}
		return it, nil
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	ids := make([]string, len(items))
	byID := make(map[string]*domain.TempleScheduleItem, len(items))
	for i := range items {
		ids[i] = items[i].ID
		byID[items[i].ID] = &items[i]
	}
	stepRows, err := r.db.Query(ctx, `
		SELECT schedule_id, position, to_char(step_time, 'HH24:MI'), text
		FROM temple_schedule_steps
		WHERE schedule_id = ANY($1)
		ORDER BY schedule_id, position`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("query temple schedule steps: %w", err)
	}
	type stepRow struct {
		scheduleID string
		step       domain.TempleScheduleStep
	}
	steps, err := scanRows(stepRows, "temple schedule step", func(s rowScanner) (stepRow, error) {
		var (
			sr stepRow
			tm sql.NullString
		)
		if err := s.Scan(&sr.scheduleID, &sr.step.Position, &tm, &sr.step.Text); err != nil {
			return sr, err
		}
		if tm.Valid {
			t, err := domain.ParseTimeOfDay(tm.String)
			if err != nil {
				return sr, err
			}
			sr.step.Time = &t
		}
		return sr, nil
	})
	if err != nil {
		return nil, err
	}
	for _, sr := range steps {
		if it := byID[sr.scheduleID]; it != nil {
			it.Steps = append(it.Steps, sr.step)
		}
	}
	return items, nil
}

// ReplaceCatalogue loads the catalogue in one transaction: every temple_id must
// exist, sessions are upserted by id without touching is_active, and the steps
// of every session in c are deleted and re-inserted. Nothing else is deleted.
func (r *templeScheduleRepository) ReplaceCatalogue(ctx context.Context, c domain.TempleScheduleCatalogue) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace temple schedules: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	known, err := templeIDSet(ctx, tx)
	if err != nil {
		return err
	}
	for _, it := range c.Items {
		if !known[it.TempleID] {
			return fmt.Errorf("temple schedule %q: temple %q does not exist", it.ID, it.TempleID)
		}
	}

	if err := upsertBatched(ctx, tx, upsertSpec{"temple_schedules",
		[]string{"id", "temple_id", "name", "start_time", "end_time", "weekdays",
			"description", "instructor", "location", "preparation", "sort_order"},
		[]string{"id"}},
		c.Items, func(it domain.TempleScheduleItem) []any {
			var end any
			if it.EndTime != nil {
				end = it.EndTime.String()
			}
			days := make([]int64, len(it.Weekdays))
			for i, d := range it.Weekdays {
				days[i] = int64(d)
			}
			return []any{it.ID, it.TempleID, it.Name, it.StartTime.String(), end, pq.Array(days),
				it.Description, it.Instructor, it.Location, it.Preparation, it.SortOrder}
		}); err != nil {
		return err
	}

	if len(c.Items) > 0 {
		ids := make([]string, len(c.Items))
		for i, it := range c.Items {
			ids[i] = it.ID
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM temple_schedule_steps WHERE schedule_id = ANY($1)`, pq.Array(ids)); err != nil {
			return fmt.Errorf("clear temple schedule steps: %w", err)
		}
	}
	type stepRow struct {
		scheduleID string
		step       domain.TempleScheduleStep
	}
	var steps []stepRow
	for _, it := range c.Items {
		for _, s := range it.Steps {
			steps = append(steps, stepRow{it.ID, s})
		}
	}
	if err := upsertBatched(ctx, tx, upsertSpec{"temple_schedule_steps",
		[]string{"schedule_id", "position", "step_time", "text"}, []string{"schedule_id", "position"}},
		steps, func(s stepRow) []any {
			var tm any
			if s.step.Time != nil {
				tm = s.step.Time.String()
			}
			return []any{s.scheduleID, s.step.Position, tm, s.step.Text}
		}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace temple schedules: %w", err)
	}
	return nil
}

func templeIDSet(ctx context.Context, tx *sql.Tx) (map[string]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM temples`)
	if err != nil {
		return nil, fmt.Errorf("query temple ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	set := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan temple id: %w", err)
		}
		set[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate temple ids: %w", err)
	}
	return set, nil
}
