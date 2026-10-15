---
type: Package
title: pkg/middleware
description: Provider-agnostic HTTP middleware (request logging, hardcoded CORS, panic recovery) and gRPC recovery interceptors, wired into appService as Recovery(CORS(Logger(WithAPIFallback(mux)))).
resource: pkg/middleware/
tags: [infra, package, architecture, networking]
status: draft
generated: { by: claude-code, at: 2026-09-22T16:10:00+07:00 }
sources:
  - path: pkg/middleware/README.md
    kind: implementation-notes
  - path: docs/specs/features/http-middleware-wiring/requirements.md
    kind: spec
  - path: docs/specs/features/http-middleware-wiring/design.md
    kind: spec
  - path: docs/specs/features/api-response-envelope/requirements.md
    kind: spec
---

# Summary

`pkg/middleware` holds small, domain-free `net/http` middleware and gRPC interceptors that every
service can reuse: request logging, CORS headers, and panic recovery. Unlike the per-package
`middleware` folders (`pkg/jwt/middleware`, `pkg/logging/middleware`, `pkg/ratelimit/middleware`,
`pkg/metrics/middleware/...`), which are glued to their owning package, this one is standalone and
imports nothing from `pkg/` except `pkg/utils` (for the response envelope).

It is **not** a factory-registry package: there is no `Register(type, FactoryFunc)`, no driver
subpackage, and no `New(config)` — the analog of `utils`/`validation`, it is consumed by calling the
functions directly. The HTTP half is now really used: `services/appService/cmd/main.go` builds the
server handler through `newHandler`, which composes all three HTTP middleware into one chain (see
below). The gRPC half has no consumer in this repo yet.

# Interface

| Member | Signature | Behaviour |
|---|---|---|
| `Logger` | `func Logger(next http.Handler) http.Handler` | Logs `Started "<METHOD>" "<PATH>"` before `next` and `Completed in <duration>` after, via the global `log.Printf`. It captures no status code or response size. |
| `CORS` | `func CORS(next http.Handler) http.Handler` | Always sets `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`, `Access-Control-Allow-Headers: Content-Type, Authorization`. An `OPTIONS` request is answered immediately with `200` + `Content-Type: application/json` + `{"success": true}` through `utils.WriteSuccessResponseWithStatus` and does **not** reach `next`. Any other method passes through unchanged. |
| `Recovery` | `func Recovery(next http.Handler) http.Handler` | Recovers a panic, logs it with the stack via `log.Printf`, and writes `500` `INTERNAL_ERROR` (type `system`) through `utils.WriteSystemError` — the same envelope as any other 500. |
| `RecoveryInterceptor` | `func RecoveryInterceptor() grpc.UnaryServerInterceptor` | Recovers a panic in a unary gRPC handler and returns `codes.Internal` with the panic value **and the stack trace embedded in the message to the client**. |
| `StreamRecoveryInterceptor` | `func StreamRecoveryInterceptor() grpc.StreamServerInterceptor` | Stream equivalent of `RecoveryInterceptor`; same stack leak. |
| `RecoveryWithLoggerInterceptor` | `func RecoveryWithLoggerInterceptor(logFunc func(msg string, keysAndValues ...interface{})) grpc.UnaryServerInterceptor` | Recovers a panic, reports `method`/`panic`/`stack` through `logFunc`, and returns `codes.Internal` with only the panic value (no stack) to the client. |
| `StreamRecoveryWithLoggerInterceptor` | `func StreamRecoveryWithLoggerInterceptor(logFunc func(msg string, keysAndValues ...interface{})) grpc.StreamServerInterceptor` | Stream equivalent of `RecoveryWithLoggerInterceptor`. |

The HTTP `Recovery` is stricter than the gRPC one on purpose: a client sees a plain
`500 INTERNAL_ERROR` with no panic value or stack, which is why `Recovery` can be outermost in the
appService chain.

# HTTP middleware chain in appService

`services/appService/cmd/main.go` composes the chain in one function, `newHandler(mux *http.ServeMux)`,
so the composition root — not a handler — owns it and tests can assert the real wiring:

```
Recovery( CORS( Logger( deliveryhttp.WithAPIFallback(mux) ) ) )
```

The order is load-bearing:

- **`Recovery` outermost** catches a panic raised by any layer inside it (`CORS`, `Logger`, the
  fallback, or a route handler) and turns it into the `500 INTERNAL_ERROR` envelope.
- **`CORS` outside `Logger`** so a preflight `OPTIONS` is answered and stops before `Logger` runs —
  preflight therefore produces **no** log lines.
- **`Logger` outside the fallback and mux** so a matched route and `ServeMux`'s own `404`/`405`
  both get their `Started`/`Completed` lines.
- **`WithAPIFallback` innermost** rewrites only `ServeMux`'s `404`/`405` bodies into the
  `utils.ApiResponse` envelope; a matched route's status/body is never touched.

Consequences visible to a client: every response (matched, `404`, `405`, preflight, panic) carries
the three `Access-Control-*` headers; a preflight is a `200` envelope rather than an empty `200`;
every non-preflight request adds two stdout log lines. The `newHandler` argument is
`*http.ServeMux` (not `http.Handler`) because `deliveryhttp.WithAPIFallback` requires a
`*http.ServeMux`.

# Dependencies

- Internal `pkg/` imports: `pkg/utils` only — `Recovery` calls `utils.WriteSystemError`, `CORS`
  calls `utils.WriteSuccessResponseWithStatus`. This is the same `middleware → utils` edge recorded
  for `pkg/`; `pkg/utils` remains a stdlib-only leaf, so there is no cycle.
- External: `google.golang.org/grpc` — imported only by `grpc.go` (`middleware.go` is stdlib +
  `pkg/utils`).
- No config struct, no environment variables, no secret: every function takes what it needs as an
  argument. CORS origin/methods/headers are hardcoded.

# Consumers

- `services/appService/cmd/main.go` — `newHandler` uses `Recovery`, `CORS` and `Logger` in the chain
  above; this is the only consumer today.
- No consumer uses the gRPC interceptors, `RecoveryWithLoggerInterceptor`, or
  `StreamRecoveryWithLoggerInterceptor` in this repo. `pkg/logging/middleware.HTTPLogger` exists as
  the structured alternative to `Logger` and is likewise unused.

# Pending work

- The wiring was verified with unit tests (`pkg/middleware/middleware_test.go` and
  `services/appService/cmd/main_test.go`) but **not** exercised on a running binary against a real
  Postgres in this pass — the `curl` checks in
  `docs/specs/features/http-middleware-wiring/tasks.md` Phase 3 remain unrun. That is why this
  concept is `draft`. Promote to `stable` once a real server is exercised.
- `RecoveryInterceptor`/`StreamRecoveryInterceptor` have no test and no consumer, and they return
  the stack trace to the client (information leak if ever exposed). Prefer the
  `...WithLogger...` variants before adopting them.
- No request ID / trace ID is attached; `Logger` and HTTP `Recovery` use the global `log.Printf`,
  so their lines are not the same JSON format as the service's `pkg/logging` logger — a documented
  trade-off, not a bug fixed here.
- `CORS` is not configurable: it cannot whitelist an origin, and `Access-Control-Allow-Origin: *`
  is incompatible with `Access-Control-Allow-Credentials`.

# Related concepts

- `docs/specs/features/login/` and `docs/specs/features/onboarding/`:
  their routes are the HTTP traffic this chain wraps; they rely on the `utils.ApiResponse` envelope
  that `Recovery` and `CORS` now write through.
- No `Package` concept exists yet for `pkg/utils` — the package this one depends on and whose
  `WriteSuccessResponseWithStatus`/`WriteSystemError` produce the preflight and panic responses. Not
  yet documented as an OKF concept in this bundle.
- No `Package` concept exists yet for `pkg/jwt` — the per-package `middleware` subpackage
  (`pkg/jwt/middleware`) that wraps the logout route sits *inside* the `mux`, i.e. further in than
  this chain. Not yet documented as an OKF concept in this bundle.
- No `Architecture Pattern` concept exists yet for the "one response envelope, written only through
  `utils.Write*`" rule this package now follows (spec:
  `docs/specs/features/api-response-envelope/`). Not yet documented as an OKF concept in this
  bundle.
