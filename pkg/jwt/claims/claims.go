package claims

import (
	"github.com/golang-jwt/jwt/v5"
)

// TokenType represents the type of token
type TokenType string

const (
	// TokenTypeAccess represents an access token
	TokenTypeAccess TokenType = "access"

	// TokenTypeRefresh represents a refresh token
	TokenTypeRefresh TokenType = "refresh"
)

// CustomClaims represents the custom claims for JWT tokens
type CustomClaims struct {
	// UserID is the unique identifier for the user
	UserID string `json:"user_id"`

	// Email is the user's email address
	Email string `json:"email,omitempty"`

	// Role is the user's role (e.g., "admin", "user", "guest")
	Role string `json:"role,omitempty"`

	// TokenType indicates whether this is an access or refresh token
	TokenType TokenType `json:"token_type"`

	// Additional custom fields can be added by services
	Extra map[string]any `json:"extra,omitempty"`

	// Embed standard JWT claims
	jwt.RegisteredClaims
}

// GetExpirationTime implements jwt.Claims interface
func (c CustomClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.ExpiresAt, nil
}

// GetIssuedAt implements jwt.Claims interface
func (c CustomClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.IssuedAt, nil
}

// GetNotBefore implements jwt.Claims interface
func (c CustomClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.NotBefore, nil
}

// GetIssuer implements jwt.Claims interface
func (c CustomClaims) GetIssuer() (string, error) {
	return c.RegisteredClaims.Issuer, nil
}

// GetSubject implements jwt.Claims interface
func (c CustomClaims) GetSubject() (string, error) {
	return c.RegisteredClaims.Subject, nil
}

// GetAudience implements jwt.Claims interface
func (c CustomClaims) GetAudience() (jwt.ClaimStrings, error) {
	return c.RegisteredClaims.Audience, nil
}

// Validate validates the custom claims (custom validation beyond standard claims)
func (c CustomClaims) Validate() error {
	// Validate custom fields
	if c.UserID == "" {
		return jwt.ErrInvalidType
	}

	if c.TokenType != TokenTypeAccess && c.TokenType != TokenTypeRefresh {
		return jwt.ErrInvalidType
	}

	return nil
}

// GetExtra retrieves a value from the extra fields
func (c CustomClaims) GetExtra(key string) (any, bool) {
	if c.Extra == nil {
		return nil, false
	}
	val, ok := c.Extra[key]
	return val, ok
}

// SetExtra sets a value in the extra fields
func (c *CustomClaims) SetExtra(key string, value any) {
	if c.Extra == nil {
		c.Extra = make(map[string]any)
	}
	c.Extra[key] = value
}
