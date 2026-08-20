package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt/claims"

	"github.com/golang-jwt/jwt/v5"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid HS256 config",
			config: &Config{
				Algorithm:              AlgorithmHS256,
				SecretKey:              "this-is-a-very-secure-secret-key-32chars",
				AccessTokenExpiration:  15 * time.Minute,
				RefreshTokenExpiration: 7 * 24 * time.Hour,
				Issuer:                 "test",
				Audience:               "test-users",
			},
			wantErr: false,
		},
		{
			name: "invalid HS256 config - short secret",
			config: &Config{
				Algorithm:              AlgorithmHS256,
				SecretKey:              "short",
				AccessTokenExpiration:  15 * time.Minute,
				RefreshTokenExpiration: 7 * 24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid config - unsupported algorithm",
			config: &Config{
				Algorithm:              "ES256",
				SecretKey:              "this-is-a-very-secure-secret-key-32chars",
				AccessTokenExpiration:  15 * time.Minute,
				RefreshTokenExpiration: 7 * 24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative expiration",
			config: &Config{
				Algorithm:              AlgorithmHS256,
				SecretKey:              "this-is-a-very-secure-secret-key-32chars",
				AccessTokenExpiration:  -1 * time.Minute,
				RefreshTokenExpiration: 7 * 24 * time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateAccessToken_HS256(t *testing.T) {
	config := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	tests := []struct {
		name    string
		userID  string
		email   string
		role    string
		extra   map[string]any
		wantErr bool
	}{
		{
			name:    "valid access token",
			userID:  "user-123",
			email:   "test@example.com",
			role:    "user",
			extra:   map[string]any{"org_id": "org-456"},
			wantErr: false,
		},
		{
			name:    "valid access token without extra",
			userID:  "user-456",
			email:   "test2@example.com",
			role:    "admin",
			extra:   nil,
			wantErr: false,
		},
		{
			name:    "invalid - empty userID",
			userID:  "",
			email:   "test@example.com",
			role:    "user",
			extra:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateAccessToken(tt.userID, tt.email, tt.role, tt.extra)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token == "" {
				t.Error("GenerateAccessToken() returned empty token")
			}

			// Validate the generated token
			if !tt.wantErr {
				parsedClaims, err := service.ValidateToken(token)
				if err != nil {
					t.Errorf("Failed to validate generated token: %v", err)
					return
				}

				if parsedClaims.UserID != tt.userID {
					t.Errorf("UserID = %v, want %v", parsedClaims.UserID, tt.userID)
				}
				if parsedClaims.Email != tt.email {
					t.Errorf("Email = %v, want %v", parsedClaims.Email, tt.email)
				}
				if parsedClaims.Role != tt.role {
					t.Errorf("Role = %v, want %v", parsedClaims.Role, tt.role)
				}
				if parsedClaims.TokenType != "access" {
					t.Errorf("TokenType = %v, want access", parsedClaims.TokenType)
				}
			}
		})
	}
}

func TestGenerateRefreshToken_HS256(t *testing.T) {
	config := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	tests := []struct {
		name    string
		userID  string
		extra   map[string]any
		wantErr bool
	}{
		{
			name:    "valid refresh token",
			userID:  "user-123",
			extra:   map[string]any{"device_id": "device-789"},
			wantErr: false,
		},
		{
			name:    "invalid - empty userID",
			userID:  "",
			extra:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateRefreshToken(tt.userID, tt.extra)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token == "" {
				t.Error("GenerateRefreshToken() returned empty token")
			}

			// Validate the generated token
			if !tt.wantErr {
				parsedClaims, err := service.ValidateToken(token)
				if err != nil {
					t.Errorf("Failed to validate generated token: %v", err)
					return
				}

				if parsedClaims.UserID != tt.userID {
					t.Errorf("UserID = %v, want %v", parsedClaims.UserID, tt.userID)
				}
				if parsedClaims.TokenType != "refresh" {
					t.Errorf("TokenType = %v, want refresh", parsedClaims.TokenType)
				}
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	config := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	// Generate a valid token
	validToken, err := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)
	if err != nil {
		t.Fatalf("Failed to generate valid token: %v", err)
	}

	// Generate an expired token by creating it with short expiration and waiting
	expiredConfig := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  1 * time.Millisecond, // Very short expiration
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}
	expiredService, _ := New(expiredConfig)
	expiredToken, _ := expiredService.GenerateAccessToken("user-123", "test@example.com", "user", nil)
	time.Sleep(10 * time.Millisecond) // Wait for token to expire

	// Generate a token with wrong signature
	wrongKeyConfig := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "different-secret-key-32-characters!",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}
	wrongKeyService, _ := New(wrongKeyConfig)
	wrongSignatureToken, _ := wrongKeyService.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		errType   error
		checkFunc func(*claims.CustomClaims) error
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
			checkFunc: func(c *claims.CustomClaims) error {
				if c.UserID != "user-123" {
					t.Errorf("UserID = %v, want user-123", c.UserID)
				}
				return nil
			},
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
			errType: ErrMissingToken,
		},
		{
			name:    "expired token",
			token:   expiredToken,
			wantErr: true,
			errType: ErrTokenExpired,
		},
		{
			name:    "invalid signature",
			token:   wrongSignatureToken,
			wantErr: true,
			errType: ErrInvalidSignature,
		},
		{
			name:    "malformed token",
			token:   "not.a.valid.token",
			wantErr: true,
			errType: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedClaims, err := service.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if err != tt.errType && err.Error() != tt.errType.Error() {
					// Check if it's wrapped
					if !isErrorType(err, tt.errType) {
						t.Errorf("ValidateToken() error = %v, want %v", err, tt.errType)
					}
				}
			}

			if !tt.wantErr && tt.checkFunc != nil {
				if err := tt.checkFunc(parsedClaims); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestRefreshAccessToken(t *testing.T) {
	config := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	// Generate a valid refresh token
	refreshToken, err := service.GenerateRefreshToken("user-123", map[string]any{"device_id": "device-789"})
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	// Generate an access token (wrong type)
	accessToken, err := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	tests := []struct {
		name         string
		refreshToken string
		email        string
		role         string
		wantErr      bool
		errType      error
	}{
		{
			name:         "valid refresh token",
			refreshToken: refreshToken,
			email:        "test@example.com",
			role:         "user",
			wantErr:      false,
		},
		{
			name:         "invalid - using access token",
			refreshToken: accessToken,
			email:        "test@example.com",
			role:         "user",
			wantErr:      true,
			errType:      ErrInvalidTokenType,
		},
		{
			name:         "invalid - malformed token",
			refreshToken: "invalid.token.here",
			email:        "test@example.com",
			role:         "user",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newAccessToken, err := service.RefreshAccessToken(tt.refreshToken, tt.email, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if newAccessToken == "" {
					t.Error("RefreshAccessToken() returned empty token")
				}

				// Validate the new access token
				claims, err := service.ValidateToken(newAccessToken)
				if err != nil {
					t.Errorf("Failed to validate new access token: %v", err)
					return
				}

				if claims.TokenType != "access" {
					t.Errorf("TokenType = %v, want access", claims.TokenType)
				}
				if claims.UserID != "user-123" {
					t.Errorf("UserID = %v, want user-123", claims.UserID)
				}
			}
		})
	}
}

func TestAlgorithmValidation(t *testing.T) {
	config := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	t.Run("reject none algorithm", func(t *testing.T) {
		// Create a token with 'none' algorithm manually
		claims := &claims.CustomClaims{
			UserID:    "user-123",
			TokenType: claims.TokenTypeAccess,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, _ := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

		_, err := service.ValidateToken(tokenString)
		if err == nil {
			t.Error("ValidateToken() should reject 'none' algorithm")
		}
		// Check for wrapped error
		if !isErrorType(err, ErrNoneAlgorithm) {
			t.Errorf("Expected ErrNoneAlgorithm, got %v", err)
		}
	})

	t.Run("reject algorithm mismatch", func(t *testing.T) {
		// Generate token with HS256
		token, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

		// Try to validate with RS256 service
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		rs256Config := &Config{
			Algorithm:              AlgorithmRS256,
			PrivateKey:             privateKey,
			PublicKey:              &privateKey.PublicKey,
			AccessTokenExpiration:  15 * time.Minute,
			RefreshTokenExpiration: 7 * 24 * time.Hour,
			Issuer:                 "test",
			Audience:               "test-users",
		}
		rs256Service, _ := New(rs256Config)

		_, err := rs256Service.ValidateToken(token)
		if err == nil {
			t.Error("ValidateToken() should reject algorithm mismatch")
		}
	})
}

func TestRS256Algorithm(t *testing.T) {
	// Generate RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	config := &Config{
		Algorithm:              AlgorithmRS256,
		PrivateKey:             privateKey,
		PublicKey:              &privateKey.PublicKey,
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service with RS256: %v", err)
	}

	t.Run("generate and validate RS256 token", func(t *testing.T) {
		token, err := service.GenerateAccessToken("user-123", "test@example.com", "admin", nil)
		if err != nil {
			t.Fatalf("Failed to generate RS256 token: %v", err)
		}

		claims, err := service.ValidateToken(token)
		if err != nil {
			t.Fatalf("Failed to validate RS256 token: %v", err)
		}

		if claims.UserID != "user-123" {
			t.Errorf("UserID = %v, want user-123", claims.UserID)
		}
	})

	t.Run("reject token signed with different key", func(t *testing.T) {
		// Generate a different key pair
		differentKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		differentConfig := &Config{
			Algorithm:              AlgorithmRS256,
			PrivateKey:             differentKey,
			PublicKey:              &differentKey.PublicKey,
			AccessTokenExpiration:  15 * time.Minute,
			RefreshTokenExpiration: 7 * 24 * time.Hour,
			Issuer:                 "test",
			Audience:               "test-users",
		}
		differentService, _ := New(differentConfig)

		// Generate token with different key
		token, _ := differentService.GenerateAccessToken("user-123", "test@example.com", "admin", nil)

		// Try to validate with original service
		_, err := service.ValidateToken(token)
		if err == nil {
			t.Error("ValidateToken() should reject token signed with different key")
		}
	})
}

func TestParseToken(t *testing.T) {
	config := &Config{
		Algorithm:              AlgorithmHS256,
		SecretKey:              "this-is-a-very-secure-secret-key-32chars",
		AccessTokenExpiration:  1 * time.Millisecond, // Very short expiration
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	// Generate a token and wait for it to expire
	expiredToken, err := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Wait for token to expire

	t.Run("parse expired token without validation", func(t *testing.T) {
		claims, err := service.ParseToken(expiredToken)
		if err != nil {
			t.Errorf("ParseToken() should parse expired token: %v", err)
			return
		}

		if claims.UserID != "user-123" {
			t.Errorf("UserID = %v, want user-123", claims.UserID)
		}
	})

	t.Run("parse empty token", func(t *testing.T) {
		_, err := service.ParseToken("")
		if err != ErrMissingToken {
			t.Errorf("ParseToken() error = %v, want %v", err, ErrMissingToken)
		}
	})
}

// Helper function to check if error is of specific type
func isErrorType(err, target error) bool {
	if err == target {
		return true
	}
	// Simple string comparison for wrapped errors
	return err != nil && target != nil &&
		(err.Error() == target.Error() ||
			len(err.Error()) > len(target.Error()) &&
				err.Error()[len(err.Error())-len(target.Error()):] == target.Error())
}
