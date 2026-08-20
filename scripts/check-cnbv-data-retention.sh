#!/bin/bash
# check-cnbv-data-retention.sh
# Checks: CNBV 7-year data retention compliance (Article 52, LICSF Mexico)
#
# WHY: CNBV (Comisión Nacional Bancaria y de Valores) mandates that:
#   - Transaction records MUST be kept for MINIMUM 7 YEARS
#   - Account history must be preserved even after account closure
#   - Audit logs must be immutable and non-deletable
# Violations can result in regulatory fines, license revocation,
# and criminal liability for executives.
#
# RULES:
#   Rule 1 — No ON DELETE CASCADE on financial tables in migration files
#             (cascade deletes would wipe child records when parent deleted)
#   Rule 2 — audit_logs table must NOT have a TTL column or expiry in migrations
#   Rule 3 — Audit log repository must not expose a Delete() method
#   Rule 4 — No AUTO_DELETE, EXPIRY, TTL pragmas on transaction/audit collections
#             in MongoDB init scripts
#   Rule 5 — Transaction/audit tables must not have retention_days < 2555 (7 years)
#             if any such config exists
#   Rule 6 — migration down files for transaction/audit tables must only
#             reverse schema shape — not delete rows
#
# Referenced by: run-all.sh [21/21]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

SEVEN_YEARS_DAYS=2555  # 7 * 365

# ── Protected financial tables ─────────────────────────────────────────────────
FINANCIAL_TABLES="transactions\|audit_logs\|payments\|transfers\|ledger_entries\|account_statements\|onboarding_sessions"

# ── Rule 1: No ON DELETE CASCADE on financial tables ─────────────────────────
for sql_file in $(find migrations/ -name "*.up.sql" 2>/dev/null | sort); do
  # Find REFERENCES clauses with ON DELETE CASCADE on financial tables
  CASCADE_VIOLATIONS=$(grep -n \
    "REFERENCES.*ON DELETE CASCADE\|ON DELETE CASCADE" \
    "$sql_file" 2>/dev/null || true)

  if [ -n "$CASCADE_VIOLATIONS" ]; then
    # Check if the CASCADE is on a financial table context
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      # Look at surrounding context to see if it's a financial table
      SQL_LINENO=$(echo "$line" | cut -d: -f1)
      LINE_TEXT=$(echo "$line" | cut -d: -f2-)

      # Skip if CNBV_EXEMPT suppression comment is on the same line
      if echo "$LINE_TEXT" | grep -qF -- "-- CNBV_EXEMPT:"; then
        continue
      fi
      # Also check the preceding line
      PREV_SQL_LINENO=$((SQL_LINENO - 1))
      PREV_LINE_CONTENT=$(sed -n "${PREV_SQL_LINENO}p" "$sql_file" 2>/dev/null || true)
      if echo "$PREV_LINE_CONTENT" | grep -qF -- "-- CNBV_EXEMPT:"; then
        continue
      fi

      # Get the CREATE TABLE block that contains this CASCADE
      TABLE_CONTEXT=$(awk "NR>=($SQL_LINENO-30) && NR<=$SQL_LINENO" "$sql_file" \
        | grep -iE "CREATE TABLE|$FINANCIAL_TABLES" | head -1 || true)

      if echo "$TABLE_CONTEXT" | grep -iqE "$FINANCIAL_TABLES"; then
        ERRORS+=("CASCADE_DELETE_FINANCIAL_TABLE")
        DETAILS+=("ON DELETE CASCADE on financial/audit table — violates 7-year CNBV retention | File: $sql_file:$SQL_LINENO")
        HINTS+=("Replace ON DELETE CASCADE with ON DELETE RESTRICT or ON DELETE SET NULL. Financial records must outlive parent records. CNBV Article 52: all transaction data must be preserved for 7 years minimum.")
      else
        # Generic cascade warning even for non-financial tables — soft warning
        ERRORS+=("CASCADE_DELETE_WARNING")
        DETAILS+=("ON DELETE CASCADE found — verify affected table is not financial | File: $sql_file:$SQL_LINENO")
        HINTS+=("Review this cascade delete. If the table stores financial or user data, replace with RESTRICT. Add comment: -- CNBV_EXEMPT: <table_name> non-financial, cascade approved by <name>")
      fi
    done <<< "$CASCADE_VIOLATIONS"
  fi
done

# ── Rule 2: audit_logs must not have TTL/expiry column in migrations ──────────
for sql_file in $(find migrations/ -name "*.up.sql" 2>/dev/null | sort); do
  # Detect TTL-like column names in audit_logs table context
  AUDIT_TTL=$(awk '/CREATE TABLE.*audit_log/,/\);/' "$sql_file" \
    | grep -iE 'expires_at|expiry|ttl|retention_until|auto_delete' || true)

  if [ -n "$AUDIT_TTL" ]; then
    ERRORS+=("AUDIT_LOG_TTL_COLUMN")
    DETAILS+=("audit_logs table has a TTL/expiry column in $sql_file — audit records must be permanent")
    HINTS+=("Remove the TTL column from audit_logs. Audit records must never expire automatically. CNBV requires audit trail for 7 years. If storage is a concern, use table partitioning and cold storage (GCS archive) for old records.")
  fi
done

# ── Rule 3: Audit log repository must NOT expose Delete() ────────────────────
AUDIT_REPO_DELETE=$(grep -rn \
  -e 'func.*AuditLog.*Delete\b' \
  -e 'func.*DeleteAudit\b' \
  -e 'func.*PurgeAudit\b' \
  --include="*.go" \
  services/ \
  2>/dev/null \
  | grep -v "_test.go" || true)

if [ -n "$AUDIT_REPO_DELETE" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    FILE=$(echo "$line" | cut -d: -f1)
    SQL_LINENO=$(echo "$line" | cut -d: -f2)
    ERRORS+=("AUDIT_LOG_DELETABLE")
    DETAILS+=("Audit log repository exposes a Delete/Purge method | File: $FILE:$SQL_LINENO")
    HINTS+=("Remove Delete/Purge from AuditLogRepository. Audit records are IMMUTABLE. If needed for compliance archival, implement an Archive() that moves records to cold storage without deleting from primary DB.")
  done <<< "$AUDIT_REPO_DELETE"
fi

# ── Rule 4: MongoDB init scripts must not set TTL on financial collections ────
for js_file in $(find migrations/mongodb/ scripts/ -name "*.js" 2>/dev/null | sort); do
  TTL_ON_FINANCIAL=$(grep -n "expireAfterSeconds" "$js_file" 2>/dev/null \
    | grep -iE "$FINANCIAL_TABLES" || true)

  if [ -n "$TTL_ON_FINANCIAL" ]; then
    ERRORS+=("MONGO_TTL_FINANCIAL_COLLECTION")
    DETAILS+=("MongoDB TTL index (expireAfterSeconds) on financial collection | File: $js_file")
    HINTS+=("Remove TTL index from financial collections. MongoDB TTL indexes auto-delete documents — this would destroy regulated records. Only session/OTP/cache collections may use TTL.")
  fi

  # Also catch any expireAfterSeconds less than 7 years on any collection in these files
  SHORT_TTL=$(grep -n "expireAfterSeconds" "$js_file" 2>/dev/null | while IFS= read -r ttl_line; do
    ttl_val=$(echo "$ttl_line" | grep -oE 'expireAfterSeconds[[:space:]]*:[[:space:]]*[0-9]+' \
      | grep -oE '[0-9]+$' || echo "0")
    if [ "$ttl_val" -gt 0 ] && [ "$ttl_val" -lt $((SEVEN_YEARS_DAYS * 86400)) ]; then
      echo "$ttl_line"
    fi
  done || true)

  if [ -n "$SHORT_TTL" ]; then
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      SQL_LINENO=$(echo "$line" | cut -d: -f1)
      LINE_TEXT=$(echo "$line" | cut -d: -f2-)

      # Skip if OTP_TTL_EXEMPT suppression comment is on the same line
      if echo "$LINE_TEXT" | grep -qF "OTP_TTL_EXEMPT:"; then
        continue
      fi
      # Also check the preceding line
      PREV_SQL_LINENO=$((SQL_LINENO - 1))
      PREV_LINE_CONTENT=$(sed -n "${PREV_SQL_LINENO}p" "$js_file" 2>/dev/null || true)
      if echo "$PREV_LINE_CONTENT" | grep -qF "OTP_TTL_EXEMPT:"; then
        continue
      fi

      ERRORS+=("MONGO_SHORT_TTL")
      DETAILS+=("MongoDB TTL index less than 7 years (CNBV minimum) | File: $js_file:$SQL_LINENO")
      HINTS+=("TTL must be >= 220,752,000 seconds (7 years) for any collection that stores financial or user data. For session/OTP collections, TTL is acceptable but must be documented.")
    done <<< "$SHORT_TTL"
  fi
done

# ── Rule 5: Down migration files for financial tables must not DELETE data ────
for down_file in $(find migrations/ -name "*.down.sql" 2>/dev/null | sort); do
  # Down migration should drop TABLE (schema rollback) — not truncate or bulk delete rows
  DESTRUCTIVE_DATA=$(grep -n \
    -e 'TRUNCATE' \
    -e 'DELETE FROM' \
    "$down_file" 2>/dev/null \
    | grep -iE "$FINANCIAL_TABLES" || true)

  if [ -n "$DESTRUCTIVE_DATA" ]; then
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      SQL_LINENO=$(echo "$line" | cut -d: -f1)
      SQL=$(echo "$line" | cut -d: -f2-)
      ERRORS+=("DOWN_MIGRATION_DELETES_DATA")
      DETAILS+=("Down migration deletes financial data rows | File: $down_file:$SQL_LINENO | SQL: $SQL")
      HINTS+=("Down migrations for financial tables must only reverse DDL (DROP TABLE, DROP COLUMN, DROP INDEX). Never include DELETE/TRUNCATE. If you need to remove test data, use a separate seeder script.")
    done <<< "$DESTRUCTIVE_DATA"
  fi
done

# ── Rule 6: Financial table definitions must include created_at column ────────
# Without audit timestamps, you cannot prove compliance with 7-year retention window
for sql_file in $(find migrations/ -name "*.up.sql" 2>/dev/null | sort); do
  # For each CREATE TABLE of a financial entity, check for created_at
  while IFS= read -r table_line; do
    [ -z "$table_line" ] && continue
    TABLE_NAME=$(echo "$table_line" | grep -oiE '(transactions|audit_logs|payments|transfers|ledger_entries)' | head -1 || true)
    [ -z "$TABLE_NAME" ] && continue

    TABLE_LINENO=$(grep -n "$table_line" "$sql_file" | head -1 | cut -d: -f1 || echo "0")

    TABLE_BLOCK=$(awk "NR>=$TABLE_LINENO && NR<=$((TABLE_LINENO+50))" "$sql_file" 2>/dev/null || true)

    if ! echo "$TABLE_BLOCK" | grep -qiE 'created_at|CreatedAt|created_time'; then
      ERRORS+=("FINANCIAL_TABLE_MISSING_CREATED_AT")
      DETAILS+=("Financial table '$TABLE_NAME' in $sql_file has no created_at column")
      HINTS+=("Add: created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(). All financial records require an immutable creation timestamp for CNBV 7-year retention audit. Without it, you cannot prove when a transaction occurred.")
    fi
  done < <(grep -i "CREATE TABLE" "$sql_file" 2>/dev/null | grep -iE "$FINANCIAL_TABLES" || true)
done

# ── Export results ────────────────────────────────────────────────────────────
echo "CNBV_RETENTION_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
