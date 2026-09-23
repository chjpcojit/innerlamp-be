// Package logsms is a ⚠️ DEV-ONLY sms.Sender that logs the message instead
// of delivering it. It is the only implementation today — wire in a real
// provider (Twilio, AWS SNS, a GCP equivalent) before production by adding a
// sibling subpackage that registers its own sms.ProviderType; pkg/sms.Sender
// does not change to support it.
package logsms

import (
	"context"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/sms"
)

func init() {
	sms.Register(sms.ProviderLog, New)
}

// logSender logs every message it is asked to send and never fails —
// appropriate for local/dev, wrong for anywhere a caller needs to know
// whether an SMS actually reached a phone.
type logSender struct {
	logger logging.Logger
}

// New builds the log-only Sender. cfg.Logger is required: this provider's
// entire behavior is logging, so a nil logger here is a caller error, not
// something to paper over with a silent stdlib fallback.
func New(cfg *sms.Config) (sms.Sender, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("%w: logger is required for the log sms provider", sms.ErrInvalidConfig)
	}
	return &logSender{logger: cfg.Logger}, nil
}

// Send logs phoneNumber and message and returns nil.
func (s *logSender) Send(ctx context.Context, phoneNumber, message string) error {
	s.logger.WithContext(ctx).WithFields(logging.Fields{
		"phone_number": phoneNumber,
		"message":      message,
	}).Info("DEV MODE: sms not actually sent, logging instead")
	return nil
}
