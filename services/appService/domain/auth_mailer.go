package domain

import (
	"context"
	"time"
)

// AuthMailer is the only email port the auth use cases see (onboarding
// design.md "Mailer"). The implementation (repository.EmailAuthMailer) owns
// the wording; callers pass only the recipient, the clear code and how long
// it stays valid. No method takes a password: no auth email ever carries one
// (AC19, NFR-ONB-03).
type AuthMailer interface {
	// SendRegisterCode emails the registration code. ttl is the code's
	// validity, shown to the user in minutes.
	SendRegisterCode(ctx context.Context, to, code string, ttl time.Duration) error

	// SendAccountExistsNotice tells the owner of an already-registered email
	// that someone tried to register with it, and points them to login or
	// forgot-password. It carries no code.
	SendAccountExistsNotice(ctx context.Context, to string) error

	// SendPasswordResetCode emails the password-reset code. ttl is the code's
	// validity, shown to the user in minutes.
	SendPasswordResetCode(ctx context.Context, to, code string, ttl time.Duration) error
}
