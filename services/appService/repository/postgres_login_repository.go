package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type loginRepository struct {
	db database.Database
}

// NewLoginRepository returns the Postgres-backed domain.LoginRepository for
// US-002. Every SQL statement and transaction boundary of login, refresh and
// logout lives here.
func NewLoginRepository(db database.Database) domain.LoginRepository {
	return &loginRepository{db: db}
}

// userColumns is the explicit column list every users read in this file uses,
// so a column added to the table later cannot silently change what is scanned.
const userColumns = `id, email, email_verified_at, phone_number, password_hash, dharma_name, ordination_temple_id, created_at, updated_at`

// FindUserByPhone is flow 1 Step 3. No row is ErrUserNotFound — including a
// number that exists only in pending_registration, which lives in another table
// and therefore simply is not found here (AC2).
func (r *loginRepository) FindUserByPhone(ctx context.Context, phoneNumber string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE phone_number = $1`
	return scanUser(r.db.QueryRow(ctx, query, phoneNumber))
}

// FindUserByID rebuilds the account behind a session for the refresh flow.
func (r *loginRepository) FindUserByID(ctx context.Context, userID int64) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return scanUser(r.db.QueryRow(ctx, query, userID))
}

func scanUser(row *sql.Row) (*domain.User, error) {
	var (
		u                  domain.User
		emailVerifiedAt    sql.NullTime
		dharmaName         sql.NullString
		ordinationTempleID sql.NullString
	)
	if err := row.Scan(&u.ID, &u.Email, &emailVerifiedAt, &u.PhoneNumber, &u.PasswordHash, &dharmaName, &ordinationTempleID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	u.DharmaName = nullStringToPtr(dharmaName)
	u.OrdinationTempleID = nullStringToPtr(ordinationTempleID)
	return &u, nil
}

// FindLoginAttempt is flow 1 Step 2: a plain read, no row is (nil, nil).
func (r *loginRepository) FindLoginAttempt(ctx context.Context, phoneNumber string) (*domain.LoginAttempt, error) {
	const query = `
		SELECT phone_number, failed_count, locked_until, last_failed_at
		FROM login_attempts
		WHERE phone_number = $1
	`
	var (
		a           domain.LoginAttempt
		lockedUntil sql.NullTime
	)
	if err := r.db.QueryRow(ctx, query, phoneNumber).Scan(&a.PhoneNumber, &a.FailedCount, &lockedUntil, &a.LastFailedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find login attempt: %w", err)
	}
	if lockedUntil.Valid {
		a.LockedUntil = &lockedUntil.Time
	}
	return &a, nil
}

// RecordLoginFailure is flow 1 Step 5: one atomic UPSERT, so concurrent wrong
// guesses on the same number serialize on the row lock instead of all reading
// the same starting count (AC7).
//
// A row is stale — count restarts at 1 — when its lock has already expired or
// its previous failure is older than the reset window. The "new count" used to
// decide locking is written out in both SET expressions because a SET clause
// cannot see the value another column of the same statement is being given.
func (r *loginRepository) RecordLoginFailure(ctx context.Context, phoneNumber string, maxAttempts int, lockDuration, resetWindow time.Duration) (*domain.LoginAttempt, error) {
	const query = `
		INSERT INTO login_attempts (phone_number, failed_count, locked_until, last_failed_at, updated_at)
		VALUES (
			$1, 1,
			CASE WHEN 1 >= $2 THEN now() + make_interval(secs => $3) ELSE NULL END,
			now(), now()
		)
		ON CONFLICT (phone_number) DO UPDATE SET
			failed_count = CASE
				WHEN (login_attempts.locked_until IS NOT NULL AND login_attempts.locked_until <= now())
				  OR login_attempts.last_failed_at < now() - make_interval(secs => $4)
				THEN 1
				ELSE login_attempts.failed_count + 1
			END,
			locked_until = CASE
				WHEN (CASE
					WHEN (login_attempts.locked_until IS NOT NULL AND login_attempts.locked_until <= now())
					  OR login_attempts.last_failed_at < now() - make_interval(secs => $4)
					THEN 1
					ELSE login_attempts.failed_count + 1
				END) >= $2
				THEN now() + make_interval(secs => $3)
				ELSE NULL
			END,
			last_failed_at = now(),
			updated_at = now()
		RETURNING failed_count, locked_until, last_failed_at
	`
	a := domain.LoginAttempt{PhoneNumber: phoneNumber}
	var lockedUntil sql.NullTime
	if err := r.db.QueryRow(ctx, query, phoneNumber, maxAttempts, lockDuration.Seconds(), resetWindow.Seconds()).
		Scan(&a.FailedCount, &lockedUntil, &a.LastFailedAt); err != nil {
		return nil, fmt.Errorf("record login failure: %w", err)
	}
	if lockedUntil.Valid {
		a.LockedUntil = &lockedUntil.Time
	}
	return &a, nil
}

// CreateLoginSession is flow 1 Step 6 (a)–(f), one transaction.
func (r *loginRepository) CreateLoginSession(ctx context.Context, user *domain.User, device domain.DeviceInfo, sessionTTL time.Duration, issueTokens domain.TokenIssuer) (*domain.LoginResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// (b) The upsert takes the row lock on user_devices, which is what
	// serializes two concurrent logins from one device: the second waits here
	// until the first commits, then revokes the session the first just made.
	const upsertDevice = `
		INSERT INTO user_devices
			(user_id, device_id, platform, device_model, os_version, app_version, device_name, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, device_id) DO UPDATE SET
			platform = EXCLUDED.platform,
			device_model = EXCLUDED.device_model,
			os_version = EXCLUDED.os_version,
			app_version = EXCLUDED.app_version,
			device_name = EXCLUDED.device_name,
			ip_address = EXCLUDED.ip_address,
			user_agent = EXCLUDED.user_agent,
			last_seen_at = now()
		RETURNING id
	`
	var deviceRowID int64
	if err := tx.QueryRowContext(ctx, upsertDevice,
		user.ID,
		deviceFingerprint(device),
		device.Platform,
		device.DeviceModel,
		nullIfEmpty(device.OSVersion),
		nullIfEmpty(device.AppVersion),
		nullIfEmpty(device.DeviceName),
		nullIfEmpty(device.IPAddress),
		nullIfEmpty(device.UserAgent),
	).Scan(&deviceRowID); err != nil {
		return nil, fmt.Errorf("upsert device: %w", err)
	}

	// (c) Only this device's session: other devices and US-001's device-less
	// sessions are untouched (AC3/AC4).
	if _, err := tx.ExecContext(ctx, revokeByDeviceSQL, deviceRowID, domain.RevokeReasonSupersededByLogin); err != nil {
		return nil, fmt.Errorf("revoke superseded session: %w", err)
	}

	// (d) The ID exists before minting because the access token carries it as
	// claim "sid".
	sessionID := uuid.NewString()
	accessToken, refreshToken, err := issueTokens(strconv.FormatInt(user.ID, 10), sessionID)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	// (e)
	const insertSession = `
		INSERT INTO user_sessions (user_id, device_id, session_id, refresh_token_hash, token_version, expires_at)
		VALUES ($1, $2, $3, $4, 1, $5)
	`
	if _, err := tx.ExecContext(ctx, insertSession,
		user.ID,
		deviceRowID,
		sessionID,
		domain.HashRefreshToken(refreshToken),
		time.Now().Add(sessionTTL),
	); err != nil {
		return nil, fmt.Errorf("create login session: insert session: %w", err)
	}

	// (f) In the same transaction: a login whose transaction rolls back must not
	// have wiped the brute-force counter.
	if _, err := tx.ExecContext(ctx, `DELETE FROM login_attempts WHERE phone_number = $1`, user.PhoneNumber); err != nil {
		return nil, fmt.Errorf("reset login attempts: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create login session: %w", err)
	}

	return &domain.LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

const revokeByDeviceSQL = `
	UPDATE user_sessions
	SET is_active = false, is_revoked = true, revoked_at = now(), revoked_reason = $2, updated_at = now()
	WHERE device_id = $1 AND is_revoked = false
`

// FindSessionByRefreshHash is flow 2 Step 2. A row that matches but is revoked,
// inactive or expired is ErrRefreshTokenInvalid — the same as no row — so the
// caller only ever branches on the reuse case.
func (r *loginRepository) FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (*domain.Session, bool, error) {
	const query = `
		SELECT id, session_id, user_id, device_id, token_version, expires_at, is_revoked, is_active,
		       (refresh_token_hash <> $1) AS matched_previous
		FROM user_sessions
		WHERE refresh_token_hash = $1 OR previous_refresh_token_hash = $1
		LIMIT 1
	`
	var (
		s               domain.Session
		deviceID        sql.NullInt64
		isActive        bool
		matchedPrevious bool
	)
	if err := r.db.QueryRow(ctx, query, refreshTokenHash).Scan(
		&s.ID, &s.SessionID, &s.UserID, &deviceID, &s.TokenVersion, &s.ExpiresAt, &s.IsRevoked, &isActive, &matchedPrevious,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, domain.ErrRefreshTokenInvalid
		}
		return nil, false, fmt.Errorf("find session by refresh hash: %w", err)
	}
	if deviceID.Valid {
		s.DeviceID = &deviceID.Int64
	}

	if s.IsRevoked || !isActive || !s.ExpiresAt.After(time.Now()) {
		return nil, false, domain.ErrRefreshTokenInvalid
	}
	return &s, matchedPrevious, nil
}

// RotateSession is flow 2 Step 3, one transaction. FOR UPDATE makes two
// concurrent refreshes with the same token serialize: the loser re-reads a hash
// that no longer matches and gets ErrRefreshTokenInvalid.
func (r *loginRepository) RotateSession(ctx context.Context, sessionRowID int64, oldHash string, sessionTTL time.Duration, issueTokens domain.TokenIssuer) (*domain.RefreshResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const lockSession = `
		SELECT user_id, session_id, refresh_token_hash, is_revoked, is_active, expires_at
		FROM user_sessions
		WHERE id = $1
		FOR UPDATE
	`
	var (
		userID      int64
		sessionID   string
		currentHash string
		isRevoked   bool
		isActive    bool
		expiresAt   time.Time
	)
	if err := tx.QueryRowContext(ctx, lockSession, sessionRowID).Scan(&userID, &sessionID, &currentHash, &isRevoked, &isActive, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("lock session: %w", err)
	}
	if currentHash != oldHash || isRevoked || !isActive || !expiresAt.After(time.Now()) {
		return nil, domain.ErrRefreshTokenInvalid
	}

	accessToken, refreshToken, err := issueTokens(strconv.FormatInt(userID, 10), sessionID)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	const rotate = `
		UPDATE user_sessions SET
			previous_refresh_token_hash = refresh_token_hash,
			refresh_token_hash = $2,
			token_version = token_version + 1,
			last_activity_at = now(),
			updated_at = now(),
			expires_at = $3
		WHERE id = $1
	`
	if _, err := tx.ExecContext(ctx, rotate, sessionRowID, domain.HashRefreshToken(refreshToken), time.Now().Add(sessionTTL)); err != nil {
		return nil, fmt.Errorf("rotate session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit rotate session: %w", err)
	}

	return &domain.RefreshResult{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// RevokeSession revokes one session of one user. Both IDs are in the WHERE so a
// caller cannot end someone else's session by guessing a session_id; zero rows
// changed is success (idempotent — AC13/AC14).
func (r *loginRepository) RevokeSession(ctx context.Context, userID int64, sessionID, reason string) error {
	const query = `
		UPDATE user_sessions
		SET is_active = false, is_revoked = true, revoked_at = now(), revoked_reason = $3, updated_at = now()
		WHERE user_id = $1 AND session_id = $2 AND is_revoked = false
	`
	if _, err := r.db.Exec(ctx, query, userID, sessionID, reason); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// deviceFingerprint is user_devices.device_id: SHA-256 of
// platform|os_version|device_model|user_agent, computed here and never taken
// from the client. Two devices sharing all four are indistinguishable (a known
// gap recorded in design.md).
func deviceFingerprint(d domain.DeviceInfo) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{d.Platform, d.OSVersion, d.DeviceModel, d.UserAgent}, "|")))
	return hex.EncodeToString(sum[:])
}

// nullIfEmpty stores an absent optional field as NULL rather than "".
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
