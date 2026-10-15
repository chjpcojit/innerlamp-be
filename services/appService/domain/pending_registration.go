package domain

import "time"

// PendingRegistration holds everything needed to create a User once the email
// is verified. It maps the pending_registration table.
//
// This struct is what keeps US-001's central invariant: no users row ever
// exists in an unverified state. The account payload — including the password
// hash, never the plaintext — waits here between the request-otp and
// verify-otp steps, and the users row is inserted only after the code matches.
//
// One row per email (email is UNIQUE, migration 0037): a fresh request for the
// same email overwrites the payload only when no live code exists or the
// payload is the same (AC24). PhoneNumber is NOT unique here — two
// pending registrations may hold the same number, and whoever verifies first
// keeps it (onboarding AC7, AC18).
//
// AccountExists marks a row created for an email that already has an account
// (AC2): the caller got the same answer as a new email, the owner got a notice
// instead of a code, and verify-otp treats every code as wrong (AC16).
//
// ExpiresAt is PendingRegistrationTTL (24h) after the latest request-otp.
type PendingRegistration struct {
	ID           int64
	Email        string
	PhoneNumber  string
	PasswordHash string
	FullName     *string
	DharmaName   *string
	// OrdinationTempleID is temples.id chosen at step 1, nil when none (v7.0).
	OrdinationTempleID *string
	AccountExists      bool
	ExpiresAt          time.Time
	CreatedAt          time.Time
}
