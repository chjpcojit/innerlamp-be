package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"golang.org/x/crypto/bcrypt"
)

// otpKeyspace is the number of distinct six-digit codes. Fixed by the code's
// format (six digits, leading zeros allowed), not a deployment setting.
const otpKeyspace = 1_000_000

// OTPService is the purpose-agnostic OTP mechanism (otp-mechanism v2.0): it
// generates codes, hashes them, applies the send policy, and verifies codes —
// reserving each attempt atomically and locking a code after too many — without
// knowing anything about registration or password reset.
//
// It does not send anything. The caller emails the raw code it gets back,
// because the wording differs per flow (register code, account-exists notice,
// reset code), and releases the send slot when that email fails
// (otp-mechanism AC10). It never touches pending_registration or users
// (otp-mechanism AC11).
type OTPService struct {
	repo   domain.OTPRepository
	logger logging.Logger
	policy config.OTPConfig

	// now is the clock. time.Now in production; tests in this package replace
	// it to pin expiry and lock deadlines.
	now func() time.Time
}

// NewOTPService wires the shared OTP mechanism. policy comes from
// config.OTPConfig; DevFixedCode is honoured whenever set, because
// AppConfig.Validate already refuses it outside app.environment "dev"
// (otp-mechanism AC3).
func NewOTPService(repo domain.OTPRepository, logger logging.Logger, policy config.OTPConfig) *OTPService {
	return &OTPService{repo: repo, logger: logger, policy: policy, now: time.Now}
}

// OTPIssueResult is what a successful Issue hands back: the clear code for the
// caller to email, and the window numbers for the response body.
type OTPIssueResult struct {
	Code               string
	ExpiresInSeconds   int
	ResendAfterSeconds int
	ResendRemaining    int
}

// Generate produces a fresh code and its bcrypt hash (otp-mechanism Step I-1).
// They are produced together so a caller can never hash one code and email
// another.
//
// The code is six digits drawn uniformly from crypto/rand, leading zeros kept
// (AC2), unless policy.DevFixedCode is set (dev only, AC3).
func (s *OTPService) Generate() (code, hash string, err error) {
	code, err = s.newCode()
	if err != nil {
		return "", "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash otp: %w", err)
	}
	return code, string(h), nil
}

func (s *OTPService) newCode() (string, error) {
	if s.policy.DevFixedCode != "" {
		return s.policy.DevFixedCode, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(otpKeyspace))
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// EnsureNotLocked returns *domain.OTPLockedError when email has an active code
// for purpose whose wrong-guess lock is still in force. A missing code means
// "not locked".
//
// request-otp and resend-otp call this before issuing: a fresh code would
// otherwise hand a guesser a new attempt counter and make the lock of AC15
// worthless.
func (s *OTPService) EnsureNotLocked(ctx context.Context, email string, purpose domain.OTPPurpose) error {
	otp, err := s.repo.FindActiveOTP(ctx, email, purpose)
	if err != nil && !errors.Is(err, domain.ErrOTPInvalid) {
		return fmt.Errorf("find active otp: %w", err)
	}
	if otp != nil && isLocked(otp, s.now()) {
		return &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}
	return nil
}

// Issue runs the shared issue flow for email/purpose: generate a code, then in
// one transaction apply the send policy, invalidate the active code and insert
// the new one (otp-mechanism AC4). It does not send the code.
func (s *OTPService) Issue(ctx context.Context, email string, purpose domain.OTPPurpose) (*OTPIssueResult, error) {
	code, hash, err := s.Generate()
	if err != nil {
		return nil, err
	}

	state, err := s.repo.IssueOTP(ctx, email, purpose, hash, s.CodeExpiry(s.now()), s.SendLimits())
	if err != nil {
		return nil, fmt.Errorf("issue otp: %w", err)
	}

	return &OTPIssueResult{
		Code:               code,
		ExpiresInSeconds:   s.ExpiresInSeconds(),
		ResendAfterSeconds: s.ResendAfterSeconds(),
		ResendRemaining:    state.Remaining,
	}, nil
}

// ApplySendLimit applies exactly Issue's send policy without creating a code
// (otp-mechanism AC6), for a caller that must look like it sent one.
func (s *OTPService) ApplySendLimit(ctx context.Context, email string, purpose domain.OTPPurpose) (*domain.SendState, error) {
	state, err := s.repo.ApplySendLimit(ctx, email, purpose, s.SendLimits())
	if err != nil {
		return nil, fmt.Errorf("apply send limit: %w", err)
	}
	return state, nil
}

// ReleaseSendLimit drops the send counter for email/purpose so a send that
// failed after its slot was taken does not count (otp-mechanism AC10).
func (s *OTPService) ReleaseSendLimit(ctx context.Context, email string, purpose domain.OTPPurpose) error {
	if err := s.repo.ReleaseSendLimit(ctx, email, purpose); err != nil {
		return fmt.Errorf("release send limit: %w", err)
	}
	return nil
}

// Verify runs otp-mechanism Steps V-1…V-4: find the active code, refuse it when
// locked or expired, reserve one attempt atomically, and only then compare the
// hash. On success it returns the row; the caller marks it used inside its own
// transaction (AC9).
//
// A wrong code is always ErrOTPInvalid, never "n tries left" (AC8): that number
// is exactly what someone brute-forcing a 10^6 keyspace would pace themselves
// by.
func (s *OTPService) Verify(ctx context.Context, email string, purpose domain.OTPPurpose, code string) (*domain.OTPCode, error) {
	otp, err := s.repo.FindActiveOTP(ctx, email, purpose)
	if err != nil {
		if errors.Is(err, domain.ErrOTPInvalid) {
			return nil, domain.ErrOTPExpired
		}
		return nil, fmt.Errorf("find active otp: %w", err)
	}

	now := s.now()
	if isLocked(otp, now) {
		return nil, &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}
	if !otp.ExpiresAt.After(now) {
		return nil, domain.ErrOTPExpired
	}

	lockUntil := now.Add(s.policy.LockDuration.Duration())
	_, ok, err := s.repo.ConsumeOTPAttempt(ctx, otp.ID, s.policy.MaxAttempts, lockUntil)
	if err != nil {
		return nil, fmt.Errorf("consume otp attempt: %w", err)
	}
	if !ok {
		// No attempt left to reserve: a concurrent guess took the last one and
		// set the lock. Report that lock; the row read above predates it.
		if otp.LockedUntil != nil && otp.LockedUntil.After(now) {
			lockUntil = *otp.LockedUntil
		}
		return nil, &domain.OTPLockedError{LockedUntil: lockUntil}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.OTPHash), []byte(code)); err != nil {
		return nil, domain.ErrOTPInvalid
	}
	return otp, nil
}

// CodeExpiry returns the moment a code generated at now expires. Exposed so
// register's SavePendingAndIssueOTP can persist the same deadline Issue uses.
func (s *OTPService) CodeExpiry(now time.Time) time.Time {
	return now.Add(s.policy.TTL.Duration())
}

// TTL is the code lifetime, passed to the mailer so the email states it.
func (s *OTPService) TTL() time.Duration {
	return s.policy.TTL.Duration()
}

// SendLimits projects the OTP policy onto the repository's SendLimits shape.
// One method so the three send-policy numbers can never be threaded
// inconsistently.
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

// isLocked reports whether the row's lock is still in force. A nil deadline
// means the row was never locked.
func isLocked(otp *domain.OTPCode, now time.Time) bool {
	return otp.LockedUntil != nil && otp.LockedUntil.After(now)
}
