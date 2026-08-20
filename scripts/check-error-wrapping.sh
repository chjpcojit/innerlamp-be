#!/bin/bash
# check-error-wrapping.sh
# Checks: infrastructure/repo errors in usecase/repository layers must be wrapped
# with fmt.Errorf("context: %w", err) to preserve stack context for production debugging.
#
# Rule: When err comes from a repository/infra call (uc.xxxRepo.Xxx, uc.xxxSvc.Xxx),
# it MUST be wrapped. Domain sentinel errors (domain.ErrXxx) should pass through directly.
#
# Pattern detected: a repo/infra call assigns err, but the next return of err is bare
# (not inside fmt.Errorf). Detected heuristically via grep on same-line patterns.

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Repo/infra call patterns that produce errors needing wrapping ─────────────
# These are method calls that go into DB, cache, external service etc.
INFRA_CALL_PATTERN="uc\.[a-zA-Z]*[Rr]epo\.\|uc\.[a-zA-Z]*[Ss]vc\.\|uc\.[a-zA-Z]*[Ss]ervice\.\|uc\.[a-zA-Z]*[Cc]lient\."

# ── Scan usecase layer ────────────────────────────────────────────────────────
for uc_file in $(find services/ -path "*/usecase/*.go" ! -name "*_test.go" 2>/dev/null | sort); do
  # Find lines where err is assigned from an infra call WITHOUT fmt.Errorf wrapping
  # Pattern: "if err := uc.someRepo.Method(...); err != nil {" is correct
  # Pattern: "return nil, err" right after infra call WITHOUT fmt.Errorf — violation

  # Look for bare "return nil, err" or "return err" that are NOT:
  # 1. Preceded by fmt.Errorf on the same line
  # 2. Returning a domain sentinel error variable (domain.Err*)
  BARE_RETURNS=$(grep -n \
    -E "^\s+return (nil, err|err)$" \
    "$uc_file" 2>/dev/null || true)

  if [ -n "$BARE_RETURNS" ]; then
    while IFS= read -r line; do
      if [ -z "$line" ]; then continue; fi

      LINENO=$(echo "$line" | cut -d: -f1)
      CONTENT=$(echo "$line" | cut -d: -f2-)

      # Get the line that assigned err (look back up to 3 lines)
      PREV_LINES=$(sed -n "$((LINENO - 3)),$((LINENO - 1))p" "$uc_file" 2>/dev/null || true)

      # If err came from an infra call → must be wrapped
      if echo "$PREV_LINES" | grep -qE "$INFRA_CALL_PATTERN"; then
        # Confirm it's not already wrapped (fmt.Errorf exists in context)
        if ! echo "$PREV_LINES" | grep -q "fmt.Errorf"; then
          ERRORS+=("UNWRAPPED_ERROR")
          DETAILS+=("Bare 'return err' after infrastructure call — missing fmt.Errorf wrap | File: $uc_file:$LINENO | Line: $CONTENT")
          HINTS+=("Wrap the error: return nil, fmt.Errorf(\"<operation name>: %w\", err)  This preserves stack context for production debugging and log correlation.")
        fi
      fi
    done <<< "$BARE_RETURNS"
  fi
done

# ── Scan repository layer ─────────────────────────────────────────────────────
for repo_file in $(find services/ -path "*/repository/*.go" ! -name "*_test.go" 2>/dev/null | sort); do
  # In repository layer: calls to pgx, mongo, redis must be wrapped
  DB_CALL_PATTERN="\.Query\|\.Exec\|\.FindOne\|\.Find(\|\.InsertOne\|\.UpdateOne\|\.DeleteOne\|\.Get(\|\.Set(\|\.Del("

  BARE_RETURNS=$(grep -n \
    -E "^\s+return (nil, err|err)$" \
    "$repo_file" 2>/dev/null || true)

  if [ -n "$BARE_RETURNS" ]; then
    while IFS= read -r line; do
      if [ -z "$line" ]; then continue; fi

      LINENO=$(echo "$line" | cut -d: -f1)
      CONTENT=$(echo "$line" | cut -d: -f2-)

      PREV_LINES=$(sed -n "$((LINENO - 3)),$((LINENO - 1))p" "$repo_file" 2>/dev/null || true)

      if echo "$PREV_LINES" | grep -qE "$DB_CALL_PATTERN"; then
        if ! echo "$PREV_LINES" | grep -q "fmt.Errorf"; then
          ERRORS+=("UNWRAPPED_ERROR")
          DETAILS+=("Bare 'return err' after DB/cache call — missing fmt.Errorf wrap | File: $repo_file:$LINENO | Line: $CONTENT")
          HINTS+=("Wrap the error: return nil, fmt.Errorf(\"<repo method>: %w\", err)  Include entity name for context, e.g. \"get session by id: %w\"")
        fi
      fi
    done <<< "$BARE_RETURNS"
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "ERROR_WRAP_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
