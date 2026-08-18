package middleware

import (
	"context"
	"net/http"
	"strings"

	"base_project/pkg/jwt"
	"base_project/pkg/jwt/claims"
	jwtcontext "base_project/pkg/jwt/context"
	"base_project/pkg/logging"
)

// HTTPAuthMiddleware provides HTTP authentication middleware
type HTTPAuthMiddleware struct {
	jwtService jwt.Service
	logger     logging.Logger
	options    HTTPAuthOptions
}

// HTTPAuthOptions configures the HTTP authentication middleware
type HTTPAuthOptions struct {
	// TokenExtractor is a custom function to extract token from request
	// If nil, uses default bearer token extraction
	TokenExtractor func(*http.Request) string

	// SkipPaths are paths that should skip authentication
	SkipPaths []string

	// ErrorHandler is a custom error handler
	// If nil, uses default error response
	ErrorHandler func(http.ResponseWriter, *http.Request, error)

	// RequireTokenType specifies required token type (access or refresh)
	// If empty, any token type is accepted
	RequireTokenType string
}

// NewHTTPAuthMiddleware creates a new HTTP authentication middleware
func NewHTTPAuthMiddleware(jwtService jwt.Service, logger logging.Logger) *HTTPAuthMiddleware {
	return &HTTPAuthMiddleware{
		jwtService: jwtService,
		logger:     logger,
		options:    HTTPAuthOptions{},
	}
}

// NewHTTPAuthMiddlewareWithOptions creates a new HTTP authentication middleware with custom options
func NewHTTPAuthMiddlewareWithOptions(jwtService jwt.Service, logger logging.Logger, options HTTPAuthOptions) *HTTPAuthMiddleware {
	return &HTTPAuthMiddleware{
		jwtService: jwtService,
		logger:     logger,
		options:    options,
	}
}

// Authenticate is the main authentication middleware
func (m *HTTPAuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should skip authentication
		if m.shouldSkipPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from request
		token := m.extractToken(r)
		if token == "" {
			m.handleError(w, r, jwt.ErrMissingToken)
			return
		}

		// Validate token
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			// Log error details for debugging but don't expose sensitive info
			m.logger.WithContext(r.Context()).WithFields(logging.Fields{
				"error_type": "token_validation_failed",
				"path":       r.URL.Path,
				"method":     r.Method,
			}).Warn("Authentication failed")
			m.handleError(w, r, err)
			return
		}

		// Validate token type if required
		if m.options.RequireTokenType != "" && string(claims.TokenType) != m.options.RequireTokenType {
			m.logger.WithContext(r.Context()).WithFields(logging.Fields{
				"error_type": "invalid_token_type",
				"path":       r.URL.Path,
				"method":     r.Method,
			}).Warn("Invalid token type")
			m.handleError(w, r, jwt.ErrInvalidTokenType)
			return
		}

		// Add claims to context
		ctx := jwtcontext.WithClaims(r.Context(), claims)
		ctx = jwtcontext.WithUserID(ctx, claims.UserID)

		// Log successful authentication
		m.logger.WithContext(ctx).WithFields(logging.Fields{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
			"path":    r.URL.Path,
			"method":  r.Method,
		}).Debug("Request authenticated")

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateFunc is a convenience wrapper for http.HandlerFunc
func (m *HTTPAuthMiddleware) AuthenticateFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Authenticate(next).ServeHTTP
}

// OptionalAuthenticate tries to authenticate but doesn't fail if token is missing or invalid
// Useful for endpoints that provide enhanced features for authenticated users
func (m *HTTPAuthMiddleware) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		token := m.extractToken(r)
		if token == "" {
			// No token, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate token
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			// Invalid token, log but continue without authentication
			m.logger.WithContext(r.Context()).WithFields(logging.Fields{
				"error": err.Error(),
			}).Debug("Optional authentication failed")
			next.ServeHTTP(w, r)
			return
		}

		// Add claims to context
		ctx := jwtcontext.WithClaims(r.Context(), claims)
		ctx = jwtcontext.WithUserID(ctx, claims.UserID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole creates a middleware that requires specific role
func (m *HTTPAuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context (should be set by Authenticate middleware)
			claims, ok := jwtcontext.GetClaims(r.Context())
			if !ok {
				m.logger.WithContext(r.Context()).Warn("No claims in context for role check")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if user has required role
			if !roleMap[claims.Role] {
				m.logger.WithContext(r.Context()).WithFields(logging.Fields{
					"user_id":        claims.UserID,
					"user_role":      claims.Role,
					"required_roles": roles,
				}).Warn("Insufficient permissions")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extracts the JWT token from the request
func (m *HTTPAuthMiddleware) extractToken(r *http.Request) string {
	// Use custom extractor if provided
	if m.options.TokenExtractor != nil {
		return m.options.TokenExtractor(r)
	}

	// Default: Extract from Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// shouldSkipPath checks if the path should skip authentication
func (m *HTTPAuthMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.options.SkipPaths {
		if path == skipPath || strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// handleError handles authentication errors
func (m *HTTPAuthMiddleware) handleError(w http.ResponseWriter, r *http.Request, err error) {
	// Use custom error handler if provided
	if m.options.ErrorHandler != nil {
		m.options.ErrorHandler(w, r, err)
		return
	}

	// Default error handling
	statusCode := http.StatusUnauthorized
	message := "Unauthorized"

	switch err {
	case jwt.ErrMissingToken:
		message = "Missing authentication token"
	case jwt.ErrTokenExpired:
		message = "Token has expired"
	case jwt.ErrInvalidSignature:
		message = "Invalid token"
	case jwt.ErrInvalidTokenType:
		message = "Invalid token type"
	case jwt.ErrNoneAlgorithm, jwt.ErrAlgorithmMismatch:
		message = "Invalid token"
	default:
		message = "Authentication failed"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`)) // #nosec
}

// GetClaimsFromContext retrieves JWT claims from the request context
// This is a convenience function for handlers
func GetClaimsFromContext(ctx context.Context) (*claims.CustomClaims, bool) {
	return jwtcontext.GetClaims(ctx)
}

// GetUserIDFromContext retrieves user ID from the request context
// This is a convenience function for handlers
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	return jwtcontext.GetUserID(ctx)
}

// MustGetClaimsFromContext retrieves JWT claims from context or panics
// Use this only when you're certain the claims exist
func MustGetClaimsFromContext(ctx context.Context) *claims.CustomClaims {
	return jwtcontext.MustGetClaims(ctx)
}

// MustGetUserIDFromContext retrieves user ID from context or panics
// Use this only when you're certain the user ID exists
func MustGetUserIDFromContext(ctx context.Context) string {
	return jwtcontext.MustGetUserID(ctx)
}

// TokenExtractorFromHeader creates a token extractor from a specific header
func TokenExtractorFromHeader(headerName string) func(*http.Request) string {
	return func(r *http.Request) string {
		return r.Header.Get(headerName)
	}
}

// TokenExtractorFromCookie creates a token extractor from a cookie
func TokenExtractorFromCookie(cookieName string) func(*http.Request) string {
	return func(r *http.Request) string {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			return ""
		}
		return cookie.Value
	}
}

// TokenExtractorFromQuery creates a token extractor from a query parameter
func TokenExtractorFromQuery(paramName string) func(*http.Request) string {
	return func(r *http.Request) string {
		return r.URL.Query().Get(paramName)
	}
}

// ChainTokenExtractors chains multiple token extractors
// Returns the first non-empty token found
func ChainTokenExtractors(extractors ...func(*http.Request) string) func(*http.Request) string {
	return func(r *http.Request) string {
		for _, extractor := range extractors {
			if token := extractor(r); token != "" {
				return token
			}
		}
		return ""
	}
}

// ExtractBearerFromRequest extracts the Bearer token value from the Authorization
// header of r. Returns an empty string when the header is absent or does not
// follow the "Bearer <token>" format.
//
// Use this helper in middleware so that delivery-layer files never reference
// r.Header.Get("Authorization") directly (api-standards.md Section 5).
func ExtractBearerFromRequest(r *http.Request) string {
	return strings.TrimPrefix(
		strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
		"bearer ",
	)
}
