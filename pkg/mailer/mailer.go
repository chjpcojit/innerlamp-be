// Package mailer sends transactional emails.
//
// Send is the generic entry point: callers build the subject and body
// themselves (appService's repository.EmailAuthMailer does this for the
// register / password-reset codes). The two older purpose-specific methods
// stay for backofficeService.
//
// The package is deliberately just an interface plus a single dev-only
// implementation (LogMailer) —
// no provider registry like pkg/database/pkg/storage. Add one (Register/
// New/FactoryFunc, following those packages' pattern) if/when a second
// real provider (SendGrid, SES, ...) is wired in; it would be premature
// abstraction before that.
package mailer

import "context"

// Message is one email. From is the sender address (the caller's configured
// sender, e.g. appService mailer.from). To is a single recipient address.
// Text is the plain-text body and is required; HTML is optional (empty =
// text only).
type Message struct {
	From    string
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer sends transactional emails related to authentication.
type Mailer interface {
	// Send delivers msg.
	Send(ctx context.Context, msg Message) error

	// SendPasswordResetEmail sends the reset link/code for a forgot-password
	// request — US-003 AC1. resetToken is the raw (unhashed) token; the
	// caller is responsible for only ever persisting its hash.
	SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error

	// SendAccountInviteEmail tells a newly created backoffice account how to
	// sign in: its login email and one-time temporary password, which the
	// account must change on first login. tempPassword is the clear value.
	SendAccountInviteEmail(ctx context.Context, toEmail, name, tempPassword string) error
}
