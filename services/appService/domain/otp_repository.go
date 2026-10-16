package domain

import (
	"context"
	"time"
)

// OTPRepository persists the OTP mechanism's own state — the otp_codes rows,
// their wrong-guess counters, and the per-(email, purpose) send limits — and
// nothing else (otp-mechanism AC11).
//
// It is deliberately separate from RegistrationRepository: password reset
// injects this port and gets the whole mechanism without depending on
// register's account creation. The concrete Postgres repository implements
// both interfaces from one struct, so this split is a dependency-boundary
// concern, not a data-location one.
//
// Every method scopes its row by the pair (email, purpose), with the email
// already normalised by the caller (otp-mechanism AC1).
type OTPRepository interface {
	// IssueOTP applies the send policy, invalidates the active code for
	// email/purpose, and inserts a fresh one — all in one transaction
	// (otp-mechanism Step I-2). A refused send writes nothing and returns
	// *OTPResendTooSoonError / *OTPSendLimitedError.
	IssueOTP(ctx context.Context, email string, purpose OTPPurpose, otpHash string, expiresAt time.Time, limits SendLimits) (*SendState, error)

	// ApplySendLimit applies exactly IssueOTP's send policy without creating a
	// code, for a caller that must look like it sent one (password reset for
	// an email with no account, otp-mechanism AC6).
	ApplySendLimit(ctx context.Context, email string, purpose OTPPurpose, limits SendLimits) (*SendState, error)

	// FindActiveOTP loads the most recent code for email/purpose that a verify
	// should act on: unused, unexpired and never locked, or still holding its
	// wrong-guess lock (so the caller learns the deadline). A row whose lock
	// has lifted is dead. Returns ErrOTPInvalid when there is none.
	FindActiveOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTPCode, error)

	// ConsumeOTPAttempt reserves one verify attempt on otpID atomically, before
	// the code is compared (otp-mechanism Step V-3):
	//
	//	UPDATE otp_codes SET attempts = attempts + 1,
	//	       locked_until = CASE WHEN attempts + 1 >= max THEN lockUntil ELSE locked_until END
	//	WHERE id = otpID AND used_at IS NULL AND attempts < max
	//	RETURNING attempts
	//
	// ok is false when no attempt could be reserved (the cap was already
	// reached, or the row was consumed), which the caller reports as locked.
	// Concurrent verifies therefore can never exceed maxAttempts in total.
	ConsumeOTPAttempt(ctx context.Context, otpID int64, maxAttempts int, lockUntil time.Time) (attempts int, ok bool, err error)

	// ReleaseSendLimit drops the send counter for email/purpose, so a send
	// that failed after its slot was taken does not count (otp-mechanism AC10).
	ReleaseSendLimit(ctx context.Context, email string, purpose OTPPurpose) error

	// InvalidateActiveOTP marks every unused, unlocked code for email/purpose as
	// used. A caller whose email never left (the mailer failed after the code
	// was committed) retires the code the user never received, so a corrected
	// resubmission is not mistaken for a clash with a live code. Having no live
	// code is not an error. Mã đã commit nhưng thư không gửi được thì phải vô
	// hiệu, nếu không form sửa lại gửi lên sẽ bị coi là đụng mã còn sống.
	InvalidateActiveOTP(ctx context.Context, email string, purpose OTPPurpose) error
}
