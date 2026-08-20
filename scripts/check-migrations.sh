#!/bin/bash
# check-migrations.sh
# Checks: database migration safety for banking production deployments
# Banking compliance: CNBV 7-year data retention — irreversible data loss is a violation
# Protects against: DROP TABLE without review, missing down files, sequential numbering gaps

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

MIGRATIONS_DIR="migrations"

# Guard: if no migrations folder, nothing to check
if [ ! -d "$MIGRATIONS_DIR" ]; then
  echo "MIGRATION_ERRORS=0"
  echo "MIGRATION_SKIPPED=no migrations/ directory found"
  exit 0
fi

# ── 1. Every *.up.sql must have a matching *.down.sql ─────────────────────────
for up_file in $(find "$MIGRATIONS_DIR" -maxdepth 1 -name "*.up.sql" | sort); do
  down_file="${up_file/.up.sql/.down.sql}"
  if [ ! -f "$down_file" ]; then
    ERRORS+=("MISSING_DOWN_MIGRATION")
    DETAILS+=("No rollback file for: $up_file")
    HINTS+=("Create $down_file that reverses every operation in $up_file. Required for safe production deployment.")
  fi
done

# ── 2. Warning: destructive DDL in up migrations ─────────────────────────────
# DROP TABLE / TRUNCATE in an up migration can destroy CNBV-required 7-year data
DESTRUCTIVE_PATTERN="^\s*DROP TABLE\|^\s*TRUNCATE TABLE\|^\s*DROP COLUMN\|^\s*ALTER TABLE.*DROP"

for up_file in $(find "$MIGRATIONS_DIR" -maxdepth 1 -name "*.up.sql" | sort); do
  DESTRUCTIVE_OPS=$(grep -in "$DESTRUCTIVE_PATTERN" "$up_file" 2>/dev/null || true)
  if [ -n "$DESTRUCTIVE_OPS" ]; then
    while IFS= read -r line; do
      if [ -n "$line" ]; then
        SQL_LINENO=$(echo "$line" | cut -d: -f1)
        STATEMENT=$(echo "$line" | cut -d: -f2-)
        ERRORS+=("DESTRUCTIVE_MIGRATION")
        DETAILS+=("Destructive DDL in up migration | File: $up_file:$SQL_LINENO | Statement: $STATEMENT")
        HINTS+=("WARNING: This permanently deletes data. CNBV requires 7-year retention. Ensure data is archived before dropping. Add '-- REVIEWED: <senior-dev> <date>' comment above the statement to suppress this check.")
      fi
    done <<< "$DESTRUCTIVE_OPS"
  fi
done

# Suppress reviewed destructive ops (developer added explicit review comment)
# Re-run and filter out lines preceded by -- REVIEWED:
TOTAL_BEFORE=${#ERRORS[@]}
FILTERED_ERRORS=()
FILTERED_DETAILS=()
FILTERED_HINTS=()

for i in "${!ERRORS[@]}"; do
  if [ "${ERRORS[$i]}" = "DESTRUCTIVE_MIGRATION" ]; then
    # Extract file from detail
    FILE=$(echo "${DETAILS[$i]}" | grep -o "migrations/[^ |:]*\.sql" | head -1 || true)
    SQL_LINENO=$(echo "${DETAILS[$i]}" | grep -oE "\.sql:[0-9]+" | grep -oE "[0-9]+" | head -1 || true)

    if [ -n "$FILE" ] && [ -n "$SQL_LINENO" ]; then
      PREV_LINE=$((SQL_LINENO - 1))
      REVIEW_COMMENT=$(sed -n "${PREV_LINE}p" "$FILE" 2>/dev/null || true)
      if echo "$REVIEW_COMMENT" | grep -qF -- '-- REVIEWED:'; then
        continue  # Suppress — reviewed by senior dev
      fi
    fi
  fi
  FILTERED_ERRORS+=("${ERRORS[$i]}")
  FILTERED_DETAILS+=("${DETAILS[$i]}")
  FILTERED_HINTS+=("${HINTS[$i]}")
done

ERRORS=("${FILTERED_ERRORS[@]+"${FILTERED_ERRORS[@]}"}")
DETAILS=("${FILTERED_DETAILS[@]+"${FILTERED_DETAILS[@]}"}")
HINTS=("${FILTERED_HINTS[@]+"${FILTERED_HINTS[@]}"}")

# ── 3. Sequential numbering — no gaps, no duplicates ─────────────────────────
# ── 3. Sequential numbering — no gaps, no duplicates ─────────────────────────
# bash 3.x compatible: use temp file instead of associative array
SEEN_NUMBERS_FILE=$(mktemp)
trap 'rm -f "$SEEN_NUMBERS_FILE"' EXIT

for up_file in $(find "$MIGRATIONS_DIR" -maxdepth 1 -name "*.up.sql" | sort); do
  filename=$(basename "$up_file")
  number=$(echo "$filename" | grep -o "^[0-9]*" || true)

  if [ -z "$number" ]; then
    ERRORS+=("INVALID_MIGRATION_NAME")
    DETAILS+=("Migration file does not start with a number: $up_file")
    HINTS+=("Rename to format: NNN_description.up.sql  e.g. 013_add_users_index.up.sql")
    continue
  fi

  # Check for duplicate number
  if grep -qF "NUM=$number " "$SEEN_NUMBERS_FILE" 2>/dev/null; then
    PREV_FILE=$(grep "NUM=$number " "$SEEN_NUMBERS_FILE" | cut -d' ' -f2- | head -1)
    ERRORS+=("DUPLICATE_MIGRATION_NUMBER")
    DETAILS+=("Duplicate migration number $number: $up_file conflicts with $PREV_FILE")
    HINTS+=("Rename one migration to the next available number. Duplicates cause non-deterministic execution.")
  else
    echo "NUM=$number $up_file" >> "$SEEN_NUMBERS_FILE"
  fi
done

# Check for gaps using sorted list
SORTED_NUMBERS=($(grep -o 'NUM=[0-9]*' "$SEEN_NUMBERS_FILE" | cut -d= -f2 | sort -n))
if [ ${#SORTED_NUMBERS[@]} -gt 1 ]; then
  for idx in $(seq 1 $((${#SORTED_NUMBERS[@]} - 1))); do
    prev=${SORTED_NUMBERS[$((idx - 1))]}
    curr=${SORTED_NUMBERS[$idx]}
    prev_stripped=$((10#$prev))
    curr_stripped=$((10#$curr))
    expected=$((prev_stripped + 1))
    if [ "$curr_stripped" -ne "$expected" ] && [ "$curr_stripped" -ne "$prev_stripped" ]; then
      ERRORS+=("MIGRATION_SEQUENCE_GAP")
      DETAILS+=("Gap in migration sequence: found $prev then $curr (missing $expected)")
      HINTS+=("Gaps in migration numbers cause confusion. Fill the gap or document why it was intentionally skipped.")
    fi
  done
fi

# ── 4. Foreign key columns must have an index in the same migration ───────────
for up_file in $(find "$MIGRATIONS_DIR" -maxdepth 1 -name "*.up.sql" | sort); do
  # Find FK columns defined in this migration
  FK_COLS=$(grep -in "REFERENCES\|FOREIGN KEY" "$up_file" 2>/dev/null \
    | grep -oE "[a-z_]+_id\b" | sort -u || true)

  for col in $FK_COLS; do
    # Check if an index on this column exists anywhere in the same file
    if ! grep -qi "INDEX.*$col\|$col.*INDEX\|CREATE INDEX.*$col" "$up_file" 2>/dev/null; then
      ERRORS+=("MISSING_FK_INDEX")
      DETAILS+=("Foreign key column '$col' in $up_file has no index")
      HINTS+=("Add: CREATE INDEX idx_<table>_${col} ON <table>(${col}); — unindexed FKs cause full table scans on JOINs")
    fi
  done
done

# ── Export results ────────────────────────────────────────────────────────────
echo "MIGRATION_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
