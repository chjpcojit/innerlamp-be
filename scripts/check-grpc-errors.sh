#!/bin/bash
# check-grpc-errors.sh
# Checks: gRPC handlers must return status.Error(codes.XXX, ...) — not raw errors
# Banking compliance: correct gRPC status codes required for frontend error handling
# Referenced by: run-all.sh [13/16]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Find all gRPC handler files ───────────────────────────────────────────────
GRPC_DIRS=$(find services/ -type d -name "grpc" 2>/dev/null | tr '\n' ' ')

if [ -z "$GRPC_DIRS" ]; then
  echo "GRPC_ERRORS=0"
  echo "GRPC_SKIPPED=no delivery/grpc directories found"
  exit 0
fi

# ── Rule 1: gRPC handlers must not return raw errors ─────────────────────────
# Detect: return nil, err  OR  return nil, fmt.Errorf(...)  without status.Error wrap
# inside files under delivery/grpc/
for grpc_file in $(find $GRPC_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  # Check if file has gRPC handler signatures (returns proto.Message + error)
  if ! grep -q "context.Context" "$grpc_file" 2>/dev/null; then
    continue
  fi

  # Find bare "return nil, err" lines (not inside a status.Error call)
  BARE_RETURNS=$(grep -n "return nil, err\b" "$grpc_file" 2>/dev/null || true)
  if [ -n "$BARE_RETURNS" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        LINENO=$(echo "$line" | cut -d: -f1)
        ERRORS+=("GRPC_RAW_ERROR")
        DETAILS+=("Bare 'return nil, err' in gRPC handler — client receives codes.Unknown | File: $grpc_file:$LINENO")
        HINTS+=("Wrap with status.Error: return nil, status.Error(codes.Internal, err.Error()) — import google.golang.org/grpc/status and google.golang.org/grpc/codes")
      fi
    done <<< "$BARE_RETURNS"
  fi

  # Find fmt.Errorf returns not wrapped in status.Error
  FMT_RETURNS=$(grep -n "return nil, fmt\.Errorf" "$grpc_file" 2>/dev/null || true)
  if [ -n "$FMT_RETURNS" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        LINENO=$(echo "$line" | cut -d: -f1)
        ERRORS+=("GRPC_RAW_ERROR")
        DETAILS+=("fmt.Errorf in gRPC handler — client receives codes.Unknown | File: $grpc_file:$LINENO")
        HINTS+=("Map domain errors to gRPC status codes: ErrNotFound→codes.NotFound, ErrUnauthorized→codes.Unauthenticated, ErrInvalidInput→codes.InvalidArgument")
      fi
    done <<< "$FMT_RETURNS"
  fi

  # Find domain errors returned directly (domain.ErrXxx without status.Error)
  DOMAIN_RETURNS=$(grep -n "return nil, domain\.Err" "$grpc_file" 2>/dev/null || true)
  if [ -n "$DOMAIN_RETURNS" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        LINENO=$(echo "$line" | cut -d: -f1)
        ERRORS+=("GRPC_RAW_ERROR")
        DETAILS+=("Domain error returned directly in gRPC handler — client receives codes.Unknown | File: $grpc_file:$LINENO")
        HINTS+=("Map domain errors to status codes: return nil, status.Error(codes.NotFound, domain.ErrSessionNotFound.Error()). Create a mapDomainError() helper in the handler file.")
      fi
    done <<< "$DOMAIN_RETURNS"
  fi
done

# ── Rule 2: gRPC handler files must import grpc/status and grpc/codes ─────────
for grpc_file in $(find $GRPC_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  # Only check files that have handler-like functions
  if ! grep -q "func.*context\.Context.*proto\." "$grpc_file" 2>/dev/null; then
    continue
  fi

  if ! grep -q "google.golang.org/grpc/status\|google.golang.org/grpc/codes" "$grpc_file" 2>/dev/null; then
    ERRORS+=("GRPC_MISSING_STATUS_IMPORT")
    DETAILS+=("gRPC handler file missing status/codes import | File: $grpc_file")
    HINTS+=("Add imports: \"google.golang.org/grpc/codes\" and \"google.golang.org/grpc/status\"")
  fi
done

# ── Rule 3: Recommended — error mapping helper function ──────────────────────
# Detect gRPC handler files with >3 error returns but no mapError/mapDomainError helper
for grpc_file in $(find $GRPC_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  ERROR_COUNT=$(grep -c "status\.Error(codes\." "$grpc_file" 2>/dev/null || true)
  HAS_MAPPER=$(grep -q "func map.*[Ee]rror\|func.*toStatus" "$grpc_file" 2>/dev/null && echo "yes" || echo "no")

  if [ "$ERROR_COUNT" -gt 3 ] && [ "$HAS_MAPPER" = "no" ]; then
    ERRORS+=("GRPC_MISSING_ERROR_MAPPER")
    DETAILS+=("gRPC handler has $ERROR_COUNT status.Error calls but no error mapping helper | File: $grpc_file")
    HINTS+=("Extract a mapDomainError(err error) (codes.Code, string) helper to centralize domain→gRPC error code mapping and avoid duplication")
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "GRPC_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
