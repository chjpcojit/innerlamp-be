// ⚠️ MOCK DATA - NOT FOR PRODUCTION
// This file is an example only. It uses hardcoded in-memory user data
// and plaintext passwords for illustration purposes.
//
// Before using in production:
//   - Replace the in-memory user map with a real database repository
//   - Use bcrypt-hashed passwords (see pkg/password/)
//   - Load credentials via pkg/secrets/ from GCP Secret Manager
//   - Implement proper user lookup against Cloud SQL / MongoDB
//
// MOCK_DATA_MARKER: users map, plaintext passwords
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"base_project/pkg/jwt"
	"base_project/pkg/jwt/middleware"
	"base_project/pkg/logging"
	"base_project/pkg/logging/formatter"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // Never send password in response
	Role     string `json:"role"`
}

// ⚠️ MOCK DATA - NOT FOR PRODUCTION
// Replace with a real database repository backed by Cloud SQL or MongoDB.
// Passwords must be hashed with bcrypt (see pkg/password/) — never store plaintext.
var users = map[string]*User{
	"admin@example.com": {
		ID:       "user-1",
		Email:    "admin@example.com",
		Password: "admin123", // ⚠️ MOCK ONLY — use bcrypt hash in production
		Role:     "admin",
	},
	"user@example.com": {
		ID:       "user-2",
		Email:    "user@example.com",
		Password: "user123", // ⚠️ MOCK ONLY — use bcrypt hash in production
		Role:     "user",
	},
}

// Server holds the dependencies
type Server struct {
	jwtService     jwt.Service
	logger         logging.Logger
	authMiddleware *middleware.HTTPAuthMiddleware
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents login response with tokens
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

// RefreshRequest represents token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents token refresh response
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// ProfileResponse represents user profile response
type ProfileResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

func main() {
	// Create JWT service with HS256
	jwtConfig := &jwt.Config{
		Algorithm:              jwt.AlgorithmHS256,
		SecretKey:              "your-super-secret-key-min-32-chars!!",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "example-api",
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

	// Create authentication middleware
	authMiddleware := middleware.NewHTTPAuthMiddleware(jwtService, logger)

	// Create server
	server := &Server{
		jwtService:     jwtService,
		logger:         logger,
		authMiddleware: authMiddleware,
	}

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/api/login", server.loginHandler)
	mux.HandleFunc("/api/refresh", server.refreshHandler)
	mux.HandleFunc("/api/health", server.healthHandler)

	// Protected routes
	mux.Handle("/api/profile", authMiddleware.Authenticate(http.HandlerFunc(server.profileHandler)))
	mux.Handle("/api/users", authMiddleware.Authenticate(http.HandlerFunc(server.usersHandler)))

	// Admin only routes
	mux.Handle("/api/admin/users", authMiddleware.Authenticate(
		authMiddleware.RequireRole("admin")(
			http.HandlerFunc(server.adminUsersHandler),
		),
	))

	// Optional authentication (enhanced for authenticated users)
	mux.Handle("/api/content", authMiddleware.OptionalAuthenticate(http.HandlerFunc(server.contentHandler)))

	// Start server
	addr := ":8080"
	logger.Info(fmt.Sprintf("Starting HTTP server on %s", addr))
	logger.Info("Try the following:")
	logger.Info("  curl -X POST http://localhost:8080/api/login -H 'Content-Type: application/json' -d '{\"email\":\"admin@example.com\",\"password\":\"admin123\"}'")
	logger.Info("  curl http://localhost:8080/api/profile -H 'Authorization: Bearer <access_token>'")
	logger.Info("  curl http://localhost:8080/api/admin/users -H 'Authorization: Bearer <access_token>'")

	if err := http.ListenAndServe(addr, mux); err != nil { // #nosec
		log.Fatalf("Server failed: %v", err)
	}
}

// loginHandler handles user login
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate credentials
	user, ok := users[req.Email]
	if !ok || user.Password != req.Password {
		s.logger.WithContext(r.Context()).Warn("Failed login attempt", "email", req.Email)
		s.sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate access token
	accessToken, err := s.jwtService.GenerateAccessToken(
		user.ID,
		user.Email,
		user.Role,
		map[string]any{
			"login_time": time.Now().Unix(),
		},
	)
	if err != nil {
		s.logger.WithContext(r.Context()).Error("Failed to generate access token", "error", err.Error())
		s.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate refresh token
	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID, nil)
	if err != nil {
		s.logger.WithContext(r.Context()).Error("Failed to generate refresh token", "error", err.Error())
		s.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log successful login
	s.logger.WithContext(r.Context()).Info("User logged in successfully", "user_id", user.ID, "email", user.Email)

	// Send response
	s.sendJSON(w, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, http.StatusOK)
}

// refreshHandler handles token refresh
func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate refresh token
	claims, err := s.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		s.logger.WithContext(r.Context()).Warn("Invalid refresh token", "error", err.Error())
		s.sendError(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Verify it's a refresh token
	if claims.TokenType != "refresh" {
		s.sendError(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	// Get user from claims
	var user *User
	for _, u := range users {
		if u.ID == claims.UserID {
			user = u
			break
		}
	}

	if user == nil {
		s.sendError(w, "User not found", http.StatusNotFound)
		return
	}

	// Generate new access token
	newAccessToken, err := s.jwtService.GenerateAccessToken(
		user.ID,
		user.Email,
		user.Role,
		nil,
	)
	if err != nil {
		s.logger.WithContext(r.Context()).Error("Failed to generate new access token", "error", err.Error())
		s.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.logger.WithContext(r.Context()).Info("Token refreshed", "user_id", user.ID)

	// Send response
	s.sendJSON(w, RefreshResponse{
		AccessToken: newAccessToken,
	}, http.StatusOK)
}

// profileHandler returns authenticated user's profile
func (s *Server) profileHandler(w http.ResponseWriter, r *http.Request) {
	// Get claims from context (set by authentication middleware)
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		s.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s.logger.WithContext(r.Context()).Debug("Profile accessed", "user_id", claims.UserID)

	// Send user profile
	s.sendJSON(w, ProfileResponse{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, http.StatusOK)
}

// usersHandler lists all users (authenticated users only)
func (s *Server) usersHandler(w http.ResponseWriter, r *http.Request) {
	// Get claims from context
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		s.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s.logger.WithContext(r.Context()).Debug("Users list accessed", "user_id", claims.UserID)

	// Return list of users (without passwords)
	var userList []*User
	for _, u := range users {
		userList = append(userList, &User{
			ID:    u.ID,
			Email: u.Email,
			Role:  u.Role,
		})
	}

	s.sendJSON(w, userList, http.StatusOK)
}

// adminUsersHandler is an admin-only endpoint
func (s *Server) adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Get claims from context
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		s.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s.logger.WithContext(r.Context()).Info("Admin users accessed", "user_id", claims.UserID)

	// Return detailed user list (admin view)
	s.sendJSON(w, map[string]any{
		"message": "This is an admin-only endpoint",
		"users":   users,
		"count":   len(users),
	}, http.StatusOK)
}

// contentHandler demonstrates optional authentication
func (s *Server) contentHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get claims (may not exist if not authenticated)
	claims, ok := middleware.GetClaimsFromContext(r.Context())

	var response map[string]any
	if ok {
		// User is authenticated - show personalized content
		response = map[string]any{
			"message": "Welcome back!",
			"user_id": claims.UserID,
			"email":   claims.Email,
			"content": "This is personalized content for authenticated users",
			"premium": true,
		}
	} else {
		// User is not authenticated - show public content
		response = map[string]any{
			"message": "Welcome, guest!",
			"content": "This is public content. Login for more features.",
			"premium": false,
		}
	}

	s.sendJSON(w, response, http.StatusOK)
}

// healthHandler is a health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}, http.StatusOK)
}

// sendJSON sends a JSON response
func (s *Server) sendJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data) // #nosec
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	s.sendJSON(w, ErrorResponse{Error: message}, statusCode)
}
