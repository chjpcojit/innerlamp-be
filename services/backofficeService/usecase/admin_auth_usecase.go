// Package usecase holds backofficeService's application logic.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// dummyPasswordHash is compared against when the email is unknown or the
// account is inactive, so every login path pays exactly one bcrypt comparison
// and response time does not reveal which emails exist.
var dummyPasswordHash = mustHash("timing-equalizer-not-a-password")

func mustHash(s string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("generate dummy password hash: %v", err))
	}
	return h
}

// AdminAuthUsecase authenticates backoffice operators.
type AdminAuthUsecase struct {
	admins domain.AdminRepository
	jwt    jwt.Service
	logger logging.Logger
}

// NewAdminAuthUsecase wires the use case.
func NewAdminAuthUsecase(admins domain.AdminRepository, jwtService jwt.Service, logger logging.Logger) *AdminAuthUsecase {
	return &AdminAuthUsecase{admins: admins, jwt: jwtService, logger: logger}
}

// NormalizeEmail trims and lower-cases an email; shared with cmd/seedadmin so
// stored and looked-up values agree.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Login verifies the credentials and issues an access token whose role claim
// is the admin's role.
func (u *AdminAuthUsecase) Login(ctx context.Context, input domain.AdminLoginInput) (*domain.AdminLoginResult, error) {
	email := NormalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	admin, err := u.admins.FindByEmail(ctx, email)
	switch {
	case err == nil:
	case errors.Is(err, domain.ErrAdminNotFound):
		admin = nil
	default:
		return nil, fmt.Errorf("find admin: %w", err)
	}

	hash := dummyPasswordHash
	if admin != nil {
		hash = []byte(admin.PasswordHash)
	}
	matches := bcrypt.CompareHashAndPassword(hash, []byte(input.Password)) == nil

	if admin == nil || !matches || !admin.IsActive {
		return nil, domain.ErrInvalidCredentials
	}

	// Checked only after the password matched, so a wrong guess learns nothing.
	if admin.MustChangePassword && admin.TempPasswordExpiresAt != nil && time.Now().After(*admin.TempPasswordExpiresAt) {
		return nil, domain.ErrPasswordExpired
	}

	token, err := u.issueToken(admin)
	if err != nil {
		return nil, err
	}

	// A failed bookkeeping write must not fail a login that already succeeded.
	if err := u.admins.TouchLastLogin(ctx, admin.ID, time.Now()); err != nil {
		u.logger.Warn("update admin last login failed", "admin_id", admin.ID, "error", err)
	}

	u.logger.Info("admin login succeeded", "admin_id", admin.ID)
	return &domain.AdminLoginResult{AccessToken: token, Admin: admin}, nil
}

// issueToken mints an access token; while the admin must change their password
// it carries the must_change_password claim, which the route guard honours.
func (u *AdminAuthUsecase) issueToken(admin *domain.Admin) (string, error) {
	var extra map[string]any
	if admin.MustChangePassword {
		extra = map[string]any{domain.ExtraMustChangePassword: true}
	}
	token, err := u.jwt.GenerateAccessToken(strconv.FormatInt(admin.ID, 10), admin.Email, admin.Role, extra)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return token, nil
}

// ChangePassword verifies the current password, stores the new one (clearing
// must_change_password) and returns a fresh token without the restriction.
func (u *AdminAuthUsecase) ChangePassword(ctx context.Context, input domain.ChangePasswordInput) (*domain.AdminLoginResult, error) {
	admin, err := u.admins.FindByID(ctx, input.AdminID)
	if errors.Is(err, domain.ErrAdminNotFound) {
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find admin: %w", err)
	}
	if !admin.IsActive {
		return nil, domain.ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.CurrentPassword)) != nil {
		return nil, domain.ErrWrongCurrentPassword
	}
	if len(input.NewPassword) < domain.MinPasswordLen || input.NewPassword == input.CurrentPassword {
		return nil, domain.ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if err := u.admins.UpdatePassword(ctx, admin.ID, string(hash)); err != nil {
		return nil, fmt.Errorf("update password: %w", err)
	}
	admin.MustChangePassword = false

	token, err := u.issueToken(admin)
	if err != nil {
		return nil, err
	}
	u.logger.Info("admin password changed", "admin_id", admin.ID)
	return &domain.AdminLoginResult{AccessToken: token, Admin: admin}, nil
}

// EnsureBootstrapAdmin creates the default admin, flagged must-change-password,
// when no admin exists yet. It is a no-op once any admin exists, so removing or
// renaming the default admin later never resurrects it.
func (u *AdminAuthUsecase) EnsureBootstrapAdmin(ctx context.Context, input domain.BootstrapAdminInput) error {
	email := NormalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return errors.New("bootstrap admin needs an email and a password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	created, err := u.admins.CreateBootstrapIfEmpty(ctx, email, string(hash), input.Name)
	if err != nil {
		return err
	}
	if created {
		u.logger.Warn("bootstrap admin created with the default password; it must be changed on first login", "email", email)
	}
	return nil
}

// ValidateSession reports whether a validly signed token issued at issuedAt for
// adminID is still honoured. issuedAt has 1-second precision (JWT iat), so the
// cut-off is compared at the same precision: a token minted in the very second
// of a password change is not revoked, which is what keeps the replacement
// token returned by ChangePassword usable.
func (u *AdminAuthUsecase) ValidateSession(ctx context.Context, adminID int64, issuedAt time.Time) error {
	admin, err := u.admins.FindByID(ctx, adminID)
	if errors.Is(err, domain.ErrAdminNotFound) {
		return domain.ErrSessionRevoked
	}
	if err != nil {
		return fmt.Errorf("find admin: %w", err)
	}
	if !admin.IsActive {
		return domain.ErrSessionRevoked
	}
	if admin.TokensValidAfter != nil && issuedAt.Before(admin.TokensValidAfter.Truncate(time.Second)) {
		return domain.ErrSessionRevoked
	}
	return nil
}
