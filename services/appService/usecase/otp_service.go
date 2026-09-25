package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"golang.org/x/crypto/bcrypt"
)

// OTPService is the purpose-agnostic OTP mechanism: it generates codes, hashes
// them, sends them, and verifies them — counting wrong guesses and locking a
// code after too many — without knowing anything about the register flow.
//
// It owns the OTP policy (config.OTPConfig), the generic persistence port
// (domain.OTPRepository) and the delivery port (domain.OTPSender). It never
// touches pending_registration or users: whether a user exists, what the
// account payload is, and who creates the session are a caller's concern, not
// this service's. A future OTP flow (reset_password, login-by-OTP, verify_email)
// injects this same type and reuses the whole mechanism.
type OTPService struct {
	repo   domain.OTPRepository
	sender domain.OTPSender
	logger logging.Logger
	policy config.OTPConfig
}

// NewOTPService wires the shared OTP mechanism. Every collaborator is a port or
// a plain config value; no concrete repository, sender or signing implementation
// is visible here.
func NewOTPService(
	repo domain.OTPRepository,
	sender domain.OTPSender,
	logger logging.Logger,
	policy config.OTPConfig,
) *OTPService {
	return &OTPService{
		repo:   repo,
		sender: sender,
		logger: logger,
		policy: policy,
	}
}

// OTPIssueResult reports the outcome of a successful Issue so the caller can
// describe the window it just opened without hardcoding the numbers.
type OTPIssueResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
	ResendRemaining    int
}

// Generate produces a fresh code and its bcrypt hash. The code is what gets
// sent to the user; the hash is what gets persisted. They are produced together
// so a caller can never hash one code and send another.
//
// The code is a constant today ("111111" in dev) and that is a deliberate
// placeholder, not an oversight — isolated so switching to a cryptographically
// random six-digit code touches nothing else. Every caller treats the code as
// opaque.
func (s *OTPService) Generate() (code, hash string, err error) {
	code = generateOTP()
	h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash otp: %w", err)
	}
	return code, string(h), nil
}

// Send delivers code to phoneNumber for purpose, translating any sender failure
// into ErrSMSSendFailed.
//
// The failure is reported rather than swallowed even though the OTP row is
// already committed: a flow whose code never arrived is not a success, and
// answering success would leave the caller waiting for an SMS that is not
// coming. The wrapped sentinel is what lets delivery pick the SMS_SEND_FAILED
// code without matching on message text; the cause is kept for the log only.
func (s *OTPService) Send(ctx context.Context, phoneNumber, code string, purpose domain.OTPPurpose) error {
	if err := s.sender.SendOTP(ctx, phoneNumber, code, purpose); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrSMSSendFailed, err)
	}
	return nil
}

// EnsureNotLocked returns *domain.OTPLockedError when phoneNumber has an active
// OTP for purpose whose wrong-guess lock is still in force. A missing OTP means
// "not locked".
//
// The request-otp and resend-otp flows call this before issuing a fresh code:
// while a number is locked out after too many wrong guesses, a fresh code would
// hand the attacker a new attempt counter, so the send is refused instead.
func (s *OTPService) EnsureNotLocked(ctx context.Context, phoneNumber string, purpose domain.OTPPurpose) error {
	otp, err := s.repo.FindActiveOTP(ctx, phoneNumber, purpose)
	if err != nil && !errors.Is(err, domain.ErrOTPInvalid) {
		return fmt.Errorf("find active otp: %w", err)
	}
	if otp != nil && isLocked(otp, time.Now()) {
		return &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}
	return nil
}

// Issue runs the shared send flow for phoneNumber/purpose: generate a code and
// hash, persist the row (applying the send policy), then send the code. It does
// not read or write pending_registration or users.
func (s *OTPService) Issue(ctx context.Context, phoneNumber string, purpose domain.OTPPurpose) (*OTPIssueResult, error) {
	code, hash, err := s.Generate()
	if err != nil {
		return nil, err
	}

	state, err := s.repo.IssueOTP(ctx, phoneNumber, purpose, hash, s.CodeExpiry(time.Now()), s.SendLimits())
	if err != nil {
		return nil, wrapSendError("issue otp", err)
	}

	if err := s.Send(ctx, phoneNumber, code, purpose); err != nil {
		return nil, err
	}

	return &OTPIssueResult{
		ExpiresInSeconds:   s.ExpiresInSeconds(),
		ResendAfterSeconds: s.ResendAfterSeconds(),
		ResendRemaining:    state.Remaining,
	}, nil
}

// Verify runs the shared check flow: find the active OTP, refuse if its
// wrong-guess lock holds, compare the code against the hash, and — on a miss —
// record the attempt and lock the code on the Nth consecutive miss. On success
// it returns the verified row so the caller can consume it as its own
// completion step (e.g. register marks it used while creating the account).
func (s *OTPService) Verify(ctx context.Context, phoneNumber string, purpose domain.OTPPurpose, code string) (*domain.OTPCode, error) {
	otp, err := s.repo.FindActiveOTP(ctx, phoneNumber, purpose)
	if err != nil {
		if errors.Is(err, domain.ErrOTPInvalid) {
			return nil, domain.ErrOTPExpired
		}
		return nil, fmt.Errorf("find active otp: %w", err)
	}

	// Checked before the comparison: while the lock holds, the code is not
	// examined and no attempt is recorded, so an attacker can neither extend
	// the lock by guessing nor keep trying for free.
	if isLocked(otp, time.Now()) {
		return nil, &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.OTPHash), []byte(code)); err != nil {
		return nil, s.recordFailedAttempt(ctx, otp)
	}

	return otp, nil
}

// recordFailedAttempt counts a wrong code and locks the number once the limit
// is reached, then reports the failure.
//
// The returned error is always ErrOTPInvalid and never says how many tries
// remain: that number is exactly what someone brute-forcing a 10^6 keyspace
// would use to pace themselves. Once the count crosses MaxAttempts the lock is
// set for LockDuration from this failure, not from the first one.
func (s *OTPService) recordFailedAttempt(ctx context.Context, otp *domain.OTPCode) error {
	attempts, err := s.repo.IncrementOTPAttempts(ctx, otp.ID)
	if err != nil {
		// The guess was wrong either way, so the caller still gets the same
		// answer; the failure to record it is what an operator needs to see.
		s.logger.Error("failed to record otp attempt", "error", err, "otp_id", otp.ID)
		return domain.ErrOTPInvalid
	}

	if attempts >= s.policy.MaxAttempts {
		if err := s.repo.LockOTP(ctx, otp.ID, time.Now().Add(s.policy.LockDuration.Duration())); err != nil {
			s.logger.Error("failed to lock otp", "error", err, "otp_id", otp.ID)
		}
	}

	return domain.ErrOTPInvalid
}

// CodeExpiry returns the moment a code generated at now expires. It is exposed
// so a caller that persists the OTP row through its own repository method
// (register's SavePendingAndIssueOTP) can compute the same deadline Issue uses.
func (s *OTPService) CodeExpiry(now time.Time) time.Time {
	return now.Add(s.policy.TTL.Duration())
}

// SendLimits projects the OTP policy onto the repository's SendLimits shape.
// Kept as one method so the three send-policy numbers can never be threaded
// inconsistently (e.g. a cap paired with the wrong lock duration).
func (s *OTPService) SendLimits() domain.SendLimits {
	return domain.SendLimits{
		Cooldown:     s.policy.SendCooldown.Duration(),
		LockDuration: s.policy.SendLockDuration.Duration(),
		MaxAttempts:  s.policy.MaxSendAttempts,
	}
}

// ExpiresInSeconds reports the code lifetime in whole seconds.
func (s *OTPService) ExpiresInSeconds() int {
	return int(s.policy.TTL.Duration().Seconds())
}

// ResendAfterSeconds reports the send cooldown in whole seconds.
func (s *OTPService) ResendAfterSeconds() int {
	return int(s.policy.SendCooldown.Duration().Seconds())
}

// generateOTP returns the code sent to the user.
//
// It is a constant today (US-001 technical notes: 111111 in dev) and that is a
// deliberate placeholder, not an oversight — isolated in this one function so
// that switching to a cryptographically random six-digit code touches nothing
// else. Every caller treats the return value as opaque.
func generateOTP() string {
	return "111111"
}

// isLocked reports whether the row's lock is still in force. A nil deadline
// means the row was never locked.
func isLocked(otp *domain.OTPCode, now time.Time) bool {
	return otp.LockedUntil != nil && otp.LockedUntil.After(now)
}

// wrapSendError passes the send policy's typed refusals through unchanged and
// wraps everything else as infrastructure. The typed errors are business
// outcomes delivery maps to 429 with locked_until / retry_after_seconds;
// folding them into a generic message would make the handler answer 500 for a
// request that was merely too soon.
func wrapSendError(op string, err error) error {
	if _, ok := errors.AsType[*domain.OTPResendTooSoonError](err); ok {
		return err
	}
	if _, ok := errors.AsType[*domain.OTPSendLimitedError](err); ok {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}
