#!/bin/bash
# check-business-rules.sh
# Checks: error codes from BA schema exist in domain/errors.go
# Usage: ./check-business-rules.sh docs/ba-schemas/feature-schema.md

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

SCHEMA_FILE="${1:-}"

# Skip if no schema file provided
if [ -z "$SCHEMA_FILE" ]; then
  echo "BR_ERRORS=0"
  echo "BR_SKIPPED=no schema file provided"
  exit 0
fi

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "BR_ERRORS=1"
  echo "ERR_TYPE_0=SCHEMA_NOT_FOUND"
  echo "ERR_DETAIL_0=Schema file not found: $SCHEMA_FILE"
  echo "ERR_HINT_0=Run BA Parser agent first to generate the schema"
  exit 0
fi

# Find domain errors file
ERRORS_FILE=$(find services/ -path "*/domain/errors.go" -type f 2>/dev/null | head -1)
if [ -z "$ERRORS_FILE" ]; then
  ERRORS+=("MISSING_ERROR_FILE")
  DETAILS+=("Cannot check BR coverage: internal/domain/errors.go not found")
  HINTS+=("Create internal/domain/errors.go with sentinel error variables")
  echo "BR_ERRORS=${#ERRORS[@]}"
  echo "ERR_TYPE_0=${ERRORS[0]}"
  echo "ERR_DETAIL_0=${DETAILS[0]}"
  echo "ERR_HINT_0=${HINTS[0]}"
  exit 0
fi

# Extract error codes from schema (format: ERR001: ERROR_CODE → HTTP NNN)
SCHEMA_ERRORS=$(grep -E "^- ERR[0-9]+:" "$SCHEMA_FILE" \
  | sed 's/.*ERR[0-9]*: \([A-Z_]*\).*/\1/' 2>/dev/null || true)

if [ -z "$SCHEMA_ERRORS" ]; then
  echo "BR_ERRORS=0"
  echo "BR_SKIPPED=no error codes found in schema (check schema format)"
  exit 0
fi

# Check each schema error code exists in domain errors
while IFS= read -r error_code; do
  if [ -z "$error_code" ]; then
    continue
  fi

  # Convert ERROR_CODE to Go sentinel format
  # INSUFFICIENT_BALANCE → ErrInsufficientBalance or ErrAccountInsufficientBalance
  GO_ERR_PATTERN=$(echo "$error_code" \
    | sed 's/_\([A-Z]\)/\1/g' \
    | sed 's/^./\u&/')

  if ! grep -q "Err.*${GO_ERR_PATTERN}\|Err.*$(echo $error_code | tr '_' ' ' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) tolower(substr($i,2)); print}' | tr -d ' ')" \
    "$ERRORS_FILE" 2>/dev/null; then
    ERRORS+=("MISSING_ERROR_HANDLER")
    DETAILS+=("Schema error $error_code not found in $ERRORS_FILE")
    HINTS+=("Add: var Err$(echo $error_code | sed 's/_\([A-Z]\)/\1/g' | sed 's/./\u&/') = errors.New(\"innerlamp: ...\") to domain/errors.go")
  fi
done <<< "$SCHEMA_ERRORS"

# Check regression: existing tests pass
REGRESSION_OUT=$(go test ./... 2>&1) || true
FAILED_TESTS=$(echo "$REGRESSION_OUT" | grep "^--- FAIL" || true)
if [ -n "$FAILED_TESTS" ]; then
  while IFS= read -r failed; do
    if [ -n "$failed" ]; then
      ERRORS+=("REGRESSION")
      DETAILS+=("Test failed: $failed")
      HINTS+=("Existing test broken by change — fix implementation to match expected behavior")
    fi
  done <<< "$FAILED_TESTS"
fi

# Export results
echo "BR_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done