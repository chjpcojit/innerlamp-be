// ⚠️ MOCK DATA - NOT FOR PRODUCTION
// This file is an example only. It uses hardcoded in-memory user data
// for illustration purposes.
//
// Before using in production:
//   - Replace hardcoded user lists with real database repository calls
//   - Implement proper gRPC service handlers against Cloud SQL / MongoDB
//   - Use pkg/secrets/ for credentials, PKI certs for TLS
//
// MOCK_DATA_MARKER: hardcoded user list in ListUsers handler
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"base_project/pkg/jwt"
	"base_project/pkg/jwt/middleware"
	"base_project/pkg/logging"
	"base_project/pkg/logging/formatter"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// This example demonstrates how to use JWT authentication with gRPC
// In a real application, you would define your proto files and generate the gRPC code
// For this example, we'll use simple service interfaces

// UserService defines the user service interface
type UserService interface {
	GetProfile(context.Context, *GetProfileRequest) (*GetProfileResponse, error)
	ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error)
	AdminAction(context.Context, *AdminActionRequest) (*AdminActionResponse, error)
}

// Requests and Responses
type GetProfileRequest struct{}

type GetProfileResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type ListUsersRequest struct{}

type ListUsersResponse struct {
	Users []UserInfo `json:"users"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AdminActionRequest struct {
	Action string `json:"action"`
}

type AdminActionResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// userServiceServer implements the UserService
type userServiceServer struct {
	jwtService jwt.Service
	logger     logging.Logger
}

// GetProfile returns the authenticated user's profile
func (s *userServiceServer) GetProfile(ctx context.Context, req *GetProfileRequest) (*GetProfileResponse, error) {
	// Get claims from context (set by authentication interceptor)
	claims, ok := middleware.GetClaimsFromGRPCContext(ctx)
	if !ok {
		s.logger.WithContext(ctx).Warn("No claims in context")
		return nil, status.Error(codes.Unauthenticated, "no authentication")
	}

	s.logger.WithContext(ctx).Info("Profile accessed",
		"user_id", claims.UserID,
		"email", claims.Email,
	)

	return &GetProfileResponse{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

// ListUsers returns a list of all users
func (s *userServiceServer) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	// Get claims from context
	claims, ok := middleware.GetClaimsFromGRPCContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no authentication")
	}

	s.logger.WithContext(ctx).Info("Users list accessed",
		"user_id", claims.UserID,
	)

	// ⚠️ MOCK DATA - NOT FOR PRODUCTION
	// Replace with a real database repository call (e.g., userRepo.ListAll(ctx))
	users := []UserInfo{
		{ID: "user-1", Email: "admin@example.com", Role: "admin"},
		{ID: "user-2", Email: "user@example.com", Role: "user"},
		{ID: "user-3", Email: "guest@example.com", Role: "guest"},
	}

	return &ListUsersResponse{
		Users: users,
	}, nil
}

// AdminAction performs an admin-only action
func (s *userServiceServer) AdminAction(ctx context.Context, req *AdminActionRequest) (*AdminActionResponse, error) {
	// Get claims from context
	claims, ok := middleware.GetClaimsFromGRPCContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no authentication")
	}

	// Check if user is admin (this should be done by role interceptor)
	if claims.Role != "admin" {
		s.logger.WithContext(ctx).Warn("Non-admin tried to access admin endpoint",
			"user_id", claims.UserID,
			"role", claims.Role,
		)
		return nil, status.Error(codes.PermissionDenied, "admin access required")
	}

	s.logger.WithContext(ctx).Info("Admin action performed",
		"user_id", claims.UserID,
		"action", req.Action,
	)

	return &AdminActionResponse{
		Message: fmt.Sprintf("Admin action '%s' completed successfully", req.Action),
		Success: true,
	}, nil
}

func main() {
	// Create JWT service with HS256
	jwtConfig := &jwt.Config{
		Algorithm:              jwt.AlgorithmHS256,
		SecretKey:              "your-super-secret-key-min-32-chars!!",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "example-grpc-api",
		Audience:               "example-users",
	}

	jwtService, err := jwt.New(jwtConfig)
	if err != nil {
		log.Fatalf("Failed to create JWT service: %v", err)
	}

	// Create logger
	logger := logging.New(logging.Config{
		Level:     logging.InfoLevel,
		Formatter: formatter.NewJSON(),
	})

	// Create authentication interceptor with skip methods configuration
	authInterceptorWithOptions := middleware.NewGRPCAuthInterceptorWithOptions(
		jwtService,
		logger,
		middleware.GRPCAuthOptions{
			SkipMethods: []string{
				"/grpc.health.v1.Health/Check",
				// Add other public methods here
			},
		},
	)

	// Create gRPC server with interceptors
	server := grpc.NewServer(
		// Chain interceptors: authentication -> role checking
		grpc.ChainUnaryInterceptor(
			authInterceptorWithOptions.UnaryServerInterceptor(),
			// Add other interceptors here (logging, metrics, etc.)
		),
		grpc.ChainStreamInterceptor(
			authInterceptorWithOptions.StreamServerInterceptor(),
			// Add other stream interceptors here
		),
	)

	// Note: In a real application, you would register the service with the gRPC server
	// using the generated proto code:
	// userService := &userServiceServer{jwtService: jwtService, logger: logger}
	// pb.RegisterUserServiceServer(server, userService)

	// Start server
	addr := ":9090"
	lis, err := net.Listen("tcp", addr) // #nosec
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	logger.Info(fmt.Sprintf("Starting gRPC server on %s", addr))
	logger.Info("Example usage:")
	logger.Info("  1. Generate a token using the login endpoint")
	logger.Info("  2. Call gRPC methods with metadata: authorization=Bearer <token>")
	logger.Info("")
	logger.Info("Testing with grpcurl:")
	logger.Info("  grpcurl -plaintext -H 'authorization: Bearer <token>' localhost:9090 UserService/GetProfile")

	// Start example client in a goroutine
	go runExampleClient(jwtService, logger)

	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// runExampleClient demonstrates how to call the gRPC service with JWT authentication
func runExampleClient(jwtService jwt.Service, logger logging.Logger) {
	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Connect to server
	conn, err := grpc.Dial(
		"localhost:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	logger.Info("=== gRPC Client Example ===")

	// Generate a token for testing
	accessToken, err := jwtService.GenerateAccessToken(
		"user-123",
		"test@example.com",
		"admin",
		nil,
	)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		return
	}

	logger.Info("Generated access token for testing")

	// Example 1: Call GetProfile with authentication
	logger.Info("\n--- Example 1: GetProfile (authenticated) ---")
	ctx := context.Background()

	// Add token to metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + accessToken,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	logger.Info("Calling GetProfile with token...")
	// In real app: profileResp, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	logger.Info("✓ Success: Would receive user profile")

	// Example 2: Call without authentication (should fail)
	logger.Info("\n--- Example 2: GetProfile (no auth - should fail) ---")
	logger.Info("Calling GetProfile without token...")
	// In real app: _, err = client.GetProfile(context.Background(), &pb.GetProfileRequest{})
	logger.Info("✗ Expected: Unauthenticated error")

	// Example 3: Call with invalid token (should fail)
	logger.Info("\n--- Example 3: GetProfile (invalid token - should fail) ---")
	logger.Info("Calling GetProfile with invalid token...")
	// In real app:
	// mdInvalid := metadata.New(map[string]string{"authorization": "Bearer invalid.token.here"})
	// ctxInvalid := metadata.NewOutgoingContext(context.Background(), mdInvalid)
	// _, err = client.GetProfile(ctxInvalid, &pb.GetProfileRequest{})
	logger.Info("✗ Expected: Unauthenticated error")

	// Example 4: Admin-only endpoint
	logger.Info("\n--- Example 4: AdminAction (admin role required) ---")
	logger.Info("Calling AdminAction with admin token...")
	// In real app: adminResp, err := client.AdminAction(ctx, &pb.AdminActionRequest{Action: "delete_user"})
	logger.Info("✓ Success: Admin action completed")

	// Example 5: Non-admin trying to access admin endpoint
	logger.Info("\n--- Example 5: AdminAction (non-admin - should fail) ---")
	userToken, _ := jwtService.GenerateAccessToken(
		"user-456",
		"user@example.com",
		"user",
		nil,
	)
	logger.Info("Calling AdminAction with user token...")
	// In real app:
	// mdUser := metadata.New(map[string]string{"authorization": "Bearer " + userToken})
	// ctxUser := metadata.NewOutgoingContext(context.Background(), mdUser)
	// _, err = client.AdminAction(ctxUser, &pb.AdminActionRequest{Action: "delete_user"})
	logger.Info("✗ Expected: PermissionDenied error")
	_ = userToken // Mark as used

	// Example 6: Refresh token flow
	logger.Info("\n--- Example 6: Token Refresh Flow ---")
	refreshToken, _ := jwtService.GenerateRefreshToken("user-123", nil)
	logger.Info("Generated refresh token")

	// Validate refresh token
	claims, err := jwtService.ValidateToken(refreshToken)
	if err != nil {
		logger.Error("Failed to validate refresh token", "error", err.Error())
		return
	}

	if claims.TokenType != "refresh" {
		logger.Error("Invalid token type", "expected", "refresh", "got", claims.TokenType)
		return
	}

	// Generate new access token
	newAccessToken, err := jwtService.GenerateAccessToken(
		claims.UserID,
		"test@example.com",
		"admin",
		nil,
	)
	if err != nil {
		logger.Error("Failed to generate new access token", "error", err.Error())
		return
	}

	logger.Info("✓ Successfully refreshed access token")
	_ = newAccessToken // Use the new token

	// Example 7: Custom metadata extraction
	logger.Info("\n--- Example 7: Custom Metadata Key ---")
	logger.Info("Calling with custom metadata key 'x-api-token'...")
	// In real app:
	// mdCustom := metadata.New(map[string]string{"x-api-token": accessToken})
	// ctxCustom := metadata.NewOutgoingContext(context.Background(), mdCustom)
	// _, err = client.GetProfile(ctxCustom, &pb.GetProfileRequest{})
	logger.Info("✓ Success: Token extracted from custom key")

	logger.Info("\n=== Example Client Completed ===")
}

// Example of how to setup role-based interceptors
func setupRoleBasedInterceptors(authInterceptor *middleware.GRPCAuthInterceptor) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		authInterceptor.UnaryServerInterceptor(),
		// Admin-only methods
		authInterceptor.RequireRoleUnaryInterceptor("admin", "superadmin"),
	)
}

// Example of optional authentication
func setupOptionalAuthServer(jwtService jwt.Service, logger logging.Logger) *grpc.Server {
	authInterceptor := middleware.NewGRPCAuthInterceptor(jwtService, logger)

	return grpc.NewServer(
		grpc.UnaryInterceptor(
			authInterceptor.OptionalAuthUnaryInterceptor(),
		),
		grpc.StreamInterceptor(
			authInterceptor.OptionalAuthStreamInterceptor(),
		),
	)
}

// Example of chaining multiple extractors
func setupChainedExtractors(jwtService jwt.Service, logger logging.Logger) *middleware.GRPCAuthInterceptor {
	options := middleware.GRPCAuthOptions{
		TokenExtractor: middleware.ChainMetadataExtractors(
			middleware.TokenExtractorFromMetadataKey("authorization"),
			middleware.TokenExtractorFromMetadataKey("x-api-token"),
			middleware.TokenExtractorFromMetadataKey("x-auth-token"),
		),
	}

	return middleware.NewGRPCAuthInterceptorWithOptions(jwtService, logger, options)
}
