package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type annEnv struct {
	db     database.Database
	repo   domain.AnnouncementRepository
	bodies domain.IssuingBodyRepository
}

func newAnnEnv(t *testing.T) *annEnv {
	t.Helper()
	db := connectTestDB(t) // truncates temples CASCADE, which clears both new tables
	return &annEnv{db: db, repo: NewTempleAnnouncementRepository(db), bodies: NewIssuingBodyRepository(db)}
}

func (e *annEnv) temple(t *testing.T, id string) {
	t.Helper()
	_, err := e.db.Exec(context.Background(), `
		INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority,
			latitude, longitude, address)
		VALUES ($1, 'Chùa', 'CHUA', 'south', 'Bắc tông', 1990, 'DI_TICH_CAP_TINH', 1, 10.7, 106.6, 'x')`, id)
	require.NoError(t, err)
}

func TestIssuingBodyRepository(t *testing.T) {
	e := newAnnEnv(t)
	ctx := context.Background()
	e.temple(t, "temple_south_001")
	e.temple(t, "temple_south_002")

	b, err := e.bodies.Insert(ctx, "temple_south_001", "Ban Trị sự")
	require.NoError(t, err)
	assert.Contains(t, b.ID, "tib_")
	assert.False(t, b.CreatedAt.IsZero())

	t.Run("same name in another case is a duplicate", func(t *testing.T) {
		_, err := e.bodies.Insert(ctx, "temple_south_001", "BAN TRỊ SỰ")
		assert.ErrorIs(t, err, domain.ErrIssuingBodyExists)
	})
	t.Run("same name in another temple is fine", func(t *testing.T) {
		_, err := e.bodies.Insert(ctx, "temple_south_002", "Ban Trị sự")
		assert.NoError(t, err)
	})
	t.Run("unknown temple", func(t *testing.T) {
		_, err := e.bodies.Insert(ctx, "temple_nope", "Ban")
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})
	t.Run("list is per temple, ordered by name", func(t *testing.T) {
		_, err := e.bodies.Insert(ctx, "temple_south_001", "Ban Hoằng pháp")
		require.NoError(t, err)
		got, err := e.bodies.ListByTemple(ctx, "temple_south_001")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "Ban Hoằng pháp", got[0].Name)
		empty, err := e.bodies.ListByTemple(ctx, "temple_none")
		require.NoError(t, err)
		assert.Empty(t, empty)
	})
}

func TestAnnouncementRepository(t *testing.T) {
	e := newAnnEnv(t)
	ctx := context.Background()
	e.temple(t, "temple_south_001")
	e.temple(t, "temple_south_002")
	body, err := e.bodies.Insert(ctx, "temple_south_001", "Ban Trị sự")
	require.NoError(t, err)
	other, err := e.bodies.Insert(ctx, "temple_south_002", "Ban Khác")
	require.NoError(t, err)

	base := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	admin := "7"
	newAnn := func(title string, publishedAt time.Time) domain.Announcement {
		return domain.Announcement{
			TempleID: "temple_south_001", IssuingBodyID: body.ID, Title: title, Body: "Nội dung",
			Source: domain.AnnouncementSourceAdmin, PublishedAt: publishedAt, ExpiresAt: publishedAt.Add(24 * time.Hour), CreatedBy: &admin,
		}
	}

	t.Run("insert and find round trip with body name", func(t *testing.T) {
		got, err := e.repo.Insert(ctx, newAnn("Lễ Vu Lan", base))
		require.NoError(t, err)
		assert.Contains(t, got.ID, "tann_")
		assert.Equal(t, "Ban Trị sự", got.IssuingBodyName)
		assert.Equal(t, "7", *got.CreatedBy)
		assert.Nil(t, got.RevokedAt)
		assert.True(t, base.Equal(got.PublishedAt))
		found, err := e.repo.FindByID(ctx, got.ID)
		require.NoError(t, err)
		assert.Equal(t, got.ID, found.ID)
	})

	t.Run("find unknown", func(t *testing.T) {
		_, err := e.repo.FindByID(ctx, "tann_nope")
		assert.ErrorIs(t, err, domain.ErrAnnouncementNotFound)
	})

	t.Run("CHECK expires_at > published_at is enforced", func(t *testing.T) {
		a := newAnn("Sai hạn", base)
		a.ExpiresAt = base
		_, err := e.repo.Insert(ctx, a)
		require.Error(t, err)
		assert.NotErrorIs(t, err, domain.ErrTempleNotFound)
	})

	t.Run("insert for a missing temple maps to ErrTempleNotFound", func(t *testing.T) {
		a := newAnn("Mồ côi", base)
		a.TempleID = "temple_gone"
		_, err := e.repo.Insert(ctx, a)
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})

	t.Run("exists and belongs", func(t *testing.T) {
		ok, err := e.repo.TempleExists(ctx, "temple_south_001")
		require.NoError(t, err)
		assert.True(t, ok)
		ok, err = e.repo.TempleExists(ctx, "temple_nope")
		require.NoError(t, err)
		assert.False(t, ok)
		ok, err = e.repo.IssuingBodyBelongs(ctx, body.ID, "temple_south_001")
		require.NoError(t, err)
		assert.True(t, ok)
		ok, err = e.repo.IssuingBodyBelongs(ctx, other.ID, "temple_south_001")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("update changes editable fields only", func(t *testing.T) {
		a, err := e.repo.Insert(ctx, newAnn("Cũ", base.Add(time.Hour)))
		require.NoError(t, err)
		a.Title, a.Body = "Mới", "Mới nữa"
		a.ExpiresAt = a.ExpiresAt.Add(time.Hour)
		a.PublishedAt = base.Add(99 * time.Hour) // must be ignored
		got, err := e.repo.Update(ctx, *a)
		require.NoError(t, err)
		assert.Equal(t, "Mới", got.Title)
		assert.True(t, got.ExpiresAt.Equal(a.ExpiresAt))
		assert.True(t, got.PublishedAt.Equal(base.Add(time.Hour)))
		_, err = e.repo.Update(ctx, domain.Announcement{ID: "tann_nope", IssuingBodyID: body.ID, Title: "x", Body: "x", ExpiresAt: base})
		assert.ErrorIs(t, err, domain.ErrAnnouncementNotFound)
	})

	t.Run("revoke is idempotent and keeps the first revoked_at", func(t *testing.T) {
		a, err := e.repo.Insert(ctx, newAnn("Thu hồi", base.Add(2*time.Hour)))
		require.NoError(t, err)
		first, err := e.repo.Revoke(ctx, a.ID, base)
		require.NoError(t, err)
		require.NotNil(t, first.RevokedAt)
		second, err := e.repo.Revoke(ctx, a.ID, base.Add(time.Hour))
		require.NoError(t, err)
		assert.True(t, first.RevokedAt.Equal(*second.RevokedAt))
		_, err = e.repo.Revoke(ctx, "tann_nope", base)
		assert.ErrorIs(t, err, domain.ErrAnnouncementNotFound)
	})

	t.Run("list is stable: published_at DESC then id, paged, any state", func(t *testing.T) {
		e2 := newAnnEnv(t)
		e2.temple(t, "temple_south_001")
		b, err := e2.bodies.Insert(ctx, "temple_south_001", "Ban")
		require.NoError(t, err)
		same := base.Add(10 * time.Hour)
		mk := func(title string, at time.Time) domain.Announcement {
			a := newAnn(title, at)
			a.IssuingBodyID = b.ID
			return a
		}
		var ids []string
		for _, a := range []domain.Announcement{mk("old", base), mk("same1", same), mk("same2", same), mk("new", same.Add(time.Hour))} {
			got, err := e2.repo.Insert(ctx, a)
			require.NoError(t, err)
			ids = append(ids, got.ID)
		}
		_, err = e2.repo.Revoke(ctx, ids[0], base) // revoked ones are still listed
		require.NoError(t, err)

		p1, total, err := e2.repo.ListByTemple(ctx, "temple_south_001", 0, 2)
		require.NoError(t, err)
		p2, _, err := e2.repo.ListByTemple(ctx, "temple_south_001", 2, 2)
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		require.Len(t, p1, 2)
		require.Len(t, p2, 2)
		assert.Equal(t, "new", p1[0].Title)
		first, second := p1[1], p2[0] // the two "same" rows, split across pages
		assert.True(t, first.PublishedAt.Equal(second.PublishedAt))
		assert.Less(t, first.ID, second.ID, "id breaks the tie")
		assert.Equal(t, "old", p2[1].Title)
		assert.NotNil(t, p2[1].RevokedAt)

		past, _, err := e2.repo.ListByTemple(ctx, "temple_south_001", 100, 2)
		require.NoError(t, err)
		assert.Empty(t, past)
	})

	t.Run("deleting the temple cascades to its announcements and bodies", func(t *testing.T) {
		e3 := newAnnEnv(t)
		e3.temple(t, "temple_south_001")
		b, err := e3.bodies.Insert(ctx, "temple_south_001", "Ban")
		require.NoError(t, err)
		a := newAnn("x", base)
		a.IssuingBodyID = b.ID
		got, err := e3.repo.Insert(ctx, a)
		require.NoError(t, err)
		_, err = e3.db.Exec(ctx, `DELETE FROM temples WHERE id = 'temple_south_001'`)
		require.NoError(t, err)
		_, err = e3.repo.FindByID(ctx, got.ID)
		assert.ErrorIs(t, err, domain.ErrAnnouncementNotFound)
		bs, err := e3.bodies.ListByTemple(ctx, "temple_south_001")
		require.NoError(t, err)
		assert.Empty(t, bs)
	})
}
