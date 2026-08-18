package middleware

import (
	"context"
	"testing"

	"base_project/pkg/jwt/claims"
	jwtcontext "base_project/pkg/jwt/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestNewGRPCAuthInterceptor(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	interceptor := NewGRPCAuthInterceptor(service, logger)

	if interceptor == nil {
		t.Fatal("Expected interceptor to be created")
	}

	if interceptor.jwtService == nil {
		t.Error("Expected jwtService to be set")
	}

	if interceptor.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name         string
		token        string
		expectedCode codes.Code
		expectClaims bool
	}{
		{
			name:         "valid token",
			token:        validToken,
			expectedCode: codes.OK,
			expectClaims: true,
		},
		{
			name:         "missing token",
			token:        "",
			expectedCode: codes.Unauthenticated,
			expectClaims: false,
		},
		{
			name:         "invalid token",
			token:        "invalid.token.here",
			expectedCode: codes.Unauthenticated,
			expectClaims: false,
		},
		{
			name:         "bearer prefix token",
			token:        "Bearer " + validToken,
			expectedCode: codes.OK,
			expectClaims: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := func(ctx context.Context, req any) (any, error) {
				// Check if claims are in context
				claims, ok := jwtcontext.GetClaims(ctx)
				if tt.expectClaims && !ok {
					t.Error("Expected claims in context")
				}
				if tt.expectClaims && claims.UserID != "user-123" {
					t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
				}

				return "success", nil
			}

			// Create context with metadata
			ctx := context.Background()
			if tt.token != "" {
				md := metadata.New(map[string]string{
					"authorization": tt.token,
				})
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Execute interceptor
			_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

			// Check error code
			if tt.expectedCode == codes.OK {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Error("Expected gRPC status error")
					} else if st.Code() != tt.expectedCode {
						t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
					}
				}
			}
		})
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name         string
		token        string
		expectedCode codes.Code
		expectClaims bool
	}{
		{
			name:         "valid token",
			token:        validToken,
			expectedCode: codes.OK,
			expectClaims: true,
		},
		{
			name:         "missing token",
			token:        "",
			expectedCode: codes.Unauthenticated,
			expectClaims: false,
		},
		{
			name:         "invalid token",
			token:        "invalid.token.here",
			expectedCode: codes.Unauthenticated,
			expectClaims: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := func(srv any, stream grpc.ServerStream) error {
				// Check if claims are in context
				claims, ok := jwtcontext.GetClaims(stream.Context())
				if tt.expectClaims && !ok {
					t.Error("Expected claims in context")
				}
				if tt.expectClaims && claims.UserID != "user-123" {
					t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
				}

				return nil
			}

			// Create context with metadata
			ctx := context.Background()
			if tt.token != "" {
				md := metadata.New(map[string]string{
					"authorization": tt.token,
				})
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Create mock server stream
			stream := &mockServerStream{ctx: ctx}

			// Create server info
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestStreamMethod",
			}

			// Execute interceptor
			err := interceptor.StreamServerInterceptor()(nil, stream, info, handler)

			// Check error code
			if tt.expectedCode == codes.OK {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Error("Expected gRPC status error")
					} else if st.Code() != tt.expectedCode {
						t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
					}
				}
			}
		})
	}
}

func TestSkipMethods(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	options := GRPCAuthOptions{
		SkipMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/test.Service/PublicMethod",
		},
	}

	interceptor := NewGRPCAuthInterceptorWithOptions(service, logger, options)

	tests := []struct {
		name         string
		method       string
		token        string
		expectedCode codes.Code
	}{
		{
			name:         "skip health check without token",
			method:       "/grpc.health.v1.Health/Check",
			token:        "",
			expectedCode: codes.OK,
		},
		{
			name:         "skip public method without token",
			method:       "/test.Service/PublicMethod",
			token:        "",
			expectedCode: codes.OK,
		},
		{
			name:         "protected method requires token",
			method:       "/test.Service/ProtectedMethod",
			token:        "",
			expectedCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := func(ctx context.Context, req any) (any, error) {
				return "success", nil
			}

			// Create context with metadata
			ctx := context.Background()
			if tt.token != "" {
				md := metadata.New(map[string]string{
					"authorization": tt.token,
				})
				ctx = metadata.NewIncomingContext(ctx, md)
			} else {
				// Add empty metadata
				md := metadata.New(map[string]string{})
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.method,
			}

			// Execute interceptor
			_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

			// Check error code
			if tt.expectedCode == codes.OK {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Error("Expected gRPC status error")
					} else if st.Code() != tt.expectedCode {
						t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
					}
				}
			}
		})
	}
}

func TestTokenExtraction(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name         string
		metadataKey  string
		tokenValue   string
		expectedCode codes.Code
	}{
		{
			name:         "authorization header",
			metadataKey:  "authorization",
			tokenValue:   validToken,
			expectedCode: codes.OK,
		},
		{
			name:         "authorization with Bearer prefix",
			metadataKey:  "authorization",
			tokenValue:   "Bearer " + validToken,
			expectedCode: codes.OK,
		},
		{
			name:         "x-api-token header",
			metadataKey:  "x-api-token",
			tokenValue:   validToken,
			expectedCode: codes.OK,
		},
		{
			name:         "x-auth-token header",
			metadataKey:  "x-auth-token",
			tokenValue:   validToken,
			expectedCode: codes.OK,
		},
		{
			name:         "token header",
			metadataKey:  "token",
			tokenValue:   validToken,
			expectedCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := func(ctx context.Context, req any) (any, error) {
				return "success", nil
			}

			// Create context with metadata
			md := metadata.New(map[string]string{
				tt.metadataKey: tt.tokenValue,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Execute interceptor
			_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

			// Check error code
			if tt.expectedCode == codes.OK {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

func TestCustomMetadataExtractor(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	options := GRPCAuthOptions{
		TokenExtractor: TokenExtractorFromMetadataKey("custom-token"),
	}

	interceptor := NewGRPCAuthInterceptorWithOptions(service, logger, options)

	// Create test handler
	handler := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Create context with custom metadata key
	md := metadata.New(map[string]string{
		"custom-token": validToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Create server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Execute interceptor
	_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

	// Should succeed
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestChainMetadataExtractors(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	options := GRPCAuthOptions{
		TokenExtractor: ChainMetadataExtractors(
			TokenExtractorFromMetadataKey("custom-token"),
			TokenExtractorFromMetadataKey("authorization"),
			TokenExtractorFromMetadataKey("x-api-token"),
		),
	}

	interceptor := NewGRPCAuthInterceptorWithOptions(service, logger, options)

	tests := []struct {
		name       string
		metadata   map[string]string
		shouldPass bool
	}{
		{
			name: "token in custom-token",
			metadata: map[string]string{
				"custom-token": validToken,
			},
			shouldPass: true,
		},
		{
			name: "token in authorization (second extractor)",
			metadata: map[string]string{
				"authorization": validToken,
			},
			shouldPass: true,
		},
		{
			name: "token in x-api-token (third extractor)",
			metadata: map[string]string{
				"x-api-token": validToken,
			},
			shouldPass: true,
		},
		{
			name: "no token in any extractor",
			metadata: map[string]string{
				"other-header": "value",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := func(ctx context.Context, req any) (any, error) {
				return "success", nil
			}

			// Create context with metadata
			md := metadata.New(tt.metadata)
			ctx := metadata.NewIncomingContext(context.Background(), md)

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Execute interceptor
			_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

			if tt.shouldPass && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if !tt.shouldPass && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestGRPCRequireTokenType(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()

	// Generate different token types
	accessToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)
	refreshToken, _ := service.GenerateRefreshToken("user-123", nil)

	tests := []struct {
		name             string
		token            string
		requireTokenType string
		expectedCode     codes.Code
	}{
		{
			name:             "access token required and provided",
			token:            accessToken,
			requireTokenType: "access",
			expectedCode:     codes.OK,
		},
		{
			name:             "refresh token when access required",
			token:            refreshToken,
			requireTokenType: "access",
			expectedCode:     codes.Unauthenticated,
		},
		{
			name:             "refresh token required and provided",
			token:            refreshToken,
			requireTokenType: "refresh",
			expectedCode:     codes.OK,
		},
		{
			name:             "access token when refresh required",
			token:            accessToken,
			requireTokenType: "refresh",
			expectedCode:     codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := GRPCAuthOptions{
				RequireTokenType: tt.requireTokenType,
			}

			interceptor := NewGRPCAuthInterceptorWithOptions(service, logger, options)

			// Create test handler
			handler := func(ctx context.Context, req any) (any, error) {
				return "success", nil
			}

			// Create context with metadata
			md := metadata.New(map[string]string{
				"authorization": tt.token,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Execute interceptor
			_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

			// Check error code
			if tt.expectedCode == codes.OK {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Error("Expected gRPC status error")
					} else if st.Code() != tt.expectedCode {
						t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
					}
				}
			}
		})
	}
}

func TestGRPCRequireRoleInterceptors(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate tokens with different roles
	adminToken, _ := service.GenerateAccessToken("admin-123", "admin@example.com", "admin", nil)
	userToken, _ := service.GenerateAccessToken("user-123", "user@example.com", "user", nil)

	tests := []struct {
		name          string
		token         string
		requiredRoles []string
		expectedCode  codes.Code
	}{
		{
			name:          "admin has admin role",
			token:         adminToken,
			requiredRoles: []string{"admin"},
			expectedCode:  codes.OK,
		},
		{
			name:          "user does not have admin role",
			token:         userToken,
			requiredRoles: []string{"admin"},
			expectedCode:  codes.PermissionDenied,
		},
		{
			name:          "user has user role",
			token:         userToken,
			requiredRoles: []string{"user"},
			expectedCode:  codes.OK,
		},
		{
			name:          "admin matches one of multiple roles",
			token:         adminToken,
			requiredRoles: []string{"admin", "superuser"},
			expectedCode:  codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First authenticate, then check role
			authHandler := func(ctx context.Context, req any) (any, error) {
				// This is called after authentication
				roleInterceptor := interceptor.RequireRoleUnaryInterceptor(tt.requiredRoles...)
				return roleInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}, func(ctx context.Context, req any) (any, error) {
					return "success", nil
				})
			}

			// Create context with metadata
			md := metadata.New(map[string]string{
				"authorization": tt.token,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Execute authentication interceptor
			_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, authHandler)

			// Check error code
			if tt.expectedCode == codes.OK {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Error("Expected gRPC status error")
					} else if st.Code() != tt.expectedCode {
						t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
					}
				}
			}
		})
	}
}

func TestGRPCOptionalAuthInterceptors(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	tests := []struct {
		name         string
		token        string
		expectClaims bool
	}{
		{
			name:         "valid token",
			token:        validToken,
			expectClaims: true,
		},
		{
			name:         "missing token - should still succeed",
			token:        "",
			expectClaims: false,
		},
		{
			name:         "invalid token - should still succeed",
			token:        "invalid.token.here",
			expectClaims: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := func(ctx context.Context, req any) (any, error) {
				// Check if claims are in context
				_, ok := jwtcontext.GetClaims(ctx)
				if tt.expectClaims && !ok {
					t.Error("Expected claims in context")
				}
				if !tt.expectClaims && ok {
					t.Error("Did not expect claims in context")
				}

				return "success", nil
			}

			// Create context with metadata
			ctx := context.Background()
			if tt.token != "" {
				md := metadata.New(map[string]string{
					"authorization": tt.token,
				})
				ctx = metadata.NewIncomingContext(ctx, md)
			} else {
				// Add empty metadata
				md := metadata.New(map[string]string{})
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Create server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Execute optional auth interceptor
			_, err := interceptor.OptionalAuthUnaryInterceptor()(ctx, nil, info, handler)

			// Should never return error
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestGetClaimsFromGRPCContext(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "admin", nil)

	// Create test handler that uses GetClaimsFromGRPCContext
	handler := func(ctx context.Context, req any) (any, error) {
		claims, ok := GetClaimsFromGRPCContext(ctx)
		if !ok {
			t.Error("Expected claims in context")
			return nil, status.Error(codes.Internal, "no claims")
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

		return "success", nil
	}

	// Create context with metadata
	md := metadata.New(map[string]string{
		"authorization": validToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Create server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Execute interceptor
	_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestGetUserIDFromGRPCContext(t *testing.T) {
	service := createTestJWTService()
	logger := createTestLogger()
	interceptor := NewGRPCAuthInterceptor(service, logger)

	// Generate a valid token
	validToken, _ := service.GenerateAccessToken("user-123", "test@example.com", "user", nil)

	// Create test handler that uses GetUserIDFromGRPCContext
	handler := func(ctx context.Context, req any) (any, error) {
		userID, ok := GetUserIDFromGRPCContext(ctx)
		if !ok {
			t.Error("Expected user ID in context")
			return nil, status.Error(codes.Internal, "no user ID")
		}

		if userID != "user-123" {
			t.Errorf("Expected user_id 'user-123', got '%s'", userID)
		}

		return "success", nil
	}

	// Create context with metadata
	md := metadata.New(map[string]string{
		"authorization": validToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Create server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Execute interceptor
	_, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestMustGetClaimsFromGRPCContext(t *testing.T) {
	// Create a context with claims
	ctx := context.Background()
	testClaims := &claims.CustomClaims{
		UserID: "user-123",
		Email:  "test@example.com",
		Role:   "admin",
	}

	ctx = jwtcontext.WithClaims(ctx, testClaims)

	// Should not panic
	claims := MustGetClaimsFromGRPCContext(ctx)
	if claims.UserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
	}
}

func TestMustGetClaimsFromGRPCContextPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic but didn't get one")
		}
	}()

	// Create a context without claims
	ctx := context.Background()

	// Should panic
	MustGetClaimsFromGRPCContext(ctx)
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg any) error {
	return nil
}

func (m *mockServerStream) RecvMsg(msg any) error {
	return nil
}
