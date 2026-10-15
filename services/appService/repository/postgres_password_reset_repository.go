package repository

import (
	"context"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type passwordResetRepository struct {
	db database.Database
}

// NewPasswordResetRepository returns the Postgres-backed
// domain.PasswordResetRepository (login US-003).
func NewPasswordResetRepository(db database.Database) domain.PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

// FindVerifiedUserByEmail is Step P-3 / C-3. users.email is stored lower-case
// and the caller passes the normalised form.
func (r *passwordResetRepository) FindVerifiedUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1 AND email_verified_at IS NOT NULL`
	return scanUser(r.db.QueryRow(ctx, query, email))
}

// ResetPassword is Step C-4: password, code, sessions and sign-in lock change
// together or not at all.
func (r *passwordResetRepository) ResetPassword(ctx context.Context, userID int64, newPasswordHash string, otpID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// The code goes first: its used_at IS NULL guard is what serialises two
	// concurrent confirms with the same code — the loser updates nothing and
	// the password stays whatever the winner set.
	res, err := tx.ExecContext(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1 AND used_at IS NULL`, otpID)
	if err != nil {
		return fmt.Errorf("mark otp used: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("mark otp used: %w", err)
	} else if n == 0 {
		return domain.ErrOTPExpired
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE user_sessions
		SET is_active = false, is_revoked = true, revoked_at = now(), revoked_reason = $2, updated_at = now()
		WHERE user_id = $1 AND is_revoked = false
	`, userID, domain.RevokeReasonPasswordReset); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM login_attempts WHERE email = (SELECT email FROM users WHERE id = $1)`, userID); err != nil {
		return fmt.Errorf("clear login attempts: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reset password: %w", err)
	}
	return nil
}
