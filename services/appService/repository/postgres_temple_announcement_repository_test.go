package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var annNow = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func newAnnouncementRepos(t *testing.T) (domain.TempleAnnouncementRepository, domain.AnnouncementPreferenceRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewTempleAnnouncementRepository(db), NewAnnouncementPreferenceRepository(db), db
}

func insertIssuingBody(t *testing.T, db database.Database, id, templeID, name string) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO temple_issuing_bodies (id, temple_id, name) VALUES ($1, $2, $3)`, id, templeID, name)
	require.NoError(t, err)
}

// insertAnnouncement inserts one row; created_at is irrelevant, the window is explicit.
func insertAnnouncement(t *testing.T, db database.Database, id, templeID, bodyID string, published, expires time.Time, revoked *time.Time) {
	t.Helper()
	_, err := db.Exec(context.Background(), `
		INSERT INTO temple_announcements (id, temple_id, issuing_body_id, title, body, published_at, expires_at, revoked_at)
		VALUES ($1, $2, $3, $4, 'nội dung', $5, $6, $7)`,
		id, templeID, bodyID, "Tiêu đề "+id, published, expires, revoked)
	require.NoError(t, err)
}

func annIDs(items []domain.TempleAnnouncement) []string {
	out := make([]string, len(items))
	for i, a := range items {
		out[i] = a.ID
	}
	return out
}

func TestTempleAnnouncementRepository_ListActiveByTemple(t *testing.T) {
	repo, _, db := newAnnouncementRepos(t)
	ctx := context.Background()
	insertTemple(t, db, "ta")
	insertTemple(t, db, "tb")
	insertIssuingBody(t, db, "tib_a", "ta", "Ban Trị sự")
	insertIssuingBody(t, db, "tib_b", "tb", "Ban B")
	day := 24 * time.Hour
	revoked := annNow.Add(-time.Hour)

	insertAnnouncement(t, db, "tann_old", "ta", "tib_a", annNow.Add(-3*day), annNow.Add(5*day), nil)
	insertAnnouncement(t, db, "tann_new", "ta", "tib_a", annNow.Add(-1*day), annNow.Add(5*day), nil)
	insertAnnouncement(t, db, "tann_tie_b", "ta", "tib_a", annNow.Add(-2*day), annNow.Add(5*day), nil)
	insertAnnouncement(t, db, "tann_tie_a", "ta", "tib_a", annNow.Add(-2*day), annNow.Add(5*day), nil)
	insertAnnouncement(t, db, "tann_expired", "ta", "tib_a", annNow.Add(-3*day), annNow.Add(-time.Second), nil)
	insertAnnouncement(t, db, "tann_edge", "ta", "tib_a", annNow.Add(-3*day), annNow, nil) // expires_at == now: not active
	insertAnnouncement(t, db, "tann_revoked", "ta", "tib_a", annNow.Add(-3*day), annNow.Add(5*day), &revoked)
	insertAnnouncement(t, db, "tann_future", "ta", "tib_a", annNow.Add(time.Hour), annNow.Add(5*day), nil)
	insertAnnouncement(t, db, "tann_other", "tb", "tib_b", annNow.Add(-1*day), annNow.Add(5*day), nil)

	t.Run("only the temple's active rows, newest first then id", func(t *testing.T) {
		got, err := repo.ListActiveByTemple(ctx, "ta", annNow, domain.AnnouncementPageRequest{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, []string{"tann_new", "tann_tie_a", "tann_tie_b", "tann_old"}, annIDs(got.Items))
		assert.Equal(t, 4, got.Total)
		assert.Equal(t, domain.IssuingBodyRef{ID: "tib_a", Name: "Ban Trị sự"}, got.Items[0].IssuingBody)
		assert.Equal(t, "ta", got.Items[0].TempleID)
	})

	t.Run("paging is stable and total counts all active", func(t *testing.T) {
		p1, err := repo.ListActiveByTemple(ctx, "ta", annNow, domain.AnnouncementPageRequest{Page: 1, PageSize: 3})
		require.NoError(t, err)
		p2, err := repo.ListActiveByTemple(ctx, "ta", annNow, domain.AnnouncementPageRequest{Page: 2, PageSize: 3})
		require.NoError(t, err)
		assert.Equal(t, []string{"tann_new", "tann_tie_a", "tann_tie_b"}, annIDs(p1.Items))
		assert.Equal(t, []string{"tann_old"}, annIDs(p2.Items))
		assert.Equal(t, 4, p2.Total)
	})

	t.Run("page past the end is empty, not nil", func(t *testing.T) {
		got, err := repo.ListActiveByTemple(ctx, "ta", annNow, domain.AnnouncementPageRequest{Page: 9, PageSize: 10})
		require.NoError(t, err)
		assert.NotNil(t, got.Items)
		assert.Empty(t, got.Items)
		assert.Equal(t, 4, got.Total)
	})

	t.Run("a temple without announcements is empty", func(t *testing.T) {
		insertTemple(t, db, "tc")
		got, err := repo.ListActiveByTemple(ctx, "tc", annNow, domain.AnnouncementPageRequest{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Empty(t, got.Items)
		assert.Zero(t, got.Total)
	})

	t.Run("published_at in the future appears once now reaches it", func(t *testing.T) {
		got, err := repo.ListActiveByTemple(ctx, "ta", annNow.Add(time.Hour), domain.AnnouncementPageRequest{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Contains(t, annIDs(got.Items), "tann_future")
	})
}

func TestAnnouncementPreferenceRepository(t *testing.T) {
	_, prefs, db := newAnnouncementRepos(t)
	ctx := context.Background()
	insertTemple(t, db, "ta")
	uid := insertUser(t, db, "0900000001", "a@example.com")

	t.Run("default is on with no temple (TA12)", func(t *testing.T) {
		s, err := prefs.GetSettings(ctx, uid)
		require.NoError(t, err)
		assert.True(t, s.Receive)
		assert.Nil(t, s.Temple)
	})

	t.Run("set off, then the temple joins in; changing temple keeps the switch (TA15)", func(t *testing.T) {
		require.NoError(t, prefs.SetReceive(ctx, uid, false))
		_, err := db.Exec(ctx, `UPDATE users SET ordination_temple_id = 'ta' WHERE id = $1`, uid)
		require.NoError(t, err)
		s, err := prefs.GetSettings(ctx, uid)
		require.NoError(t, err)
		assert.False(t, s.Receive)
		require.NotNil(t, s.Temple)
		assert.Equal(t, domain.TempleRef{ID: "ta", Name: "Chùa ta"}, *s.Temple)
	})

	t.Run("unknown user", func(t *testing.T) {
		_, err := prefs.GetSettings(ctx, 987654)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.ErrorIs(t, prefs.SetReceive(ctx, 987654, true), domain.ErrUserNotFound)
	})
}
