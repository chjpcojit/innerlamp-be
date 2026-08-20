#!/bin/bash
# check-dependencies.sh
# Checks: CVE vulnerabilities in Go dependencies, go.mod hygiene
# Banking compliance: CNBV requires no known vulnerabilities in production dependencies
# Tools required: govulncheck (install via setup-harness.sh)

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── 1. govulncheck — CVE scan trên toàn bộ dependencies ─────────────────────
if ! command -v govulncheck &>/dev/null; then
  ERRORS+=("DEPENDENCY_TOOL_MISSING")
  DETAILS+=("govulncheck not installed — cannot scan for CVE vulnerabilities")
  HINTS+=("Run: go install golang.org/x/vuln/cmd/govulncheck@latest  OR  ./scripts/setup-harness.sh")
  echo "DEP_ERRORS=${#ERRORS[@]}"
  echo "ERR_TYPE_0=${ERRORS[0]}"
  echo "ERR_DETAIL_0=${DETAILS[0]}"
  echo "ERR_HINT_0=${HINTS[0]}"
  exit 0
fi

VULN_OUT=$(govulncheck ./... 2>&1 || true)

# govulncheck exits non-zero when vulnerabilities found — parse output
if echo "$VULN_OUT" | grep -q "^Vulnerability #\|^Found "; then
  while IFS= read -r line; do
    if [[ "$line" =~ ^Vulnerability|^Found ]]; then
      ERRORS+=("CVE_VULNERABILITY")
      DETAILS+=("$line")
      HINTS+=("Upgrade or replace the affected dependency. Run 'govulncheck ./...' for details. CNBV compliance: no known CVEs in production.")
    fi
  done <<< "$VULN_OUT"
fi

# ── 2. go mod verify — vendor/ integrity check ─────────────────────────────
VERIFY_OUT=$(go mod verify 2>&1 || true)
if echo "$VERIFY_OUT" | grep -q "FAIL\|hash mismatch"; then
  while IFS= read -r line; do
    if echo "$line" | grep -q "FAIL\|hash mismatch"; then
      ERRORS+=("VENDOR_INTEGRITY_FAIL")
      DETAILS+=("go mod verify failed: $line")
      HINTS+=("vendor/ contents do not match go.sum checksums — possible tampering. Run: go mod download && go mod verify")
    fi
  done <<< "$VERIFY_OUT"
fi

# ── 3. go.mod stale check — go.mod/go.sum not updated after dependency change ─
# Run go mod tidy in dry-run mode by checking if it would produce a diff
TIDY_OUT=$(go mod tidy 2>&1 || true)
# Check if go.mod or go.sum changed after tidy
MOD_DIFF=$(git diff --name-only go.mod go.sum 2>/dev/null || true)
if [ -n "$MOD_DIFF" ]; then
  ERRORS+=("STALE_GO_MOD")
  DETAILS+=("go.mod or go.sum not in sync with imports: $MOD_DIFF")
  HINTS+=("Run 'go mod tidy' and commit the updated go.mod and go.sum files")
  # Restore to avoid leaving repo dirty after check
  git checkout -- go.mod go.sum 2>/dev/null || true
fi

# ── 4. Direct dependency license check (GPL-incompatible alert) ─────────────
# Detect copyleft licenses that cannot be used in proprietary banking software
# Uses go.mod to get direct deps, checks module paths for known problematic licenses
COPYLEFT_PATTERNS="gpl|lgpl|agpl|copyleft"
DIRECT_DEPS=$(grep -E "^\s+[a-z]" go.mod | awk '{print $1}' | grep -viE "^go$|^toolchain$" || true)

while IFS= read -r dep; do
  if [ -z "$dep" ]; then continue; fi
  # Check known copyleft packages (conservative list — add more as needed)
  if echo "$dep" | grep -qiE "mysql/mysql-server|mariadb|gpl"; then
    ERRORS+=("LICENSE_VIOLATION")
    DETAILS+=("Potential copyleft license in dependency: $dep")
    HINTS+=("Review license of $dep — GPL/LGPL/AGPL incompatible with proprietary banking software. Use MIT/Apache/BSD only.")
  fi
done <<< "$DIRECT_DEPS"

# ── Export results ────────────────────────────────────────────────────────────
echo "DEP_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
