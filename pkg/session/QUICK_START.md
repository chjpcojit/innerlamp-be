# Session Management - Quick Start Guide

Fast implementation guide for secure session management with JWT rotation and device fingerprinting.

## 30-Second Setup

```go
import (
    "base_project/pkg/session"
    "base_project/pkg/jwt"
    "chjpcoj.com.innerlamp/pkg/utils" // envelope response chuẩn — xem pkg/utils/README.md
)

// 1. Create JWT service
jwtService, _ := jwt.New(&jwt.Config{
    SecretKey: "your-secret-key",
    AccessTokenTTL: 15 * time.Minute,
    RefreshTokenTTL: 7 * 24 * time.Hour,
})

// 2. Create session manager
sessionMgr, _ := session.NewMongoSessionManager(
    database,
    jwtService,
    session.DefaultConfig(),
)

// 3. Create session on login
newSession, _ := sessionMgr.CreateSession(
    ctx,
    userID,
    email,
    role,
    deviceInfo,
    ipAddress,
    userAgent,
)

// 4. Use tokens
accessToken := newSession.AccessToken
refreshToken := newSession.RefreshToken
```

## Common Scenarios

### Login Flow

```go
// User logs in
session, err := sessionMgr.CreateSession(
    ctx,
    user.ID,
    user.Email,
    user.Role,
    &session.DeviceInfo{
        Platform:    "iOS",
        OSVersion:   "17.0",
        DeviceModel: "iPhone 15 Pro",
        AppVersion:  "1.0.0",
    },
    r.RemoteAddr,
    r.UserAgent(),
)

// Return tokens to client
response := LoginResponse{
    AccessToken:  session.AccessToken,
    RefreshToken: session.RefreshToken,
    ExpiresIn:    900, // 15 minutes
}
```

### Validate Request

```go
// On each authenticated request
err := sessionMgr.ValidateSession(ctx, sessionID, accessToken)
if err == session.ErrSessionExpired {
    // Rotate tokens
    rotated, _ := sessionMgr.RotateTokens(ctx, sessionID)
    // Return new tokens
}
```

### Token Rotation

```go
// When access token expires
rotated, err := sessionMgr.RotateTokens(ctx, sessionID)
if err != nil {
    return err
}

// Return new tokens to client
response := TokenResponse{
    AccessToken:  rotated.AccessToken,
    RefreshToken: rotated.RefreshToken,
    ExpiresIn:    900,
}
```

### Logout

```go
// Single device logout
err := sessionMgr.RevokeSession(ctx, sessionID, "user logout")

// All devices logout
count, err := sessionMgr.RevokeAllUserSessions(ctx, userID, "user logout all")
```

### View Active Sessions

```go
// Get user's active sessions
sessions, err := sessionMgr.GetUserActiveSessions(ctx, userID)

for _, s := range sessions {
    fmt.Printf("%s - Last active: %v\n",
        s.DeviceInfo.DeviceName,
        s.LastActivityAt,
    )
}
```

## Configuration Options

```go
config := &session.Config{
    SessionTTL:         24 * time.Hour,      // How long session is valid
    AccessTokenTTL:     15 * time.Minute,    // Access token lifetime
    RefreshTokenTTL:    7 * 24 * time.Hour,  // Refresh token lifetime
    CleanupInterval:    1 * time.Hour,       // Cleanup frequency
    MaxSessionsPerUser: 5,                   // Max concurrent sessions
}
```

## Cleanup Expired Sessions (Cloud Run Compatible)

**⚠️ Important:** Do NOT use background goroutines in Cloud Run (stateless containers).

### Option 1: MongoDB TTL Index (Automatic - Recommended)

```go
// TTL index automatically created by NewMongoSessionManager
// MongoDB automatically deletes expired sessions
// No code needed - works when Cloud Run is scaled to zero
```

### Option 2: Cloud Scheduler + Endpoint

```go
// 1. Create cleanup endpoint
func CleanupHandler(sessionMgr session.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        count, _ := sessionMgr.CleanupExpiredSessions(r.Context())
        utils.WriteSuccessResponse(w, map[string]int{"cleaned": count})
    }
}

// 2. Setup Cloud Scheduler (runs hourly)
// gcloud scheduler jobs create http session-cleanup \
//   --schedule="0 * * * *" \
//   --uri="https://your-service.run.app/admin/cleanup" \
//   --http-method=POST
```

### ❌ Don't Use Background Jobs:

```go
// ❌ BAD in Cloud Run - goroutine lost when container stops
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        sessionMgr.CleanupExpiredSessions(ctx)
    }
}()
```

## Security Events

### Password Change

```go
// Revoke all sessions when password changes
count, err := sessionMgr.RevokeAllUserSessions(
    ctx,
    userID,
    "password changed",
)
```

### Suspicious Activity

```go
// Revoke session on suspicious activity
err := sessionMgr.RevokeSession(
    ctx,
    sessionID,
    "suspicious activity detected",
)
```

### Keep Current, Logout Others

```go
// User wants to logout from other devices
count, err := sessionMgr.RevokeOtherSessions(
    ctx,
    userID,
    currentSessionID,
    "logout other devices",
)
```

## Error Handling

```go
err := sessionMgr.ValidateSession(ctx, sessionID, accessToken)

switch err {
case session.ErrSessionExpired:
    // Rotate tokens or redirect to login
case session.ErrSessionRevoked:
    // Force re-login
case session.ErrSessionNotFound:
    // Redirect to login
case nil:
    // Session is valid
default:
    // Handle other errors
}
```

## HTTP Middleware Example

```go
func SessionMiddleware(sessionMgr session.Manager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r) // Extract from Authorization header
            sessionID := extractSessionID(r) // From JWT claims or header

            err := sessionMgr.ValidateSession(r.Context(), sessionID, token)
            if err != nil {
                utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## Device Fingerprinting

```go
// Client sends device info
deviceInfo := &session.DeviceInfo{
    Platform:     "iOS",           // iOS, Android, Web
    OSVersion:    "17.0",          // OS version
    AppVersion:   "1.0.0",         // App version
    DeviceModel:  "iPhone 15 Pro", // Device model
    DeviceName:   "John's iPhone", // User-assigned name
    ScreenSize:   "1179x2556",     // Screen resolution
    Timezone:     "America/Mexico_City",
    Language:     "es-MX",
}

// Fingerprint automatically computed as SHA-256 hash
// of: Platform|OSVersion|DeviceModel|ScreenSize|UserAgent
```

## Complete Example: Login Endpoint

```go
func LoginHandler(sessionMgr session.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Authenticate user
        user, err := authenticateUser(r)
        if err != nil {
            utils.WriteBusinessError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials", nil)
            return
        }

        // 2. Parse device info from request
        var req struct {
            DeviceInfo session.DeviceInfo `json:"device_info"`
        }
        json.NewDecoder(r.Body).Decode(&req)

        // 3. Create session
        newSession, err := sessionMgr.CreateSession(
            r.Context(),
            user.ID,
            user.Email,
            user.Role,
            &req.DeviceInfo,
            r.RemoteAddr,
            r.UserAgent(),
        )
        if err != nil {
            utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
            return
        }

        // 4. Return tokens
        response := map[string]interface{}{
            "access_token":  newSession.AccessToken,
            "refresh_token": newSession.RefreshToken,
            "expires_in":    900, // 15 minutes
            "session_id":    newSession.SessionID,
        }

        utils.WriteSuccessResponse(w, response)
    }
}
```

## MongoDB Collections

Sessions are stored in the `user_sessions` collection with automatic indexes:

```javascript
{
  "_id": ObjectId,
  "user_id": "user123",
  "session_id": "unique_session_id",
  "device_id": "device_fingerprint_hash",
  "device_info": {
    "platform": "iOS",
    "os_version": "17.0",
    "device_model": "iPhone 15 Pro",
    "app_version": "1.0.0"
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "token_version": 1,
  "is_active": true,
  "is_revoked": false,
  "last_activity_at": ISODate,
  "expires_at": ISODate,
  "created_at": ISODate,
  "updated_at": ISODate
}
```

## Performance Tips

1. **Use indexes**: Indexes are automatically created
2. **Cleanup regularly**: Run cleanup job hourly
3. **Limit sessions**: Set `MaxSessionsPerUser` appropriately
4. **TTL index**: MongoDB automatically deletes expired sessions
5. **Connection pooling**: Use MongoDB connection pool

## See Also

- [Full Documentation](README.md) - Complete session management guide
- [pkg/jwt](../jwt/README.md) - JWT token service
- [pkg/audit](../audit/README.md) - Audit logging integration
