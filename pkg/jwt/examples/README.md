# JWT Examples

This directory contains comprehensive examples demonstrating how to use the JWT package for authentication in HTTP and gRPC services.

## Overview

The examples demonstrate:
- User login and token generation
- Token validation and refresh
- Protected endpoints with JWT authentication
- Role-based access control (RBAC)
- Optional authentication
- Custom token extractors
- Integration with logging and correlation IDs

## Examples

### 1. HTTP Example (`http/main.go`)

A complete HTTP API server with JWT authentication.

#### Features

- **Login endpoint**: Authenticates users and issues JWT tokens
- **Token refresh**: Generates new access tokens from refresh tokens
- **Protected endpoints**: Requires valid JWT tokens
- **Role-based access**: Admin-only endpoints
- **Optional authentication**: Enhanced content for authenticated users
- **Health check**: Public endpoint without authentication

#### Running the HTTP Example

```bash
# From project root
go run pkg/jwt/examples/http/main.go

# Or from examples directory
cd pkg/jwt/examples/http
go run main.go
```

The server will start on `http://localhost:8080`

#### Testing the HTTP Example

**1. Login**
```bash
curl -X POST http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "user": {
      "id": "user-1",
      "email": "admin@example.com",
      "role": "admin"
    }
  }
}
```

Every reply uses the `pkg/utils` envelope (see `pkg/utils/README.md`). A failure looks like:

```json
{
  "success": false,
  "error": { "code": "UNAUTHORIZED", "message": "Invalid credentials", "type": "business" }
}
```

**2. Access Protected Endpoint**
```bash
TOKEN="<your_access_token>"

curl http://localhost:8080/api/profile \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "success": true,
  "data": {
    "user_id": "user-1",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

**3. List Users (Authenticated)**
```bash
curl http://localhost:8080/api/users \
  -H "Authorization: Bearer $TOKEN"
```

**4. Admin-Only Endpoint**
```bash
curl http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer $TOKEN"
```

**5. Refresh Token**
```bash
REFRESH_TOKEN="<your_refresh_token>"

curl -X POST http://localhost:8080/api/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
```

Response:
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGc..."
  }
}
```

**6. Optional Authentication**
```bash
# Without token (public content)
curl http://localhost:8080/api/content

# With token (personalized content)
curl http://localhost:8080/api/content \
  -H "Authorization: Bearer $TOKEN"
```

**7. Health Check (No Auth Required)**
```bash
curl http://localhost:8080/api/health
```

#### Available Users

```
Email: admin@example.com
Password: admin123
Role: admin

Email: user@example.com
Password: user123
Role: user
```

### 2. gRPC Example (`grpc/main.go`)

A complete gRPC service with JWT authentication interceptors.

#### Features

- **Unary interceptor**: Authenticates unary RPC calls
- **Stream interceptor**: Authenticates streaming RPC calls
- **Role-based interceptors**: Restrict access by user role
- **Optional authentication**: Enhanced features for authenticated users
- **Skip methods**: Public methods without authentication
- **Custom metadata extraction**: Flexible token extraction
- **Token refresh flow**: Complete token lifecycle management

#### Running the gRPC Example

```bash
# From project root
go run pkg/jwt/examples/grpc/main.go

# Or from examples directory
cd pkg/jwt/examples/grpc
go run main.go
```

The server will start on `localhost:9090`

#### Testing the gRPC Example

The example includes a built-in client that demonstrates various scenarios:

1. **Authenticated call**: Valid token in metadata
2. **Unauthenticated call**: No token (should fail)
3. **Invalid token**: Malformed token (should fail)
4. **Admin-only call**: Requires admin role
5. **Non-admin call**: User role accessing admin endpoint (should fail)
6. **Token refresh**: Complete refresh flow
7. **Custom metadata**: Using custom header names

**Using grpcurl:**

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Generate a token first (you'll need to call your auth service)
TOKEN="<your_access_token>"

# Call GetProfile
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  localhost:9090 \
  UserService/GetProfile

# Call ListUsers
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  localhost:9090 \
  UserService/ListUsers

# Call AdminAction (admin only)
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"action": "delete_user"}' \
  localhost:9090 \
  UserService/AdminAction
```

**Using custom metadata key:**

```bash
grpcurl -plaintext \
  -H "x-api-token: $TOKEN" \
  localhost:9090 \
  UserService/GetProfile
```

## Common Patterns

### 1. Login Flow

```
Client                    Server
  |                         |
  |-- POST /api/login ----> |
  |    (email, password)    |
  |                         |
  | <--- access_token ----- |
  |      refresh_token      |
  |                         |
```

### 2. Authenticated Request Flow

```
Client                    Server
  |                         |
  |-- GET /api/profile ---> |
  | Authorization: Bearer   |
  |    <access_token>       |
  |                         |
  |    [Middleware]         |
  |    - Extract token      |
  |    - Validate token     |
  |    - Add claims to ctx  |
  |                         |
  |    [Handler]            |
  |    - Get claims from ctx|
  |    - Process request    |
  |                         |
  | <--- user profile ----- |
  |                         |
```

### 3. Token Refresh Flow

```
Client                    Server
  |                         |
  |-- POST /api/refresh --> |
  |    refresh_token        |
  |                         |
  |    [Validate refresh]   |
  |    - Check token type   |
  |    - Verify not expired |
  |                         |
  |    [Generate new]       |
  |    - New access token   |
  |    - (Optional) New     |
  |      refresh token      |
  |                         |
  | <--- new_access_token - |
  |                         |
```

### 4. Role-Based Access Control

```
Client                    Server
  |                         |
  |-- GET /api/admin ----> |
  | Authorization: Bearer   |
  |                         |
  |    [Auth Middleware]    |
  |    - Validate token     |
  |    - Add claims to ctx  |
  |                         |
  |    [Role Middleware]    |
  |    - Check claims.Role  |
  |    - Require "admin"    |
  |                         |
  |    [Handler]            |
  |    - Admin logic        |
  |                         |
  | <--- admin data ------- |
  |                         |
```

## Integration with Your Service

### HTTP Service

```go
import (
    "base_project/pkg/jwt"
    "base_project/pkg/jwt/middleware"
    "base_project/pkg/logging"
)

// Create JWT service
jwtService, _ := jwt.New(&jwt.Config{
    Algorithm:              jwt.AlgorithmHS256,
    SecretKey:              os.Getenv("JWT_SECRET"),
    AccessTokenExpiration:  15 * time.Minute,
    RefreshTokenExpiration: 7 * 24 * time.Hour,
    Issuer:                 "my-service",
    Audience:               "my-users",
})

// Create logger
logger := logging.New(logging.Config{
    Level: logging.InfoLevel,
})

// Create middleware
authMiddleware := middleware.NewHTTPAuthMiddleware(jwtService, logger)

// Apply to routes
mux := http.NewServeMux()
mux.Handle("/api/protected", authMiddleware.Authenticate(
    http.HandlerFunc(protectedHandler),
))
```

### gRPC Service

```go
import (
    "base_project/pkg/jwt"
    "base_project/pkg/jwt/middleware"
    "google.golang.org/grpc"
)

// Create JWT service and logger (same as HTTP)

// Create interceptor
authInterceptor := middleware.NewGRPCAuthInterceptor(jwtService, logger)

// Create server with interceptors
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        authInterceptor.UnaryServerInterceptor(),
    ),
    grpc.ChainStreamInterceptor(
        authInterceptor.StreamServerInterceptor(),
    ),
)

// Register your services
pb.RegisterYourServiceServer(server, yourServiceImpl)
```

## Security Best Practices

### Token Storage (Client-Side)

**DO:**
- Store access tokens in memory or sessionStorage
- Store refresh tokens in httpOnly cookies
- Use secure cookies (HTTPS only)
- Set appropriate cookie expiration

**DON'T:**
- Store tokens in localStorage (XSS vulnerability)
- Log tokens in console or files
- Send tokens in URL parameters
- Store tokens in unencrypted form

### Token Expiration

```go
// Recommended settings
config := &jwt.Config{
    AccessTokenExpiration:  15 * time.Minute,  // Short-lived
    RefreshTokenExpiration: 7 * 24 * time.Hour, // Longer-lived
}
```

### Token Rotation

```go
// Generate new refresh token on each refresh
newAccessToken, _ := jwtService.GenerateAccessToken(...)
newRefreshToken, _ := jwtService.GenerateRefreshToken(...)

// Invalidate old refresh token in database
```

### Algorithm Selection

**HS256 (Symmetric):**
- ✅ Simple, fast
- ✅ Good for single-service applications
- ❌ Shared secret across services

**RS256 (Asymmetric):**
- ✅ Auth service signs with private key
- ✅ Other services verify with public key
- ✅ Better for microservices
- ❌ Slower performance

### Error Handling

```go
claims, err := jwtService.ValidateToken(token)
if err != nil {
    switch {
    case errors.Is(err, jwt.ErrTokenExpired):
        // Prompt client to refresh
    case errors.Is(err, jwt.ErrInvalidSignature):
        // Log security event
    default:
        // Generic error
    }
}
```

## Troubleshooting

### "missing authentication token"

- Check Authorization header is present
- Verify Bearer prefix: `Bearer <token>`
- Ensure token is not empty

### "token has expired"

- Implement token refresh flow
- Check system clock synchronization
- Verify token expiration times

### "invalid token signature"

- Ensure secret key matches between services
- For RS256, verify public key matches private key
- Check token wasn't modified

### "insufficient permissions"

- Verify user role in token claims
- Check role requirements in middleware
- Ensure role is set during token generation

## Additional Resources

- [JWT Package README](../README.md)
- [JWT Quick Start Guide](../QUICK_START.md)
- [Middleware Documentation](../middleware/)
- [JWT Specification (RFC 7519)](https://tools.ietf.org/html/rfc7519)

## Directory Structure

```
examples/
├── http/
│   └── main.go          # Complete HTTP API example
├── grpc/
│   └── main.go          # Complete gRPC service example
└── README.md            # This file
```

## Building Examples

```bash
# Build HTTP example
go build -o bin/http-example pkg/jwt/examples/http/main.go

# Build gRPC example
go build -o bin/grpc-example pkg/jwt/examples/grpc/main.go

# Run built binaries
./bin/http-example
./bin/grpc-example
```

## Contributing

When adding new examples:
1. Create a new directory under `examples/`
2. Name the main file `main.go`
3. Follow existing code structure
4. Include comprehensive comments
5. Add usage instructions to this README
6. Test all scenarios
7. Handle errors appropriately