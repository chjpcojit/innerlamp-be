#!/bin/bash
# check-compile.sh
# Checks: go build, go vet
# Output: array of errors to be collected by run-all.sh

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# --- go build ---
BUILD_OUT=$(go build ./... 2>&1) || true
if [ -n "$BUILD_OUT" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      ERRORS+=("COMPILE_ERROR")
      DETAILS+=("$line")
      HINTS+=("Fix type mismatch, missing import, or undefined reference at the indicated line")
    fi
  done <<< "$BUILD_OUT"
fi

# --- go vet ---
VET_OUT=$(go vet ./... 2>&1) || true
if [ -n "$VET_OUT" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      ERRORS+=("VET_ERROR")
      DETAILS+=("$line")
      HINTS+=("go vet issue — common causes: unused variable, unreachable code, printf format mismatch")
    fi
  done <<< "$VET_OUT"
fi

# Export results
echo "COMPILE_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done