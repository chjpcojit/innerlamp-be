package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// PasswordResetUsecase implements US-003: a code emailed to a verified account
// (request) and a new password set with that code (confirm). The code
// mechanism is the shared OTPService with purpose reset_password; this type
// adds only who may receive a code and what a verified code does.
type PasswordResetUsecase struct {
	resets    domain.PasswordResetRepository
	otp       *OTPService
	mailer    domain.AuthMailer
	passwords domain.PasswordHasher
	logger    logging.Logger

	// minResponseTime is the floor on request-otp's response time (AC25).
	minResponseTime time.Duration

	// now and sleep are the clock; tests replace them so AC25 is asserted with
	// a fake clock instead of real waiting.
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration)
}

// NewPasswordResetUsecase wires the use case from ports and config values.
func NewPasswordResetUsecase(
	resets domain.PasswordResetRepository,
	otp *OTPService,
	mailer domain.AuthMailer,
	passwords domain.PasswordHasher,
	logger logging.Logger,
	policy config.PasswordResetConfig,
) *PasswordResetUsecase {
	return &PasswordResetUsecase{
		resets:          resets,
		otp:             otp,
		mailer:          mailer,
		passwords:       passwords,
		logger:          logger,
		minResponseTime: policy.MinResponseTime.Duration(),
		now:             time.Now,
		sleep: func(ctx context.Context, d time.Duration) {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-t.C:
			case <-ctx.Done():
			}
		},
	}
}

// RequestOTP runs Steps P-2…P-5 (also serves as resend: calling again is a new
// send under the same policy).
//
// An email with no verified account gets the same body, the same send policy
// (so a 429 looks the same) and no mail (AC23/AC24). Everything after the
// format check waits out minResponseTime, errors included (AC25).
func (u *PasswordResetUsecase) RequestOTP(ctx context.Context, input domain.RequestPasswordResetInput) (*domain.RequestPasswordResetResult, error) {
	start := u.now()

	// Step P-2: no database access, so nothing to hide — answer at once (AC27).
	email, ok := validation.NormalizeEmail(input.Email)
	if !ok {
		return nil, domain.ErrInvalidEmail
	}

	err := u.requestOTP(ctx, email)

	// Step P-5.
	if wait := u.minResponseTime - u.now().Sub(start); wait > 0 {
		u.sleep(ctx, wait)
	}
	if err != nil {
		return nil, err
	}

	return &domain.RequestPasswordResetResult{
		ExpiresInSeconds:   u.otp.ExpiresInSeconds(),
		ResendAfterSeconds: u.otp.ResendAfterSeconds(),
	}, nil
}

func (u *PasswordResetUsecase) requestOTP(ctx context.Context, email string) error {
	const purpose = domain.OTPPurposeResetPassword

	// Step P-3.
	_, err := u.resets.FindVerifiedUserByEmail(ctx, email)
	hasAccount := true
	switch {
	case err == nil:
	case errors.Is(err, domain.ErrUserNotFound):
		hasAccount = false
	default:
		return fmt.Errorf("find verified user: %w", err)
	}

	// Step P-4.
	if !hasAccount {
		if _, err := u.otp.ApplySendLimit(ctx, email, purpose); err != nil {
			return err
		}
		return nil
	}

	// A fresh code would hand a guesser a new attempt counter (AC26).
	if err := u.otp.EnsureNotLocked(ctx, email, purpose); err != nil {
		return err
	}
	issued, err := u.otp.Issue(ctx, email, purpose)
	if err != nil {
		return err
	}
	if err := u.mailer.SendPasswordResetCode(ctx, email, issued.Code, u.otp.TTL()); err != nil {
		// Swallowed on purpose (except the shared budget below): a distinct
		// failure response would reveal that the email has an account. The slot is released so the failed send
		// does not count; the user presses "send again" (Known gaps).
		u.logger.Error("password reset email failed", "error", err)
		if relErr := u.otp.ReleaseSendLimit(ctx, email, purpose); relErr != nil {
			u.logger.Error("release send limit failed", "error", relErr)
		}
		if errors.Is(err, domain.ErrRateLimited) {
			// AC33: the exhausted daily budget is answered 429 with its wait.
			// Known gap: an email with no account never reaches the mailer,
			// so only this branch shows it (design.md Known gaps).
			return err
		}
	}
	return nil
}

// Confirm runs Steps C-2…C-4.
func (u *PasswordResetUsecase) Confirm(ctx context.Context, input domain.ConfirmPasswordResetInput) error {
	// Step C-2: all before any database access (AC30).
	email, ok := validation.NormalizeEmail(input.Email)
	if !ok {
		return domain.ErrInvalidEmail
	}
	if err := ValidatePassword(input.NewPassword); err != nil {
		return err
	}

	// Step C-3: reserves the attempt atomically, then compares. An email with
	// no account has no code, so it is ErrOTPExpired (AC29).
	otp, err := u.otp.Verify(ctx, email, domain.OTPPurposeResetPassword, input.OTPCode)
	if err != nil {
		return err
	}

	user, err := u.resets.FindVerifiedUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrOTPExpired
		}
		return fmt.Errorf("find verified user: %w", err)
	}

	// Step C-4.
	hash, err := u.passwords.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := u.resets.ResetPassword(ctx, user.ID, hash, otp.ID); err != nil {
		if errors.Is(err, domain.ErrOTPExpired) {
			return err
		}
		return fmt.Errorf("reset password: %w", err)
	}

	u.logger.Info("password reset", "user_id", user.ID)
	return nil
}
