# Session Management on Google Cloud Run

This guide explains how to properly use the session package in Cloud Run environments (stateless, auto-scaling containers).

## ⚠️ Critical: Cloud Run Characteristics

**Cloud Run containers are:**

- ✅ **Stateless** - No persistent state between requests
- ✅ **Auto-scaling** - Scale to zero when no traffic
- ✅ **Pay-per-request** - Only pay when serving requests
- ❌ **NOT suitable** for long-running background jobs
- ❌ **NO guaranteed uptime** - Containers can terminate at any time

## ❌ What NOT to Do

### Don't Use Background Goroutines

```go
// ❌ BAD - This goroutine will be lost when Cloud Run scales to zero
func main() {
    sessionMgr, _ := session.NewMongoSessionManager(db, jwtService, config)

    // ❌ This will NOT work reliably in Cloud Run
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()

        for range ticker.C {
            sessionMgr.CleanupExpiredSessions(context.Background())
        }
    }()

    // Start HTTP server
    http.ListenAndServe(":8080", router)
}
```

**Why this fails:**

1. Container scales to zero → goroutine stops
2. Container restarts → goroutine state lost
3. No guaranteed execution schedule
4. Wastes resources when container is running

## ✅ Recommended Approaches

### Option 1: MongoDB TTL Index (Automatic - Best)

**The session package automatically creates a TTL index** that handles cleanup without any application code.

```go
// ✅ GOOD - MongoDB handles cleanup automatically
sessionMgr, err := session.NewMongoSessionManager(db, jwtService, config)
// TTL index created automatically on collection initialization
// MongoDB deletes expired documents based on expires_at field
// Works 24/7 regardless of Cloud Run state
```

**How it works:**

1. TTL index created on `expires_at` field during initialization
2. MongoDB background thread monitors expired documents
3. Automatic deletion happens every 60 seconds (MongoDB default)
4. **No application code needed**
5. **Works even when Cloud Run is scaled to zero**

**TTL Index Details:**

```javascript
// Automatically created index
{
  "expires_at": 1
}
// With options:
{
  expireAfterSeconds: 0  // Delete immediately when expires_at < now()
}
```

### Option 2: Cloud Scheduler + HTTP Endpoint (Manual Control)

If you need explicit control over cleanup timing or want to run custom cleanup logic:

#### Step 1: Create Cleanup Endpoint

```go
// In delivery/http/session_handler.go
package http

import (
    "net/http"

    "base_project/pkg/logging"
    "base_project/pkg/session"
    "chjpcoj.com.innerlamp/pkg/utils" // envelope response chuẩn — xem pkg/utils/README.md
)

type SessionHandler struct {
    sessionMgr session.Manager
    logger     logging.Logger
}

func NewSessionHandler(sessionMgr session.Manager, logger logging.Logger) *SessionHandler {
    return &SessionHandler{
        sessionMgr: sessionMgr,
        logger:     logger,
    }
}

// CleanupHandler handles periodic session cleanup triggered by Cloud Scheduler
func (h *SessionHandler) CleanupHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Verify request is from Cloud Scheduler
    if r.Header.Get("X-CloudScheduler") != "true" {
        h.logger.Warn("Unauthorized cleanup attempt")
        utils.WriteBusinessError(w, http.StatusForbidden, "FORBIDDEN", "forbidden", nil)
        return
    }

    // 2. Run cleanup
    count, err := h.sessionMgr.CleanupExpiredSessions(r.Context())
    if err != nil {
        h.logger.WithError(err).Error("Session cleanup failed")
        utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil) // không trả err.Error() cho client
        return
    }

    // 3. Log and return result
    h.logger.WithFields(logging.Fields{
        "sessions_cleaned": count,
    }).Info("Session cleanup completed")

    utils.WriteSuccessResponse(w, map[string]interface{}{
        "cleaned": count,
    })
}
```

#### Step 2: Register Route

```go
// In delivery/http/router.go
func SetupRoutes(
    router *chi.Mux,
    sessionHandler *SessionHandler,
) {
    // Admin endpoints
    router.Route("/admin", func(r chi.Router) {
        // Cloud Scheduler endpoint
        r.Post("/cleanup-sessions", sessionHandler.CleanupHandler)
    })
}
```

#### Step 3: Create Cloud Scheduler Job

```bash
# Create hourly cleanup job
gcloud scheduler jobs create http session-cleanup \
  --schedule="0 * * * *" \
  --uri="https://your-service.run.app/admin/cleanup-sessions" \
  --http-method=POST \
  --headers="X-CloudScheduler=true" \
  --location=us-central1 \
  --description="Cleanup expired sessions hourly"

# For daily cleanup (midnight UTC)
gcloud scheduler jobs create http session-cleanup-daily \
  --schedule="0 0 * * *" \
  --uri="https://your-service.run.app/admin/cleanup-sessions" \
  --http-method=POST \
  --headers="X-CloudScheduler=true" \
  --location=us-central1

# Test the job
gcloud scheduler jobs run session-cleanup --location=us-central1
```

#### Step 4: Security Enhancement (Optional)

Add authentication to prevent unauthorized cleanup calls:

```go
import (
    "crypto/subtle"
    "os"
)

func (h *SessionHandler) CleanupHandler(w http.ResponseWriter, r *http.Request) {
    // Option 1: Verify Cloud Scheduler header
    if r.Header.Get("X-CloudScheduler") != "true" {
        utils.WriteBusinessError(w, http.StatusForbidden, "FORBIDDEN", "forbidden", nil)
        return
    }

    // Option 2: Use secret token
    token := r.Header.Get("X-Cleanup-Token")
    expected := os.Getenv("CLEANUP_SECRET_TOKEN")
    if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
        utils.WriteBusinessError(w, http.StatusForbidden, "FORBIDDEN", "forbidden", nil)
        return
    }

    // ... cleanup logic
}
```

## Comparison: MongoDB TTL vs Cloud Scheduler

| Feature                       | MongoDB TTL Index         | Cloud Scheduler + Endpoint       |
| ----------------------------- | ------------------------- | -------------------------------- |
| **Setup Complexity**          | ✅ Automatic (0 code)     | ⚠️ Manual (endpoint + scheduler) |
| **Reliability**               | ✅ Built-in MongoDB       | ✅ Managed GCP service           |
| **Works when scaled to zero** | ✅ Yes                    | ✅ Yes                           |
| **Cost**                      | ✅ Free (MongoDB feature) | ⚠️ Minimal (scheduler calls)     |
| **Cleanup frequency**         | ⚠️ Fixed (60 seconds)     | ✅ Customizable (cron)           |
| **Custom logic**              | ❌ No                     | ✅ Yes                           |
| **Monitoring**                | ⚠️ MongoDB logs only      | ✅ Application logs + metrics    |

**Recommendation:**

- **Use MongoDB TTL** for standard cleanup (simplest, most reliable)
- **Add Cloud Scheduler** only if you need:
  - Custom cleanup logic (e.g., send notifications)
  - Specific scheduling requirements
  - Application-level monitoring and metrics

## Implementation Checklist

### Using MongoDB TTL (Recommended)

- [x] Initialize session manager with `NewMongoSessionManager()`
- [x] TTL index automatically created
- [x] No additional code needed
- [x] Monitor MongoDB logs for cleanup activity

### Using Cloud Scheduler (Optional)

- [ ] Create cleanup endpoint handler
- [ ] Register route in router
- [ ] Deploy service to Cloud Run
- [ ] Create Cloud Scheduler job
- [ ] Add security headers/tokens
- [ ] Test scheduler job manually
- [ ] Monitor application logs

## Testing

### Test MongoDB TTL Index

```go
// Verify TTL index exists
indexes := collection.Indexes()
cursor, _ := indexes.List(context.Background())
var results []bson.M
cursor.All(context.Background(), &results)

// Should see index with expireAfterSeconds: 0
for _, idx := range results {
    if idx["name"] == "expires_at_1" {
        fmt.Println("TTL Index found:", idx)
    }
}
```

### Test Cloud Scheduler Endpoint

```bash
# Local testing
curl -X POST http://localhost:8080/admin/cleanup-sessions \
  -H "X-CloudScheduler: true"

# Cloud Run testing
curl -X POST https://your-service.run.app/admin/cleanup-sessions \
  -H "X-CloudScheduler: true"

# Test scheduler job
gcloud scheduler jobs run session-cleanup --location=us-central1
```

## Monitoring

### MongoDB TTL Monitoring

```bash
# Check MongoDB logs for TTL activity
# MongoDB logs show TTL thread activity every 60 seconds
```

### Cloud Scheduler Monitoring

```go
// In CleanupHandler
func (h *SessionHandler) CleanupHandler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()

    count, err := h.sessionMgr.CleanupExpiredSessions(r.Context())
    duration := time.Since(start)

    h.logger.WithFields(logging.Fields{
        "sessions_cleaned": count,
        "duration_ms":      duration.Milliseconds(),
        "status":           "success",
    }).Info("Session cleanup completed")

    // Optional: Send metrics to Cloud Monitoring
    // metricsClient.RecordCleanup(count, duration)
}
```

## Common Questions

### Q: Will MongoDB TTL work if my Cloud Run service is scaled to zero?

**A:** Yes! MongoDB TTL is a database feature, not an application feature. It runs independently of your Cloud Run containers.

### Q: How often does MongoDB TTL cleanup run?

**A:** MongoDB TTL thread runs approximately every 60 seconds. Expired documents are deleted shortly after expiration.

### Q: Can I customize TTL cleanup frequency?

**A:** No, MongoDB TTL frequency is fixed. Use Cloud Scheduler if you need custom timing.

### Q: What happens if cleanup fails?

**A:** MongoDB TTL retries automatically. For Cloud Scheduler, set up retry policies and error notifications.

### Q: Do I need both TTL and Cloud Scheduler?

**A:** No. Use MongoDB TTL for automatic cleanup. Add Cloud Scheduler only if you need custom logic or monitoring.

## Migration Guide

If you have existing code with background goroutines:

### Before (Incorrect for Cloud Run)

```go
// ❌ Remove this code
go func() {
    ticker := time.NewTicker(config.CleanupInterval)
    defer ticker.Stop()
    for range ticker.C {
        sessionMgr.CleanupExpiredSessions(context.Background())
    }
}()
```

### After (Cloud Run Compatible)

```go
// ✅ Option 1: Just rely on MongoDB TTL (no code needed)
sessionMgr, _ := session.NewMongoSessionManager(db, jwtService, config)

// ✅ Option 2: Add Cloud Scheduler endpoint (optional)
router.Post("/admin/cleanup-sessions", sessionHandler.CleanupHandler)
```

## Best Practices

1. **Prefer MongoDB TTL** - Simplest and most reliable for Cloud Run
2. **Avoid background jobs** - Cloud Run is request-driven, not job-driven
3. **Use Cloud Scheduler** - For any scheduled tasks that need application logic
4. **Monitor TTL index** - Ensure it's created and working correctly
5. **Test cleanup** - Verify expired sessions are actually deleted
6. **Secure endpoints** - Always verify Cloud Scheduler requests
7. **Log cleanup activity** - For debugging and monitoring

## Related Resources

- [Google Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)
- [MongoDB TTL Indexes](https://docs.mongodb.com/manual/core/index-ttl/)
- [Session Package README](./README.md)
- [Session Quick Start](./QUICK_START.md)
