package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sessionTTL matches pkg/session.DefaultConfig().RefreshTokenTTL (see
// docs/db/register-erd.md field notes on expires_at).
const sessionTTL = 7 * 24 * time.Hour

type userRepository struct {
	db database.Database
}

// NewUserRepository returns a PostgreSQL-backed domain.UserRepository.
func NewUserRepository(db database.Database) domain.UserRepository {
	return &userRepository{db: db}
}

// Register inserts the user, device, and session rows in a single
// transaction, generating the JWT pair via issueTokens once the user's ID
// is known.
func (r *userRepository) Register(ctx context.Context, user *domain.User, device *domain.Device, issueTokens domain.TokenIssuer) (*domain.Session, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, phone_number, dharma_name, ordination_place, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, now(), now())
		 RETURNING id, created_at, updated_at`,
		user.Email, user.PasswordHash, user.PhoneNumber, nullableString(user.DharmaName), nullableString(user.OrdinationPlace),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return nil, domain.ErrEmailTaken
		}
		if isUniqueViolation(err, "users_phone_number_key") {
			return nil, domain.ErrPhoneTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	device.UserID = user.ID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO user_devices (user_id, device_id, platform, os_version, app_version, device_model, device_name, ip_address, user_agent, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		 RETURNING id, created_at`,
		device.UserID, device.DeviceID, device.Platform, device.OSVersion, device.AppVersion, device.DeviceModel, device.DeviceName, device.IPAddress, device.UserAgent,
	).Scan(&device.ID, &device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	session, err := insertSession(ctx, tx, user.ID, device.ID, issueTokens)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return session, nil
}

// FindByEmail looks up a user by email for login — US-002 AC1/AC2/AC3.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{}
	var dharmaName, ordinationPlace sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, phone_number, dharma_name, ordination_place, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.PhoneNumber, &dharmaName, &ordinationPlace, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	user.DharmaName = dharmaName.String
	user.OrdinationPlace = ordinationPlace.String
	return user, nil
}

// CreateSession upserts the device row for userID and issues a new session
// for it — US-002 AC4: logging in from a new device never disturbs another
// device's session row, since each device row (and its sessions) is keyed
// independently by (user_id, device_id).
func (r *userRepository) CreateSession(ctx context.Context, userID int64, device *domain.Device, issueTokens domain.TokenIssuer) (*domain.Session, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	device.UserID = userID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO user_devices (user_id, device_id, platform, os_version, app_version, device_model, device_name, ip_address, user_agent, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		 ON CONFLICT (user_id, device_id) DO UPDATE SET
		   platform = EXCLUDED.platform,
		   os_version = EXCLUDED.os_version,
		   app_version = EXCLUDED.app_version,
		   device_model = EXCLUDED.device_model,
		   device_name = EXCLUDED.device_name,
		   ip_address = EXCLUDED.ip_address,
		   user_agent = EXCLUDED.user_agent
		 RETURNING id, created_at`,
		device.UserID, device.DeviceID, device.Platform, device.OSVersion, device.AppVersion, device.DeviceModel, device.DeviceName, device.IPAddress, device.UserAgent,
	).Scan(&device.ID, &device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert device: %w", err)
	}

	session, err := insertSession(ctx, tx, userID, device.ID, issueTokens)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return session, nil
}

// CreatePasswordResetToken persists a new reset token — US-003 AC1.
func (r *userRepository) CreatePasswordResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*domain.PasswordResetToken, error) {
	t := &domain.PasswordResetToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	err := r.db.QueryRow(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, now())
		 RETURNING id, created_at`,
		userID, tokenHash, expiresAt,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert password reset token: %w", err)
	}

	return t, nil
}

// ResetPassword atomically validates tokenHash, rotates the account's
// password, marks the token used, and revokes every active session for the
// account — US-003 AC3/AC4.
func (r *userRepository) ResetPassword(ctx context.Context, tokenHash, newPasswordHash string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		tokenID   string
		userID    int64
		expiresAt time.Time
		usedAt    sql.NullTime
	)
	err = tx.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1 FOR UPDATE`,
		tokenHash,
	).Scan(&tokenID, &userID, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrResetTokenInvalid
		}
		return fmt.Errorf("find reset token: %w", err)
	}

	if usedAt.Valid || time.Now().After(expiresAt) {
		return domain.ErrResetTokenInvalid
	}

	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`, newPasswordHash, userID); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`, tokenID); err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE user_sessions SET is_revoked = true, is_active = false, revoked_at = now(), revoked_reason = 'password_reset' WHERE user_id = $1 AND is_revoked = false`,
		userID,
	); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// FindByID looks up a user by primary key — US-004 AC1 (profile view).
func (r *userRepository) FindByID(ctx context.Context, userID int64) (*domain.User, error) {
	user := &domain.User{}
	var dharmaName, ordinationPlace, avatarURL sql.NullString

	row := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, phone_number, dharma_name, ordination_place, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID,
	)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.PhoneNumber, &dharmaName, &ordinationPlace, &avatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.DharmaName = dharmaName.String
	user.OrdinationPlace = ordinationPlace.String
	user.AvatarURL = avatarURL.String
	return user, nil
}

// UpdateProfile overwrites dharma_name/ordination_place for userID —
// US-004 AC2/AC5: an empty field clears the column (nullableString maps ""
// to SQL NULL, same convention Register uses for these two columns).
func (r *userRepository) UpdateProfile(ctx context.Context, userID int64, input domain.ProfileUpdateInput) (*domain.User, error) {
	user := &domain.User{ID: userID}
	var dharmaName, ordinationPlace, avatarURL sql.NullString

	err := r.db.QueryRow(ctx,
		`UPDATE users SET dharma_name = $1, ordination_place = $2, updated_at = now()
		 WHERE id = $3
		 RETURNING id, email, password_hash, phone_number, dharma_name, ordination_place, avatar_url, created_at, updated_at`,
		nullableString(input.DharmaName), nullableString(input.OrdinationPlace), userID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.PhoneNumber, &dharmaName, &ordinationPlace, &avatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}

	user.DharmaName = dharmaName.String
	user.OrdinationPlace = ordinationPlace.String
	user.AvatarURL = avatarURL.String
	return user, nil
}

// UpdateAvatarURL persists a new avatar_url for userID, replacing any
// previous value — US-004 AC3.
func (r *userRepository) UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) (*domain.User, error) {
	user := &domain.User{ID: userID}
	var dharmaName, ordinationPlace, storedAvatarURL sql.NullString

	err := r.db.QueryRow(ctx,
		`UPDATE users SET avatar_url = $1, updated_at = now()
		 WHERE id = $2
		 RETURNING id, email, password_hash, phone_number, dharma_name, ordination_place, avatar_url, created_at, updated_at`,
		avatarURL, userID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.PhoneNumber, &dharmaName, &ordinationPlace, &storedAvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("update avatar url: %w", err)
	}

	user.DharmaName = dharmaName.String
	user.OrdinationPlace = ordinationPlace.String
	user.AvatarURL = storedAvatarURL.String
	return user, nil
}

// insertSession issues a fresh JWT pair via issueTokens and inserts the
// resulting session row, shared by Register and CreateSession.
func insertSession(ctx context.Context, tx *sql.Tx, userID, deviceID int64, issueTokens domain.TokenIssuer) (*domain.Session, error) {
	accessToken, refreshToken, err := issueTokens(strconv.FormatInt(userID, 10))
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	now := time.Now()
	session := &domain.Session{
		UserID:         userID,
		DeviceID:       deviceID,
		SessionID:      uuid.New().String(),
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		TokenVersion:   1,
		IsActive:       true,
		LastActivityAt: now,
		ExpiresAt:      now.Add(sessionTTL),
	}

	err = tx.QueryRowContext(ctx,
		`INSERT INTO user_sessions (user_id, device_id, session_id, access_token, refresh_token, token_version, is_active, is_revoked, last_activity_at, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8, $9, now(), now())
		 RETURNING id, created_at, updated_at`,
		session.UserID, session.DeviceID, session.SessionID, session.AccessToken, session.RefreshToken, session.TokenVersion, session.IsActive, session.LastActivityAt, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return session, nil
}

// isUniqueViolation reports whether err is a Postgres unique_violation
// (23505) on the given constraint name.
func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505" && pqErr.Constraint == constraint
	}
	return false
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
