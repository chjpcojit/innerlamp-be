package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/mailer"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// tempPasswordBytes of entropy → a 16-character URL-safe temporary password.
const tempPasswordBytes = 12

// AdminUserUsecase manages backoffice accounts. Who may call it is decided at
// the route (admin role only); it holds no caller identity.
type AdminUserUsecase struct {
	admins domain.AdminRepository
	mailer mailer.Mailer
	// fixedTempPassword, when set, replaces the random temporary password.
	// ⚠️ DEV ONLY, see config.AccountConfig.
	fixedTempPassword string
	// tempPasswordTTL is how long a temporary password works; 0 = no expiry.
	tempPasswordTTL time.Duration
	logger          logging.Logger
}

// NewAdminUserUsecase wires the use case. Pass an empty fixedTempPassword for a
// random per-account password (the only safe value outside dev), and a zero
// tempPasswordTTL for temporary passwords that never expire.
func NewAdminUserUsecase(admins domain.AdminRepository, m mailer.Mailer, fixedTempPassword string, tempPasswordTTL time.Duration, logger logging.Logger) *AdminUserUsecase {
	return &AdminUserUsecase{admins: admins, mailer: m, fixedTempPassword: fixedTempPassword, tempPasswordTTL: tempPasswordTTL, logger: logger}
}

// tempExpiry is when a temporary password issued now stops working; nil when
// expiry is disabled.
func (u *AdminUserUsecase) tempExpiry() *time.Time {
	if u.tempPasswordTTL <= 0 {
		return nil
	}
	t := time.Now().Add(u.tempPasswordTTL)
	return &t
}

// newTempPassword returns a temporary password and its bcrypt hash.
func (u *AdminUserUsecase) newTempPassword() (password, hash string, err error) {
	password, err = u.tempPassword()
	if err != nil {
		return "", "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash temporary password: %w", err)
	}
	return password, string(h), nil
}

// CreateAccount creates an account flagged must_change_password and emails it
// the temporary password, so that password only works to reach change-password.
// The password is never returned to the caller. If the email cannot be sent the
// account is deleted again, so no account exists that nobody can sign in to.
func (u *AdminUserUsecase) CreateAccount(ctx context.Context, input domain.CreateAccountInput) (*domain.CreateAccountResult, error) {
	email := NormalizeEmail(input.Email)
	name := strings.TrimSpace(input.Name)
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("%w: email is not valid", domain.ErrInvalidAccountInput)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidAccountInput)
	}
	if !domain.ValidRole(input.Role) {
		return nil, fmt.Errorf("%w: unknown role %q", domain.ErrInvalidAccountInput, input.Role)
	}

	password, hash, err := u.newTempPassword()
	if err != nil {
		return nil, err
	}

	admin, err := u.admins.CreateAccount(ctx, email, hash, name, input.Role, u.tempExpiry())
	if err != nil {
		return nil, err
	}

	if err := u.mailer.SendAccountInviteEmail(ctx, admin.Email, name, password); err != nil {
		u.logger.Error("send account invite failed; rolling back account", "admin_id", admin.ID, "error", err)
		if delErr := u.admins.DeleteByID(ctx, admin.ID); delErr != nil {
			u.logger.Error("rollback of account failed", "admin_id", admin.ID, "error", delErr)
		}
		return nil, domain.ErrInviteEmailFailed
	}

	u.logger.Info("admin account created", "admin_id", admin.ID, "role", admin.Role)
	return &domain.CreateAccountResult{Admin: admin}, nil
}

func (u *AdminUserUsecase) tempPassword() (string, error) {
	if u.fixedTempPassword != "" {
		return u.fixedTempPassword, nil
	}
	raw := make([]byte, tempPasswordBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate temporary password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// ResetPassword replaces an account's password with a new temporary one and
// emails it: the account must change it again, the new password expires like
// any other, and every session the account had is revoked. It is how an
// account whose temporary password expired (or was lost) gets back in. An
// admin may not reset themselves; they use change-password. If the email fails
// the account keeps the new, unsent password, so the admin should retry.
func (u *AdminUserUsecase) ResetPassword(ctx context.Context, actorID, targetID int64) (*domain.CreateAccountResult, error) {
	if actorID == targetID {
		return nil, domain.ErrCannotResetSelf
	}
	password, hash, err := u.newTempPassword()
	if err != nil {
		return nil, err
	}
	admin, err := u.admins.ResetTempPassword(ctx, targetID, hash, u.tempExpiry())
	if err != nil {
		return nil, err
	}
	if err := u.mailer.SendAccountInviteEmail(ctx, admin.Email, admin.Name, password); err != nil {
		u.logger.Error("send password reset invite failed", "admin_id", admin.ID, "error", err)
		return nil, domain.ErrInviteEmailFailed
	}
	u.logger.Info("admin password reset", "admin_id", admin.ID, "by", actorID)
	return &domain.CreateAccountResult{Admin: admin}, nil
}
