# Session Management Package

Secure session management with JWT rotation, device fingerprinting, and TTL-based cleanup for MongoDB.

## Features

- **JWT Token Rotation** - Automatic access and refresh token rotation
- **Device Fingerprinting** - Unique device identification and tracking
- **Session TTL** - Automatic session expiration and cleanup
- **Multi-Device Support** - Manage concurrent sessions across devices
- **Session Revocation** - Revoke individual or all user sessions
- **Activity Tracking** - Track last activity for security monitoring
- **Session Limits** - Enforce maximum concurrent sessions per user

## Installation

```go
import "base_project/pkg/session"
```

## Quick Start

### Initialize Session Manager

```go
import (
    "base_project/pkg/session"
    "base_project/pkg/jwt"
)

// Create JWT service
jwtConfig := &jwt.Config{
    SecretKey:        "your-secret-key",
    AccessTokenTTL:   15 * time.Minute,
    RefreshTokenTTL:  7 * 24 * time.Hour,
    Issuer:           "movil-app",
}
jwtService, _ := jwt.New(jwtConfig)

// Create session manager
sessionConfig := session.DefaultConfig()
sessionConfig.MaxSessionsPerUser = 5

sessionMgr, _ := session.NewMongoSessionManager(
    database,
    jwtService,
    sessionConfig,
)
```

### Create New Session

```go
// Device information from client
deviceInfo := &session.DeviceInfo{
    Platform:     "iOS",
    OSVersion:    "17.0",
    AppVersion:   "1.0.0",
    DeviceModel:  "iPhone 15 Pro",
    DeviceName:   "My iPhone",
    ScreenSize:   "1179x2556",
    Timezone:     "America/Mexico_City",
    Language:     "es-MX",
}

// Create session
newSession, err := sessionMgr.CreateSession(
    ctx,
    "user123",                    // User ID
    "user@example.com",           // Email
    "customer",                   // Role
    deviceInfo,                   // Device info
    "192.168.1.1",               // IP address
    "Mozilla/5.0...",            // User agent
)

if err != nil {
    // Handle error
}

// Session contains access_token and refresh_token
fmt.Printf("Access Token: %s\n", newSession.AccessToken)
fmt.Printf("Refresh Token: %s\n", newSession.RefreshToken)
```

### Validate Session

```go
// On each request, validate session
err := sessionMgr.ValidateSession(
    ctx,
    sessionID,
    accessToken,
)

if err == session.ErrSessionExpired {
    // Rotate tokens
} else if err != nil {
    // Handle other errors
}
```

### Rotate Tokens

```go
// When access token expires
rotatedSession, err := sessionMgr.RotateTokens(ctx, sessionID)
if err != nil {
    // Handle error
}

// Use new tokens
newAccessToken := rotatedSession.AccessToken
newRefreshToken := rotatedSession.RefreshToken
```

### Revoke Session

```go
// Logout - revoke current session
err := sessionMgr.RevokeSession(ctx, sessionID, "user logout")

// Revoke all user sessions (password change, security breach)
count, err := sessionMgr.RevokeAllUserSessions(ctx, userID, "password changed")
fmt.Printf("Revoked %d sessions\n", count)

// Revoke other sessions (keep current active)
count, err := sessionMgr.RevokeOtherSessions(ctx, userID, currentSessionID, "logout other devices")
```

### Get User Sessions

```go
// Get all active sessions for a user
sessions, err := sessionMgr.GetUserActiveSessions(ctx, userID)

for _, sess := range sessions {
    fmt.Printf("Device: %s, Last Activity: %v\n",
        sess.DeviceInfo.DeviceName,
        sess.LastActivityAt,
    )
}
```

### Cleanup Expired Sessions

```go
// Periodic cleanup (run as background job)
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        count, err := sessionMgr.CleanupExpiredSessions(context.Background())
        if err != nil {
            log.Printf("Cleanup failed: %v", err)
        } else {
            log.Printf("Cleaned up %d expired sessions", count)
        }
    }
}()
```

## Configuration

### Default Configuration

```go
config := session.DefaultConfig()
// Returns:
// SessionTTL:         24 hours
// AccessTokenTTL:     15 minutes
// RefreshTokenTTL:    7 days
// MaxSessionsPerUser: 5
// Note: MongoDB TTL index handles cleanup automatically - no CleanupInterval needed
```

### Custom Configuration

```go
config := &session.Config{
    SessionTTL:         48 * time.Hour,      // Longer session
    AccessTokenTTL:     30 * time.Minute,    // Longer access token
    RefreshTokenTTL:    14 * 24 * time.Hour, // 2 weeks refresh
    MaxSessionsPerUser: 10,                  // More concurrent sessions
}
// Note: MongoDB TTL index handles cleanup automatically - no CleanupInterval needed
```

## Device Fingerprinting

### How It Works

Device fingerprints are generated using:

- Platform (iOS, Android, Web)
- OS Version
- Device Model
- Screen Size
- User Agent

The fingerprint is a SHA-256 hash of these combined values.

### Example

```go
deviceInfo := &session.DeviceInfo{
    Platform:     "iOS",
    OSVersion:    "17.0",
    DeviceModel:  "iPhone 15 Pro",
    ScreenSize:   "1179x2556",
}
userAgent := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"

// Fingerprint automatically generated:
// SHA256("iOS|17.0|iPhone 15 Pro|1179x2556|Mozilla/5.0...")
```

### Use Cases

1. **Detect device changes** - Same user, different device
2. **Security alerts** - New device login notifications
3. **Session binding** - Tie session to specific device
4. **Analytics** - Track device usage patterns

## Session Lifecycle

### 1. Creation

```
User Login
    ↓
Create Session
    ↓
Generate Tokens (Access + Refresh)
    ↓
Store in MongoDB
    ↓
Return Tokens to Client
```

### 2. Validation

```
Client Request
    ↓
Extract Access Token
    ↓
Validate Session
    ↓
Update Last Activity
    ↓
Process Request
```

### 3. Rotation

```
Access Token Expired
    ↓
Client Sends Refresh Token
    ↓
Validate Refresh Token
    ↓
Generate New Tokens
    ↓
Increment Token Version
    ↓
Return New Tokens
```

### 4. Revocation

```
Logout / Security Event
    ↓
Revoke Session
    ↓
Mark as Revoked
    ↓
Set Revoked Timestamp
    ↓
Invalidate Tokens
```

### 5. Cleanup

```
MongoDB TTL Index (Automatic)
    ↓
Monitor expires_at field
    ↓
Auto-delete expired documents

Optional: Cloud Scheduler + HTTP Endpoint
    ↓
CleanupExpiredSessions()
    ↓
Manual cleanup trigger
```

## Security Best Practices

### 1. Token Storage

**Client-Side:**

```javascript
// DO: Store in secure storage (iOS Keychain, Android KeyStore)
await SecureStore.setItemAsync("access_token", token);

// DON'T: Store in AsyncStorage or localStorage
```

### 2. Token Transmission

```go
// Always use HTTPS
// Never log tokens
// Use Authorization header
Authorization: Bearer <access_token>
```

### 3. Session Revocation Scenarios

```go
// Password changed
sessionMgr.RevokeAllUserSessions(ctx, userID, "password changed")

// Suspicious activity detected
sessionMgr.RevokeSession(ctx, sessionID, "suspicious activity")

// User requests logout from all devices
sessionMgr.RevokeAllUserSessions(ctx, userID, "user logout all")

// Keep current, revoke others
sessionMgr.RevokeOtherSessions(ctx, userID, currentSessionID, "logout other devices")
```

### 4. Session Limits

```go
// Enforce maximum concurrent sessions
config := &session.Config{
    MaxSessionsPerUser: 3, // Max 3 devices
}

// When limit reached, oldest session is automatically revoked
```

## MongoDB Indexes

The package automatically creates these indexes:

```javascript
// Unique session ID
db.user_sessions.createIndex({ session_id: 1 }, { unique: true });

// User lookup
db.user_sessions.createIndex({ user_id: 1 });

// Refresh token lookup
db.user_sessions.createIndex({ refresh_token: 1 });

// Device tracking
db.user_sessions.createIndex({ device_id: 1 });

// TTL index for automatic cleanup
db.user_sessions.createIndex({ expires_at: 1 }, { expireAfterSeconds: 0 });

// Active user sessions
db.user_sessions.createIndex({ user_id: 1, is_active: 1 });

// User-device combo
db.user_sessions.createIndex({ user_id: 1, device_id: 1 });
```

## Error Handling

```go
switch err {
case session.ErrSessionNotFound:
    // Session doesn't exist - redirect to login
case session.ErrSessionExpired:
    // Session expired - rotate tokens or re-login
case session.ErrSessionRevoked:
    // Session revoked - force re-login
case session.ErrInvalidDeviceID:
    // Device mismatch - security alert
case session.ErrInvalidRefreshToken:
    // Invalid refresh token - force re-login
default:
    // Other errors
}
```

## Integration Examples

### HTTP Middleware

```go
func SessionMiddleware(sessionMgr session.Manager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract token from header
            authHeader := r.Header.Get("Authorization")
            token := strings.TrimPrefix(authHeader, "Bearer ")

            // Get session ID from context (set by JWT middleware)
            sessionID := r.Context().Value("session_id").(string)

            // Validate session
            err := sessionMgr.ValidateSession(r.Context(), sessionID, token)
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### gRPC Interceptor

```go
func SessionUnaryInterceptor(sessionMgr session.Manager) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // Extract metadata
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }

        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing token")
        }

        token := strings.TrimPrefix(tokens[0], "Bearer ")
        sessionID := md.Get("session-id")[0]

        // Validate session
        err := sessionMgr.ValidateSession(ctx, sessionID, token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid session")
        }

        return handler(ctx, req)
    }
}
```

## Performance Considerations

- **Index Optimization**: All common queries use indexes
- **TTL Index**: MongoDB automatically deletes expired sessions
- **MongoDB TTL Index**: Automatic cleanup - no application overhead
- **Connection Pooling**: Use MongoDB connection pooling
- **Token Caching**: Consider Redis for high-traffic scenarios
- **Cloud Run**: Stateless design - no background jobs needed

## Testing

Run tests:

```bash
go test ./pkg/session/... -v -cover
```

## Related Packages

- [pkg/jwt](../jwt/README.md) - JWT token generation and validation
- [pkg/audit](../audit/README.md) - Audit logging for session events
- [pkg/logging](../logging/README.md) - Structured logging with correlation IDs

## License

Part of base_project microservices architecture.
