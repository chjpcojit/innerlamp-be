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
// US-001). The OTPRepository methods take purpose as a parameter instead,
// since password reset shares them.
const otpPurposeRegister = domain.OTPPurposeRegister

// Constraint names Postgres assigned to the inline UNIQUE columns in
// migrations/0001_register_onboarding.sql. isUniqueViolation matches on
// these exact names — renaming either constraint in a future migration
// silently turns a race-loser's 409 into a 500, so migrations/README.md
// documents them as a schema invariant that must not move without this file
// moving with it.
//
// users_email_key can only fire if an account appeared for the email after
// request-otp saw none (pending_registration is unique per email, so no second
// registration can race it); it is reported as ErrOTPInvalid, the same answer
// an account_exists row gets (onboarding AC16).
const (
	usersEmailKeyConstraint       = "users_email_key"
	usersPhoneNumberKeyConstraint = "users_phone_number_key"

	// Default Postgres names for the ordination_temple_id foreign keys (v7.0).
	usersOrdinationTempleFKConstraint   = "users_ordination_temple_id_fkey"
	pendingOrdinationTempleFKConstraint = "pending_registration_ordination_temple_id_fkey"
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

// PhoneTaken reports whether an account already holds phoneNumber (AC7). Only
// users counts: pending_registration.phone_number is not unique since 0037.
func (r *registrationRepository) PhoneTaken(ctx context.Context, phoneNumber string) (bool, error) {
	var taken bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1)`, phoneNumber).Scan(&taken); err != nil {
		return false, fmt.Errorf("check phone taken: %w", err)
	}
	return taken, nil
}

// EmailHasAccount reports whether an account already uses email (Step R-5).
// users.email is stored lower-case (chk_users_email_lowercase), so the caller's
// normalised value matches exactly.
func (r *registrationRepository) EmailHasAccount(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email has account: %w", err)
	}
	return exists, nil
}

// SavePendingAndIssueOTP is step 1's write: apply the send policy, UPSERT the
// payload on email, invalidate whatever code was active, and insert a fresh
// one — all in one transaction (onboarding design Step R-7).
//
// The send policy runs FIRST: a refused attempt must leave no pending row and
// no OTP behind, and must not consume a send slot. Its typed errors
// (*OTPResendTooSoonError / *OTPSendLimitedError) are returned as-is for the
// use case to hand to delivery.
func (r *registrationRepository) SavePendingAndIssueOTP(ctx context.Context, pending *domain.PendingRegistration, otpHash string, expiresAt time.Time, limits domain.SendLimits, canReplace func(existing *domain.PendingRegistration) bool) (*domain.SendState, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Serialise every request for this email, including the first one (no row
	// to FOR UPDATE yet): without it two concurrent first requests would both
	// pass the check and the later upsert would overwrite the earlier payload.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "pending_registration:"+pending.Email); err != nil {
		return nil, fmt.Errorf("lock pending registration: %w", err)
	}

	// AC24: a live code means someone is mid-registration for this email; only
	// the same payload may replace it. Runs before the send policy so a
	// rejected request consumes nothing and leaves no trace.
	if err := guardLivePending(ctx, tx, pending.Email, canReplace); err != nil {
		return nil, err
	}

	state, err := applySendLimits(ctx, tx, pending.Email, otpPurposeRegister, limits, time.Now())
	if err != nil {
		return nil, err
	}

	const upsertPending = `
		INSERT INTO pending_registration
			(email, phone_number, password_hash, full_name, dharma_name, ordination_temple_id, account_exists, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (email) DO UPDATE SET
			phone_number = EXCLUDED.phone_number,
			password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			dharma_name = EXCLUDED.dharma_name,
			ordination_temple_id = EXCLUDED.ordination_temple_id,
			account_exists = EXCLUDED.account_exists,
			expires_at = EXCLUDED.expires_at,
			created_at = now()
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, upsertPending,
		pending.Email,
		pending.PhoneNumber,
		pending.PasswordHash,
		nullableString(pending.FullName),
		nullableString(pending.DharmaName),
		nullableString(pending.OrdinationTempleID),
		pending.AccountExists,
		pending.ExpiresAt,
	).Scan(&pending.ID, &pending.CreatedAt); err != nil {
		// A temple deleted between request-otp's TempleExists and this write.
		if isForeignKeyViolation(err, pendingOrdinationTempleFKConstraint) {
			return nil, domain.ErrInvalidTemple
		}
		return nil, fmt.Errorf("upsert pending registration: %w", err)
	}

	if err := invalidateActiveOTP(ctx, tx, pending.Email, otpPurposeRegister); err != nil {
		return nil, err
	}

	if err := insertOTP(ctx, tx, pending.Email, otpPurposeRegister, otpHash, expiresAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit save pending and issue otp: %w", err)
	}
	return state, nil
}

// guardLivePending implements AC24 inside the caller's transaction: lock the
// pending row, and when a live register code exists refuse the write unless
// canReplace approves the stored payload.
func guardLivePending(ctx context.Context, tx *sql.Tx, email string, canReplace func(existing *domain.PendingRegistration) bool) error {
	if canReplace == nil {
		return nil
	}
	var (
		existing   domain.PendingRegistration
		fullName   sql.NullString
		dharmaName sql.NullString
		templeID   sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
		SELECT id, email, phone_number, password_hash, full_name, dharma_name, ordination_temple_id, account_exists, expires_at, created_at
		FROM pending_registration WHERE email = $1 FOR UPDATE`, email).Scan(
		&existing.ID, &existing.Email, &existing.PhoneNumber, &existing.PasswordHash,
		&fullName, &dharmaName, &templeID, &existing.AccountExists, &existing.ExpiresAt, &existing.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock pending registration row: %w", err)
	}
	existing.FullName = nullStringToPtr(fullName)
	existing.DharmaName = nullStringToPtr(dharmaName)
	existing.OrdinationTempleID = nullStringToPtr(templeID)

	var liveUntil sql.NullTime
	if err := tx.QueryRowContext(ctx, `
		SELECT max(expires_at) FROM otp_codes
		WHERE email = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()`,
		email, otpPurposeRegister).Scan(&liveUntil); err != nil {
		return fmt.Errorf("check live otp: %w", err)
	}
	if !liveUntil.Valid || canReplace(&existing) {
		return nil
	}
	secs := int(math.Ceil(time.Until(liveUntil.Time).Seconds()))
	if secs < 1 {
		secs = 1
	}
	return &domain.RegistrationInProgressError{RetryAfterSeconds: secs}
}

// IssueOTP is the shared OTP issue flow's write: apply the send policy,
// invalidate the active code, and insert its replacement (otp-mechanism
// Step I-2).
//
// Like SavePendingAndIssueOTP the send policy runs first and inside this
// transaction, so a refused send writes nothing and does not burn a slot.
// It does NOT touch pending_registration: the payload request-otp wrote is left
// as it is, and its own (much longer) TTL bounds the attempt. "Issue" rather
// than "reissue" because this is the generic path — a future feature's first
// send goes through here too, and resend is just one more issue.
func (r *registrationRepository) IssueOTP(ctx context.Context, email string, purpose domain.OTPPurpose, otpHash string, expiresAt time.Time, limits domain.SendLimits) (*domain.SendState, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	state, err := applySendLimits(ctx, tx, email, purpose, limits, time.Now())
	if err != nil {
		return nil, err
	}

	if err := invalidateActiveOTP(ctx, tx, email, purpose); err != nil {
		return nil, err
	}

	if err := insertOTP(ctx, tx, email, purpose, otpHash, expiresAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit issue otp: %w", err)
	}
	return state, nil
}

// ApplySendLimit is IssueOTP without the code: the same send policy in its own
// transaction, nothing else written (otp-mechanism AC6).
func (r *registrationRepository) ApplySendLimit(ctx context.Context, email string, purpose domain.OTPPurpose, limits domain.SendLimits) (*domain.SendState, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	state, err := applySendLimits(ctx, tx, email, purpose, limits, time.Now())
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit apply send limit: %w", err)
	}
	return state, nil
}

// FindPendingRegistration loads the payload for email. A missing row and an
// expired row are both reported so the use case can treat "start over from
// step 1" as one branch — see domain.RegistrationRepository's doc.
func (r *registrationRepository) FindPendingRegistration(ctx context.Context, email string) (*domain.PendingRegistration, error) {
	const query = `
		SELECT id, phone_number, email, password_hash, full_name, dharma_name, ordination_temple_id, account_exists, expires_at, created_at
		FROM pending_registration
		WHERE email = $1
	`

	var (
		p                  domain.PendingRegistration
		fullName           sql.NullString
		dharmaName         sql.NullString
		ordinationTempleID sql.NullString
	)
	row := r.db.QueryRow(ctx, query, email)
	if err := row.Scan(&p.ID, &p.PhoneNumber, &p.Email, &p.PasswordHash, &fullName, &dharmaName, &ordinationTempleID, &p.AccountExists, &p.ExpiresAt, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPendingRegistrationNotFound
		}
		return nil, fmt.Errorf("find pending registration: %w", err)
	}
	p.FullName = nullStringToPtr(fullName)
	p.DharmaName = nullStringToPtr(dharmaName)
	p.OrdinationTempleID = nullStringToPtr(ordinationTempleID)

	if !p.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrPendingRegistrationExpired
	}

	return &p, nil
}

// FindActiveOTP loads the newest row for email/purpose that a verify should
// act on. A row counts as active when either:
//   - its wrong-guess lock is still in force (locked_until > now()) — so a
//     caller mid-lockout learns the deadline instead of a generic "expired"; or
//   - it is unused, unexpired, and was never locked.
//
// Once a lock lifts, the row it locked drops out of both clauses (its
// locked_until is set but past) and this returns ErrOTPInvalid: a lifted lock
// does not make the old code usable again, the caller requests a new one.
//
// Locked rows sort first so a code issued after a lock (which EnsureNotLocked
// should already prevent) can never hide the lock.
func (r *registrationRepository) FindActiveOTP(ctx context.Context, email string, purpose domain.OTPPurpose) (*domain.OTPCode, error) {
	const query = `
		SELECT id, email, purpose, otp_hash, attempts, locked_until, expires_at, used_at, created_at
		FROM otp_codes
		WHERE email = $1
		  AND purpose = $2
		  AND (locked_until > now()
		       OR (used_at IS NULL AND locked_until IS NULL AND expires_at > now()))
		ORDER BY (locked_until IS NOT NULL) DESC, created_at DESC
		LIMIT 1
	`

	otp, err := scanOTPRow(r.db.QueryRow(ctx, query, email, string(purpose)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOTPInvalid
		}
		return nil, fmt.Errorf("find active otp: %w", err)
	}
	return otp, nil
}

// ConsumeOTPAttempt reserves one verify attempt in a single statement
// (otp-mechanism Step V-3): the row lock Postgres takes for the UPDATE
// serialises concurrent guesses, and "attempts < max" in the WHERE clause
// means at most maxAttempts reservations ever succeed. The reservation that
// reaches maxAttempts sets locked_until in the same statement. No row back
// means nothing left to reserve (cap reached, or the code was consumed).
func (r *registrationRepository) ConsumeOTPAttempt(ctx context.Context, otpID int64, maxAttempts int, lockUntil time.Time) (int, bool, error) {
	const query = `
		UPDATE otp_codes
		SET attempts = attempts + 1,
		    locked_until = CASE WHEN attempts + 1 >= $2 THEN $3 ELSE locked_until END
		WHERE id = $1 AND used_at IS NULL AND attempts < $2
		RETURNING attempts
	`

	var attempts int
	if err := r.db.QueryRow(ctx, query, otpID, maxAttempts, lockUntil).Scan(&attempts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("consume otp attempt: %w", err)
	}
	return attempts, true, nil
}

// CreateUserAndSession is step 2's write: consume the code, insert the user
// with email_verified_at = now(), drop the pending payload and the send
// counter, mint tokens inside the transaction via issueTokens, and open the
// first session — all one transaction (onboarding design Step V-5).
func (r *registrationRepository) CreateUserAndSession(ctx context.Context, pending *domain.PendingRegistration, otpID int64, sessionTTL time.Duration, issueTokens domain.TokenIssuer) (*domain.RegisterResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Consume the code first. "used_at IS NULL" makes two concurrent verifies
	// of the same right code serialise on this row: the second one updates
	// nothing and stops here, before it could create a second account.
	consumed, err := tx.ExecContext(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1 AND used_at IS NULL`, otpID)
	if err != nil {
		return nil, fmt.Errorf("consume otp: %w", err)
	}
	if n, err := consumed.RowsAffected(); err != nil {
		return nil, fmt.Errorf("consume otp: %w", err)
	} else if n == 0 {
		return nil, domain.ErrOTPExpired
	}

	const insertUser = `
		INSERT INTO users (email, email_verified_at, phone_number, password_hash, full_name, dharma_name, ordination_temple_id)
		VALUES ($1, now(), $2, $3, $4, $5, $6)
		RETURNING id, email_verified_at, created_at, updated_at
	`
	var (
		user            domain.User
		emailVerifiedAt time.Time
	)
	user.Email = pending.Email
	user.PhoneNumber = pending.PhoneNumber
	user.PasswordHash = pending.PasswordHash
	user.FullName = pending.FullName
	user.DharmaName = pending.DharmaName
	user.OrdinationTempleID = pending.OrdinationTempleID

	err = tx.QueryRowContext(ctx, insertUser,
		pending.Email,
		pending.PhoneNumber,
		pending.PasswordHash,
		nullableString(pending.FullName),
		nullableString(pending.DharmaName),
		nullableString(pending.OrdinationTempleID),
	).Scan(&user.ID, &emailVerifiedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err, usersEmailKeyConstraint) {
			return nil, domain.ErrOTPInvalid
		}
		if isUniqueViolation(err, usersPhoneNumberKeyConstraint) {
			return nil, domain.ErrPhoneTaken
		}
		if isForeignKeyViolation(err, usersOrdinationTempleFKConstraint) {
			return nil, domain.ErrInvalidTemple
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	user.EmailVerifiedAt = &emailVerifiedAt

	if _, err := tx.ExecContext(ctx, `DELETE FROM pending_registration WHERE id = $1`, pending.ID); err != nil {
		return nil, fmt.Errorf("delete pending registration: %w", err)
	}

	// The registration is complete, so the send counter has done its job: drop
	// it in the same transaction rather than leave rate-limit state for an
	// email that now has an account. Correctness does not depend on this — the
	// counter also ages out.
	if _, err := tx.ExecContext(ctx, `DELETE FROM otp_send_limits WHERE email = $1 AND purpose = $2`,
		pending.Email, string(otpPurposeRegister)); err != nil {
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

// ReleaseSendLimit drops the send counter for email/purpose, so a send whose
// email failed does not count (otp-mechanism AC10). The verify path runs the
// same DELETE inline, inside its own transaction.
func (r *registrationRepository) ReleaseSendLimit(ctx context.Context, email string, purpose domain.OTPPurpose) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM otp_send_limits WHERE email = $1 AND purpose = $2`,
		email, string(purpose)); err != nil {
		return fmt.Errorf("release otp send limit: %w", err)
	}
	return nil
}

// invalidateActiveOTP marks every unused row for email/purpose as used,
// without touching rows already locked or spent. Shared by
// SavePendingAndIssueOTP (flow 1 Step 4(d)) and IssueOTP (Issue OTP Step 2(c))
// — both invalidate exactly the same way, so this is written once.
func invalidateActiveOTP(ctx context.Context, tx *sql.Tx, email string, purpose domain.OTPPurpose) error {
	const query = `
		UPDATE otp_codes SET used_at = now()
		WHERE email = $1 AND purpose = $2 AND used_at IS NULL
	`
	if _, err := tx.ExecContext(ctx, query, email, string(purpose)); err != nil {
		return fmt.Errorf("invalidate active otp: %w", err)
	}
	return nil
}

// applySendLimits atomically applies the per-(email, purpose) send policy
// (onboarding AC9, otp-mechanism AC4) for one
// send attempt and, when the send is allowed, records it. It must run inside
// the transaction that also writes the OTP row, so a refused attempt writes
// nothing and an allowed one cannot "consume a slot without sending a code".
//
// The row is upserted then locked FOR UPDATE, which serialises two concurrent
// sends for the same email — without it both could read SendCount=2 and both
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
func applySendLimits(ctx context.Context, tx *sql.Tx, email string, purpose domain.OTPPurpose, limits domain.SendLimits, now time.Time) (*domain.SendState, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO otp_send_limits (email, purpose) VALUES ($1, $2)
		ON CONFLICT (email, purpose) DO NOTHING`, email, string(purpose)); err != nil {
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
		WHERE email = $1 AND purpose = $2
		FOR UPDATE`, email, string(purpose)).Scan(&sendCount, &lastSentAt, &lockedUntil); err != nil {
		return nil, fmt.Errorf("lock otp send limit: %w", err)
	}

	// (0) first send of a cycle is free.
	if !lastSentAt.Valid || !now.Before(lastSentAt.Time.Add(limits.LockDuration)) {
		if err := updateSendLimit(ctx, tx, email, purpose, 0, now, nil); err != nil {
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
		if err := updateSendLimit(ctx, tx, email, purpose, sendCount, lastSentAt.Time, &lockUntil); err != nil {
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
	if err := updateSendLimit(ctx, tx, email, purpose, sendCount, now, lockUntil); err != nil {
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
func updateSendLimit(ctx context.Context, tx *sql.Tx, email string, purpose domain.OTPPurpose, sendCount int, lastSentAt time.Time, lockedUntil *time.Time) error {
	const query = `
		UPDATE otp_send_limits
		SET send_count = $3, last_sent_at = $4, locked_until = $5, updated_at = now()
		WHERE email = $1 AND purpose = $2
	`
	if _, err := tx.ExecContext(ctx, query, email, string(purpose), sendCount, lastSentAt, lockedUntil); err != nil {
		return fmt.Errorf("update otp send limit: %w", err)
	}
	return nil
}

// insertOTP writes a fresh otp_codes row. The row carries no send counter — the
// send rate limit lives in otp_send_limits — so there is nothing to pass back.
func insertOTP(ctx context.Context, tx *sql.Tx, email string, purpose domain.OTPPurpose, otpHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO otp_codes (email, purpose, otp_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.ExecContext(ctx, query, email, string(purpose), otpHash, expiresAt); err != nil {
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
	if err := row.Scan(&otp.ID, &otp.Email, &purpose, &otp.OTPHash, &otp.Attempts, &lockedUntil, &otp.ExpiresAt, &usedAt, &otp.CreatedAt); err != nil {
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

// isForeignKeyViolation is isUniqueViolation's sibling for 23503, matched by
// constraint name for the same reason: it tells a vanished temple apart from any
// other foreign-key failure.
func isForeignKeyViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23503" && pqErr.Constraint == constraint
}

// TempleExists reports whether temples has a row with this id (AC8).
func (r *registrationRepository) TempleExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM temples WHERE id = $1)`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check temple exists: %w", err)
	}
	return exists, nil
}
