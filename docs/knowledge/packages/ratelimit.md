---
type: Package
title: pkg/ratelimit
description: Request rate limiting (sliding window and three other algorithms) on an in-memory or Redis store, and where appService applies each limit, with what key, budget and store.
resource: pkg/ratelimit/
tags: [pkg, ratelimit, redis, security]
status: draft
generated: { by: claude-code, at: 2026-10-14T19:40:00Z }
sources:
  - path: pkg/ratelimit/README.md
    kind: implementation-notes
  - path: services/appService/cmd/ratelimit_store.go
    kind: implementation
  - path: services/appService/config/config.dev.yaml
    kind: config
  - path: docs/assumptions/ratelimit/A-001-redis-error-falls-back-to-memory.md
    kind: assumption
---

# Summary

`pkg/ratelimit` answers one question per request: may this key (a client IP, an account id) make
one more call inside the current window? It is a leaf package with two stores instead of a
factory registry: `store.MemoryStore` (counters in the process) and `store.RedisStore` (counters
in Redis, one Lua script per check so concurrent instances cannot race). `pkg/ratelimit/middleware`
wraps a store as `net/http` middleware or gRPC interceptors and writes the `429 RATE_LIMITED`
envelope.

appService uses it through two wrappers in `delivery/http` — `ClientIPRateLimit` (key = client IP)
and `UserRateLimit` (key = the access token's `user_id`) — both on the sliding-window algorithm.
Since 2026-10-14 every limiter shares one store chosen by `rate_limit_store` in the config: Redis
from `docker-compose.yml` in dev, so the counts are the same on every instance. Each limiter owns a
key scope inside that store (`<key_prefix><scope>:<ip or user id>`), so two limiters keyed by the
same IP never count into one bucket.

This is a request-rate brake only. The business lockouts of the auth flows (OTP send/verify
attempts, failed logins) are a different mechanism stored in Postgres; they are listed below so the
whole picture of "what stops abuse on which route" is in one place.

# Interface

| Member | Signature | Behaviour |
|---|---|---|
| `ratelimit.Store` | `Allow(ctx, key string, limit int64, window time.Duration, burst int64, algo Algorithm) (*Result, error)` | One check: counts the request if allowed, returns `Allowed`, `Remaining`, `ResetTime`, `RetryAfter`. |
| `store.NewMemoryStore` | `func NewMemoryStore() *MemoryStore` | Per-process counters in one map behind one mutex. Keys are never evicted (TD-002). |
| `store.NewRedisStore` | `func NewRedisStore(client redis.UniversalClient) *RedisStore` | Lua script per algorithm. The sliding window uses Redis `TIME` (no clock skew between instances), a unique member per request, and `PEXPIRE` = window, so every key expires on its own. |
| `store.NewFallbackStore` | `func NewFallbackStore(primary, secondary ratelimit.Store, onError func(error)) *FallbackStore` | Asks `primary`; on error calls `onError` and answers from `secondary`. appService uses Redis → memory (A-001). |
| `middleware.NewHTTPMiddleware` | `func NewHTTPMiddleware(cfg ratelimit.Config) (*HTTPMiddleware, error)` | Sets `X-RateLimit-Limit/Remaining/Reset`; when denied writes `429` + `Retry-After` (whole seconds, rounded up, at least 1) + `details.retry_after_seconds`. **Fails open** when the store returns an error, which is why appService puts `FallbackStore` in front of Redis. |
| `deliveryhttp.NewClientIPRateLimit` | `(max int64, window time.Duration, trustForwardedFor bool, opts ...RateLimitOption)` | Sliding window per client IP: TCP peer, or the last `X-Forwarded-For` entry when `trustForwardedFor`. |
| `deliveryhttp.NewUserRateLimit` | `(max int64, window time.Duration, opts ...RateLimitOption)` | Sliding window per account. Must sit inside the JWT middleware; a request without claims gets `401` and is not counted. |
| `deliveryhttp.WithSharedStore` | `(store ratelimit.Store, scope string) RateLimitOption` | Puts the limiter's counters in the shared store under `<scope>:<id>`. Without it a limiter keeps its own private `MemoryStore` (what unit tests use). |

# Drivers / providers

| Store | Where | Status |
|---|---|---|
| Redis | `pkg/ratelimit/store/redis.go` | Used by appService (`rate_limit_store.backend: redis`). Sliding window verified against a real Redis 7: limits, expiry, 100 concurrent callers on a limit of 20 let exactly 20 through. |
| Memory | `pkg/ratelimit/store/memory.go` | appService's fallback while Redis is down, and the whole store when `backend: memory` (the E2E harness sets this). |

Fixed on 2026-10-14, because Redis could not be used without it: every Redis script returned the
Lua boolean `false` on a denied request, which reaches Go as `nil`, and the unchecked type assertion
panicked, so the first request over a limit became a `500` instead of a `429`. All four scripts now
return integers and the Go side checks the reply shape. The sliding window also stopped colliding
on same-timestamp members, and both stores now return a real `RetryAfter` (it was always 0).

# Lifecycle

`newRateLimitStore` (`services/appService/cmd/ratelimit_store.go`) runs once at startup, after the
config is loaded:

1. `RateLimitStoreConfig.Validate()` — `backend` must be `redis` or `memory`; for Redis, `addr`,
   `key_prefix`, `dial_timeout`, `op_timeout` are required. There is no default.
2. Redis: builds one `redis.Client` (`ReadTimeout` = `WriteTimeout` = `PoolTimeout` = `op_timeout`,
   100 ms in dev) and pings it within `dial_timeout`. An unreachable Redis stops the service, like
   the database.
3. Wraps it as `FallbackStore(RedisStore, MemoryStore)`. A Redis error during a request is logged
   at most once per 10 s and the check is answered by the per-instance memory count.
4. Every limiter in `cmd/main.go` gets `WithSharedStore(store, key_prefix + "<name>:<ip|user>")`.

Dev wiring: `docker-compose.yml` service `redis` (`redis:7.4.2-alpine`, `--requirepass`, no
persistence, `maxmemory 128mb`, `volatile-ttl`). `make run` waits for it (`redis-up`);
`make redis-ratelimits` lists live counters with their size and TTL; `make redis-cli` opens a shell.

# Dependencies

From the imports: `pkg/ratelimit/store` → `pkg/ratelimit`, `github.com/redis/go-redis/v9`;
`pkg/ratelimit/middleware` → `pkg/ratelimit`, `pkg/ratelimit/store`, `pkg/utils`. The root
package imports nothing from `pkg/`. `FallbackStore` takes an `onError` callback rather than a
logger, so the store package still does not import `pkg/logging`.

# Consumers

## Request-rate limits (this package)

All sliding window, all in the shared store. Key prefix in dev: `innerlamp:app:rl:`.

| Routes | Key | Budget (dev) | Scope in the store | Wired at |
|---|---|---|---|---|
| `POST /v1/temple-suggestions` | client IP | 5 / 1h | `temple_suggestion:ip` | `cmd/main.go` (`templeSuggestionLimit`), after `webAPIKey` |
| 9 × `GET /v1/administrative-units/...` (one shared budget) | client IP | 60 / 1m | `administrative_units:ip` | `cmd/main.go` (`administrativeUnitLimit`), after `webAPIKey` |
| 5 × `/v1/users/me` profile + avatar routes (one shared budget) | user id | 10 / 1m | `profile:user` | `cmd/profile_routes.go`, inside JWT auth |
| 3 × `/v1/users/me` temple-announcement routes | user id | 60 / 1m | `temple_announcement:user` | `cmd/temple_announcement_routes.go`, inside JWT auth |

`trust_forwarded_for` is `false` in dev for both IP limiters. backofficeService imports nothing
from this package.

## Business lockouts (Postgres, not this package)

| Flow | Counted per | Rule (dev) | Where |
|---|---|---|---|
| OTP send (`request-otp`, `resend-otp`) | phone + purpose | 60 s cooldown, 3 sends then locked 30 min | `usecase/otp_service.go`, `config.dev.yaml` `otp` |
| OTP verify | phone + purpose | 3 wrong codes then locked 30 min | `usecase/otp_service.go` |
| Login | phone | 5 failures within 15 min then locked 15 min | `usecase/login_usecase.go`, `config.dev.yaml` `login` |

## Routes with no limit of either kind

`POST /v1/auth/login`, `/refresh`, `/logout` (request rate), `POST /v1/attestation/challenge`,
`/v1/assertion/challenge`, `/v1/attestation/verify`, all catalogue `GET`s, memorials, genealogies,
invitations, lamps, `GET /v1/calendar/events`, temple schedule/events, and every backofficeService
route including `POST /v1/admin/auth/login`.

# Pending work

Proposed fixes from the 2026-10-14 review, highest priority first. None of these is implemented;
the user asked for them to be proposed only.

1. **Per-IP limit on the auth routes.** `request-otp`, `resend-otp`, `verify-otp`, `login`,
   `refresh` have only per-phone lockouts, so one client can walk through many phone numbers (SMS
   pumping, password spraying). Add `ClientIPRateLimit` on each (for example 10/min and 50/h per IP
   for OTP sends, 20/min for login), plus a global SMS budget per hour as a cost ceiling once a real
   SMS provider is wired.
2. **Admin login (backofficeService).** No limit and no lockout. Add a per-IP limiter and a
   per-email failure lockout like appService's login, using the same Redis.
3. **Attestation challenge routes.** Public, and each call inserts a row in
   `attestation_challenges`. Add a per-IP limiter.
4. **Atomic lockout counters.** OTP verify and login read the attempt count, compare, then
   increment, so parallel requests pass the limit. Reserve the attempt atomically
   (`UPDATE ... SET attempts = attempts + 1 WHERE ... AND attempts < $max RETURNING`) before the
   bcrypt compare. Tracked in the security report as M-3.
5. **Client IP behind the load balancer.** `trust_forwarded_for` takes the *last*
   `X-Forwarded-For` entry. That is right on bare Cloud Run, but behind a Google external HTTPS
   load balancer the last entry is the balancer and the client is the one before it, so every
   caller would share one bucket. Make the number of trusted hops a config value and set it per
   deployment.
6. **Carrier-grade NAT.** Vietnamese mobile networks put many subscribers behind one public IP.
   Pure per-IP budgets on routes used during registration (administrative units, 60/min) can 429
   real users. Keep per-IP budgets generous and pair them with a per-device key (attestation) or
   per-phone key where one exists.
7. **Memory store eviction** (TD-002) and **Redis TLS** (TD-001).
8. **Edge limit.** Add a Cloud Armor rate-based rule in front of the service as the outer layer;
   the in-process limits stay as the per-route, per-account layer.

Also unverified: the Redis wiring was exercised on a local Redis 7.0 and Postgres 16 with the real
binary (two instances shared one count; stopping Redis mid-run fell back to a per-instance count;
startup stops when Redis is missing), but the E2E suite runs with `backend: memory` and the
compose file itself was not started here (no Docker in the authoring environment).

# Related concepts

- [packages/middleware.md](middleware.md) — the Recovery/CORS/Logger chain the limiters run inside.
- [features/login.md](../features/login.md) — the login lockout listed above.
- [features/register.md](../features/register.md) — the OTP send/verify lockouts listed above.
- [features/attestation.md](../features/attestation.md) — the unlimited challenge routes.
