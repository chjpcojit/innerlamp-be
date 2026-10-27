package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
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
// 0002_login_devices_and_rate_limit.sql, 0003_otp_send_limits.sql and the
// later files listed below. Every
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
		"0006_admins.sql", // temple_suggestions.reviewed_by references admins
		"0007_deities.sql",
		"0015_deities_deva_type.sql", // widens deities.type CHECK to allow DEVA (deity type filter tests)
		"0011_temple_content.sql",    // temples.introduction etc.: postgres_temple_repository.go selects them
		"0012_articles.sql",
		"0013_sutras.sql",
		"0014_dharma_talks.sql",
		"0016_article_thumbnail.sql", // articles.thumbnail_url: postgres_article_repository.go selects it
		"0017_gallery_images.sql",
		"0018_memorials.sql",
		"0019_sutra_audio_search.sql",              // sutras.audio_url + unaccent: postgres_sutra_repository.go selects/uses them
		"0020_sutra_reference_license_summary.sql", // sutras.summary etc.: postgres_sutra_repository.go selects/inserts them
		"0022_temple_suggestions.sql",
		"0023_registration_ordination_temple.sql",   // users/pending_registration.ordination_temple_id -> temples
		"0024_temple_province.sql",                  // temples.province: postgres_temple_repository.go selects/inserts it
		"0027_lamps.sql",                            // lamps: postgres_lamp_repository.go (references users)
		"0026_administrative_units.sql",             // addr_*: postgres_administrative_unit_repository.go
		"0028_genealogy.sql",                        // genealogy_*: postgres_genealogy_*_repository.go (references users)
		"0029_users_avatar.sql",                     // users.avatar_object: postgres_profile_repository.go selects/updates it
		"0030_temple_suggestions_address_codes.sql", // temple_suggestions.province_code/ward_code/ward: postgres_temple_suggestion_repository.go
		"0031_calendar_events.sql",                  // calendar_*: postgres_calendar_event_repository.go
		"0032_temple_address_codes.sql",             // temples.province_code/ward_code/ward: postgres_temple_repository.go (v1.7)
		"0033_users_full_name.sql",                  // users/pending_registration.full_name: registration + login repositories select/insert it
		"0034_temple_schedules.sql",                 // temple_schedules/steps: postgres_temple_schedule_repository.go
		"0035_temple_events.sql",                    // temple_events: postgres_temple_event_repository.go
		"0036_temple_announcements.sql",             // temple_announcements/issuing bodies + users.receive_temple_announcements
		"0037_auth_email_identity.sql",              // otp_codes/otp_send_limits/pending_registration keyed by email
		"0038_login_attempts_email.sql",             // login_attempts keyed by email (login v2 Phase 1)
		"0040_lamps_recipient_optional.sql",         // lamps.recipient nullable (lamp v2.0)
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
	_, err := db.Exec(ctx, `TRUNCATE TABLE user_sessions, user_devices, login_attempts, otp_send_limits, otp_codes, pending_registration, attestation_keys, attestation_challenges, temple_suggestions, temples, deities, gallery_images, articles, sutras, dharma_talks, users RESTART IDENTITY CASCADE`)
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

func insertOTPRow(t *testing.T, db database.Database, email string, purpose domain.OTPPurpose, expiresAt time.Time, usedAt, lockedUntil *time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO otp_codes (email, purpose, otp_hash, expires_at, used_at, locked_until)
		 VALUES ($1, $2, 'hash', $3, $4, $5) RETURNING id`,
		email, string(purpose), expiresAt, usedAt, lockedUntil,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// insertSendLimit seeds an otp_send_limits row, so a test can start a cycle
// part-way through without driving real sends to that point first.
func insertSendLimit(t *testing.T, db database.Database, email string, purpose domain.OTPPurpose, sendCount int, lastSentAt time.Time, lockedUntil *time.Time) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO otp_send_limits (email, purpose, send_count, last_sent_at, locked_until)
		 VALUES ($1, $2, $3, $4, $5)`,
		email, string(purpose), sendCount, lastSentAt, lockedUntil,
	)
	require.NoError(t, err)
}

func queryInt(t *testing.T, db database.Database, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(context.Background(), query, args...).Scan(&n))
	return n
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

func TestRegistrationRepository_PhoneTakenAndEmailHasAccount(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	insertUser(t, db, "+84900000001", "owner@example.com")
	insertPending(t, db, "+84900000002", "pending@example.com", time.Now().Add(time.Hour))

	taken, err := repo.PhoneTaken(ctx, "+84900000001")
	require.NoError(t, err)
	require.True(t, taken, "AC7: a number on an account is taken")

	taken, err = repo.PhoneTaken(ctx, "+84900000002")
	require.NoError(t, err)
	require.False(t, taken, "AC7: a number held only by a pending registration does not block")

	has, err := repo.EmailHasAccount(ctx, "owner@example.com")
	require.NoError(t, err)
	require.True(t, has)

	has, err = repo.EmailHasAccount(ctx, "pending@example.com")
	require.NoError(t, err)
	require.False(t, has, "a pending row is not an account")
}

func newPending(email, phone string) *domain.PendingRegistration {
	name := "Nguyen Van Test"
	return &domain.PendingRegistration{
		Email: email, PhoneNumber: phone, PasswordHash: "pw-hash", FullName: &name,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
}

// ageSendLimit moves the cycle's last send back past the cooldown, so a test
// can send again without sleeping.
func ageSendLimit(t *testing.T, db database.Database, email string) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`UPDATE otp_send_limits SET last_sent_at = now() - interval '2 minutes' WHERE email = $1`, email)
	require.NoError(t, err)
}

func TestRegistrationRepository_SavePendingAndIssueOTP(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()

	t.Run("writes pending and code keyed by email, first send is free", func(t *testing.T) {
		truncateAll(t, db)
		dharma := "Thich Test"
		pending := newPending("a@example.com", "+84900000010")
		pending.DharmaName = &dharma

		state, err := repo.SavePendingAndIssueOTP(ctx, pending, "otphash", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		require.NotZero(t, pending.ID)
		require.Equal(t, 0, state.SendCount)
		require.Equal(t, 3, state.Remaining)

		got, err := repo.FindPendingRegistration(ctx, "a@example.com")
		require.NoError(t, err)
		require.Equal(t, "+84900000010", got.PhoneNumber)
		require.Equal(t, "Thich Test", *got.DharmaName)
		require.False(t, got.AccountExists)

		otp, err := otpRepo.FindActiveOTP(ctx, "a@example.com", domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.Equal(t, "otphash", otp.OTPHash)
		require.Equal(t, "a@example.com", otp.Email)
	})

	t.Run("same email overwrites the pending row and invalidates the old code", func(t *testing.T) {
		truncateAll(t, db)
		email := "same@example.com"
		_, err := repo.SavePendingAndIssueOTP(ctx, newPending(email, "+84900000011"), "hash-1", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		first, err := otpRepo.FindActiveOTP(ctx, email, domain.OTPPurposeRegister)
		require.NoError(t, err)

		ageSendLimit(t, db, email)
		second := newPending(email, "+84900000012")
		second.PasswordHash = "pw-hash-2"
		second.AccountExists = true
		_, err = repo.SavePendingAndIssueOTP(ctx, second, "hash-2", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)

		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM pending_registration WHERE email = $1`, email))
		got, err := repo.FindPendingRegistration(ctx, email)
		require.NoError(t, err)
		require.Equal(t, "+84900000012", got.PhoneNumber)
		require.Equal(t, "pw-hash-2", got.PasswordHash)
		require.True(t, got.AccountExists)

		active, err := otpRepo.FindActiveOTP(ctx, email, domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.NotEqual(t, first.ID, active.ID)
		require.Equal(t, "hash-2", active.OTPHash)
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE email = $1 AND used_at IS NULL`, email))
	})

	t.Run("two pending registrations may hold the same phone", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000013"
		_, err := repo.SavePendingAndIssueOTP(ctx, newPending("one@example.com", phone), "h1", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		_, err = repo.SavePendingAndIssueOTP(ctx, newPending("two@example.com", phone), "h2", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)

		require.Equal(t, 2, queryInt(t, db, `SELECT count(*) FROM pending_registration WHERE phone_number = $1`, phone))
	})

	t.Run("refused inside the cooldown and writes nothing", func(t *testing.T) {
		truncateAll(t, db)
		email := "cool@example.com"
		_, err := repo.SavePendingAndIssueOTP(ctx, newPending(email, "+84900000014"), "hash-1", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)

		again := newPending(email, "+84900000015")
		_, err = repo.SavePendingAndIssueOTP(ctx, again, "hash-2", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.ErrorIs(t, err, domain.ErrOTPResendTooSoon)

		got, err := repo.FindPendingRegistration(ctx, email)
		require.NoError(t, err)
		require.Equal(t, "+84900000014", got.PhoneNumber, "a refused send must not upsert the payload")
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE email = $1`, email))
	})
}

func TestOTPRepository_IssueOTP(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()

	truncateAll(t, db)
	email := "c@example.com"
	pendingExpiry := time.Now().Add(24 * time.Hour).Truncate(time.Millisecond)
	insertPending(t, db, "+84900000020", email, pendingExpiry)
	oldID := insertOTPRow(t, db, email, domain.OTPPurposeRegister, time.Now().Add(10*time.Minute), nil, nil)
	insertSendLimit(t, db, email, domain.OTPPurposeRegister, 1, time.Now().Add(-5*time.Minute), nil)

	state, err := otpRepo.IssueOTP(ctx, email, domain.OTPPurposeRegister, "new-hash", time.Now().Add(10*time.Minute), testSendLimits())
	require.NoError(t, err)
	require.Equal(t, 2, state.SendCount)
	require.Equal(t, 1, state.Remaining)

	active, err := otpRepo.FindActiveOTP(ctx, email, domain.OTPPurposeRegister)
	require.NoError(t, err)
	require.Equal(t, "new-hash", active.OTPHash)
	require.NotEqual(t, oldID, active.ID)

	pending, err := repo.FindPendingRegistration(ctx, email)
	require.NoError(t, err)
	require.WithinDuration(t, pendingExpiry, pending.ExpiresAt, 2*time.Second, "issue leaves the payload alone")

	// Another purpose of the same email has its own budget (otp-mechanism AC1).
	state, err = otpRepo.IssueOTP(ctx, email, domain.OTPPurposeResetPassword, "reset-hash", time.Now().Add(10*time.Minute), testSendLimits())
	require.NoError(t, err)
	require.Equal(t, 0, state.SendCount)
	active, err = otpRepo.FindActiveOTP(ctx, email, domain.OTPPurposeRegister)
	require.NoError(t, err)
	require.Equal(t, "new-hash", active.OTPHash, "a reset code does not invalidate the register code")
}

// TestOTPRepository_ApplySendLimit pins otp-mechanism AC6: the same refusals
// as IssueOTP, and no code is ever written.
func TestOTPRepository_ApplySendLimit(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	purpose := domain.OTPPurposeResetPassword

	t.Run("allowed: counts like a send, creates no code", func(t *testing.T) {
		truncateAll(t, db)
		email := "ok@example.com"
		state, err := otpRepo.ApplySendLimit(ctx, email, purpose, testSendLimits())
		require.NoError(t, err)
		require.Equal(t, 0, state.SendCount)
		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM otp_codes`))

		_, err = otpRepo.ApplySendLimit(ctx, email, purpose, testSendLimits())
		require.ErrorIs(t, err, domain.ErrOTPResendTooSoon, "the first call started the cooldown")
	})

	t.Run("same 429s as IssueOTP", func(t *testing.T) {
		for name, seed := range map[string]struct {
			count    int
			lastSent time.Duration
			want     error
		}{
			"cooldown": {1, -10 * time.Second, domain.ErrOTPResendTooSoon},
			"cap":      {3, -5 * time.Minute, domain.ErrOTPSendLimited},
		} {
			t.Run(name, func(t *testing.T) {
				truncateAll(t, db)
				a, b := "apply@example.com", "issue@example.com"
				insertSendLimit(t, db, a, purpose, seed.count, time.Now().Add(seed.lastSent), nil)
				insertSendLimit(t, db, b, purpose, seed.count, time.Now().Add(seed.lastSent), nil)

				_, errApply := otpRepo.ApplySendLimit(ctx, a, purpose, testSendLimits())
				_, errIssue := otpRepo.IssueOTP(ctx, b, purpose, "h", time.Now().Add(time.Minute), testSendLimits())

				require.ErrorIs(t, errApply, seed.want)
				require.ErrorIs(t, errIssue, seed.want)
				require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM otp_codes`))
			})
		}
	})
}

// TestRegistrationRepository_SendLimits pins the send-policy state machine:
// the cap setting a temporary lock, the cycle resetting once it ages out, a
// refusal not consuming a slot, and the counter being released.
func TestRegistrationRepository_SendLimits(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	limits := testSendLimits()
	reg := domain.OTPPurposeRegister

	t.Run("cap sets a 30m lock and a locked cycle keeps refusing", func(t *testing.T) {
		truncateAll(t, db)
		email := "cap@example.com"
		insertSendLimit(t, db, email, reg, 3, time.Now().Add(-5*time.Minute), nil)

		before := time.Now()
		_, err := otpRepo.IssueOTP(ctx, email, reg, "hash", time.Now().Add(time.Minute), limits)
		limited, ok := errors.AsType[*domain.OTPSendLimitedError](err)
		require.True(t, ok, "got %v", err)
		require.WithinDuration(t, before.Add(30*time.Minute), limited.LockedUntil, time.Minute)

		_, err = otpRepo.IssueOTP(ctx, email, reg, "hash", time.Now().Add(time.Minute), limits)
		require.ErrorIs(t, err, domain.ErrOTPSendLimited)
	})

	t.Run("a cycle older than the lock duration resets to the free first send", func(t *testing.T) {
		truncateAll(t, db)
		email := "reset@example.com"
		insertSendLimit(t, db, email, reg, 3, time.Now().Add(-31*time.Minute), nil)

		state, err := otpRepo.IssueOTP(ctx, email, reg, "hash", time.Now().Add(time.Minute), limits)
		require.NoError(t, err)
		require.Equal(t, 0, state.SendCount)
		require.Equal(t, 3, state.Remaining)
	})

	t.Run("cooldown refusal does not consume a slot", func(t *testing.T) {
		truncateAll(t, db)
		email := "cooldown@example.com"
		insertSendLimit(t, db, email, reg, 1, time.Now().Add(-10*time.Second), nil)

		_, err := otpRepo.IssueOTP(ctx, email, reg, "hash", time.Now().Add(time.Minute), limits)
		require.ErrorIs(t, err, domain.ErrOTPResendTooSoon)
		require.Equal(t, 1, queryInt(t, db, `SELECT send_count FROM otp_send_limits WHERE email = $1`, email))
	})

	t.Run("completing a registration releases the counter", func(t *testing.T) {
		truncateAll(t, db)
		email := "done@example.com"
		insertSendLimit(t, db, email, reg, 2, time.Now().Add(-time.Minute), nil)
		pendingID := insertPending(t, db, "+84900000083", email, time.Now().Add(time.Hour))
		otpID := insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)

		pending := &domain.PendingRegistration{ID: pendingID, PhoneNumber: "+84900000083", Email: email, PasswordHash: "hash"}
		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.NoError(t, err)
		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM otp_send_limits WHERE email = $1`, email))
	})

	t.Run("ReleaseSendLimit drops the counter", func(t *testing.T) {
		truncateAll(t, db)
		email := "release@example.com"
		insertSendLimit(t, db, email, reg, 2, time.Now().Add(-time.Minute), nil)

		require.NoError(t, otpRepo.ReleaseSendLimit(ctx, email, reg))
		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM otp_send_limits WHERE email = $1`, email))
	})
}

// InvalidateActiveOTP retires the live code of one (email, purpose) after its
// email could not be sent. Rows that are already spent, locked, for another
// email or for another purpose stay as they were.
// Vô hiệu mã còn sống của đúng (email, mục đích); không đụng hàng khác.
func TestOTPRepository_InvalidateActiveOTP(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	reg := domain.OTPPurposeRegister
	email := "undelivered@example.com"

	t.Run("marks the live code used", func(t *testing.T) {
		truncateAll(t, db)
		id := insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)

		require.NoError(t, otpRepo.InvalidateActiveOTP(ctx, email, reg))
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE id = $1 AND used_at IS NOT NULL`, id))
	})

	t.Run("leaves other emails and purposes alone", func(t *testing.T) {
		truncateAll(t, db)
		otherEmail := insertOTPRow(t, db, "other@example.com", reg, time.Now().Add(time.Minute), nil, nil)
		otherPurpose := insertOTPRow(t, db, email, domain.OTPPurposeResetPassword, time.Now().Add(time.Minute), nil, nil)

		require.NoError(t, otpRepo.InvalidateActiveOTP(ctx, email, reg))
		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE id IN ($1, $2) AND used_at IS NOT NULL`, otherEmail, otherPurpose))
	})

	t.Run("does not touch a code that is already locked", func(t *testing.T) {
		truncateAll(t, db)
		locked := time.Now().Add(10 * time.Minute)
		id := insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, &locked)

		require.NoError(t, otpRepo.InvalidateActiveOTP(ctx, email, reg))
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE id = $1 AND used_at IS NULL`, id))
	})

	t.Run("no live code is not an error", func(t *testing.T) {
		truncateAll(t, db)
		require.NoError(t, otpRepo.InvalidateActiveOTP(ctx, email, reg))
	})
}

func TestRegistrationRepository_FindPendingRegistration(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		truncateAll(t, db)
		_, err := repo.FindPendingRegistration(ctx, "none@example.com")
		require.ErrorIs(t, err, domain.ErrPendingRegistrationNotFound)
	})

	t.Run("expired", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000031", "e@example.com", time.Now().Add(-time.Second))
		_, err := repo.FindPendingRegistration(ctx, "e@example.com")
		require.ErrorIs(t, err, domain.ErrPendingRegistrationExpired)
	})

	t.Run("found", func(t *testing.T) {
		truncateAll(t, db)
		insertPending(t, db, "+84900000032", "f@example.com", time.Now().Add(time.Minute))
		p, err := repo.FindPendingRegistration(ctx, "f@example.com")
		require.NoError(t, err)
		require.Equal(t, "+84900000032", p.PhoneNumber)
		require.False(t, p.AccountExists)
	})
}

func TestOTPRepository_FindActiveOTP(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	email := "find@example.com"
	reg := domain.OTPPurposeRegister

	t.Run("no row at all", func(t *testing.T) {
		truncateAll(t, db)
		_, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("unused unexpired row is active", func(t *testing.T) {
		truncateAll(t, db)
		insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)
		otp, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.NoError(t, err)
		require.Nil(t, otp.LockedUntil)
	})

	t.Run("used row is not active", func(t *testing.T) {
		truncateAll(t, db)
		usedAt := time.Now()
		insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), &usedAt, nil)
		_, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("expired unused row is not active", func(t *testing.T) {
		truncateAll(t, db)
		insertOTPRow(t, db, email, reg, time.Now().Add(-time.Second), nil, nil)
		_, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("locked row is active even past expires_at, so the caller learns the deadline", func(t *testing.T) {
		truncateAll(t, db)
		lockedUntil := time.Now().Add(15 * time.Minute)
		insertOTPRow(t, db, email, reg, time.Now().Add(-time.Minute), nil, &lockedUntil)
		otp, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.NoError(t, err)
		require.WithinDuration(t, lockedUntil, *otp.LockedUntil, time.Second)
	})

	t.Run("a lifted lock is dead even before expires_at", func(t *testing.T) {
		truncateAll(t, db)
		lockedUntil := time.Now().Add(-time.Minute)
		insertOTPRow(t, db, email, reg, time.Now().Add(5*time.Minute), nil, &lockedUntil)
		_, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
	})

	t.Run("a live lock wins over a newer unlocked row", func(t *testing.T) {
		truncateAll(t, db)
		lockedUntil := time.Now().Add(15 * time.Minute)
		lockedID := insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, &lockedUntil)
		time.Sleep(10 * time.Millisecond)
		insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)
		otp, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.NoError(t, err)
		require.Equal(t, lockedID, otp.ID)
	})

	t.Run("picks the newest row and ignores other emails", func(t *testing.T) {
		truncateAll(t, db)
		insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)
		time.Sleep(10 * time.Millisecond)
		newestID := insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)
		insertOTPRow(t, db, "other@example.com", reg, time.Now().Add(time.Minute), nil, nil)
		otp, err := otpRepo.FindActiveOTP(ctx, email, reg)
		require.NoError(t, err)
		require.Equal(t, newestID, otp.ID)
	})
}

// TestOTPRepository_ConsumeOTPAttempt pins otp-mechanism AC7 / onboarding
// AC15 against real Postgres: the reservation that reaches the cap sets the
// lock, nothing reserves past it, and concurrent guesses cannot exceed it.
func TestOTPRepository_ConsumeOTPAttempt(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	reg := domain.OTPPurposeRegister
	lockUntil := time.Now().Add(30 * time.Minute).Truncate(time.Microsecond)

	t.Run("sequential: third reservation locks, fourth fails", func(t *testing.T) {
		truncateAll(t, db)
		id := insertOTPRow(t, db, "seq@example.com", reg, time.Now().Add(time.Minute), nil, nil)

		for want := 1; want <= 3; want++ {
			n, ok, err := otpRepo.ConsumeOTPAttempt(ctx, id, 3, lockUntil)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, want, n)

			var locked sql.NullTime
			require.NoError(t, db.QueryRow(ctx, `SELECT locked_until FROM otp_codes WHERE id = $1`, id).Scan(&locked))
			require.Equal(t, want == 3, locked.Valid, "lock set exactly by the reservation reaching the cap (attempt %d)", want)
		}

		_, ok, err := otpRepo.ConsumeOTPAttempt(ctx, id, 3, lockUntil)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("a consumed code reserves nothing", func(t *testing.T) {
		truncateAll(t, db)
		usedAt := time.Now()
		id := insertOTPRow(t, db, "used@example.com", reg, time.Now().Add(time.Minute), &usedAt, nil)
		_, ok, err := otpRepo.ConsumeOTPAttempt(ctx, id, 3, lockUntil)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("10 concurrent guesses keep exactly 3", func(t *testing.T) {
		truncateAll(t, db)
		id := insertOTPRow(t, db, "race@example.com", reg, time.Now().Add(time.Minute), nil, nil)

		const guesses = 10
		results := make(chan bool, guesses)
		errs := make(chan error, guesses)
		start := make(chan struct{})
		for range guesses {
			go func() {
				<-start
				_, ok, err := otpRepo.ConsumeOTPAttempt(ctx, id, 3, lockUntil)
				errs <- err
				results <- ok
			}()
		}
		close(start)

		reserved := 0
		for range guesses {
			require.NoError(t, <-errs)
			if <-results {
				reserved++
			}
		}
		require.Equal(t, 3, reserved)
		require.Equal(t, 3, queryInt(t, db, `SELECT attempts FROM otp_codes WHERE id = $1`, id))
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE id = $1 AND locked_until IS NOT NULL`, id))
	})
}

// TestOTPRepository_SendLockDoesNotBlockVerify pins onboarding AC13: a locked
// send budget leaves the last code findable and verifiable.
func TestOTPRepository_SendLockDoesNotBlockVerify(t *testing.T) {
	db := connectTestDB(t)
	otpRepo := NewOTPRepository(db)
	ctx := context.Background()
	email := "sendlock@example.com"
	sendLocked := time.Now().Add(30 * time.Minute)
	insertSendLimit(t, db, email, domain.OTPPurposeRegister, 3, time.Now(), &sendLocked)
	id := insertOTPRow(t, db, email, domain.OTPPurposeRegister, time.Now().Add(time.Minute), nil, nil)

	otp, err := otpRepo.FindActiveOTP(ctx, email, domain.OTPPurposeRegister)
	require.NoError(t, err)
	require.Equal(t, id, otp.ID)
	require.Nil(t, otp.LockedUntil)
	_, ok, err := otpRepo.ConsumeOTPAttempt(ctx, id, 3, time.Now().Add(30*time.Minute))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestRegistrationRepository_CreateUserAndSession(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()
	reg := domain.OTPPurposeRegister

	// seed writes a live pending row and its code, as step 1 would.
	seed := func(t *testing.T, email, phone string) (*domain.PendingRegistration, int64) {
		t.Helper()
		pendingID := insertPending(t, db, phone, email, time.Now().Add(time.Hour))
		otpID := insertOTPRow(t, db, email, reg, time.Now().Add(time.Minute), nil, nil)
		return &domain.PendingRegistration{ID: pendingID, PhoneNumber: phone, Email: email, PasswordHash: "hash"}, otpID
	}

	t.Run("creates verified user, deletes pending, consumes code, opens session", func(t *testing.T) {
		truncateAll(t, db)
		pending, otpID := seed(t, "g@example.com", "+84900000060")

		before := time.Now().Add(-time.Second)
		result, err := repo.CreateUserAndSession(ctx, pending, otpID, 168*time.Hour, stubTokenIssuer)
		require.NoError(t, err)
		require.NotZero(t, result.User.ID)
		require.NotNil(t, result.User.EmailVerifiedAt, "AC14: email_verified_at = now()")
		require.True(t, result.User.EmailVerifiedAt.After(before))
		require.Equal(t, "access-"+strconv.FormatInt(result.User.ID, 10), result.AccessToken)
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM users WHERE id = $1 AND email_verified_at IS NOT NULL`, result.User.ID))

		_, err = repo.FindPendingRegistration(ctx, "g@example.com")
		require.ErrorIs(t, err, domain.ErrPendingRegistrationNotFound)
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE id = $1 AND used_at IS NOT NULL`, otpID))

		var storedHash string
		require.NoError(t, db.QueryRow(ctx, `SELECT refresh_token_hash FROM user_sessions WHERE user_id = $1`, result.User.ID).Scan(&storedHash))
		require.Equal(t, domain.HashRefreshToken(result.RefreshToken), storedHash)
	})

	t.Run("session id handed to the issuer equals the persisted session_id", func(t *testing.T) {
		truncateAll(t, db)
		pending, otpID := seed(t, "sid@example.com", "+84900000065")

		var gotUserID, gotSessionID string
		issuer := func(userID, sessionID string) (string, string, error) {
			gotUserID, gotSessionID = userID, sessionID
			return "access", "refresh", nil
		}
		result, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, issuer)
		require.NoError(t, err)
		require.Equal(t, strconv.FormatInt(result.User.ID, 10), gotUserID)

		var stored string
		require.NoError(t, db.QueryRow(ctx, `SELECT session_id FROM user_sessions WHERE user_id = $1`, result.User.ID).Scan(&stored))
		require.Equal(t, gotSessionID, stored)
	})

	// AC18: two pendings share a phone; whoever verifies first keeps it, the
	// second gets ErrPhoneTaken and its transaction rolls back cleanly.
	t.Run("phone taken at account creation rolls back", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000064"
		first, firstOTP := seed(t, "first@example.com", phone)
		second, secondOTP := seed(t, "second@example.com", phone)

		_, err := repo.CreateUserAndSession(ctx, first, firstOTP, time.Hour, stubTokenIssuer)
		require.NoError(t, err)

		_, err = repo.CreateUserAndSession(ctx, second, secondOTP, time.Hour, stubTokenIssuer)
		require.ErrorIs(t, err, domain.ErrPhoneTaken)

		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM users WHERE email = 'second@example.com'`))
		_, err = repo.FindPendingRegistration(ctx, "second@example.com")
		require.NoError(t, err, "pending row survives the rollback")
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE id = $1 AND used_at IS NULL`, secondOTP), "code not consumed by the rolled-back tx")
	})

	t.Run("a code already consumed creates nothing", func(t *testing.T) {
		truncateAll(t, db)
		pending, otpID := seed(t, "twice@example.com", "+84900000066")
		_, err := db.Exec(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1`, otpID)
		require.NoError(t, err)

		_, err = repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.ErrorIs(t, err, domain.ErrOTPExpired)
		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM users`))
	})

	t.Run("email that gained an account meanwhile reads as a wrong code", func(t *testing.T) {
		truncateAll(t, db)
		pending, otpID := seed(t, "late@example.com", "+84900000067")
		insertUser(t, db, "+84900000068", "late@example.com")

		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.ErrorIs(t, err, domain.ErrOTPInvalid)
		_, err = repo.FindPendingRegistration(ctx, "late@example.com")
		require.NoError(t, err)
	})

	t.Run("token issuer failure rolls back the whole transaction", func(t *testing.T) {
		truncateAll(t, db)
		pending, otpID := seed(t, "h@example.com", "+84900000063")
		failing := func(string, string) (string, string, error) { return "", "", errors.New("signing key unavailable") }

		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, failing)
		require.Error(t, err)
		require.Equal(t, 0, queryInt(t, db, `SELECT count(*) FROM users WHERE email = 'h@example.com'`))
		_, err = repo.FindPendingRegistration(ctx, "h@example.com")
		require.NoError(t, err)
	})
}

func TestRegistrationRepository_DeletePendingRegistration(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	id := insertPending(t, db, "+84900000070", "i@example.com", time.Now().Add(time.Minute))
	require.NoError(t, repo.DeletePendingRegistration(ctx, id))

	_, err := repo.FindPendingRegistration(ctx, "i@example.com")
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

func insertTemple(t *testing.T, db database.Database, id string) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority, latitude, longitude, address)
		 VALUES ($1, $2, 'CHUA', 'north', 'Thiền', 1000, 'CHUA_XEP_HANG', 7, 21.0, 105.8, 'Hà Nội')`, id, "Chùa "+id)
	require.NoError(t, err)
}

func TestRegistrationRepository_TempleExists(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()
	truncateAll(t, db)
	insertTemple(t, db, "temple_test_001")

	ok, err := repo.TempleExists(ctx, "temple_test_001")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = repo.TempleExists(ctx, "temple_missing")
	require.NoError(t, err)
	require.False(t, ok)
}

// TestRegistrationRepository_OrdinationTemple covers AC4/AC8/AC19 at the SQL
// level: the id survives pending -> user, NULL stays NULL, and the FK is what
// turns a vanished temple into ErrInvalidTemple at verify time.
func TestRegistrationRepository_OrdinationTemple(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()

	t.Run("id round-trips pending to user", func(t *testing.T) {
		truncateAll(t, db)
		insertTemple(t, db, "temple_test_002")
		phone := "+84900000070"
		templeID := "temple_test_002"

		pending := &domain.PendingRegistration{
			PhoneNumber: phone, Email: "t1@example.com", PasswordHash: "hash",
			OrdinationTempleID: &templeID, ExpiresAt: time.Now().Add(time.Hour),
		}
		_, err := repo.SavePendingAndIssueOTP(ctx, pending, "otphash", time.Now().Add(90*time.Second), testSendLimits(), nil)
		require.NoError(t, err)

		loaded, err := repo.FindPendingRegistration(ctx, "t1@example.com")
		require.NoError(t, err)
		require.NotNil(t, loaded.OrdinationTempleID)
		require.Equal(t, templeID, *loaded.OrdinationTempleID)

		active, err := NewOTPRepository(db).FindActiveOTP(ctx, "t1@example.com", domain.OTPPurposeRegister)
		require.NoError(t, err)
		otpID := active.ID
		result, err := repo.CreateUserAndSession(ctx, loaded, otpID, time.Hour, stubTokenIssuer)
		require.NoError(t, err)
		require.NotNil(t, result.User.OrdinationTempleID)
		require.Equal(t, templeID, *result.User.OrdinationTempleID)

		var stored sql.NullString
		require.NoError(t, db.QueryRow(ctx, `SELECT ordination_temple_id FROM users WHERE id = $1`, result.User.ID).Scan(&stored))
		require.Equal(t, templeID, stored.String)
	})

	t.Run("nil stays NULL", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000071"
		pending := &domain.PendingRegistration{
			PhoneNumber: phone, Email: "t2@example.com", PasswordHash: "hash",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		_, err := repo.SavePendingAndIssueOTP(ctx, pending, "otphash", time.Now().Add(90*time.Second), testSendLimits(), nil)
		require.NoError(t, err)

		loaded, err := repo.FindPendingRegistration(ctx, "t2@example.com")
		require.NoError(t, err)
		require.Nil(t, loaded.OrdinationTempleID)
	})

	t.Run("temple vanished before verify maps to ErrInvalidTemple and rolls back", func(t *testing.T) {
		truncateAll(t, db)
		phone := "+84900000072"
		missing := "temple_never_existed"
		pendingID := insertPending(t, db, phone, "t3@example.com", time.Now().Add(time.Hour))
		otpID := insertOTPRow(t, db, "t3@example.com", domain.OTPPurposeRegister, time.Now().Add(90*time.Second), nil, nil)

		pending := &domain.PendingRegistration{
			ID: pendingID, PhoneNumber: phone, Email: "t3@example.com", PasswordHash: "hash",
			OrdinationTempleID: &missing,
		}
		_, err := repo.CreateUserAndSession(ctx, pending, otpID, time.Hour, stubTokenIssuer)
		require.ErrorIs(t, err, domain.ErrInvalidTemple)

		var users int
		require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users))
		require.Zero(t, users)
		_, err = repo.FindPendingRegistration(ctx, "t3@example.com")
		require.NoError(t, err, "pending payload must survive the rollback")
	})
}

// AC24: while a live code exists only the same payload may replace the row.
// Always run with TEST_DB_NAME=innerlamp_test.
func TestRegistrationRepository_SavePendingAndIssueOTP_LiveCodeGuard(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRegistrationRepository(db)
	ctx := context.Background()
	deny := func(*domain.PendingRegistration) bool { return false }
	allow := func(*domain.PendingRegistration) bool { return true }

	type snapshot struct {
		phone, hash, otpHash string
		otpRows, limitRows   int
		sendCount            int
	}
	snap := func(t *testing.T, email string) snapshot {
		t.Helper()
		var s snapshot
		require.NoError(t, db.QueryRow(ctx, `SELECT phone_number, password_hash FROM pending_registration WHERE email = $1`, email).Scan(&s.phone, &s.hash))
		require.NoError(t, db.QueryRow(ctx, `SELECT otp_hash FROM otp_codes WHERE email = $1 AND used_at IS NULL`, email).Scan(&s.otpHash))
		s.otpRows = queryInt(t, db, `SELECT count(*) FROM otp_codes WHERE email = $1`, email)
		s.limitRows = queryInt(t, db, `SELECT count(*) FROM otp_send_limits WHERE email = $1`, email)
		s.sendCount = queryInt(t, db, `SELECT send_count FROM otp_send_limits WHERE email = $1`, email)
		return s
	}

	t.Run("different payload with a live code is refused and nothing changes", func(t *testing.T) {
		truncateAll(t, db)
		email := "victim@example.com"
		victim := newPending(email, "+84900000100")
		victim.PasswordHash = "victim-hash"
		_, err := repo.SavePendingAndIssueOTP(ctx, victim, "victim-otp", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		ageSendLimit(t, db, email) // past the cooldown: only the guard can refuse
		before := snap(t, email)

		attacker := newPending(email, "+84900000999")
		attacker.PasswordHash = "attacker-hash"
		var seen *domain.PendingRegistration
		_, err = repo.SavePendingAndIssueOTP(ctx, attacker, "attacker-otp", time.Now().Add(10*time.Minute), testSendLimits(),
			func(existing *domain.PendingRegistration) bool { seen = existing; return false })

		require.ErrorIs(t, err, domain.ErrRegistrationInProgress)
		busy, ok := errors.AsType[*domain.RegistrationInProgressError](err)
		require.True(t, ok)
		require.GreaterOrEqual(t, busy.RetryAfterSeconds, 1)
		require.LessOrEqual(t, busy.RetryAfterSeconds, 600)
		require.NotNil(t, seen)
		require.Equal(t, "victim-hash", seen.PasswordHash)
		require.Equal(t, "+84900000100", seen.PhoneNumber)
		require.Equal(t, before, snap(t, email), "row, codes and send counters are untouched")
	})

	t.Run("same payload replaces the row and issues a new code", func(t *testing.T) {
		truncateAll(t, db)
		email := "same-payload@example.com"
		_, err := repo.SavePendingAndIssueOTP(ctx, newPending(email, "+84900000101"), "otp-1", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		ageSendLimit(t, db, email)

		again := newPending(email, "+84900000101")
		again.PasswordHash = "pw-hash-new"
		_, err = repo.SavePendingAndIssueOTP(ctx, again, "otp-2", time.Now().Add(10*time.Minute), testSendLimits(), allow)
		require.NoError(t, err)

		got := snap(t, email)
		require.Equal(t, "pw-hash-new", got.hash)
		require.Equal(t, "otp-2", got.otpHash)
	})

	// The mailer failed after the code was committed: the usecase retires the
	// undelivered code, and the user's corrected resubmission goes through even
	// though its payload differs. Without InvalidateActiveOTP it would be refused.
	// Mail lỗi -> vô hiệu mã chưa gửi tới -> form sửa lại (nội dung khác) vẫn gửi được.
	t.Run("a different payload is accepted once the undelivered code is invalidated", func(t *testing.T) {
		truncateAll(t, db)
		email := "mail-failed@example.com"
		first := newPending(email, "+84900000102")
		first.PasswordHash = "first-hash"
		_, err := repo.SavePendingAndIssueOTP(ctx, first, "otp-never-delivered", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		ageSendLimit(t, db, email)

		corrected := newPending(email, "+84900000103")
		corrected.PasswordHash = "corrected-hash"
		_, err = repo.SavePendingAndIssueOTP(ctx, corrected, "otp-2", time.Now().Add(10*time.Minute), testSendLimits(), deny)
		require.ErrorIs(t, err, domain.ErrRegistrationInProgress, "precondition: the live code refuses a different payload")

		require.NoError(t, NewOTPRepository(db).InvalidateActiveOTP(ctx, email, domain.OTPPurposeRegister))

		_, err = repo.SavePendingAndIssueOTP(ctx, corrected, "otp-2", time.Now().Add(10*time.Minute), testSendLimits(), deny)
		require.NoError(t, err)
		got := snap(t, email)
		require.Equal(t, "corrected-hash", got.hash)
		require.Equal(t, "+84900000103", got.phone)
		require.Equal(t, "otp-2", got.otpHash)
	})

	t.Run("callback is not consulted when the previous code is expired or used", func(t *testing.T) {
		truncateAll(t, db)
		email := "expired@example.com"
		_, err := repo.SavePendingAndIssueOTP(ctx, newPending(email, "+84900000102"), "otp-1", time.Now().Add(10*time.Minute), testSendLimits(), nil)
		require.NoError(t, err)
		ageSendLimit(t, db, email)
		_, err = db.Exec(ctx, `UPDATE otp_codes SET expires_at = now() - interval '1 minute' WHERE email = $1`, email)
		require.NoError(t, err)

		_, err = repo.SavePendingAndIssueOTP(ctx, newPending(email, "+84900000777"), "otp-2", time.Now().Add(10*time.Minute), testSendLimits(), deny)
		require.NoError(t, err)
		require.Equal(t, "+84900000777", snap(t, email).phone)
	})

	t.Run("no pending row means the callback is never called", func(t *testing.T) {
		truncateAll(t, db)
		_, err := repo.SavePendingAndIssueOTP(ctx, newPending("fresh@example.com", "+84900000103"), "otp", time.Now().Add(10*time.Minute), testSendLimits(),
			func(*domain.PendingRegistration) bool { t.Fatal("must not be called"); return false })
		require.NoError(t, err)
	})

	t.Run("concurrent different payloads leave exactly one whole payload", func(t *testing.T) {
		truncateAll(t, db)
		email := "race@example.com"
		const n = 12
		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			okCount  int
			rejected int
			other    []error
		)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				p := newPending(email, fmt.Sprintf("+8490000%04d", i))
				p.PasswordHash = fmt.Sprintf("hash-%d", i)
				phone, hash := p.PhoneNumber, p.PasswordHash
				_, err := repo.SavePendingAndIssueOTP(ctx, p, fmt.Sprintf("otp-%d", i), time.Now().Add(10*time.Minute), testSendLimits(),
					func(e *domain.PendingRegistration) bool { return e.PhoneNumber == phone && e.PasswordHash == hash })
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					okCount++
				case errors.Is(err, domain.ErrRegistrationInProgress):
					rejected++
				default:
					other = append(other, err)
				}
			}(i)
		}
		wg.Wait()

		require.Empty(t, other, "no failure besides the guard (the first request wins; the rest see a live code)")
		require.Equal(t, 1, okCount, "only the first writer passes while its code is live")
		require.Equal(t, n-1, rejected)
		require.Equal(t, 1, queryInt(t, db, `SELECT count(*) FROM pending_registration WHERE email = $1`, email))
		got := snap(t, email)
		var i int
		_, err := fmt.Sscanf(got.hash, "hash-%d", &i)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("+8490000%04d", i), got.phone, "phone and password hash come from the same request")
		require.Equal(t, fmt.Sprintf("otp-%d", i), got.otpHash)
	})
}
