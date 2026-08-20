#!/bin/bash
# check-ctx-deadline.sh
# Checks: external HTTP/gRPC calls in infrastructure/services must have context timeout.
#
# Banking SLA: P95 < 2s per transaction (copilot-instructions.md §7).
# If an external call (Bankaool API, Twilio SMS, OCR, etc.) hangs without timeout,
# the whole request hangs — violating SLA and potentially exhausting Cloud Run concurrency.
#
# Rule: Any file in infrastructure/services/ that calls http.NewRequest, http.Do,
# http.Get, http.Post, or gRPC Dial must have context.WithTimeout or context.WithDeadline
# in the same file.

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── External call signatures that need timeout protection ─────────────────────
HTTP_CALL_PATTERN="http\.NewRequest\|http\.NewRequestWithContext\|http\.Get\|http\.Post\|http\.Do\|\.Do(req"
GRPC_CALL_PATTERN="grpc\.Dial\|grpc\.NewClient\|grpc\.DialContext"
EXTERNAL_CALL_PATTERN="${HTTP_CALL_PATTERN}\|${GRPC_CALL_PATTERN}"

# Timeout/deadline indicators
TIMEOUT_PATTERN="context\.WithTimeout\|context\.WithDeadline\|WithTimeout\|WithDeadline"

# ── Scan infrastructure/services layer ───────────────────────────────────────
for infra_file in $(find services/ \( -path "*/infrastructure/services/*.go" -o -path "*/infrastructure/adapters/*.go" \) ! -name "*_test.go" 2>/dev/null | sort); do
  # Check if this file makes external calls
  if ! grep -qE "$EXTERNAL_CALL_PATTERN" "$infra_file" 2>/dev/null; then
    continue
  fi

  # Check if the file sets a timeout anywhere
  if ! grep -qE "$TIMEOUT_PATTERN" "$infra_file" 2>/dev/null; then
    # Find the specific call lines for context in the error report
    CALL_LINES=$(grep -nE "$EXTERNAL_CALL_PATTERN" "$infra_file" 2>/dev/null | head -3 || true)
    FIRST_LINE=$(echo "$CALL_LINES" | head -1)
    LINENO=$(echo "$FIRST_LINE" | cut -d: -f1)

    ERRORS+=("MISSING_CONTEXT_TIMEOUT")
    DETAILS+=("External call without context timeout | File: $infra_file:${LINENO:-1} | External calls found: $(echo "$CALL_LINES" | wc -l | tr -d ' ')")
    HINTS+=("Add timeout before the call: ctx, cancel := context.WithTimeout(ctx, 10*time.Second); defer cancel(). Recommended timeouts: SMS/Email=10s, OCR=30s, Bankaool API=15s. Reference: copilot-instructions.md §7 P95 < 2s SLA")
  fi
done

# ── Also check usecase layer — direct HTTP calls are a red flag ───────────────
# (usecase should never call HTTP directly, but if it does AND has no timeout → double violation)
for uc_file in $(find services/ -path "*/usecase/*.go" ! -name "*_test.go" 2>/dev/null | sort); do
  if grep -qE "$HTTP_CALL_PATTERN" "$uc_file" 2>/dev/null; then
    CALL_LINES=$(grep -nE "$HTTP_CALL_PATTERN" "$uc_file" 2>/dev/null | head -1 || true)
    LINENO=$(echo "$CALL_LINES" | cut -d: -f1)
    ERRORS+=("HTTP_IN_USECASE_NO_TIMEOUT")
    DETAILS+=("UseCase makes direct HTTP call (should use infrastructure adapter) and may have no timeout | File: $uc_file:${LINENO:-1}")
    HINTS+=("Move HTTP call to infrastructure/services/ adapter. UseCase must not import net/http directly — this is also an ARCH_VIOLATION. Use dependency injection via interface.")
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "CTX_DEADLINE_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
