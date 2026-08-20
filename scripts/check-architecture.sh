#!/bin/bash
# check-architecture.sh
# Checks: layer import violations for Clean Architecture
# Output: structured violations

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

MODULE=$(go list -m 2>/dev/null || echo "unknown")

# ── Dynamic path discovery (works for services/<svc>/ layout) ─────────────────
USECASE_DIRS=$(find services -type d -name "usecase" 2>/dev/null | tr '\n' ' ')
DOMAIN_DIRS=$(find services -type d -name "domain" 2>/dev/null | tr '\n' ' ')

# --- Rule 1: usecase must not import infrastructure ---
INFRA_IN_USECASE=""
if [ -n "$USECASE_DIRS" ]; then
  INFRA_IN_USECASE=$(grep -rn "\"${MODULE}/.*infrastructure" $USECASE_DIRS 2>/dev/null || true)
fi
if [ -n "$INFRA_IN_USECASE" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      IMPORT=$(echo "$line" | cut -d: -f3-)
      ERRORS+=("ARCH_VIOLATION")
      DETAILS+=("UseCase imports Infrastructure | File: $FILE:$LINENO | Import: $IMPORT")
      HINTS+=("Replace concrete infrastructure type with interface defined in domain/repository/")
    fi
  done <<< "$INFRA_IN_USECASE"
fi

# --- Rule 2: domain must not import internal packages ---
# Excludes _test.go — a test file importing its own package is valid Go
INTERNAL_IN_DOMAIN=""
if [ -n "$DOMAIN_DIRS" ]; then
  INTERNAL_IN_DOMAIN=$(grep -rn "\"${MODULE}/services/" $DOMAIN_DIRS \
    --include="*.go" 2>/dev/null \
    | grep -v "_test.go" || true)
fi
if [ -n "$INTERNAL_IN_DOMAIN" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      IMPORT=$(echo "$line" | cut -d: -f3-)
      ERRORS+=("ARCH_VIOLATION")
      DETAILS+=("Domain imports internal package | File: $FILE:$LINENO | Import: $IMPORT")
      HINTS+=("Domain layer must only import standard library packages")
    fi
  done <<< "$INTERNAL_IN_DOMAIN"
fi

# --- Rule 3: usecase must not use DB/SQL packages directly ---
DB_IN_USECASE=""
if [ -n "$USECASE_DIRS" ]; then
  DB_IN_USECASE=$(grep -rn \
    -e '"database/sql"' \
    -e '"github.com/jackc/pgx' \
    -e '"github.com/lib/pq"' \
    $USECASE_DIRS 2>/dev/null || true)
fi
if [ -n "$DB_IN_USECASE" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      IMPORT=$(echo "$line" | cut -d: -f3-)
      ERRORS+=("ARCH_VIOLATION")
      DETAILS+=("DB package in UseCase | File: $FILE:$LINENO | Import: $IMPORT")
      HINTS+=("UseCase must not depend on DB. Use repository interface from domain/ instead")
    fi
  done <<< "$DB_IN_USECASE"
fi

# --- Rule 4: domain must not use DB/SQL packages ---
DB_IN_DOMAIN=""
if [ -n "$DOMAIN_DIRS" ]; then
  DB_IN_DOMAIN=$(grep -rn \
    -e '"database/sql"' \
    -e '"github.com/jackc/pgx' \
    -e '"github.com/lib/pq"' \
    $DOMAIN_DIRS 2>/dev/null || true)
fi
if [ -n "$DB_IN_DOMAIN" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      IMPORT=$(echo "$line" | cut -d: -f3-)
      ERRORS+=("ARCH_VIOLATION")
      DETAILS+=("DB package in Domain | File: $FILE:$LINENO | Import: $IMPORT")
      HINTS+=("Domain layer must be pure Go with no infrastructure dependencies")
    fi
  done <<< "$DB_IN_DOMAIN"
fi

# --- Rule 5: usecase must not import net/http ---
HTTP_IN_USECASE=""
if [ -n "$USECASE_DIRS" ]; then
  HTTP_IN_USECASE=$(grep -rn '"net/http"' $USECASE_DIRS 2>/dev/null || true)
fi
if [ -n "$HTTP_IN_USECASE" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("ARCH_VIOLATION")
      DETAILS+=("HTTP package in UseCase | File: $FILE:$LINENO")
      HINTS+=("HTTP concerns belong in interface/ layer only")
    fi
  done <<< "$HTTP_IN_USECASE"
fi

# Export results
echo "ARCH_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done