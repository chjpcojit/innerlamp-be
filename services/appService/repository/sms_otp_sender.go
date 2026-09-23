package repository

import (
	"context"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/sms"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// smsOTPSender implements domain.OTPSender on top of pkg/sms.Sender.
//
// pkg/sms is a generic "send a message to a number" package with no
// knowledge of OTPs, purposes, or this service's domain types — that is
// what keeps it reusable outside US-001. Turning an OTP code and purpose
// into message text is a service-level concern, so it lives here, next to
// every other domain-port implementation this package holds.
type smsOTPSender struct {
	sender sms.Sender
}

// NewSMSOTPSender returns the domain.OTPSender backed by sender.
func NewSMSOTPSender(sender sms.Sender) domain.OTPSender {
	return &smsOTPSender{sender: sender}
}

// SendOTP formats otpCode into a message and hands it to the underlying
// sms.Sender. purpose is not yet reflected in the message text — every
// purpose but OTPPurposeRegister is unimplemented above this layer (see
// domain.RegistrationRepository's doc), so there is nothing purpose-specific
// to say yet; US-002/US-003 can vary the copy here without touching the
// port or pkg/sms.
func (s *smsOTPSender) SendOTP(ctx context.Context, phoneNumber, otpCode string, purpose domain.OTPPurpose) error {
	message := fmt.Sprintf("InnerLamp verification code: %s. It expires shortly — do not share it with anyone.", otpCode)
	return s.sender.Send(ctx, phoneNumber, message)
}
