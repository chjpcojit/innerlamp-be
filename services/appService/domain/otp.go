package domain

import "time"

// OTPPurpose discriminates what an otp_codes row is for. It maps the purpose
// column, whose CHECK constraint accepts exactly these four values.
//
// All four are declared here even though US-001 implements only register:
// login (US-002) and reset_password (US-003) will issue OTPs through the same
// table, and declaring them now means chk_otp_codes_purpose never needs an
// ALTER later. verify_email is unused by any current story — it is kept so the
// set is complete and the constraint reads as the closed list it is.
type OTPPurpose string

const (
	OTPPurposeRegister      OTPPurpose = "register"
	OTPPurposeLogin         OTPPurpose = "login"
	OTPPurposeResetPassword OTPPurpose = "reset_password"
	OTPPurposeVerifyEmail   OTPPurpose = "verify_email"
)

// OTPCode is one issued one-time code. It maps the otp_codes table, shared by
// every OTP flow, scoped by the pair (Email, Purpose) and discriminated by
// Purpose (otp-mechanism AC1).
//
// OTPHash is a bcrypt hash, never the six-digit code itself. A 10^6 keyspace
// falls to a precomputed table in seconds under a fast hash like SHA-256, so
// the code is hashed exactly the way a password is.
//
// The row carries its own wrong-guess limit — Attempts and LockedUntil. Each
// verify reserves one attempt atomically before comparing (OTPRepository.
// ConsumeOTPAttempt), and the reservation that reaches the cap sets
// LockedUntil in the same statement (otp-mechanism AC7).
//
// The *send* limit is deliberately NOT here: it lives in OTPSendLimit, a
// per-(email, purpose) row that outlives this one and whose lock must not
// block verification (onboarding AC13).
type OTPCode struct {
	ID          int64
	Email       string
	Purpose     OTPPurpose
	OTPHash     string
	Attempts    int
	LockedUntil *time.Time
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
}
