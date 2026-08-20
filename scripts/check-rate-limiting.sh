#!/bin/bash
# check-rate-limiting.sh
# Checks: HTTP endpoints must be protected by rate limiting via pkg/ratelimit
#
# WHY: InnerLamp global instructions §4 define strict rate limits:
#   - Anonymous:     10 req/min
#   - Authenticated: 100 req/min
#   - Admin:         1000 req/min
# Without rate limiting, the platform is vulnerable to:
#   - Brute-force attacks on OTP/password endpoints
#   - DoS via resource exhaustion
#   - Credential stuffing (OWASP A7)
# InnerLamp uses pkg/ratelimit (token bucket, Redis-backed for distributed environments)
#
# RULES:
#   Rule 1 — delivery/http handler files must import pkg/ratelimit
#   Rule 2 — mux/router registration files must wire up ratelimit middleware
#   Rule 3 — OTP, login, password-reset endpoints must ALWAYS have rate limiting
#             (auth endpoints are highest-risk: brute-force vector)
#   Rule 4 — Rate limit must use Redis backend in non-test code (memory = single instance only)
#
# Referenced by: run-all.sh [20/21]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

MODULE=$(go list -m 2>/dev/null || echo "unknown")

# ── Detect delivery/http directories ─────────────────────────────────────────
HTTP_DIRS=$(find services/ -type d -name "http" -path "*/delivery/*" 2>/dev/null | tr '\n' ' ')
HTTP_FILES=$(find services/ -type d -name "http" -path "*/delivery/*" \
  2>/dev/null \
  -exec find {} -name "*.go" ! -name "*_test.go" \; 2>/dev/null || true)

if [ -z "$HTTP_FILES" ]; then
  # No HTTP delivery files found — skip
  echo "RATE_LIMIT_ERRORS=0"
  exit 0
fi

# ── Rule 1: Handler registration files must import pkg/ratelimit ──────────────
# "Handler registration file" heuristic: file that calls http.Handle, http.HandleFunc,
# mux.Handle, router.GET, r.POST etc.
ROUTER_FILES=$(grep -rl \
  -e 'http\.Handle\b' \
  -e 'http\.HandleFunc\b' \
  -e '\.Handle(' \
  -e '\.HandleFunc(' \
  -e '\.GET(' \
  -e '\.POST(' \
  -e '\.PUT(' \
  -e '\.DELETE(' \
  --include="*.go" \
  services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -i 'delivery\|handler\|router\|route\|server' || true)

for router_file in $ROUTER_FILES; do
  if ! grep -q "ratelimit" "$router_file" 2>/dev/null; then
    ERRORS+=("MISSING_RATE_LIMIT_MIDDLEWARE")
    DETAILS+=("HTTP router/handler registration has no rate limit middleware | File: $router_file")
    HINTS+=("Import and wire pkg/ratelimit middleware. Example: mux.Use(ratelimit.HTTPMiddleware(limiter)). Anonymous endpoints: 10 req/min. Authenticated: 100 req/min. Use Redis backend for distributed rate limiting across Cloud Run instances.")
  fi
done

# ── Rule 2: Auth-sensitive endpoints MUST have rate limiting ──────────────────
# These endpoints are highest brute-force risk — non-negotiable
SENSITIVE_PATTERNS='otp\|login\|password\|pin\|auth\|signin\|verify\|token\|refresh'

SENSITIVE_FILES=$(find services/ -name "*.go" ! -name "*_test.go" \
  -path "*/delivery/*" \
  2>/dev/null \
  | xargs grep -liE "$SENSITIVE_PATTERNS" 2>/dev/null || true)

for sensitive_file in $SENSITIVE_FILES; do
  if ! grep -q "ratelimit" "$sensitive_file" 2>/dev/null; then
    ERRORS+=("AUTH_ENDPOINT_NO_RATE_LIMIT")
    DETAILS+=("Authentication/OTP endpoint handler has no rate limit protection | File: $sensitive_file")
    HINTS+=("CRITICAL: OTP and login endpoints without rate limiting are vulnerable to brute-force attacks. Add pkg/ratelimit with strict limit (max 5 attempts per minute per IP). Use Redis backend. Consider exponential backoff after 3 failures.")
  fi
done

# ── Rule 3: Rate limiter must use Redis backend in production-path code ────────
# Memory limiter is only for single-instance use — Cloud Run scales horizontally;
# a per-instance memory limiter is trivially bypassed by hitting different instances.
MEMORY_LIMITER=$(grep -rn \
  -e 'ratelimit\.NewMemory\b' \
  -e 'NewInMemory\b' \
  -e 'memory\.New\b' \
  --include="*.go" \
  services/ \
  2>/dev/null \
  | grep -v "_test.go" || true)

if [ -n "$MEMORY_LIMITER" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    FILE=$(echo "$line" | cut -d: -f1)
    LINENO=$(echo "$line" | cut -d: -f2)
    ERRORS+=("RATE_LIMIT_MEMORY_BACKEND")
    DETAILS+=("In-memory rate limiter used in service code | File: $FILE:$LINENO")
    HINTS+=("Use Redis-backed rate limiter: ratelimit.NewRedis(redisClient, ...). In-memory limiters do NOT work across Cloud Run instances — each instance has its own counter, allowing 10x more requests than the limit if 10 instances are running.")
  done <<< "$MEMORY_LIMITER"
fi

# ── Rule 4: cmd/main.go must register rate limit middleware before starting server ─
for main_file in $(find services/ -name "main.go" -path "*/cmd/*" 2>/dev/null | sort); do
  has_ratelimit=$(grep -c "ratelimit" "$main_file" 2>/dev/null || echo "0")
  has_http_server=$(grep -c \
    -e 'http\.ListenAndServe' \
    -e 'http\.Serve(' \
    -e 'ListenAndServe' \
    "$main_file" 2>/dev/null || echo "0")

  if [ "$has_http_server" -gt 0 ] && [ "$has_ratelimit" -eq 0 ]; then
    ERRORS+=("MAIN_MISSING_RATE_LIMIT_SETUP")
    DETAILS+=("cmd/main.go starts HTTP server but does not configure rate limiting | File: $main_file")
    HINTS+=("Initialize rate limiter in main.go and inject into router middleware chain. Use pkg/ratelimit.NewRedis(redisClient, cfg.RateLimit) and pass to HTTP server setup.")
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "RATE_LIMIT_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
