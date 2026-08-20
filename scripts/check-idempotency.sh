#!/bin/bash
# check-idempotency.sh
# Checks: usecase Execute() functions that perform write operations must use idempotency
# Banking compliance: InnerLamp global instructions §3 — all financial transactions MUST
# include an idempotency key to prevent duplicate transactions.
# Package: pkg/idempotency (Manager interface with Reserve/Complete pattern)

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

MODULE=$(go list -m 2>/dev/null || echo "unknown")

# Write operation signatures that mutate state and require idempotency protection
WRITE_PATTERNS="\.Create(\|\.Update(\|\.Save(\|\.Insert(\|\.Upsert("

# ── Collect all usecase files that perform write operations ───────────────────
for uc_file in $(find services/ -path "*/usecase/*.go" ! -name "*_test.go" 2>/dev/null | sort); do
  # Skip if file has no write operation at all
  if ! grep -qE "$WRITE_PATTERNS" "$uc_file" 2>/dev/null; then
    continue
  fi

  # Skip read-only usecases explicitly (lookup, get, list, search, postal)
  basename_lc=$(basename "$uc_file" | tr '[:upper:]' '[:lower:]')
  case "$basename_lc" in
    get_*|list_*|search_*|lookup_*|postal_*|*_lookup.go|*_get.go|*_list.go)
      continue
      ;;
  esac

  # Check if file imports the idempotency package
  if ! grep -q "idempotency" "$uc_file" 2>/dev/null; then
    ERRORS+=("MISSING_IDEMPOTENCY")
    DETAILS+=("UseCase has write operations but no idempotency protection | File: $uc_file")
    HINTS+=("Inject pkg/idempotency.Manager and call idempotencyMgr.Reserve(ctx, req.IdempotencyKey, hash, ttl) at start of Execute(). Return cached response if reservation returns false (duplicate request). Import: \"${MODULE}/pkg/idempotency\"")
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "IDEMPOTENCY_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
