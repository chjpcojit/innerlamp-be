package repository

import (
	"context"
	"database/sql"
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

// findLoginAttempt reads the counter row for email; nil when there is none.
func findLoginAttempt(t *testing.T, db database.Database, email string) *domain.LoginAttempt {
	t.Helper()
	var (
		a           = domain.LoginAttempt{Email: email}
		lockedUntil *time.Time
	)
	err := db.QueryRow(context.Background(),
		`SELECT failed_count, locked_until, last_failed_at FROM login_attempts WHERE email = $1`, email).
		Scan(&a.FailedCount, &lockedUntil, &a.LastFailedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	require.NoError(t, err)
	a.LockedUntil = lockedUntil
	return &a
}

func TestLoginRepository_FindUser(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	t.Run("by email found", func(t *testing.T) {
		truncateAll(t, db)
		id := insertUser(t, db, "+84900000101", "a@example.com")

		u, err := repo.FindUserByEmail(ctx, "a@example.com")
		require.NoError(t, err)
		assert.Equal(t, id, u.ID)
		assert.Equal(t, "+84900000101", u.PhoneNumber)
		assert.Equal(t, "x", u.PasswordHash)
		assert.Nil(t, u.EmailVerifiedAt, "an unverified account is returned; the use case refuses it")
		assert.Nil(t, u.DharmaName)
	})

	t.Run("by email missing is ErrUserNotFound", func(t *testing.T) {
		truncateAll(t, db)
		_, err := repo.FindUserByEmail(ctx, "nobody@example.com")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("email only in pending_registration is ErrUserNotFound (AC2)", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000102", "p@example.com", time.Now().Add(time.Hour))
		_, err := repo.FindUserByEmail(ctx, "p@example.com")
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

func TestLoginRepository_ReserveLoginAttempt(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()
	const email = "lock@example.com"

	reserve := func(t *testing.T, resetWindow time.Duration) *domain.LoginAttempt {
		t.Helper()
		a, err := repo.ReserveLoginAttempt(ctx, email, 5, 15*time.Minute, resetWindow)
		require.NoError(t, err)
		return a
	}

	t.Run("counts up and locks on the fifth attempt", func(t *testing.T) {
		truncateAll(t, db)
		for i := 1; i <= 4; i++ {
			a := reserve(t, 15*time.Minute)
			assert.Equal(t, i, a.FailedCount)
			assert.Nil(t, a.LockedUntil, "attempt %d must not lock", i)
			assert.False(t, a.AlreadyLocked)
		}

		a := reserve(t, 15*time.Minute)
		assert.Equal(t, 5, a.FailedCount)
		require.NotNil(t, a.LockedUntil)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), *a.LockedUntil, 10*time.Second)
		assert.False(t, a.AlreadyLocked, "the attempt that trips the lock is still counted and compared")
	})

	t.Run("while locked nothing changes and AlreadyLocked is set", func(t *testing.T) {
		truncateAll(t, db)
		for range 5 {
			reserve(t, 15*time.Minute)
		}
		before := findLoginAttempt(t, db, email)

		a := reserve(t, 15*time.Minute)
		assert.True(t, a.AlreadyLocked)
		assert.Equal(t, 5, a.FailedCount)
		require.NotNil(t, a.LockedUntil)
		assert.True(t, before.LockedUntil.Equal(*a.LockedUntil), "the lock deadline must not move")

		after := findLoginAttempt(t, db, email)
		assert.Equal(t, before.FailedCount, after.FailedCount)
		assert.True(t, before.LastFailedAt.Equal(after.LastFailedAt))
	})

	t.Run("attempt older than the reset window restarts at 1", func(t *testing.T) {
		truncateAll(t, db)
		for range 3 {
			reserve(t, 15*time.Minute)
		}
		_, err := db.Exec(ctx, `UPDATE login_attempts SET last_failed_at = now() - interval '16 minutes' WHERE email = $1`, email)
		require.NoError(t, err)

		a := reserve(t, 15*time.Minute)
		assert.Equal(t, 1, a.FailedCount)
		assert.Nil(t, a.LockedUntil)
	})

	t.Run("attempt inside the reset window keeps counting", func(t *testing.T) {
		truncateAll(t, db)
		for range 3 {
			reserve(t, 15*time.Minute)
		}
		_, err := db.Exec(ctx, `UPDATE login_attempts SET last_failed_at = now() - interval '14 minutes' WHERE email = $1`, email)
		require.NoError(t, err)

		a := reserve(t, 15*time.Minute)
		assert.Equal(t, 4, a.FailedCount)
	})

	t.Run("expired lock restarts at 1", func(t *testing.T) {
		truncateAll(t, db)
		for range 5 {
			reserve(t, time.Hour)
		}
		_, err := db.Exec(ctx, `UPDATE login_attempts SET locked_until = now() - interval '1 second' WHERE email = $1`, email)
		require.NoError(t, err)

		a := reserve(t, time.Hour)
		assert.Equal(t, 1, a.FailedCount)
		assert.Nil(t, a.LockedUntil)
		assert.False(t, a.AlreadyLocked)
	})

	// AC5: ten concurrent wrong-password requests get at most five password
	// comparisons — the reservation is what the use case gates the comparison
	// on, so "not AlreadyLocked" is exactly "compared".
	t.Run("10 concurrent attempts admit at most 5 comparisons (AC5)", func(t *testing.T) {
		truncateAll(t, db)
		const workers = 10
		var (
			wg       sync.WaitGroup
			admitted atomic.Int32
			errs     = make(chan error, workers)
		)
		for range workers {
			wg.Go(func() {
				a, err := repo.ReserveLoginAttempt(ctx, email, 5, 15*time.Minute, 15*time.Minute)
				if err == nil && !a.AlreadyLocked {
					admitted.Add(1)
				}
				errs <- err
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		assert.Equal(t, int32(5), admitted.Load())
		a := findLoginAttempt(t, db, email)
		assert.Equal(t, 5, a.FailedCount)
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

	t.Run("clears the brute-force counter (AC5)", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000406")
		for range 3 {
			_, err := repo.ReserveLoginAttempt(ctx, user.Email, 5, 15*time.Minute, 15*time.Minute)
			require.NoError(t, err)
		}

		_, err := repo.CreateLoginSession(ctx, user, testDevice("iPhone 15"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)

		assert.Nil(t, findLoginAttempt(t, db, user.Email))
	})

	t.Run("issuer failure rolls back everything", func(t *testing.T) {
		truncateAll(t, db)
		user := loginTestUser(t, db, "+84900000407")
		_, err := repo.ReserveLoginAttempt(ctx, user.Email, 5, 15*time.Minute, 15*time.Minute)
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

		assert.NotNil(t, findLoginAttempt(t, db, user.Email), "a rolled-back login must not have cleared the counter")
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

func TestLoginRepository_IsSessionActive(t *testing.T) {
	db := connectTestDB(t)
	repo := NewLoginRepository(db)
	ctx := context.Background()

	truncateAll(t, db)
	user := loginTestUser(t, db, "+84900000501")
	other := loginTestUser(t, db, "+84900000502")
	_, rowID := seedSession(t, db, repo, user)
	sid := sessionIDOf(t, db, rowID)

	active, err := repo.IsSessionActive(ctx, user.ID, sid)
	require.NoError(t, err)
	assert.True(t, active)

	t.Run("another user's id does not vouch for the session", func(t *testing.T) {
		active, err := repo.IsSessionActive(ctx, other.ID, sid)
		require.NoError(t, err)
		assert.False(t, active)
	})

	t.Run("unknown session", func(t *testing.T) {
		active, err := repo.IsSessionActive(ctx, user.ID, "nope")
		require.NoError(t, err)
		assert.False(t, active)
	})

	t.Run("expired session", func(t *testing.T) {
		_, err := db.Exec(ctx, `UPDATE user_sessions SET expires_at = now() - interval '1 second' WHERE session_id = $1`, sid)
		require.NoError(t, err)
		active, err := repo.IsSessionActive(ctx, user.ID, sid)
		require.NoError(t, err)
		assert.False(t, active)
	})

	t.Run("revoked session", func(t *testing.T) {
		_, err := db.Exec(ctx, `UPDATE user_sessions SET expires_at = now() + interval '1 day' WHERE session_id = $1`, sid)
		require.NoError(t, err)
		require.NoError(t, repo.RevokeSession(ctx, user.ID, sid, domain.RevokeReasonLogout))
		active, err := repo.IsSessionActive(ctx, user.ID, sid)
		require.NoError(t, err)
		assert.False(t, active)
	})
}

func insertVerifiedUser(t *testing.T, db database.Database, phone, email string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.QueryRow(context.Background(),
		`INSERT INTO users (email, email_verified_at, phone_number, password_hash) VALUES ($1, now(), $2, 'old-hash') RETURNING id`,
		email, phone).Scan(&id))
	return id
}

func TestPasswordResetRepository(t *testing.T) {
	db := connectTestDB(t)
	repo := NewPasswordResetRepository(db)
	logins := NewLoginRepository(db)
	ctx := context.Background()

	insertOTP := func(t *testing.T, email string) int64 {
		t.Helper()
		var id int64
		require.NoError(t, db.QueryRow(ctx,
			`INSERT INTO otp_codes (email, purpose, otp_hash, expires_at) VALUES ($1, 'reset_password', 'h', now() + interval '10 minutes') RETURNING id`,
			email).Scan(&id))
		return id
	}

	t.Run("FindVerifiedUserByEmail: verified only", func(t *testing.T) {
		truncateAll(t, db)
		id := insertVerifiedUser(t, db, "+84900000601", "v@example.com")
		insertUser(t, db, "+84900000602", "u@example.com") // email_verified_at NULL
		insertPending(t, db, "+84900000603", "p@example.com", time.Now().Add(time.Hour))

		u, err := repo.FindVerifiedUserByEmail(ctx, "v@example.com")
		require.NoError(t, err)
		assert.Equal(t, id, u.ID)

		for _, email := range []string{"u@example.com", "p@example.com", "none@example.com"} {
			_, err := repo.FindVerifiedUserByEmail(ctx, email)
			assert.ErrorIs(t, err, domain.ErrUserNotFound, email)
		}
	})

	t.Run("ResetPassword: password, code, every session and the lock change together (AC28)", func(t *testing.T) {
		truncateAll(t, db)
		const email = "r@example.com"
		id := insertVerifiedUser(t, db, "+84900000604", email)
		user := &domain.User{ID: id, Email: email, PhoneNumber: "+84900000604"}
		for _, model := range []string{"iPhone", "Pixel"} {
			_, err := logins.CreateLoginSession(ctx, user, testDevice(model), testSessionTTL, uniqueTokenIssuer)
			require.NoError(t, err)
		}
		for range 5 {
			_, err := logins.ReserveLoginAttempt(ctx, email, 5, 15*time.Minute, 15*time.Minute)
			require.NoError(t, err)
		}
		require.NotNil(t, findLoginAttempt(t, db, email))
		otpID := insertOTP(t, email)

		require.NoError(t, repo.ResetPassword(ctx, id, "new-hash", otpID))

		var hash string
		require.NoError(t, db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash))
		assert.Equal(t, "new-hash", hash)

		var active int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_sessions WHERE user_id = $1 AND is_revoked = false`, id).Scan(&active))
		assert.Zero(t, active)
		var reason string
		require.NoError(t, db.QueryRow(ctx, `SELECT DISTINCT revoked_reason FROM user_sessions WHERE user_id = $1`, id).Scan(&reason))
		assert.Equal(t, domain.RevokeReasonPasswordReset, reason)

		var usedAt *time.Time
		require.NoError(t, db.QueryRow(ctx, `SELECT used_at FROM otp_codes WHERE id = $1`, otpID).Scan(&usedAt))
		assert.NotNil(t, usedAt)

		assert.Nil(t, findLoginAttempt(t, db, email), "the sign-in lock is lifted")
	})

	t.Run("a used code cannot reset again and changes nothing", func(t *testing.T) {
		truncateAll(t, db)
		const email = "twice@example.com"
		id := insertVerifiedUser(t, db, "+84900000605", email)
		otpID := insertOTP(t, email)
		require.NoError(t, repo.ResetPassword(ctx, id, "first", otpID))

		err := repo.ResetPassword(ctx, id, "second", otpID)

		assert.ErrorIs(t, err, domain.ErrOTPExpired)
		var hash string
		require.NoError(t, db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash))
		assert.Equal(t, "first", hash)
	})

	t.Run("other users' sessions and counters are untouched", func(t *testing.T) {
		truncateAll(t, db)
		id := insertVerifiedUser(t, db, "+84900000606", "a@example.com")
		other := loginTestUser(t, db, "+84900000607")
		_, err := logins.CreateLoginSession(ctx, other, testDevice("iPhone"), testSessionTTL, uniqueTokenIssuer)
		require.NoError(t, err)
		_, err = logins.ReserveLoginAttempt(ctx, other.Email, 5, 15*time.Minute, 15*time.Minute)
		require.NoError(t, err)
		otpID := insertOTP(t, "a@example.com")

		require.NoError(t, repo.ResetPassword(ctx, id, "h", otpID))

		assert.Equal(t, 1, countActiveSessions(t, db, other.ID))
		assert.NotNil(t, findLoginAttempt(t, db, other.Email))
	})
}
