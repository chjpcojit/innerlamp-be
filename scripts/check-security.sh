#!/bin/bash
# check-security.sh
# Checks: gosec, banking-specific security rules
# Banking compliance: PCI DSS, CNBV requirements

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# --- Check if gosec is installed ---
if ! command -v gosec &> /dev/null; then
  ERRORS+=("SECURITY_TOOL_MISSING")
  DETAILS+=("gosec not installed - cannot run security checks")
  HINTS+=("Install gosec: go install github.com/securego/gosec/v2/cmd/gosec@latest")
  echo "SECURITY_ERRORS=${#ERRORS[@]}"
  echo "ERR_TYPE_0=${ERRORS[0]}"
  echo "ERR_DETAIL_0=${DETAILS[0]}"
  echo "ERR_HINT_0=${HINTS[0]}"
  exit 0
fi

# --- Run gosec ---
GOSEC_OUT=$(gosec -fmt json ./... 2>/dev/null || true)
ISSUES=$(echo "$GOSEC_OUT" | jq -r '.Issues[]? | "\(.file):\(.line) [\(.severity)] \(.rule_id): \(.details)"' 2>/dev/null || true)

if [ -n "$ISSUES" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      ERRORS+=("GOSEC_VIOLATION")
      DETAILS+=("$line")
      HINTS+=("Review gosec output and fix security issue")
    fi
  done <<< "$ISSUES"
fi

# --- Banking-specific checks ---

# ── Dynamic path discovery (works for services/<svc>/ layout) ─────────────────
DOMAIN_DIRS=$(find services/ -type d -name "domain" 2>/dev/null | tr '\n' ' ')
USECASE_DIRS=$(find services/ -type d -name "usecase" 2>/dev/null | tr '\n' ' ')

# 1. Hardcoded secrets/passwords
# Exclusions:
#   - 'password == ""'  : empty-string guard checks (comparison, not assignment)
#   - 'password != ""'  : same pattern, inverse comparison
HARDCODED=$(grep -rn "password.*=.*\"" --include="*.go" services/ 2>/dev/null \
  | grep -v "_test.go" \
  | grep -v 'password\s*[!=]=\s*""' \
  || true)
if [ -n "$HARDCODED" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      ERRORS+=("HARDCODED_SECRET")
      DETAILS+=("Potential hardcoded credential: $line")
      HINTS+=("Use pkg/secrets/ to retrieve secrets from GCP Secret Manager")
    fi
  done <<< "$HARDCODED"
fi

# 2. SQL injection risk (raw SQL strings)
SQL_RISK=$(grep -rn "db.Exec.*+\|db.Query.*+\|fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT" services/ 2>/dev/null || true)
if [ -n "$SQL_RISK" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      ERRORS+=("SQL_INJECTION_RISK")
      DETAILS+=("Potential SQL injection: $line")
      HINTS+=("Use parameterized queries with $1, $2, etc. NEVER concatenate SQL strings")
    fi
  done <<< "$SQL_RISK"
fi

# 3. Card data storage (PCI DSS violation)
CARD_STORAGE=""
if [ -n "$DOMAIN_DIRS" ]; then
  CARD_STORAGE=$(grep -rn "CardNumber\|CVV\|card_number\|cvv" $DOMAIN_DIRS 2>/dev/null | grep -v "_test.go" || true)
fi
if [ -n "$CARD_STORAGE" ]; then
  while IFS= read -r line; do
    if [[ ! "$line" =~ "Last4\|last_4" ]]; then
      ERRORS+=("PCI_DSS_VIOLATION")
      DETAILS+=("Potential card data storage: $line")
      HINTS+=("PCI DSS: Never store full card numbers or CVV. Use Stripe tokenization. Store last 4 digits only.")
    fi
  done <<< "$CARD_STORAGE"
fi

# 4. Missing encryption for sensitive data
SENSITIVE_FIELDS=""
if [ -n "$DOMAIN_DIRS" ]; then
  SENSITIVE_FIELDS=$(grep -rn "SSN\|CURP\|RFC\|DateOfBirth" $DOMAIN_DIRS 2>/dev/null || true)
fi
if [ -n "$SENSITIVE_FIELDS" ]; then
  # Check if encryption is used
  for file in $(echo "$SENSITIVE_FIELDS" | cut -d: -f1 | sort -u); do
    if ! grep -q "encryption\|Encrypt\|Decrypt" "$file" 2>/dev/null; then
      ERRORS+=("MISSING_ENCRYPTION")
      DETAILS+=("Sensitive PII in $file not encrypted")
      HINTS+=("Use pkg/encryption/ (AES-256-GCM) for SSN, CURP, RFC, date of birth")
    fi
  done
fi

# 5. Missing audit logging for privileged operations
PRIVILEGED_OPS=$(find services/ -path "*/usecase/*" \( -name "*admin*.go" -o -name "*delete*.go" \) 2>/dev/null)
for file in $PRIVILEGED_OPS; do
  if [ ! -f "$file" ]; then continue; fi
  if ! grep -q "audit\|AuditLog" "$file" 2>/dev/null; then
    ERRORS+=("MISSING_AUDIT_PRIVILEGED")
    DETAILS+=("Privileged operation in $file missing audit log")
    HINTS+=("CNBV compliance: All admin/delete operations must be audited")
  fi
done

# 6. JWT secret hardcoding
JWT_SECRET=$(grep -rn "jwtSecret\|JWT_SECRET.*=" services/ 2>/dev/null | grep -v "os.Getenv\|secrets." || true)
if [ -n "$JWT_SECRET" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      ERRORS+=("JWT_SECRET_HARDCODED")
      DETAILS+=("JWT secret not from secure source: $line")
      HINTS+=("Retrieve JWT secret from GCP Secret Manager via pkg/secrets/")
    fi
  done <<< "$JWT_SECRET"
fi

# Export results
echo "SECURITY_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
