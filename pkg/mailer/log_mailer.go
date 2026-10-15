package mailer

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
)

var _ Mailer = (*LogMailer)(nil)

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

// Send logs the whole message — including any code in its body — instead of
// sending it. That is exactly why this mailer is dev-only.
func (m *LogMailer) Send(ctx context.Context, msg Message) error {
	m.logger.WithContext(ctx).WithFields(logging.Fields{
		"from":     msg.From,
		"to_email": msg.To,
		"subject":  msg.Subject,
		"text":     msg.Text,
		"html":     msg.HTML,
	}).Info("DEV MODE: email not actually sent, logging instead")
	return nil
}

// SendPasswordResetEmail logs the reset email instead of sending it.
func (m *LogMailer) SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error {
	m.logger.WithContext(ctx).WithFields(logging.Fields{
		"to_email":    toEmail,
		"reset_token": resetToken,
	}).Info("DEV MODE: password reset email not actually sent, logging instead")
	return nil
}

// SendAccountInviteEmail logs the invite instead of sending it. It logs the
// temporary password, which is why this mailer is dev-only.
func (m *LogMailer) SendAccountInviteEmail(ctx context.Context, toEmail, name, tempPassword string) error {
	m.logger.WithContext(ctx).WithFields(logging.Fields{
		"to_email":      toEmail,
		"name":          name,
		"temp_password": tempPassword,
	}).Info("DEV MODE: account invite email not actually sent, logging instead")
	return nil
}
