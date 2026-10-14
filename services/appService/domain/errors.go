package domain

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for US-001. Callers match with errors.Is; the delivery layer
// maps each one to an HTTP status and error code, so the pairing of a sentinel
// with a status lives in one place (delivery/http/errors.go) rather than being
// re-decided per handler.
//
// Messages are written for the client, not the operator: they are what a 4xx
// body carries. Internal causes belong in a wrapped error, not in these strings.
//
// Deliberately absent: an "email not found" / "phone not found" pair. Verifying
// an OTP for a number with no pending registration returns ErrOTPInvalid, not a
// distinct not-found — a distinguishable error would let a caller probe which
// phone numbers have registrations in flight (AC6's user-enumeration concern,
// same reasoning as login's ErrInvalidCredentials).
var (
	// ErrPhoneTaken — the phone number already has an account. Detected by
	// FindConflicts (proactive, AC6) and by CreateUserAndSession's
	// unique-violation translation (the race-loser path). A pending
	// registration does NOT raise this: the same phone may restart at step 1
	// and overwrite its own payload (v5.1).
	ErrPhoneTaken = errors.New("phone number already in use")

	// ErrEmailTaken — the email belongs to an account, or to another phone's
	// in-flight registration (AC6). Detected by FindConflicts and by
	// CreateUserAndSession's unique-violation translation.
	ErrEmailTaken = errors.New("email already in use")

	// ErrInvalidPhone — the phone number survived normalization but is not a
	// usable number. Detected by the use case before any DB access (AC7).
	ErrInvalidPhone = errors.New("invalid phone number format, expected E.164 e.g. +84...")

	// ErrInvalidEmail — the email is syntactically unusable. Detected by the
	// use case before any DB access (AC7).
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidPassword — the password is below the minimum length (AC7).
	// Deliberately says only "at least 8 characters" and nothing about the
	// submitted value.
	ErrInvalidPassword = errors.New("password must be at least 8 characters")

	// ErrInvalidDharmaName — the dharma name is longer than profile.dharma_name_max_chars
	// characters after trimming and NFC normalisation. The same sentinel serves
	// registration and the profile edit, so one field has one rule and one code.
	ErrInvalidDharmaName = errors.New("dharma name is too long")

	// ErrInvalidFullName — the full name is blank, or longer than
	// profile.full_name_max_chars characters after trimming and NFC normalisation.
	// Mapped to 400 INVALID_FULL_NAME.
	ErrInvalidFullName = errors.New("full name is required and must not be too long")

	// ErrInvalidTemple — ordination_temple_id does not match any temple (AC18/
	// AC19). Mapped to 400 INVALID_TEMPLE.
	ErrInvalidTemple = errors.New("ordination temple does not exist")

	// ErrOTPInvalid — the submitted OTP is wrong, or there is no active OTP
	// to check it against (AC4). One sentinel for every "this code does not
	// work" case on purpose: distinguishing "wrong" from "already used" from
	// "never issued" tells an attacker which guesses were closer, and tells a
	// stranger whether a number is mid-registration.
	ErrOTPInvalid = errors.New("otp code is invalid")

	// ErrOTPExpired — the OTP was valid but its window has closed (AC4).
	// Separate from ErrOTPInvalid because it is not a guess: the caller did
	// nothing wrong, it just took too long, so a "request a new code" prompt
	// is the useful response rather than "check what you typed". Returned by
	// verify-otp only — resend-otp treats a gone code as "issue a new one"
	// (v5.1).
	ErrOTPExpired = errors.New("otp code has expired")

	// ErrOTPLocked — the OTP is locked after too many wrong guesses (AC12).
	// The typed OTPLockedError carries the moment the lock lifts; this plain
	// sentinel is what errors.Is matches and what a caller with no interest
	// in the timestamp checks.
	ErrOTPLocked = errors.New("too many failed attempts, try again later")

	// ErrOTPResendTooSoon — the send cooldown has not elapsed (AC5). Applies to
	// both request-otp and resend-otp. The typed OTPResendTooSoonError carries
	// the remaining seconds.
	ErrOTPResendTooSoon = errors.New("please wait before requesting another code")

	// ErrOTPSendLimited — the per-phone send cap is reached and sending is
	// locked until it lifts (AC5). Unlike the old permanent "resend exhausted"
	// state, the lock is temporary and the counter resets itself; the typed
	// OTPSendLimitedError carries the moment it lifts. Returned by the
	// repository's send-limit check, so both request-otp and resend-otp share
	// it.
	ErrOTPSendLimited = errors.New("too many otp requests, try again later")

	// ErrPendingRegistrationNotFound — no pending registration exists for
	// the phone number. Returned by the repository and surfaced by delivery
	// as ErrOTPInvalid's status, never as a 404: a 404 would confirm which
	// numbers are mid-registration.
	ErrPendingRegistrationNotFound = errors.New("pending registration not found")

	// ErrPendingRegistrationExpired — a pending registration exists but is
	// past its deadline. Since v5.1 that deadline is PendingRegistrationTTL
	// (24h), independent of the OTP's 90s, so this no longer implies the code
	// expired too — it means the whole attempt is old enough to discard.
	ErrPendingRegistrationExpired = errors.New("pending registration has expired")

	// ErrUnsupportedOTPPurpose — resend was asked for a purpose US-001 does
	// not implement. Only OTPPurposeRegister is wired; the other three
	// purposes are declared for US-002/US-003 and rejected here rather than
	// silently issuing a code no flow can consume.
	ErrUnsupportedOTPPurpose = errors.New("unsupported otp purpose")

	// ErrSMSSendFailed — the OTP was persisted but the SMS provider refused
	// the send. The account state is already committed at this point, yet a
	// registration whose code never arrived is not a success: delivery maps
	// this to 500 SMS_SEND_FAILED rather than returning 200 for a message the
	// user will never see. A distinct sentinel (not a generic internal error)
	// so the handler can pick that code without matching on message text.
	ErrSMSSendFailed = errors.New("failed to send otp via sms")

	// ErrInvalidCredentials — US-002's one login failure. A wrong password, a
	// phone number with no account, and a phone number that exists only in
	// pending_registration all produce this same value, with this same
	// message, deliberately: distinguishing them would turn the login endpoint
	// into a way to enumerate which numbers are registered (AC2).
	//
	// The message names both fields rather than the one that was wrong, for
	// the same reason — "wrong password" alone already confirms the account
	// exists.
	ErrInvalidCredentials = errors.New("Số điện thoại hoặc mật khẩu không đúng")

	// ErrLoginLocked — the phone number is locked out after too many
	// consecutive failures (AC7). Like ErrOTPLocked, the typed
	// LoginLockedError carries the deadline and this sentinel is what
	// errors.Is matches.
	ErrLoginLocked = errors.New("too many failed login attempts, try again later")

	// ErrInvalidDevice — the request carried no usable device description,
	// missing platform or device_model (AC5). Same category as
	// ErrInvalidPhone: a malformed request, not a guess, so it neither counts
	// against the brute-force limit nor touches the database.
	ErrInvalidDevice = errors.New("invalid device information")

	// ErrRefreshTokenInvalid — US-002's one refresh failure (AC10). A token
	// that is malformed, badly signed, expired, already rotated, belonging to
	// a revoked session, or matching no session at all all produce this one
	// code, so a caller cannot learn which of those it was.
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid")

	// ErrUserNotFound — no users row for the lookup key. Internal to the
	// repository/usecase boundary: login never lets it reach the handler, it
	// maps it to ErrInvalidCredentials and keeps going through the bcrypt
	// comparison so the response time does not betray the miss (AC2).
	ErrUserNotFound = errors.New("user not found")

	// ErrAttestationChallengeInvalid — the challenge (nonce) referenced by a
	// verify or assertion is unknown, expired, or already used (AC6/AC14).
	// One sentinel for all three states on purpose: a distinguishable response
	// would let a caller probe which challenges are live. The verify endpoint
	// maps it to 400; the assertion middleware maps it to 401.
	ErrAttestationChallengeInvalid = errors.New("attestation challenge is invalid")

	// ErrAttestationUpstreamError — the attestation backend (Google's Play
	// Integrity decode) could not be reached or timed out (AC7). The verify
	// endpoint maps it to 502, and the challenge is NOT consumed so the caller
	// can retry the same nonce.
	ErrAttestationUpstreamError = errors.New("attestation upstream error")

	// ErrAttestationKeyNotFound — no registered key for the key_id an
	// assertion named (AC13). The assertion middleware maps it to 401 with
	// reason unknown_key.
	ErrAttestationKeyNotFound = errors.New("attestation key not found")

	// ErrAttestationCounterNotIncreased — the assertion's authenticator counter
	// is not strictly greater than the stored one, i.e. a replay or rollback
	// (AC12). The assertion middleware maps it to 401 with reason
	// counter_not_increased, and the stored counter is NOT updated.
	ErrAttestationCounterNotIncreased = errors.New("attestation counter did not increase")

	// ErrAttestationAssertionInvalid — the assertion signature or RP ID did not
	// verify against the stored public key (design.md assertion flow Step 3).
	// The assertion middleware maps it to 401 with reason assertion_invalid.
	ErrAttestationAssertionInvalid = errors.New("attestation assertion is invalid")
)

// OTPLockedError reports that an OTP is locked and when it unlocks.
//
// The timestamp cannot live on the sentinel — sentinels are shared, immutable
// values, and the lock deadline is per-row. Error() returns the sentinel's
// message and Unwrap() returns the sentinel itself, so both styles work:
// errors.Is(err, ErrOTPLocked) for the category, errors.As for the deadline.
type OTPLockedError struct {
	LockedUntil time.Time
}

func (e *OTPLockedError) Error() string { return ErrOTPLocked.Error() }

func (e *OTPLockedError) Unwrap() error { return ErrOTPLocked }

// OTPResendTooSoonError reports that the resend cooldown is still running and
// how many seconds remain, for the Retry-After the client needs.
//
// Same shape as OTPLockedError: the sentinel carries the classification, this
// typed error carries the number, and errors.Is still sees through Unwrap.
type OTPResendTooSoonError struct {
	RetryAfterSeconds int
}

func (e *OTPResendTooSoonError) Error() string {
	return fmt.Sprintf("%s (%d seconds remaining)", ErrOTPResendTooSoon, e.RetryAfterSeconds)
}

func (e *OTPResendTooSoonError) Unwrap() error { return ErrOTPResendTooSoon }

// OTPSendLimitedError reports that the per-phone send cap is reached and when
// sending unlocks — AC5, and the payload behind the 429's locked_until detail.
//
// Same shape as OTPLockedError for the same reason: the sentinel is a shared
// immutable value and cannot carry a per-row deadline, so the classification
// lives on the sentinel and the timestamp on this typed error, with Unwrap
// connecting the two for errors.Is.
type OTPSendLimitedError struct {
	LockedUntil time.Time
}

func (e *OTPSendLimitedError) Error() string { return ErrOTPSendLimited.Error() }

func (e *OTPSendLimitedError) Unwrap() error { return ErrOTPSendLimited }

// LoginLockedError reports that a phone number is locked out and when the lock
// lifts — AC7, and the payload behind the 429's locked_until detail.
//
// Same shape as OTPLockedError for the same reason: the sentinel is a shared
// immutable value and cannot carry a per-row deadline, so the classification
// lives on the sentinel and the timestamp on this typed error, with Unwrap
// connecting the two for errors.Is.
type LoginLockedError struct {
	LockedUntil time.Time
}

func (e *LoginLockedError) Error() string { return ErrLoginLocked.Error() }

func (e *LoginLockedError) Unwrap() error { return ErrLoginLocked }
