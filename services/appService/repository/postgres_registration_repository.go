// Package repository holds US-001's Postgres implementation of
// domain.RegistrationRepository — every SQL statement and transaction
// boundary the registration flow touches lives here, and nowhere else.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// otpPurposeRegister is used wherever a statement is hardcoded to the
// register flow (SavePendingAndIssueOTP is only ever called by step 1 of
// US-001). IssueOTP and FindActiveOTP take purpose as a parameter instead,
// since the endpoint they serve is shared with US-002/US-003.
const otpPurposeRegister = domain.OTPPurposeRegister

// Constraint names Postgres assigned to the inline UNIQUE columns in
// migrations/0001_register_onboarding.sql. isUniqueViolation matches on
// these exact names — renaming either constraint in a future migration
// silently turns a race-loser's 409 into a 500, so migrations/README.md
// documents them as a schema invariant that must not move without this file
// moving with it.
const (
	usersEmailKeyConstraint       = "users_email_key"
	usersPhoneNumberKeyConstraint = "users_phone_number_key"
)

type registrationRepository struct {
	db database.Database
}

// The concrete store implements both the register-only port and the generic
// OTP port from one struct. These lines are what make that true at compile
// time: adding a method to either interface without implementing it here fails
// the build rather than failing later, at the first call site.
var (
	_ domain.RegistrationRepository = (*registrationRepository)(nil)
	_ domain.OTPRepository          = (*registrationRepository)(nil)
)

// NewRegistrationRepository returns the Postgres-backed
// domain.RegistrationRepository for US-001.
func NewRegistrationRepository(db database.Database) domain.RegistrationRepository {
	return &registrationRepository{db: db}
}

// NewOTPRepository returns the same Postgres-backed store, typed as the
// generic domain.OTPRepository — the purpose-agnostic OTP mechanism's
// persistence port. It is a second view of the same concrete struct, not a
// separate repository, so the OTP SQL helpers are shared between the two.
func NewOTPRepository(db database.Database) domain.OTPRepository {
	return &registrationRepository{db: db}
}

// FindConflicts answers AC6 in one round trip: three EXISTS subqueries, one
// query, so the answers are consistent with each other rather than each
// reflecting a slightly different instant.
//
// Only a *different* phone's live pending row counts against the email. The
// same phone's own pending row is that caller's in-flight attempt, and
// request-otp overwrites it (v5.1) — counting it would make a reinstall look
// like EMAIL_TAKEN for the whole PendingRegistrationTTL. An expired row counts
// for nothing: it is an abandoned attempt the cleanup job has not removed yet.
func (r *registrationRepository) FindConflicts(ctx context.Context, phoneNumber, email string) (*domain.RegistrationConflicts, error) {
	const query = `
		SELECT
			EXISTS(SELECT 1 FROM users WHERE phone_number = $1) AS phone_in_users,
			EXISTS(SELECT 1 FROM users WHERE email = $2) AS email_in_users,
			EXISTS(
				SELECT 1 FROM pending_registration
				WHERE email = $2 AND phone_number <> $1 AND expires_at > now()
			) AS email_in_pending
	`

	var conflicts domain.RegistrationConflicts
	row := r.db.QueryRow(ctx, query, phoneNumber, email)
	if err := row.Scan(
		&conflicts.PhoneInUsers,
		&conflicts.EmailInUsers,
		&conflicts.EmailInPending,
	); err != nil {
		return nil, fmt.Errorf("find conflicts: %w", err)
	}

	return &conflicts, nil
}

// SavePendingAndIssueOTP is step 1's write: apply the send policy, UPSERT the
// payload, invalidate whatever OTP was active, and insert a fresh one — all in
// one transaction, per design.md flow 1 Step 4.
//
// The send policy runs FIRST: a refused attempt must leave no pending row and
// no OTP behind, and must not consume a send slot. Its typed errors
// (*OTPResendTooSoonError / *OTPSendLimitedError) are returned as-is for the
// use case to hand to delivery.
func (r *registrationRepository) SavePendingAndIssueOTP(ctx context.Context, pending *domain.PendingRegistration, otpHash string, expiresAt time.Time, limits domain.SendLimits) (*domain.SendState, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	state, err := applySendLimits(ctx, tx, pending.PhoneNumber, otpPurposeRegister, limits, time.Now())
	if err != nil {
		return nil, err
	}

	const upsertPending = `
		INSERT INTO pending_registration
			(phone_number, email, password_hash, dharma_name, ordination_place, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (phone_number) DO UPDATE SET
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			dharma_name = EXCLUDED.dharma_name,
			ordination_place = EXCLUDED.ordination_place,
			expires_at = EXCLUDED.expires_at,
			created_at = now()
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, upsertPending,
		pending.PhoneNumber,
		pending.Email,
		pending.PasswordHash,
		nullableString(pending.DharmaName),
		nullableString(pending.OrdinationPlace),
		pending.ExpiresAt,
	).Scan(&pending.ID, &pending.CreatedAt); err != nil {
		return nil, fmt.Errorf("upsert pending registration: %w", err)
	}

	if err := invalidateActiveOTP(ctx, tx, pending.PhoneNumber, otpPurposeRegister); err != nil {
		return nil, err
	}

	if err := insertOTP(ctx, tx, pending.PhoneNumber, otpPurposeRegister, otpHash, expiresAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit save pending and issue otp: %w", err)
	}
	return state, nil
}

// IssueOTP is the shared OTP send flow's write: apply the send policy,
// invalidate the active OTP, and insert its replacement — design.md flow for
// Issue OTP Step 2.
//
// Like SavePendingAndIssueOTP the send policy runs first and inside this
// transaction, so a refused send writes nothing and does not burn a slot.
// It does NOT touch pending_registration: the payload request-otp wrote is left
// as it is, and its own (much longer) TTL bounds the attempt. "Issue" rather
// than "reissue" because this is the generic path — a future feature's first
// send goes through here too, and resend is just one more issue.
func (r *registrationRepository) IssueOTP(ctx context.Context, phoneNumber string, purpose domain.OTPPurpose, otpHash string, expiresAt time.Time, limits domain.SendLimits) (*domain.SendState, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	state, err := applySendLimits(ctx, tx, phoneNumber, purpose, limits, time.Now())
	if err != nil {
		return nil, err
	}

	if err := invalidateActiveOTP(ctx, tx, phoneNumber, purpose); err != nil {
		return nil, err
	}

	if err := insertOTP(ctx, tx, phoneNumber, purpose, otpHash, expiresAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit issue otp: %w", err)
	}
	return state, nil
}

// FindPendingRegistration loads the payload for phoneNumber. A missing row
// and an expired row are both reported so the use case can treat "start over
// from step 1" as one branch — see domain.RegistrationRepository's doc.
func (r *registrationRepository) FindPendingRegistration(ctx context.Context, phoneNumber string) (*domain.PendingRegistration, error) {
	const query = `
		SELECT id, phone_number, email, password_hash, dharma_name, ordination_place, expires_at, created_at
		FROM pending_registration
		WHERE phone_number = $1
	`

	var (
		p               domain.PendingRegistration
		dharmaName      sql.NullString
		ordinationPlace sql.NullString
	)
	row := r.db.QueryRow(ctx, query, phoneNumber)
	if err := row.Scan(&p.ID, &p.PhoneNumber, &p.Email, &p.PasswordHash, &dharmaName, &ordinationPlace, &p.ExpiresAt, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPendingRegistrationNotFound
		}
		return nil, fmt.Errorf("find pending registration: %w", err)
	}
	p.DharmaName = nullStringToPtr(dharmaName)
	p.OrdinationPlace = nullStringToPtr(ordinationPlace)

	if !p.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrPendingRegistrationExpired
	}

	return &p, nil
}

// FindActiveOTP loads the newest row for phoneNumber/purpose that a verify or
// resend call should act on. A row counts as active when either:
//   - it is unused and not yet past its own expiry (the ordinary case), or
//   - its lock is still in force (locked_until > now()) — so a caller mid
//     lockout still learns the deadline instead of a generic "expired".
//
// Once a lock lifts, the row it locked (which also carries used_at) drops
// out of both clauses and this returns ErrOTPInvalid — matching design.md's
// "hết 30 phút... phải gọi lại request-otp": a lifted lock does not make the
// old code usable again, it just stops blocking a fresh one.
func (r *registrationRepository) FindActiveOTP(ctx context.Context, phoneNumber string, purpose domain.OTPPurpose) (*domain.OTPCode, error) {
	const query = `
		SELECT id, phone_number, purpose, otp_hash, attempts, locked_until, expires_at, used_at, created_at
		FROM otp_codes
		WHERE phone_number = $1
		  AND purpose = $2
		  AND (locked_until > now() OR (used_at IS NULL AND expires_at > now()))
		ORDER BY created_at DESC
		LIMIT 1
	`

	otp, err := scanOTPRow(r.db.QueryRow(ctx, query, phoneNumber, string(purpose)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOTPInvalid
		}
		return nil, fmt.Errorf("find active otp: %w", err)
	}
	return otp, nil
}

// IncrementOTPAttempts records one wrong guess and returns the new total —
// AC4/AC12. The use case decides at what total to lock; this only counts.
func (r *registrationRepository) IncrementOTPAttempts(ctx context.Context, otpID int64) (int, error) {
	const query = `UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts`

	var attempts int
	if err := r.db.QueryRow(ctx, query, otpID).Scan(&attempts); err != nil {
		return 0, fmt.Errorf("increment otp attempts: %w", err)
	}
	return attempts, nil
}

// LockOTP marks otpID locked until lockedUntil and simultaneously used, so
// the row can never be replayed — AC12.
func (r *registrationRepository) LockOTP(ctx context.Context, otpID int64, lockedUntil time.Time) error {
	const query = `UPDATE otp_codes SET locked_until = $2, used_at = now() WHERE id = $1`

	if _, err := r.db.Exec(ctx, query, otpID, lockedUntil); err != nil {
		return fmt.Errorf("lock otp: %w", err)
	}
	return nil
}

// CreateUserAndSession is step 2's write: re-check the email, insert the
// user, drop the pending payload, consume the OTP, mint tokens inside the
// transaction via issueTokens, and open the first session — all one
// transaction, per design.md flow 2 Step 5-6.
func (r *registrationRepository) CreateUserAndSession(ctx context.Context, pending *domain.PendingRegistration, otpID int64, sessionTTL time.Duration, issueTokens domain.TokenIssuer) (*domain.RegisterResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Re-check inside the transaction: pending_registration.email is
	// deliberately not unique, so two people can both clear FindConflicts at
	// step 1 for the same address. Only this check, backed by the real
	// UNIQUE constraint below, is authoritative.
	var emailExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, pending.Email).Scan(&emailExists); err != nil {
		return nil, fmt.Errorf("recheck email: %w", err)
	}
	if emailExists {
		return nil, domain.ErrEmailTaken
	}

	const insertUser = `
		INSERT INTO users (email, email_verified_at, phone_number, password_hash, dharma_name, ordination_place)
		VALUES ($1, NULL, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	var user domain.User
	user.Email = pending.Email
	user.PhoneNumber = pending.PhoneNumber
	user.PasswordHash = pending.PasswordHash
	user.DharmaName = pending.DharmaName
	user.OrdinationPlace = pending.OrdinationPlace

	err = tx.QueryRowContext(ctx, insertUser,
		pending.Email,
		pending.PhoneNumber,
		pending.PasswordHash,
		nullableString(pending.DharmaName),
		nullableString(pending.OrdinationPlace),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err, usersEmailKeyConstraint) {
			return nil, domain.ErrEmailTaken
		}
		if isUniqueViolation(err, usersPhoneNumberKeyConstraint) {
			return nil, domain.ErrPhoneTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM pending_registration WHERE id = $1`, pending.ID); err != nil {
		return nil, fmt.Errorf("delete pending registration: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1`, otpID); err != nil {
		return nil, fmt.Errorf("consume otp: %w", err)
	}

	// The registration is complete, so the send counter has done its job: drop
	// it in the same transaction rather than leave rate-limit state for a phone
	// that now has an account. Correctness does not depend on this — the counter
	// also ages out — but a fresh registration must not inherit a stale one.
	if _, err := tx.ExecContext(ctx, `DELETE FROM otp_send_limits WHERE phone_number = $1 AND purpose = $2`,
		pending.PhoneNumber, string(otpPurposeRegister)); err != nil {
		return nil, fmt.Errorf("release otp send limit: %w", err)
	}

	// The session ID must exist before minting: the access token carries it as
	// claim "sid" and the very same value is stored below.
	sessionID := uuid.NewString()

	accessToken, refreshToken, err := issueTokens(strconv.FormatInt(user.ID, 10), sessionID)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	const insertSession = `
		INSERT INTO user_sessions (user_id, session_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.ExecContext(ctx, insertSession,
		user.ID,
		sessionID,
		domain.HashRefreshToken(refreshToken),
		time.Now().Add(sessionTTL),
	); err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create user and session: %w", err)
	}

	return &domain.RegisterResult{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// DeletePendingRegistration drops the payload for pendingID.
func (r *registrationRepository) DeletePendingRegistration(ctx context.Context, pendingID int64) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM pending_registration WHERE id = $1`, pendingID); err != nil {
		return fmt.Errorf("delete pending registration: %w", err)
	}
	return nil
}

// ReleaseSendLimit drops the send counter for phoneNumber/purpose. Exported so
// a caller can clear it outside CreateUserAndSession's transaction; the
// production verify path calls the same DELETE inline, inside its own tx.
func (r *registrationRepository) ReleaseSendLimit(ctx context.Context, phoneNumber string, purpose domain.OTPPurpose) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM otp_send_limits WHERE phone_number = $1 AND purpose = $2`,
		phoneNumber, string(purpose)); err != nil {
		return fmt.Errorf("release otp send limit: %w", err)
	}
	return nil
}

// invalidateActiveOTP marks every unused row for phoneNumber/purpose as used,
// without touching rows already locked or spent. Shared by
// SavePendingAndIssueOTP (flow 1 Step 4(d)) and IssueOTP (Issue OTP Step 2(c))
// — both invalidate exactly the same way, so this is written once.
func invalidateActiveOTP(ctx context.Context, tx *sql.Tx, phoneNumber string, purpose domain.OTPPurpose) error {
	const query = `
		UPDATE otp_codes SET used_at = now()
		WHERE phone_number = $1 AND purpose = $2 AND used_at IS NULL
	`
	if _, err := tx.ExecContext(ctx, query, phoneNumber, string(purpose)); err != nil {
		return fmt.Errorf("invalidate active otp: %w", err)
	}
	return nil
}

// applySendLimits atomically applies the per-phone send policy (AC5) for one
// send attempt and, when the send is allowed, records it. It must run inside
// the transaction that also writes the OTP row, so a refused attempt writes
// nothing and an allowed one cannot "consume a slot without sending a code".
//
// The row is upserted then locked FOR UPDATE, which serialises two concurrent
// sends for the same number — without it both could read SendCount=2 and both
// write 3, and the cap would leak a send.
//
// Branches, in the order design.md fixes:
//
//	(0) no row yet, or the cycle aged out (LastSentAt older than LockDuration)
//	    — the send is the free first of a cycle: SendCount stays 0.
//	(a) the send lock is in force         -> *OTPSendLimitedError
//	(b) inside the cooldown               -> *OTPResendTooSoonError
//	(c) the cap is already reached        -> set the lock, *OTPSendLimitedError
//	(d) otherwise count it; reaching the cap sets the lock but this send goes out.
func applySendLimits(ctx context.Context, tx *sql.Tx, phoneNumber string, purpose domain.OTPPurpose, limits domain.SendLimits, now time.Time) (*domain.SendState, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO otp_send_limits (phone_number, purpose) VALUES ($1, $2)
		ON CONFLICT (phone_number, purpose) DO NOTHING`, phoneNumber, string(purpose)); err != nil {
		return nil, fmt.Errorf("ensure otp send limit: %w", err)
	}

	var (
		sendCount   int
		lastSentAt  sql.NullTime
		lockedUntil sql.NullTime
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT send_count, last_sent_at, locked_until
		FROM otp_send_limits
		WHERE phone_number = $1 AND purpose = $2
		FOR UPDATE`, phoneNumber, string(purpose)).Scan(&sendCount, &lastSentAt, &lockedUntil); err != nil {
		return nil, fmt.Errorf("lock otp send limit: %w", err)
	}

	// (0) first send of a cycle is free.
	if !lastSentAt.Valid || !now.Before(lastSentAt.Time.Add(limits.LockDuration)) {
		if err := updateSendLimit(ctx, tx, phoneNumber, purpose, 0, now, nil); err != nil {
			return nil, err
		}
		return &domain.SendState{SendCount: 0, Remaining: limits.MaxAttempts}, nil
	}

	// (a) send lock in force.
	if lockedUntil.Valid && lockedUntil.Time.After(now) {
		return nil, &domain.OTPSendLimitedError{LockedUntil: lockedUntil.Time}
	}

	// (b) cooldown.
	if elapsed := now.Sub(lastSentAt.Time); elapsed < limits.Cooldown {
		return nil, &domain.OTPResendTooSoonError{
			RetryAfterSeconds: int(math.Ceil((limits.Cooldown - elapsed).Seconds())),
		}
	}

	// (c) cap already reached: set the lock and refuse. LastSentAt is left
	// untouched so a refused attempt cannot push the cycle's reset out.
	if sendCount >= limits.MaxAttempts {
		lockUntil := now.Add(limits.LockDuration)
		if err := updateSendLimit(ctx, tx, phoneNumber, purpose, sendCount, lastSentAt.Time, &lockUntil); err != nil {
			return nil, err
		}
		return nil, &domain.OTPSendLimitedError{LockedUntil: lockUntil}
	}

	// (d) allow and count; reaching the cap also sets the lock, but this send
	// still goes out.
	sendCount++
	var lockUntil *time.Time
	if sendCount >= limits.MaxAttempts {
		t := now.Add(limits.LockDuration)
		lockUntil = &t
	}
	if err := updateSendLimit(ctx, tx, phoneNumber, purpose, sendCount, now, lockUntil); err != nil {
		return nil, err
	}
	remaining := limits.MaxAttempts - sendCount
	if remaining < 0 {
		remaining = 0
	}
	return &domain.SendState{SendCount: sendCount, Remaining: remaining}, nil
}

// updateSendLimit writes one otp_send_limits row's counter columns. A nil
// lockedUntil clears the lock; a nil lastSentAt is never passed, so the column
// always holds a real instant once a cycle has started.
func updateSendLimit(ctx context.Context, tx *sql.Tx, phoneNumber string, purpose domain.OTPPurpose, sendCount int, lastSentAt time.Time, lockedUntil *time.Time) error {
	const query = `
		UPDATE otp_send_limits
		SET send_count = $3, last_sent_at = $4, locked_until = $5, updated_at = now()
		WHERE phone_number = $1 AND purpose = $2
	`
	if _, err := tx.ExecContext(ctx, query, phoneNumber, string(purpose), sendCount, lastSentAt, lockedUntil); err != nil {
		return fmt.Errorf("update otp send limit: %w", err)
	}
	return nil
}

// insertOTP writes a fresh otp_codes row. The row carries no send counter — the
// send rate limit lives in otp_send_limits — so there is nothing to pass back.
func insertOTP(ctx context.Context, tx *sql.Tx, phoneNumber string, purpose domain.OTPPurpose, otpHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO otp_codes (phone_number, purpose, otp_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.ExecContext(ctx, query, phoneNumber, string(purpose), otpHash, expiresAt); err != nil {
		return fmt.Errorf("insert otp: %w", err)
	}
	return nil
}

// otpRowScanner is satisfied by both *sql.Row and *sql.Row-returning helpers,
// so scanOTPRow works whether the caller is reading through the plain
// database.Database or a *sql.Tx.
type otpRowScanner interface {
	Scan(dest ...any) error
}

// scanOTPRow reads one otp_codes row, converting its nullable columns.
func scanOTPRow(row otpRowScanner) (*domain.OTPCode, error) {
	var (
		otp         domain.OTPCode
		purpose     string
		lockedUntil sql.NullTime
		usedAt      sql.NullTime
	)
	if err := row.Scan(&otp.ID, &otp.PhoneNumber, &purpose, &otp.OTPHash, &otp.Attempts, &lockedUntil, &otp.ExpiresAt, &usedAt, &otp.CreatedAt); err != nil {
		return nil, err
	}
	otp.Purpose = domain.OTPPurpose(purpose)
	if lockedUntil.Valid {
		otp.LockedUntil = &lockedUntil.Time
	}
	if usedAt.Valid {
		otp.UsedAt = &usedAt.Time
	}
	return &otp, nil
}

// nullableString turns a nil or empty pointer into a SQL NULL — AC9: a blank
// profile field must read back as absent, not as an empty string.
func nullableString(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

// nullStringToPtr is nullableString's inverse for reading a row back.
func nullStringToPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// The digest stored in user_sessions.refresh_token_hash is computed by
// domain.HashRefreshToken, not by a helper here. Two layers must agree on it
// byte for byte — this file writes the column and US-002's use case has to
// hash a presented token before it can look a session up — and a private copy
// on each side would be two definitions of the same security contract. See
// domain/session_token.go.

// isUniqueViolation reports whether err is a Postgres unique-violation (SQL
// state 23505) against the named constraint. Matching by constraint name,
// not by message text, is what lets CreateUserAndSession tell an email
// collision from a phone collision.
func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23505" && pqErr.Constraint == constraint
}
