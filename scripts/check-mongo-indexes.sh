#!/bin/bash
# check-mongo-indexes.sh
# Checks: MongoDB fields used in queries must have indexes in migration scripts
# Banking compliance: SLA requires P50 < 100ms — full collection scan violates this
# Referenced by: run-all.sh [15/16]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

MONGO_MIGRATION_DIR="migrations/mongodb"

# ── Guard: skip if no MongoDB migrations directory ────────────────────────────
if [ ! -d "$MONGO_MIGRATION_DIR" ]; then
  echo "MONGO_IDX_ERRORS=0"
  echo "MONGO_IDX_SKIPPED=no migrations/mongodb directory found"
  exit 0
fi

# ── Collect all indexes defined in migration scripts ─────────────────────────
# Pattern: createIndex({ field_name: 1 }) or createIndex({ field: -1 })
# Also: ensureIndex({ field: 1 })
INDEXED_FIELDS_FILE=$(mktemp)
trap 'rm -f "$INDEXED_FIELDS_FILE"' EXIT

# Extract field names from createIndex calls in .js migration files
for js_file in $(find "$MONGO_MIGRATION_DIR" -name "*.js" | sort); do
  grep -oE "\{[[:space:]]*[a-z_]+[[:space:]]*:" "$js_file" 2>/dev/null \
    | grep -oE "[a-z_]+" \
    | grep -v "^db$\|^collection$" >> "$INDEXED_FIELDS_FILE" || true
done

# ── Rule 1: Fields used in .Find() / .FindOne() queries must have an index ───
REPO_DIRS=$(find services/ -type d -name "repository" ! -path "*/domain/*" 2>/dev/null | tr '\n' ' ')

if [ -n "$REPO_DIRS" ]; then
  for repo_file in $(find $REPO_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
    # Only check files that use mongo driver patterns
    if ! grep -q "bson\.M\|bson\.D\|mongo\.Collection\|FindOne\|Find(" "$repo_file" 2>/dev/null; then
      continue
    fi

    # Extract field names from bson.M{"field_name": ...} query patterns
    QUERY_FIELDS=$(grep -oE 'bson\.(M|D)\{[^}]*"[a-z_]+"' "$repo_file" 2>/dev/null \
      | grep -oE '"[a-z_]+"' \
      | tr -d '"' \
      | sort -u || true)

    while IFS= read -r field; do
      if [ -z "$field" ]; then continue; fi

      # Skip common non-index fields
      case "$field" in
        _id|id) continue ;;
      esac

      # Check if this field has an index in migration files
      if ! grep -qF "$field" "$INDEXED_FIELDS_FILE" 2>/dev/null; then
        ERRORS+=("MISSING_MONGO_INDEX")
        DETAILS+=("MongoDB query field '$field' has no index defined in migrations/mongodb/ | File: $repo_file")
        HINTS+=("Add to migrations/mongodb/: db.getCollection('your_collection').createIndex({ $field: 1 }, { background: true }). Full collection scan violates P50 < 100ms SLA.")
      fi
    done <<< "$QUERY_FIELDS"
  done
fi

# ── Rule 2: TTL indexes for session/OTP collections ──────────────────────────
# Session and OTP documents must expire — check TTL index exists
TTL_PATTERN="expireAfterSeconds\|TTL\|ttl"

SESSION_REPO=$(find services/ -path "*/repository/*session*" -name "*.go" 2>/dev/null | head -1)
OTP_REPO=$(find services/ -path "*/repository/*otp*" -name "*.go" 2>/dev/null | head -1)

if [ -n "$SESSION_REPO" ]; then
  # Check if there's a TTL index for sessions in MongoDB migrations
  if ! grep -q "expireAfterSeconds" "$MONGO_MIGRATION_DIR"/*.js 2>/dev/null; then
    ERRORS+=("MISSING_MONGO_TTL_INDEX")
    DETAILS+=("No TTL index found for session/OTP collection in MongoDB migrations")
    HINTS+=("Add TTL index to auto-expire sessions: db.sessions.createIndex({ 'created_at': 1 }, { expireAfterSeconds: 900 }). Manual cleanup is unreliable at scale.")
  fi
fi

# ── Rule 3: Compound index for common multi-field queries ─────────────────────
# Detect queries filtering on 2+ fields — single field indexes are suboptimal
for repo_file in $(find services/ -path "*/repository/*.go" ! -name "*_test.go" 2>/dev/null); do
  if ! grep -q "bson\.M\|bson\.D" "$repo_file" 2>/dev/null; then
    continue
  fi

  # Count bson.M fields within a single Find call — rough heuristic
  MULTI_FIELD=$(grep -E 'bson\.M\{.*"[a-z_]+".*"[a-z_]+"' "$repo_file" 2>/dev/null || true)
  if [ -n "$MULTI_FIELD" ]; then
    LINECOUNT=$(echo "$MULTI_FIELD" | wc -l | tr -d ' ')
    if [ "$LINECOUNT" -gt 0 ]; then
      ERRORS+=("POTENTIAL_MISSING_COMPOUND_INDEX")
      DETAILS+=("Multi-field MongoDB query detected — consider compound index | File: $repo_file")
      HINTS+=("Review query patterns and add compound indexes: db.collection.createIndex({ field1: 1, field2: 1 }). Compound indexes are more efficient than multiple single-field indexes for multi-field queries.")
    fi
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "MONGO_IDX_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
