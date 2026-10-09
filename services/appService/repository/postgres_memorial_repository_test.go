package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func intp(i int) *int       { return &i }
func boolp(b bool) *bool    { return &b }
func strp(s string) *string { return &s }

func fullDateMemorial(userID int64, name string) *domain.Memorial {
	d := time.Date(2015, 3, 4, 0, 0, 0, 0, time.UTC)
	return &domain.Memorial{
		UserID: userID, FullName: name, Relationship: domain.RelationshipFather,
		DeathCalendar: domain.DeathCalendarSolar, DeathDate: &d,
	}
}

func newMemorialRepo(t *testing.T) (domain.MemorialRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewMemorialRepository(db), db
}

func TestMemorialRepository_CreateAndFind_FullDate(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	in := fullDateMemorial(uid, "Nguyễn Văn A")
	in.DeathCalendar = domain.DeathCalendarLunar // typed as lunar, stored as the solar date
	in.Note = strp("nhớ ơn")
	in.Reminder = &domain.Reminder{DaysBefore: 3, Time: "07:30", Timezone: "Asia/Ho_Chi_Minh"}
	in.AddToOSReminder = true

	created, err := repo.Create(ctx, in, 100)
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.False(t, created.CreatedAt.IsZero())

	got, err := repo.FindByIDAndUser(ctx, created.ID, uid)
	require.NoError(t, err)
	assert.Equal(t, uid, got.UserID)
	assert.Equal(t, "Nguyễn Văn A", got.FullName)
	assert.Equal(t, domain.RelationshipFather, got.Relationship)
	assert.Equal(t, domain.DeathCalendarLunar, got.DeathCalendar)
	require.NotNil(t, got.DeathDate)
	assert.Equal(t, "2015-03-04", got.DeathDate.Format(time.DateOnly))
	assert.Nil(t, got.DeathDay)
	assert.Nil(t, got.DeathMonth)
	assert.Nil(t, got.DeathLeapMonth, "year known: partial fields round-trip as nil")
	assert.Equal(t, "nhớ ơn", *got.Note)
	assert.Nil(t, got.PhotoObjectKey)
	assert.Equal(t, domain.Reminder{DaysBefore: 3, Time: "07:30", Timezone: "Asia/Ho_Chi_Minh"}, *got.Reminder)
	assert.True(t, got.AddToOSReminder)
}

func TestMemorialRepository_CreateAndFind_PartialDate(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	in := &domain.Memorial{
		UserID: uid, FullName: "Bà ngoại", Relationship: domain.RelationshipGrandmotherMaternal,
		DeathCalendar: domain.DeathCalendarLunar, DeathDay: intp(15), DeathMonth: intp(7), DeathLeapMonth: boolp(true),
	}
	created, err := repo.Create(ctx, in, 100)
	require.NoError(t, err)

	got, err := repo.FindByIDAndUser(ctx, created.ID, uid)
	require.NoError(t, err)
	assert.Nil(t, got.DeathDate, "year unknown: no solar date")
	assert.Equal(t, 15, *got.DeathDay)
	assert.Equal(t, 7, *got.DeathMonth)
	assert.True(t, *got.DeathLeapMonth)
	assert.Nil(t, got.Note, "no note round-trips as nil")
	assert.Nil(t, got.Reminder, "no reminder round-trips as nil")
	assert.False(t, got.AddToOSReminder)
}

func TestMemorialRepository_FindByIDAndUser_NotFound(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	owner := insertUser(t, db, "0900000001", "a@example.com")
	other := insertUser(t, db, "0900000002", "b@example.com")
	created, err := repo.Create(ctx, fullDateMemorial(owner, "A"), 100)
	require.NoError(t, err)

	_, err = repo.FindByIDAndUser(ctx, created.ID+1000, owner)
	assert.ErrorIs(t, err, domain.ErrMemorialNotFound, "no such id")

	_, err = repo.FindByIDAndUser(ctx, created.ID, other)
	assert.ErrorIs(t, err, domain.ErrMemorialNotFound, "someone else's row looks exactly like a missing one")
}

func TestMemorialRepository_ListByUser(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	owner := insertUser(t, db, "0900000001", "a@example.com")
	other := insertUser(t, db, "0900000002", "b@example.com")

	first, err := repo.Create(ctx, fullDateMemorial(owner, "first"), 100)
	require.NoError(t, err)
	second, err := repo.Create(ctx, fullDateMemorial(owner, "second"), 100)
	require.NoError(t, err)
	_, err = repo.Create(ctx, fullDateMemorial(other, "not mine"), 100)
	require.NoError(t, err)

	got, err := repo.ListByUser(ctx, owner)
	require.NoError(t, err)
	require.Len(t, got, 2, "only the owner's rows")
	assert.Equal(t, second.ID, got[0].ID, "newest first")
	assert.Equal(t, first.ID, got[1].ID)

	none, err := repo.ListByUser(ctx, insertUser(t, db, "0900000003", "c@example.com"))
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestMemorialRepository_Update(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	owner := insertUser(t, db, "0900000001", "a@example.com")
	other := insertUser(t, db, "0900000002", "b@example.com")
	created, err := repo.Create(ctx, fullDateMemorial(owner, "Old name"), 100)
	require.NoError(t, err)
	// Put a photo key in place directly: Update must not touch it.
	_, err = db.Exec(ctx, `UPDATE memorials SET photo_object_key = 'k' WHERE id = $1`, created.ID)
	require.NoError(t, err)

	t.Run("owner replaces the editable fields", func(t *testing.T) {
		upd := &domain.Memorial{
			ID: created.ID, UserID: owner, FullName: "New name", Relationship: domain.RelationshipMother,
			DeathCalendar: domain.DeathCalendarLunar, DeathDay: intp(1), DeathMonth: intp(1), DeathLeapMonth: boolp(false),
			Note:     strp("n"),
			Reminder: &domain.Reminder{DaysBefore: 0, Time: "00:00", Timezone: "UTC"},
		}
		got, err := repo.Update(ctx, upd)

		require.NoError(t, err)
		assert.Equal(t, "New name", got.FullName)
		assert.Equal(t, domain.RelationshipMother, got.Relationship)
		assert.Nil(t, got.DeathDate, "switched to the year-unknown form")
		assert.Equal(t, 1, *got.DeathDay)
		assert.Equal(t, domain.Reminder{DaysBefore: 0, Time: "00:00", Timezone: "UTC"}, *got.Reminder)
		assert.Equal(t, owner, got.UserID)
		assert.True(t, got.CreatedAt.Equal(created.CreatedAt), "created_at is not touched")
		assert.False(t, got.UpdatedAt.Before(created.UpdatedAt))
		require.NotNil(t, got.PhotoObjectKey, "the photo is not touched by Update")
		assert.Equal(t, "k", *got.PhotoObjectKey)
	})

	t.Run("another user cannot update it", func(t *testing.T) {
		upd := fullDateMemorial(other, "Hijacked")
		upd.ID = created.ID
		_, err := repo.Update(ctx, upd)
		assert.ErrorIs(t, err, domain.ErrMemorialNotFound)

		got, err := repo.FindByIDAndUser(ctx, created.ID, owner)
		require.NoError(t, err)
		assert.Equal(t, "New name", got.FullName, "row unchanged")
	})
}

func TestMemorialRepository_Delete(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	owner := insertUser(t, db, "0900000001", "a@example.com")
	other := insertUser(t, db, "0900000002", "b@example.com")
	created, err := repo.Create(ctx, fullDateMemorial(owner, "A"), 100)
	require.NoError(t, err)

	assert.ErrorIs(t, repo.Delete(ctx, created.ID, other), domain.ErrMemorialNotFound, "not yours")
	_, err = repo.FindByIDAndUser(ctx, created.ID, owner)
	require.NoError(t, err, "still there after a refused delete")

	require.NoError(t, repo.Delete(ctx, created.ID, owner))
	_, err = repo.FindByIDAndUser(ctx, created.ID, owner)
	assert.ErrorIs(t, err, domain.ErrMemorialNotFound)

	assert.ErrorIs(t, repo.Delete(ctx, created.ID, owner), domain.ErrMemorialNotFound, "second delete")
}

func TestMemorialRepository_Create_LimitIsPerUser(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	a := insertUser(t, db, "0900000001", "a@example.com")
	b := insertUser(t, db, "0900000002", "b@example.com")

	for i := 0; i < 2; i++ {
		_, err := repo.Create(ctx, fullDateMemorial(a, fmt.Sprint("m", i)), 2)
		require.NoError(t, err)
	}
	_, err := repo.Create(ctx, fullDateMemorial(a, "third"), 2)
	assert.ErrorIs(t, err, domain.ErrMemorialLimitReached)

	_, err = repo.Create(ctx, fullDateMemorial(b, "other user is unaffected"), 2)
	require.NoError(t, err)

	got, err := repo.ListByUser(ctx, a)
	require.NoError(t, err)
	assert.Len(t, got, 2, "the refused create left nothing behind")
}

func TestMemorialRepository_Create_ConcurrentCannotExceedLimit(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	const max, attempts = 5, 20

	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.Create(ctx, fullDateMemorial(uid, fmt.Sprint("m", i)), max)
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	ok, refused := 0, 0
	for err := range results {
		switch err {
		case nil:
			ok++
		case domain.ErrMemorialLimitReached:
			refused++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, max, ok)
	assert.Equal(t, attempts-max, refused)

	got, err := repo.ListByUser(ctx, uid)
	require.NoError(t, err)
	assert.Len(t, got, max, "never more than the cap, however the requests interleave")
}

// The database must refuse data the use case would have rejected, so a bug above
// it cannot store an impossible row.
func TestMemorialRepository_DBConstraints(t *testing.T) {
	_, db := newMemorialRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	const base = `INSERT INTO memorials (user_id, full_name, relationship, death_calendar, death_date, death_day, death_month, death_leap_month, reminder_days_before, reminder_time, reminder_timezone, add_to_os_reminder) VALUES ($1, 'x', %s)`
	tests := []struct {
		name string
		sql  string
	}{
		{"unknown relationship", `'UNCLE', 'solar', '2015-03-04', NULL, NULL, NULL, NULL, NULL, NULL, false`},
		{"unknown calendar", `'FATHER', 'gregorian', '2015-03-04', NULL, NULL, NULL, NULL, NULL, NULL, false`},
		{"death_date before 1900", `'FATHER', 'solar', '1899-12-31', NULL, NULL, NULL, NULL, NULL, NULL, false`},
		{"both forms at once", `'FATHER', 'lunar', '2015-03-04', 1, 1, false, NULL, NULL, NULL, false`},
		{"neither form", `'FATHER', 'lunar', NULL, NULL, NULL, NULL, NULL, NULL, NULL, false`},
		{"partial form without leap flag", `'FATHER', 'lunar', NULL, 1, 1, NULL, NULL, NULL, NULL, false`},
		{"leap month on solar", `'FATHER', 'solar', NULL, 1, 1, true, NULL, NULL, NULL, false`},
		{"lunar day 31", `'FATHER', 'lunar', NULL, 31, 1, false, NULL, NULL, NULL, false`},
		{"month 13", `'FATHER', 'lunar', NULL, 1, 13, false, NULL, NULL, NULL, false`},
		{"reminder days 31", `'FATHER', 'solar', '2015-03-04', NULL, NULL, NULL, 31, '07:30', 'UTC', false`},
		{"reminder half set", `'FATHER', 'solar', '2015-03-04', NULL, NULL, NULL, 3, NULL, NULL, false`},
		{"OS reminder without reminder", `'FATHER', 'solar', '2015-03-04', NULL, NULL, NULL, NULL, NULL, NULL, true`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(ctx, fmt.Sprintf(base, tt.sql), uid)
			assert.Error(t, err)
		})
	}
}

func TestMemorialRepository_DeletingUserDeletesMemorials(t *testing.T) {
	repo, db := newMemorialRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	_, err := repo.Create(ctx, fullDateMemorial(uid, "A"), 100)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid)
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM memorials`).Scan(&n))
	assert.Zero(t, n, "ON DELETE CASCADE")
}
