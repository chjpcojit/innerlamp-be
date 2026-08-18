---
type: Reference
title: Rate Limiting Stores & Policies
description: Dynamic request throttling via Token Bucket, Sliding Window, Fixed Window, and Leaky Bucket using Redis or In-Memory backends.
tags: [rate-limiting, sliding-window, token-bucket, redis, store, middleware]
openwiki:
  roles: [domain, integration]
  change_kinds: [lifecycle, public-api]
  source_paths: [pkg/ratelimit/ratelimit.go, pkg/ratelimit/store/memory.go, pkg/ratelimit/store/redis.go, pkg/ratelimit/middleware/http.go]
  symbols: [Algorithm, Config, Store, Result, MemoryStore, RedisStore, HTTPMiddleware]
  test_paths: [pkg/ratelimit/store/memory_test.go, pkg/ratelimit/middleware/http_test.go]
  invariants: [Limiter fails open on storage backend errors, Blocked HTTP calls return 429 with Retry-After.]
  validation_commands: [go test -v ./pkg/ratelimit/...]
---

# Rate Limiting Stores & Policies

The `pkg/ratelimit` package provides high-performance request throttling boundaries. It is designed to run either as localized in-memory buckets or as global distributed limiters synced over Redis.

---

## 🛠️ Supported Throttling Algorithms

The rate-limiting model supports four core algorithms configured via the `Algorithm` string:
1. **Token Bucket** (`token_bucket`): Allows traffic bursts up to a configured `Burst` size, continuously refilling tokens at a constant rate over `Window` duration.
2. **Sliding Window Log** (`sliding_window`): Tracks precise timestamp logs for requests, offering smooth throttling without edge spikes.
3. **Fixed Window Counter** (`fixed_window`): Aggregates hits inside atomic window blocks (e.g., 1-minute frames), resetting totals at absolute time boundaries.
4. **Leaky Bucket** (`leaky_bucket`): Enforces a rigid, queue-like constant egress output rate, smoothing out burst patterns completely.

---

## 💽 Backend Storage Stores

Rate limiters bind to a standard `Store` interface that handles state checking and atomicity:

```go
type Store interface {
    Allow(ctx context.Context, key string, limit int64, window time.Duration, burst int64, algo Algorithm) (*Result, error)
}
```

### 1. Local In-Memory Store (`MemoryStore`)
* **File Source**: `/pkg/ratelimit/store/memory.go`
* **Mechanism**: Utilizes a local map of specific algorithm structures protected by a thread-safe `sync.RWMutex`.
* **Usecase**: Perfect for isolated task-runners, daemon workers, or single-instance deployments where cross-node sync is unnecessary.

### 2. Distributed Redis Store (`RedisStore`)
* **File Source**: `/pkg/ratelimit/store/redis.go`
* **Mechanism**: Implements algorithms over `github.com/redis/go-redis/v9` using multi-command transactions or optimized Lua scripts to ensure absolute evaluation atomicity.
* **Usecase**: Required for distributed serverless clusters (Cloud Run, GKE) where multiple instances share a common egress load balancer.

---

## 📥 Ingress Integration Middleware (`HTTPMiddleware`)

The framework provides an `HTTPMiddleware` wrapper that validates requests before routing them to the internal business logic.

### 🛡️ Fail-Open Operational Invariant
To ensure rate limiting issues do not cause system-wide blackouts:
* **Invariance**: If the rate limit storage backend (e.g., Redis cluster) throws a connection or execution error, **the limiter fails open**. It logs the incident and immediately forwards the request to the downstream handler rather than returning a 500 error to the consumer.

### 🏷️ REST Header Metadata Standards
For every request checked, the middleware injects standard rate metadata headers into the HTTP response:
* `X-RateLimit-Limit`: Maximum requests configured for the matching boundary.
* `X-RateLimit-Remaining`: Count of remaining requests allowed inside the active window.
* `X-RateLimit-Reset`: Unix timestamp when the active window resets.

If a request exceeds the limit, the limiter interrupts the chain:
* Returns **`429 Too Many Requests`**.
* Appends **`Retry-After`** (seconds remaining before reset).

---

## 🔗 Semantic Relationships (OKF Mapped)

* **Configuration Management**: Limit definitions and rate values [is configured through](config-secrets.md) `pkg/config` models, allowing operations teams to easily tweak limits.
* **Client Wrapping**: Outbound HTTP integration callers documented in [Resilient Networking & Middlewares](networking.md) can wrap their connections inside rate limit handlers to protect external APIs from quota exhaustion.

---

## 🧪 Focused Validation

Throttling mechanics are verified under `/pkg/ratelimit/store/memory_test.go` and `/pkg/ratelimit/middleware/http_test.go`.

To validate algorithm curves and middleware header injections:
```bash
# Test localized limiters and algorithms
go test -v ./pkg/ratelimit/store/...

# Test HTTP limit response headers
go test -v ./pkg/ratelimit/middleware/...
```
