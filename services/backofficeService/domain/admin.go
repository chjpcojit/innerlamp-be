// Package domain holds backofficeService's entities, repository ports and
// sentinel errors. It imports nothing from usecase, repository or delivery.
package domain

import (
	"context"
	"errors"
	"time"
)

// Backoffice roles; the value is the "role" claim on an admin token. Only
// RoleAdmin may manage accounts. There is no permission model beyond this yet
// (RBAC is a later step), so the other roles are labels for now.
const (
	RoleAdmin         = "admin"
	RoleContentEditor = "content_editor"
	RoleViewer        = "viewer"
)

// ValidRole reports whether role is one the backoffice knows.
func ValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleContentEditor, RoleViewer:
		return true
	}
	return false
}

var (
	// ErrInvalidCredentials covers unknown email, wrong password and a
	// deactivated account alike, so a caller cannot probe which emails exist.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAdminNotFound — no admins row for the lookup key. Internal to the use
	// case; never reaches a client.
	ErrAdminNotFound = errors.New("admin not found")

	// ErrPasswordChangeRequired — the token belongs to an admin who must set a
	// new password before using any other route.
	ErrPasswordChangeRequired = errors.New("password change required")

	// ErrSessionRevoked — the token is validly signed but no longer honoured:
	// issued before the admin's tokens_valid_after, or the admin is gone/inactive.
	ErrSessionRevoked = errors.New("session revoked")

	// ErrInvalidAccountInput — create-account input failed validation (bad email,
	// empty name, unknown role); the message says which.
	ErrInvalidAccountInput = errors.New("invalid account input")

	// ErrEmailTaken — an admins row already uses that email.
	ErrEmailTaken = errors.New("email already in use")

	// ErrInviteEmailFailed — the account was not created because its invite
	// email could not be sent.
	ErrInviteEmailFailed = errors.New("could not send the invite email")

	// ErrPasswordExpired — the temporary password is correct but past its
	// expiry; an admin must reset it.
	ErrPasswordExpired = errors.New("temporary password expired")

	// ErrCannotResetSelf — an admin may not reset their own password through
	// the reset route; they use change-password.
	ErrCannotResetSelf = errors.New("cannot reset your own password")

	// ErrWrongCurrentPassword — change-password was given a bad current password.
	ErrWrongCurrentPassword = errors.New("current password is incorrect")

	// ErrWeakPassword — the new password is too short or equals the current one.
	ErrWeakPassword = errors.New("new password is too weak")
)

// MinPasswordLen is the minimum length of a password an admin may set.
const MinPasswordLen = 12

// ExtraMustChangePassword is the access-token extra claim set while the admin
// must change their password.
const ExtraMustChangePassword = "must_change_password"

// Admin is a backoffice operator account.
type Admin struct {
	ID           int64
	Email        string
	PasswordHash string
	Name         string
	Role         string
	IsActive     bool
	// MustChangePassword is true for the bootstrap admin until they set a new
	// password.
	MustChangePassword bool
	// TokensValidAfter, when set, revokes every token issued before it.
	TokensValidAfter *time.Time
	// TempPasswordExpiresAt, when set on a must-change account, is when its
	// temporary password stops working.
	TempPasswordExpiresAt *time.Time
	LastLoginAt           *time.Time
}

// AdminLoginInput is the input to the login use case.
type AdminLoginInput struct {
	Email    string
	Password string
}

// AdminLoginResult is what a successful login returns.
type AdminLoginResult struct {
	AccessToken string
	Admin       *Admin
}

// ChangePasswordInput is the input to the change-password use case.
type ChangePasswordInput struct {
	AdminID         int64
	CurrentPassword string
	NewPassword     string
}

// BootstrapAdminInput describes the default admin created on first startup.
type BootstrapAdminInput struct {
	Email    string
	Password string
	Name     string
}

// CreateAccountInput is the input to the create-account use case.
type CreateAccountInput struct {
	Email string
	Name  string
	Role  string
}

// CreateAccountResult is the new account. Its temporary password is emailed to
// it and deliberately not returned to the caller.
type CreateAccountResult struct {
	Admin *Admin
}

// AdminRepository is the persistence port for admin accounts.
type AdminRepository interface {
	// FindByEmail returns ErrAdminNotFound when no row matches.
	FindByEmail(ctx context.Context, email string) (*Admin, error)
	// TouchLastLogin records a successful login.
	TouchLastLogin(ctx context.Context, id int64, at time.Time) error
	// FindByID returns ErrAdminNotFound when no row matches.
	FindByID(ctx context.Context, id int64) (*Admin, error)
	// UpdatePassword stores a new hash, clears must_change_password and the
	// temp-password expiry, and sets
	// tokens_valid_after to now, revoking every earlier token.
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	// CreateBootstrapIfEmpty inserts a must_change_password admin only when the
	// admins table is empty, atomically. created is false when it was not empty.
	CreateBootstrapIfEmpty(ctx context.Context, email, passwordHash, name string) (created bool, err error)
	// DeleteByID removes an account; used to roll back a just-created account
	// whose invite email could not be sent.
	DeleteByID(ctx context.Context, id int64) error
	// CreateAccount inserts an account with the given role,
	// must_change_password = TRUE and the temp-password expiry (nil = none).
	// Returns ErrEmailTaken on a duplicate email.
	CreateAccount(ctx context.Context, email, passwordHash, name, role string, tempExpiresAt *time.Time) (*Admin, error)
	// ResetTempPassword replaces the password with a temporary one: sets
	// must_change_password, the expiry, and tokens_valid_after = now (revoking
	// every session). Returns ErrAdminNotFound when no row matches.
	ResetTempPassword(ctx context.Context, id int64, passwordHash string, tempExpiresAt *time.Time) (*Admin, error)
	// Create inserts a new admin; used by cmd/seedadmin.
	Create(ctx context.Context, email, passwordHash, name string) (*Admin, error)
}
