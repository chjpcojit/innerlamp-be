package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"base_project/pkg/jwt"
	"base_project/pkg/jwt/claims"
	jwtcontext "base_project/pkg/jwt/context"
	"base_project/pkg/logging"
	"base_project/pkg/logging/formatter"
)

func createTestJWTService() jwt.Service {
	config := &jwt.Config{
		Algorithm:              jwt.AlgorithmHS256,
		SecretKey:              "test-secret-key-must-be-32-chars!",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	service, _ := jwt.New(config)
	return service
}

func createTestLogger() logging.Logger {
	return logging.New(logging.Config{
		Level:     logging.DebugLevel,
		Formatter: formatter.NewJSON(),
	})
}

func TestNewHTTPAuthMiddleware(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	middleware := NewHTTPAuthMiddleware(service, logger)

	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	if middleware.jwtService == nil {
		t.Error("Expected jwtService to be set")
	}

	if middleware.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestAuthenticate(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	middleware := NewHTTPAuthMiddleware(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectClaims   bool
	}{
		{
			name:           "valid token",
			token:          "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectClaims:   true,
		},
		{
			name:           "missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectClaims:   false,
		},
		{
			name:           "invalid token format",
			token:          "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
			expectClaims:   false,
		},
		{
			name:           "invalid token",
			token:          "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectClaims:   false,
		},
		{
			name:           "missing bearer prefix",
			token:          validToken,
			expectedStatus: http.StatusUnauthorized,
			expectClaims:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if claims are in context
				claims, ok := jwtcontext.GetClaims(r.Context())
				if tt.expectClaims && !ok {
					t.Error("Expected claims in context")
				}
				if tt.expectClaims && claims.UserID != "user-123" {
					t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
				}

				w.WriteHeader(http.StatusOK)
			}))

			// Create request
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			// Record response
			rr := httptest.NewRecorder()

			// Execute handler
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestOptionalAuthenticate(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	middleware := NewHTTPAuthMiddleware(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectClaims   bool
	}{
		{
			name:           "valid token",
			token:          "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectClaims:   true,
		},
		{
			name:           "missing token - should still succeed",
			token:          "",
			expectedStatus: http.StatusOK,
			expectClaims:   false,
		},
		{
			name:           "invalid token - should still succeed",
			token:          "Bearer invalid.token.here",
			expectedStatus: http.StatusOK,
			expectClaims:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := middleware.OptionalAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if claims are in context
				claims, ok := jwtcontext.GetClaims(r.Context())
				if tt.expectClaims && !ok {
					t.Error("Expected claims in context")
				}
				if !tt.expectClaims && ok {
					t.Error("Did not expect claims in context")
				}
				if tt.expectClaims && claims != nil && claims.UserID != "user-123" {
					t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
				}

				w.WriteHeader(http.StatusOK)
			}))

			// Create request
			req := httptest.NewRequest("GET", "/optional", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			// Record response
			rr := httptest.NewRecorder()

			// Execute handler
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	middleware := NewHTTPAuthMiddleware(service, logger)

	// Generate tokens with different roles
	adminToken, _ := service.GenerateAccessToken("admin-123", "admin@example.com", "admin", nil)
	userToken, _ := service.GenerateAccessToken("user-123", "user@example.com", "user", nil)

	tests := []struct {
		name           string
		token          string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "admin has admin role",
			token:          adminToken,
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user does not have admin role",
			token:          userToken,
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user has user role",
			token:          userToken,
			requiredRoles:  []string{"user"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "admin matches one of multiple roles",
			token:          adminToken,
			requiredRoles:  []string{"admin", "superuser"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user does not match any of multiple roles",
			token:          userToken,
			requiredRoles:  []string{"admin", "superuser"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler with role requirement
			handler := middleware.Authenticate(
				middleware.RequireRole(tt.requiredRoles...)(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					}),
				),
			)

			// Create request
			req := httptest.NewRequest("GET", "/admin", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)

			// Record response
			rr := httptest.NewRecorder()

			// Execute handler
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestSkipPaths(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	options := HTTPAuthOptions{
		SkipPaths: []string{"/health", "/public"},
	}

	middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

	tests := []struct {
		name           string
		path           string
		token          string
		expectedStatus int
	}{
		{
			name:           "skip /health without token",
			path:           "/health",
			token:          "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "skip /public without token",
			path:           "/public",
			token:          "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "skip /public/info without token",
			path:           "/public/info",
			token:          "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "protected path requires token",
			path:           "/protected",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create request
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Record response
			rr := httptest.NewRecorder()

			// Execute handler
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestCustomTokenExtractor(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	t.Run("extract from custom header", func(t *testing.T) {
		options := HTTPAuthOptions{
			TokenExtractor: TokenExtractorFromHeader("X-API-Token"),
		}

		middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

		// Create test handler
		handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Create request with custom header
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("X-API-Token", validToken)

		// Record response
		rr := httptest.NewRecorder()

		// Execute handler
		handler.ServeHTTP(rr, req)

		// Should succeed
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("extract from cookie", func(t *testing.T) {
		options := HTTPAuthOptions{
			TokenExtractor: TokenExtractorFromCookie("auth_token"),
		}

		middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

		// Create test handler
		handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Create request with cookie
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: validToken,
		})

		// Record response
		rr := httptest.NewRecorder()

		// Execute handler
		handler.ServeHTTP(rr, req)

		// Should succeed
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("extract from query parameter", func(t *testing.T) {
		options := HTTPAuthOptions{
			TokenExtractor: TokenExtractorFromQuery("token"),
		}

		middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

		// Create test handler
		handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Create request with query parameter
		req := httptest.NewRequest("GET", "/protected?token="+validToken, nil)

		// Record response
		rr := httptest.NewRecorder()

		// Execute handler
		handler.ServeHTTP(rr, req)

		// Should succeed
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("chain token extractors", func(t *testing.T) {
		options := HTTPAuthOptions{
			TokenExtractor: ChainTokenExtractors(
				TokenExtractorFromHeader("X-API-Token"),
				TokenExtractorFromCookie("auth_token"),
				func(r *http.Request) string {
					// Extract from Authorization header as fallback
					auth := r.Header.Get("Authorization")
					if len(auth) > 7 && auth[:7] == "Bearer " {
						return auth[7:]
					}
					return ""
				},
			),
		}

		middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

		// Create test handler
		handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Test with Authorization header (should use third extractor)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestRequireTokenType(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	// Generate different token types
	accessToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)
	refreshToken, _ := service.GenerateRefreshToken("user-123", nil)

	tests := []struct {
		name             string
		token            string
		requireTokenType string
		expectedStatus   int
	}{
		{
			name:             "access token required and provided",
			token:            accessToken,
			requireTokenType: "access",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "refresh token when access required",
			token:            refreshToken,
			requireTokenType: "access",
			expectedStatus:   http.StatusUnauthorized,
		},
		{
			name:             "refresh token required and provided",
			token:            refreshToken,
			requireTokenType: "refresh",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "access token when refresh required",
			token:            accessToken,
			requireTokenType: "refresh",
			expectedStatus:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := HTTPAuthOptions{
				RequireTokenType: tt.requireTokenType,
			}

			middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

			// Create test handler
			handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create request
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)

			// Record response
			rr := httptest.NewRecorder()

			// Execute handler
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	middleware := NewHTTPAuthMiddleware(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "admin", nil)

	// Create test handler that uses GetClaimsFromContext
	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaimsFromContext(r.Context())
		if !ok {
			t.Error("Expected claims in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if claims.UserID != "user-123" {
			t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
		}

		if claims.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", claims.Email)
		}

		if claims.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", claims.Role)
		}

		w.WriteHeader(http.StatusOK)
	}))

	// Create request
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	// Record response
	rr := httptest.NewRecorder()

	// Execute handler
	handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	middleware := NewHTTPAuthMiddleware(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	// Create test handler that uses GetUserIDFromContext
	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("Expected user ID in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if userID != "user-123" {
			t.Errorf("Expected user_id 'user-123', got '%s'", userID)
		}

		w.WriteHeader(http.StatusOK)
	}))

	// Create request
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	// Record response
	rr := httptest.NewRecorder()

	// Execute handler
	handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestCustomErrorHandler(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	customErrorCalled := false
	options := HTTPAuthOptions{
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			customErrorCalled = true
			w.WriteHeader(http.StatusTeapot) // Use a unique status code
			w.Write([]byte("custom error"))
		},
	}

	middleware := NewHTTPAuthMiddlewareWithOptions(service, logger, options)

	// Create test handler
	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request without token
	req := httptest.NewRequest("GET", "/protected", nil)

	// Record response
	rr := httptest.NewRecorder()

	// Execute handler
	handler.ServeHTTP(rr, req)

	// Check that custom error handler was called
	if !customErrorCalled {
		t.Error("Expected custom error handler to be called")
	}

	// Check custom status code
	if rr.Code != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, rr.Code)
	}

	// Check custom body
	if rr.Body.String() != "custom error" {
		t.Errorf("Expected body 'custom error', got '%s'", rr.Body.String())
	}
}

func TestMustGetClaimsFromContext(t *testing.T) {
	// Create a context with claims
	ctx := context.Background()
	testClaims := &claims.CustomClaims{
		UserID: "user-123",
		Email:  "test@example.com",
		Role:   "admin",
	}

	ctx = jwtcontext.WithClaims(ctx, testClaims)

	// Should not panic
	claims := MustGetClaimsFromContext(ctx)
	if claims.UserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
	}
}

func TestMustGetClaimsFromContextPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic but didn't get one")
		}
	}()

	// Create a context without claims
	ctx := context.Background()

	// Should panic
	MustGetClaimsFromContext(ctx)
}
