package domain

import "time"

// User maps to the users table (docs/db/register-erd.md).
type User struct {
	ID              int64
	Email           string
	PasswordHash    string
	PhoneNumber     string
	DharmaName      string
	OrdinationPlace string
	AvatarURL       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Device maps to the user_devices table — one row per device a user has
// registered/logged in from.
type Device struct {
	ID          int64
	UserID      int64
	DeviceID    string // fingerprint hash, computed server-side
	Platform    string
	OSVersion   string
	AppVersion  string
	DeviceModel string
	DeviceName  string
	IPAddress   string
	UserAgent   string
	CreatedAt   time.Time
}

// Session maps to the user_sessions table and holds the JWT token pair
// issued for one device.
type Session struct {
	ID             int64
	UserID         int64
	DeviceID       int64
	SessionID      string
	AccessToken    string
	RefreshToken   string
	TokenVersion   int
	IsActive       bool
	IsRevoked      bool
	LastActivityAt time.Time
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
