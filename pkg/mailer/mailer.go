// Package mailer sends transactional emails. It is a new external
// dependency introduced for US-003 (quên mật khẩu) — the onboarding spec
// calls out that no such dependency exists in the architecture yet.
//
// Only one email currently needs sending, so this package is deliberately
// just an interface plus a single dev-only implementation (LogMailer) —
// no provider registry like pkg/database/pkg/storage. Add one (Register/
// New/FactoryFunc, following those packages' pattern) if/when a second
// real provider (SendGrid, SES, ...) is wired in; it would be premature
// abstraction before that.
package mailer

import "context"

// Mailer sends transactional emails related to authentication.
type Mailer interface {
	// SendPasswordResetEmail sends the reset link/code for a forgot-password
	// request — US-003 AC1. resetToken is the raw (unhashed) token; the
	// caller is responsible for only ever persisting its hash.
	SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error
}
