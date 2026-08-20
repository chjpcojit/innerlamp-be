package middleware

import (
	"context"
	"slices"
	"strings"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/logging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCAuthInterceptor provides gRPC authentication interceptors
type GRPCAuthInterceptor struct {
	jwtService jwt.Service
	logger     logging.Logger
	options    GRPCAuthOptions
}

// GRPCAuthOptions configures the gRPC authentication interceptor
type GRPCAuthOptions struct {
	// TokenExtractor is a custom function to extract token from metadata
	// If nil, uses default extraction from authorization metadata
	TokenExtractor func(metadata.MD) string

	// SkipMethods are gRPC methods that should skip authentication
	// Format: "/package.Service/Method"
	SkipMethods []string

	// RequireTokenType specifies required token type (access or refresh)
	// If empty, any token type is accepted
	RequireTokenType string
}

// NewGRPCAuthInterceptor creates a new gRPC authentication interceptor
func NewGRPCAuthInterceptor(jwtService jwt.Service, logger logging.Logger) *GRPCAuthInterceptor {
	return &GRPCAuthInterceptor{
		jwtService: jwtService,
		logger:     logger,
		options:    GRPCAuthOptions{},
	}
}

// NewGRPCAuthInterceptorWithOptions creates a new gRPC authentication interceptor with custom options
func NewGRPCAuthInterceptorWithOptions(jwtService jwt.Service, logger logging.Logger, options GRPCAuthOptions) *GRPCAuthInterceptor {
	return &GRPCAuthInterceptor{
		jwtService: jwtService,
		logger:     logger,
		options:    options,
	}
}

// UnaryServerInterceptor returns a unary server interceptor for authentication
func (i *GRPCAuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Check if method should skip authentication
		if i.shouldSkipMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Authenticate request
		newCtx, err := i.authenticate(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		// Call handler with authenticated context
		return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor for authentication
func (i *GRPCAuthInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method should skip authentication
		if i.shouldSkipMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// Authenticate request
		newCtx, err := i.authenticate(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		// Wrap the stream with the new context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		// Call handler with authenticated stream
		return handler(srv, wrappedStream)
	}
}

// authenticate performs the authentication logic
func (i *GRPCAuthInterceptor) authenticate(ctx context.Context, method string) (context.Context, error) {
	// Extract metadata from context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		i.logger.WithContext(ctx).WithFields(logging.Fields{
			"method": method,
		}).Warn("No metadata in gRPC request")
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Extract token from metadata
	token := i.extractToken(md)
	if token == "" {
		i.logger.WithContext(ctx).WithFields(logging.Fields{
			"method": method,
		}).Warn("Missing authentication token")
		return nil, status.Error(codes.Unauthenticated, "missing authentication token")
	}

	// Validate token
	claims, err := i.jwtService.ValidateToken(token)
	if err != nil {
		i.logger.WithContext(ctx).WithFields(logging.Fields{
			"error":  err.Error(),
			"method": method,
		}).Warn("Token validation failed")
		return nil, i.mapJWTErrorToGRPCStatus(err)
	}

	// Validate token type if required
	if i.options.RequireTokenType != "" && string(claims.TokenType) != i.options.RequireTokenType {
		i.logger.WithContext(ctx).WithFields(logging.Fields{
			"expected_type": i.options.RequireTokenType,
			"actual_type":   claims.TokenType,
			"user_id":       claims.UserID,
			"method":        method,
		}).Warn("Invalid token type")
		return nil, status.Error(codes.Unauthenticated, "invalid token type")
	}

	// Add claims to context
	newCtx := jwtcontext.WithClaims(ctx, claims)
	newCtx = jwtcontext.WithUserID(newCtx, claims.UserID)

	// Log successful authentication
	i.logger.WithContext(newCtx).WithFields(logging.Fields{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
		"method":  method,
	}).Debug("gRPC request authenticated")

	return newCtx, nil
}

// extractToken extracts the JWT token from gRPC metadata
func (i *GRPCAuthInterceptor) extractToken(md metadata.MD) string {
	// Use custom extractor if provided
	if i.options.TokenExtractor != nil {
		return i.options.TokenExtractor(md)
	}

	// Default: Try multiple metadata keys
	// Check "authorization" key first (most common)
	if values := md.Get("authorization"); len(values) > 0 {
		token := values[0]
		// Remove "Bearer " prefix if present
		if len(token) > 7 && strings.ToLower(token[:7]) == "bearer " {
			return token[7:]
		}
		return token
	}

	// Check "x-api-token" key
	if values := md.Get("x-api-token"); len(values) > 0 {
		return values[0]
	}

	// Check "x-auth-token" key
	if values := md.Get("x-auth-token"); len(values) > 0 {
		return values[0]
	}

	// Check "token" key
	if values := md.Get("token"); len(values) > 0 {
		return values[0]
	}

	return ""
}

// shouldSkipMethod checks if the method should skip authentication
func (i *GRPCAuthInterceptor) shouldSkipMethod(method string) bool {
	return slices.Contains(i.options.SkipMethods, method)
}

// mapJWTErrorToGRPCStatus maps JWT errors to gRPC status codes
func (i *GRPCAuthInterceptor) mapJWTErrorToGRPCStatus(err error) error {
	switch err {
	case jwt.ErrMissingToken:
		return status.Error(codes.Unauthenticated, "missing authentication token")
	case jwt.ErrTokenExpired:
		return status.Error(codes.Unauthenticated, "token has expired")
	case jwt.ErrInvalidSignature:
		return status.Error(codes.Unauthenticated, "invalid token signature")
	case jwt.ErrInvalidTokenType:
		return status.Error(codes.Unauthenticated, "invalid token type")
	case jwt.ErrNoneAlgorithm, jwt.ErrAlgorithmMismatch:
		return status.Error(codes.Unauthenticated, "invalid token")
	case jwt.ErrInvalidClaims:
		return status.Error(codes.Unauthenticated, "invalid token claims")
	default:
		return status.Error(codes.Unauthenticated, "authentication failed")
	}
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// RequireRoleUnaryInterceptor creates a unary interceptor that requires specific roles
func (i *GRPCAuthInterceptor) RequireRoleUnaryInterceptor(roles ...string) grpc.UnaryServerInterceptor {
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role] = true
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Get claims from context (should be set by authentication interceptor)
		claims, ok := jwtcontext.GetClaims(ctx)
		if !ok {
			i.logger.WithContext(ctx).WithFields(logging.Fields{
				"method": info.FullMethod,
			}).Warn("No claims in context for role check")
			return nil, status.Error(codes.Unauthenticated, "no authentication")
		}

		// Check if user has required role
		if !roleMap[claims.Role] {
			i.logger.WithContext(ctx).WithFields(logging.Fields{
				"user_id":        claims.UserID,
				"user_role":      claims.Role,
				"required_roles": roles,
				"method":         info.FullMethod,
			}).Warn("Insufficient permissions")
			return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
		}

		return handler(ctx, req)
	}
}

// RequireRoleStreamInterceptor creates a stream interceptor that requires specific roles
func (i *GRPCAuthInterceptor) RequireRoleStreamInterceptor(roles ...string) grpc.StreamServerInterceptor {
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role] = true
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Get claims from context (should be set by authentication interceptor)
		claims, ok := jwtcontext.GetClaims(ss.Context())
		if !ok {
			i.logger.WithContext(ss.Context()).WithFields(logging.Fields{
				"method": info.FullMethod,
			}).Warn("No claims in context for role check")
			return status.Error(codes.Unauthenticated, "no authentication")
		}

		// Check if user has required role
		if !roleMap[claims.Role] {
			i.logger.WithContext(ss.Context()).WithFields(logging.Fields{
				"user_id":        claims.UserID,
				"user_role":      claims.Role,
				"required_roles": roles,
				"method":         info.FullMethod,
			}).Warn("Insufficient permissions")
			return status.Error(codes.PermissionDenied, "insufficient permissions")
		}

		return handler(srv, ss)
	}
}

// OptionalAuthUnaryInterceptor returns a unary interceptor that tries to authenticate but doesn't fail
func (i *GRPCAuthInterceptor) OptionalAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// No metadata, continue without authentication
			return handler(ctx, req)
		}

		// Extract token from metadata
		token := i.extractToken(md)
		if token == "" {
			// No token, continue without authentication
			return handler(ctx, req)
		}

		// Try to validate token
		claims, err := i.jwtService.ValidateToken(token)
		if err != nil {
			// Invalid token, log but continue without authentication
			i.logger.WithContext(ctx).WithFields(logging.Fields{
				"error":  err.Error(),
				"method": info.FullMethod,
			}).Debug("Optional authentication failed")
			return handler(ctx, req)
		}

		// Add claims to context
		newCtx := jwtcontext.WithClaims(ctx, claims)
		newCtx = jwtcontext.WithUserID(newCtx, claims.UserID)

		// Call handler with authenticated context
		return handler(newCtx, req)
	}
}

// OptionalAuthStreamInterceptor returns a stream interceptor that tries to authenticate but doesn't fail
func (i *GRPCAuthInterceptor) OptionalAuthStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// No metadata, continue without authentication
			return handler(srv, ss)
		}

		// Extract token from metadata
		token := i.extractToken(md)
		if token == "" {
			// No token, continue without authentication
			return handler(srv, ss)
		}

		// Try to validate token
		claims, err := i.jwtService.ValidateToken(token)
		if err != nil {
			// Invalid token, log but continue without authentication
			i.logger.WithContext(ctx).WithFields(logging.Fields{
				"error":  err.Error(),
				"method": info.FullMethod,
			}).Debug("Optional authentication failed")
			return handler(srv, ss)
		}

		// Add claims to context
		newCtx := jwtcontext.WithClaims(ctx, claims)
		newCtx = jwtcontext.WithUserID(newCtx, claims.UserID)

		// Wrap the stream with the new context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		// Call handler with authenticated stream
		return handler(srv, wrappedStream)
	}
}

// TokenExtractorFromMetadataKey creates a token extractor from a specific metadata key
func TokenExtractorFromMetadataKey(key string) func(metadata.MD) string {
	return func(md metadata.MD) string {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
		return ""
	}
}

// ChainMetadataExtractors chains multiple metadata token extractors
// Returns the first non-empty token found
func ChainMetadataExtractors(extractors ...func(metadata.MD) string) func(metadata.MD) string {
	return func(md metadata.MD) string {
		for _, extractor := range extractors {
			if token := extractor(md); token != "" {
				return token
			}
		}
		return ""
	}
}

// GetClaimsFromGRPCContext retrieves JWT claims from the gRPC context
// This is a convenience function for gRPC handlers
func GetClaimsFromGRPCContext(ctx context.Context) (*claims.CustomClaims, bool) {
	return jwtcontext.GetClaims(ctx)
}

// GetUserIDFromGRPCContext retrieves user ID from the gRPC context
// This is a convenience function for gRPC handlers
func GetUserIDFromGRPCContext(ctx context.Context) (string, bool) {
	return jwtcontext.GetUserID(ctx)
}

// MustGetClaimsFromGRPCContext retrieves JWT claims from context or panics
// Use this only when you're certain the claims exist
func MustGetClaimsFromGRPCContext(ctx context.Context) *claims.CustomClaims {
	return jwtcontext.MustGetClaims(ctx)
}

// MustGetUserIDFromGRPCContext retrieves user ID from context or panics
// Use this only when you're certain the user ID exists
func MustGetUserIDFromGRPCContext(ctx context.Context) string {
	return jwtcontext.MustGetUserID(ctx)
}
