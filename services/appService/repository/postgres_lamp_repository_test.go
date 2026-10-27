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

var lampTestNow = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

const (
	lampTestWindow = 5 * time.Second
	lampTestMax    = 20
)

func newLampRepo(t *testing.T) (domain.LampRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewLampRepository(db), db
}

func setDharmaName(t *testing.T, db database.Database, userID int64, name string) {
	t.Helper()
	_, err := db.Exec(context.Background(), `UPDATE users SET dharma_name = $2 WHERE id = $1`, userID, name)
	require.NoError(t, err)
}

// lampAtTime builds the lamp the use case would hand over for a user at a given instant.
func lampAtTime(userID int64, recipient string, at time.Time, days int) domain.Lamp {
	return domain.Lamp{
		UserID: userID, LampType: domain.LampTypeOilLamp, Intention: domain.IntentionPeace,
		Recipient: &recipient, DurationDays: days,
		LitAt: at, EndAt: at.Add(time.Duration(days) * 24 * time.Hour),
	}
}

func lightParams(l domain.Lamp, max int) domain.LightLampParams {
	return domain.LightLampParams{Lamp: l, MaxActive: max, DedupeWindow: lampTestWindow}
}

func mustLight(t *testing.T, repo domain.LampRepository, l domain.Lamp) *domain.Lamp {
	t.Helper()
	got, created, err := repo.Light(context.Background(), lightParams(l, 1000))
	require.NoError(t, err)
	require.True(t, created)
	return got
}

func TestLampRepository_Light_StoresLampAndCopiesDharmaName(t *testing.T) {
	repo, db := newLampRepo(t)
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setDharmaName(t, db, uid, "Thích Minh An")

	prayer := "Mong bà bình an"
	in := lampAtTime(uid, "Bà ngoại", lampTestNow, 7)
	in.LampType = domain.LampTypeLotusCandle
	in.Intention = domain.IntentionSalvation
	in.Prayer = &prayer

	got, created, err := repo.Light(context.Background(), lightParams(in, lampTestMax))

	require.NoError(t, err)
	assert.True(t, created)
	assert.NotZero(t, got.ID)
	assert.Equal(t, uid, got.UserID)
	assert.Equal(t, domain.LampTypeLotusCandle, got.LampType)
	assert.Equal(t, domain.IntentionSalvation, got.Intention)
	require.NotNil(t, got.Recipient)
	assert.Equal(t, "Bà ngoại", *got.Recipient)
	require.NotNil(t, got.Prayer)
	assert.Equal(t, prayer, *got.Prayer)
	assert.Equal(t, 7, got.DurationDays)
	assert.False(t, got.IsAnonymous)
	require.NotNil(t, got.LighterName)
	assert.Equal(t, "Thích Minh An", *got.LighterName)
	assert.True(t, got.LitAt.Equal(lampTestNow), "lit_at round-trips: %s", got.LitAt)
	assert.True(t, got.EndAt.Equal(lampTestNow.Add(7*24*time.Hour)), "end_at round-trips: %s", got.EndAt)
}

func TestLampRepository_Light_NameRules(t *testing.T) {
	repo, db := newLampRepo(t)
	withName := insertUser(t, db, "0900000001", "a@example.com")
	setDharmaName(t, db, withName, "Thích Minh An")
	noName := insertUser(t, db, "0900000002", "b@example.com")

	t.Run("anonymous lamp keeps no name even though the user has one", func(t *testing.T) {
		in := lampAtTime(withName, "r", lampTestNow, 1)
		in.IsAnonymous = true
		got := mustLight(t, repo, in)
		assert.True(t, got.IsAnonymous)
		assert.Nil(t, got.LighterName)
	})

	t.Run("user without a dharma name gets no name", func(t *testing.T) {
		got := mustLight(t, repo, lampAtTime(noName, "r", lampTestNow, 1))
		assert.False(t, got.IsAnonymous)
		assert.Nil(t, got.LighterName)
		assert.Nil(t, got.Prayer, "no prayer round-trips as nil")
	})

	t.Run("a later dharma name change does not touch lamps already lit", func(t *testing.T) {
		old := mustLight(t, repo, lampAtTime(withName, "before", lampTestNow.Add(time.Minute), 1))
		require.NotNil(t, old.LighterName)

		setDharmaName(t, db, withName, "Thích Tuệ Minh")
		fresh := mustLight(t, repo, lampAtTime(withName, "after", lampTestNow.Add(2*time.Minute), 1))

		page, err := repo.Garden(context.Background(), domain.GardenQuery{Limit: 50, Now: lampTestNow})
		require.NoError(t, err)
		names := map[string]string{}
		for _, it := range page.Items {
			if it.LighterName != nil {
				names[*it.Recipient] = *it.LighterName
			}
		}
		assert.Equal(t, "Thích Minh An", names["before"], "lamp %d keeps the name it was lit with", old.ID)
		assert.Equal(t, "Thích Tuệ Minh", names["after"], "lamp %d uses the new name", fresh.ID)
	})
}

func TestLampRepository_Light_DoubleTap(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	first := mustLight(t, repo, lampAtTime(uid, "Bà ngoại", lampTestNow, 7))

	t.Run("identical lamp inside the window returns the first one and writes nothing", func(t *testing.T) {
		again, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, "Bà ngoại", lampTestNow.Add(3*time.Second), 7), lampTestMax))
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, first.ID, again.ID)
		assert.Equal(t, 1, countRowsIn(t, db, "lamps"))
	})

	t.Run("identical lamp after the window is a new lamp", func(t *testing.T) {
		again, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, "Bà ngoại", lampTestNow.Add(6*time.Second), 7), lampTestMax))
		require.NoError(t, err)
		assert.True(t, created)
		assert.NotEqual(t, first.ID, again.ID)
	})

	t.Run("any different field inside the window is a new lamp", func(t *testing.T) {
		at := lampTestNow.Add(20 * time.Second)
		base := lampAtTime(uid, "Ông nội", at, 7)
		_, created, err := repo.Light(ctx, lightParams(base, lampTestMax))
		require.NoError(t, err)
		require.True(t, created)

		variants := map[string]func(l *domain.Lamp){
			"recipient": func(l *domain.Lamp) { r := "Ông ngoại"; l.Recipient = &r },
			"type":      func(l *domain.Lamp) { l.LampType = domain.LampTypeFlowerLantern },
			"intention": func(l *domain.Lamp) { l.Intention = domain.IntentionGratitude },
			"duration":  func(l *domain.Lamp) { l.DurationDays = 21; l.EndAt = at.Add(time.Second + 21*24*time.Hour) },
			"anonymous": func(l *domain.Lamp) { l.IsAnonymous = true },
			"prayer":    func(l *domain.Lamp) { p := "x"; l.Prayer = &p },
		}
		for name, mutate := range variants {
			l := base
			l.LitAt = at.Add(time.Second)
			mutate(&l)
			_, created, err := repo.Light(ctx, lightParams(l, lampTestMax))
			require.NoError(t, err, name)
			assert.True(t, created, "%s differs, so it is not a double tap", name)
		}
	})

	t.Run("a null prayer matches a null prayer", func(t *testing.T) {
		other := insertUser(t, db, "0900000002", "b@example.com")
		a := mustLight(t, repo, lampAtTime(other, "r", lampTestNow, 1))
		b, created, err := repo.Light(ctx, lightParams(lampAtTime(other, "r", lampTestNow.Add(time.Second), 1), lampTestMax))
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, a.ID, b.ID)
	})

	t.Run("another user's identical lamp is not a double tap", func(t *testing.T) {
		other := insertUser(t, db, "0900000003", "c@example.com")
		_, created, err := repo.Light(ctx, lightParams(lampAtTime(other, "Bà ngoại", lampTestNow.Add(time.Second), 7), lampTestMax))
		require.NoError(t, err)
		assert.True(t, created)
	})
}

func TestLampRepository_Light_Cap(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	for i := 0; i < 3; i++ {
		_, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, fmt.Sprintf("r%d", i), lampTestNow, 7), 3))
		require.NoError(t, err)
		require.True(t, created)
	}

	t.Run("at the cap a new lamp is refused", func(t *testing.T) {
		got, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, "r-over", lampTestNow, 7), 3))
		require.ErrorIs(t, err, domain.ErrLampLimitReached)
		assert.Nil(t, got)
		assert.False(t, created)
		assert.Equal(t, 3, countRowsIn(t, db, "lamps"))
	})

	t.Run("a double tap at the cap still returns the lamp, not a refusal", func(t *testing.T) {
		got, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, "r2", lampTestNow.Add(time.Second), 7), 3))
		require.NoError(t, err)
		assert.False(t, created)
		require.NotNil(t, got.Recipient)
		assert.Equal(t, "r2", *got.Recipient)
	})

	t.Run("another user is unaffected", func(t *testing.T) {
		other := insertUser(t, db, "0900000002", "b@example.com")
		_, created, err := repo.Light(ctx, lightParams(lampAtTime(other, "r", lampTestNow, 7), 3))
		require.NoError(t, err)
		assert.True(t, created)
	})

	t.Run("expired lamps free capacity", func(t *testing.T) {
		// Two of the three lamps go out one day after they were lit.
		_, err := db.Exec(ctx, `UPDATE lamps SET lit_at = $2, end_at = $3 WHERE user_id = $1 AND recipient IN ('r0','r1')`,
			uid, lampTestNow.Add(-48*time.Hour), lampTestNow.Add(-24*time.Hour))
		require.NoError(t, err)

		_, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, "r-new", lampTestNow.Add(time.Minute), 1), 3))
		require.NoError(t, err)
		assert.True(t, created)
	})
}

func TestLampRepository_Light_ConcurrentNeverExceedsCap(t *testing.T) {
	repo, db := newLampRepo(t)
	uid := insertUser(t, db, "0900000001", "a@example.com")

	const attempts, max = 12, 3
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		created int
		refused int
		other   []error
	)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Distinct recipients so none is mistaken for a double tap.
			_, ok, err := repo.Light(context.Background(),
				lightParams(lampAtTime(uid, fmt.Sprintf("r%d", i), lampTestNow, 7), max))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && ok:
				created++
			case err != nil && err == domain.ErrLampLimitReached:
				refused++
			default:
				other = append(other, err)
			}
		}(i)
	}
	wg.Wait()

	require.Empty(t, other)
	assert.Equal(t, max, created)
	assert.Equal(t, attempts-max, refused)
	assert.Equal(t, max, countRowsIn(t, db, "lamps"))
}

func TestLampRepository_Light_ConcurrentDoubleTapWritesOnce(t *testing.T) {
	repo, db := newLampRepo(t)
	uid := insertUser(t, db, "0900000001", "a@example.com")

	const taps = 8
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		created int
		ids     = map[int64]bool{}
	)
	for i := 0; i < taps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, ok, err := repo.Light(context.Background(), lightParams(lampAtTime(uid, "same", lampTestNow, 7), lampTestMax))
			if !assert.NoError(t, err) {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if ok {
				created++
			}
			ids[got.ID] = true
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, created, "exactly one tap writes")
	assert.Len(t, ids, 1, "every tap sees the same lamp")
	assert.Equal(t, 1, countRowsIn(t, db, "lamps"))
}

func TestLampRepository_Light_ExpiredLampsDoNotBlockDoubleTapLookup(t *testing.T) {
	repo, db := newLampRepo(t)
	uid := insertUser(t, db, "0900000001", "a@example.com")
	old := mustLight(t, repo, lampAtTime(uid, "same", lampTestNow.Add(-time.Hour), 1))

	got, created, err := repo.Light(context.Background(), lightParams(lampAtTime(uid, "same", lampTestNow, 1), lampTestMax))

	require.NoError(t, err)
	assert.True(t, created)
	assert.NotEqual(t, old.ID, got.ID)
}

// ---- Garden ----

func lightAs(t *testing.T, repo domain.LampRepository, uid int64, recipient string, typ domain.LampType, litAt time.Time, days int) *domain.Lamp {
	t.Helper()
	l := lampAtTime(uid, recipient, litAt, days)
	l.LampType = typ
	return mustLight(t, repo, l)
}

func gardenIDs(page *domain.GardenPage) []int64 {
	ids := make([]int64, 0, len(page.Items))
	for _, it := range page.Items {
		ids = append(ids, it.ID)
	}
	return ids
}

func TestLampRepository_Garden_AllUsersNewestFirstWithoutExpired(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	a := insertUser(t, db, "0900000001", "a@example.com")
	b := insertUser(t, db, "0900000002", "b@example.com")
	setDharmaName(t, db, a, "Thích Minh An")

	l1 := lightAs(t, repo, a, "oldest", domain.LampTypeOilLamp, lampTestNow.Add(-3*time.Hour), 7)
	l2 := lightAs(t, repo, b, "middle", domain.LampTypeLotusCandle, lampTestNow.Add(-2*time.Hour), 7)
	l3 := lightAs(t, repo, a, "newest", domain.LampTypeFlowerLantern, lampTestNow.Add(-time.Hour), 7)
	lightAs(t, repo, b, "expired", domain.LampTypeOilLamp, lampTestNow.Add(-48*time.Hour), 1) // went out 24h ago

	page, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow})

	require.NoError(t, err)
	assert.Equal(t, []int64{l3.ID, l2.ID, l1.ID}, gardenIDs(page), "every user's lit lamps, newest first, expired gone")
	assert.Equal(t, 3, page.Total)
	assert.Equal(t, map[domain.LampType]int{
		domain.LampTypeOilLamp: 1, domain.LampTypeLotusCandle: 1, domain.LampTypeFlowerLantern: 1,
	}, page.Counts)
	require.NotNil(t, page.Items[0].LighterName)
	assert.Equal(t, "Thích Minh An", *page.Items[0].LighterName)
	assert.Nil(t, page.Items[1].LighterName)
}

func TestLampRepository_Garden_EmptyAndExactBoundary(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	empty, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow})
	require.NoError(t, err)
	assert.Empty(t, empty.Items)
	assert.Equal(t, 0, empty.Total)

	// end_at == now means the lamp has just gone out: not lit.
	lightAs(t, repo, uid, "boundary", domain.LampTypeOilLamp, lampTestNow.Add(-24*time.Hour), 1)
	atEnd, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow})
	require.NoError(t, err)
	assert.Empty(t, atEnd.Items, "end_at == now is expired")

	justBefore, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow.Add(-time.Microsecond)})
	require.NoError(t, err)
	assert.Len(t, justBefore.Items, 1, "one microsecond earlier it is still lit")
}

func TestLampRepository_Garden_FilterNeverChangesCounts(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	for i := 0; i < 5; i++ {
		lightAs(t, repo, uid, fmt.Sprintf("o%d", i), domain.LampTypeOilLamp, lampTestNow.Add(-time.Duration(i+1)*time.Minute), 7)
	}
	for i := 0; i < 3; i++ {
		lightAs(t, repo, uid, fmt.Sprintf("l%d", i), domain.LampTypeLotusCandle, lampTestNow.Add(-time.Duration(i+10)*time.Minute), 7)
	}
	lotus := domain.LampTypeLotusCandle

	page, err := repo.Garden(ctx, domain.GardenQuery{LampType: &lotus, Limit: 12, Now: lampTestNow})

	require.NoError(t, err)
	assert.Len(t, page.Items, 3)
	for _, it := range page.Items {
		assert.Equal(t, domain.LampTypeLotusCandle, it.LampType)
	}
	assert.Equal(t, 8, page.Total, "total ignores the filter")
	assert.Equal(t, 5, page.Counts[domain.LampTypeOilLamp])
	assert.Equal(t, 3, page.Counts[domain.LampTypeLotusCandle])

	flower := domain.LampTypeFlowerLantern
	none, err := repo.Garden(ctx, domain.GardenQuery{LampType: &flower, Limit: 12, Now: lampTestNow})
	require.NoError(t, err)
	assert.Empty(t, none.Items)
	assert.Equal(t, 8, none.Total, "an empty filtered list still reports every count")
}

func TestLampRepository_Garden_KeysetPagingIsStableWhileLampsChange(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	// Ten lamps, lit one minute apart: ids[0] is the newest.
	ids := make([]int64, 10)
	for i := range ids {
		l := lightAs(t, repo, uid, fmt.Sprintf("r%d", i), domain.LampTypeOilLamp, lampTestNow.Add(-time.Duration(i+1)*time.Minute), 7)
		ids[i] = l.ID
	}

	// Page 1 asks for 4: the repository returns 5 (limit+1).
	p1, err := repo.Garden(ctx, domain.GardenQuery{Limit: 4, Now: lampTestNow})
	require.NoError(t, err)
	require.Equal(t, ids[:5], gardenIDs(p1))
	last := p1.Items[3]
	after := &domain.Cursor{LitAt: last.LitAt, ID: last.ID}

	// Between the two loads: a brand-new lamp is lit on top, and a lamp that
	// belongs to page 2 expires.
	lightAs(t, repo, uid, "fresh", domain.LampTypeOilLamp, lampTestNow.Add(-time.Second), 7)
	_, err = db.Exec(ctx, `UPDATE lamps SET end_at = $2 WHERE id = $1`, ids[5], lampTestNow.Add(-time.Second))
	require.NoError(t, err)

	p2, err := repo.Garden(ctx, domain.GardenQuery{Limit: 4, After: after, Now: lampTestNow})
	require.NoError(t, err)

	assert.Equal(t, []int64{ids[4], ids[6], ids[7], ids[8], ids[9]}, gardenIDs(p2),
		"continues right after the last lamp shown: nothing repeated, nothing skipped, the expired one gone, the fresh one not inserted")
}

func TestLampRepository_Garden_CursorBreaksTiesByID(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	// Three lamps lit at the very same instant: only id orders them.
	a := lightAs(t, repo, uid, "a", domain.LampTypeOilLamp, lampTestNow.Add(-time.Minute), 7)
	b := lightAs(t, repo, uid, "b", domain.LampTypeOilLamp, lampTestNow.Add(-time.Minute), 7)
	c := lightAs(t, repo, uid, "c", domain.LampTypeOilLamp, lampTestNow.Add(-time.Minute), 7)

	all, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow})
	require.NoError(t, err)
	require.Equal(t, []int64{c.ID, b.ID, a.ID}, gardenIDs(all))

	next, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, After: &domain.Cursor{LitAt: b.LitAt, ID: b.ID}, Now: lampTestNow})
	require.NoError(t, err)
	assert.Equal(t, []int64{a.ID}, gardenIDs(next))
}

func TestLampRepository_Schema_AnonymousLampCannotCarryAName(t *testing.T) {
	_, db := newLampRepo(t)
	uid := insertUser(t, db, "0900000001", "a@example.com")

	_, err := db.Exec(context.Background(), `
		INSERT INTO lamps (user_id, lamp_type, intention, recipient, duration_days, is_anonymous, lighter_name, lit_at, end_at)
		VALUES ($1, 'OIL_LAMP', 'PEACE', 'r', 1, true, 'Thích Minh An', $2, $3)`,
		uid, lampTestNow, lampTestNow.Add(24*time.Hour))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chk_lamps_anonymous_no_name")
}

func countRowsIn(t *testing.T, db database.Database, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&n))
	return n
}

// ---- v2.0: optional recipient and is_mine ----

// noRecipientLamp is lampAtTime without a recipient (dedicated to all beings).
// VI: noRecipientLamp là lampAtTime nhưng không có người nhận (dành cho tất cả chúng sanh).
func noRecipientLamp(userID int64, at time.Time, days int) domain.Lamp {
	l := lampAtTime(userID, "", at, days)
	l.Recipient = nil
	return l
}

func TestLampRepository_Light_WithoutRecipientStoresNull(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	got := mustLight(t, repo, noRecipientLamp(uid, lampTestNow, 7))

	assert.Nil(t, got.Recipient)
	var isNull bool
	require.NoError(t, db.QueryRow(ctx, `SELECT recipient IS NULL FROM lamps WHERE id = $1`, got.ID).Scan(&isNull))
	assert.True(t, isNull, "no recipient is stored as NULL, never ''")

	page, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow, ViewerID: uid})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Nil(t, page.Items[0].Recipient)
}

// Two identical no-recipient requests inside the window are one lamp: NULL must
// compare equal to NULL (IS NOT DISTINCT FROM), or a double tap would light two.
// VI: Hai request giống hệt nhau, không có người nhận, trong cửa sổ bấm đúp chỉ tạo
// một đèn: NULL phải được coi bằng NULL, nếu không bấm đúp sẽ thắp hai đèn.
func TestLampRepository_Light_DoubleTapWithoutRecipient(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	first := mustLight(t, repo, noRecipientLamp(uid, lampTestNow, 7))
	again, created, err := repo.Light(ctx, lightParams(noRecipientLamp(uid, lampTestNow.Add(2*time.Second), 7), lampTestMax))

	require.NoError(t, err)
	assert.False(t, created, "a double tap without a recipient is not a new lamp")
	assert.Equal(t, first.ID, again.ID)
	assert.Equal(t, 1, countRowsIn(t, db, "lamps"))

	t.Run("a lamp with a recipient is not a duplicate of one without", func(t *testing.T) {
		withRecipient, created, err := repo.Light(ctx, lightParams(lampAtTime(uid, "Bà ngoại", lampTestNow.Add(3*time.Second), 7), lampTestMax))
		require.NoError(t, err)
		assert.True(t, created)
		assert.NotEqual(t, first.ID, withRecipient.ID)

		noneAgain, created, err := repo.Light(ctx, lightParams(noRecipientLamp(uid, lampTestNow.Add(4*time.Second), 7), lampTestMax))
		require.NoError(t, err)
		assert.False(t, created, "still the first no-recipient lamp")
		assert.Equal(t, first.ID, noneAgain.ID)
		assert.Equal(t, 2, countRowsIn(t, db, "lamps"))
	})
}

func TestLampRepository_Garden_IsMine(t *testing.T) {
	repo, db := newLampRepo(t)
	ctx := context.Background()
	me := insertUser(t, db, "0900000001", "a@example.com")
	other := insertUser(t, db, "0900000002", "b@example.com")
	setDharmaName(t, db, me, "Thích Minh An")

	mine := lightAs(t, repo, me, "mine", domain.LampTypeOilLamp, lampTestNow.Add(-3*time.Minute), 7)
	theirs := lightAs(t, repo, other, "theirs", domain.LampTypeLotusCandle, lampTestNow.Add(-2*time.Minute), 7)
	anon := lampAtTime(me, "mine anonymous", lampTestNow.Add(-time.Minute), 7)
	anon.IsAnonymous = true
	mineAnon := mustLight(t, repo, anon)

	isMine := func(page *domain.GardenPage) map[int64]bool {
		out := make(map[int64]bool, len(page.Items))
		for _, it := range page.Items {
			out[it.ID] = it.IsMine
		}
		return out
	}

	page, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow, ViewerID: me})
	require.NoError(t, err)
	assert.Equal(t, map[int64]bool{mine.ID: true, theirs.ID: false, mineAnon.ID: true}, isMine(page))
	for _, it := range page.Items {
		if it.ID == mineAnon.ID {
			assert.Nil(t, it.LighterName, "my own anonymous lamp still carries no name")
		}
	}

	asOther, err := repo.Garden(ctx, domain.GardenQuery{Limit: 12, Now: lampTestNow, ViewerID: other})
	require.NoError(t, err)
	assert.Equal(t, map[int64]bool{mine.ID: false, theirs.ID: true, mineAnon.ID: false}, isMine(asOther))

	t.Run("unchanged by lamp_type, limit and cursor", func(t *testing.T) {
		oil := domain.LampTypeOilLamp
		filtered, err := repo.Garden(ctx, domain.GardenQuery{LampType: &oil, Limit: 12, Now: lampTestNow, ViewerID: me})
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{mine.ID: true, mineAnon.ID: true}, isMine(filtered))

		p1, err := repo.Garden(ctx, domain.GardenQuery{Limit: 1, Now: lampTestNow, ViewerID: me})
		require.NoError(t, err)
		first := p1.Items[0]
		assert.True(t, first.IsMine)
		p2, err := repo.Garden(ctx, domain.GardenQuery{Limit: 1, After: &domain.Cursor{LitAt: first.LitAt, ID: first.ID}, Now: lampTestNow, ViewerID: me})
		require.NoError(t, err)
		assert.Equal(t, theirs.ID, p2.Items[0].ID)
		assert.False(t, p2.Items[0].IsMine)
	})
}
