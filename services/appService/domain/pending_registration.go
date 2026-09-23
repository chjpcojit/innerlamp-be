package domain

import "time"

// PendingRegistration holds everything needed to create a User once the phone
// number is verified. It maps the pending_registration table.
//
// This struct is what keeps US-001's central invariant: no users row ever
// exists in an unverified state. The account payload — including the bcrypt
// password hash, never the plaintext — waits here between the request-otp and
// verify-otp steps, and the users row is inserted only after the OTP matches.
//
// One row per phone number (phone_number is UNIQUE): a fresh request for the
// same number overwrites the payload. Email is deliberately NOT unique here —
// only a completed users row is authoritative for email uniqueness, and the
// final insert re-checks it inside its own transaction.
//
// ExpiresAt is set to the OTP's expiry at request time and pushed forward on
// each resend, so an abandoned registration disappears without a cleanup job.
// Because it equals the OTP TTL, an expired pending row and an expired OTP
// always go together — a caller that finds one expired can treat the whole
// attempt as dead and ask for email and password again.
type PendingRegistration struct {
	ID              int64
	PhoneNumber     string
	Email           string
	PasswordHash    string
	DharmaName      *string
	OrdinationPlace *string
	ExpiresAt       time.Time
	CreatedAt       time.Time
}
