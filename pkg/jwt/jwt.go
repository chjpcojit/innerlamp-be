package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt/claims"

	"github.com/golang-jwt/jwt/v5"
)

// Service defines the interface for JWT operations
type Service interface {
	// GenerateAccessToken generates a new access token
	GenerateAccessToken(userID, email, role string, extra map[string]any) (string, error)

	// GenerateRefreshToken generates a new refresh token
	GenerateRefreshToken(userID string, extra map[string]any) (string, error)

	// ValidateToken validates a token and returns the claims
	ValidateToken(tokenString string) (*claims.CustomClaims, error)

	// ParseToken parses a token without validation (use with caution)
	ParseToken(tokenString string) (*claims.CustomClaims, error)

	// RefreshAccessToken generates a new access token from a refresh token
	RefreshAccessToken(refreshToken string, email, role string) (string, error)
}

// tokenIDBytes is the entropy of a jti: 128 bits, so a collision between two
// tokens is not a practical concern.
const tokenIDBytes = 16

// newTokenID returns a fresh jti (JWT ID, RFC 7519 §4.1.7): 128 random bits from
// crypto/rand as 32 lowercase hex characters. It is a package variable so a test
// can inject an entropy failure; production code never reassigns it.
var newTokenID = func() (string, error) {
	b := make([]byte, tokenIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// jwtService implements the Service interface
type jwtService struct {
	config *Config
}

// New creates a new JWT service
func New(config *Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid JWT config: %w", err)
	}

	return &jwtService{
		config: config,
	}, nil
}

// GenerateAccessToken generates a new access token
func (s *jwtService) GenerateAccessToken(userID, email, role string, extra map[string]any) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID is required")
	}

	tokenID, err := newTokenID()
	if err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(s.config.AccessTokenExpiration)

	customClaims := &claims.CustomClaims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: claims.TokenTypeAccess,
		Extra:     extra,
		RegisteredClaims: jwt.RegisteredClaims{
			// jti makes every token unique: iat/exp only have 1-second precision, so
			// without it two tokens for the same user minted in the same second are
			// byte-identical (see docs/specs/features/jwt-token-uniqueness).
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.config.Audience},
		},
	}

	return s.signToken(customClaims)
}

// GenerateRefreshToken generates a new refresh token
func (s *jwtService) GenerateRefreshToken(userID string, extra map[string]any) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID is required")
	}

	tokenID, err := newTokenID()
	if err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(s.config.RefreshTokenExpiration)

	customClaims := &claims.CustomClaims{
		UserID:    userID,
		TokenType: claims.TokenTypeRefresh,
		Extra:     extra,
		RegisteredClaims: jwt.RegisteredClaims{
			// jti makes every token unique: iat/exp only have 1-second precision, so
			// without it two tokens for the same user minted in the same second are
			// byte-identical (see docs/specs/features/jwt-token-uniqueness).
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.config.Audience},
		},
	}

	return s.signToken(customClaims)
}

// ValidateToken validates a token and returns the claims
func (s *jwtService) ValidateToken(tokenString string) (*claims.CustomClaims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &claims.CustomClaims{}, func(token *jwt.Token) (any, error) {
		// Verify the algorithm to prevent algorithm confusion attacks
		if err := s.validateAlgorithm(token); err != nil {
			return nil, err
		}

		// Return the appropriate key based on the algorithm
		return s.getVerificationKey(), nil
	})

	if err != nil {
		return nil, s.mapTokenError(err)
	}

	// Extract and validate claims
	if customClaims, ok := token.Claims.(*claims.CustomClaims); ok && token.Valid {
		return customClaims, nil
	}

	return nil, ErrInvalidClaims
}

// ParseToken parses a token without full validation (use with caution)
// This is useful for extracting claims from expired tokens or in specific scenarios
func (s *jwtService) ParseToken(tokenString string) (*claims.CustomClaims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	// Parse without validation
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &claims.CustomClaims{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Check algorithm even in unverified parsing
	if err := s.validateAlgorithm(token); err != nil {
		return nil, err
	}

	if customClaims, ok := token.Claims.(*claims.CustomClaims); ok {
		return customClaims, nil
	}

	return nil, ErrInvalidClaims
}

// RefreshAccessToken generates a new access token from a refresh token
func (s *jwtService) RefreshAccessToken(refreshToken string, email, role string) (string, error) {
	// Validate the refresh token
	refreshClaims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify it's a refresh token
	if refreshClaims.TokenType != claims.TokenTypeRefresh {
		return "", ErrInvalidTokenType
	}

	// Generate a new access token with the user ID from refresh token
	return s.GenerateAccessToken(refreshClaims.UserID, email, role, refreshClaims.Extra)
}

// signToken signs the token with the configured algorithm
func (s *jwtService) signToken(customClaims *claims.CustomClaims) (string, error) {
	var token *jwt.Token

	switch s.config.Algorithm {
	case AlgorithmHS256:
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
		return token.SignedString([]byte(s.config.SecretKey))

	case AlgorithmRS256:
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, customClaims)
		return token.SignedString(s.config.PrivateKey)

	default:
		return "", ErrUnsupportedAlgorithm
	}
}

// getVerificationKey returns the key for token verification
func (s *jwtService) getVerificationKey() any {
	switch s.config.Algorithm {
	case AlgorithmHS256:
		return []byte(s.config.SecretKey)
	case AlgorithmRS256:
		return s.config.PublicKey
	default:
		return nil
	}
}

// validateAlgorithm validates the token's algorithm to prevent security vulnerabilities
func (s *jwtService) validateAlgorithm(token *jwt.Token) error {
	// Reject 'none' algorithm explicitly
	if token.Method.Alg() == "none" {
		return ErrNoneAlgorithm
	}

	// Validate algorithm matches expected algorithm
	expectedAlg := string(s.config.Algorithm)
	actualAlg := token.Method.Alg()

	if actualAlg != expectedAlg {
		return fmt.Errorf("%w: expected %s, got %s", ErrAlgorithmMismatch, expectedAlg, actualAlg)
	}

	// Validate algorithm is one of the supported types
	switch token.Method.(type) {
	case *jwt.SigningMethodHMAC:
		if s.config.Algorithm != AlgorithmHS256 {
			return ErrAlgorithmMismatch
		}
	case *jwt.SigningMethodRSA:
		if s.config.Algorithm != AlgorithmRS256 {
			return ErrAlgorithmMismatch
		}
	default:
		return ErrInvalidAlgorithm
	}

	return nil
}

// mapTokenError maps jwt library errors to custom errors
func (s *jwtService) mapTokenError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, jwt.ErrTokenExpired):
		return ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return ErrInvalidSignature
	case errors.Is(err, jwt.ErrTokenMalformed):
		return ErrInvalidToken
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return ErrInvalidToken
	default:
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
}

// Helper function to get signing key interface for different algorithms
func getSigningKey(algorithm Algorithm, secretKey string, privateKey *rsa.PrivateKey) any {
	switch algorithm {
	case AlgorithmHS256:
		return []byte(secretKey)
	case AlgorithmRS256:
		return privateKey
	default:
		return nil
	}
}
