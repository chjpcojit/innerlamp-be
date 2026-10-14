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

func newProfileRepo(t *testing.T) (domain.ProfileRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewProfileRepository(db), db
}

func opt(v string) domain.OptionalString { return domain.OptionalString{Set: true, Value: &v} }
func clear() domain.OptionalString       { return domain.OptionalString{Set: true} }

// rawUser reads the columns the profile touches straight from the table, so a
// test can tell SQL NULL from an empty string, which the entity would hide.
type rawUser struct {
	dharmaName, templeID, avatar *string
	updatedAt                    time.Time
}

func readRawUser(t *testing.T, db database.Database, id int64) rawUser {
	t.Helper()
	var r rawUser
	err := db.QueryRow(context.Background(),
		`SELECT dharma_name, ordination_temple_id, avatar_object, updated_at FROM users WHERE id = $1`, id,
	).Scan(&r.dharmaName, &r.templeID, &r.avatar, &r.updatedAt)
	require.NoError(t, err)
	return r
}

func setUserColumns(t *testing.T, db database.Database, id int64, dharma, temple, avatar *string) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`UPDATE users SET dharma_name = $2, ordination_temple_id = $3, avatar_object = $4 WHERE id = $1`,
		id, dharma, temple, avatar)
	require.NoError(t, err)
}

func TestProfileRepository_Get_JoinsTempleAndReportsEmailState(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	insertTemple(t, db, "temple_a")
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setUserColumns(t, db, uid, strp("Thanh Liên"), strp("temple_a"), strp("avatars/1/x.jpg"))

	got, err := repo.Get(ctx, uid)

	require.NoError(t, err)
	assert.Equal(t, uid, got.UserID)
	assert.Equal(t, "a@example.com", got.Email)
	assert.Equal(t, "0900000001", got.PhoneNumber)
	assert.False(t, got.EmailVerified, "email_verified_at is NULL until an email flow exists")
	require.NotNil(t, got.DharmaName)
	assert.Equal(t, "Thanh Liên", *got.DharmaName)
	require.NotNil(t, got.OrdinationTemple)
	assert.Equal(t, domain.TempleRef{ID: "temple_a", Name: "Chùa temple_a"}, *got.OrdinationTemple)
	require.NotNil(t, got.AvatarObject)
	assert.Equal(t, "avatars/1/x.jpg", *got.AvatarObject)

	_, err = db.Exec(ctx, `UPDATE users SET email_verified_at = now() WHERE id = $1`, uid)
	require.NoError(t, err)
	got, err = repo.Get(ctx, uid)
	require.NoError(t, err)
	assert.True(t, got.EmailVerified)
}

func TestProfileRepository_Get_EmptyOptionals(t *testing.T) {
	repo, db := newProfileRepo(t)
	uid := insertUser(t, db, "0900000001", "a@example.com")

	got, err := repo.Get(context.Background(), uid)

	require.NoError(t, err)
	assert.Nil(t, got.DharmaName)
	assert.Nil(t, got.OrdinationTemple)
	assert.Nil(t, got.AvatarObject)
}

func TestProfileRepository_Get_UnknownUser(t *testing.T) {
	repo, _ := newProfileRepo(t)

	_, err := repo.Get(context.Background(), 987654)

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestProfileRepository_Update_OnlyTouchesColumnsInThePatch(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	insertTemple(t, db, "temple_a")
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setUserColumns(t, db, uid, strp("Cũ"), strp("temple_a"), strp("avatars/1/x.jpg"))

	got, err := repo.Update(ctx, uid, domain.ProfilePatch{DharmaName: opt("Mới")})

	require.NoError(t, err)
	require.NotNil(t, got.DharmaName)
	assert.Equal(t, "Mới", *got.DharmaName)
	require.NotNil(t, got.OrdinationTemple, "the temple was absent from the patch and must be kept")
	assert.Equal(t, "temple_a", got.OrdinationTemple.ID)
	raw := readRawUser(t, db, uid)
	assert.Equal(t, "avatars/1/x.jpg", *raw.avatar, "the avatar is never touched by Update")
}

func TestProfileRepository_Update_ClearWritesNullNotEmptyString(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	insertTemple(t, db, "temple_a")
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setUserColumns(t, db, uid, strp("Cũ"), strp("temple_a"), nil)

	got, err := repo.Update(ctx, uid, domain.ProfilePatch{DharmaName: clear(), OrdinationTempleID: clear()})

	require.NoError(t, err)
	assert.Nil(t, got.DharmaName)
	assert.Nil(t, got.OrdinationTemple)
	raw := readRawUser(t, db, uid)
	assert.Nil(t, raw.dharmaName, "must be SQL NULL, not ''")
	assert.Nil(t, raw.templeID)
}

func TestProfileRepository_Update_SetsBothFieldsAndBumpsUpdatedAt(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	insertTemple(t, db, "temple_b")
	uid := insertUser(t, db, "0900000001", "a@example.com")
	before := readRawUser(t, db, uid).updatedAt

	got, err := repo.Update(ctx, uid, domain.ProfilePatch{DharmaName: opt("Thanh Liên"), OrdinationTempleID: opt("temple_b")})

	require.NoError(t, err)
	assert.Equal(t, "Thanh Liên", *got.DharmaName)
	assert.Equal(t, domain.TempleRef{ID: "temple_b", Name: "Chùa temple_b"}, *got.OrdinationTemple)
	assert.True(t, readRawUser(t, db, uid).updatedAt.After(before))
}

func TestProfileRepository_Update_UnknownTempleIsInvalidTempleAndChangesNothing(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setUserColumns(t, db, uid, strp("Cũ"), nil, nil)

	// A temple deleted between the use case's check and this write looks exactly
	// like this: the FK rejects it, and the dharma name in the same statement must
	// not be applied (AC14).
	_, err := repo.Update(ctx, uid, domain.ProfilePatch{DharmaName: opt("Mới"), OrdinationTempleID: opt("temple_gone")})

	assert.ErrorIs(t, err, domain.ErrInvalidTemple)
	raw := readRawUser(t, db, uid)
	assert.Equal(t, "Cũ", *raw.dharmaName)
}

func TestProfileRepository_Update_EmptyPatchChangesNothing(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setUserColumns(t, db, uid, strp("Cũ"), nil, nil)
	before := readRawUser(t, db, uid)

	got, err := repo.Update(ctx, uid, domain.ProfilePatch{})

	require.NoError(t, err)
	assert.Equal(t, "Cũ", *got.DharmaName)
	assert.Equal(t, before.updatedAt, readRawUser(t, db, uid).updatedAt)
}

func TestProfileRepository_Update_UnknownUser(t *testing.T) {
	repo, _ := newProfileRepo(t)

	_, err := repo.Update(context.Background(), 987654, domain.ProfilePatch{DharmaName: opt("x")})

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestProfileRepository_SwapAvatar_ReturnsTheKeyItReplaced(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	old, err := repo.SwapAvatar(ctx, uid, "avatars/1/first.jpg")
	require.NoError(t, err)
	assert.Nil(t, old, "no avatar before the first swap")

	old, err = repo.SwapAvatar(ctx, uid, "avatars/1/second.jpg")
	require.NoError(t, err)
	require.NotNil(t, old)
	assert.Equal(t, "avatars/1/first.jpg", *old)
	assert.Equal(t, "avatars/1/second.jpg", *readRawUser(t, db, uid).avatar)
}

func TestProfileRepository_SwapAvatar_ConcurrentSwapsEachGetTheirOwnPredecessor(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	const n = 8

	type result struct {
		key string
		old *string
		err error
	}
	results := make([]result, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("avatars/1/k%d.jpg", i)
			old, err := repo.SwapAvatar(ctx, uid, key)
			results[i] = result{key: key, old: old, err: err}
		}()
	}
	wg.Wait()

	// Every key but the winner's final one was replaced exactly once, and only
	// one swap saw "no avatar". If two swaps read the same predecessor, one image
	// would be deleted twice and another never — leaking it.
	final := *readRawUser(t, db, uid).avatar
	replaced := map[string]int{}
	nils := 0
	for _, r := range results {
		require.NoError(t, r.err)
		if r.old == nil {
			nils++
			continue
		}
		replaced[*r.old]++
	}
	assert.Equal(t, 1, nils)
	for _, r := range results {
		if r.key == final {
			assert.Zero(t, replaced[r.key], "the surviving key was never replaced")
		} else {
			assert.Equal(t, 1, replaced[r.key], "key %s must be handed back exactly once", r.key)
		}
	}
}

func TestProfileRepository_SwapAvatar_UnknownUser(t *testing.T) {
	repo, _ := newProfileRepo(t)

	_, err := repo.SwapAvatar(context.Background(), 987654, "avatars/9/x.jpg")

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestProfileRepository_ClearAvatar(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")

	t.Run("no avatar writes nothing", func(t *testing.T) {
		before := readRawUser(t, db, uid).updatedAt
		old, err := repo.ClearAvatar(ctx, uid)
		require.NoError(t, err)
		assert.Nil(t, old)
		assert.Equal(t, before, readRawUser(t, db, uid).updatedAt)
	})

	t.Run("with avatar returns the key and sets NULL", func(t *testing.T) {
		setUserColumns(t, db, uid, nil, nil, strp("avatars/1/x.jpg"))
		old, err := repo.ClearAvatar(ctx, uid)
		require.NoError(t, err)
		require.NotNil(t, old)
		assert.Equal(t, "avatars/1/x.jpg", *old)
		assert.Nil(t, readRawUser(t, db, uid).avatar)
	})

	t.Run("unknown user", func(t *testing.T) {
		_, err := repo.ClearAvatar(ctx, 987654)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestProfileRepository_TempleExists(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	insertTemple(t, db, "temple_a")

	ok, err := repo.TempleExists(ctx, "temple_a")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = repo.TempleExists(ctx, "temple_missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestProfileRepository_Update_FullNameRoundTrip(t *testing.T) {
	repo, db := newProfileRepo(t)
	ctx := context.Background()
	uid := insertUser(t, db, "0900000001", "a@example.com")
	setUserColumns(t, db, uid, strp("Pháp danh"), nil, nil)

	got, err := repo.Update(ctx, uid, domain.ProfilePatch{FullName: opt("Nguyễn Văn An")})

	require.NoError(t, err)
	require.NotNil(t, got.FullName)
	assert.Equal(t, "Nguyễn Văn An", *got.FullName)
	require.NotNil(t, got.DharmaName, "the dharma name was absent from the patch and must be kept")
	again, err := repo.Get(ctx, uid)
	require.NoError(t, err)
	require.NotNil(t, again.FullName)
	assert.Equal(t, "Nguyễn Văn An", *again.FullName)
}
