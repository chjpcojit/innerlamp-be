#!/bin/bash
# check-completeness.sh
# Checks: missing tests, constructors, implementations
# Output: structured violations

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Dynamic path discovery (works for services/<svc>/ layout) ─────────────────
DOMAIN_DIRS=$(find services/ -type d -name "domain" 2>/dev/null | tr '\n' ' ')
USECASE_DIRS=$(find services/ -type d -name "usecase" 2>/dev/null | tr '\n' ' ')
REPO_IMPL_DIRS=$(find services/ -type d -name "repository" ! -path "*/domain/*" 2>/dev/null | tr '\n' ' ')

# --- Rule 1: Every .go file (except *_test.go) needs a *_test.go ---
# Skip files with no function declarations (interface, error, type-only definition files)
for go_file in $(find services/ -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  # No exported functions → nothing to unit test (interface/error/const-only file)
  if ! grep -q "^func " "$go_file" 2>/dev/null; then
    continue
  fi
  test_file="${go_file%.go}_test.go"
  if [ ! -f "$test_file" ]; then
    ERRORS+=("MISSING_TEST")
    DETAILS+=("Test file missing for $go_file")
    HINTS+=("Create $test_file with table-driven tests")
  fi
done

# --- Rule 2: Every struct needs a constructor (NewXxx or similar) ---
# Exclusions (no constructor needed):
#   - *Request, *Response      — DTO/input-output, initialized with struct literals
#   - *Input, *Result          — value objects, caller sets fields directly
#   - *Checks, *Info           — small data holders
#   - *PG, *Document           — DB row mappers, ORM-style scan targets
#   - structs with only exported fields and no validation logic (pure data bags)
CONSTRUCTOR_SKIP_SUFFIX='Request$|Response$|Input$|Result$|Checks$|Info$|PG$|Document$'

for go_file in $(find services/ -name "*.go" ! -name "*_test.go" \( -path "*/domain/*" -o -path "*/usecase/*" \) 2>/dev/null); do
  structs=$(grep -E "^type [A-Z][a-zA-Z0-9]+ struct" "$go_file" | awk '{print $2}' || true)

  for struct_name in $structs; do
    # Skip well-known no-constructor patterns
    if echo "$struct_name" | grep -qE "$CONSTRUCTOR_SKIP_SUFFIX"; then
      continue
    fi

    # Skip structs where ALL fields are exported (pure data bag — no invariants to protect)
    # A data bag has zero unexported fields; business entities always have at least one
    total_fields=$(awk "/^type ${struct_name} struct/,/^}/" "$go_file" \
      | grep -cE "^\s+[a-zA-Z]" || true)
    unexported_fields=$(awk "/^type ${struct_name} struct/,/^}/" "$go_file" \
      | grep -cE "^\s+[a-z][a-zA-Z0-9]*\s" || true)
    if [ "$total_fields" -gt 0 ] && [ "$unexported_fields" -eq 0 ]; then
      continue
    fi

    # Constructor may live in ANY file in the same package, not only the defining file
    pkg_dir=$(dirname "$go_file")
    if ! grep -rq "func New${struct_name}" "$pkg_dir" 2>/dev/null; then
      ERRORS+=("MISSING_CONSTRUCTOR")
      DETAILS+=("Struct $struct_name in $go_file has no constructor")
      HINTS+=("Add: func New${struct_name}(...) (*${struct_name}, error)")
    fi
  done
done

# --- Rule 3: All repository interfaces have implementations ---
# Find interface definitions in */domain/repository/
for interface_file in $(find services/ -path "*/domain/repository/*.go" 2>/dev/null); do
  # Extract interface names
  interfaces=$(grep -E "^type [A-Z][a-zA-Z0-9]+ interface" "$interface_file" | awk '{print $2}' || true)
  
  for interface_name in $interfaces; do
    # Check if there's an implementation in services/*/repository/
    impl_found=""
    if [ -n "$REPO_IMPL_DIRS" ]; then
      impl_found=$(grep -r "type.*struct" $REPO_IMPL_DIRS 2>/dev/null | grep -i "$interface_name" || true)
    fi
    if [ -z "$impl_found" ]; then
      ERRORS+=("MISSING_IMPL")
      DETAILS+=("No implementation found for $interface_name from $interface_file")
      HINTS+=("Create implementation in services/<svc>/repository/ that satisfies $interface_name")
    fi
  done
done

# --- Rule 4: UseCase files have corresponding tests ---
for usecase_file in $(find services/ -path "*/usecase/*.go" ! -name "*_test.go" 2>/dev/null); do
  test_file="${usecase_file%.go}_test.go"
  if [ ! -f "$test_file" ]; then
    ERRORS+=("MISSING_USECASE_TEST")
    DETAILS+=("UseCase test missing: $usecase_file")
    HINTS+=("Create $test_file with mock repositories and business logic tests")
  fi
done

# --- Rule 5: Banking-specific: Check decimal.Decimal usage for money ---
# Detect float64 in money-related fields
money_float_usage=""
if [ -n "$DOMAIN_DIRS" ]; then
  money_float_usage=$(grep -rn "Amount.*float64\|Balance.*float64\|Price.*float64" $DOMAIN_DIRS 2>/dev/null || true)
fi
if [ -n "$money_float_usage" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("FINANCIAL_PRECISION_ERROR")
      DETAILS+=("float64 used for money at $FILE:$LINENO")
      HINTS+=("CRITICAL: Use decimal.Decimal from github.com/shopspring/decimal for all money calculations")
    fi
  done <<< "$money_float_usage"
fi

# --- Rule 6: Audit logging for transactions ---
# Check if transaction-related usecases have audit logging
for usecase_file in $(find services/ -path "*/usecase/*" \( -name "*transfer*.go" -o -name "*transaction*.go" \) 2>/dev/null); do
  if ! grep -q "audit\|AuditLog\|audit.Log" "$usecase_file" 2>/dev/null; then
    ERRORS+=("MISSING_AUDIT_LOG")
    DETAILS+=("Transaction usecase $usecase_file missing audit logging")
    HINTS+=("Add audit.Log() call for compliance (CNBV requirement)")
  fi
done

# Export results
echo "COMPLETENESS_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
