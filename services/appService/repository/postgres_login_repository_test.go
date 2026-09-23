package repository

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

const testSessionTTL = 168 * time.Hour

// uniqueTokenIssuer mints a distinct pair per call — like pkg/jwt does with its
// jti — which the shared stubTokenIssuer does not: it returns the same refresh
// token for a user every time, so it cannot exercise rotation or two sessions of
// one user.
func uniqueTokenIssuer(userID, sessionID string) (string, string, error) {
	return "access-" + sessionID, "refresh-" + uuid.NewString(), nil
}

func testDevice(model string) domain.DeviceInfo {
	return domain.DeviceInfo{
		Platform:    "ios",
		DeviceModel: model,
		OSVersion:   "17.4",
		AppVersion:  "1.0.0",
		DeviceName:  "phone",
		IPAddress:   "10.0.0.1",
		UserAgent:   "innerlamp/1.0",
	}
}

func loginTestUser(t *testing.T, db database.Database, phone string) *domain.User {
	t.Helper()
	id := insertUser(t, db, phone, phone+"@example.com")
	return &domain.User{ID: id, PhoneNumber: phone, Email: phone + "@example.com"}
}

// countActiveSessions counts unrevoked sessions bound to any device of userID.
func countActiveSessions(t *testing.T, db database.Database, userID int64) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(context.Background(),
		`SELECT count(*) FROM user_sessions WHERE user_id = $1 AND is_revoked = false AND device_id IS NOT NULL`, userID).Scan(&n))
	return n
}

func TestLoginRepository_FindUser(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("by phone found", func(t *testing.T) {
		truncateAll(t, db)
		id := insertUser(t, db, "+84900000101", "a@example.com")

		u, err := repo.FindUserByPhone(ctx, "+84900000101")
		require.NoError(t, err)
		assert.Equal(t, id, u.ID)
		assert.Equal(t, "a@example.com", u.Email)
		assert.Equal(t, "x", u.PasswordHash)
		assert.Nil(t, u.EmailVerifiedAt)
		assert.Nil(t, u.DharmaName)
	})

	t.Run("by phone missing is ErrUserNotFound", func(t *testing.T) {
		truncateAll(t, db)
		_, err := repo.FindUserByPhone(ctx, "+84900000199")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("phone only in pending_registration is ErrUserNotFound (AC2)", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000102", "p@example.com", time.Now().Add(time.Hour))
		_, err := repo.FindUserByPhone(ctx, "+84900000102")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("by id", func(t *testing.T) {
		truncateAll(t, db)
		id := insertUser(t, db, "+84900000103", "b@example.com")
		u, err := repo.FindUserByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "+84900000103", u.PhoneNumber)

		_, err = repo.FindUserByID(ctx, id+1000)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestLoginRepository_FindLoginAttempt(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("no row is nil,nil", func(t *testing.T) {
		truncateAll(t, db)
		a, err := repo.FindLoginAttempt(ctx, "+84900000201")
		require.NoError(t, err)
		assert.Nil(t, a)
	})

	t.Run("row read back", func(t *testing.T) {
		truncateAll(t, db)
		_, err := repo.RecordLoginFailure(ctx, "+84900000202", 5, 15*time.Minute, 15*time.Minute)
		require.NoError(t, err)

		a, err := repo.FindLoginAttempt(ctx, "+84900000202")
		require.NoError(t, err)
		require.NotNil(t, a)
		assert.Equal(t, 1, a.FailedCount)
		assert.Nil(t, a.LockedUntil)
	})
}

func TestLoginRepository_RecordLoginFailure(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()
	const phone = "+84900000301"

	t.Run("counts up and locks on the fifth failure", func(t *testing.T) {
		truncateAll(t, db)
		for i := 1; i <= 4; i++ {
			a, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
			require.NoError(t, err)
			assert.Equal(t, i, a.FailedCount)
			assert.Nil(t, a.LockedUntil, "failure %d must not lock", i)
		}

		a, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 5, a.FailedCount)
		require.NotNil(t, a.LockedUntil)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), *a.LockedUntil, 10*time.Second)
	})

	t.Run("failure older than the reset window restarts at 1", func(t *testing.T) {
		truncateAll(t, db)
		for range 3 {
			_, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
			require.NoError(t, err)
		}
		_, err := db.Exec(ctx, `UPDATE login_attempts SET last_failed_at = now() - interval '16 minutes' WHERE phone_number = $1`, phone)
		require.NoError(t, err)

		a, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 1, a.FailedCount)
		assert.Nil(t, a.LockedUntil)
	})

	t.Run("failure inside the reset window keeps counting", func(t *testing.T) {
		truncateAll(t, db)
		for range 3 {
			_, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
			require.NoError(t, err)
		}
		_, err := db.Exec(ctx, `UPDATE login_attempts SET last_failed_at = now() - interval '14 minutes' WHERE phone_number = $1`, phone)
		require.NoError(t, err)

		a, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 4, a.FailedCount)
	})

	t.Run("expired lock restarts at 1", func(t *testing.T) {
		truncateAll(t, db)
		for range 5 {
			_, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, time.Hour)
			require.NoError(t, err)
		}
		_, err := db.Exec(ctx, `UPDATE login_attempts SET locked_until = now() - interval '1 second' WHERE phone_number = $1`, phone)
		require.NoError(t, err)

		a, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, a.FailedCount)
		assert.Nil(t, a.LockedUntil)
	})

	t.Run("concurrent failures never lose a count and lock at the threshold", func(t *testing.T) {
		truncateAll(t, db)
		const workers = 12
		var wg sync.WaitGroup
		errs := make(chan error, workers)
		for range workers {
			wg.Go(func() {
				_, err := repo.RecordLoginFailure(ctx, phone, 5, 15*time.Minute, 15*time.Minute)
				errs <- err
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		a, err := repo.FindLoginAttempt(ctx, phone)
		require.NoError(t, err)
		assert.Equal(t, workers, a.FailedCount, "every concurrent failure must be counted exactly once")
		assert.NotNil(t, a.LockedUntil)
	})
}

func TestLoginRepository_CreateLoginSession(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("creates device and session, returns tokens (AC1)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000401")

		res, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)
		assert.Same(t, user, res.User)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEmpty(t, res.RefreshToken)

		var (
			sessionID, hash string
			version         int
			expiresAt       time.Time
			deviceCount     int
		)
		require.NoError(t, db.QueryRow(ctx,
			`SELECT session_id, refresh_token_hash, token_version, expires_at FROM user_sessions WHERE user_id = $1`, user.ID).
			Scan(&sessionID, &hash, &version, &expiresAt))
		assert.Equal(t, "access-"+sessionID, res.AccessToken, "access token must carry the stored session ID")
		assert.Equal(t, domain.HashRefreshToken(res.RefreshToken), hash, "only the hash is stored")
		assert.Equal(t, 1, version)
		assert.WithinDuration(t, time.Now().Add(testSessionTTL), expiresAt, 10*time.Second)

		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_devices WHERE user_id = $1`, user.ID).Scan(&deviceCount))
		assert.Equal(t, 1, deviceCount)
	})

	t.Run("optional device fields stored as NULL", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000402")
		_, err := repo.CreateLoginSession(ctx, user, domain.DeviceInfo{Platform: "android", DeviceModel: "Pixel"}, testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		var osVersion, deviceName, ip *string
		require.NoError(t, db.QueryRow(ctx, `SELECT os_version, device_name, ip_address FROM user_devices WHERE user_id = $1`, user.ID).Scan(&osVersion, &deviceName, &ip))
		assert.Nil(t, osVersion)
		assert.Nil(t, deviceName)
		assert.Nil(t, ip)
	})

	t.Run("same device again revokes only its previous session (AC4)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000403")

		first, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)
		second, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		assert.Equal(t, 1, countActiveSessions(t, db, user.ID))

		var reason string
		require.NoError(t, db.QueryRow(ctx,
			`SELECT revoked_reason FROM user_sessions WHERE refresh_token_hash = $1 AND is_revoked = true`, domain.HashRefreshToken(first.RefreshToken)).Scan(&reason))
		assert.Equal(t, domain.RevokeReasonSupersededByLogin, reason)

		var live bool
		require.NoError(t, db.QueryRow(ctx,
			`SELECT is_revoked = false AND is_active FROM user_sessions WHERE refresh_token_hash = $1`, domain.HashRefreshToken(second.RefreshToken)).Scan(&live))
		assert.True(t, live)

		var deviceCount int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_devices WHERE user_id = $1`, user.ID).Scan(&deviceCount))
		assert.Equal(t, 1, deviceCount, "the same fingerprint upserts one device row")
	})

	t.Run("a new device leaves other devices' sessions alone (AC3)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000404")

		_, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)
		_, err = repo.CreateLoginSession(ctx, user, testDevice("Pixel 8"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		assert.Equal(t, 2, countActiveSessions(t, db, user.ID))
	})

	t.Run("US-001 device-less session is not touched", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000405")
		_, err := db.Exec(ctx,
			`INSERT INTO user_sessions (user_id, session_id, refresh_token_hash, expires_at) VALUES ($1, 'legacy', 'legacyhash', now() + interval '1 day')`, user.ID)
		require.NoError(t, err)

		_, err = repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		var revoked bool
		require.NoError(t, db.QueryRow(ctx, `SELECT is_revoked FROM user_sessions WHERE session_id = 'legacy'`).Scan(&revoked))
		assert.False(t, revoked)
	})

	t.Run("clears the brute-force counter (AC7)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000406")
		for range 3 {
			_, err := repo.RecordLoginFailure(ctx, user.PhoneNumber, 5, 15*time.Minute, 15*time.Minute)
			require.NoError(t, err)
		}

		_, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		a, err := repo.FindLoginAttempt(ctx, user.PhoneNumber)
		require.NoError(t, err)
		assert.Nil(t, a)
	})

	t.Run("issuer failure rolls back everything", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000407")
		_, err := repo.RecordLoginFailure(ctx, user.PhoneNumber, 5, 15*time.Minute, 15*time.Minute)
		require.NoError(t, err)

		boom := errors.New("mint failed")
		_, err = repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL,
			func(string, string) (string, string, error) { return "", "", boom })
		require.ErrorIs(t, err, boom)

		var devices, sessions int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_devices`).Scan(&devices))
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_sessions`).Scan(&sessions))
		assert.Zero(t, devices)
		assert.Zero(t, sessions)

		a, err := repo.FindLoginAttempt(ctx, user.PhoneNumber)
		require.NoError(t, err)
		assert.NotNil(t, a, "a rolled-back login must not have cleared the counter")
	})

	t.Run("issuer receives the numeric user id and the stored session id", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000408")
		var gotUser, gotSID string
		res, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL,
			func(userID, sessionID string) (string, string, error) {
				gotUser, gotSID = userID, sessionID
				return "a", "r-" + sessionID, nil
			})
		require.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(user.ID, 10), gotUser)
		_, err = uuid.Parse(gotSID)
		assert.NoError(t, err)
		assert.Equal(t, "r-"+gotSID, res.RefreshToken)
	})

	t.Run("concurrent logins from one device leave exactly one active session", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000409")

		const workers = 8
		var wg sync.WaitGroup
		errs := make(chan error, workers)
		for range workers {
			wg.Go(func() {
				_, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
				errs <- err
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		assert.Equal(t, 1, countActiveSessions(t, db, user.ID))
		var total int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_sessions WHERE user_id = $1`, user.ID).Scan(&total))
		assert.Equal(t, workers, total)
	})
}

// seedSession logs user in on a device and returns the result plus the stored
// session row ID, for the refresh tests.
func seedSession(t *testing.T, db database.Database, repo domain.LoginRepository, user *domain.User) (*domain.LoginResult, int64) {
	t.Helper()
	res, err := repo.CreateLoginSession(context.Background(), user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
	require.NoError(t, err)
	var rowID int64
	require.NoError(t, db.QueryRow(context.Background(),
		`SELECT id FROM user_sessions WHERE refresh_token_hash = $1`, domain.HashRefreshToken(res.RefreshToken)).Scan(&rowID))
	return res, rowID
}

func TestLoginRepository_FindSessionByRefreshHash(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("current token matches, not flagged as previous", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000501")
		res, rowID := seedSession(t, db, repo, user)

		s, prev, err := repo.FindSessionByRefreshHash(ctx, domain.HashRefreshToken(res.RefreshToken))
		require.NoError(t, err)
		assert.False(t, prev)
		assert.Equal(t, rowID, s.ID)
		assert.Equal(t, user.ID, s.UserID)
		require.NotNil(t, s.DeviceID)
		assert.Equal(t, 1, s.TokenVersion)
	})

	t.Run("device-less US-001 session is found with nil DeviceID (AC12)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000502")
		_, err := db.Exec(ctx,
			`INSERT INTO user_sessions (user_id, session_id, refresh_token_hash, expires_at) VALUES ($1, 'legacy', 'legacyhash', now() + interval '1 day')`, user.ID)
		require.NoError(t, err)

		s, _, err := repo.FindSessionByRefreshHash(ctx, "legacyhash")
		require.NoError(t, err)
		assert.Nil(t, s.DeviceID)
		assert.Equal(t, "legacy", s.SessionID)
	})

	t.Run("already-rotated token is flagged as previous (AC11)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000503")
		res, rowID := seedSession(t, db, repo, user)
		oldHash := domain.HashRefreshToken(res.RefreshToken)
		_, err := repo.RotateSession(ctx, rowID, oldHash, testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		s, prev, err := repo.FindSessionByRefreshHash(ctx, oldHash)
		require.NoError(t, err)
		assert.True(t, prev)
		assert.Equal(t, rowID, s.ID)
	})

	t.Run("unknown, revoked and expired are all ErrRefreshTokenInvalid (AC10)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000504")
		res, rowID := seedSession(t, db, repo, user)
		hash := domain.HashRefreshToken(res.RefreshToken)

		_, _, err := repo.FindSessionByRefreshHash(ctx, "nope")
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)

		_, err = db.Exec(ctx, `UPDATE user_sessions SET expires_at = now() - interval '1 second' WHERE id = $1`, rowID)
		require.NoError(t, err)
		_, _, err = repo.FindSessionByRefreshHash(ctx, hash)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)

		_, err = db.Exec(ctx, `UPDATE user_sessions SET expires_at = now() + interval '1 day' WHERE id = $1`, rowID)
		require.NoError(t, err)
		require.NoError(t, repo.RevokeSession(ctx, user.ID, sessionIDOf(t, db, rowID), domain.RevokeReasonLogout))
		_, _, err = repo.FindSessionByRefreshHash(ctx, hash)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
	})
}

func sessionIDOf(t *testing.T, db database.Database, rowID int64) string {
	t.Helper()
	var sid string
	require.NoError(t, db.QueryRow(context.Background(), `SELECT session_id FROM user_sessions WHERE id = $1`, rowID).Scan(&sid))
	return sid
}

func TestLoginRepository_RotateSession(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("rotates hash, keeps previous, bumps version, slides expiry (AC9)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000601")
		res, rowID := seedSession(t, db, repo, user)
		oldHash := domain.HashRefreshToken(res.RefreshToken)
		// Age the session so a slide is observable.
		_, err := db.Exec(ctx, `UPDATE user_sessions SET expires_at = now() + interval '1 day' WHERE id = $1`, rowID)
		require.NoError(t, err)

		out, err := repo.RotateSession(ctx, rowID, oldHash, testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)
		assert.NotEqual(t, res.RefreshToken, out.RefreshToken)
		assert.Equal(t, "access-"+sessionIDOf(t, db, rowID), out.AccessToken, "same sid as the session")

		var (
			hash, prev *string
			version    int
			expiresAt  time.Time
		)
		require.NoError(t, db.QueryRow(ctx,
			`SELECT refresh_token_hash, previous_refresh_token_hash, token_version, expires_at FROM user_sessions WHERE id = $1`, rowID).
			Scan(&hash, &prev, &version, &expiresAt))
		assert.Equal(t, domain.HashRefreshToken(out.RefreshToken), *hash)
		assert.Equal(t, oldHash, *prev)
		assert.Equal(t, 2, version)
		assert.WithinDuration(t, time.Now().Add(testSessionTTL), expiresAt, 10*time.Second, "expiry slides forward")
	})

	t.Run("stale hash is ErrRefreshTokenInvalid and changes nothing", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000602")
		res, rowID := seedSession(t, db, repo, user)

		_, err := repo.RotateSession(ctx, rowID, "not-the-current-hash", testSessionTTL, uniqueTokenIssuer)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)

		var hash string
		require.NoError(t, db.QueryRow(ctx, `SELECT refresh_token_hash FROM user_sessions WHERE id = $1`, rowID).Scan(&hash))
		assert.Equal(t, domain.HashRefreshToken(res.RefreshToken), hash)
	})

	t.Run("revoked session is ErrRefreshTokenInvalid", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000603")
		res, rowID := seedSession(t, db, repo, user)
		require.NoError(t, repo.RevokeSession(ctx, user.ID, sessionIDOf(t, db, rowID), domain.RevokeReasonLogout))

		_, err := repo.RotateSession(ctx, rowID, domain.HashRefreshToken(res.RefreshToken), testSessionTTL, uniqueTokenIssuer)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
	})

	t.Run("issuer failure rolls back", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000604")
		res, rowID := seedSession(t, db, repo, user)

		boom := errors.New("mint failed")
		_, err := repo.RotateSession(ctx, rowID, domain.HashRefreshToken(res.RefreshToken), testSessionTTL,
			func(string, string) (string, string, error) { return "", "", boom })
		require.ErrorIs(t, err, boom)

		var version int
		require.NoError(t, db.QueryRow(ctx, `SELECT token_version FROM user_sessions WHERE id = $1`, rowID).Scan(&version))
		assert.Equal(t, 1, version)
	})

	t.Run("two concurrent rotations of one token: exactly one wins", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000605")
		res, rowID := seedSession(t, db, repo, user)
		oldHash := domain.HashRefreshToken(res.RefreshToken)

		const workers = 6
		var (
			wg      sync.WaitGroup
			winners atomic.Int32
			losers  atomic.Int32
		)
		for range workers {
			wg.Go(func() {
				_, err := repo.RotateSession(ctx, rowID, oldHash, testSessionTTL, uniqueTokenIssuer)
				switch {
				case err == nil:
					winners.Add(1)
				case errors.Is(err, domain.ErrRefreshTokenInvalid):
					losers.Add(1)
				default:
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
		wg.Wait()

		assert.EqualValues(t, 1, winners.Load())
		assert.EqualValues(t, workers-1, losers.Load())

		var version int
		require.NoError(t, db.QueryRow(ctx, `SELECT token_version FROM user_sessions WHERE id = $1`, rowID).Scan(&version))
		assert.Equal(t, 2, version)
	})
}

func TestLoginRepository_RevokeSession(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("revokes with reason (AC13)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000701")
		_, rowID := seedSession(t, db, repo, user)
		sid := sessionIDOf(t, db, rowID)

		require.NoError(t, repo.RevokeSession(ctx, user.ID, sid, domain.RevokeReasonLogout))

		var (
			revoked, active bool
			reason          string
			revokedAt       *time.Time
		)
		require.NoError(t, db.QueryRow(ctx, `SELECT is_revoked, is_active, revoked_reason, revoked_at FROM user_sessions WHERE id = $1`, rowID).
			Scan(&revoked, &active, &reason, &revokedAt))
		assert.True(t, revoked)
		assert.False(t, active)
		assert.Equal(t, domain.RevokeReasonLogout, reason)
		assert.NotNil(t, revokedAt)
	})

	t.Run("idempotent: second call and unknown session succeed and change nothing (AC14)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000702")
		_, rowID := seedSession(t, db, repo, user)
		sid := sessionIDOf(t, db, rowID)

		require.NoError(t, repo.RevokeSession(ctx, user.ID, sid, domain.RevokeReasonLogout))
		require.NoError(t, repo.RevokeSession(ctx, user.ID, sid, "second-call-reason"))
		require.NoError(t, repo.RevokeSession(ctx, user.ID, "no-such-session", domain.RevokeReasonLogout))

		var reason string
		require.NoError(t, db.QueryRow(ctx, `SELECT revoked_reason FROM user_sessions WHERE id = $1`, rowID).Scan(&reason))
		assert.Equal(t, domain.RevokeReasonLogout, reason, "the original reason must survive a retry")
	})

	t.Run("cannot revoke another user's session (AC14)", func(t *testing.T) {
		truncateAll(t, db)
		owner := loginTestUser(t, db, "+84900000703")
		attacker := loginTestUser(t, db, "+84900000704")
		_, rowID := seedSession(t, db, repo, owner)
		sid := sessionIDOf(t, db, rowID)

		require.NoError(t, repo.RevokeSession(ctx, attacker.ID, sid, domain.RevokeReasonLogout))

		var revoked bool
		require.NoError(t, db.QueryRow(ctx, `SELECT is_revoked FROM user_sessions WHERE id = $1`, rowID).Scan(&revoked))
		assert.False(t, revoked)
	})
}
