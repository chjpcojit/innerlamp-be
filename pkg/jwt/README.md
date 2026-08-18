# JWT Package

A comprehensive JWT (JSON Web Token) package for microservices with support for both HS256 (HMAC) and RS256 (RSA) algorithms.

## Features

- **Dual Algorithm Support**: HS256 (symmetric) and RS256 (asymmetric) signing algorithms
- **Token Types**: Separate access and refresh tokens with different expiration times
- **Custom Claims**: Support for user ID, email, role, and custom fields
- **Security Hardened**:
  - Explicitly rejects 'none' algorithm
  - Validates algorithm in token header matches expected algorithm
  - Prevents algorithm confusion attacks
- **Middleware Ready**: HTTP middleware and gRPC interceptors included
- **Context Integration**: Works seamlessly with correlation ID and tracing systems
- **Configuration-Based**: No environment variables required, fully configurable

## Installation

```bash
go get github.com/golang-jwt/jwt/v5
```

## Quick Start

### HS256 (Symmetric Key)

```go
package main

import (
    "time"
    "base_project/pkg/jwt"
)

func main() {
    // Create configuration
    config := &jwt.Config{
        Algorithm:              jwt.AlgorithmHS256,
        SecretKey:              "your-secret-key-must-be-at-least-32-characters-long",
        AccessTokenExpiration:  15 * time.Minute,
        RefreshTokenExpiration: 7 * 24 * time.Hour,
        Issuer:                 "my-service",
        Audience:               "my-service-users",
    }

    // Create JWT service
    jwtService, err := jwt.New(config)
    if err != nil {
        panic(err)
    }

    // Generate access token
    accessToken, err := jwtService.GenerateAccessToken(
        "user-123",           // userID
        "user@example.com",   // email
        "admin",              // role
        nil,                  // extra fields (optional)
    )

    // Generate refresh token
    refreshToken, err := jwtService.GenerateRefreshToken(
        "user-123",
        map[string]interface{}{"device_id": "device-789"},
    )

    // Validate token
    claims, err := jwtService.ValidateToken(accessToken)
    if err != nil {
        // Handle error (expired, invalid signature, etc.)
    }

    // Access claims
    userID := claims.UserID
    email := claims.Email
    role := claims.Role
}
```

### RS256 (Asymmetric Keys)

```go
package main

import (
    "time"
    "base_project/pkg/jwt"
)

func main() {
    // Load RSA keys from files
    privateKey, err := jwt.LoadRSAPrivateKeyFromFile("private_key.pem")
    if err != nil {
        panic(err)
    }

    publicKey, err := jwt.LoadRSAPublicKeyFromFile("public_key.pem")
    if err != nil {
        panic(err)
    }

    // Create configuration
    config := &jwt.Config{
        Algorithm:              jwt.AlgorithmRS256,
        PrivateKey:             privateKey,
        PublicKey:              publicKey,
        AccessTokenExpiration:  15 * time.Minute,
        RefreshTokenExpiration: 7 * 24 * time.Hour,
        Issuer:                 "my-service",
        Audience:               "my-service-users",
    }

    // Create JWT service
    jwtService, err := jwt.New(config)
    if err != nil {
        panic(err)
    }

    // Use same as HS256
    accessToken, err := jwtService.GenerateAccessToken("user-123", "user@example.com", "admin", nil)
}
```

## Configuration

### Config Structure

```go
type Config struct {
    // Algorithm: "HS256" or "RS256"
    Algorithm Algorithm

    // For HS256: Secret key (minimum 32 characters)
    SecretKey string

    // For RS256: Private key for signing
    PrivateKey *rsa.PrivateKey

    // For RS256: Public key for verification
    PublicKey *rsa.PublicKey

    // Token expiration durations
    AccessTokenExpiration  time.Duration
    RefreshTokenExpiration time.Duration

    // Standard JWT claims
    Issuer   string // "iss" claim
    Audience string // "aud" claim
}
```

### Configuration Validation

The configuration is automatically validated when creating a new service:

- HS256 requires a secret key of at least 32 characters
- RS256 requires both private and public keys
- Expiration times must be positive
- Refresh token expiration must be greater than access token expiration
- Only HS256 and RS256 algorithms are supported

## Custom Claims

### Structure

```go
type CustomClaims struct {
    UserID    string                 `json:"user_id"`
    Email     string                 `json:"email,omitempty"`
    Role      string                 `json:"role,omitempty"`
    TokenType TokenType              `json:"token_type"` // "access" or "refresh"
    Extra     map[string]interface{} `json:"extra,omitempty"`

    jwt.RegisteredClaims // iss, sub, aud, exp, nbf, iat, jti
}
```

### Working with Extra Fields

```go
// Add extra fields when generating token
extra := map[string]interface{}{
    "org_id":    "org-456",
    "tenant_id": "tenant-789",
    "permissions": []string{"read", "write"},
}

token, err := jwtService.GenerateAccessToken("user-123", "user@example.com", "admin", extra)

// Retrieve extra fields from claims
claims, err := jwtService.ValidateToken(token)
if orgID, ok := claims.GetExtra("org_id"); ok {
    // Use orgID
}

// Add extra field to claims
claims.SetExtra("new_field", "value")
```

## Token Operations

### Generate Access Token

```go
token, err := jwtService.GenerateAccessToken(
    userID,  // required
    email,   // optional (can be empty)
    role,    // optional (can be empty)
    extra,   // optional (can be nil)
)
```

### Generate Refresh Token

```go
token, err := jwtService.GenerateRefreshToken(
    userID,  // required
    extra,   // optional (can be nil)
)
```

### Validate Token

```go
claims, err := jwtService.ValidateToken(tokenString)
if err != nil {
    switch err {
    case jwt.ErrTokenExpired:
        // Handle expired token
    case jwt.ErrInvalidSignature:
        // Handle invalid signature
    case jwt.ErrAlgorithmMismatch:
        // Handle algorithm mismatch
    case jwt.ErrNoneAlgorithm:
        // Handle 'none' algorithm attempt
    default:
        // Handle other errors
    }
}
```

### Parse Token Without Validation

Useful for extracting claims from expired tokens or debugging:

```go
claims, err := jwtService.ParseToken(tokenString)
// Note: This does NOT validate expiration or signature
// Still validates algorithm to prevent security issues
```

### Refresh Access Token

```go
// Generate new access token from refresh token
newAccessToken, err := jwtService.RefreshAccessToken(
    refreshToken,
    "user@example.com", // updated email
    "admin",            // updated role
)
```

## Security Features

### Algorithm Validation

The package implements multiple layers of algorithm validation:

1. **Explicit 'none' rejection**: Tokens using the 'none' algorithm are rejected immediately
2. **Algorithm header validation**: The algorithm in the token header must match the expected algorithm
3. **Type assertion**: The signing method type is verified to match the configured algorithm
4. **Key type validation**: Ensures the key type matches the algorithm (HMAC for HS256, RSA for RS256)

```go
// This will fail - algorithm mismatch attack
hs256Service := createHS256Service()
rs256Token := createRS256Token()
_, err := hs256Service.ValidateToken(rs256Token)
// err == ErrAlgorithmMismatch

// This will fail - 'none' algorithm
noneToken := createNoneToken()
_, err := hs256Service.ValidateToken(noneToken)
// err == ErrNoneAlgorithm
```

### Key Requirements

**HS256 (Symmetric)**:
- Minimum 32 characters
- Use cryptographically secure random strings
- Never commit to version control
- Rotate periodically

**RS256 (Asymmetric)**:
- Minimum 2048-bit RSA keys (4096-bit recommended for production)
- Keep private keys secure and never expose
- Public keys can be shared for verification
- Easier key rotation (update public key distribution)

### Generating RSA Keys

```bash
# Generate private key (4096-bit)
openssl genrsa -out private_key.pem 4096

# Extract public key
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

## Error Handling

### Error Types

```go
var (
    ErrInvalidToken       = errors.New("invalid token")
    ErrTokenExpired       = errors.New("token has expired")
    ErrInvalidSignature   = errors.New("invalid token signature")
    ErrInvalidAlgorithm   = errors.New("invalid or unsupported algorithm")
    ErrAlgorithmMismatch  = errors.New("token algorithm does not match expected algorithm")
    ErrNoneAlgorithm      = errors.New("'none' algorithm is not allowed")
    ErrInvalidClaims      = errors.New("invalid token claims")
    ErrMissingToken       = errors.New("missing token")
    ErrInvalidTokenType   = errors.New("invalid token type")
    ErrInvalidKey         = errors.New("invalid signing key")
    ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
)
```

### Error Handling Pattern

```go
claims, err := jwtService.ValidateToken(tokenString)
if err != nil {
    // Check for specific errors
    if errors.Is(err, jwt.ErrTokenExpired) {
        // Prompt user to refresh token
        return http.StatusUnauthorized, "Token expired"
    }
    
    if errors.Is(err, jwt.ErrInvalidSignature) {
        // Possible tampering, log security event
        return http.StatusUnauthorized, "Invalid token"
    }
    
    // Generic error
    return http.StatusUnauthorized, "Authentication failed"
}
```

## Integration with Middleware

See the `middleware/` directory for HTTP and gRPC implementations.

### HTTP Middleware

```go
import "base_project/pkg/jwt/middleware"

// Create middleware
authMiddleware := middleware.NewHTTPAuthMiddleware(jwtService, logger)

// Apply to routes
http.Handle("/protected", authMiddleware.Authenticate(protectedHandler))
```

### gRPC Interceptor

```go
import "base_project/pkg/jwt/middleware"

// Create interceptor
authInterceptor := middleware.NewGRPCAuthInterceptor(jwtService, logger)

// Apply to server
server := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor.UnaryServerInterceptor()),
    grpc.StreamInterceptor(authInterceptor.StreamServerInterceptor()),
)
```

## Best Practices

### Token Expiration

**Access Tokens**:
- Keep short-lived (15 minutes to 1 hour)
- Reduces exposure if compromised
- Forces regular validation

**Refresh Tokens**:
- Longer lived (7-30 days)
- Store securely (httpOnly cookies or secure storage)
- Implement refresh token rotation

### Token Storage

**Client-Side**:
- Access tokens: Memory or sessionStorage
- Refresh tokens: httpOnly cookies (preferred) or secure storage
- Never store in localStorage (XSS vulnerability)

**Server-Side**:
- Store refresh tokens in database for revocation
- Maintain blacklist for revoked tokens
- Track token usage for security monitoring

### Token Rotation

```go
// Refresh flow
func handleRefresh(w http.ResponseWriter, r *http.Request) {
    refreshToken := getRefreshTokenFromCookie(r)
    
    // Validate refresh token
    claims, err := jwtService.ValidateToken(refreshToken)
    if err != nil {
        // Handle error
        return
    }
    
    // Verify it's a refresh token
    if claims.TokenType != claims.TokenTypeRefresh {
        // Reject
        return
    }
    
    // Generate new access token
    newAccessToken, err := jwtService.GenerateAccessToken(
        claims.UserID,
        claims.Email,
        claims.Role,
        claims.Extra,
    )
    
    // Optionally: Generate new refresh token (rotation)
    newRefreshToken, err := jwtService.GenerateRefreshToken(
        claims.UserID,
        claims.Extra,
    )
    
    // Return tokens
}
```

### Multi-Service Architecture

**Option 1: Shared Secret (HS256)**
- Simple setup
- All services use same secret key
- Key compromise affects all services
- Difficult key rotation

**Option 2: Asymmetric Keys (RS256)** ✅ Recommended
- Auth service holds private key (signs tokens)
- Other services hold public key (verify tokens)
- Compromise of one service doesn't expose signing key
- Easier key distribution and rotation

### Security Checklist

- [ ] Use strong secret keys (≥32 chars for HS256)
- [ ] Use adequate key sizes (≥2048 bits for RS256)
- [ ] Set appropriate expiration times
- [ ] Validate tokens on every request
- [ ] Never trust token contents without validation
- [ ] Implement token revocation mechanism
- [ ] Log authentication failures
- [ ] Monitor for suspicious patterns
- [ ] Rotate keys periodically
- [ ] Use HTTPS in production
- [ ] Implement rate limiting
- [ ] Add CORS properly

## Testing

Run tests:

```bash
go test ./pkg/jwt/...
```

The test suite includes:
- Configuration validation
- Token generation (HS256 and RS256)
- Token validation
- Algorithm validation and security
- Error cases
- Expiration handling

## Performance Considerations

### HS256 vs RS256

**HS256 (HMAC)**:
- ✅ Faster signing
- ✅ Faster verification
- ✅ Smaller tokens
- ❌ Shared secret

**RS256 (RSA)**:
- ❌ Slower signing (~10x)
- ❌ Slower verification (~3x)
- ❌ Larger tokens
- ✅ Public/private key separation
- ✅ Better for microservices

**Recommendation**: Use RS256 for microservices, HS256 for single-service applications.

### Caching

Consider caching validated tokens to reduce computation:

```go
// Example with in-memory cache
type CachedValidator struct {
    service jwt.Service
    cache   map[string]*claims.CustomClaims
    mu      sync.RWMutex
}

func (v *CachedValidator) ValidateToken(token string) (*claims.CustomClaims, error) {
    // Check cache first
    v.mu.RLock()
    if cached, ok := v.cache[token]; ok {
        v.mu.RUnlock()
        return cached, nil
    }
    v.mu.RUnlock()
    
    // Validate and cache
    claims, err := v.service.ValidateToken(token)
    if err != nil {
        return nil, err
    }
    
    v.mu.Lock()
    v.cache[token] = claims
    v.mu.Unlock()
    
    return claims, nil
}
```

**Note**: Be careful with caching - ensure cache invalidation on token expiration.

## Examples

See `examples/` directory for complete examples:
- Basic usage (HS256 and RS256)
- HTTP API with authentication
- gRPC service with authentication
- Token refresh flow
- Multi-service setup

## Troubleshooting

### "invalid token signature"

- Verify the secret key matches between generation and validation
- For RS256, ensure public key matches the private key used for signing
- Check that the token hasn't been modified

### "token has expired"

- Check system clocks are synchronized
- Adjust expiration times if too short
- Implement token refresh flow

### "algorithm mismatch"

- Ensure all services use the same algorithm
- Check configuration is correctly set
- Verify token wasn't generated by different service with different algorithm

### "invalid signing key"

- HS256: Secret must be at least 32 characters
- RS256: Verify PEM format and key type
- Check file permissions and paths

## Contributing

Follow the project's coding standards and include tests for any new functionality.

## License

Part of base_project microservices framework.