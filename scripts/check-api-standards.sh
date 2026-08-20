#!/bin/bash
# check-api-standards.sh
# Checks: API standards compliance (.github/instructions/api-standards.md)
# Rules that can be verified statically via grep:
#   1. No log.Printf / fmt.Println in delivery layer
#   2. No raw Authorization header parsing in handlers
#   3. No new logger instance created inside delivery layer
#   4. No context.Background() in usecase or repository layers
#   5. No hardcoded port/host/DSN strings in Go source
#   6. No tracing span without defer span.End()

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

DELIVERY_DIRS=$(find services -type d -name "delivery" 2>/dev/null | tr '\n' ' ')
USECASE_DIRS=$(find services -type d -name "usecase" 2>/dev/null | tr '\n' ' ')
REPO_DIRS=$(find services -type d -name "repository" 2>/dev/null | tr '\n' ' ')

# ── Rule 1: No stdlib log.Printf / fmt.Println in delivery layer ─────────────
if [ -n "$DELIVERY_DIRS" ]; then
  LOG_VIOLATIONS=$(grep -rn \
    -e 'log\.Printf\|log\.Println\|log\.Fatal\|log\.Panic' \
    -e 'fmt\.Println\|fmt\.Printf' \
    --include="*.go" $DELIVERY_DIRS 2>/dev/null | grep -v "_test.go" || true)

  if [ -n "$LOG_VIOLATIONS" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        FILE=$(echo "$line" | cut -d: -f1)
        LINENO=$(echo "$line" | cut -d: -f2)
        ERRORS+=("API_STANDARDS_VIOLATION")
        DETAILS+=("Unstructured logging in delivery layer | File: $FILE:$LINENO")
        HINTS+=("Use logctx.FromContext(ctx) to retrieve the request-scoped logger — never use log.Printf or fmt.Println. Reference: api-standards.md Section 2")
      fi
    done <<< "$LOG_VIOLATIONS"
  fi
fi

# ── Rule 2: No manual Authorization header parsing in handlers ────────────────
if [ -n "$DELIVERY_DIRS" ]; then
  AUTH_HEADER=$(grep -rn \
    'Header\.Get.*[Aa]uthorization\|Header\["[Aa]uthorization"\]' \
    --include="*.go" $DELIVERY_DIRS 2>/dev/null | grep -v "_test.go" || true)

  if [ -n "$AUTH_HEADER" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        FILE=$(echo "$line" | cut -d: -f1)
        LINENO=$(echo "$line" | cut -d: -f2)
        ERRORS+=("API_STANDARDS_VIOLATION")
        DETAILS+=("Manual Authorization header parsing in handler | File: $FILE:$LINENO")
        HINTS+=("Use jwtcontext.GetUserID(ctx) or jwtcontext.GetClaims(ctx) — never parse the Authorization header directly. Reference: api-standards.md Section 5")
      fi
    done <<< "$AUTH_HEADER"
  fi
fi

# ── Rule 3: No new logger instance created inside delivery layer ──────────────
if [ -n "$DELIVERY_DIRS" ]; then
  NEW_LOGGER=$(grep -rn \
    'logging\.New(\|logger\.New(' \
    --include="*.go" $DELIVERY_DIRS 2>/dev/null | grep -v "_test.go" || true)

  if [ -n "$NEW_LOGGER" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        FILE=$(echo "$line" | cut -d: -f1)
        LINENO=$(echo "$line" | cut -d: -f2)
        ERRORS+=("API_STANDARDS_VIOLATION")
        DETAILS+=("New logger instance created in delivery layer | File: $FILE:$LINENO")
        HINTS+=("Retrieve the request-scoped logger from context: logctx.FromContext(ctx). Creating a new logger loses the correlation ID. Reference: api-standards.md Section 2")
      fi
    done <<< "$NEW_LOGGER"
  fi
fi

# ── Rule 4: No context.Background() in usecase or repository layers ───────────
for DIR in $USECASE_DIRS $REPO_DIRS; do
  if [ -z "$DIR" ]; then continue; fi
  CTX_BG=$(grep -rn \
    'context\.Background()' \
    --include="*.go" "$DIR" 2>/dev/null | grep -v "_test.go" || true)

  if [ -n "$CTX_BG" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        FILE=$(echo "$line" | cut -d: -f1)
        LINENO=$(echo "$line" | cut -d: -f2)
        ERRORS+=("API_STANDARDS_VIOLATION")
        DETAILS+=("context.Background() used in usecase/repository — breaks tracing and cancellation | File: $FILE:$LINENO")
        HINTS+=("Pass ctx from the caller through the entire call chain. context.Background() severs the trace span and cancellation signal. Reference: api-standards.md Section 3")
      fi
    done <<< "$CTX_BG"
  fi
done

# ── Rule 5: Hardcoded port / hostname / DSN strings in non-config Go files ────
HARDCODED_CONFIG=$(grep -rn \
  -e '":8080"\|":8443"\|":3000"\|":9090"' \
  -e '"localhost"\|"127\.0\.0\.1"' \
  -e '"5432"\|"27017"\|"6379"' \
  --include="*.go" services/ 2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example" || true)

if [ -n "$HARDCODED_CONFIG" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_CONFIG")
      DETAILS+=("Hardcoded config value (port/host/DSN) in source code | File: $FILE:$LINENO")
      HINTS+=("Load all operational config from config/config.yaml — never hardcode host, port, or connection strings. Reference: goalng.md Section 10")
    fi
  done <<< "$HARDCODED_CONFIG"
fi

# ── Rule 6: tracer.Start without defer span.End() in same function ────────────
# Detect files that call .Start( but lack span.End()
SPAN_START_FILES=$(grep -rln \
  '\.Start(ctx\|otel\.Tracer' \
  --include="*.go" services/ 2>/dev/null | grep -v "_test.go" || true)

for FILE in $SPAN_START_FILES; do
  if ! grep -q 'span\.End()' "$FILE" 2>/dev/null; then
    ERRORS+=("API_STANDARDS_VIOLATION")
    DETAILS+=("Tracing span started but span.End() not found in file | File: $FILE")
    HINTS+=("Every tracer.Start() call must be immediately followed by 'defer span.End()'. Reference: api-standards.md Section 3")
  fi
done

# ── Export results ─────────────────────────────────────────────────────────────
echo "API_STANDARDS_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
