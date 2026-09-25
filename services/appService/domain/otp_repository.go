package domain

import (
	"context"
	"time"
)

// OTPRepository persists the OTP mechanism's own state — the otp_codes rows,
// their wrong-guess counters, and the per-(phone_number, purpose) send limits —
// and nothing else.
//
// It is deliberately separate from RegistrationRepository: a future OTP flow
// (reset_password, login-by-OTP, verify_email) injects this port and gets the
// whole mechanism without also depending on register's FindConflicts /
// CreateUserAndSession. The concrete Postgres repository implements both
// interfaces from one struct, so this split is a dependency-boundary concern,
// not a data-location one.
//
// Every method scopes its row by the pair (phone_number, purpose), so two
// active purposes on one phone number never share an otp_codes row, a send
// budget, or a wrong-guess lockout. This interface knows nothing about
// pending_registration or users — it neither reads nor writes them.
type OTPRepository interface {
	// IssueOTP applies the send policy, invalidates the active OTP for
	// phoneNumber/purpose, and inserts a fresh one — all in one transaction.
	//
	// Like SavePendingAndIssueOTP it applies the send policy first, inside the
	// same transaction, and returns *OTPResendTooSoonError /
	// *OTPSendLimitedError when the send is refused (writing nothing). Unlike
	// SavePendingAndIssueOTP it never touches pending_registration. "Issue"
	// rather than "reissue": this is the generic send path — resend is just one
	// more issue, and a future feature's first send goes through here too.
	IssueOTP(ctx context.Context, phoneNumber string, purpose OTPPurpose, otpHash string, expiresAt time.Time, limits SendLimits) (*SendState, error)

	// FindActiveOTP loads the most recent OTP for phoneNumber/purpose that a
	// verify or resend call should act on: neither used nor expired, or still
	// holding its wrong-guess lock (so the caller learns the deadline). Returns
	// ErrOTPInvalid when there is none, so the caller cannot tell "never
	// issued" from "already used" from "lock lifted".
	FindActiveOTP(ctx context.Context, phoneNumber string, purpose OTPPurpose) (*OTPCode, error)

	// IncrementOTPAttempts records one wrong OTP guess against otpID and
	// returns the new total. It returns the count rather than taking a
	// threshold so the rule for when to lock stays with the caller.
	IncrementOTPAttempts(ctx context.Context, otpID int64) (int, error)

	// LockOTP marks otpID unusable until lockedUntil, and also marks it used
	// so it can never be replayed. Locking is not a counter the caller can
	// forget to consult; the row itself is dead.
	LockOTP(ctx context.Context, otpID int64, lockedUntil time.Time) error

	// ReleaseSendLimit drops the send counter for phoneNumber/purpose, so a
	// completed flow leaves no rate-limit state behind. Correctness does not
	// depend on it (the counter also ages out), but a fresh flow for the same
	// number should not inherit a stale one.
	ReleaseSendLimit(ctx context.Context, phoneNumber string, purpose OTPPurpose) error
}
