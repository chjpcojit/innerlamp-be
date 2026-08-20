package domain

import (
	"context"
	"time"
)

// DeviceInfo is the device metadata collected at register time (US-002 AC4:
// each device keeps its own login state).
type DeviceInfo struct {
	Platform    string
	OSVersion   string
	AppVersion  string
	DeviceModel string
	DeviceName  string
	IPAddress   string
	UserAgent   string
}

// RegisterInput is the use-case input for US-001.
type RegisterInput struct {
	Email           string
	Password        string
	PhoneNumber     string
	DharmaName      string
	OrdinationPlace string
	Device          DeviceInfo
}

// RegisterResult is returned to the caller after a successful registration —
// AC1/AC7: register logs the user in immediately.
type RegisterResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

// TokenIssuer generates an access/refresh token pair for a persisted user
// ID. It is supplied by the use-case so the repository can mint tokens
// after the user row's ID is known, inside the same transaction that also
// creates the device and session rows.
type TokenIssuer func(userID string) (accessToken, refreshToken string, err error)

// UserRepository persists a new user together with the device it registered
// from and the login session issued for that device, atomically.
type UserRepository interface {
	Register(ctx context.Context, user *User, device *Device, issueTokens TokenIssuer) (*Session, error)

	// FindByEmail looks up a user by their (already normalized) email.
	// Returns ErrUserNotFound if no such user exists — US-002 AC3.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// CreateSession upserts the device row for userID (by device
	// fingerprint) and issues a new session for it, atomically — US-002
	// AC4: each device keeps its own session, so logging in from a new
	// device never disturbs another device's session row.
	CreateSession(ctx context.Context, userID int64, device *Device, issueTokens TokenIssuer) (*Session, error)

	// CreatePasswordResetToken persists a new reset token for userID —
	// US-003 AC1. Only tokenHash (never the raw token) is stored.
	CreatePasswordResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error)

	// ResetPassword atomically validates tokenHash (must exist, be
	// unexpired, and unused — AC4), updates the account's password,
	// marks the token used, and revokes every active session for the
	// account — AC3. Returns ErrResetTokenInvalid if the token doesn't
	// pass validation.
	ResetPassword(ctx context.Context, tokenHash, newPasswordHash string) error

	// FindByID looks up a user by primary key — US-004 AC1 (profile view).
	// Returns ErrUserNotFound if no such user exists.
	FindByID(ctx context.Context, userID int64) (*User, error)

	// UpdateProfile overwrites dharma_name/ordination_place for userID —
	// US-004 AC2/AC5. An empty field in input clears that column.
	UpdateProfile(ctx context.Context, userID int64, input ProfileUpdateInput) (*User, error)

	// UpdateAvatarURL persists a new avatar_url for userID, replacing any
	// previous value — US-004 AC3.
	UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) (*User, error)
}
