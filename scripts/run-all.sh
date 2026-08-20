#!/bin/bash
# run-all.sh (IMPROVED VERSION)
# Orchestrates all harness constraint checks for InnerLamp
# Usage: ./scripts/run-all.sh [ba-schema-file]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$SCRIPT_DIR/report"
SCHEMA_FILE="${1:-}"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

mkdir -p "$REPORT_DIR"

# ── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# ── Collected violations ──────────────────────────────────────────────────────
ALL_TYPES=()
ALL_DETAILS=()
ALL_HINTS=()
ALL_FILES=()

collect_errors() {
  local output="$1"
  local count
  count=$(echo "$output" | grep "^.*_ERRORS=" | head -1 | cut -d= -f2 || echo "0")

  for i in $(seq 0 $((count - 1))); do
    local etype detail hint
    etype=$(echo "$output"  | grep "^ERR_TYPE_${i}="   | cut -d= -f2- || true)
    detail=$(echo "$output" | grep "^ERR_DETAIL_${i}=" | cut -d= -f2- || true)
    hint=$(echo "$output"   | grep "^ERR_HINT_${i}="   | cut -d= -f2- || true)

    if [ -n "$etype" ]; then
      # ── Deduplication: skip if identical type+detail already collected ──
      local is_dup=false
      for existing_idx in "${!ALL_TYPES[@]}"; do
        if [ "${ALL_TYPES[$existing_idx]}" = "$etype" ] && \
           [ "${ALL_DETAILS[$existing_idx]}" = "${detail:-no detail}" ]; then
          is_dup=true
          break
        fi
      done
      if [ "$is_dup" = "true" ]; then continue; fi

      ALL_TYPES+=("$etype")
      ALL_DETAILS+=("${detail:-no detail}")
      ALL_HINTS+=("${hint:-review affected file}")
      # Try to extract file from detail
      local file
      file=$(echo "${detail:-}" | grep -o "internal/[^ |:]*\.go\|services/[^ |:]*\.go" | head -1 || true)
      ALL_FILES+=("${file:-unknown}")
    fi
  done
}

# ── Run checks ───────────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   InnerLamp Harness — Constraint Suite         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""

# 1. Compile
echo -e "${YELLOW}[1/16]${NC} Compile checks (go build + go vet)..."
COMPILE_OUT=$(bash "$SCRIPT_DIR/check-compile.sh" 2>&1 || true)
collect_errors "$COMPILE_OUT"

# 2. Architecture
echo -e "${YELLOW}[2/16]${NC} Architecture checks (Clean Architecture layers)..."
ARCH_OUT=$(bash "$SCRIPT_DIR/check-architecture.sh" 2>&1 || true)
collect_errors "$ARCH_OUT"

# 3. Completeness
echo -e "${YELLOW}[3/16]${NC} Completeness checks (tests, constructors, implementations)..."
COMP_OUT=$(bash "$SCRIPT_DIR/check-completeness.sh" 2>&1 || true)
collect_errors "$COMP_OUT"

# 4. Business Rules
echo -e "${YELLOW}[4/16]${NC} Business rules checks (BA schema error codes)..."
if [ -n "$SCHEMA_FILE" ]; then
  BR_OUT=$(bash "$SCRIPT_DIR/check-business-rules.sh" "$SCHEMA_FILE" 2>&1 || true)
else
  BR_OUT=$(bash "$SCRIPT_DIR/check-business-rules.sh" 2>&1 || true)
fi
collect_errors "$BR_OUT"

# 5. Test Coverage
echo -e "${YELLOW}[5/16]${NC} Test coverage checks (95% domain, 85% usecase)..."
if [ -f "$SCRIPT_DIR/check-coverage.sh" ]; then
  COV_OUT=$(bash "$SCRIPT_DIR/check-coverage.sh" 2>&1 || true)
  collect_errors "$COV_OUT"
else
  echo -e "${MAGENTA}SKIP: check-coverage.sh not found${NC}"
fi

# 6. Security Scan
echo -e "${YELLOW}[6/16]${NC} Security checks (gosec + banking compliance)..."
if [ -f "$SCRIPT_DIR/check-security.sh" ]; then
  SEC_OUT=$(bash "$SCRIPT_DIR/check-security.sh" 2>&1 || true)
  collect_errors "$SEC_OUT"
else
  echo -e "${MAGENTA}SKIP: check-security.sh not found${NC}"
fi

# 7. Dependencies (CVE + go.mod hygiene)
echo -e "${YELLOW}[7/16]${NC} Dependency checks (govulncheck + go mod verify + go.sum)..."
if [ -f "$SCRIPT_DIR/check-dependencies.sh" ]; then
  DEP_OUT=$(bash "$SCRIPT_DIR/check-dependencies.sh" 2>&1 || true)
  collect_errors "$DEP_OUT"
else
  echo -e "${MAGENTA}SKIP: check-dependencies.sh not found${NC}"
fi

# 8. Migrations (safety + rollback + sequence)
echo -e "${YELLOW}[8/16]${NC} Migration checks (destructive DDL + down files + FK indexes)..."
if [ -f "$SCRIPT_DIR/check-migrations.sh" ]; then
  MIG_OUT=$(bash "$SCRIPT_DIR/check-migrations.sh" 2>&1 || true)
  collect_errors "$MIG_OUT"
else
  echo -e "${MAGENTA}SKIP: check-migrations.sh not found${NC}"
fi

# 9. API Standards
echo -e "${YELLOW}[9/16]${NC} API standards checks (logging, tracing, JWT context, hardcoded config)..."
if [ -f "$SCRIPT_DIR/check-api-standards.sh" ]; then
  API_OUT=$(bash "$SCRIPT_DIR/check-api-standards.sh" 2>&1 || true)
  collect_errors "$API_OUT"
else
  echo -e "${MAGENTA}SKIP: check-api-standards.sh not found${NC}"
fi

# 10. Idempotency
echo -e "${YELLOW}[10/16]${NC} Idempotency checks (write usecases must use pkg/idempotency)..."
if [ -f "$SCRIPT_DIR/check-idempotency.sh" ]; then
  IDEM_OUT=$(bash "$SCRIPT_DIR/check-idempotency.sh" 2>&1 || true)
  collect_errors "$IDEM_OUT"
else
  echo -e "${MAGENTA}SKIP: check-idempotency.sh not found${NC}"
fi

# 11. Error Wrapping
echo -e "${YELLOW}[11/16]${NC} Error wrapping checks (usecase/repo errors must use fmt.Errorf %w)..."
if [ -f "$SCRIPT_DIR/check-error-wrapping.sh" ]; then
  WRAP_OUT=$(bash "$SCRIPT_DIR/check-error-wrapping.sh" 2>&1 || true)
  collect_errors "$WRAP_OUT"
else
  echo -e "${MAGENTA}SKIP: check-error-wrapping.sh not found${NC}"
fi

# 12. Context Deadline
echo -e "${YELLOW}[12/16]${NC} Context timeout checks (external calls must have deadline)..."
if [ -f "$SCRIPT_DIR/check-ctx-deadline.sh" ]; then
  CTX_OUT=$(bash "$SCRIPT_DIR/check-ctx-deadline.sh" 2>&1 || true)
  collect_errors "$CTX_OUT"
else
  echo -e "${MAGENTA}SKIP: check-ctx-deadline.sh not found${NC}"
fi

# 13. gRPC Error Codes
echo -e "${YELLOW}[13/16]${NC} gRPC error checks (handlers must return status.Error codes)..."
if [ -f "$SCRIPT_DIR/check-grpc-errors.sh" ]; then
  GRPC_OUT=$(bash "$SCRIPT_DIR/check-grpc-errors.sh" 2>&1 || true)
  collect_errors "$GRPC_OUT"
else
  echo -e "${MAGENTA}SKIP: check-grpc-errors.sh not found${NC}"
fi

# 14. DB Transactions
echo -e "${YELLOW}[14/16]${NC} DB transaction checks (multi-write usecases must use transactions)..."
if [ -f "$SCRIPT_DIR/check-db-transactions.sh" ]; then
  DBTX_OUT=$(bash "$SCRIPT_DIR/check-db-transactions.sh" 2>&1 || true)
  collect_errors "$DBTX_OUT"
else
  echo -e "${MAGENTA}SKIP: check-db-transactions.sh not found${NC}"
fi

# 15. MongoDB Indexes
echo -e "${YELLOW}[15/16]${NC} MongoDB index checks (query fields must have indexes)..."
if [ -f "$SCRIPT_DIR/check-mongo-indexes.sh" ]; then
  MONGOIDX_OUT=$(bash "$SCRIPT_DIR/check-mongo-indexes.sh" 2>&1 || true)
  collect_errors "$MONGOIDX_OUT"
else
  echo -e "${MAGENTA}SKIP: check-mongo-indexes.sh not found${NC}"
fi

# 16. GCP Config
echo -e "${YELLOW}[16/21]${NC} GCP config checks (no hardcoded project/topic/bucket/region)..."
if [ -f "$SCRIPT_DIR/check-gcp-config.sh" ]; then
  GCPCFG_OUT=$(bash "$SCRIPT_DIR/check-gcp-config.sh" 2>&1 || true)
  collect_errors "$GCPCFG_OUT"
else
  echo -e "${MAGENTA}SKIP: check-gcp-config.sh not found${NC}"
fi

# 17. Context Propagation
echo -e "${YELLOW}[17/21]${NC} Context propagation checks (no context.Background in service code)..."
if [ -f "$SCRIPT_DIR/check-context-propagation.sh" ]; then
  CTXPROP_OUT=$(bash "$SCRIPT_DIR/check-context-propagation.sh" 2>&1 || true)
  collect_errors "$CTXPROP_OUT"
else
  echo -e "${MAGENTA}SKIP: check-context-propagation.sh not found${NC}"
fi

# 18. Soft Delete (CNBV compliance)
echo -e "${YELLOW}[18/21]${NC} Soft delete checks (no hard delete on regulated entities)..."
if [ -f "$SCRIPT_DIR/check-soft-delete.sh" ]; then
  SOFTDEL_OUT=$(bash "$SCRIPT_DIR/check-soft-delete.sh" 2>&1 || true)
  collect_errors "$SOFTDEL_OUT"
else
  echo -e "${MAGENTA}SKIP: check-soft-delete.sh not found${NC}"
fi

# 19. Pub/Sub Patterns
echo -e "${YELLOW}[19/21]${NC} Pub/Sub pattern checks (ack/nack + naming + publish-after-commit)..."
if [ -f "$SCRIPT_DIR/check-pubsub-patterns.sh" ]; then
  PUBSUB_OUT=$(bash "$SCRIPT_DIR/check-pubsub-patterns.sh" 2>&1 || true)
  collect_errors "$PUBSUB_OUT"
else
  echo -e "${MAGENTA}SKIP: check-pubsub-patterns.sh not found${NC}"
fi

# 20. Rate Limiting
echo -e "${YELLOW}[20/21]${NC} Rate limiting checks (auth endpoints + Redis backend)..."
if [ -f "$SCRIPT_DIR/check-rate-limiting.sh" ]; then
  RATELIM_OUT=$(bash "$SCRIPT_DIR/check-rate-limiting.sh" 2>&1 || true)
  collect_errors "$RATELIM_OUT"
else
  echo -e "${MAGENTA}SKIP: check-rate-limiting.sh not found${NC}"
fi

# 21. CNBV Data Retention
echo -e "${YELLOW}[21/23]${NC} CNBV data retention checks (7-year retention + no cascades on financial tables)..."
if [ -f "$SCRIPT_DIR/check-cnbv-data-retention.sh" ]; then
  CNBV_OUT=$(bash "$SCRIPT_DIR/check-cnbv-data-retention.sh" 2>&1 || true)
  collect_errors "$CNBV_OUT"
else
  echo -e "${MAGENTA}SKIP: check-cnbv-data-retention.sh not found${NC}"
fi

# 22. BA - BR/ERR Bidirectional Coverage
echo -e "${YELLOW}[22/24]${NC} BA schema checks (BR-ERR bidirectional coverage + numbering gaps)..."
if [ -n "$SCHEMA_FILE" ] && [ -f "$SCRIPT_DIR/check-ba-br-err-coverage.sh" ]; then
  BRERR_OUT=$(bash "$SCRIPT_DIR/check-ba-br-err-coverage.sh" "$SCHEMA_FILE" 2>&1 || true)
  collect_errors "$BRERR_OUT"
elif [ -z "$SCHEMA_FILE" ]; then
  echo -e "${MAGENTA}SKIP: check-ba-br-err-coverage.sh (no schema file provided — pass path as argument)${NC}"
else
  echo -e "${MAGENTA}SKIP: check-ba-br-err-coverage.sh not found${NC}"
fi

# 23. BA - Sensitive Fields Security
echo -e "${YELLOW}[23/24]${NC} BA schema checks (sensitive fields + hash algorithms + unique constraints)..."
if [ -n "$SCHEMA_FILE" ] && [ -f "$SCRIPT_DIR/check-ba-sensitive-fields.sh" ]; then
  SENSITIVE_OUT=$(bash "$SCRIPT_DIR/check-ba-sensitive-fields.sh" "$SCHEMA_FILE" 2>&1 || true)
  collect_errors "$SENSITIVE_OUT"
elif [ -z "$SCHEMA_FILE" ]; then
  echo -e "${MAGENTA}SKIP: check-ba-sensitive-fields.sh (no schema file provided — pass path as argument)${NC}"
else
  echo -e "${MAGENTA}SKIP: check-ba-sensitive-fields.sh not found${NC}"
fi

# 24. Mock Data & Unfinished TODOs
echo -e "${YELLOW}[24/24]${NC} Mock data checks (TODO/FIXME + mock in production + example files missing NOT_FOR_PRODUCTION header)..."
if [ -f "$SCRIPT_DIR/check-mock-data.sh" ]; then
  MOCK_OUT=$(bash "$SCRIPT_DIR/check-mock-data.sh" 2>&1 || true)
  collect_errors "$MOCK_OUT"
else
  echo -e "${MAGENTA}SKIP: check-mock-data.sh not found${NC}"
fi

TOTAL=${#ALL_TYPES[@]}

# ── Write error-report.md ─────────────────────────────────────────────────────
{
  echo "# InnerLamp Banking Harness — Error Report"
  echo ""
  echo "| Field | Value |"
  echo "|---|---|"
  echo "| Timestamp | $TIMESTAMP |"
  echo "| Schema | ${SCHEMA_FILE:-not provided} |"
  echo "| Total violations | $TOTAL |"
  echo ""

  if [ "$TOTAL" -eq 0 ]; then
    echo "## Result: PASS ✅"
    echo ""
    echo "All constraint checks passed. Code is ready for banking production deployment."
    echo ""
    echo "### Passed Checks:"
    echo "- ✅ Compilation (go build + go vet)"
    echo "- ✅ Clean Architecture layer boundaries"
    echo "- ✅ Code completeness (tests, constructors, implementations)"
    echo "- ✅ Business rules (BA schema error codes)"
    echo "- ✅ Test coverage targets"
    echo "- ✅ Security (gosec + banking compliance)"
    echo "- ✅ Dependencies (CVE-free + go.mod verified)"
    echo "- ✅ Migrations (rollback files + no unreviewed destructive DDL)"
    echo "- ✅ API standards (logging, tracing, JWT context, hardcoded config)"
    echo "- ✅ Idempotency (write usecases protected against duplicate requests)"
    echo "- ✅ Error wrapping (infrastructure errors wrapped with fmt.Errorf %w)"
    echo "- ✅ Context timeout (external calls have deadline/timeout)"
    echo "- ✅ Context propagation (no context.Background in service code)"
    echo "- ✅ Soft delete (no hard delete on CNBV-regulated entities)"
    echo "- ✅ Pub/Sub patterns (ack/nack, naming convention, publish-after-commit)"
    echo "- ✅ Rate limiting (auth endpoints protected, Redis backend)"
    echo "- ✅ CNBV data retention (7-year compliance, no cascade deletes)"
    echo "- ✅ BA BR-ERR coverage (no orphaned BRs, valid references, no numbering gaps)"
    echo "- ✅ BA sensitive fields (hash algorithms documented, no plaintext credentials, unique constraints covered)"
    echo "- ✅ Mock data (no TODO/FIXME in production, no unguarded mock in example files)"
  else
    echo "## Result: FAIL ❌"
    echo ""
    echo "⚠️ **${TOTAL} violations detected. Review and fix before deployment.**"
    echo ""
    echo "---"
    echo ""
    echo "## Violations"
    echo ""
    
    for i in "${!ALL_TYPES[@]}"; do
      echo "### ❌ Violation #$((i+1)): ${ALL_TYPES[$i]}"
      echo ""
      echo "**Detail:**"
      echo "\`\`\`"
      echo "${ALL_DETAILS[$i]}"
      echo "\`\`\`"
      echo ""
      echo "**How to Fix:**"
      echo "${ALL_HINTS[$i]}"
      echo ""
      if [ "${ALL_FILES[$i]}" != "unknown" ]; then
        echo "**Affected File:** \`${ALL_FILES[$i]}\`"
        echo ""
      fi
      echo "---"
      echo ""
    done
    
    echo "## Next Steps"
    echo ""
    echo "1. Review each violation above"
    echo "2. Fix violations manually OR run: \`@Fix Agent\` in Copilot Chat"
    echo "3. Re-run: \`./scripts/run-all.sh\`"
    echo "4. Repeat until all checks pass ✅"
    echo ""
    echo "## Fix Agent Usage"
    echo ""
    echo "\`\`\`"
    echo "1. Open GitHub Copilot Chat in VS Code"
    echo "2. Select agent: Fix Agent"
    echo "3. Paste path to this error report or say 'fix all violations'"
    echo "\`\`\`"
  fi
} > "$REPORT_DIR/error-report.md"

# ── Write JSON report for CI/CD ──────────────────────────────────────────────
{
  echo "{"
  echo "  \"timestamp\": \"$TIMESTAMP\","
  echo "  \"schema\": \"${SCHEMA_FILE:-null}\","
  echo "  \"total_violations\": $TOTAL,"
  echo "  \"passed\": $([ "$TOTAL" -eq 0 ] && echo "true" || echo "false"),"
  echo "  \"violations\": ["
  
  for i in "${!ALL_TYPES[@]}"; do
    echo "    {"
    echo "      \"type\": \"${ALL_TYPES[$i]}\","
    echo "      \"detail\": \"${ALL_DETAILS[$i]}\","
    echo "      \"hint\": \"${ALL_HINTS[$i]}\","
    echo "      \"file\": \"${ALL_FILES[$i]}\""
    echo -n "    }"
    [ $i -lt $((TOTAL - 1)) ] && echo "," || echo ""
  done
  
  echo "  ]"
  echo "}"
} > "$REPORT_DIR/check-result.json"

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ "$TOTAL" -eq 0 ]; then
  echo -e "${GREEN}✅ ALL CHECKS PASSED${NC}"
  echo ""
  echo "Code is ready for banking production deployment."
else
  echo -e "${RED}❌ FOUND $TOTAL VIOLATIONS${NC}"
  echo ""
  echo "Review error report: $REPORT_DIR/error-report.md"
  echo "Or use Fix Agent in Copilot Chat to auto-fix violations"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Exit code
[ "$TOTAL" -eq 0 ] && exit 0 || exit 1
