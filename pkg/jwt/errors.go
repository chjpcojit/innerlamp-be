package jwt

import "errors"

// Common JWT errors
var (
	// ErrInvalidToken is returned when the token is invalid or malformed
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired is returned when the token has expired
	ErrTokenExpired = errors.New("token has expired")

	// ErrInvalidSignature is returned when the token signature is invalid
	ErrInvalidSignature = errors.New("invalid token signature")

	// ErrInvalidAlgorithm is returned when the token uses an unsupported or invalid algorithm
	ErrInvalidAlgorithm = errors.New("invalid or unsupported algorithm")

	// ErrAlgorithmMismatch is returned when the token's algorithm doesn't match the expected algorithm
	ErrAlgorithmMismatch = errors.New("token algorithm does not match expected algorithm")

	// ErrNoneAlgorithm is returned when the token uses the 'none' algorithm
	ErrNoneAlgorithm = errors.New("'none' algorithm is not allowed")

	// ErrInvalidClaims is returned when the token claims are invalid
	ErrInvalidClaims = errors.New("invalid token claims")

	// ErrMissingToken is returned when no token is provided
	ErrMissingToken = errors.New("missing token")

	// ErrInvalidTokenType is returned when the token type is not as expected
	ErrInvalidTokenType = errors.New("invalid token type")

	// ErrInvalidKey is returned when the signing key is invalid
	ErrInvalidKey = errors.New("invalid signing key")

	// ErrUnsupportedAlgorithm is returned when trying to use an unsupported algorithm
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
)
