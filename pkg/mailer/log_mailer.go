package mailer

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
)

// LogMailer is a ⚠️ DEV-ONLY Mailer that logs instead of sending real
// email. Wire in a real provider (SendGrid/SES/GCP equivalent) before
// production — see package doc.
type LogMailer struct {
	logger logging.Logger
}

// NewLogMailer returns a Mailer that logs the email it would have sent.
func NewLogMailer(logger logging.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

// SendPasswordResetEmail logs the reset email instead of sending it.
func (m *LogMailer) SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error {
	m.logger.WithContext(ctx).WithFields(logging.Fields{
		"to_email":    toEmail,
		"reset_token": resetToken,
	}).Info("DEV MODE: password reset email not actually sent, logging instead")
	return nil
}
