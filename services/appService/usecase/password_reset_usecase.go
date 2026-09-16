package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/mailer"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// resetTokenTTL is the reset link/code validity window — US-003 AC1
// ("có hiệu lực trong thời gian giới hạn (ví dụ 15 phút)").
const resetTokenTTL = 15 * time.Minute

// minPasswordLength is the shortest password the reset flow accepts. The
// delivery layer already rejects a shorter one through the request struct's
// `validate:"min=8"` tag; this is the backstop for Reset being called
// directly, not a replacement for it.
const minPasswordLength = 8

// PasswordResetUsecase implements US-003 (Quên mật khẩu).
type PasswordResetUsecase struct {
	users  domain.UserRepository
	mailer mailer.Mailer
	logger logging.Logger

	// newResetToken generates the raw (unhashed) reset token. Overridable
	// in tests for deterministic assertions; defaults to a
	// crypto/rand-backed generator.
	newResetToken func() (string, error)
}

// NewPasswordResetUsecase wires the password-reset use-case.
func NewPasswordResetUsecase(users domain.UserRepository, m mailer.Mailer, logger logging.Logger) *PasswordResetUsecase {
	return &PasswordResetUsecase{
		users:         users,
		mailer:        m,
		logger:        logger,
		newResetToken: generateResetToken,
	}
}

func generateResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// hashResetToken hashes a raw reset token the same way on issue and on
// redemption — only the hash is ever persisted (US-003 "Ghi chú kỹ thuật").
func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ForgotPassword issues a reset token and emails it. It always reports
// success regardless of whether the email is registered — AC2, the same
// no-enumeration rule as US-002 AC2/AC3. Only a genuine infrastructure
// failure (not "user not found") is returned as an error; a mail-send
// failure is logged but still treated as success, so it can't be used to
// distinguish "email exists" from "email doesn't exist" either.
func (uc *PasswordResetUsecase) ForgotPassword(ctx context.Context, input domain.ForgotPasswordInput) error {
	email := validation.NormalizeEmail(input.Email)

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("find user: %w", err)
	}

	rawToken, err := uc.newResetToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	if _, err := uc.users.CreatePasswordResetToken(ctx, user.ID, hashResetToken(rawToken), time.Now().Add(resetTokenTTL)); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}

	if err := uc.mailer.SendPasswordResetEmail(ctx, user.Email, rawToken); err != nil {
		uc.logger.WithContext(ctx).WithError(err).Error("failed to send password reset email")
		return nil
	}

	uc.logger.WithContext(ctx).WithFields(logging.Fields{"user_id": user.ID}).Info("password reset email sent")
	return nil
}

// Reset validates the new password and redeems the reset token, rotating
// the account's password and revoking every existing session — AC3/AC4.
func (uc *PasswordResetUsecase) Reset(ctx context.Context, input domain.ResetPasswordInput) error {
	if len(input.NewPassword) < minPasswordLength {
		return domain.ErrInvalidPassword
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := uc.users.ResetPassword(ctx, hashResetToken(input.Token), string(newHash)); err != nil {
		return err
	}

	uc.logger.WithContext(ctx).Info("password reset completed")
	return nil
}
