package domain

import "context"

// RequestPasswordResetInput is the use-case input for
// POST /v1/auth/password-reset/request-otp.
type RequestPasswordResetInput struct {
	Email string
}

// RequestPasswordResetResult is identical whether or not the email has an
// account (login AC23/AC24): it carries only the window numbers.
type RequestPasswordResetResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
}

// ConfirmPasswordResetInput is the use-case input for
// POST /v1/auth/password-reset/confirm.
type ConfirmPasswordResetInput struct {
	Email       string
	OTPCode     string
	NewPassword string
}

// PasswordResetRepository is the persistence port of US-003. The OTP state
// itself (codes, attempts, send limits) goes through OTPRepository via the
// shared OTPService; this port holds only what reset adds.
type PasswordResetRepository interface {
	// FindVerifiedUserByEmail returns the account for an already-normalised
	// email only when its email_verified_at is set. ErrUserNotFound otherwise,
	// including an email that exists only in pending_registration.
	FindVerifiedUserByEmail(ctx context.Context, email string) (*User, error)

	// ResetPassword is Step C-4, one transaction: set users.password_hash,
	// mark the code otpID used, revoke every active session of userID with
	// RevokeReasonPasswordReset, and delete the account's login_attempts row
	// so a sign-in lock is lifted (AC28).
	//
	// A code that was consumed in the meantime (used_at already set) returns
	// ErrOTPExpired and changes nothing, so one code cannot reset twice.
	ResetPassword(ctx context.Context, userID int64, newPasswordHash string, otpID int64) error
}
