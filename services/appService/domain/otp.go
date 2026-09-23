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

// OTPCode is one issued one-time password. It maps the otp_codes table, shared
// by every OTP flow and discriminated by Purpose.
//
// OTPHash is a bcrypt hash, never the six-digit code itself. A 10^6 keyspace
// falls to a precomputed table in seconds under a fast hash like SHA-256, so
// the OTP is hashed exactly the way a password is — the fact that the value is
// short-lived does not make a fast hash safe.
//
// The row carries its own wrong-guess rate limiting — Attempts and LockedUntil —
// so AC12 (lockout after repeated wrong codes) needs no second store and
// survives a process restart. UsedAt is set both on success and on lock, so a
// locked row can never be replayed; that is why RegistrationRepository.
// FindActiveOTP matches on UsedAt OR LockedUntil rather than on UsedAt alone.
//
// The *send* limit (AC5) is deliberately NOT here: it lives in OTPSendLimit,
// a per-phone row that outlives this one and whose lock must not block
// VerifyOTP. See OTPSendLimit's doc comment.
type OTPCode struct {
	ID          int64
	PhoneNumber string
	Purpose     OTPPurpose
	OTPHash     string
	Attempts    int
	LockedUntil *time.Time
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
}
