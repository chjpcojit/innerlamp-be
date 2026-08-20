#!/bin/bash
# check-context-propagation.sh
# Checks: context.Background() / context.TODO() must NOT appear in service code
# outside of cmd/main.go entry points and _test.go files.
#
# WHY: InnerLamp uses OpenTelemetry distributed tracing (pkg/tracing) and structured
# logging with correlation IDs (pkg/logging). Using context.Background() mid-call-chain
# BREAKS trace propagation, log correlation, and deadline/cancellation inheritance.
# Each request's context must flow uninterrupted from delivery → usecase → repository.
#
# ALLOWED locations:
#   - cmd/main.go               (server startup bootstrap)
#   - pkg/**/*.go               (cross-service utility roots)
#   - *_test.go                 (test setup)
#   - files that explicitly contain "// BACKGROUND_CTX_OK: <reason>"
#
# Banking compliance: Violating context propagation breaks request tracing required
# for CNBV audit trail and incident investigation.
# Referenced by: run-all.sh [17/21]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Patterns to detect ───────────────────────────────────────────────────────
CTX_PATTERN='context\.Background()\|context\.TODO()'

# ── Scan all service Go files that are NOT whitelisted locations ─────────────
while IFS= read -r go_file; do
  # Skip test files
  [[ "$go_file" == *"_test.go" ]] && continue

  # Skip cmd/main.go and cmd/**/*.go (entry points — legitimate context roots)
  [[ "$go_file" == */cmd/* ]] && continue

  # Skip pkg/ (library utilities can be context roots when called from outside)
  [[ "$go_file" == */pkg/* ]] && continue

  # Skip if file has explicit approval comment
  if grep -q "BACKGROUND_CTX_OK" "$go_file" 2>/dev/null; then
    continue
  fi

  # Find violations line by line
  while IFS= read -r match; do
    [ -z "$match" ] && continue
    LINENO=$(echo "$match" | cut -d: -f1)
    CODE=$(echo "$match" | cut -d: -f2-)
    ERRORS+=("CONTEXT_PROPAGATION_VIOLATION")
    DETAILS+=("context.Background()/TODO() in non-entry-point | File: $go_file:$LINENO | Code: $CODE")
    HINTS+=("Replace with the 'ctx' parameter received from the caller. context.Background() breaks distributed tracing and log correlation. If this is truly a root context, add comment: // BACKGROUND_CTX_OK: <reason>")
  done < <(grep -n "$CTX_PATTERN" "$go_file" 2>/dev/null || true)

done < <(find services/ -name "*.go" 2>/dev/null | sort)

# ── Rule 2: context.WithValue must use typed keys (not raw string literals) ──
# Using string literals as context keys causes collisions between packages.
# Correct: type contextKey string; const requestIDKey contextKey = "request_id"
STRKEY_VIOLATIONS=$(grep -rn \
  'context\.WithValue(.*,\s*"' \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" || true)

if [ -n "$STRKEY_VIOLATIONS" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    FILE=$(echo "$line" | cut -d: -f1)
    LINENO=$(echo "$line" | cut -d: -f2)
    CODE=$(echo "$line" | cut -d: -f3-)
    ERRORS+=("CONTEXT_STRING_KEY")
    DETAILS+=("context.WithValue uses raw string key | File: $FILE:$LINENO | Code: $CODE")
    HINTS+=("Use a typed unexported key: type ctxKey string; const myKey ctxKey = \"value\". Raw string keys cause silent collisions between packages.")
  done <<< "$STRKEY_VIOLATIONS"
fi

# ── Rule 3: Goroutines spawned from handlers must receive a copy of ctx ──────
# Pattern: go func() { ... } without ctx in the goroutine body
# Heuristic: goroutine inside a function that has a ctx parameter but the
# anonymous func does not reference it.
DETACHED_GOROUTINE=$(grep -rn \
  'go func()' \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" || true)

if [ -n "$DETACHED_GOROUTINE" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    FILE=$(echo "$line" | cut -d: -f1)
    LINENO=$(echo "$line" | cut -d: -f2)
    # Check if surrounding function has a ctx param
    if grep -q "ctx context.Context" "$FILE" 2>/dev/null; then
      ERRORS+=("DETACHED_GOROUTINE")
      DETAILS+=("go func() without context | File: $FILE:$LINENO | Goroutine may escape request context lifetime")
      HINTS+=("Pass a detached context explicitly: detachedCtx := context.WithoutCancel(ctx) then go func(ctx context.Context) { ... }(detachedCtx). Never capture caller ctx directly in long-lived goroutines.")
    fi
  done <<< "$DETACHED_GOROUTINE"
fi

# ── Export results ────────────────────────────────────────────────────────────
echo "CTX_PROPAGATION_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
