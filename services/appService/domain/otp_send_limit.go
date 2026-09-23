package domain

import "time"

// OTPSendLimit is the per-(phone_number, purpose) send counter behind AC5.
//
// It is deliberately separate from OTPCode: the row lives across many OTP rows
// (a code expires after TTL, the counter must outlive it), and it is the store
// for the *send* lock only — never for the wrong-guess lock, which stays on
// OTPCode so that VerifyOTP already honours it. Keeping the two apart is what
// lets the send lock block request-otp/resend-otp without blocking a user from
// verifying the last code they were sent.
//
// SendCount counts only the *counted* sends of the current cycle: the first
// send of a cycle is free and leaves it at 0. LastSentAt anchors both the send
// cooldown and the cycle reset: once it is older than SendLimits.LockDuration,
// the next send starts a fresh cycle.
type OTPSendLimit struct {
	PhoneNumber string
	Purpose     OTPPurpose
	SendCount   int
	LastSentAt  *time.Time
	LockedUntil *time.Time
}

// SendLimits is the tunable policy the use case passes down to the repository
// for one send attempt. It mirrors the relevant fields of config.OTPConfig but
// is declared here so the repository can apply the policy without importing
// config — the same arrangement as sessionTTL being a plain time.Duration.
type SendLimits struct {
	// Cooldown is the floor between two sends.
	Cooldown time.Duration

	// LockDuration is how long sending is blocked once MaxAttempts is reached,
	// and equally how long a cycle may sit idle before its counter resets.
	LockDuration time.Duration

	// MaxAttempts is the cap on counted sends in one cycle.
	MaxAttempts int
}

// SendState reports the outcome of an allowed send so the use case can tell the
// client how many sends remain before the cap. Remaining is never negative.
type SendState struct {
	SendCount int
	Remaining int
}
