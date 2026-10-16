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
// Deliberately absent: an "email taken" error for registration. An email that
// already has an account gets the same answer as a new one (onboarding AC2), so
// no sentinel exists that could tell them apart.
var (
	// ErrPhoneTaken — the phone number already has an account. Detected by
	// PhoneTaken at request-otp (AC7) and by CreateUserAndSession's
	// unique-violation translation at verify-otp (AC18). A number held only by
	// another pending registration does NOT raise this.
	ErrPhoneTaken = errors.New("phone number already in use")

	// ErrInvalidPhone — the country is not in the BR-ONB-04 list or the number
	// has the wrong digit count for it (onboarding AC4). Detected before any DB
	// access. Login (still phone-based) reuses it for an unparsable number.
	ErrInvalidPhone = errors.New("invalid phone number")

	// ErrInvalidEmail — the normalised email fails rules/email.json (onboarding
	// AC3). Detected by the use case before any DB access.
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidPassword — the password has fewer than 8 or more than 64
	// characters after NFC (BR-ONB-06, onboarding AC5). Says nothing about the
	// submitted value.
	ErrInvalidPassword = errors.New("password must be 8 to 64 characters")

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

	// ErrOTPSendLimited — the per-(email, purpose) send cap is reached and sending is
	// locked until it lifts (AC5). Unlike the old permanent "resend exhausted"
	// state, the lock is temporary and the counter resets itself; the typed
	// OTPSendLimitedError carries the moment it lifts. Returned by the
	// repository's send-limit check, so both request-otp and resend-otp share
	// it.
	ErrOTPSendLimited = errors.New("too many otp requests, try again later")

	// ErrPendingRegistrationNotFound — no pending registration exists for the
	// email. Returned by the repository; the register use case reports it as
	// ErrPendingRegistrationExpired (AC12, AC17), so "never started" and "too
	// old" look the same to the caller.
	ErrPendingRegistrationNotFound = errors.New("pending registration not found")

	// ErrPendingRegistrationExpired — a pending registration exists but is
	// past its deadline, PendingRegistrationTTL (24h) after the latest
	// request-otp. Mapped to 400 PENDING_REGISTRATION_EXPIRED.
	ErrPendingRegistrationExpired = errors.New("pending registration has expired")

	// ErrEmailSendFailed — the code was persisted but the mailer refused the
	// send (onboarding AC10). The use case releases the send slot first, so the
	// failed send does not count; delivery maps this to 500 EMAIL_SEND_FAILED
	// rather than answering 200 for an email the user will never see.
	ErrEmailSendFailed = errors.New("failed to send email")

	// ErrInvalidCredentials — US-002's one login failure. A wrong password, an
	// email with no account, an email that exists only in
	// pending_registration and an account whose email was never verified all
	// produce this same value, with this same message, deliberately:
	// distinguishing them would turn the login endpoint into a way to
	// enumerate which emails are registered (AC2).
	//
	// The message names both fields rather than the one that was wrong, for
	// the same reason — "wrong password" alone already confirms the account
	// exists.
	ErrInvalidCredentials = errors.New("Email hoặc mật khẩu không đúng")

	// ErrLoginLocked — the email is locked out after too many consecutive
	// failures (AC5). Like ErrOTPLocked, the typed
	// LoginLockedError carries the deadline and this sentinel is what
	// errors.Is matches.
	ErrLoginLocked = errors.New("too many failed login attempts, try again later")

	// ErrRateLimited — a budget that is not per-email ran out: today the daily
	// email budget (login AC33). The typed RateLimitedError carries the wait.
	ErrRateLimited = errors.New("too many requests, try again later")

	// ErrSessionRevoked — the access token is validly signed but the session
	// its "sid" names is revoked, expired or not the token user's (login
	// AC21). Answered 401 like any other unusable token.
	ErrSessionRevoked = errors.New("session is no longer active")

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

// OTPSendLimitedError reports that the per-(email, purpose) send cap is reached and when
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

// RateLimitedError carries how long until the exhausted budget resets; the
// sentinel ErrRateLimited is what errors.Is matches.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string { return ErrRateLimited.Error() }

func (e *RateLimitedError) Unwrap() error { return ErrRateLimited }
