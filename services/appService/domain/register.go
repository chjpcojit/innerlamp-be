package domain

import (
	"context"
	"time"
)

// RequestRegisterOTPInput is the use-case input for step 1 of US-001 —
// POST /v1/auth/register/request-otp. It carries the whole account payload
// because step 1 is the only step that receives it: step 2 sends back nothing
// but the phone number and the OTP.
type RequestRegisterOTPInput struct {
	Email       string
	Password    string
	PhoneNumber string
	FullName    string
	DharmaName  string
	// OrdinationTempleID is an optional temples.id; blank means "no temple".
	OrdinationTempleID string
}

// RequestOTPResult reports the OTP window just opened, so the handler can put
// the numbers in the 200 body without hardcoding them.
//
// These live here rather than as constants in delivery because the TTLs are a
// use-case decision (they gate the resend cooldown and the lockout), and the
// response is the only place they are observable. A handler that spelled
// "90"/"60" itself would silently disagree with the use case the moment either
// changed.
type RequestOTPResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
}

// ResendOTPInput is the use-case input for POST /v1/auth/resend-otp. The same
// endpoint serves every OTP flow, so Purpose selects which one — US-001
// implements only OTPPurposeRegister and rejects the rest.
type ResendOTPInput struct {
	PhoneNumber string
	Purpose     OTPPurpose
}

// ResendOTPResult reports the new OTP window plus how many resends are left, so
// the caller learns it is approaching the cap (AC5) before hitting it.
type ResendOTPResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
	ResendRemaining    int
}

// VerifyRegisterOTPInput is the use-case input for step 2 of US-001 —
// POST /v1/auth/register/verify-otp. Only these two fields: everything else was
// stored in PendingRegistration at step 1 and is read from there, so the client
// never resends (or gets to change) the email or password at this point.
type VerifyRegisterOTPInput struct {
	PhoneNumber string
	OTPCode     string
}

// RegisterResult is what a verified registration returns. The account is
// created already logged in (AC1/AC7), so the session tokens come back from the
// same call that created the user.
type RegisterResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

// RegistrationConflicts reports which unique keys a registration attempt would
// collide with. It is a struct rather than separate calls because step 1 needs
// every answer from one query — checking them in separate round-trips would
// leave a window where the second check runs against changed data.
//
// A pending_registration row for the *same* phone is deliberately NOT a
// conflict: it is that caller's own in-flight attempt, and request-otp is
// allowed to overwrite it (the write UPSERTs on phone_number) so someone who
// reinstalled the app can start over without waiting out
// PendingRegistrationTTL. A live pending row only counts against the email, and
// only when it belongs to a *different* phone — that is somebody else's claim
// on a globally-unique key (AC6).
type RegistrationConflicts struct {
	PhoneInUsers   bool
	EmailInUsers   bool
	EmailInPending bool
}

// HasPhoneConflict reports whether the phone number already belongs to a real
// account (AC6).
func (c RegistrationConflicts) HasPhoneConflict() bool {
	return c.PhoneInUsers
}

// HasEmailConflict reports whether the email is already spoken for — by a real
// account (AC6) or by another phone's in-flight registration.
func (c RegistrationConflicts) HasEmailConflict() bool {
	return c.EmailInUsers || c.EmailInPending
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
// creates (AC1/AC7).
//
// The generic OTP mechanism (otp_codes rows, wrong-guess counters, send limits)
// lives behind domain.OTPRepository instead, so a future OTP flow can inject
// that narrower port without dragging register's conflict-checking and
// account-creation along. The one OTP-shaped write that stays here is
// SavePendingAndIssueOTP, because it must commit the pending payload and a fresh
// OTP in the *same* transaction — an atomicity that a standalone
// OTPRepository.IssueOTP cannot express.
//
// Every write that must not be observed half-done is a single method wrapping
// one transaction, rather than a set of primitives the use case sequences —
// otherwise a crash between two calls would leave a pending row with no OTP, or
// a user with no session.
type RegistrationRepository interface {
	// FindConflicts reports which unique keys the phone number and email
	// already occupy — AC6. A phone or email belonging to a real account is
	// always a conflict. A live pending_registration row is a conflict only for
	// the email, and only when that row belongs to a *different* phone: the
	// same phone's own pending row is the same caller's in-flight attempt and
	// request-otp overwrites it (v5.1). Only unexpired rows count, so an
	// abandoned attempt stops holding an email once its TTL passes.
	FindConflicts(ctx context.Context, phoneNumber, email string) (*RegistrationConflicts, error)

	// TempleExists reports whether temples has a row with this id. Used by
	// request-otp to reject an unknown ordination_temple_id early (AC18); the
	// FK on users/pending_registration is the final guard.
	TempleExists(ctx context.Context, id string) (bool, error)

	// SavePendingAndIssueOTP stores the account payload under phoneNumber and
	// issues a fresh OTP for it, in one transaction — step 1 of AC1.
	//
	// The payload is UPSERTed on phone_number and any still-active OTP for that
	// number is invalidated. Before either, the per-phone send policy is
	// applied (design.md §"OTP send-limit policy"): a blocked attempt writes
	// nothing and returns *OTPResendTooSoonError / *OTPSendLimitedError, so a
	// caller cannot burn a send slot without actually receiving a code. The
	// returned SendState tells the use case how many sends remain.
	//
	// No users row is written here: that is the whole point of the two-step
	// flow.
	SavePendingAndIssueOTP(ctx context.Context, pending *PendingRegistration, otpHash string, expiresAt time.Time, limits SendLimits) (*SendState, error)

	// FindPendingRegistration loads the payload waiting for phoneNumber.
	// Returns ErrPendingRegistrationNotFound if none exists, and
	// ErrPendingRegistrationExpired if the stored payload is past
	// ExpiresAt — the caller treats both as "start over from step 1".
	FindPendingRegistration(ctx context.Context, phoneNumber string) (*PendingRegistration, error)

	// CreateUserAndSession consumes the pending payload and turns it into a
	// real account plus its first login session, in one transaction —
	// AC2/AC3/AC7.
	//
	// Inside that transaction it re-checks the email, inserts the users row
	// with email_verified_at NULL, deletes the pending row, marks otpID used,
	// and writes the session with sessionTTL. issueTokens is called between
	// the insert and the session write, so the tokens describe a user ID the
	// transaction has not yet committed. Returns ErrPhoneTaken/ErrEmailTaken
	// if the re-check loses a race.
	CreateUserAndSession(ctx context.Context, pending *PendingRegistration, otpID int64, sessionTTL time.Duration, issueTokens TokenIssuer) (*RegisterResult, error)

	// DeletePendingRegistration drops the payload for pendingID. Exported
	// because abandoning a registration is a real operation, though the
	// production verify path does not need it: CreateUserAndSession deletes
	// the pending row as part of its own transaction.
	DeletePendingRegistration(ctx context.Context, pendingID int64) error
}

// OTPSender delivers a one-time password to a phone number.
//
// One method for every purpose rather than one per flow: resend uses the same
// delivery path as the first send, and US-002/US-003 will call this same method
// with a different purpose. The use case depends on this port instead of
// pkg/sms so that domain stays free of concrete infrastructure and tests can
// assert delivery without a network.
type OTPSender interface {
	// SendOTP delivers otpCode to phoneNumber. An error means the message was
	// not handed off, and the use case turns it into a failed request rather
	// than a success the user never receives — the account state was already
	// committed, but a registration the caller cannot complete is not a
	// success.
	SendOTP(ctx context.Context, phoneNumber, otpCode string, purpose OTPPurpose) error
}
