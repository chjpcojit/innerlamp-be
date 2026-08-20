#!/bin/bash
# check-coverage.sh
# Checks: test coverage against InnerLamp targets
# Domain: 95%, UseCase: 85%, Repository: 70%, Delivery: 75%

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# Run tests with coverage
echo "Running tests with coverage..."
go test -coverprofile=coverage.out ./... > /dev/null 2>&1 || true

# Parse coverage by package
while IFS= read -r line; do
  # Format: github.com/innerlamp/users/internal/domain    coverage: 87.5% of statements
  if [[ "$line" =~ coverage:\ ([0-9.]+)%\ of\ statements ]]; then
    coverage="${BASH_REMATCH[1]}"
    pkg=$(echo "$line" | awk '{print $1}')
    
    # Determine target based on package path
    if [[ "$pkg" =~ /domain ]]; then
      target=95.0
      layer="Domain"
    elif [[ "$pkg" =~ /usecase ]]; then
      target=85.0
      layer="UseCase"
    elif [[ "$pkg" =~ /repository ]]; then
      target=70.0
      layer="Repository"
    elif [[ "$pkg" =~ /delivery ]]; then
      target=75.0
      layer="Delivery"
    else
      target=60.0
      layer="Infrastructure"
    fi
    
    # Check if coverage meets target
    if (( $(echo "$coverage < $target" | bc -l) )); then
      ERRORS+=("INSUFFICIENT_COVERAGE")
      DETAILS+=("$layer coverage: ${coverage}% (target: ${target}%) | Package: $pkg")
      HINTS+=("Add more tests to reach ${target}% coverage. Critical for banking compliance.")
    fi
  fi
done < <(go tool cover -func=coverage.out 2>/dev/null)

# Check for critical untested functions (money calculations)
critical_funcs=$(grep -rn "func.*Calculate\|func.*Transfer\|func.*Debit\|func.*Credit" services/ | cut -d: -f1 | sort -u)

for file in $critical_funcs; do
  test_file="${file%.go}_test.go"
  if [ ! -f "$test_file" ]; then
    ERRORS+=("CRITICAL_FUNCTION_UNTESTED")
    DETAILS+=("Critical financial function in $file has no tests")
    HINTS+=("URGENT: Banking compliance requires 100% test coverage for money calculations")
  fi
done

# Export results
echo "COVERAGE_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done

# Cleanup
rm -f coverage.out
