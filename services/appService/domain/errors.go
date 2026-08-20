package domain

import "errors"

// Register errors — US-001 AC2/AC3/AC4.
var (
	ErrEmailTaken      = errors.New("email already in use")
	ErrPhoneTaken      = errors.New("phone number already in use")
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPhone    = errors.New("invalid phone number format, expected E.164 e.g. +84...")
	ErrInvalidPassword = errors.New("password must be at least 8 characters")
)

// Login errors — US-002 AC2/AC3. ErrUserNotFound is repository-internal;
// the login use-case must map it to ErrInvalidCredentials before it ever
// reaches delivery, so unknown-email and wrong-password look identical to
// the caller (no user-enumeration via error message).
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("email hoặc mật khẩu không đúng")
)

// ErrResetTokenInvalid covers a missing, expired, or already-used password
// reset token — US-003 AC4. The spec only requires a generic invalid/expired
// message, not that these three cases be told apart from each other.
var ErrResetTokenInvalid = errors.New("reset token is invalid or has expired")

// ErrSutraNotFound covers an unknown sutra slug — US-006 (detail view).
var ErrSutraNotFound = errors.New("sutra not found")

// ErrMantraNotFound covers an unknown mantra slug — US-008 (detail view).
var ErrMantraNotFound = errors.New("mantra not found")

// ErrBodhisattvaNotFound covers an unknown bodhisattva slug — US-010
// (detail view).
var ErrBodhisattvaNotFound = errors.New("bodhisattva not found")
