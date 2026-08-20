package domain

import "time"

// PasswordResetToken maps to the password_reset_tokens table (US-003).
// Only the token's SHA-256 hash is ever persisted, never the raw value
// that goes out in the email.
type PasswordResetToken struct {
	ID        string
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// ForgotPasswordInput is the use-case input for US-003 AC1/AC2.
type ForgotPasswordInput struct {
	Email string
}

// ResetPasswordInput is the use-case input for US-003 AC3/AC4.
type ResetPasswordInput struct {
	Token       string
	NewPassword string
}
