#!/bin/bash
# check-ba-br-err-coverage.sh
# Checks: bidirectional BR ↔ ERR coverage in BA schema
#
# WHY: BA schemas must maintain consistent BR/ERR relationships to ensure:
#   - Every blocking Business Rule has a corresponding Error Case (user-facing feedback)
#   - Every Error Case references a real Business Rule (no phantom errors)
#   - No numbering gaps in BR/ERR sequences (traceability for CNBV compliance audit)
#
# RULES:
#   Rule 1 — Every ERR that says "(maps to BRxxx)" must reference a BR that exists
#             in the Business Rules section
#   Rule 2 — Every BR that describes a blocking/rejection outcome must have at least
#             one ERR entry that maps to it
#   Rule 3 — BR numbers must be sequential with no gaps
#             (gaps break audit trail references in CNBV documentation)
#   Rule 4 — ERR numbers must be sequential with no gaps
#
# Expected format in BA schema:
#   Business Rules: **BR001:** text text text
#   Error Cases:    **ERR001:** `ERROR_CODE` → HTTP 4xx (maps to BR001)
#
# Usage: ./check-ba-br-err-coverage.sh docs/analysis/feature/schema.md
#
# Referenced by: run-all.sh [22/23]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

SCHEMA_FILE="${1:-}"

if [ -z "$SCHEMA_FILE" ]; then
  echo "BRERR_ERRORS=0"
  echo "BRERR_SKIPPED=no schema file provided"
  exit 0
fi

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "BRERR_ERRORS=1"
  echo "ERR_TYPE_0=SCHEMA_NOT_FOUND"
  echo "ERR_DETAIL_0=Schema file not found: $SCHEMA_FILE"
  echo "ERR_HINT_0=Run BA Parser agent first to generate the schema file"
  exit 0
fi

# ── Extract BR and ERR numbers ────────────────────────────────────────────────
# Matches **BR001:** format (colon inside the bold markers)
BR_NUMBERS=$(grep -oE '\*\*BR[0-9]+:\*\*' "$SCHEMA_FILE" 2>/dev/null \
  | grep -oE '[0-9]+' | sed 's/^0*//' | sort -n | uniq || true)

ERR_NUMBERS=$(grep -oE '\*\*ERR[0-9]+:\*\*' "$SCHEMA_FILE" 2>/dev/null \
  | grep -oE '[0-9]+' | sed 's/^0*//' | sort -n | uniq || true)

if [ -z "$BR_NUMBERS" ]; then
  echo "BRERR_ERRORS=0"
  echo "BRERR_SKIPPED=no BR entries found in schema (expected format: **BR001:** text)"
  exit 0
fi

if [ -z "$ERR_NUMBERS" ]; then
  echo "BRERR_ERRORS=0"
  echo "BRERR_SKIPPED=no ERR entries found in schema (expected format: **ERR001:** text)"
  exit 0
fi

# ── Rule 1: ERR → BR reference validity ──────────────────────────────────────
# Each "maps to BRxxx" must reference a BR that actually exists
while IFS= read -r err_line; do
  [ -z "$err_line" ] && continue

  err_num=$(echo "$err_line" | grep -oE '\*\*ERR[0-9]+:\*\*' | grep -oE '[0-9]+' | head -1 || true)
  [ -z "$err_num" ] && continue
  err_padded=$(printf '%03d' "${err_num#0}")

  br_refs=$(echo "$err_line" | grep -oE '\bBR[0-9]+\b' || true)
  [ -z "$br_refs" ] && continue

  while IFS= read -r br_ref; do
    [ -z "$br_ref" ] && continue
    br_num=$(echo "$br_ref" | grep -oE '[0-9]+' | sed 's/^0*//')
    br_padded=$(printf '%03d' "${br_num}")
    if ! echo "$BR_NUMBERS" | grep -qx "$br_num" 2>/dev/null; then
      ERRORS+=("ERR_REFERENCES_NONEXISTENT_BR")
      DETAILS+=("ERR${err_padded} references BR${br_padded} which does not exist in Business Rules section")
      HINTS+=("Add BR${br_padded} to the Business Rules section, or fix the reference in ERR${err_padded}")
    fi
  done <<< "$br_refs"
done < <(grep -E '\*\*ERR[0-9]+\*\*.*maps to' "$SCHEMA_FILE" 2>/dev/null || true)

# ── Rule 2: BR → ERR coverage ─────────────────────────────────────────────────
# BRs with blocking/rejection outcomes must have at least one ERR mapping to them
BLOCKING_KEYWORDS='[Bb]lock|[Dd]isplay.*[Ee]rror|[Rr]eturn.*[Ee]rror|[Ll]imit exceeded|[Ee]xceed.*[Ll]imit|[Tt]oo many|[Rr]eject|[Ee]xpir|[Pp]revent|[Ff]ail|[Dd]isallow|[Ff]orbid'

while IFS= read -r n; do
  [ -z "$n" ] && continue
  br_padded=$(printf '%03d' "$n")

  br_line=$(grep -E "\*\*BR${br_padded}\*\*" "$SCHEMA_FILE" 2>/dev/null | head -1 || true)
  [ -z "$br_line" ] && continue

  # Only check BRs with user-facing blocking conditions
  if ! echo "$br_line" | grep -qE "$BLOCKING_KEYWORDS"; then continue; fi

  # Check if any ERR "maps to BR{n}"
  mapped=$(grep -E '\*\*ERR[0-9]+:\*\*.*maps to.*\bBR'"${br_padded}"'\b' "$SCHEMA_FILE" 2>/dev/null || true)
  if [ -z "$mapped" ]; then
    ERRORS+=("BR_HAS_NO_ERR_MAPPING")
    DETAILS+=("BR${br_padded} has a blocking/rejection condition but no ERR entry maps to it")
    HINTS+=("Add an Error Case: '- **ERRxxx:** \`ERROR_CODE\` → HTTP NNN (maps to BR${br_padded})'")
  fi
done <<< "$BR_NUMBERS"

# ── Rule 3: BR numbering gaps ─────────────────────────────────────────────────
prev=0
while IFS= read -r n; do
  [ -z "$n" ] && continue
  n_int=$((10#${n}))
  expected=$((prev + 1))
  if [ "$n_int" -gt "$expected" ]; then
    for gap in $(seq "$expected" "$((n_int - 1))"); do
      gap_padded=$(printf '%03d' "$gap")
      prev_padded=$(printf '%03d' "$prev")
      n_padded=$(printf '%03d' "$n_int")
      ERRORS+=("BR_NUMBER_GAP")
      DETAILS+=("BR numbering gap: BR${prev_padded} → BR${gap_padded} is missing → BR${n_padded} in $SCHEMA_FILE")
      HINTS+=("Add BR${gap_padded} or renumber BRs sequentially — gaps break CNBV audit trail traceability")
    done
  fi
  prev=$n_int
done <<< "$BR_NUMBERS"

# ── Rule 4: ERR numbering gaps ────────────────────────────────────────────────
prev=0
while IFS= read -r n; do
  [ -z "$n" ] && continue
  n_int=$((10#${n}))
  expected=$((prev + 1))
  if [ "$n_int" -gt "$expected" ]; then
    for gap in $(seq "$expected" "$((n_int - 1))"); do
      gap_padded=$(printf '%03d' "$gap")
      prev_padded=$(printf '%03d' "$prev")
      n_padded=$(printf '%03d' "$n_int")
      ERRORS+=("ERR_NUMBER_GAP")
      DETAILS+=("ERR numbering gap: ERR${prev_padded} → ERR${gap_padded} is missing → ERR${n_padded} in $SCHEMA_FILE")
      HINTS+=("Add ERR${gap_padded} or renumber ERRs sequentially — gaps break CNBV audit trail traceability")
    done
  fi
  prev=$n_int
done <<< "$ERR_NUMBERS"

# ── Output ────────────────────────────────────────────────────────────────────
echo "BRERR_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_${i}=${ERRORS[$i]}"
  echo "ERR_DETAIL_${i}=${DETAILS[$i]}"
  echo "ERR_HINT_${i}=${HINTS[$i]}"
done
if [ ${#ERRORS[@]} -eq 0 ]; then
  echo "BRERR_OK=BR-ERR bidirectional coverage check passed"
fi
