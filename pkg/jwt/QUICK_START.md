# JWT Package - Quick Start Guide

Get started with JWT authentication in 5 minutes.

## Installation

```bash
go get github.com/golang-jwt/jwt/v5
```

## Basic Usage (HS256)

### 1. Create JWT Service

```go
import (
    "time"
    "base_project/pkg/jwt"
)

// Create configuration
config := &jwt.Config{
    Algorithm:              jwt.AlgorithmHS256,
    SecretKey:              "your-secret-key-at-least-32-characters-long!",
    AccessTokenExpiration:  15 * time.Minute,
    RefreshTokenExpiration: 7 * 24 * time.Hour,
    Issuer:                 "my-service",
    Audience:               "my-service-users",
}

// Create service
jwtService, err := jwt.New(config)
if err != nil {
    panic(err)
}
```

### 2. Generate Tokens

```go
// Generate access token
accessToken, err := jwtService.GenerateAccessToken(
    "user-123",              // userID (required)
    "user@example.com",      // email (optional)
    "admin",                 // role (optional)
    nil,                     // extra fields (optional)
)

// Generate refresh token
refreshToken, err := jwtService.GenerateRefreshToken(
    "user-123",              // userID (required)
    nil,                     // extra fields (optional)
)
```

### 3. Validate Tokens

```go
// Validate and extract claims
claims, err := jwtService.ValidateToken(accessToken)
if err != nil {
    // Handle error (expired, invalid signature, etc.)
    log.Printf("Token validation failed: %v", err)
    return
}

// Access user information
userID := claims.UserID
email := claims.Email
role := claims.Role
```

### 4. Refresh Access Token

```go
// Generate new access token from refresh token
newAccessToken, err := jwtService.RefreshAccessToken(
    refreshToken,
    "user@example.com",     // updated email
    "admin",                // updated role
)
```

## HTTP Authentication Example

### Setup Middleware

```go
import (
    "net/http"
    "base_project/pkg/jwt/middleware"
    "base_project/pkg/logging"
    "chjpcoj.com.innerlamp/pkg/utils" // envelope response chuẩn — xem pkg/utils/README.md
)

func main() {
    // Create JWT service
    jwtService, _ := jwt.New(config)
    
    // Create logger
    logger := logging.New(&logging.Config{Level: "info"})
    
    // Create authentication middleware
    authMiddleware := middleware.NewHTTPAuthMiddleware(jwtService, logger)
    
    // Protected route
    http.Handle("/api/profile", authMiddleware.Authenticate(
        http.HandlerFunc(profileHandler),
    ))
    
    // Public route
    http.HandleFunc("/api/login", loginHandler)
    
    http.ListenAndServe(":8080", nil)
}
```

### Login Handler (Generate Tokens)

```go
func loginHandler(w http.ResponseWriter, r *http.Request) {
    // Validate credentials (pseudo-code)
    user := authenticateUser(r)
    if user == nil {
        utils.WriteBusinessError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials", nil)
        return
    }
    
    // Generate tokens
    accessToken, _ := jwtService.GenerateAccessToken(
        user.ID,
        user.Email,
        user.Role,
        nil,
    )
    
    refreshToken, _ := jwtService.GenerateRefreshToken(user.ID, nil)
    
    // Return tokens
    utils.WriteSuccessResponse(w, map[string]string{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
    })
}
```

### Protected Handler (Access User Info)

```go
import "base_project/pkg/jwt/middleware"

func profileHandler(w http.ResponseWriter, r *http.Request) {
    // Get claims from context (set by middleware)
    claims, ok := middleware.GetClaimsFromContext(r.Context())
    if !ok {
        utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
        return
    }
    
    // Access user information
    userID := claims.UserID
    email := claims.Email
    role := claims.Role
    
    // Your business logic here
    utils.WriteSuccessResponse(w, map[string]interface{}{
        "user_id": userID,
        "email":   email,
        "role":    role,
    })
}
```

## gRPC Authentication Example

### Setup Interceptor

```go
import (
    "google.golang.org/grpc"
    "base_project/pkg/jwt/middleware"
)

func main() {
    // Create JWT service
    jwtService, _ := jwt.New(config)
    
    // Create logger
    logger := logging.New(&logging.Config{Level: "info"})
    
    // Create authentication interceptor
    authInterceptor := middleware.NewGRPCAuthInterceptor(jwtService, logger)
    
    // Create gRPC server with interceptor
    server := grpc.NewServer(
        grpc.UnaryInterceptor(authInterceptor.UnaryServerInterceptor()),
        grpc.StreamInterceptor(authInterceptor.StreamServerInterceptor()),
    )
    
    // Register your services
    pb.RegisterYourServiceServer(server, &yourServiceImpl{})
    
    // Start server
    lis, _ := net.Listen("tcp", ":9090")
    server.Serve(lis)
}
```

### gRPC Handler (Access User Info)

```go
import "base_project/pkg/jwt/middleware"

func (s *yourServiceImpl) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
    // Get claims from context (set by interceptor)
    claims, ok := middleware.GetClaimsFromContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "no authentication")
    }
    
    // Access user information
    userID := claims.UserID
    email := claims.Email
    role := claims.Role
    
    // Your business logic here
    return &pb.GetProfileResponse{
        UserId: userID,
        Email:  email,
        Role:   role,
    }, nil
}
```

## RS256 (Asymmetric Keys)

For microservices architecture, use RS256 with public/private key pairs:

### Generate RSA Keys

```bash
# Generate private key (4096-bit)
openssl genrsa -out private_key.pem 4096

# Extract public key
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

### Auth Service (Signs Tokens)

```go
import "base_project/pkg/jwt"

// Load private key for signing
privateKey, err := jwt.LoadRSAPrivateKeyFromFile("private_key.pem")
publicKey, err := jwt.LoadRSAPublicKeyFromFile("public_key.pem")

config := &jwt.Config{
    Algorithm:              jwt.AlgorithmRS256,
    PrivateKey:             privateKey,
    PublicKey:              publicKey,
    AccessTokenExpiration:  15 * time.Minute,
    RefreshTokenExpiration: 7 * 24 * time.Hour,
    Issuer:                 "auth-service",
    Audience:               "all-services",
}

jwtService, _ := jwt.New(config)

// Generate tokens (same as HS256)
accessToken, _ := jwtService.GenerateAccessToken("user-123", "user@example.com", "admin", nil)
```

### Other Services (Verify Tokens)

```go
// Load only public key for verification
publicKey, err := jwt.LoadRSAPublicKeyFromFile("public_key.pem")

config := &jwt.Config{
    Algorithm:              jwt.AlgorithmRS256,
    PublicKey:              publicKey,
    PrivateKey:             publicKey.(*rsa.PublicKey), // Won't be used for verification
    AccessTokenExpiration:  15 * time.Minute,
    RefreshTokenExpiration: 7 * 24 * time.Hour,
    Issuer:                 "auth-service",
    Audience:               "all-services",
}

jwtService, _ := jwt.New(config)

// Validate tokens (same as HS256)
claims, _ := jwtService.ValidateToken(accessToken)
```

## Error Handling

```go
import "errors"

claims, err := jwtService.ValidateToken(token)
if err != nil {
    switch {
    case errors.Is(err, jwt.ErrTokenExpired):
        // Token expired - prompt refresh
        return http.StatusUnauthorized, "Token expired"
        
    case errors.Is(err, jwt.ErrInvalidSignature):
        // Invalid signature - possible tampering
        return http.StatusUnauthorized, "Invalid token"
        
    case errors.Is(err, jwt.ErrAlgorithmMismatch):
        // Algorithm mismatch - security issue
        return http.StatusUnauthorized, "Invalid token"
        
    case errors.Is(err, jwt.ErrNoneAlgorithm):
        // 'none' algorithm attempt - security issue
        return http.StatusUnauthorized, "Invalid token"
        
    default:
        // Other errors
        return http.StatusUnauthorized, "Authentication failed"
    }
}
```

## Working with Extra Claims

```go
// Add extra fields during generation
extra := map[string]interface{}{
    "org_id":      "org-456",
    "tenant_id":   "tenant-789",
    "permissions": []string{"read", "write", "delete"},
}

token, _ := jwtService.GenerateAccessToken("user-123", "user@example.com", "admin", extra)

// Retrieve extra fields after validation
claims, _ := jwtService.ValidateToken(token)

if orgID, ok := claims.GetExtra("org_id"); ok {
    fmt.Printf("Organization: %v\n", orgID)
}

if permissions, ok := claims.GetExtra("permissions"); ok {
    perms := permissions.([]interface{})
    // Use permissions
}
```

## Token Refresh Flow

```go
func handleRefresh(w http.ResponseWriter, r *http.Request) {
    // Get refresh token from request
    refreshToken := r.Header.Get("X-Refresh-Token")
    
    // Validate refresh token
    claims, err := jwtService.ValidateToken(refreshToken)
    if err != nil {
        utils.WriteBusinessError(w, http.StatusUnauthorized, "REFRESH_TOKEN_INVALID", "invalid refresh token", nil)
        return
    }
    
    // Verify it's a refresh token
    if claims.TokenType != "refresh" {
        utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token type", nil)
        return
    }
    
    // Fetch updated user info from database
    user := getUserFromDB(claims.UserID)
    
    // Generate new access token
    newAccessToken, err := jwtService.GenerateAccessToken(
        user.ID,
        user.Email,
        user.Role,
        nil,
    )
    
    // Optionally: Generate new refresh token (rotation)
    newRefreshToken, err := jwtService.GenerateRefreshToken(user.ID, nil)
    
    // Return new tokens
    utils.WriteSuccessResponse(w, map[string]string{
        "access_token":  newAccessToken,
        "refresh_token": newRefreshToken,
    })
}
```

## Configuration Best Practices

### Development

```go
config := &jwt.Config{
    Algorithm:              jwt.AlgorithmHS256,
    SecretKey:              "dev-secret-key-32-characters-long!",
    AccessTokenExpiration:  1 * time.Hour,      // Longer for testing
    RefreshTokenExpiration: 30 * 24 * time.Hour, // 30 days
    Issuer:                 "dev-service",
    Audience:               "dev-users",
}
```

### Production

```go
// Load secret from environment or secret manager
secretKey := os.Getenv("JWT_SECRET_KEY")

config := &jwt.Config{
    Algorithm:              jwt.AlgorithmRS256,  // Use RS256 in production
    PrivateKey:             loadPrivateKey(),
    PublicKey:              loadPublicKey(),
    AccessTokenExpiration:  15 * time.Minute,    // Short-lived
    RefreshTokenExpiration: 7 * 24 * time.Hour,  // 7 days
    Issuer:                 "prod-auth-service",
    Audience:               "prod-users",
}
```

## Common Patterns

### Middleware Chain

```go
// Combine with logging and CORS
http.Handle("/api/protected",
    corsMiddleware(
        loggingMiddleware(
            authMiddleware.Authenticate(
                http.HandlerFunc(protectedHandler),
            ),
        ),
    ),
)
```

### Optional Authentication

```go
func optionalAuthHandler(w http.ResponseWriter, r *http.Request) {
    // Try to get claims (may be nil for unauthenticated requests)
    claims, _ := middleware.GetClaimsFromContext(r.Context())
    
    if claims != nil {
        // User is authenticated
        // Return personalized content
    } else {
        // User is not authenticated
        // Return public content
    }
}
```

### Role-Based Access Control

```go
func adminOnlyHandler(w http.ResponseWriter, r *http.Request) {
    claims, ok := middleware.GetClaimsFromContext(r.Context())
    if !ok {
        utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
        return
    }
    
    // Check role
    if claims.Role != "admin" {
        utils.WriteBusinessError(w, http.StatusForbidden, "FORBIDDEN", "forbidden", nil)
        return
    }
    
    // Admin-only logic
}
```

## Testing

```go
import "testing"

func TestJWTFlow(t *testing.T) {
    // Create test service
    config := &jwt.Config{
        Algorithm:              jwt.AlgorithmHS256,
        SecretKey:              "test-secret-key-32-characters-long!",
        AccessTokenExpiration:  15 * time.Minute,
        RefreshTokenExpiration: 7 * 24 * time.Hour,
        Issuer:                 "test",
        Audience:               "test-users",
    }
    
    service, err := jwt.New(config)
    if err != nil {
        t.Fatalf("Failed to create service: %v", err)
    }
    
    // Generate token
    token, err := service.GenerateAccessToken("test-user", "test@example.com", "user", nil)
    if err != nil {
        t.Fatalf("Failed to generate token: %v", err)
    }
    
    // Validate token
    claims, err := service.ValidateToken(token)
    if err != nil {
        t.Fatalf("Failed to validate token: %v", err)
    }
    
    // Verify claims
    if claims.UserID != "test-user" {
        t.Errorf("Expected user_id 'test-user', got '%s'", claims.UserID)
    }
}
```

## Next Steps

- See [README.md](README.md) for comprehensive documentation
- Check [examples/](examples/) for complete working examples
- Review [middleware/](middleware/) for HTTP and gRPC integration
- Read security best practices in the main README

## Troubleshooting

**"invalid signing key"**
- HS256: Ensure secret is at least 32 characters
- RS256: Verify PEM file format and paths

**"token has expired"**
- Implement token refresh flow
- Check system clock synchronization

**"algorithm mismatch"**
- Ensure all services use the same algorithm
- Verify configuration is consistent

**"invalid token signature"**
- Verify secret key matches
- For RS256, ensure public key matches private key

For more help, see the main [README.md](README.md) or check the tests for usage examples.