package domain

import (
	"context"
	"time"
)

// RequestRegisterOTPInput is the use-case input for step 1 of US-001 —
// POST /v1/auth/register/request-otp. It carries the whole account payload
// because step 1 is the only step that receives it: step 2 sends back nothing
// but the email and the code.
type RequestRegisterOTPInput struct {
	Email    string
	Password string
	// PhoneCountry is the ISO 3166 code picked in the client (BR-ONB-04);
	// PhoneNumber is the number as typed. Together they give the E.164 form.
	PhoneCountry string
	PhoneNumber  string
	FullName     string
	DharmaName   string
	// OrdinationTempleID is an optional temples.id; blank means "no temple".
	OrdinationTempleID string
}

// RequestOTPResult reports the code window just opened, so the handler can put
// the numbers in the 200 body without hardcoding them. They come from
// config.OTPConfig through OTPService; a handler that spelled them itself would
// silently disagree with the policy the moment either changed.
type RequestOTPResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
}

// ResendOTPInput is the use-case input for POST /v1/auth/register/resend-otp.
// Only the email: the payload already waits in pending_registration.
type ResendOTPInput struct {
	Email string
}

// ResendOTPResult reports the new code window plus how many counted sends are
// left before the send cap (AC11).
type ResendOTPResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
	ResendRemaining    int
}

// VerifyRegisterOTPInput is the use-case input for step 2 of US-001 —
// POST /v1/auth/register/verify-otp. Only these two fields: everything else was
// stored in PendingRegistration at step 1 and is read from there, so the client
// never resends (or gets to change) the password or phone at this point.
type VerifyRegisterOTPInput struct {
	Email   string
	OTPCode string
}

// RegisterResult is what a verified registration returns. The account is
// created already logged in (AC14), so the session tokens come back from the
// same call that created the user.
type RegisterResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

// TokenIssuer generates an access/refresh token pair for a persisted user ID
// and the session the pair belongs to.
// It is supplied by the use case so the repository can mint tokens after the
// user row's ID is known, inside the same transaction that also writes the
// session row.
//
// sessionID is chosen by the repository BEFORE minting and is the exact value
// it then stores as user_sessions.session_id. The access token carries it as
// claim "sid", which is how logout finds the session to revoke; minting first
// and generating the ID afterwards would leave the token pointing at nothing.
//
// Without this callback the ID would only exist after commit, forcing either a
// second transaction (a user row that can exist with no session if the second
// one fails) or a pre-generated ID. domain never imports pkg/jwt — the closure
// the use case passes in is the only thing that knows how a token is built.
type TokenIssuer func(userID, sessionID string) (accessToken, refreshToken string, err error)

// RegistrationRepository persists the register flow's own state: the pending
// registration payload, the user row, and the login session that registration
// creates (AC14).
//
// The generic OTP mechanism (otp_codes rows, wrong-guess counters, send limits)
// lives behind domain.OTPRepository instead. The one OTP-shaped write that stays
// here is SavePendingAndIssueOTP, because it must commit the pending payload and
// a fresh code in the *same* transaction — an atomicity a standalone
// OTPRepository.IssueOTP cannot express.
//
// Every write that must not be observed half-done is a single method wrapping
// one transaction, rather than a set of primitives the use case sequences.
type RegistrationRepository interface {
	// PhoneTaken reports whether the E.164 phone number already belongs to an
	// account (AC7). A pending registration holding the same number does not
	// count: only users.phone_number is unique.
	PhoneTaken(ctx context.Context, phoneNumber string) (bool, error)

	// EmailHasAccount reports whether the normalised email already belongs to
	// an account. The answer only picks which email step 1 sends (AC2); it is
	// never surfaced to the caller.
	EmailHasAccount(ctx context.Context, email string) (bool, error)

	// TempleExists reports whether temples has a row with this id. Used by
	// request-otp to reject an unknown ordination_temple_id early (AC8); the
	// FK on users/pending_registration is the final guard.
	TempleExists(ctx context.Context, id string) (bool, error)

	// SavePendingAndIssueOTP stores the account payload under pending.Email and
	// issues a fresh code for (email, register), in one transaction (Step R-7).
	//
	// Inside the transaction the email is serialised (advisory lock + FOR
	// UPDATE on the pending row), then: if a live code exists (otp_codes row
	// for (email, register) with used_at NULL and expires_at in the future)
	// and canReplace(existing) is false, nothing is written and
	// *RegistrationInProgressError is returned (AC24). Only then is the send
	// policy applied — a refused send writes nothing and returns
	// *OTPResendTooSoonError / *OTPSendLimitedError. Then the payload is
	// UPSERTed on email (AccountExists included), any still-active code for the
	// pair is invalidated, and the new code is inserted. No users row is
	// written.
	//
	// canReplace decides "same payload" (the repository must not hash or
	// compare passwords). It is called at most once, only when a pending row
	// and a live code both exist, with the locked row. nil means "always".
	SavePendingAndIssueOTP(ctx context.Context, pending *PendingRegistration, otpHash string, expiresAt time.Time, limits SendLimits, canReplace func(existing *PendingRegistration) bool) (*SendState, error)

	// FindPendingRegistration loads the payload waiting for the normalised
	// email. Returns ErrPendingRegistrationNotFound if none exists, and
	// ErrPendingRegistrationExpired if it is past ExpiresAt.
	FindPendingRegistration(ctx context.Context, email string) (*PendingRegistration, error)

	// CreateUserAndSession consumes the pending payload and turns it into a
	// real account plus its first login session, in one transaction (Step V-5).
	//
	// Inside that transaction it marks otpID used (ErrOTPExpired if a
	// concurrent verify already did), inserts the users row with
	// email_verified_at = now(), deletes the pending row and the send counter,
	// and writes the session with sessionTTL. issueTokens is called between
	// the insert and the session write. A phone number taken meanwhile returns
	// ErrPhoneTaken, a vanished temple ErrInvalidTemple; either rolls back.
	CreateUserAndSession(ctx context.Context, pending *PendingRegistration, otpID int64, sessionTTL time.Duration, issueTokens TokenIssuer) (*RegisterResult, error)

	// DeletePendingRegistration drops the payload for pendingID. Exported
	// because abandoning a registration is a real operation, though the
	// production verify path does not need it: CreateUserAndSession deletes
	// the pending row as part of its own transaction.
	DeletePendingRegistration(ctx context.Context, pendingID int64) error
}
