package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// connectTestDB opens a connection to a real Postgres instance for
// integration-testing this package's SQL. It is the only place these tests
// touch infrastructure: mocking database/sql here would prove the Go code
// compiles, not that the SQL is correct — see
// .claude/skills/golang-clean-arch-tdd/references/layer_conventions.md.
//
// Connection settings default to the values in
// services/appService/config/config.dev.yaml and are overridable through
// TEST_DB_* env vars for CI. Any test using this helper is skipped under
// `-short`, and skipped anyway if no Postgres answers — that is what lets
// `go test -short ./...` stay green with no database present.
func connectTestDB(t *testing.T) database.Database {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping postgres integration test in -short mode")
	}

	cfg := &database.Config{
		Type:     database.PostgreSQL,
		Host:     envOr("TEST_DB_HOST", "localhost"),
		Port:     envOrInt(t, "TEST_DB_PORT", 5432),
		Username: envOr("TEST_DB_USER", "postgres"),
		Password: envOr("TEST_DB_PASSWORD", "postgres"),
		Database: envOr("TEST_DB_NAME", "innerlamp"),
		SSLMode:  envOr("TEST_DB_SSLMODE", "disable"),
		Timeout:  5 * time.Second,
	}

	db, err := database.New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Connect(ctx); err != nil {
		t.Skipf("postgres not available at %s:%d: %v", cfg.Host, cfg.Port, err)
	}
	t.Cleanup(func() { _ = db.Close() })

	applyMigration(t, db)
	truncateAll(t, db)

	return db
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(t *testing.T, key string, fallback int) int {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	require.NoErrorf(t, err, "invalid %s=%q", key, v)
	return n
}

// applyMigration runs 0001_register_onboarding.sql,
// 0002_login_devices_and_rate_limit.sql and 0003_otp_send_limits.sql. Every
// statement in them is CREATE/ADD/DROP ... IF (NOT) EXISTS, so running them
// against an already-migrated database is a no-op — this is what lets each
// test file bootstrap its own schema instead of depending on a separate
// migration step having run first.
func applyMigration(t *testing.T, db database.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, file := range []string{
		"0001_register_onboarding.sql",
		"0002_login_devices_and_rate_limit.sql",
		"0003_otp_send_limits.sql",
		"0004_attestation.sql",
		"0005_temples.sql",
		"0007_deities.sql",
		"0011_temple_content.sql", // temples.introduction etc.: postgres_temple_repository.go selects them
		"0012_articles.sql",
		"0013_sutras.sql",
		"0014_dharma_talks.sql",
		"0016_article_thumbnail.sql", // articles.thumbnail_url: postgres_article_repository.go selects it
		"0017_gallery_images.sql",
		"0018_memorials.sql",
		"0019_sutra_audio_search.sql", // sutras.audio_url + unaccent: postgres_sutra_repository.go selects/uses them
		"0020_sutra_reference_license_summary.sql", // sutras.summary etc.: postgres_sutra_repository.go selects/inserts them
	} {
		sqlBytes, err := os.ReadFile("../../../migrations/" + file)
		require.NoError(t, err)
		_, err = db.Exec(ctx, string(sqlBytes))
		require.NoError(t, err)
	}
}

// truncateAll clears every table this feature touches so each test starts
// from empty, regardless of what an earlier test (or an earlier run) left
// behind.
func truncateAll(t *testing.T, db database.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Exec(ctx, `TRUNCATE TABLE user_sessions, user_devices, login_attempts, otp_send_limits, otp_codes, pending_registration, attestation_keys, attestation_challenges, temples, deities, gallery_images, articles, sutras, dharma_talks, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
}

func stubTokenIssuer(userID, sessionID string) (string, string, error) {
	return "access-" + userID, "refresh-" + userID, nil
}

func insertUser(t *testing.T, db database.Database, phoneNumber, email string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO users (email, phone_number, password_hash) VALUES ($1, $2, 'x') RETURNING id`,
		email, phoneNumber,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertPending(t *testing.T, db database.Database, phoneNumber, email string, expiresAt time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO pending_registration (phone_number, email, password_hash, expires_at) VALUES ($1, $2, 'hash', $3) RETURNING id`,
		phoneNumber, email, expiresAt,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertOTPRow(t *testing.T, db database.Database, phoneNumber string, purpose domain.OTPPurpose, expiresAt time.Time, usedAt, lockedUntil *time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO otp_codes (phone_number, purpose, otp_hash, expires_at, used_at, locked_until)
		 VALUES ($1, $2, 'hash', $3, $4, $5) RETURNING id`,
		phoneNumber, string(purpose), expiresAt, usedAt, lockedUntil,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// insertSendLimit seeds an otp_send_limits row, so a test can start a cycle
// part-way through without driving real sends to that point first.
func insertSendLimit(t *testing.T, db database.Database, phoneNumber string, purpose domain.OTPPurpose, sendCount int, lastSentAt time.Time, lockedUntil *time.Time) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO otp_send_limits (phone_number, purpose, send_count, last_sent_at, locked_until)
		 VALUES ($1, $2, $3, $4, $5)`,
		phoneNumber, string(purpose), sendCount, lastSentAt, lockedUntil,
	)
	require.NoError(t, err)
}

// testSendLimits mirrors the shape the use case threads down; the values match
// config.dev.yaml's otp block.
func testSendLimits() domain.SendLimits {
	return domain.SendLimits{
		Cooldown:     60 * time.Second,
		LockDuration: 30 * time.Minute,
		MaxAttempts:  3,
	}
}

func TestRegistrationRepository_FindConflicts(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	t.Run("no conflict", func(t *testing.T) {
		truncateAll(t, db)
		c, err := repo.FindConflicts(ctx, "+84900000001", "nobody@example.com")
		require.NoError(t, err)
		require.False(t, c.HasPhoneConflict())
		require.False(t, c.HasEmailConflict())
	})

	t.Run("phone and email taken in users", func(t *testing.T) {
		truncateAll(t, db)
		insertUser(t, db, "+84900000002", "taken@example.com")

		c, err := repo.FindConflicts(ctx, "+84900000002", "taken@example.com")
		require.NoError(t, err)
		require.True(t, c.PhoneInUsers)
		require.True(t, c.EmailInUsers)
		require.True(t, c.HasPhoneConflict())
		require.True(t, c.HasEmailConflict())
	})

	// v5.1: a phone's own live pending row is that caller's in-flight attempt,
	// not a claim against them. request-otp overwrites it, so a reinstall can
	// restart without waiting out the (now 24h) PendingRegistrationTTL.
	t.Run("own live pending is not a conflict", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000003", "mine@example.com", time.Now().Add(24*time.Hour))

		c, err := repo.FindConflicts(ctx, "+84900000003", "mine@example.com")
		require.NoError(t, err)
		require.False(t, c.HasPhoneConflict())
		require.False(t, c.HasEmailConflict())
	})

	// AC6: the unverified email of *another* phone's attempt is still a claim
	// on a globally-unique key, so it must be found even though it is not in
	// users yet.
	t.Run("another phone's live pending holds the email", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000005", "shared@example.com", time.Now().Add(24*time.Hour))

		c, err := repo.FindConflicts(ctx, "+84900000006", "shared@example.com")
		require.NoError(t, err)
		require.True(t, c.EmailInPending)
		require.True(t, c.HasEmailConflict())
		require.False(t, c.HasPhoneConflict())
	})

	// An expired pending row is an abandoned attempt, not a claim on the email:
	// nothing deletes it until the cleanup job runs, so counting it would block
	// that email for the whole TTL.
	t.Run("expired pending rows are not conflicts", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000004", "expired@example.com", time.Now().Add(-time.Second))

		c, err := repo.FindConflicts(ctx, "+84900000004", "expired@example.com")
		require.NoError(t, err)
		require.False(t, c.HasPhoneConflict())
		require.False(t, c.HasEmailConflict())

		insertPending(t, db, "+84900000005", "shared@example.com", time.Now().Add(-time.Second))
		c, err = repo.FindConflicts(ctx, "+84900000006", "shared@example.com")
		require.NoError(t, err)
		require.False(t, c.HasEmailConflict())
	})

	t.Run("expired pending does not hide a real account", func(t *testing.T) {
		truncateAll(t, db)
		insertUser(t, db, "+84900000007", "real@example.com")
		insertPending(t, db, "+84900000008", "real@example.com", time.Now().Add(-time.Second))

		c, err := repo.FindConflicts(ctx, "+84900000008", "real@example.com")
		require.NoError(t, err)
		require.True(t, c.EmailInUsers)
		require.True(t, c.HasEmailConflict())
	})
}

func TestRegistrationRepository_SavePendingAndIssueOTP(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()

	t.Run("happy path writes pending and otp, first send is free", func(t *testing.T) {
		truncateAll(t, db)
		dharma := "Thich Test"
		pending := &domain.PendingRegistration{
			PhoneNumber:  "+84900000010",
			Email:        "a@example.com",
			PasswordHash: "hash",
			DharmaName:   &dharma,
			// Set by the caller (the use case), same as PendingRegistrationTTL —
			// distinct from the expiresAt argument below, which is the OTP's own
			// expiry (OTPConfig.TTL). They happen to share a value in
			// config.dev.yaml, but SavePendingAndIssueOTP must not conflate them.
			ExpiresAt: time.Now().Add(90 * time.Second),
		}
		expiresAt := time.Now().Add(90 * time.Second).Truncate(time.Millisecond)

		state, err := repo.SavePendingAndIssueOTP(ctx, pending, "otphash", expiresAt, testSendLimits())
		require.NoError(t, err)
		require.NotZero(t, pending.ID)

		// The first send of a cycle is free: the counter stays at 0 and all
		// three counted attempts remain.
		require.Equal(t, 0, state.SendCount)
		require.Equal(t, 3, state.Remaining)

		got, err := repo.FindPendingRegistration(ctx, "+84900000010")
		require.NoError(t, err)
		require.Equal(t, "a@example.com", got.Email)
		require.Equal(t, "Thich Test", *got.DharmaName)

		otp, err := otpRepo.FindActiveOTP(ctx, "+84900000010", domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.Equal(t, "otphash", otp.OTPHash)

		var sendCount int
		require.NoError(t, db.QueryRow(ctx, `SELECT send_count FROM otp_send_limits WHERE phone_number = $1 AND purpose = 'register'`, "+84900000010").Scan(&sendCount))
		require.Equal(t, 0, sendCount)
	})

	t.Run("second call inside the cooldown is refused and writes nothing", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000011"
		pending := &domain.PendingRegistration{PhoneNumber: phone, Email: "b@example.com", PasswordHash: "hash1", ExpiresAt: time.Now().Add(90 * time.Second)}
		_, err := repo.SavePendingAndIssueOTP(ctx, pending, "hash-1", time.Now().Add(90*time.Second), testSendLimits())
		require.NoError(t, err)

		firstOTP, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.NoError(t, err)

		pending2 := &domain.PendingRegistration{PhoneNumber: phone, Email: "b2@example.com", PasswordHash: "hash2", ExpiresAt: time.Now().Add(90 * time.Second)}
		_, err = repo.SavePendingAndIssueOTP(ctx, pending2, "hash-2", time.Now().Add(90*time.Second), testSendLimits())
		require.ErrorIs(t, err, domain.ErrOTPResendTooSoon)

		// Nothing was written: the payload and the active OTP are untouched, and
		// no extra otp_codes row exists.
		got, err := repo.FindPendingRegistration(ctx, phone)
		require.NoError(t, err)
		require.Equal(t, "b@example.com", got.Email, "a refused send must not upsert the payload")

		active, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.Equal(t, firstOTP.ID, active.ID)

		var otpRows int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM otp_codes WHERE phone_number = $1`, phone).Scan(&otpRows))
		require.Equal(t, 1, otpRows)
	})
}

func TestOTPRepository_IssueOTP(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()

	t.Run("invalidates old otp and counts the send, without touching pending", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000020"
		pendingExpiry := time.Now().Add(24 * time.Hour).Truncate(time.Millisecond)
		insertPending(t, db, phone, "c@example.com", pendingExpiry)
		oldOTPID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)
		// Seed a live cycle so this resend is a counted send, not the free first.
		insertSendLimit(t, db, phone, domain.OTPPurposeRegister, 1, time.Now().Add(-5*time.Minute), nil)

		state, err := otpRepo.IssueOTP(ctx, phone, domain.OTPPurposeRegister, "new-hash", time.Now().Add(90*time.Second), testSendLimits())
		require.NoError(t, err)
		require.Equal(t, 2, state.SendCount, "a counted send advances the counter")
		require.Equal(t, 1, state.Remaining)

		var newHash string
		require.NoError(t, db.QueryRow(ctx, `SELECT otp_hash FROM otp_codes WHERE phone_number = $1 AND used_at IS NULL`, phone).Scan(&newHash))
		require.Equal(t, "new-hash", newHash)

		var oldUsedAt sql.NullTime
		require.NoError(t, db.QueryRow(ctx, `SELECT used_at FROM otp_codes WHERE id = $1`, oldOTPID).Scan(&oldUsedAt))
		require.True(t, oldUsedAt.Valid)

		// v5.1: resend leaves the payload alone — the payload's own TTL bounds
		// the attempt, and the replacement code's 90 seconds has nothing to do
		// with it.
		pending, err := repo.FindPendingRegistration(ctx, phone)
		require.NoError(t, err)
		require.WithinDuration(t, pendingExpiry, pending.ExpiresAt, 2*time.Second)
	})

	t.Run("does not touch pending_registration for a non-register purpose", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000021"
		originalExpiry := time.Now().Add(90 * time.Second)
		insertPending(t, db, phone, "d@example.com", originalExpiry)
		insertOTPRow(t, db, phone, domain.OTPPurposeLogin, time.Now().Add(90*time.Second), nil, nil)

		_, err := otpRepo.IssueOTP(ctx, phone, domain.OTPPurposeLogin, "hash", time.Now().Add(9*time.Minute), testSendLimits())
		require.NoError(t, err)

		pending, err := repo.FindPendingRegistration(ctx, phone)
		require.NoError(t, err)
		require.WithinDuration(t, originalExpiry, pending.ExpiresAt, 2*time.Second)
	})
}

// TestRegistrationRepository_SendLimits pins the AC5 state machine at the layer
// that owns it: the free first send, the counted sends, the cap setting a
// temporary lock, the cycle resetting once it ages out, and the counter being
// released when the registration completes.
func TestRegistrationRepository_SendLimits(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	limits := testSendLimits()

	t.Run("cap sets a 30m lock and a locked cycle keeps refusing", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000080"
		// A live cycle that has already used all three counted sends, past its
		// cooldown so only the cap can refuse this attempt.
		insertSendLimit(t, db, phone, domain.OTPPurposeRegister, 3, time.Now().Add(-5*time.Minute), nil)
		insertPending(t, db, phone, "cap@example.com", time.Now().Add(90*time.Second))
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		before := time.Now()
		_, err := otpRepo.IssueOTP(ctx, phone, domain.OTPPurposeRegister, "hash", time.Now().Add(90*time.Second), limits)
		require.ErrorIs(t, err, domain.ErrOTPSendLimited)

		var limited *domain.OTPSendLimitedError
		require.ErrorAs(t, err, &limited)
		require.WithinDuration(t, before.Add(30*time.Minute), limited.LockedUntil, time.Minute)

		// The lock is persisted, so the next attempt refuses immediately —
		// without extending the deadline.
		_, err = otpRepo.IssueOTP(ctx, phone, domain.OTPPurposeRegister, "hash", time.Now().Add(90*time.Second), limits)
		require.ErrorIs(t, err, domain.ErrOTPSendLimited)
	})

	t.Run("a cycle older than the lock duration resets to the free first send", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000081"
		// Last send well past LockDuration, so the cycle has aged out.
		insertSendLimit(t, db, phone, domain.OTPPurposeRegister, 3, time.Now().Add(-31*time.Minute), nil)
		insertPending(t, db, phone, "reset@example.com", time.Now().Add(90*time.Second))
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		state, err := otpRepo.IssueOTP(ctx, phone, domain.OTPPurposeRegister, "hash", time.Now().Add(90*time.Second), limits)
		require.NoError(t, err)
		require.Equal(t, 0, state.SendCount, "an aged-out cycle starts over at the free first send")
		require.Equal(t, 3, state.Remaining)
	})

	t.Run("cooldown refusal does not consume a slot", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000082"
		insertSendLimit(t, db, phone, domain.OTPPurposeRegister, 1, time.Now().Add(-10*time.Second), nil)
		insertPending(t, db, phone, "cooldown@example.com", time.Now().Add(90*time.Second))
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		_, err := otpRepo.IssueOTP(ctx, phone, domain.OTPPurposeRegister, "hash", time.Now().Add(90*time.Second), limits)
		require.ErrorIs(t, err, domain.ErrOTPResendTooSoon)

		var sendCount int
		require.NoError(t, db.QueryRow(ctx, `SELECT send_count FROM otp_send_limits WHERE phone_number = $1 AND purpose = 'register'`, phone).Scan(&sendCount))
		require.Equal(t, 1, sendCount, "a refused send must not consume a slot")
	})

	t.Run("completing a registration releases the counter", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000083"
		insertSendLimit(t, db, phone, domain.OTPPurposeRegister, 2, time.Now().Add(-time.Minute), nil)
		pendingID := insertPending(t, db, phone, "done@example.com", time.Now().Add(time.Minute))
		otpID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{ID: pendingID, PhoneNumber: phone, Email: "done@example.com", PasswordHash: "hash"}
		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.NoError(t, err)

		var count int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM otp_send_limits WHERE phone_number = $1`, phone).Scan(&count))
		require.Equal(t, 0, count, "a completed registration must not leave rate-limit state behind")
	})

	t.Run("ReleaseSendLimit drops the counter", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000084"
		insertSendLimit(t, db, phone, domain.OTPPurposeRegister, 2, time.Now().Add(-time.Minute), nil)

		require.NoError(t, otpRepo.ReleaseSendLimit(ctx, phone, domain.OTPPurposeRegister))

		var count int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM otp_send_limits WHERE phone_number = $1`, phone).Scan(&count))
		require.Equal(t, 0, count)
	})
}

func TestRegistrationRepository_FindPendingRegistration(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		truncateAll(t, db)
		_, err := repo.FindPendingRegistration(ctx, "+84900000030")
		require.ErrorIs(t, err, domain.ErrPendingRegistrationNotFound)
	})

	t.Run("expired", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000031", "e@example.com", time.Now().Add(-time.Second))
		_, err := repo.FindPendingRegistration(ctx, "+84900000031")
		require.ErrorIs(t, err, domain.ErrPendingRegistrationExpired)
	})

	t.Run("found", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000032", "f@example.com", time.Now().Add(time.Minute))
		p, err := repo.FindPendingRegistration(ctx, "+84900000032")
		require.NoError(t, err)
		require.Equal(t, "f@example.com", p.Email)
	})
}

func TestOTPRepository_FindActiveOTP(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	phone := "+84900000040"

	t.Run("no row at all", func(t *testing.T) {
		truncateAll(t, db)
		_, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("unused unexpired row is active", func(t *testing.T) {
		truncateAll(t, db)
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)
		otp, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.Nil(t, otp.LockedUntil)
	})

	t.Run("used row (verified or invalidated) is not active", func(t *testing.T) {
		truncateAll(t, db)
		usedAt := time.Now()
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), &usedAt, nil)
		_, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("expired unused row is not active", func(t *testing.T) {
		truncateAll(t, db)
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(-time.Second), nil, nil)
		_, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("locked row is active even past its own expires_at, so the caller learns about the lock", func(t *testing.T) {
		truncateAll(t, db)
		usedAt := time.Now()
		lockedUntil := time.Now().Add(15 * time.Minute)
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(-time.Minute), &usedAt, &lockedUntil)

		otp, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.NotNil(t, otp.LockedUntil)
		require.WithinDuration(t, lockedUntil, *otp.LockedUntil, time.Second)
	})

	t.Run("a lifted lock is dead, not active — caller must request a new code", func(t *testing.T) {
		truncateAll(t, db)
		usedAt := time.Now().Add(-20 * time.Minute)
		lockedUntil := time.Now().Add(-5 * time.Minute)
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(-19*time.Minute), &usedAt, &lockedUntil)

		_, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("picks the newest row when several exist", func(t *testing.T) {
		truncateAll(t, db)
		oldUsedAt := time.Now()
		insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), &oldUsedAt, nil)
		time.Sleep(10 * time.Millisecond)
		newestID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		otp, err := otpRepo.FindActiveOTP(ctx, phone, domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.Equal(t, newestID, otp.ID)
	})
}

func TestOTPRepository_IncrementOTPAttemptsAndLockOTP(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()

	truncateAll(t, db)
	otpID := insertOTPRow(t, db, "+84900000050", domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

	attempts, err := otpRepo.IncrementOTPAttempts(ctx, otpID)
	require.NoError(t, err)
	require.Equal(t, 1, attempts)

	attempts, err = otpRepo.IncrementOTPAttempts(ctx, otpID)
	require.NoError(t, err)
	require.Equal(t, 2, attempts)

	lockedUntil := time.Now().Add(15 * time.Minute)
	require.NoError(t, otpRepo.LockOTP(ctx, otpID, lockedUntil))

	var (
		gotLockedUntil sql.NullTime
		gotUsedAt      sql.NullTime
	)
	require.NoError(t, db.QueryRow(ctx, `SELECT locked_until, used_at FROM otp_codes WHERE id = $1`, otpID).Scan(&gotLockedUntil, &gotUsedAt))
	require.True(t, gotLockedUntil.Valid)
	require.True(t, gotUsedAt.Valid, "locking must also mark the row used so it can never be replayed")
}

func TestRegistrationRepository_CreateUserAndSession(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	t.Run("happy path creates user, deletes pending, consumes otp, opens session", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000060"
		pendingID := insertPending(t, db, phone, "g@example.com", time.Now().Add(time.Minute))
		otpID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{
			ID:           pendingID,
			PhoneNumber:  phone,
			Email:        "g@example.com",
			PasswordHash: "hash",
		}

		result, err := repo.CreateUserAndSession(ctx, pending, otpID, 168*time.Hour, stubTokenIssuer)
		require.NoError(t, err)
		require.NotZero(t, result.User.ID)
		require.Equal(t, "g@example.com", result.User.Email)
		require.Nil(t, result.User.EmailVerifiedAt, "AC2/AC3: email must not be marked verified at registration")
		require.Equal(t, "access-"+strconv.FormatInt(result.User.ID, 10), result.AccessToken)
		require.Equal(t, "refresh-"+strconv.FormatInt(result.User.ID, 10), result.RefreshToken)

		_, err = repo.FindPendingRegistration(ctx, phone)
		require.ErrorIs(t, err, domain.ErrPendingRegistrationNotFound, "pending row must be gone in the same tx")

		var usedAt sql.NullTime
		require.NoError(t, db.QueryRow(ctx, `SELECT used_at FROM otp_codes WHERE id = $1`, otpID).Scan(&usedAt))
		require.True(t, usedAt.Valid)

		var sessionCount int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM user_sessions WHERE user_id = $1`, result.User.ID).Scan(&sessionCount))
		require.Equal(t, 1, sessionCount)

		var storedHash string
		require.NoError(t, db.QueryRow(ctx, `SELECT refresh_token_hash FROM user_sessions WHERE user_id = $1`, result.User.ID).Scan(&storedHash))
		require.Equal(t, domain.HashRefreshToken(result.RefreshToken), storedHash)
		require.NotEqual(t, result.RefreshToken, storedHash, "the raw refresh token must never be persisted")
	})

	t.Run("session id handed to the issuer equals the persisted user_sessions.session_id", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000065"
		pendingID := insertPending(t, db, phone, "sid@example.com", time.Now().Add(time.Minute))
		otpID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{ID: pendingID, PhoneNumber: phone, Email: "sid@example.com", PasswordHash: "hash"}

		var gotUserID, gotSessionID string
		issuer := func(userID, sessionID string) (string, string, error) {
			gotUserID, gotSessionID = userID, sessionID
			return "access", "refresh", nil
		}

		result, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, issuer)
		require.NoError(t, err)
		require.NotEmpty(t, gotSessionID, "sessionID must exist before the tokens are minted")
		require.Equal(t, strconv.FormatInt(result.User.ID, 10), gotUserID)

		var storedSessionID string
		require.NoError(t, db.QueryRow(ctx, `SELECT session_id FROM user_sessions WHERE user_id = $1`, result.User.ID).Scan(&storedSessionID))
		require.Equal(t, gotSessionID, storedSessionID, "the sid baked into the access token must be the row logout will look up")
	})

	t.Run("email taken by a completed account is rejected inside the transaction", func(t *testing.T) {
		truncateAll(t, db)
		insertUser(t, db, "+84900000061", "taken2@example.com")

		phone := "+84900000062"
		pendingID := insertPending(t, db, phone, "taken2@example.com", time.Now().Add(time.Minute))
		otpID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{ID: pendingID, PhoneNumber: phone, Email: "taken2@example.com", PasswordHash: "hash"}
		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.ErrorIs(t, err, domain.ErrEmailTaken)

		// Rejected before commit: the pending row must still be there.
		_, err = repo.FindPendingRegistration(ctx, phone)
		require.NoError(t, err)
	})

	t.Run("phone unique violation on insert is translated, not just the email re-check", func(t *testing.T) {
		truncateAll(t, db)
		// Only email is re-checked inside the transaction (pending_registration.email
		// is deliberately not unique — see design.md flow 2 Step 5). phone_number is
		// unique on both pending_registration and users, so this path can only be
		// exercised by forcing the actual INSERT to collide, bypassing the re-check
		// — which is exactly what isUniqueViolation exists to catch.
		insertUser(t, db, "+84900000064", "someone-else@example.com")

		phone := "+84900000064"
		pendingID := insertPending(t, db, phone, "unique-email@example.com", time.Now().Add(time.Minute))
		otpID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{ID: pendingID, PhoneNumber: phone, Email: "unique-email@example.com", PasswordHash: "hash"}
		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.ErrorIs(t, err, domain.ErrPhoneTaken)

		_, err = repo.FindPendingRegistration(ctx, phone)
		require.NoError(t, err, "rejected before commit — pending row must survive")
	})

	t.Run("token issuer failure rolls back the whole transaction", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000063"
		pendingID := insertPending(t, db, phone, "h@example.com", time.Now().Add(time.Minute))
		otpID := insertOTPRow(t, db, phone, domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{ID: pendingID, PhoneNumber: phone, Email: "h@example.com", PasswordHash: "hash"}
		failingIssuer := func(userID, sessionID string) (string, string, error) {
			return "", "", errors.New("signing key unavailable")
		}

		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, failingIssuer)
		require.Error(t, err)

		var userCount int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM users WHERE email = $1`, "h@example.com").Scan(&userCount))
		require.Equal(t, 0, userCount, "a failed token issue must not leave a committed user with no session")

		_, err = repo.FindPendingRegistration(ctx, phone)
		require.NoError(t, err, "pending row must survive a rolled-back transaction")
	})
}

func TestRegistrationRepository_DeletePendingRegistration(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	truncateAll(t, db)
	phone := "+84900000070"
	insertPending(t, db, phone, "i@example.com", time.Now().Add(time.Minute))
	pending, err := repo.FindPendingRegistration(ctx, phone)
	require.NoError(t, err)

	require.NoError(t, repo.DeletePendingRegistration(ctx, pending.ID))

	_, err = repo.FindPendingRegistration(ctx, phone)
	require.ErrorIs(t, err, domain.ErrPendingRegistrationNotFound)
}

// --- helper-function unit tests: no DB required, always run under -short ---

func TestNullableString(t *testing.T) {
	require.Nil(t, nullableString(nil))
	empty := ""
	require.Nil(t, nullableString(&empty))
	val := "value"
	require.Equal(t, "value", nullableString(&val))
}

func TestHashRefreshToken(t *testing.T) {
	h1 := domain.HashRefreshToken("token-a")
	h2 := domain.HashRefreshToken("token-a")
	h3 := domain.HashRefreshToken("token-b")
	require.Equal(t, h1, h2)
	require.NotEqual(t, h1, h3)
	require.Len(t, h1, 64, "must fit user_sessions.refresh_token_hash VARCHAR(64)")
}

func TestIsUniqueViolation(t *testing.T) {
	require.False(t, isUniqueViolation(errors.New("boom"), usersEmailKeyConstraint))

	matching := &pq.Error{Code: "23505", Constraint: usersEmailKeyConstraint}
	require.True(t, isUniqueViolation(matching, usersEmailKeyConstraint))

	wrongConstraint := &pq.Error{Code: "23505", Constraint: usersPhoneNumberKeyConstraint}
	require.False(t, isUniqueViolation(wrongConstraint, usersEmailKeyConstraint))

	wrongCode := &pq.Error{Code: "23503", Constraint: usersEmailKeyConstraint}
	require.False(t, isUniqueViolation(wrongCode, usersEmailKeyConstraint))
}
