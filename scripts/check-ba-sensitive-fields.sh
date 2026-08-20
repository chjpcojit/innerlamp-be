#!/bin/bash
# check-ba-sensitive-fields.sh
# Checks: PII and credential fields in BA schema have documented security measures
#
# WHY: CNBV + PCI DSS + OWASP require that:
#   - Credential fields (password, pin, otp) must NEVER be stored in plaintext
#   - Every *_hash field must document which algorithm is used (bcrypt, AES-256-GCM...)
#   - Fields with "unique" constraint must have a BR that handles the duplicate scenario
#     (otherwise the use case silently crashes on duplicate key errors in production)
#
# RULES:
#   Rule 1 — Every field named *_hash must document its algorithm in the description
#             Accepted: bcrypt, AES-256, AES-256-GCM, AES, PBKDF2, Argon2, encrypted, hashed
#   Rule 2 — Credential fields (password, pin, otp_code, secret) that are persisted
#             must have a corresponding *_hash counterpart in the schema
#             (otp_code → otp_hash, password → password_hash, pin → pin_hash)
#   Rule 3 — Fields with "unique" constraint must have at least one BR that handles
#             the already-exists / duplicate scenario
#
# Expected format in BA schema:
#   Entities:        `field_name: type` — description text
#   *_hash fields:   `pin_hash: string` — Bcrypt hash (cost factor 12)
#   unique fields:   `curp: string (18 chars, unique)` — ...
#
# Usage: ./check-ba-sensitive-fields.sh docs/analysis/feature/schema.md
#
# Referenced by: run-all.sh [23/23]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

SCHEMA_FILE="${1:-}"

if [ -z "$SCHEMA_FILE" ]; then
  echo "SENSITIVE_ERRORS=0"
  echo "SENSITIVE_SKIPPED=no schema file provided"
  exit 0
fi

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "SENSITIVE_ERRORS=1"
  echo "ERR_TYPE_0=SCHEMA_NOT_FOUND"
  echo "ERR_DETAIL_0=Schema file not found: $SCHEMA_FILE"
  echo "ERR_HINT_0=Run BA Parser agent first to generate the schema file"
  exit 0
fi

# ── Rule 1: *_hash fields must document their algorithm ───────────────────────
# Scan lines that contain a `*_hash` field definition
while IFS= read -r line; do
  field_name=$(echo "$line" | grep -oE '`[a-z][a-z0-9_]*_hash[^`]*`' | head -1 \
    | tr -d '`' | cut -d: -f1 || true)
  [ -z "$field_name" ] && continue

  # Description is the part after the field definition (after — or :)
  desc=$(echo "$line" | sed 's/.*`[^`]*`[[:space:]]*//')

  if ! echo "$desc" | grep -qiE 'bcrypt|AES-256|AES-256-GCM|AES|PBKDF2|[Aa]rgon2|[Ee]ncrypt|[Hh]ashed|[Ss]alted'; then
    ERRORS+=("HASH_FIELD_MISSING_ALGORITHM")
    DETAILS+=("Field '$field_name' is a hash/encrypted field but its description does not document the algorithm in $SCHEMA_FILE")
    HINTS+=("Document the algorithm next to the field, e.g.: '— Bcrypt hash (cost factor 12)' or '— AES-256-GCM encrypted'. Required for CNBV security audit.")
  fi
done < <(grep -E '`[a-z][a-z0-9_]*_hash[^`]*`' "$SCHEMA_FILE" 2>/dev/null || true)

# ── Rule 2: Credential fields must have a *_hash counterpart ─────────────────
# Covers: password, pin, otp_code, secret (when persisted as a field definition)
CRED_FIELDS='`password:|`pin:|`otp_code:|`secret:'

while IFS= read -r line; do
  field_name=$(echo "$line" | grep -oE '`(password|pin|otp_code|secret):' | head -1 \
    | tr -d '`' | tr -d ':' || true)
  [ -z "$field_name" ] && continue

  # Skip if this field IS already a hash variant (e.g. `password_hash:`)
  if echo "$field_name" | grep -qE '_hash$'; then continue; fi

  # Derive expected hash field: otp_code → otp_hash, others → {name}_hash
  if [ "$field_name" = "otp_code" ]; then
    expected_hash="otp_hash"
  else
    expected_hash="${field_name}_hash"
  fi

  # Check that the hash counterpart field exists anywhere in the schema
  if ! grep -qE "\`${expected_hash}:" "$SCHEMA_FILE" 2>/dev/null; then
    ERRORS+=("CREDENTIAL_STORED_WITHOUT_HASH")
    DETAILS+=("Field '$field_name' stores a credential but '$expected_hash' counterpart is not documented in $SCHEMA_FILE")
    HINTS+=("Add field '$expected_hash' with algorithm documentation (e.g., Bcrypt cost 12), OR explicitly mark '$field_name' as transient/not-persisted. CNBV + PCI DSS: credentials must never be stored in plaintext.")
  fi
done < <(grep -E "$CRED_FIELDS" "$SCHEMA_FILE" 2>/dev/null || true)

# ── Rule 3: Unique-constrained fields must have a BR for duplicate handling ───
# Match fields that contain the word "unique" in their description
SEEN_UNIQUE_FIELDS=":"  # colon-delimited list of already-reported fields (dedup)

while IFS= read -r line; do
  field_name=$(echo "$line" | grep -oE '`[a-z][a-z0-9_]+:' | head -1 \
    | tr -d '`' | tr -d ':' || true)
  [ -z "$field_name" ] && continue

  # Skip auto-generated primary key fields (always unique by definition, no user-facing duplicate scenario)
  [ "$field_name" = "id" ] && continue

  # Deduplicate — only report the same field name once
  if echo "$SEEN_UNIQUE_FIELDS" | grep -q ":${field_name}:"; then continue; fi
  SEEN_UNIQUE_FIELDS="${SEEN_UNIQUE_FIELDS}${field_name}:"

  # Also search with underscores replaced by spaces (handles "VOTER ID" for voter_id)
  field_humanized=$(echo "$field_name" | tr '_' ' ')

  # Look for a BR that handles the duplicate/already-exists scenario for this field
  # Format in schema: **BR001:** (colon inside bold markers)
  has_br=$(grep -iE '\*\*BR[0-9]+:\*\*.*'"$field_name"'.*(already exist|duplicate|exists in system|already registered|already in use)' \
    "$SCHEMA_FILE" 2>/dev/null || true)
  if [ -z "$has_br" ] && [ "$field_humanized" != "$field_name" ]; then
    has_br=$(grep -iE '\*\*BR[0-9]+:\*\*.*'"$field_humanized"'.*(already exist|duplicate|exists in system|already registered|already in use)' \
      "$SCHEMA_FILE" 2>/dev/null || true)
  fi

  if [ -z "$has_br" ]; then
    ERRORS+=("UNIQUE_FIELD_NO_DUPLICATE_BR")
    DETAILS+=("Field '$field_name' has 'unique' constraint but no Business Rule handles the duplicate/already-exists scenario in $SCHEMA_FILE")
    field_upper=$(echo "$field_name" | tr '[:lower:]' '[:upper:]')
    HINTS+=("Add a Business Rule: '**BRxxx:** If $field_name already exists in system → Block with error ${field_upper}_ALREADY_EXISTS'")
  fi
done < <(grep -E '`[a-z][a-z0-9_]+:.*\bunique\b' "$SCHEMA_FILE" 2>/dev/null || true)

# ── Output ────────────────────────────────────────────────────────────────────
echo "SENSITIVE_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_${i}=${ERRORS[$i]}"
  echo "ERR_DETAIL_${i}=${DETAILS[$i]}"
  echo "ERR_HINT_${i}=${HINTS[$i]}"
done
if [ ${#ERRORS[@]} -eq 0 ]; then
  echo "SENSITIVE_OK=Sensitive field security documentation check passed"
fi
