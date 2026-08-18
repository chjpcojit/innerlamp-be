package jwt

import (
	"crypto/rsa"
	"fmt"
	"time"
)

// Algorithm represents the signing algorithm
type Algorithm string

const (
	// AlgorithmHS256 represents HMAC-SHA256 algorithm (symmetric)
	AlgorithmHS256 Algorithm = "HS256"

	// AlgorithmRS256 represents RSA-SHA256 algorithm (asymmetric)
	AlgorithmRS256 Algorithm = "RS256"
)

// Config holds the JWT configuration
type Config struct {
	// Algorithm specifies the signing algorithm (HS256 or RS256)
	Algorithm Algorithm

	// SecretKey is used for HS256 algorithm (symmetric key)
	SecretKey string

	// PrivateKey is used for RS256 algorithm signing (asymmetric)
	PrivateKey *rsa.PrivateKey

	// PublicKey is used for RS256 algorithm verification (asymmetric)
	PublicKey *rsa.PublicKey

	// AccessTokenExpiration is the duration for access token validity
	AccessTokenExpiration time.Duration

	// RefreshTokenExpiration is the duration for refresh token validity
	RefreshTokenExpiration time.Duration

	// Issuer is the token issuer (iss claim)
	Issuer string

	// Audience is the intended audience (aud claim)
	Audience string
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate algorithm
	if c.Algorithm != AlgorithmHS256 && c.Algorithm != AlgorithmRS256 {
		return fmt.Errorf("%w: must be HS256 or RS256", ErrUnsupportedAlgorithm)
	}

	// Validate keys based on algorithm
	switch c.Algorithm {
	case AlgorithmHS256:
		if c.SecretKey == "" {
			return fmt.Errorf("%w: secret key is required for HS256", ErrInvalidKey)
		}
		if len(c.SecretKey) < 32 {
			return fmt.Errorf("%w: secret key must be at least 32 characters for HS256", ErrInvalidKey)
		}
	case AlgorithmRS256:
		if c.PrivateKey == nil {
			return fmt.Errorf("%w: private key is required for RS256 signing", ErrInvalidKey)
		}
		if c.PublicKey == nil {
			return fmt.Errorf("%w: public key is required for RS256 verification", ErrInvalidKey)
		}
	}

	// Validate expiration times
	if c.AccessTokenExpiration <= 0 {
		return fmt.Errorf("access token expiration must be positive")
	}
	if c.RefreshTokenExpiration <= 0 {
		return fmt.Errorf("refresh token expiration must be positive")
	}
	if c.RefreshTokenExpiration <= c.AccessTokenExpiration {
		return fmt.Errorf("refresh token expiration must be greater than access token expiration")
	}

	// Issuer and audience are optional but recommended
	// No validation needed

	return nil
}

// DefaultConfig returns a default configuration for HS256
// Note: This should NOT be used in production. Generate secure keys instead.
func DefaultConfig(secretKey string) *Config {
	return &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              secretKey,
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour, // 7 days
		Issuer:                 "base_project",
		Audience:               "base_project_users",
	}
}
